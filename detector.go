package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

// CameraVerdict is deliberately conservative: the camera image can confirm
// that a bus-like vehicle is present, but it cannot read a plate reliably.
// The live Namtang GPS feed remains the source of the bus identity.
type CameraVerdict struct {
	BusVisible  bool    `json:"bus_visible"`
	LikelyMatch string  `json:"likely_match"` // yes | unsure
	Description string  `json:"description"`
	Color       string  `json:"color,omitempty"`
	Confidence  float64 `json:"confidence"`
}

type hsvRange struct {
	name                               string
	hMin, hMax, sMin, sMax, vMin, vMax float64
}

// The first three ranges come from WIMB's OpenCV HSV masks. The remaining
// ranges cover other common Bangkok transit liveries. Hue is in degrees.
var busColorRanges = []hsvRange{
	{"orange", 14, 24, .45, 1, .40, 1},
	{"yellow", 46, 86, .25, 1, .38, 1},
	{"blue", 204, 224, .22, 1, .35, 1},
	{"red", 345, 15, .40, 1, .32, 1},
	{"green", 85, 165, .28, 1, .28, 1},
	{"pink", 315, 344, .25, 1, .40, 1},
}

// CheckCameraForBus performs an entirely local color/shape check inspired by
// thanawatnew/wimb. It makes no network calls and needs no API key.
func CheckCameraForBus(frame []byte, routeName, headsign, busID string) (*CameraVerdict, error) {
	img, _, err := image.Decode(bytes.NewReader(frame))
	if err != nil {
		return nil, fmt.Errorf("decode camera frame: %w", err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 32 || h < 32 {
		return nil, fmt.Errorf("camera frame is too small: %dx%d", w, h)
	}

	// Evaluate connected components separately per livery color. Restricting
	// the top of the image removes sky/sign noise while retaining distant buses.
	bestScore, bestColor := 0.0, ""
	for _, r := range busColorRanges {
		mask := make([]bool, w*h)
		for y := h / 8; y < h; y++ {
			for x := 0; x < w; x++ {
				rr, gg, bb, aa := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				hh, ss, vv := rgbToHSV(rr, gg, bb, aa)
				if hsvInRange(hh, ss, vv, r) {
					mask[y*w+x] = true
				}
			}
		}
		// A one-pixel close joins compression-broken paint panels.
		mask = closeMask(mask, w, h)
		if score := largestBusLikeComponent(mask, w, h); score > bestScore {
			bestScore, bestColor = score, r.name
		}
	}

	visible := bestScore >= .42
	confidence := math.Min(.95, bestScore)
	if !visible {
		return &CameraVerdict{false, "unsure", "No clear bus-shaped color region was detected in this frame.", "", confidence}, nil
	}
	description := fmt.Sprintf("A %s bus-like vehicle is visible; GPS places bus %s at this camera.", bestColor, busID)
	return &CameraVerdict{true, "yes", description, bestColor, confidence}, nil
}

func rgbToHSV(r16, g16, b16, _ uint32) (h, s, v float64) {
	r, g, b := float64(r16)/65535, float64(g16)/65535, float64(b16)/65535
	maxv, minv := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := maxv - minv
	v = maxv
	if maxv != 0 {
		s = d / maxv
	}
	if d == 0 {
		return 0, s, v
	}
	switch maxv {
	case r:
		h = 60 * math.Mod((g-b)/d, 6)
	case g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}
	if h < 0 {
		h += 360
	}
	return
}

func hsvInRange(h, s, v float64, r hsvRange) bool {
	hue := h >= r.hMin && h <= r.hMax
	if r.hMin > r.hMax {
		hue = h >= r.hMin || h <= r.hMax
	}
	return hue && s >= r.sMin && s <= r.sMax && v >= r.vMin && v <= r.vMax
}

func closeMask(src []bool, w, h int) []bool {
	dilated := make([]bool, len(src))
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			for dy := -1; dy <= 1 && !dilated[y*w+x]; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if src[(y+dy)*w+x+dx] {
						dilated[y*w+x] = true
						break
					}
				}
			}
		}
	}
	out := make([]bool, len(src))
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			out[y*w+x] = true
			for dy := -1; dy <= 1 && out[y*w+x]; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if !dilated[(y+dy)*w+x+dx] {
						out[y*w+x] = false
						break
					}
				}
			}
		}
	}
	return out
}

func largestBusLikeComponent(mask []bool, w, h int) float64 {
	seen, queue := make([]bool, len(mask)), make([]int, 0, 1024)
	best := 0.0
	for start, on := range mask {
		if !on || seen[start] {
			continue
		}
		seen[start], queue = true, append(queue[:0], start)
		count, minX, maxX := 0, w, 0
		minY, maxY := h, 0
		for len(queue) > 0 {
			i := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			x, y := i%w, i/w
			count++
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
			for _, n := range []int{i - 1, i + 1, i - w, i + w} {
				if n >= 0 && n < len(mask) && mask[n] && !seen[n] && (n/w == y || n%w == x) {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
		bw, bh := maxX-minX+1, maxY-minY+1
		areaRatio := float64(bw*bh) / float64(w*h)
		density, aspect := float64(count)/float64(bw*bh), float64(bw)/float64(bh)
		if areaRatio < .006 || areaRatio > .45 || aspect < .45 || aspect > 4.2 || density < .18 {
			continue
		}
		// A bus-sized, reasonably rectangular, lower-frame region scores highly.
		size := math.Min(1, areaRatio/.045)
		shape := math.Min(1, density/.55)
		position := .65 + .35*float64(maxY)/float64(h)
		score := size * shape * position
		if score > best {
			best = score
		}
	}
	return best
}
