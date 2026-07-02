// One-off generator for the PWA app icons (run: go run ./tools/genicons).
// Draws a simple bus glyph on a rounded amber square with the stdlib only.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

var (
	bg     = color.RGBA{0xF5, 0x9E, 0x0B, 0xFF} // amber
	body   = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	window = color.RGBA{0x10, 0x14, 0x18, 0xFF}
	wheel  = color.RGBA{0x10, 0x14, 0x18, 0xFF}
)

func main() {
	for _, spec := range []struct {
		name string
		size int
	}{
		{"static/icon-192.png", 192},
		{"static/icon-512.png", 512},
		{"static/apple-touch-icon.png", 180},
	} {
		if err := writeIcon(spec.name, spec.size); err != nil {
			log.Fatal(err)
		}
		log.Println("wrote", spec.name)
	}
}

func writeIcon(path string, s int) error {
	img := image.NewRGBA(image.Rect(0, 0, s, s))
	f := func(v float64) int { return int(v * float64(s)) }

	// rounded background
	r := f(0.18)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			if inRoundedRect(x, y, 0, 0, s, s, r) {
				img.Set(x, y, bg)
			}
		}
	}

	// bus body
	bx0, by0, bx1, by1 := f(0.16), f(0.22), f(0.84), f(0.66)
	br := f(0.06)
	for y := by0; y < by1; y++ {
		for x := bx0; x < bx1; x++ {
			if inRoundedRect(x, y, bx0, by0, bx1-bx0, by1-by0, br) {
				img.Set(x, y, body)
			}
		}
	}

	// windows band
	wx0, wy0, wx1, wy1 := f(0.22), f(0.28), f(0.78), f(0.44)
	for y := wy0; y < wy1; y++ {
		for x := wx0; x < wx1; x++ {
			img.Set(x, y, window)
		}
	}

	// wheels
	drawCircle(img, f(0.30), f(0.70), f(0.075), wheel)
	drawCircle(img, f(0.70), f(0.70), f(0.075), wheel)
	drawCircle(img, f(0.30), f(0.70), f(0.032), body)
	drawCircle(img, f(0.70), f(0.70), f(0.032), body)

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

func inRoundedRect(x, y, rx, ry, w, h, r int) bool {
	x -= rx
	y -= ry
	if x < 0 || y < 0 || x >= w || y >= h {
		return false
	}
	cx, cy := x, y
	if x < r {
		cx = r
	} else if x >= w-r {
		cx = w - r - 1
	}
	if y < r {
		cy = r
	} else if y >= h-r {
		cy = h - r - 1
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, c)
			}
		}
	}
}
