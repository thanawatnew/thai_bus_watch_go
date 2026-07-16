package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed static/cameras.json
var bundledCamerasJSON []byte

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
	cameras     []Camera
	fetchedAt   time.Time
	attemptedAt time.Time
	refreshing  bool
}

const bmaRetryBackoff = 5 * time.Minute

func init() {
	// Oracle/Render IP ranges cannot reliably reach bmatraffic.com. A bundled
	// catalog keeps location matching available; live frames still come from
	// BMA and are loaded directly by the user's browser.
	_ = json.Unmarshal(bundledCamerasJSON, &cameraCache.cameras)
	normalizeCameraFeeds(cameraCache.cameras)
}

func normalizeCameraFeeds(cameras []Camera) {
	for i := range cameras {
		// Public camera links expose only BMA's numeric camera ID. The internal
		// camera address remains available to server-side frame handling only.
		if strings.TrimSpace(cameras[i].ID) != "" {
			cameras[i].FeedURL = bmaPlayURL + url.QueryEscape(cameras[i].ID)
		}
	}
}

// CachedBMACameras returns immediately. Bus-detail HTTP requests use this so a
// slow or unreachable BMA site can never hold up the rest of the UI.
func CachedBMACameras() []Camera {
	cameraCache.Lock()
	defer cameraCache.Unlock()
	return append([]Camera(nil), cameraCache.cameras...)
}

// RefreshBMACamerasAsync refreshes the rarely-changing camera list without
// delaying a browser response. Failed attempts are backed off because some
// cloud networks cannot reach bmatraffic.com reliably.
func RefreshBMACamerasAsync() {
	cameraCache.Lock()
	if cameraCache.refreshing || time.Since(cameraCache.attemptedAt) < bmaRetryBackoff {
		cameraCache.Unlock()
		return
	}
	cameraCache.refreshing = true
	cameraCache.attemptedAt = time.Now()
	cameraCache.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		cameras, err := fetchBMACameras(ctx)

		cameraCache.Lock()
		defer cameraCache.Unlock()
		cameraCache.refreshing = false
		if err == nil && len(cameras) > 0 {
			cameraCache.cameras = cameras
			cameraCache.fetchedAt = time.Now()
		}
	}()
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
	cameraCache.attemptedAt = time.Now()

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
			FeedURL:     bmaPlayURL + url.QueryEscape(id),
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

// UpcomingCamera returns the closest camera in front of the vehicle. Heading
// follows the GPS convention: 0=north, 90=east. A generous 70-degree cone
// tolerates curved roads and noisy low-speed headings. A camera within 2 m is
// treated as alongside the bus; the browser provides the post-pass time buffer.
func UpcomingCamera(busLat, busLon, heading float64, cameras []Camera) (Camera, float64, bool) {
	if len(cameras) == 0 || math.IsNaN(heading) {
		return Camera{}, 0, false
	}
	heading = math.Mod(heading+360, 360)
	bestDistance := math.Inf(1)
	var best Camera
	for _, cam := range cameras {
		d := HaversineMeters(busLat, busLon, cam.Lat, cam.Lon)
		if d > 5000 {
			continue
		}
		bearing := InitialBearing(busLat, busLon, cam.Lat, cam.Lon)
		delta := math.Abs(bearing - heading)
		if delta > 180 {
			delta = 360 - delta
		}
		if (d <= 2 || delta <= 70) && d < bestDistance {
			best, bestDistance = cam, d
		}
	}
	if math.IsInf(bestDistance, 1) {
		return Camera{}, 0, false
	}
	return best, bestDistance, true
}

// CamerasNearShape removes cameras that the route does not actually pass.
// This avoids selecting a nearby camera on a crossing road.
func CamerasNearShape(cameras []Camera, shape []LatLon, maxDistanceM float64) []Camera {
	if len(shape) < 2 {
		return cameras
	}
	out := make([]Camera, 0, len(cameras))
	for _, cam := range cameras {
		best := math.Inf(1)
		for i := 1; i < len(shape); i++ {
			d := pointSegmentMeters(cam.Lat, cam.Lon, shape[i-1].Lat, shape[i-1].Lon, shape[i].Lat, shape[i].Lon)
			if d < best {
				best = d
			}
		}
		if best <= maxDistanceM {
			out = append(out, cam)
		}
	}
	return out
}

func pointSegmentMeters(pLat, pLon, aLat, aLon, bLat, bLon float64) float64 {
	const metersPerDegree = 111195.0
	refLat := (pLat + aLat + bLat) / 3 * math.Pi / 180
	x := func(lon float64) float64 { return lon * metersPerDegree * math.Cos(refLat) }
	y := func(lat float64) float64 { return lat * metersPerDegree }
	px, py, ax, ay, bx, by := x(pLon), y(pLat), x(aLon), y(aLat), x(bLon), y(bLat)
	dx, dy := bx-ax, by-ay
	t := 0.0
	if denom := dx*dx + dy*dy; denom > 0 {
		t = ((px-ax)*dx + (py-ay)*dy) / denom
	}
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

func InitialBearing(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	phi1, phi2 := toRad(lat1), toRad(lat2)
	dLon := toRad(lon2 - lon1)
	y := math.Sin(dLon) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLon)
	return math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360)
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
