package main

import "testing"

func TestUpcomingCameraRejectsPassedCamera(t *testing.T) {
	busLat, busLon := 13.76817, 100.50230
	cameras := []Camera{
		{ID: "behind", Lat: 13.76854, Lon: 100.50265}, // north-east
		{ID: "ahead", Lat: 13.76780, Lon: 100.50195},  // south-west
	}
	cam, _, ok := UpcomingCamera(busLat, busLon, 224, cameras)
	if !ok || cam.ID != "ahead" {
		t.Fatalf("got camera %q, ok=%v; want ahead", cam.ID, ok)
	}
}

func TestInitialBearing(t *testing.T) {
	if got := InitialBearing(0, 0, 1, 0); got < 359 && got > 1 {
		t.Fatalf("north bearing = %f", got)
	}
	if got := InitialBearing(0, 0, 0, 1); got < 89 || got > 91 {
		t.Fatalf("east bearing = %f", got)
	}
}

func TestUpcomingCameraDropsPassedCamera(t *testing.T) {
	// The bus is travelling south and the camera is behind it to the north.
	cam := Camera{ID: "just-passed", Lat: 13.70005, Lon: 100.5}
	if got, _, ok := UpcomingCamera(13.70000, 100.5, 180, []Camera{cam}); ok {
		t.Fatalf("passed camera should be dropped for timed UI handoff, got %q", got.ID)
	}
}

func TestCamerasNearShape(t *testing.T) {
	shape := []LatLon{{Lat: 13.7, Lon: 100.5}, {Lat: 13.71, Lon: 100.5}}
	cameras := []Camera{{ID: "route", Lat: 13.705, Lon: 100.5001}, {ID: "cross", Lat: 13.705, Lon: 100.503}}
	got := CamerasNearShape(cameras, shape, 120)
	if len(got) != 1 || got[0].ID != "route" {
		t.Fatalf("route cameras = %+v", got)
	}
}
