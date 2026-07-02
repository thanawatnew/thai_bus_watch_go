package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	bmaIndexURL = "http://www.bmatraffic.com/index.aspx"
	bmaPlayURL  = "http://www.bmatraffic.com/PlayVideo.aspx?ID="
	bmaCacheTTL = time.Hour
)

type Camera struct {
	ID          string  `json:"id"`
	NameTH      string  `json:"name_th"`
	NameEN      string  `json:"name_en"`
	DirectionTH string  `json:"direction_th"`
	DirectionEN string  `json:"direction_en"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	IP          string  `json:"ip"`
	Icon        string  `json:"icon"`
	FeedURL     string  `json:"feed_url"`
}

var cameraCache struct {
	sync.Mutex
	cameras   []Camera
	fetchedAt time.Time
}

// GetBMACameras scrapes the BMA traffic page for the camera list.
// The list rarely changes, so results are cached for an hour; a stale
// cache is served if a refresh fails.
func GetBMACameras(ctx context.Context) ([]Camera, error) {
	cameraCache.Lock()
	defer cameraCache.Unlock()

	if cameraCache.cameras != nil && time.Since(cameraCache.fetchedAt) < bmaCacheTTL {
		return cameraCache.cameras, nil
	}

	cameras, err := fetchBMACameras(ctx)
	if err != nil {
		if cameraCache.cameras != nil {
			return cameraCache.cameras, nil
		}
		return nil, err
	}

	cameraCache.cameras = cameras
	cameraCache.fetchedAt = time.Now()
	return cameras, nil
}

func fetchBMACameras(ctx context.Context) ([]Camera, error) {
	body, err := HTTPGet(ctx, bmaIndexURL)
	if err != nil {
		return nil, err
	}

	block, err := ExtractLocationsVariable(string(body))
	if err != nil {
		return nil, err
	}

	cameras := ParseLocations(block)
	if len(cameras) == 0 {
		return nil, errors.New("found var locations, but could not parse cameras")
	}

	return cameras, nil
}

func ExtractLocationsVariable(html string) (string, error) {
	idx := strings.Index(html, "var locations")
	if idx == -1 {
		return "", errors.New("cannot find `var locations` in BMA index.aspx")
	}

	startRel := strings.Index(html[idx:], "[")
	if startRel == -1 {
		return "", errors.New("found `var locations`, but cannot find opening `[`")
	}

	start := idx + startRel
	depth := 0
	inString := false
	escaped := false
	quote := byte(0)

	for i := start; i < len(html); i++ {
		c := html[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				inString = false
			}
			continue
		}

		if c == '\'' || c == '"' {
			inString = true
			quote = c
			continue
		}

		if c == '[' {
			depth++
		}

		if c == ']' {
			depth--
			if depth == 0 {
				return html[start : i+1], nil
			}
		}
	}

	return "", errors.New("cannot find closing `]` for var locations")
}

func ParseLocations(locationsBlock string) []Camera {
	/*
	 *	Expected BMA format:
	 *
	 *	['1456','DS-01-01 แยกเกียกกาย','-','แยกเกียกกาย','-',13.79723,100.52159,'10.102.101.2','pin-right.png']
	 */

	q := `['"]([^'"]*)['"]`

	re := regexp.MustCompile(
		`\[\s*` +
			q + `\s*,\s*` +
			q + `\s*,\s*` +
			q + `\s*,\s*` +
			q + `\s*,\s*` +
			q + `\s*,\s*` +
			`(-?\d+(?:\.\d+)?)\s*,\s*` +
			`(-?\d+(?:\.\d+)?)\s*,\s*` +
			q + `\s*,\s*` +
			q +
			`\s*\]`,
	)

	matches := re.FindAllStringSubmatch(locationsBlock, -1)
	cameras := make([]Camera, 0, len(matches))

	for _, m := range matches {
		lat, err1 := strconv.ParseFloat(m[6], 64)
		lon, err2 := strconv.ParseFloat(m[7], 64)

		if err1 != nil || err2 != nil {
			continue
		}

		id := strings.TrimSpace(m[1])

		cameras = append(cameras, Camera{
			ID:          id,
			NameTH:      strings.TrimSpace(m[2]),
			NameEN:      strings.TrimSpace(m[3]),
			DirectionTH: strings.TrimSpace(m[4]),
			DirectionEN: strings.TrimSpace(m[5]),
			Lat:         lat,
			Lon:         lon,
			IP:          strings.TrimSpace(m[8]),
			Icon:        strings.TrimSpace(m[9]),
			FeedURL:     bmaPlayURL + id,
		})
	}

	return cameras
}

func NearestCamera(busLat float64, busLon float64, cameras []Camera) (Camera, float64) {
	best := cameras[0]
	bestDistance := HaversineMeters(busLat, busLon, best.Lat, best.Lon)

	for _, cam := range cameras[1:] {
		d := HaversineMeters(busLat, busLon, cam.Lat, cam.Lon)
		if d < bestDistance {
			best = cam
			bestDistance = d
		}
	}

	return best, bestDistance
}

func HaversineMeters(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	toRad := func(deg float64) float64 {
		return deg * math.Pi / 180.0
	}

	phi1 := toRad(lat1)
	phi2 := toRad(lat2)
	dPhi := toRad(lat2 - lat1)
	dLambda := toRad(lon2 - lon1)

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(dLambda/2)*math.Sin(dLambda/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

func HTTPGet(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 thai-bus-watch/1.0")
	req.Header.Set("Accept", "application/json,text/html,*/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s failed: %w", target, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s returned HTTP %d: %s", target, resp.StatusCode, string(body))
	}

	return body, nil
}
