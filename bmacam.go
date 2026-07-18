package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// bmatraffic.com serves camera video as JPEG frames from show.aspx, but only
// to clients that hold an ASP.NET session cookie (set on index.aspx) and have
// opened PlayVideo.aspx?ID=<cam> recently. Without that, frames come back as
// a blank placeholder (~1.4 KB). This proxy keeps one session alive and
// re-opens PlayVideo per camera so phones can view cameras over HTTPS.
type camSession struct {
	mu         sync.Mutex
	client     *http.Client
	sessionAt  time.Time
	lastPlayed map[string]time.Time
}

var bmaCam = newCamSession()
var bmaRelayClient = &http.Client{Timeout: 15 * time.Second}

const (
	camSessionTTL  = 20 * time.Minute
	camPlayTTL     = 45 * time.Second // re-open PlayVideo if idle longer than this
	blankFrameSize = 3000             // frames smaller than this are the blank placeholder
)

func newCamSession() *camSession {
	jar, _ := cookiejar.New(nil)
	return &camSession{
		client:     &http.Client{Jar: jar, Timeout: 20 * time.Second},
		lastPlayed: map[string]time.Time{},
	}
}

func (c *camSession) get(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) thai-bus-watch/2.0")
	req.Header.Set("Referer", "http://www.bmatraffic.com/index.aspx")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bmatraffic HTTP %d", resp.StatusCode)
	}
	return body, nil
}

// ensure opens (or refreshes) the ASP.NET session and starts the camera stream.
// Caller must hold c.mu.
func (c *camSession) ensure(ctx context.Context, camID string) error {
	if time.Since(c.sessionAt) > camSessionTTL {
		if _, err := c.get(ctx, bmaIndexURL); err != nil {
			return fmt.Errorf("cannot open bmatraffic session: %w", err)
		}
		c.sessionAt = time.Now()
		c.lastPlayed = map[string]time.Time{}
	}
	if time.Since(c.lastPlayed[camID]) > camPlayTTL {
		if _, err := c.get(ctx, bmaPlayURL+url.QueryEscape(camID)); err != nil {
			return fmt.Errorf("cannot start camera %s: %w", camID, err)
		}
		c.lastPlayed[camID] = time.Now()
	}
	return nil
}

// Frame fetches the latest JPEG frame for a camera, establishing the session
// and stream on demand. Retries once after a blank frame (the stream needs a
// moment to warm up after PlayVideo).
func (c *camSession) Frame(ctx context.Context, camID string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensure(ctx, camID); err != nil {
		return nil, err
	}

	frameURL := fmt.Sprintf("%sshow.aspx?image=%s&time=%d",
		"http://www.bmatraffic.com/", url.QueryEscape(camID), time.Now().UnixNano())

	frame, err := c.get(ctx, frameURL)
	if err != nil {
		return nil, err
	}

	for attempt := 0; len(frame) < blankFrameSize && attempt < 3; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1200 * time.Millisecond):
		}
		frameURL = fmt.Sprintf("http://www.bmatraffic.com/show.aspx?image=%s&time=%d",
			url.QueryEscape(camID), time.Now().UnixNano())
		frame, err = c.get(ctx, frameURL)
		if err != nil {
			return nil, err
		}
	}

	if len(frame) < blankFrameSize {
		// The stream is no longer viewable. Force the next request to revisit
		// PlayVideo instead of waiting for the normal idle timeout.
		delete(c.lastPlayed, camID)
		return nil, fmt.Errorf("camera %s is not sending frames right now (id=%s size=%d)", camID, camID, len(frame))
	}
	// A good frame proves that this camera session is still active. Keep using
	// show.aspx directly and avoid repeating the index/PlayVideo flow.
	c.lastPlayed[camID] = time.Now()
	return frame, nil
}

// GetCameraFrameByID validates the public numeric ID against the camera catalog
// before passing it through BMA's PlayVideo/show.aspx session flow.
func GetCameraFrameByID(ctx context.Context, cameraID string) ([]byte, error) {
	for _, camera := range CachedBMACameras() {
		if camera.ID == cameraID {
			return bmaCam.Frame(ctx, camera.ID)
		}
	}
	return nil, fmt.Errorf("unknown camera ID %q", cameraID)
}

// GetCameraFrameForRequest uses a secure camera-only upstream when configured.
// This lets Oracle serve the public same-origin endpoint even when its network
// cannot reach BMA's HTTP-only site directly.
func GetCameraFrameForRequest(ctx context.Context, cameraID string) ([]byte, error) {
	relayBase := strings.TrimRight(strings.TrimSpace(os.Getenv("BMA_CAMERA_RELAY_URL")), "/")
	if relayBase == "" {
		return GetCameraFrameByID(ctx, cameraID)
	}
	relayURL, err := url.Parse(relayBase)
	if err != nil || relayURL.Scheme != "https" || relayURL.Host == "" {
		return nil, fmt.Errorf("invalid BMA_CAMERA_RELAY_URL")
	}
	relayURL.Path = strings.TrimRight(relayURL.Path, "/") + "/camera/" + url.PathEscape(cameraID) + "/frame"
	relayURL.RawQuery = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, relayURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BUS287-Oracle/1.0")
	resp, err := bmaRelayClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("camera relay: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("camera relay HTTP %d", resp.StatusCode)
	}
	frame, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("camera relay: %w", err)
	}
	if len(frame) < blankFrameSize {
		return nil, fmt.Errorf("camera relay returned an invalid frame")
	}
	return frame, nil
}

// CheckCameraRelay reports whether the configured camera-only relay is
// reachable. Camera availability is deliberately separate from the main app
// health so bus tracking remains usable when the relay is offline.
func CheckCameraRelay(ctx context.Context) error {
	relayBase := strings.TrimRight(strings.TrimSpace(os.Getenv("BMA_CAMERA_RELAY_URL")), "/")
	if relayBase == "" {
		return nil
	}
	relayURL, err := url.Parse(relayBase)
	if err != nil || relayURL.Scheme != "https" || relayURL.Host == "" {
		return fmt.Errorf("invalid BMA_CAMERA_RELAY_URL")
	}
	relayURL.Path = strings.TrimRight(relayURL.Path, "/") + "/healthz"
	relayURL.RawQuery = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, relayURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "BUS287-Oracle/1.0")
	resp, err := bmaRelayClient.Do(req)
	if err != nil {
		return fmt.Errorf("camera relay: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("camera relay health HTTP %d", resp.StatusCode)
	}
	return nil
}
