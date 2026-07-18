package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestCameraSessionReusesActiveCamera(t *testing.T) {
	camera := newCamSession()
	camera.sessionAt = time.Now()
	camera.lastPlayed["1443"] = time.Now()
	camera.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("active camera unexpectedly repeated setup request to %s", request.URL)
		return nil, nil
	})

	if err := camera.ensure(context.Background(), "1443"); err != nil {
		t.Fatal(err)
	}
}

func TestCameraSessionReopensOnlyIdleCamera(t *testing.T) {
	camera := newCamSession()
	camera.sessionAt = time.Now()
	var requests []string
	camera.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	if err := camera.ensure(context.Background(), "1443"); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || !strings.Contains(requests[0], "PlayVideo.aspx") {
		t.Fatalf("setup requests = %v; want only PlayVideo.aspx", requests)
	}
}

func TestValidFrameExtendsActiveCameraWithoutRepeatingSetup(t *testing.T) {
	camera := newCamSession()
	camera.sessionAt = time.Now()
	camera.lastPlayed["1443"] = time.Now().Add(-44 * time.Second)
	var requests []string
	camera.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", blankFrameSize+1))),
			Header:     make(http.Header),
		}, nil
	})

	if _, err := camera.Frame(context.Background(), "1443"); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || !strings.Contains(requests[0], "show.aspx") {
		t.Fatalf("frame requests = %v; want only show.aspx", requests)
	}
	if time.Since(camera.lastPlayed["1443"]) > time.Second {
		t.Fatal("valid frame did not extend the active camera window")
	}
}

func TestCameraIDExistsInCatalog(t *testing.T) {
	found := false
	for _, camera := range CachedBMACameras() {
		if camera.ID == "1443" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("camera 1443 is missing from catalog")
	}
}

func TestCameraFrameRequestUsesSecureRelay(t *testing.T) {
	frame := strings.Repeat("j", blankFrameSize+1)
	previousClient := bmaRelayClient
	bmaRelayClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/camera/1443/frame" {
			t.Fatalf("relay path = %q", request.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(frame)),
			Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
		}, nil
	})}
	t.Cleanup(func() { bmaRelayClient = previousClient })
	t.Setenv("BMA_CAMERA_RELAY_URL", "https://camera-relay.example")

	got, err := GetCameraFrameForRequest(context.Background(), "1443")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != frame {
		t.Fatalf("relay frame size = %d", len(got))
	}
}

func TestCameraFrameRequestRejectsInsecureRelay(t *testing.T) {
	t.Setenv("BMA_CAMERA_RELAY_URL", "http://camera-relay.example")
	if _, err := GetCameraFrameForRequest(context.Background(), "1443"); err == nil {
		t.Fatal("insecure relay URL was accepted")
	}
}
