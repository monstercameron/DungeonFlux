package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // decodes turnaround crops and level stills
	"math"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // decodes build-time UI art
)

// billboardRefMaxSide caps the longest side of an image sent to the video
// model; a 1920 × 1080 PNG level still is ≈ 3 MB, which as a data URI makes
// a request of many megabytes for no gain at 480p.
const billboardRefMaxSide = 1024

// compactReference downscales an image to billboardRefMaxSide, flattens any
// transparency onto light grey, and re-encodes it as JPEG. Undecodable input
// is returned unchanged. The output is deterministic, so the cache key can
// stay on the original bytes.
func compactReference(data []byte) []byte {
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	bounds := source.Bounds()
	scale := math.Min(1, float64(billboardRefMaxSide)/float64(max(bounds.Dx(), bounds.Dy())))
	width, height := max(1, int(math.Round(float64(bounds.Dx())*scale))), max(1, int(math.Round(float64(bounds.Dy())*scale)))
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{R: 200, G: 200, B: 200, A: 255}}, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(canvas, canvas.Bounds(), source, bounds, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, canvas, &jpeg.Options{Quality: 90}); err != nil {
		return data
	}
	return out.Bytes()
}
