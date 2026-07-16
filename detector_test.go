package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func cameraJPEG(t *testing.T, bus color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 352, 288))
	for y := 0; y < 288; y++ {
		for x := 0; x < 352; x++ {
			img.Set(x, y, color.RGBA{70, 70, 70, 255})
		}
	}
	if bus != nil {
		for y := 145; y < 225; y++ {
			for x := 105; x < 245; x++ {
				img.Set(x, y, bus)
			}
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestCheckCameraForBus(t *testing.T) {
	t.Run("orange bus", func(t *testing.T) {
		v, err := CheckCameraForBus(cameraJPEG(t, color.RGBA{230, 100, 20, 255}), "8", "", "12-3456")
		if err != nil {
			t.Fatal(err)
		}
		if !v.BusVisible || v.Color != "orange" {
			t.Fatalf("unexpected verdict: %+v", v)
		}
	})
	t.Run("empty road", func(t *testing.T) {
		v, err := CheckCameraForBus(cameraJPEG(t, nil), "8", "", "12-3456")
		if err != nil {
			t.Fatal(err)
		}
		if v.BusVisible {
			t.Fatalf("unexpected verdict: %+v", v)
		}
	})
	t.Run("invalid image", func(t *testing.T) {
		if _, err := CheckCameraForBus([]byte("no image"), "", "", ""); err == nil {
			t.Fatal("expected decode error")
		}
	})
}
