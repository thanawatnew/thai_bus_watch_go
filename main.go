package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

func main() {
	tripID := flag.String("trip-id", "", "one-shot CLI mode: Namtang trip ID, example: 7179")
	busNumber := flag.String("bus", "", "one-shot CLI mode: bus number, example: 11-9253")
	cameraSnapshot := flag.String("write-camera-snapshot", "", "write the current BMA camera catalog to this JSON file")
	cameraRelay := flag.Bool("camera-relay", false, "serve only validated BMA camera frames")
	addr := flag.String("addr", "", "listen address for server mode (default :8080 or $PORT)")
	flag.Parse()
	if *cameraSnapshot != "" {
		writeCameraSnapshot(*cameraSnapshot)
		return
	}

	if *tripID != "" || *busNumber != "" {
		runCLI(*tripID, *busNumber)
		return
	}

	listen := *addr
	if listen == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		listen = ":" + port
	}
	if *cameraRelay {
		log.Printf("BUS287 camera relay listening on %s", listen)
		log.Fatal(http.ListenAndServe(listen, NewCameraRelayHandler()))
	}

	publicURL := os.Getenv("SELF_URL")
	if publicURL == "" {
		publicURL = os.Getenv("RENDER_EXTERNAL_URL")
	}

	tg := NewTelegram(os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"))
	if tg.Configured() {
		go tg.Init(context.Background(), publicURL)
	} else {
		log.Println("TELEGRAM_BOT_TOKEN not set — notifications disabled, map still works")
	}

	srv := NewServer(tg)
	log.Printf("Thai Bus Watch listening on %s", listen)
	log.Fatal(http.ListenAndServe(listen, srv.Handler()))
}

func writeCameraSnapshot(path string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cameras, err := fetchBMACameras(ctx)
	if err != nil {
		fail(err)
	}
	body, err := json.MarshalIndent(cameras, "", "  ")
	if err != nil {
		fail(err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %d cameras to %s\n", len(cameras), path)
}

// runCLI preserves the original one-shot behaviour of this tool:
// look up one bus once, print it together with the nearest BMA camera.
func runCLI(tripID, busNumber string) {
	if tripID == "" || busNumber == "" {
		fmt.Fprintln(os.Stderr, "CLI mode needs both -trip-id and -bus")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trip, err := GetTrip(ctx, tripID)
	if err != nil {
		fail(err)
	}

	bus, found := FindBus(trip, busNumber)
	if !found {
		available := make([]string, 0, len(trip.GpsList))
		for _, b := range trip.GpsList {
			available = append(available, b.ID)
		}
		fmt.Fprintf(os.Stderr, "bus %q not found in trip/%s gpsList\navailable buses:\n", busNumber, tripID)
		for _, a := range available {
			fmt.Fprintln(os.Stderr, a)
		}
		os.Exit(1)
	}

	cameras, err := GetBMACameras(ctx)
	if err != nil {
		fail(err)
	}

	camera, distance := NearestCamera(bus.BestLat(), bus.BestLon(), cameras)

	fmt.Printf(
		"🚌 Bus found\n\nTrip ID: %s\nRoute: %s → %s\nFull ID: %s\n\n📍 Bus GPS\nLat: %.6f\nLon: %.6f\nSpeed: %.0f km/h\nNext stop: %s\n\n🎥 Nearest BMA camera\nCamera ID: %s\nName: %s\nDistance: %.0f meters\n\nCamera feed:\n%s\n\n",
		tripID, trip.RouteShortName, trip.TripHeadsign, bus.ID,
		bus.BestLat(), bus.BestLon(), float64(bus.Speed), orDefault(bus.NextStopName, "-"),
		camera.ID, orDefault(camera.NameTH, "-"), math.Round(distance), camera.FeedURL,
	)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{
		"bus":             bus,
		"nearest_camera":  camera,
		"distance_meters": math.Round(distance),
	})
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
