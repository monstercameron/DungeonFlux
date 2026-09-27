package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func heroTestPNG(t *testing.T, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type fakeHeroImage struct {
	png   []byte
	calls int
}

func (f *fakeHeroImage) GenerateWithReferences(context.Context, ports.ImageRequest, [][]byte) (ports.ImageStream, error) {
	f.calls++
	return &fakeHeroStream{events: []ports.ImageEvent{{PNG: []byte("partial"), Partial: true}, {PNG: f.png}}}, nil
}

type fakeHeroStream struct{ events []ports.ImageEvent }

func (s *fakeHeroStream) Recv() (ports.ImageEvent, error) {
	if len(s.events) == 0 {
		return ports.ImageEvent{}, io.EOF
	}
	e := s.events[0]
	s.events = s.events[1:]
	return e, nil
}
func (s *fakeHeroStream) Close() error { return nil }

type fakeHeroVideo struct {
	submits int
	reqs    []ports.VideoRequest
}

func (f *fakeHeroVideo) Submit(_ context.Context, req ports.VideoRequest) (ports.VideoJob, error) {
	f.submits++
	f.reqs = append(f.reqs, req)
	return ports.VideoJob{Vendor: "fake", ID: "job" + string(rune('0'+f.submits))}, nil
}
func (f *fakeHeroVideo) Poll(context.Context, ports.VideoJob) (ports.VideoStatus, error) {
	return ports.VideoStatus{State: vocab.JobDone, URL: "u"}, nil
}
func (f *fakeHeroVideo) Download(context.Context, string) ([]byte, error) {
	return []byte("mp4-bytes-" + string(rune('0'+f.submits))), nil
}

func heroTestRoot(t *testing.T) (string, heroIntroSpec) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"version":1,"assets":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ref := filepath.Join(root, "ref.png")
	if err := os.WriteFile(ref, []byte("reference sheet"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, heroIntroSpec{ID: "female", Reference: ref, Person: "a woman", Costume: "the rogue's leathers"}
}

func TestRenderHeroIntro_GeneratesOnceThenServesFromCache(t *testing.T) {
	root, hero := heroTestRoot(t)
	img := &fakeHeroImage{png: heroTestPNG(t, color.NRGBA{200, 180, 150, 255})}
	vid := &fakeHeroVideo{}
	budget := 6.0
	var out bytes.Buffer
	if err := renderHeroIntro(context.Background(), root, hero, heroIntroVendors{image: img, video: vid}, &budget, time.Millisecond, &out); err != nil {
		t.Fatal(err)
	}
	if img.calls != 1 || vid.submits != 2 {
		t.Fatalf("calls: image %d video %d, want 1 and 2", img.calls, vid.submits)
	}
	enter, idle := vid.reqs[0], vid.reqs[1]
	if !bytes.Equal(enter.LastFrame, img.png) || !bytes.Equal(idle.FirstFrame, img.png) || !bytes.Equal(idle.LastFrame, img.png) {
		t.Fatal("entrance must end on the portrait and the idle loop must start and end on it")
	}
	if bytes.Equal(enter.FirstFrame, img.png) {
		t.Fatal("entrance must start from the shadow frame, not the portrait")
	}
	if want := 6.0 - heroPortraitUSD - float64(heroEnterSeconds+heroIdleSeconds)*heroVideoUSDPerS; budget < want-1e-9 || budget > want+1e-9 {
		t.Fatalf("budget %.3f, want %.3f", budget, want)
	}
	for _, name := range []string{"hero_default_female", "hero_default_female_enter", "hero_default_female_idle"} {
		if !strings.Contains(out.String(), name) {
			t.Fatalf("report is missing %s: %s", name, out.String())
		}
	}
	// A second run with no vendors must be served entirely from the cache.
	if err := renderHeroIntro(context.Background(), root, hero, heroIntroVendors{}, &budget, time.Millisecond, io.Discard); err != nil {
		t.Fatalf("cache-only rerun: %v", err)
	}
}

func TestRenderHeroIntro_RespectsTheSpendCap(t *testing.T) {
	root, hero := heroTestRoot(t)
	img := &fakeHeroImage{png: heroTestPNG(t, color.NRGBA{200, 180, 150, 255})}
	vid := &fakeHeroVideo{}
	budget := heroPortraitUSD + 1 // enough for the portrait and nothing else
	err := renderHeroIntro(context.Background(), root, hero, heroIntroVendors{image: img, video: vid}, &budget, time.Millisecond, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "spend cap") {
		t.Fatalf("want a spend-cap error, got %v", err)
	}
	if vid.submits != 0 {
		t.Fatalf("submitted %d videos past the cap", vid.submits)
	}
}

func TestRenderHeroIntro_CacheMissWithoutKeysFails(t *testing.T) {
	root, hero := heroTestRoot(t)
	budget := 6.0
	if err := renderHeroIntro(context.Background(), root, hero, heroIntroVendors{}, &budget, time.Millisecond, io.Discard); err == nil {
		t.Fatal("a cache miss with no vendor must fail, not pretend")
	}
}

func TestHeroShadowFrame_IsNearlyBlackAndCool(t *testing.T) {
	data, err := heroShadowFrame(heroTestPNG(t, color.NRGBA{255, 255, 255, 255}))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(2, 2).RGBA()
	if r>>8 > 20 || g>>8 > 20 || b>>8 > 30 || b <= r {
		t.Fatalf("shadow pixel %d,%d,%d is not a cool near-black", r>>8, g>>8, b>>8)
	}
	if _, err := heroShadowFrame([]byte("not an image")); err == nil {
		t.Fatal("an undecodable portrait must be an error")
	}
}

func TestHeroJob_ResumesAPersistedSubmission(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "job.json.tmp")
	if err := os.WriteFile(path, []byte(`{"Vendor":"fake","ID":"kept"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	vid := &fakeHeroVideo{}
	budget := 0.0
	job, err := heroJob(context.Background(), vid, ports.VideoRequest{Seconds: 4}, path, &budget)
	if err != nil || job.ID != "kept" || vid.submits != 0 {
		t.Fatalf("resume = %+v, %v, submits %d", job, err, vid.submits)
	}
}

func TestHeroPlaceholder_IsADarkBlurredBlend(t *testing.T) {
	bright := heroTestPNG(t, color.NRGBA{250, 240, 230, 255})
	dim := heroTestPNG(t, color.NRGBA{10, 10, 10, 255})
	data, err := heroPlaceholder([][]byte{bright, dim, nil})
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(4, 6).RGBA()
	if r>>8 > 30 || b>>8 > 40 || b <= r {
		t.Fatalf("placeholder pixel %d,%d,%d must be a cool dark blend", r>>8, g>>8, b>>8)
	}
	if _, err := heroPlaceholder(nil); err == nil {
		t.Fatal("no portraits must be an error")
	}
	root, _ := heroTestRoot(t)
	var out bytes.Buffer
	if err := renderHeroPlaceholder(root, [][]byte{bright}, &out); err != nil {
		t.Fatal(err)
	}
	if err := renderHeroPlaceholder(root, [][]byte{bright}, &out); err != nil || !strings.Contains(out.String(), `"cached":true`) {
		t.Fatalf("second placeholder render must hit the cache: %v %s", err, out.String())
	}
}
