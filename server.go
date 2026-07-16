package main

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed static
var staticFS embed.FS

type Server struct {
	tg      *Telegram
	watches *WatchManager
}

func NewServer(tg *Telegram) *Server {
	return &Server{tg: tg, watches: NewWatchManager(tg)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /api/camera/{id}/frame", s.handleCameraFrame)
	mux.HandleFunc("POST /api/telegram/webhook", s.handleTelegramWebhook)
	mux.HandleFunc("GET /api/trip/{id}", s.handleTrip)
	mux.HandleFunc("GET /api/nearby", s.handleNearby)
	mux.HandleFunc("GET /api/passing/{id}", s.handlePassingTrips)
	mux.HandleFunc("GET /api/arrivals", s.handleArrivals)
	mux.HandleFunc("GET /api/bus", s.handleBus)
	mux.HandleFunc("GET /api/telegram/status", s.handleTelegramStatus)
	mux.HandleFunc("GET /api/watch", s.handleWatchList)
	mux.HandleFunc("POST /api/watch", s.handleWatchCreate)
	mux.HandleFunc("DELETE /api/watch/{id}", s.handleWatchDelete)

	return mux
}

func (s *Server) handlePassingTrips(w http.ResponseWriter, r *http.Request) {
	stopID := strings.TrimSpace(r.PathValue("id"))
	if stopID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "stop id is required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	trips, err := GetPassingTrips(ctx, stopID)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, trips)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func (s *Server) handleTrip(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	trip, err := GetTrip(ctx, r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, trip)
}

func (s *Server) handleNearby(w http.ResponseWriter, r *http.Request) {
	lat := strings.TrimSpace(r.URL.Query().Get("lat"))
	lon := strings.TrimSpace(r.URL.Query().Get("lon"))
	if lat == "" || lon == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lat and lon are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	stops, err := GetNearbyStops(ctx, lat, lon)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, stops)
}

func (s *Server) handleArrivals(w http.ResponseWriter, r *http.Request) {
	stopID := strings.TrimSpace(r.URL.Query().Get("stop"))
	tripID := strings.TrimSpace(r.URL.Query().Get("trip"))
	if stopID == "" || tripID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "stop and trip are required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	trips, err := GetPassingTrips(ctx, stopID)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	for _, trip := range trips {
		if strconv.FormatInt(trip.TripID, 10) == tripID {
			writeJSON(w, http.StatusOK, trip)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "trip does not serve this stop"})
}

// handleBus returns one bus's live position plus the nearest BMA traffic camera.
func (s *Server) handleBus(w http.ResponseWriter, r *http.Request) {
	tripID := strings.TrimSpace(r.URL.Query().Get("trip"))
	busID := strings.TrimSpace(r.URL.Query().Get("bus"))
	if tripID == "" || busID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "trip and bus are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	trip, err := GetTrip(ctx, tripID)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}

	bus, found := FindBus(trip, busID)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bus not found on this trip"})
		return
	}

	resp := map[string]any{
		"bus":       bus,
		"routeName": trip.RouteShortName,
		"headsign":  trip.TripHeadsign,
	}

	cameras := CachedBMACameras()
	if len(cameras) == 0 {
		RefreshBMACamerasAsync()
	}
	if len(cameras) > 0 {
		routeCameras := CamerasNearShape(cameras, trip.ShapeGeom, 120)
		if len(routeCameras) == 0 {
			routeCameras = cameras
		}
		heading := float64(bus.SnappedHeading)
		if heading == 0 {
			heading = float64(bus.Heading)
		}
		cam, dist, ahead := UpcomingCamera(bus.BestLat(), bus.BestLon(), heading, routeCameras)
		if !ahead {
			cam, dist = NearestCamera(bus.BestLat(), bus.BestLon(), routeCameras)
		}
		resp["nearestCamera"] = cam
		resp["cameraDistanceM"] = math.Round(dist)
		resp["cameraSelection"] = map[bool]string{true: "ahead", false: "nearest"}[ahead]
		resp["cameraOnRoute"] = len(CamerasNearShape([]Camera{cam}, trip.ShapeGeom, 120)) == 1
		type cameraCandidate struct {
			Camera    Camera  `json:"camera"`
			DistanceM float64 `json:"distanceM"`
		}
		candidates := make([]cameraCandidate, 0)
		for _, candidate := range routeCameras {
			if c, d, ok := UpcomingCamera(bus.BestLat(), bus.BestLon(), heading, []Camera{candidate}); ok {
				candidates = append(candidates, cameraCandidate{Camera: c, DistanceM: math.Round(d)})
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].DistanceM < candidates[j].DistanceM })
		resp["cameraCandidates"] = candidates
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.tg.Status())
}

// handleCameraFrame proxies a live JPEG frame from bmatraffic.com so the
// HTTPS web app can show camera video (the source site is HTTP-only and
// session-bound, which iPhones refuse to load directly).
func (s *Server) handleCameraFrame(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	frame, err := GetCameraFrame(ctx, r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(frame)
}

func (s *Server) handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if !s.tg.Configured() ||
		r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != s.tg.WebhookSecret() {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var update tgUpdate
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&update); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.tg.ProcessUpdate(r.Context(), update)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleWatchList(w http.ResponseWriter, r *http.Request) {
	list := s.watches.List()
	out := make([]map[string]any, 0, len(list))
	for _, wa := range list {
		wa.mu.Lock()
		out = append(out, map[string]any{
			"id":        wa.ID,
			"tripId":    wa.TripID,
			"busId":     wa.BusID,
			"routeName": wa.RouteName,
			"headsign":  wa.Headsign,
			"alert":     wa.Alert,
			"status":    wa.Status,
			"lastSeen":  wa.LastSeen,
			"createdAt": wa.CreatedAt.Unix(),
			"expiresAt": wa.ExpiresAt.Unix(),
		})
		wa.mu.Unlock()
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleWatchCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TripID      string      `json:"tripId"`
		BusID       string      `json:"busId"`
		Alert       *AlertPoint `json:"alert"`
		DurationMin int         `json:"durationMin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.TripID) == "" || strings.TrimSpace(req.BusID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tripId and busId are required"})
		return
	}
	if req.Alert != nil && req.Alert.RadiusM <= 0 {
		req.Alert.RadiusM = 500
	}
	if !s.tg.Ready() {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "telegram is not connected yet — open your bot chat and send /start first",
		})
		return
	}

	watch, err := s.watches.Start(req.TripID, req.BusID, req.Alert, time.Duration(req.DurationMin)*time.Minute)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": watch.ID, "status": "active"})
}

func (s *Server) handleWatchDelete(w http.ResponseWriter, r *http.Request) {
	if !s.watches.Stop(r.PathValue("id")) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "watch not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}
