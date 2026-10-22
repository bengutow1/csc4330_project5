// Author: Caden Lund
package demo

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
)

// The demo's plate photos are drawn, not shot: a plate on a table with its food on it. Each one's
// food sits in its own color and place, so no two photos are the same JPEG, and app.plate's
// unique photo hash never sees one twice.
const (
	photoSide   = 96
	plateRadius = 40
	foodRadius  = 22
)

var (
	table = color.RGBA{R: 0xC8, G: 0x9F, B: 0x73, A: 0xFF}
	china = color.RGBA{R: 0xFB, G: 0xF8, B: 0xF4, A: 0xFF}
)

// platePhoto is the nth plate's photo as a JPEG.
func platePhoto(n int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, photoSide, photoSide))
	food := color.RGBA{R: uint8(120 + n*37%120), G: uint8(60 + n*53%120), B: uint8(30 + n*29%80), A: 0xFF}
	// The food wanders a little around the middle of the plate.
	fx, fy := photoSide/2+n%7-3, photoSide/2+n%5-2
	for y := range photoSide {
		for x := range photoSide {
			c := table
			if within(x, y, photoSide/2, photoSide/2, plateRadius) {
				c = china
			}
			if within(x, y, fx, fy, foodRadius) {
				c = food
			}
			img.SetRGBA(x, y, c)
		}
	}
	var out bytes.Buffer
	err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("draw plate photo %d: %w", n, err)
	}
	return out.Bytes(), nil
}

// within says whether (x, y) falls inside the circle of radius r around (cx, cy).
func within(x, y, cx, cy, r int) bool {
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}
