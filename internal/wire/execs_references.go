package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"golang.org/x/image/webp"
)

type referenceRegistry struct {
	mu     sync.RWMutex
	assets map[domain.SeatID]media.ReferenceAssets
}

type referenceCaptureInbox struct {
	next     ports.Inbox
	registry *referenceRegistry
}

func (i *referenceCaptureInbox) Post(ctx context.Context, env domain.Envelope) bool {
	if i != nil && i.registry != nil {
		i.registry.capture(env.Event)
	}
	if i == nil || i.next == nil {
		return false
	}
	return i.next.Post(ctx, env)
}

func (r *referenceRegistry) capture(event domain.Event) {
	ready, ok := event.(domain.AssetReady)
	if !ok {
		return
	}
	parts := strings.Split(ready.Slot, ":")
	if len(parts) != 3 || parts[0] != "reference" {
		return
	}
	seat := domain.SeatID(0)
	if _, err := fmt.Sscanf(parts[1], "%d", &seat); err != nil || seat <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value := r.assets[seat]
	if parts[2] == "sheet" {
		value.Sheet = ready.Asset.ID
	} else {
		if value.Angles == nil {
			value.Angles = make(map[vocab.ReferenceAngle]domain.AssetID)
		}
		value.Angles[vocab.ReferenceAngle(parts[2])] = ready.Asset.ID
	}
	r.assets[seat] = value
}

func (r *referenceRegistry) snapshot() map[domain.SeatID]media.ReferenceAssets {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[domain.SeatID]media.ReferenceAssets, len(r.assets))
	for seat, value := range r.assets {
		angles := make(map[vocab.ReferenceAngle]domain.AssetID, len(value.Angles))
		for angle, id := range value.Angles {
			angles[angle] = id
		}
		value.Angles = angles
		result[seat] = value
	}
	return result
}

func (r *referenceRegistry) forEffect(slot string) map[domain.SeatID]media.ReferenceAssets {
	result := r.snapshot()
	if seat := seatFromSlot(slot); seat > 0 {
		if _, ok := result[seat]; !ok {
			result[seat] = media.ReferenceAssets{}
		}
	}
	if _, ok := result[0]; !ok {
		result[0] = mergeReferences(result)
	}
	return result
}

func seatFromSlot(slot string) domain.SeatID {
	parts := strings.Split(slot, ":")
	if len(parts) < 2 {
		return 0
	}
	var seat domain.SeatID
	if _, err := fmt.Sscanf(parts[1], "%d", &seat); err != nil {
		return 0
	}
	return seat
}

func mergeReferences(values map[domain.SeatID]media.ReferenceAssets) media.ReferenceAssets {
	merged := media.ReferenceAssets{Angles: make(map[vocab.ReferenceAngle]domain.AssetID)}
	for _, value := range values {
		if merged.Sheet == "" {
			merged.Sheet = value.Sheet
		}
		for angle, id := range value.Angles {
			if _, exists := merged.Angles[angle]; !exists {
				merged.Angles[angle] = id
			}
		}
	}
	return merged
}

func referenceInput(source media.ReferenceSource) func(context.Context, []domain.AssetID) error {
	return func(ctx context.Context, ids []domain.AssetID) error {
		for _, id := range ids {
			data, err := source(ctx, id)
			if err != nil {
				return err
			}
			if len(data) == 0 {
				return fmt.Errorf("wire: reference asset %q is empty", id)
			}
		}
		return nil
	}
}

func referenceFrame(source media.ReferenceSource) func(context.Context, []domain.AssetID) ([]byte, error) {
	return func(ctx context.Context, ids []domain.AssetID) ([]byte, error) {
		if len(ids) == 0 {
			return nil, errors.New("wire: reference frame has no assets")
		}
		return source(ctx, ids[0])
	}
}

func loadVoiceFallbacks(path string) map[string][]byte {
	ids := map[string]string{
		"attack_effort": "sfx_sword_slash", "hurt": "sfx_blade_hit",
		"spell_cast": "sfx_check_success", "downed": "sfx_blade_hit",
		"victory": "sfx_check_success", "heal": "sfx_check_success",
	}
	fallbacks := make(map[string][]byte, len(ids))
	for cue, id := range ids {
		if data, err := loadManifestAsset(path, id); err == nil {
			fallbacks[cue] = data
		}
	}
	return fallbacks
}

func referenceFallback(effect domain.GenerateCharacterReference) ([]byte, map[vocab.ReferenceAngle][]byte) {
	species, speciesErr := loadManifestAsset(manifestPath(), "ui/species_"+assetSlug(effect.Species))
	class, classErr := loadManifestAsset(manifestPath(), "ui/class_"+assetSlug(effect.Class))
	if speciesErr != nil || classErr != nil {
		return nil, nil
	}
	sheet, err := composeReferenceSheet(species, class)
	if err != nil {
		return nil, nil
	}
	return sheet, nil
}

func composeReferenceSheet(species, class []byte) ([]byte, error) {
	speciesImage, err := decodeReferenceImage(species)
	if err != nil {
		return nil, err
	}
	classImage, err := decodeReferenceImage(class)
	if err != nil {
		return nil, err
	}
	panelWidth := speciesImage.Bounds().Dx() + classImage.Bounds().Dx()
	height := maxInt(speciesImage.Bounds().Dy(), classImage.Bounds().Dy())
	canvas := image.NewRGBA(image.Rect(0, 0, panelWidth*4, height))
	for index := 0; index < 4; index++ {
		origin := image.Pt(index*panelWidth, 0)
		draw.Draw(canvas, image.Rectangle{Min: origin, Max: origin.Add(speciesImage.Bounds().Size())}, speciesImage, speciesImage.Bounds().Min, draw.Over)
		classOrigin := origin.Add(image.Pt(speciesImage.Bounds().Dx(), 0))
		draw.Draw(canvas, image.Rectangle{Min: classOrigin, Max: classOrigin.Add(classImage.Bounds().Size())}, classImage, classImage.Bounds().Min, draw.Over)
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func decodeReferenceImage(data []byte) (image.Image, error) {
	if value, err := webp.Decode(bytes.NewReader(data)); err == nil {
		return value, nil
	}
	return png.Decode(bytes.NewReader(data))
}

func loadManifestAsset(path, id string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest BuildManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	entry, ok := manifest.Assets[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	take, ok := selectedTake(entry)
	if !ok {
		return nil, os.ErrNotExist
	}
	root := filepath.Dir(path)
	source, err := safeManifestPath(root, take.Path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(source)
}

func manifestPath() string {
	return filepath.Join("artifacts", "runtime", "buildtime", "manifest.json")
}

func assetSlug(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
