package wire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	videofal "github.com/monstercameron/DungeonFlux/internal/adapters/video/fal"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/runtime"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// falConcurrency is fal's default per-key concurrency (plan §0.21.5).
const falConcurrency = 2

// billboardHTTPTimeout bounds one fal queue request (not the whole render).
const billboardHTTPTimeout = 2 * time.Minute

// levelStillName is the build-time manifest name of a scene's clean level
// still at a camera preset, e.g. level_still_64bb46d5_tactical.
func levelStillName(scene, preset string) string {
	return "level_still_" + scene + "_" + preset
}

// billboardEngine decorates an engine: it starts a seat's loops when its
// turnaround is Ready and puts ready loops on the combat tokens.
type billboardEngine struct {
	ports.Engine
	hub *billboardHub
}

// newBillboardEngine wraps engine for one run; the hub's per-run state is
// reset, so a new game starts with no loops and a fresh fal cap.
func newBillboardEngine(engine ports.Engine, hub *billboardHub) ports.Engine {
	if hub == nil {
		return engine
	}
	hub.reset()
	return &billboardEngine{Engine: engine, hub: hub}
}

func (e *billboardEngine) Step(env domain.Envelope) domain.StepOut {
	out := e.Engine.Step(env)
	if effect, ok := e.hub.observe(env.Event, e.seatClass); ok {
		out.Effects = append(append([]domain.Effect(nil), out.Effects...), effect)
	}
	return out
}

func (e *billboardEngine) View() domain.View { return e.hub.decorate(e.Engine.View()) }

func (e *billboardEngine) seatClass(seat domain.SeatID) string {
	for _, view := range e.Engine.View().Seats {
		if view.Seat == seat && view.Character != nil {
			return view.Character.Class
		}
	}
	return ""
}

// billboardServices builds the loop generator's shared pieces: the fal
// reference-to-video adapter (nil without a live key), the concurrency pool,
// the asset writer and reader, and the persistent cache.
func billboardServices(cfg config.Config, assets *assetStore, cache ports.Cache, logger *slog.Logger) (media.BillboardGeneratorConfig, error) {
	pool, err := media.NewPool(map[vocab.VendorName]int{vocab.VendorFal: falConcurrency})
	if err != nil {
		return media.BillboardGeneratorConfig{}, err
	}
	base := media.BillboardGeneratorConfig{Model: videofal.ReferenceModel, Assets: assets, Cache: cache, Source: assets.Read, Pool: pool}
	if adapter, live := billboardAdapter(cfg); live {
		client := httpx.NewVendorClient(adapter.Vendor, billboardHTTPTimeout, logger)
		base.Video = videofal.NewReference(adapter.APIKey, videofal.ReferenceModel, adapter.BaseURL, client)
	}
	return base, nil
}

// billboardLoopsExecutor resolves a seat's loops: build-time catalogue (the
// thrall), then the cache, then fal when the hub allows paid generation.
// Work runs under the hub's run context, not the reference slot's scope, so
// leaving Creation does not cancel a 3-minute render; a new run does.
func billboardLoopsExecutor(hub *billboardHub, base media.BillboardGeneratorConfig, logger *slog.Logger) runtime.Executor[domain.GenerateBillboardLoops] {
	return func(ctx context.Context, effect domain.GenerateBillboardLoops, scope domain.Scope, in ports.Inbox) {
		hubCtx, ledger := hub.run()
		runCtx, cancel := context.WithCancel(hubCtx)
		stop := context.AfterFunc(ctx, cancel)
		defer stop()
		defer cancel()
		if ctx.Err() != nil {
			cancel()
		}
		config := base
		config.Budget = ledger
		generator := media.NewBillboardGenerator(config)
		executor := media.NewBillboardLoopsExecutor(media.BillboardLoopsConfig{
			Generator: generator, Generate: hub.enabled && generator.CanGenerate(),
			Catalogue: func(seat domain.SeatID, action string) (domain.Asset, bool) {
				url, ok := hub.thrall[action]
				return domain.Asset{ID: domain.AssetID(url), URL: url, Kind: string(vocab.AssetVideo), MIME: "video/mp4"}, ok && seat == 0
			},
			Subject: func(ctx context.Context, seat domain.SeatID) (media.BillboardSubject, error) {
				return hub.loadSubject(ctx, seat, base.Source)
			},
			LevelStill: hub.levelStill,
			OnResult:   billboardLog(logger),
		})
		executor.Execute(runCtx, effect, scope, in)
	}
}

// loadSubject reads the seat's front and three-quarter turnaround crops.
func (h *billboardHub) loadSubject(ctx context.Context, seat domain.SeatID, read media.ReferenceSource) (media.BillboardSubject, error) {
	refs, class := h.subject(seat)
	if !refs.Ready() || read == nil {
		return media.BillboardSubject{}, fmt.Errorf("wire: seat %d has no ready turnaround", seat)
	}
	subject := media.BillboardSubject{Look: lookFor(class), Weapon: media.BillboardWeapon(class)}
	for _, angle := range []vocab.ReferenceAngle{vocab.ReferenceFront, vocab.ReferenceThreeQuarter} {
		data, err := read(ctx, refs.Angles[angle])
		if err != nil {
			return media.BillboardSubject{}, fmt.Errorf("wire: read %s reference: %w", angle, err)
		}
		subject.References = append(subject.References, data)
	}
	return subject, nil
}

func lookFor(class string) string {
	class = strings.ToLower(strings.TrimSpace(class))
	if class == "" {
		return "hero"
	}
	return class + " hero"
}

func billboardLog(logger *slog.Logger) func(string, media.BillboardResult, error) {
	return func(slot string, result media.BillboardResult, err error) {
		if logger == nil {
			return
		}
		if err != nil {
			logger.Warn("billboard loop failed", "slot", slot, "err", err, "disabled", errors.Is(err, media.ErrBillboardDisabled))
			return
		}
		logger.Info("billboard loop ready", "slot", slot, "cached", result.Cached, "asset", result.Asset.ID, "key", result.Key, "latency_ms", result.Latency.Milliseconds())
	}
}

// billboardHubFor builds the hub from the resolved one-shot.
func billboardHubFor(cfg config.Config, manifest ManifestResult) *billboardHub {
	return newBillboardHub(cfg, manifest.OneShot.Catalogue, manifest.OneShot.Encounter.Battlefield.SceneURL)
}

// defaultBillboardHub is used when a caller built executors without a hub
// (tests); it never generates.
func defaultBillboardHub(cfg config.Config) *billboardHub {
	manifest, err := LoadManifest(filepath.Join("artifacts", "runtime", "buildtime", "manifest.json"), nil)
	if err != nil {
		return newBillboardHub(config.Config{}, nil, "")
	}
	return billboardHubFor(cfg, manifest)
}
