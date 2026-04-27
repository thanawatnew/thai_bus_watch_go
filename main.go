package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	namtangTripTemplate = "https://namtang-api.otp.go.th/front/trip/%s?locale=en"
	bmaIndexURL         = "http://www.bmatraffic.com/index.aspx"
	bmaPlayURL          = "http://www.bmatraffic.com/PlayVideo.aspx?ID="
)

type BusLocation struct {
	TripID    string  `json:"trip_id"`
	BusNumber string  `json:"bus_number"`
	FullID    string  `json:"full_id"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Raw       any     `json:"raw,omitempty"`
}

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

type Result struct {
	Bus            BusLocation `json:"bus"`
	NearestCamera  Camera      `json:"nearest_camera"`
	DistanceMeters float64     `json:"distance_meters"`
	TelegramSent   bool        `json:"telegram_sent"`
}

func main() {
	tripID := flag.String("trip-id", "", "Namtang trip ID, example: 7179")
	busNumber := flag.String("bus", "", "bus number, example: 11-9253")
	telegramToken := flag.String("telegram-token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token")
	telegramChatID := flag.String("telegram-chat-id", os.Getenv("TELEGRAM_CHAT_ID"), "Telegram chat id")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)

	if strings.TrimSpace(*tripID) == "" {
		fmt.Print("Enter Namtang trip ID, example 7179: ")
		text, _ := reader.ReadString('\n')
		*tripID = strings.TrimSpace(text)
	}

	if strings.TrimSpace(*busNumber) == "" {
		fmt.Print("Enter bus number, example 11-9253: ")
		text, _ := reader.ReadString('\n')
		*busNumber = strings.TrimSpace(text)
	}

	if *tripID == "" || *busNumber == "" {
		fmt.Fprintln(os.Stderr, "missing trip ID or bus number")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namtangURL := fmt.Sprintf(namtangTripTemplate, url.PathEscape(*tripID))

	bus, err := GetBusLocationFromTrip(ctx, namtangURL, *tripID, *busNumber)
	if err != nil {
		fail(err)
	}

	cameras, err := GetBMACameras(ctx)
	if err != nil {
		fail(err)
	}

	camera, distance := NearestCamera(bus.Lat, bus.Lon, cameras)

	msg := BuildMessage(bus, camera, distance)

	result := Result{
		Bus:            bus,
		NearestCamera:  camera,
		DistanceMeters: math.Round(distance),
	}

	if strings.TrimSpace(*telegramToken) != "" && strings.TrimSpace(*telegramChatID) != "" {
		if err := SendTelegramMessage(ctx, *telegramToken, *telegramChatID, msg); err != nil {
			fail(err)
		}
		result.TelegramSent = true
	} else {
		fmt.Println()
		fmt.Println("Telegram env not set, printing result only:")
		fmt.Println("--------------------------------------------------")
		fmt.Println(msg)
		fmt.Println("--------------------------------------------------")
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}

func GetBusLocationFromTrip(ctx context.Context, apiURL string, tripID string, wantedBus string) (BusLocation, error) {
	body, err := HTTPGet(ctx, apiURL)
	if err != nil {
		return BusLocation{}, err
	}

	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return BusLocation{}, fmt.Errorf("cannot parse Namtang trip JSON: %w", err)
	}

	gpsItems := FindGPSListItems(root)
	if len(gpsItems) == 0 {
		return BusLocation{}, errors.New("cannot find gpsList in Namtang trip JSON")
	}

	wanted := normalizeBusID(wantedBus)
	available := []string{}

	for _, item := range gpsItems {
		id := getStringAny(item, "id", "bus_id", "vehicle_id", "license", "license_plate", "plate")
		if id != "" {
			available = append(available, id)
		}

		if !strings.Contains(normalizeBusID(id), wanted) {
			continue
		}

		lat, okLat := getFloatAny(item, "lat", "latitude")
		lon, okLon := getFloatAny(item, "lon", "lng", "longitude")

		if !okLat || !okLon {
			return BusLocation{}, fmt.Errorf("found bus %q but cannot parse lat/lon", wantedBus)
		}

		return BusLocation{
			TripID:    tripID,
			BusNumber: wantedBus,
			FullID:    id,
			Lat:       lat,
			Lon:       lon,
			Raw:       item,
		}, nil
	}

	return BusLocation{}, fmt.Errorf(
		"bus %q not found in trip/%s gpsList\navailable buses:\n%s",
		wantedBus,
		tripID,
		strings.Join(uniqueFirstN(available, 40), "\n"),
	)
}

func FindGPSListItems(root any) []map[string]any {
	var found []map[string]any

	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
			case map[string]any:
				for k, child := range x {
					if strings.EqualFold(k, "gpsList") {
						if arr, ok := child.([]any); ok {
							for _, item := range arr {
								if m, ok := item.(map[string]any); ok {
									found = append(found, m)
								}
							}
						}
					}
					walk(child)
				}

			case []any:
				for _, child := range x {
					walk(child)
				}
		}
	}

	walk(root)
	return found
}

func GetBMACameras(ctx context.Context) ([]Camera, error) {
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

func BuildMessage(bus BusLocation, cam Camera, distance float64) string {
	return fmt.Sprintf(
		"🚌 Bus found\n\n"+
		"Trip ID: %s\n"+
		"Bus number: %s\n"+
		"Full ID: %s\n\n"+
		"📍 Bus GPS\n"+
		"Lat: %.6f\n"+
		"Lon: %.6f\n\n"+
		"🎥 Nearest BMA camera\n"+
		"Camera ID: %s\n"+
		"Name: %s\n"+
		"Direction: %s\n"+
		"Camera lat: %.6f\n"+
		"Camera lon: %.6f\n"+
		"Distance: %.0f meters\n\n"+
		"Camera feed:\n%s",
		bus.TripID,
		bus.BusNumber,
		bus.FullID,
		bus.Lat,
		bus.Lon,
		cam.ID,
		emptyDash(cam.NameTH),
			   emptyDash(cam.DirectionTH),
			   cam.Lat,
		    cam.Lon,
		    math.Round(distance),
			   cam.FeedURL,
	)
}

func SendTelegramMessage(ctx context.Context, token string, chatID string, text string) error {
	apiURL := "https://api.telegram.org/bot" + token + "/sendMessage"

	form := url.Values{}
	form.Set("chat_id", chatID)
		form.Set("text", text)
			form.Set("disable_web_page_preview", "false")

				req, err := http.NewRequestWithContext(
					ctx,
					http.MethodPost,
					apiURL,
					strings.NewReader(form.Encode()),
				)
				if err != nil {
					return err
				}

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()

				body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					return fmt.Errorf("Telegram API error HTTP %d: %s", resp.StatusCode, string(body))
				}

				return nil
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

func getStringAny(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch x := v.(type) {
				case string:
					return strings.TrimSpace(x)
				case float64:
					return strconv.FormatFloat(x, 'f', -1, 64)
				case int:
					return strconv.Itoa(x)
			}
		}
	}
	return ""
}

func getFloatAny(m map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch x := v.(type) {
				case float64:
					return x, true
				case string:
					f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
					return f, err == nil
				case int:
					return float64(x), true
			}
		}
	}
	return 0, false
}

func normalizeBusID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func uniqueFirstN(items []string, n int) []string {
	seen := map[string]bool{}
	out := []string{}

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)

		if len(out) >= n {
			break
		}
	}

	if len(out) == 0 {
		return []string{"no buses found in gpsList"}
	}

	return out
}

func emptyDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
