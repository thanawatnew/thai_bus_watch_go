package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	watchPollInterval  = 20 * time.Second
	watchDefaultLife   = 90 * time.Minute
	watchMaxLife       = 4 * time.Hour
	watchMaxMisses     = 15 // consecutive polls without the bus before giving up (~5 min)
	keepAliveInterval  = 5 * time.Minute
	liveLocationPeriod = 0x7FFFFFFF // Telegram magic value: live until explicitly stopped

	cameraPassRadiusM = 130              // bus within this distance of a camera → snapshot
	cameraCooldown    = 10 * time.Minute // per camera per watch
)

type AlertPoint struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	RadiusM float64 `json:"radiusM"`
	Label   string  `json:"label"`
}

type BusSnapshot struct {
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	SpeedKmh     float64 `json:"speedKmh"`
	Heading      int     `json:"heading"`
	NextStopName string  `json:"nextStopName"`
	NextStopDist float64 `json:"nextStopDistM"`
	UpdatedAt    int64   `json:"updatedAt"`
}

type Watch struct {
	ID        string      `json:"id"`
	TripID    string      `json:"tripId"`
	BusID     string      `json:"busId"`
	RouteName string      `json:"routeName"`
	Headsign  string      `json:"headsign"`
	Alert     *AlertPoint `json:"alert,omitempty"`
	CreatedAt time.Time   `json:"createdAt"`
	ExpiresAt time.Time   `json:"expiresAt"`

	mu         sync.Mutex
	Status     string       `json:"status"` // active | arrived | lost | expired | stopped
	LastSeen   *BusSnapshot `json:"lastSeen,omitempty"`
	fullBusID  string
	liveMsgID  int64
	alertFired bool
	misses     int
	everSeen   bool
	cameraSent map[string]time.Time // camera ID -> last snapshot sent
	cancel     context.CancelFunc
}

type WatchManager struct {
	tg *Telegram

	mu      sync.RWMutex
	watches map[string]*Watch
	nextID  int
}

func NewWatchManager(tg *Telegram) *WatchManager {
	m := &WatchManager{tg: tg, watches: map[string]*Watch{}}
	go m.keepAliveLoop()
	return m
}

func (m *WatchManager) List() []*Watch {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Watch, 0, len(m.watches))
	for _, w := range m.watches {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (m *WatchManager) activeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, w := range m.watches {
		w.mu.Lock()
		if w.Status == "active" {
			n++
		}
		w.mu.Unlock()
	}
	return n
}

func (m *WatchManager) Start(tripID, busID string, alert *AlertPoint, duration time.Duration) (*Watch, error) {
	if duration <= 0 {
		duration = watchDefaultLife
	}
	if duration > watchMaxLife {
		duration = watchMaxLife
	}

	// Validate against live data before accepting the watch.
	ctx, cancelProbe := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancelProbe()
	trip, err := GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("cannot load trip %s: %w", tripID, err)
	}
	bus, found := FindBus(trip, busID)
	if !found {
		return nil, fmt.Errorf("bus %q is not reporting GPS on this route right now", busID)
	}

	m.mu.Lock()
	m.nextID++
	w := &Watch{
		ID:        strconv.Itoa(m.nextID),
		TripID:    tripID,
		BusID:     busID,
		RouteName: trip.RouteShortName,
		Headsign:  trip.TripHeadsign,
		Alert:     alert,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		Status:    "active",
		fullBusID: bus.ID,
	}
	m.watches[w.ID] = w
	m.mu.Unlock()

	wctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go m.run(wctx, w)

	return w, nil
}

func (m *WatchManager) Stop(id string) bool {
	m.mu.RLock()
	w, ok := m.watches[id]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	w.finish("stopped")
	return true
}

func (w *Watch) finish(status string) {
	w.mu.Lock()
	if w.Status == "active" {
		w.Status = status
	}
	cancel := w.cancel
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (m *WatchManager) run(ctx context.Context, w *Watch) {
	log.Printf("watch %s: started (trip %s, bus %s)", w.ID, w.TripID, w.BusID)

	if m.tg.Ready() {
		alertLine := ""
		if w.Alert != nil {
			alertLine = fmt.Sprintf("\n🔔 You will be alerted when it is within %.0f m of %s.", w.Alert.RadiusM, orDefault(w.Alert.Label, "your pin"))
		}
		_ = m.tg.SendMessage(ctx, fmt.Sprintf(
			"👀 Watching bus %s\nRoute %s → %s%s\n\nA live map pin follows below — tap it to watch the bus move.",
			w.fullBusID, w.RouteName, w.Headsign, alertLine))
	}

	ticker := time.NewTicker(watchPollInterval)
	defer ticker.Stop()

	m.poll(ctx, w)
	for {
		select {
		case <-ctx.Done():
			m.cleanupLive(w)
			return
		case <-ticker.C:
			if time.Now().After(w.ExpiresAt) {
				w.mu.Lock()
				w.Status = "expired"
				w.mu.Unlock()
				if m.tg.Ready() {
					_ = m.tg.SendMessage(ctx, fmt.Sprintf("⏰ Watch for bus %s expired. Start a new one from the app if you still need it.", w.BusID))
				}
				m.cleanupLive(w)
				w.finish("expired")
				return
			}
			if done := m.poll(ctx, w); done {
				m.cleanupLive(w)
				return
			}
		}
	}
}

func (m *WatchManager) cleanupLive(w *Watch) {
	w.mu.Lock()
	msgID := w.liveMsgID
	w.liveMsgID = 0
	w.mu.Unlock()
	if msgID != 0 && m.tg.Ready() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		m.tg.StopLiveLocation(ctx, msgID)
	}
}

// poll fetches the trip once and pushes any due notifications.
// Returns true when the watch has reached a terminal state.
func (m *WatchManager) poll(ctx context.Context, w *Watch) bool {
	tctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	trip, err := GetTrip(tctx, w.TripID)
	if err != nil {
		log.Printf("watch %s: trip fetch failed: %v", w.ID, err)
		return false // transient — try again next tick
	}

	bus, found := FindBus(trip, w.BusID)
	if !found {
		w.mu.Lock()
		w.misses++
		misses := w.misses
		everSeen := w.everSeen
		w.mu.Unlock()

		if misses >= watchMaxMisses {
			if m.tg.Ready() {
				verb := "never appeared"
				if everSeen {
					verb = "stopped reporting GPS (likely finished its run)"
				}
				_ = m.tg.SendMessage(ctx, fmt.Sprintf("🛑 Bus %s %s. Watch ended.", w.BusID, verb))
			}
			w.finish("lost")
			return true
		}
		return false
	}

	lat, lon := bus.BestLat(), bus.BestLon()
	heading := int(bus.SnappedHeading)
	if heading == 0 {
		heading = int(bus.Heading)
	}

	w.mu.Lock()
	w.misses = 0
	w.everSeen = true
	w.LastSeen = &BusSnapshot{
		Lat:          lat,
		Lon:          lon,
		SpeedKmh:     float64(bus.Speed),
		Heading:      heading,
		NextStopName: bus.NextStopName,
		NextStopDist: float64(bus.DistanceToNextStop),
		UpdatedAt:    bus.Received,
	}
	liveMsgID := w.liveMsgID
	alertFired := w.alertFired
	w.mu.Unlock()

	if m.tg.Ready() {
		if liveMsgID == 0 {
			if id, err := m.tg.SendLiveLocation(ctx, lat, lon, heading, liveLocationPeriod); err == nil {
				w.mu.Lock()
				w.liveMsgID = id
				w.mu.Unlock()
			} else {
				log.Printf("watch %s: live location failed: %v", w.ID, err)
			}
		} else {
			if err := m.tg.EditLiveLocation(ctx, liveMsgID, lat, lon, heading); err != nil {
				log.Printf("watch %s: live update failed: %v", w.ID, err)
			}
		}
	}

	go m.checkCameraPass(w, lat, lon, trip.ShapeGeom)

	if w.Alert != nil && !alertFired {
		dist := HaversineMeters(lat, lon, w.Alert.Lat, w.Alert.Lon)
		if dist <= w.Alert.RadiusM {
			w.mu.Lock()
			w.alertFired = true
			w.mu.Unlock()

			if m.tg.Ready() {
				eta := ""
				if bus.Speed > 3 {
					minutes := dist / 1000.0 / float64(bus.Speed) * 60.0
					eta = fmt.Sprintf(" (~%.0f min at current speed)", math.Max(1, minutes))
				}
				_ = m.tg.SendMessage(ctx, fmt.Sprintf(
					"🚍🔔 Bus %s is %.0f m from %s%s!\n\nRoute %s → %s\nNext stop: %s\nSpeed: %.0f km/h\n\nOpen in Maps:\nhttps://maps.google.com/?q=%.6f,%.6f",
					w.fullBusID, dist, orDefault(w.Alert.Label, "your pin"), eta,
					w.RouteName, w.Headsign, orDefault(bus.NextStopName, "-"), float64(bus.Speed),
					lat, lon))
			}

			w.mu.Lock()
			w.Status = "arrived"
			w.mu.Unlock()
			w.finish("arrived")
			return true
		}
	}

	return false
}

// checkCameraPass sends a traffic-camera snapshot to Telegram when the bus
// passes close to a BMA camera, with a local color/shape verdict on whether a
// bus is visible in the frame. Runs in its own goroutine so camera calls never
// delay the tracking loop.
func (m *WatchManager) checkCameraPass(w *Watch, lat, lon float64, shape []LatLon) {
	if !m.tg.Ready() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cameras, err := GetBMACameras(ctx)
	if err != nil || len(cameras) == 0 {
		return
	}
	if routeCameras := CamerasNearShape(cameras, shape, 120); len(routeCameras) > 0 {
		cameras = routeCameras
	}
	w.mu.Lock()
	heading := 0.0
	if w.LastSeen != nil {
		heading = float64(w.LastSeen.Heading)
	}
	w.mu.Unlock()
	cam, dist, ahead := UpcomingCamera(lat, lon, heading, cameras)
	if !ahead {
		return // never send a camera the bus has already passed
	}
	if dist > cameraPassRadiusM {
		return
	}

	w.mu.Lock()
	if w.cameraSent == nil {
		w.cameraSent = map[string]time.Time{}
	}
	if time.Since(w.cameraSent[cam.ID]) < cameraCooldown {
		w.mu.Unlock()
		return
	}
	w.cameraSent[cam.ID] = time.Now()
	w.mu.Unlock()

	frame, err := GetCameraFrame(ctx, cam.IP)
	if err != nil {
		log.Printf("watch %s: camera %s frame failed: %v", w.ID, cam.ID, err)
		return
	}

	caption := fmt.Sprintf("🎥 Bus %s should be passing camera %s now (%.0f m away)\n%s",
		w.BusID, cam.ID, dist, orDefault(cam.NameTH, cam.NameEN))

	if v, err := CheckCameraForBus(frame, w.RouteName, w.Headsign, w.fullBusID); err == nil {
		verdict := map[string]string{"yes": "✅ likely your bus", "no": "❌ doesn't look like your bus", "unsure": "🤔 can't tell"}[v.LikelyMatch]
		if verdict == "" {
			verdict = "🤔 can't tell"
		}
		if !v.BusVisible {
			verdict = "🚫 no bus visible in frame"
		}
		caption += fmt.Sprintf("\n\n🔎 Local check: %s (%.0f%%)\n%s", verdict, v.Confidence*100, v.Description)
	} else {
		log.Printf("watch %s: local camera check failed: %v", w.ID, err)
	}

	if err := m.tg.SendPhoto(ctx, frame, caption); err != nil {
		log.Printf("watch %s: camera photo send failed: %v", w.ID, err)
	}
}

// keepAliveLoop pings our own public URL while watches are active so that
// free-tier hosts (Render) do not spin the service down mid-watch.
func (m *WatchManager) keepAliveLoop() {
	selfURL := os.Getenv("SELF_URL")
	if selfURL == "" {
		selfURL = os.Getenv("RENDER_EXTERNAL_URL")
	}
	if selfURL == "" {
		return
	}

	for range time.Tick(keepAliveInterval) {
		if m.activeCount() == 0 {
			continue
		}
		resp, err := http.Get(selfURL + "/healthz")
		if err != nil {
			log.Printf("keep-alive ping failed: %v", err)
			continue
		}
		resp.Body.Close()
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
