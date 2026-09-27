//go:build live

package fal

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func TestLiveFalSubmit(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	key := os.Getenv("DF_FAL_KEY")
	if key == "" {
		t.Fatal("DF_FAL_KEY is not set while DF_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	frame, err := tinyPNG()
	if err != nil {
		t.Fatal(err)
	}
	job, err := New(key, "", "", nil).Submit(ctx, ports.VideoRequest{FirstFrame: frame, Prompt: "a blue dot", Seconds: 4, Resolution: "480p"})
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" || job.Vendor != "fal" {
		t.Fatalf("invalid fal job: %#v", job)
	}
}

func tinyPNG() ([]byte, error) {
	var b bytes.Buffer
	frame := image.NewNRGBA(image.Rect(0, 0, 1024, 1024))
	for i := range frame.Pix {
		frame.Pix[i] = 255
	}
	if err := png.Encode(&b, frame); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
