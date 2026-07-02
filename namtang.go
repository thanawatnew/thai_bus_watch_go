package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	namtangTripTemplate   = "https://namtang-api.otp.go.th/front/trip/%s?locale=en"
	namtangNearbyTemplate = "https://namtang-api.otp.go.th/front/nearby?lat=%s&lon=%s&locale=en"
)

// FlexFloat unmarshals JSON numbers that arrive either as numbers or strings
// (the Namtang gpsList mixes both, e.g. "lat":"13.79" but "speed":9).
type FlexFloat float64

func (f *FlexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		*f = 0
		return nil
	}
	*f = FlexFloat(v)
	return nil
}

type LatLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Stop struct {
	StopID   int64  `json:"stopId"`
	StopName string `json:"stopName"`
	Location LatLon `json:"location"`
}

type BusGPS struct {
	ID                 string    `json:"id"`
	Lat                FlexFloat `json:"lat"`
	Lon                FlexFloat `json:"lon"`
	Speed              FlexFloat `json:"speed"`
	Heading            FlexFloat `json:"heading"`
	Time               int64     `json:"time"`
	Received           int64     `json:"received"`
	IsReversed         bool      `json:"is_reversed"`
	IsApproachingStop  bool      `json:"is_approaching_stop"`
	SnappedLat         FlexFloat `json:"snapped_lat"`
	SnappedLon         FlexFloat `json:"snapped_lon"`
	SnappedHeading     FlexFloat `json:"snapped_heading"`
	PrevStopName       string    `json:"prev_stop_name"`
	NextStopName       string    `json:"next_stop_name"`
	DistanceToNextStop FlexFloat `json:"distance_to_next_stop"`
}

// BestLat prefers the road-snapped coordinate when available.
func (b BusGPS) BestLat() float64 {
	if b.SnappedLat != 0 {
		return float64(b.SnappedLat)
	}
	return float64(b.Lat)
}

func (b BusGPS) BestLon() float64 {
	if b.SnappedLon != 0 {
		return float64(b.SnappedLon)
	}
	return float64(b.Lon)
}

type Trip struct {
	TripID         int64    `json:"tripId"`
	TripHeadsign   string   `json:"tripHeadsign"`
	RouteShortName string   `json:"routeShortName"`
	RouteLongName  string   `json:"routeLongName"`
	RouteColor     string   `json:"routeColor"`
	StopList       []Stop   `json:"stopList"`
	ShapeGeom      []LatLon `json:"shapeGeom"`
	GpsList        []BusGPS `json:"gpsList"`
}

type namtangEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type PassingTrip struct {
	TripID        int64  `json:"tripId"`
	Name          string `json:"name"`
	RouteLongName string `json:"routeLongName"`
	TripHeadsign  string `json:"tripHeadsign"`
	Color         string `json:"color"`
	AirCondition  bool   `json:"airCondition"`
	HasGps        bool   `json:"hasGps"`
}

type NearbyStop struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	Location     LatLon        `json:"location"`
	PassingTrips []PassingTrip `json:"passingTrips"`
}

func fetchNamtang(ctx context.Context, apiURL string, out any) error {
	body, err := HTTPGet(ctx, apiURL)
	if err != nil {
		return err
	}
	var env namtangEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("cannot parse Namtang response: %w", err)
	}
	if env.Code != 200 {
		return fmt.Errorf("Namtang API error %d: %s", env.Code, env.Message)
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("cannot parse Namtang data: %w", err)
	}
	return nil
}

func GetTrip(ctx context.Context, tripID string) (*Trip, error) {
	var trip Trip
	apiURL := fmt.Sprintf(namtangTripTemplate, url.PathEscape(tripID))
	if err := fetchNamtang(ctx, apiURL, &trip); err != nil {
		return nil, err
	}
	return &trip, nil
}

func GetNearbyStops(ctx context.Context, lat, lon string) ([]NearbyStop, error) {
	var stops []NearbyStop
	apiURL := fmt.Sprintf(namtangNearbyTemplate, url.QueryEscape(lat), url.QueryEscape(lon))
	if err := fetchNamtang(ctx, apiURL, &stops); err != nil {
		return nil, err
	}
	return stops, nil
}

// FindBus locates a bus in a trip's gpsList by fuzzy plate match,
// e.g. "11-9612" matches "11-9612 กรุงเทพมหานคร".
func FindBus(trip *Trip, wantedBus string) (BusGPS, bool) {
	wanted := normalizeBusID(wantedBus)
	for _, item := range trip.GpsList {
		if wanted != "" && strings.Contains(normalizeBusID(item.ID), wanted) {
			return item, true
		}
	}
	return BusGPS{}, false
}

func normalizeBusID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	return s
}
