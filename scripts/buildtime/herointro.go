package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"github.com/monstercameron/DungeonFlux/internal/wire"
)

// Demo default heroes for the creation screen: one male and one female hero,
// each a portrait still, an entrance clip that brings them out of the dark and
// ends exactly on that still, and an idle loop that starts and ends on it, so
// the TV can play entrance then idle with no visible cut.
//
// The portraits are edits of the two battle reference sheets (paladin, rogue),
// so the hero who steps out of the shadows wears the costume they fight in.

const (
	heroIntroModel    = "bytedance/seedance-2.0/fast/image-to-video"
	heroIntroVersion  = "hero-intro-v1"
	heroEnterSeconds  = 5
	heroIdleSeconds   = 4
	heroVideoUSDPerS  = 0.242 // documented 720p Seedance 2.0 Fast rate
	heroPortraitUSD   = 0.25  // conservative per-image estimate for a reference edit
	heroPortraitSize  = "1024x1536"
	heroShadowDimming = 0.035
)

type heroIntroSpec struct {
	ID        string `json:"id"`
	Reference string `json:"reference"`
	Person    string `json:"person"`
	Costume   string `json:"costume"`
}

type heroIntroConfig struct {
	Heroes []heroIntroSpec `json:"heroes"`
}

// heroVideoGen is an image-to-video vendor that can also fetch its result.
type heroVideoGen interface {
	ports.VideoGen
	Download(ctx context.Context, url string) ([]byte, error)
}

type heroIntroVendors struct {
	image interface {
		GenerateWithReferences(context.Context, ports.ImageRequest, [][]byte) (ports.ImageStream, error)
	}
	video heroVideoGen
}

func runHeroIntros(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("herointros", flag.ContinueOnError)
	root := flags.String("root", "artifacts/runtime/buildtime", "build-time manifest root")
	configPath := flags.String("config", "config/hero-intros.json", "hero configuration")
	maxUSD := flags.Float64("max-usd", 6, "maximum estimated generation spend")
	cacheOnly := flags.Bool("cache-only", false, "require verified cached assets; never call a vendor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	data, err := os.ReadFile(*configPath)
	if err != nil {
		return err
	}
	var config heroIntroConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	if len(config.Heroes) == 0 {
		return errors.New("herointros: no heroes configured")
	}
	var vendors heroIntroVendors
	if !*cacheOnly {
		if key := envKey("DF_OPENAI_API_KEY"); key != "" {
			vendors.image = wire.NewHeroIntroImage(key, envKey("DF_OPENAI_IMAGES_URL"))
		}
		if key := envKey("DF_FAL_KEY"); key != "" {
			vendors.video = wire.NewHeroIntroVideo(key, heroIntroModel)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	budget := *maxUSD
	var portraits [][]byte
	for _, hero := range config.Heroes {
		if err := renderHeroIntro(ctx, *root, hero, vendors, &budget, 5*time.Second, out); err != nil {
			return fmt.Errorf("hero %s: %w", hero.ID, err)
		}
		data, _ := cachedHeroAssetAny(*root, "hero_default_"+hero.ID)
		portraits = append(portraits, data)
	}
	if err := renderHeroPlaceholder(*root, portraits, out); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(map[string]any{"estimated_spent_usd": *maxUSD - budget})
}

func heroPortraitPrompt(hero heroIntroSpec) string {
	return "Use the reference character sheet only for the outfit: keep " + hero.Costume + " exactly, with the same colors, materials and details. " +
		"Paint ONE full-body hero, " + hero.Person + ", standing in a relaxed heroic three-quarter pose facing the viewer, centered, head in the top fifth and boots just above the bottom edge. " +
		"The background is deep blue-black darkness with faint drifting mist and nothing else; a warm lantern rim light from the right and cool moonlight from the left sculpt the figure. " +
		"Painterly dark-fantasy realism in the same art style as the reference. Vertical portrait. No text, no frame, no turnaround layout, no second figure."
}

const heroEnterPrompt = "Static camera, one continuous shot. The first frame is almost total darkness. The hero slowly steps forward out of the shadows and mist into the light, " +
	"the warm lantern glow rising across their face and armor, the cloak settling as they stop. They end standing still in exactly the pose of the final frame, looking at the viewer. " +
	"No cuts, no camera move, no text."

const heroIdlePrompt = "Static camera, one continuous shot. The hero stands in place and breathes slowly; the cloak and loose hair stir in a light breeze; the lantern light flickers softly on the armor; " +
	"thin mist drifts behind them. The pose never changes and the first and last frames match for a seamless loop. No cuts, no camera move, no text."

// heroIntroKey fingerprints everything that shapes an asset, so a changed
// prompt, reference or version re-renders and an unchanged one never does.
func heroIntroKey(parts ...[]byte) string {
	h := sha256.New()
	h.Write([]byte(heroIntroVersion))
	for _, p := range parts {
		h.Write([]byte{0})
		h.Write(p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func renderHeroIntro(ctx context.Context, root string, hero heroIntroSpec, vendors heroIntroVendors, budget *float64, interval time.Duration, out io.Writer) error {
	if hero.ID == "" || hero.Reference == "" {
		return errors.New("id and reference are required")
	}
	reference, err := os.ReadFile(hero.Reference)
	if err != nil {
		return err
	}
	prefix := "hero_default_" + hero.ID
	prompt := heroPortraitPrompt(hero)
	portraitKey := heroIntroKey(reference, []byte(prompt))
	portrait, cached, err := heroPortrait(ctx, root, prefix, portraitKey, prompt, reference, vendors, budget)
	if err != nil {
		return err
	}
	report(out, prefix, cached)
	shadow, err := heroShadowFrame(portrait)
	if err != nil {
		return err
	}
	clips := []struct {
		name, prompt string
		first        []byte
		seconds      int
	}{
		{prefix + "_enter", heroEnterPrompt, shadow, heroEnterSeconds},
		{prefix + "_idle", heroIdlePrompt, portrait, heroIdleSeconds},
	}
	for _, clip := range clips {
		req := ports.VideoRequest{FirstFrame: clip.first, LastFrame: portrait, Prompt: clip.prompt, Seconds: clip.seconds, Resolution: "720p"}
		key := heroIntroKey(portrait, clip.first, []byte(clip.prompt), []byte{byte(clip.seconds)})
		cached, err := heroClip(ctx, root, clip.name, key, req, vendors.video, budget, interval)
		if err != nil {
			return fmt.Errorf("%s: %w", clip.name, err)
		}
		report(out, clip.name, cached)
	}
	return nil
}

func report(out io.Writer, name string, cached bool) {
	_ = json.NewEncoder(out).Encode(map[string]any{"asset": name, "cached": cached})
}

// heroPortrait returns the cached portrait, or generates, stores and
// registers it. The manifest's cache_key ties the take to its inputs.
func heroPortrait(ctx context.Context, root, name, key, prompt string, reference []byte, vendors heroIntroVendors, budget *float64) ([]byte, bool, error) {
	if data, ok := cachedHeroAsset(root, name, key); ok {
		return data, true, nil
	}
	if vendors.image == nil {
		return nil, false, errors.New("cache miss; generation requires DF_OPENAI_API_KEY")
	}
	if *budget < heroPortraitUSD {
		return nil, false, errors.New("estimated spend cap exceeded")
	}
	*budget -= heroPortraitUSD
	stream, err := vendors.image.GenerateWithReferences(ctx, ports.ImageRequest{Prompt: prompt, Size: heroPortraitSize}, [][]byte{reference})
	if err != nil {
		return nil, false, err
	}
	defer stream.Close()
	var final []byte
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, false, err
		}
		if !event.Partial {
			final = event.PNG
		}
	}
	if len(final) == 0 {
		return nil, false, errors.New("image vendor returned no final image")
	}
	if err := registerHeroAsset(root, name, "IMAGE_STILL", ".png", final, 0, map[string]string{"cache_key": key, "model": "gpt-image-2.5-flare", "size": heroPortraitSize, "prompt_version": heroIntroVersion}); err != nil {
		return nil, false, err
	}
	return final, false, nil
}

// heroShadowFrame darkens the portrait almost to black with a cool tint, the
// entrance clip's first frame: the figure is barely there, so the model
// animates the hero emerging rather than a fade.
func heroShadowFrame(portrait []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(portrait))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := src.At(x, y).RGBA()
			dim := func(v uint32, lift float64) uint8 { return uint8(float64(v>>8)*heroShadowDimming + lift) }
			dst.SetNRGBA(x, y, color.NRGBA{R: dim(r, 4), G: dim(g, 7), B: dim(bl, 13), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func heroClip(ctx context.Context, root, name, key string, req ports.VideoRequest, vendor heroVideoGen, budget *float64, interval time.Duration) (bool, error) {
	if _, ok := cachedHeroAsset(root, name, key); ok {
		return true, nil
	}
	if vendor == nil {
		return false, errors.New("cache miss; generation requires DF_FAL_KEY")
	}
	jobDir := filepath.Join(root, "herointro-jobs")
	if err := os.MkdirAll(jobDir, 0o700); err != nil {
		return false, err
	}
	jobPath := filepath.Join(jobDir, key+".json.tmp")
	job, err := heroJob(ctx, vendor, req, jobPath, budget)
	if err != nil {
		return false, err
	}
	data, err := pollHeroClip(ctx, vendor, job, interval)
	if err != nil {
		return false, err
	}
	if err := registerHeroAsset(root, name, "VIDEO", ".mp4", data, int64(req.Seconds)*1000, map[string]string{"cache_key": key, "model": heroIntroModel, "resolution": req.Resolution, "audio": "false", "prompt_version": heroIntroVersion}); err != nil {
		return false, err
	}
	return false, os.Remove(jobPath)
}

// heroJob persists the submitted job before polling, so an interrupted run
// resumes the same paid render instead of submitting a new one.
func heroJob(ctx context.Context, vendor heroVideoGen, req ports.VideoRequest, path string, budget *float64) (ports.VideoJob, error) {
	var job ports.VideoJob
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &job); err != nil || job.ID == "" {
			return job, errors.New("cached job is unreadable")
		}
		return job, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return job, err
	}
	estimate := float64(req.Seconds) * heroVideoUSDPerS
	if *budget < estimate {
		return job, errors.New("estimated spend cap exceeded")
	}
	*budget -= estimate
	job, err := vendor.Submit(ctx, req)
	if err != nil {
		return job, err
	}
	data, err := json.Marshal(job)
	if err != nil {
		return job, err
	}
	return job, os.WriteFile(path, data, 0o600)
}

func pollHeroClip(ctx context.Context, vendor heroVideoGen, job ports.VideoJob, interval time.Duration) ([]byte, error) {
	for {
		status, err := vendor.Poll(ctx, job)
		if err != nil {
			return nil, err
		}
		switch status.State {
		case vocab.JobDone:
			data, err := vendor.Download(ctx, status.URL)
			if err == nil && len(data) == 0 {
				err = errors.New("vendor returned an empty video")
			}
			return data, err
		case vocab.JobFailed:
			return nil, errors.New("vendor job failed")
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func registerHeroAsset(root, name, kind, ext string, data []byte, durationMS int64, metadata map[string]string) error {
	tmp, err := os.CreateTemp(root, "herointro-*"+ext)
	if err != nil {
		return err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	writer, err := NewManifestWriter(root)
	if err != nil {
		return err
	}
	if _, err := writer.AddFile(name, kind, path, 1); err != nil {
		return err
	}
	if err := writer.SetMetadata(name, durationMS, 0, metadata); err != nil {
		return err
	}
	_, err = writer.Write()
	return err
}

// cachedHeroAsset returns the selected take's bytes when its cache_key
// matches and the file still hashes to the recorded SHA-256.
func cachedHeroAsset(root, name, key string) ([]byte, bool) {
	w, err := NewManifestWriter(root)
	if err != nil {
		return nil, false
	}
	asset, ok := w.manifest.Assets[name]
	if !ok || asset.Metadata["cache_key"] != key {
		return nil, false
	}
	for _, take := range asset.Takes {
		if take.Number != asset.Selected {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, take.Path))
		sum := sha256.Sum256(data)
		if err == nil && len(data) > 0 && hex.EncodeToString(sum[:]) == take.SHA256 {
			return data, true
		}
	}
	return nil, false
}
