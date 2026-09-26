//go:build live

package evolink

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

func TestLiveEvoLinkSubmit(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	key := os.Getenv("DF_EVOLINK_API_KEY")
	if key == "" {
		t.Fatal("DF_EVOLINK_API_KEY is not set while DF_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	job, err := New(key, "", nil).Submit(ctx, ports.VideoRequest{FirstFrame: tinyPNG(), Prompt: "a blue dot", Seconds: 4, Resolution: "480p"})
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" || job.Vendor != "evolink" {
		t.Fatalf("invalid EvoLink job: %#v", job)
	}
}

func tinyPNG() []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1024, 1024)))
	return b.Bytes()
}
