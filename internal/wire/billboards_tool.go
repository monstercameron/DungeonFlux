package wire

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync/atomic"

	videofal "github.com/monstercameron/DungeonFlux/internal/adapters/video/fal"
	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// BillboardTool renders billboard loops outside a room (build time and
// verification) with the server's generator, SQLite cache table, and asset
// store layout, so a loop made here is a cache hit for a server started on
// the same data directory.
type BillboardTool struct {
	storage   *Storage
	assets    *assetStore
	generator *media.BillboardGenerator
	calls     *countingVideo
	ledger    *budget.Ledger
}

// OpenBillboardTool opens dataDir's store. An empty falKey makes the tool
// cache-only; maxUSD (> 0) caps fal spend for this tool's lifetime.
func OpenBillboardTool(ctx context.Context, dataDir, falKey string, maxUSD float64, logger *slog.Logger) (*BillboardTool, error) {
	storage, err := OpenStorage(ctx, dataDir, logger)
	if err != nil {
		return nil, err
	}
	assets := newAssetStore(dataDir)
	cfg := config.Config{}
	if falKey != "" {
		cfg.Adapters = map[string]config.AdapterConfig{"billboard": {Vendor: "fal", Mode: "live", APIKey: falKey}}
	}
	base, err := billboardServices(cfg, assets, storage.Cache, logger)
	if err != nil {
		_ = storage.Close()
		return nil, err
	}
	tool := &BillboardTool{storage: storage, assets: assets, calls: &countingVideo{next: base.Video}}
	if base.Video != nil {
		base.Video = tool.calls
	}
	caps := map[vocab.VendorName]float64{}
	if maxUSD > 0 {
		caps[vocab.VendorFal] = maxUSD
	}
	tool.ledger, _ = budget.NewLedger(caps)
	base.Budget = tool.ledger
	tool.generator = media.NewBillboardGenerator(base)
	return tool, nil
}

// NewBillboardToolForTest wraps a scripted vendor; tests only.
func NewBillboardToolForTest(ctx context.Context, dataDir string, video ports.ReferenceVideoGen) (*BillboardTool, error) {
	tool, err := OpenBillboardTool(ctx, dataDir, "", 0, nil)
	if err != nil {
		return nil, err
	}
	pool, _ := media.NewPool(map[vocab.VendorName]int{vocab.VendorFal: falConcurrency})
	tool.calls = &countingVideo{next: video}
	tool.generator = media.NewBillboardGenerator(media.BillboardGeneratorConfig{Video: tool.calls, Model: videofal.ReferenceModel,
		Assets: tool.assets, Cache: tool.storage.Cache, Source: tool.assets.Read, Pool: pool, Budget: tool.ledger, PollEvery: -1})
	return tool, nil
}

// Generate returns the cached loop or renders a new one.
func (t *BillboardTool) Generate(ctx context.Context, spec media.BillboardSpec) (media.BillboardResult, error) {
	return t.generator.Generate(ctx, spec)
}

// Key returns spec's cache key.
func (t *BillboardTool) Key(spec media.BillboardSpec) string { return t.generator.Key(spec) }

// VendorCalls counts fal submissions and polls made through this tool.
func (t *BillboardTool) VendorCalls() int64 { return t.calls.count.Load() }

// SpentUSD reports the settled fal estimate for this tool's renders.
func (t *BillboardTool) SpentUSD() float64 { return t.ledger.Costs().Spent }

// AssetPath is the file that holds asset.
func (t *BillboardTool) AssetPath(asset domain.Asset) string {
	return filepath.Join(t.assets.root, filepath.Base(string(asset.ID)))
}

// Close releases the store.
func (t *BillboardTool) Close() error { return t.storage.Close() }

// countingVideo counts every vendor request, proving a cache hit made none.
type countingVideo struct {
	next  ports.ReferenceVideoGen
	count atomic.Int64
}

func (c *countingVideo) SubmitReference(ctx context.Context, req ports.ReferenceVideoRequest) (ports.VideoJob, error) {
	c.count.Add(1)
	return c.next.SubmitReference(ctx, req)
}

func (c *countingVideo) Poll(ctx context.Context, job ports.VideoJob) (ports.VideoStatus, error) {
	c.count.Add(1)
	return c.next.Poll(ctx, job)
}

func (c *countingVideo) Download(ctx context.Context, url string) ([]byte, error) {
	c.count.Add(1)
	return c.next.Download(ctx, url)
}
