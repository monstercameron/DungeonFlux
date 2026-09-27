package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/wire"
)

type killcamSpec struct {
	Class  string `json:"class"`
	Hero   string `json:"hero_reference"`
	Weapon string `json:"weapon"`
}

type killcamConfig struct {
	Level  string        `json:"battlefield_reference"`
	Enemy  string        `json:"enemy_reference"`
	Heroes []killcamSpec `json:"heroes"`
}

func runKillcams(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("killcams", flag.ContinueOnError)
	root := flags.String("root", "artifacts/runtime/buildtime", "build-time manifest root")
	configPath := flags.String("config", "config/killcams.json", "reference configuration")
	maxUSD := flags.Float64("max-usd", 5, "maximum estimated generation spend")
	cacheOnly := flags.Bool("cache-only", false, "require verified cached clips; never call fal")
	if err := flags.Parse(args); err != nil {
		return err
	}
	data, err := os.ReadFile(*configPath)
	if err != nil {
		return err
	}
	var config killcamConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	if len(config.Heroes) == 0 {
		return errors.New("killcams: no hero references")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var vendor ports.ReferenceVideoGen
	if !*cacheOnly && envKey("DF_FAL_KEY") != "" {
		vendor = wire.NewKillcamVideo(envKey("DF_FAL_KEY"))
	}
	remaining := *maxUSD
	for _, hero := range config.Heroes {
		for _, outcome := range []string{"victory", "defeat"} {
			request, err := killcamRequest(*root, config, hero, outcome)
			if err != nil {
				return err
			}
			name := "killcam_" + hero.Class + "_" + outcome
			cached, err := renderKillcam(ctx, *root, name, hero.Class, outcome, request, vendor, &remaining, 5*time.Second)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if err := json.NewEncoder(out).Encode(map[string]any{"asset": name, "cached": cached, "estimated_spent_usd": *maxUSD - remaining}); err != nil {
				return err
			}
		}
	}
	return nil
}

func killcamRequest(root string, config killcamConfig, hero killcamSpec, outcome string) (ports.ReferenceVideoRequest, error) {
	if hero.Class == "" || hero.Weapon == "" || (outcome != "victory" && outcome != "defeat") {
		return ports.ReferenceVideoRequest{}, errors.New("killcams: class, weapon and valid outcome required")
	}
	heroImage, err := os.ReadFile(hero.Hero)
	if err != nil {
		return ports.ReferenceVideoRequest{}, err
	}
	enemy, err := manifestBytes(root, config.Enemy)
	if err != nil {
		return ports.ReferenceVideoRequest{}, err
	}
	level, err := manifestBytes(root, config.Level)
	if err != nil {
		return ports.ReferenceVideoRequest{}, err
	}
	prompt := "Create a four-second epic dark-fantasy finishing-blow cinematic. @Image1 is the hero identity reference sheet: preserve their exact face, species, costume and equipment; show ONE hero, never a sheet or a montage. @Image2 is the drowned thrall enemy: preserve its undead sailor identity. @Image3 is the battlefield: the fight happens on this same wooded path with the same terrain, lighting and atmosphere. Wide 16:9 film composition, low tracking camera, painterly realism, teal shadows, amber rim light, selective shallow depth of field, floating embers, rain and volumetric light. "
	if outcome == "victory" {
		prompt += "The hero delivers one decisive finishing strike using their " + hero.Weapon + ". A brief slow-motion impact at second two sends a wave of golden sparks through the thrall; the thrall collapses and dissolves into dark river mist. End on the triumphant hero in a low-angle silhouette. The hero wins unmistakably. "
	} else {
		prompt += "The drowned thrall whips its rusted bell chain in a powerful sweeping slam; at second two the hero is struck, knocked to the ground and lies defeated while the thrall looms above. End with a slow overhead pullback through drifting river mist. The villain wins unmistakably; the hero remains intact, no dismemberment. "
	}
	prompt += "One continuous action with a speed ramp into impact and a composed final beat. No text, no UI, no borders, no extra fighters, no gore. Keep the physical positions and identities coherent."
	return ports.ReferenceVideoRequest{Prompt: prompt, References: [][]byte{heroImage, enemy, level}, Seconds: 4, Resolution: "720p", Aspect: "16:9"}, nil
}

func killcamKey(req ports.ReferenceVideoRequest) string {
	data, _ := json.Marshal(struct {
		Version, Model string
		Request        ports.ReferenceVideoRequest
	}{"killcam-v1", wire.KillcamVideoModel, req})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func cachedKillcam(root, name, key string) bool {
	w, err := NewManifestWriter(root)
	if err != nil {
		return false
	}
	asset := w.manifest.Assets[name]
	if asset.Metadata["cache_key"] != key || asset.DurationMS != 4000 {
		return false
	}
	for _, take := range asset.Takes {
		if take.Number != asset.Selected {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, take.Path))
		hash := sha256.Sum256(data)
		return err == nil && len(data) > 0 && hex.EncodeToString(hash[:]) == take.SHA256
	}
	return false
}
