package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// BillboardCacheAdapter is the cache table's adapter column for loops.
const BillboardCacheAdapter = "billboard_loop"

// ErrBillboardDeadline reports a job that did not finish before the deadline.
var ErrBillboardDeadline = errors.New("media: billboard loop deadline exceeded")

// ErrBillboardDisabled reports a cache miss when paid generation is off.
var ErrBillboardDisabled = errors.New("media: billboard generation disabled")

// BillboardGeneratorConfig supplies the generator's services. Video may be
// nil, which makes the generator cache-only.
type BillboardGeneratorConfig struct {
	Video       ports.ReferenceVideoGen
	Model       string
	Assets      ports.AssetWriter
	Cache       ports.Cache
	Source      ReferenceSource
	Budget      *budget.Ledger
	EstimateUSD float64
	Pool        *Pool
	Clock       clock.Clock
	PollEvery   time.Duration
	Deadline    time.Duration
}

// BillboardResult is one resolved loop.
type BillboardResult struct {
	Asset   domain.Asset
	Key     string
	Cached  bool
	Latency time.Duration
}

// BillboardGenerator turns reference images and a level still into a
// green-screen loop, content-addressed through ports.Cache.
type BillboardGenerator struct {
	config BillboardGeneratorConfig
}

// NewBillboardGenerator applies the defaults: $0.46 per 480p 4 s clip (fal
// bills Seedance 2.0 Fast at $0.0112 per 1000 video tokens, ≈ 40.6k tokens
// for 496 × 864 × 24 fps × 4 s), a 5 s poll, and a 6 minute deadline.
func NewBillboardGenerator(config BillboardGeneratorConfig) *BillboardGenerator {
	if config.EstimateUSD <= 0 {
		config.EstimateUSD = 0.46
	}
	if config.Clock == nil {
		config.Clock = clock.Real{}
	}
	if config.PollEvery < 0 {
		config.PollEvery = 0
	} else if config.PollEvery == 0 {
		config.PollEvery = 5 * time.Second
	}
	if config.Deadline <= 0 {
		config.Deadline = 6 * time.Minute
	}
	return &BillboardGenerator{config: config}
}

// CanGenerate reports whether a cache miss can be generated.
func (g *BillboardGenerator) CanGenerate() bool {
	return g != nil && g.config.Video != nil && g.config.Assets != nil
}

// Key returns spec's cache key for this generator's model.
func (g *BillboardGenerator) Key(spec BillboardSpec) string {
	return BillboardCacheKey(g.config.Model, spec)
}

// Lookup returns the cached loop for spec without any vendor call. A cached
// asset whose file is gone counts as a miss.
func (g *BillboardGenerator) Lookup(ctx context.Context, spec BillboardSpec) (domain.Asset, bool, error) {
	if g == nil || g.config.Cache == nil {
		return domain.Asset{}, false, nil
	}
	data, ok, err := g.config.Cache.Get(ctx, BillboardCacheAdapter, g.Key(spec))
	if err != nil || !ok {
		return domain.Asset{}, false, err
	}
	var asset domain.Asset
	if err := json.Unmarshal(data, &asset); err != nil || asset.ID == "" {
		return domain.Asset{}, false, nil
	}
	if g.config.Source != nil {
		if body, readErr := g.config.Source(ctx, asset.ID); readErr != nil || len(body) == 0 {
			return domain.Asset{}, false, nil
		}
	}
	return asset, true, nil
}

// Generate returns the cached loop, or renders, stores, and caches a new one.
func (g *BillboardGenerator) Generate(ctx context.Context, spec BillboardSpec) (BillboardResult, error) {
	reservation, result, err := g.Prepare(ctx, spec)
	if err != nil || result.Cached {
		return result, err
	}
	return g.Render(ctx, spec, reservation)
}

// Prepare validates spec and checks the cache. On a hit it returns the
// cached result; on a miss it holds the clip's estimate against the budget
// and returns the reservation for Render. Callers that prepare several loops
// in order (idle before attack before hit) decide which ones a tight cap
// admits, independent of how the renders later interleave.
func (g *BillboardGenerator) Prepare(ctx context.Context, spec BillboardSpec) (*budget.Reservation, BillboardResult, error) {
	if err := validateBillboardSpec(spec); err != nil {
		return nil, BillboardResult{}, err
	}
	key := g.Key(spec)
	if asset, ok, err := g.Lookup(ctx, spec); err != nil {
		return nil, BillboardResult{}, err
	} else if ok {
		return nil, BillboardResult{Asset: asset, Key: key, Cached: true}, nil
	}
	if !g.CanGenerate() {
		return nil, BillboardResult{}, ErrBillboardDisabled
	}
	reservation, err := g.reserve()
	if err != nil {
		return nil, BillboardResult{}, err
	}
	return reservation, BillboardResult{Key: key}, nil
}

// Render generates spec under a reservation from Prepare (nil without a
// ledger), waiting for a fal slot, and settles or releases it.
func (g *BillboardGenerator) Render(ctx context.Context, spec BillboardSpec, reservation *budget.Reservation) (BillboardResult, error) {
	key := g.Key(spec)
	started := g.config.Clock.Now()
	var video []byte
	job := func(run context.Context) error {
		var renderErr error
		video, renderErr = g.render(run, spec)
		return renderErr
	}
	var err error
	if g.config.Pool != nil {
		err = g.config.Pool.Run(ctx, vocab.VendorFal, job)
	} else {
		err = job(ctx)
	}
	if err != nil {
		if reservation != nil {
			reservation.Release()
		}
		return BillboardResult{}, err
	}
	if reservation != nil {
		_, _ = reservation.Settle(g.config.EstimateUSD)
	}
	asset, err := g.store(ctx, spec, key, video)
	if err != nil {
		return BillboardResult{}, err
	}
	return BillboardResult{Asset: asset, Key: key, Latency: g.config.Clock.Since(started)}, nil
}

func validateBillboardSpec(spec BillboardSpec) error {
	if !BillboardActions(spec.Action) {
		return fmt.Errorf("media: unknown billboard action %q", spec.Action)
	}
	if len(spec.Subject.References) == 0 || len(spec.LevelStill) == 0 {
		return errors.New("media: billboard needs identity references and a level still")
	}
	for _, image := range spec.Subject.References {
		if len(image) == 0 {
			return errors.New("media: billboard reference image is empty")
		}
	}
	return nil
}

func (g *BillboardGenerator) reserve() (*budget.Reservation, error) {
	if g.config.Budget == nil {
		return nil, nil
	}
	return g.config.Budget.Reserve(vocab.VendorFal, g.config.EstimateUSD)
}

// render submits the job, polls until it finishes or the deadline passes,
// and downloads the clip.
func (g *BillboardGenerator) render(ctx context.Context, spec BillboardSpec) ([]byte, error) {
	images := make([][]byte, 0, len(spec.Subject.References)+1)
	for _, image := range append(append([][]byte(nil), spec.Subject.References...), spec.LevelStill) {
		images = append(images, compactReference(image))
	}
	request := ports.ReferenceVideoRequest{Prompt: BillboardPrompt(spec), References: images,
		Seconds: BillboardSeconds, Resolution: BillboardResolution, Aspect: BillboardAspect}
	started := g.config.Clock.Now()
	job, err := g.config.Video.SubmitReference(ctx, request)
	if err != nil {
		return nil, err
	}
	for {
		status, pollErr := g.config.Video.Poll(ctx, job)
		if pollErr != nil {
			return nil, pollErr
		}
		if status.State == vocab.JobDone {
			return g.config.Video.Download(ctx, status.URL)
		}
		if g.config.Clock.Since(started) >= g.config.Deadline {
			return nil, ErrBillboardDeadline
		}
		if err := g.wait(ctx); err != nil {
			return nil, err
		}
	}
}

func (g *BillboardGenerator) wait(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if g.config.PollEvery <= 0 {
		return nil
	}
	timer := g.config.Clock.NewTimer(g.config.PollEvery)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C():
		return nil
	}
}

func (g *BillboardGenerator) store(ctx context.Context, spec BillboardSpec, key string, video []byte) (domain.Asset, error) {
	if len(video) == 0 {
		return domain.Asset{}, errors.New("media: billboard download was empty")
	}
	asset, err := g.config.Assets.Write(ctx, vocab.AssetVideo, "video/mp4", video, ports.AssetMeta{InputHash: key, DurationMS: BillboardSeconds * 1000})
	if err != nil {
		return domain.Asset{}, err
	}
	asset.DurationMS = BillboardSeconds * 1000
	asset.Meta = map[string]string{"action": spec.Action, "model": g.config.Model, "prompt_version": BillboardPromptVersion}
	if spec.Action == BillboardAttack || spec.Action == BillboardHit {
		asset.Meta["contact_ms"] = strconv.Itoa(BillboardContactMS)
	}
	if g.config.Cache != nil {
		encoded, encodeErr := json.Marshal(asset)
		if encodeErr != nil {
			return domain.Asset{}, encodeErr
		}
		if err := g.config.Cache.Put(ctx, BillboardCacheAdapter, key, encoded); err != nil {
			return domain.Asset{}, err
		}
	}
	return asset, nil
}
