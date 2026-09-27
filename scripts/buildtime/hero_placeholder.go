package main

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
)

// heroPlaceholderName is the neutral figure the creation card shows before a
// player picks a gender: every default hero blended, darkened and blurred
// until only a presence in the dark remains. The entrance clips start from
// the same darkness, so the card never cuts to black between the two.
const heroPlaceholderName = "hero_default_placeholder"

func renderHeroPlaceholder(root string, portraits [][]byte, out io.Writer) error {
	data, err := heroPlaceholder(portraits)
	if err != nil {
		return err
	}
	key := heroIntroKey(append([][]byte{[]byte("placeholder")}, portraits...)...)
	if _, ok := cachedHeroAsset(root, heroPlaceholderName, key); ok {
		report(out, heroPlaceholderName, true)
		return nil
	}
	if err := registerHeroAsset(root, heroPlaceholderName, "IMAGE_STILL", ".png", data, 0, map[string]string{"cache_key": key, "method": "blend of default heroes, 9% brightness, cool tint, 3x box blur r14", "prompt_version": heroIntroVersion}); err != nil {
		return err
	}
	report(out, heroPlaceholderName, false)
	return nil
}

func heroPlaceholder(portraits [][]byte) ([]byte, error) {
	var imgs []image.Image
	for _, p := range portraits {
		if len(p) == 0 {
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(p))
		if err != nil {
			return nil, err
		}
		imgs = append(imgs, img)
	}
	if len(imgs) == 0 {
		return nil, errors.New("herointros: no portraits to build the placeholder from")
	}
	b := imgs[0].Bounds()
	w, h := b.Dx(), b.Dy()
	ch := [3][]float64{make([]float64, w*h), make([]float64, w*h), make([]float64, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum [3]float64
			for _, img := range imgs {
				r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				sum[0] += float64(r >> 8)
				sum[1] += float64(g >> 8)
				sum[2] += float64(bl >> 8)
			}
			for c := range ch {
				ch[c][y*w+x] = sum[c] / float64(len(imgs))
			}
		}
	}
	for c := range ch {
		for pass := 0; pass < 3; pass++ {
			ch[c] = boxBlur(ch[c], w, h, 14)
		}
	}
	lift := [3]float64{5, 8, 15}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		px := func(c int) uint8 {
			v := ch[c][i]*0.09 + lift[c]
			if v > 255 {
				v = 255
			}
			return uint8(v)
		}
		dst.SetNRGBA(i%w, i/w, color.NRGBA{R: px(0), G: px(1), B: px(2), A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// boxBlur is one separable box-blur pass of radius r; three passes
// approximate a Gaussian.
func boxBlur(src []float64, w, h, r int) []float64 {
	tmp := make([]float64, len(src))
	out := make([]float64, len(src))
	for y := 0; y < h; y++ {
		var acc float64
		row := src[y*w : (y+1)*w]
		for x := -r; x <= r; x++ {
			acc += row[clampInt(x, 0, w-1)]
		}
		for x := 0; x < w; x++ {
			tmp[y*w+x] = acc / float64(2*r+1)
			acc += row[clampInt(x+r+1, 0, w-1)] - row[clampInt(x-r, 0, w-1)]
		}
	}
	for x := 0; x < w; x++ {
		var acc float64
		for y := -r; y <= r; y++ {
			acc += tmp[clampInt(y, 0, h-1)*w+x]
		}
		for y := 0; y < h; y++ {
			out[y*w+x] = acc / float64(2*r+1)
			acc += tmp[clampInt(y+r+1, 0, h-1)*w+x] - tmp[clampInt(y-r, 0, h-1)*w+x]
		}
	}
	return out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// cachedHeroAssetAny returns the selected take of name regardless of its
// cache key, for building the placeholder from whatever portraits exist.
func cachedHeroAssetAny(root, name string) ([]byte, bool) {
	w, err := NewManifestWriter(root)
	if err != nil {
		return nil, false
	}
	asset, ok := w.manifest.Assets[name]
	if !ok {
		return nil, false
	}
	return cachedHeroAsset(root, name, asset.Metadata["cache_key"])
}
