package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"net/url"
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
		return nil, fmt.Errorf("camera %s is not sending frames right now (id=%s size=%d)", camID, camID, len(frame))
	}
	return frame, nil
}

// GetCameraFrame is the package-level helper used by the server and watcher.
func GetCameraFrame(ctx context.Context, cameraAddress string) ([]byte, error) {
	addr, err := netip.ParseAddr(cameraAddress)
	if err != nil || !addr.Is4() {
		return nil, fmt.Errorf("invalid camera address %q", cameraAddress)
	}
	return bmaCam.Frame(ctx, cameraAddress)
}
