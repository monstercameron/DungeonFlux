package wire

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// billboardHub is the wire-side state of the live billboard loops (plan
// §0.21.5): it watches reference-turnaround slots, starts a seat's loops once
// its turnaround is Ready, remembers the loops that arrive, and decorates the
// combat tokens with them. The pure engine is not involved.
type billboardHub struct {
	mu        sync.Mutex
	enabled   bool
	perRunUSD float64
	sceneID   string
	thrall    map[string]string
	prepared  map[string]map[string]string
	refs      map[domain.SeatID]media.ReferenceAssets
	classes   map[domain.SeatID]string
	started   map[domain.SeatID]domain.AssetID
	clips     map[domain.SeatID]map[string]string
	ledger    *budget.Ledger
	still     func() ([]byte, error)
	runCtx    context.Context
	cancel    context.CancelFunc
}

// newBillboardHub reads the billboard settings: loops are generated only
// when live_pc_loops is on, sequence (safe) mode is off, and a live fal
// adapter with a key is configured (billboardAdapter).
func newBillboardHub(cfg config.Config, oneShotCatalogue []domain.Asset, sceneURL string) *billboardHub {
	_, live := billboardAdapter(cfg)
	hub := &billboardHub{
		enabled:   live && cfg.Features.LivePCLoops && !cfg.Features.SequenceMode,
		perRunUSD: cfg.Budget.PerRunUSD,
		sceneID:   sceneID(sceneURL),
		thrall:    thrallClips(oneShotCatalogue),
		prepared:  preparedHeroClips(oneShotCatalogue, sceneURL),
	}
	hub.still = func() ([]byte, error) {
		return loadManifestAsset(manifestPath(), levelStillName(hub.sceneID, "tactical"))
	}
	hub.reset()
	return hub
}

// billboardAdapter returns the fal adapter used for billboard loops: an
// adapter named "billboard", else the "video" adapter when it is fal.
func billboardAdapter(cfg config.Config) (config.AdapterConfig, bool) {
	for _, name := range []string{"billboard", "video"} {
		adapter, ok := cfg.Adapters[name]
		if ok && strings.EqualFold(adapter.Vendor, "fal") {
			return adapter, adapter.Mode == "live" && strings.TrimSpace(adapter.APIKey) != ""
		}
	}
	return config.AdapterConfig{}, false
}

func sceneID(sceneURL string) string {
	name := sceneURL[strings.LastIndex(sceneURL, "/")+1:]
	return strings.TrimSuffix(name, ".json")
}

func thrallClips(catalogue []domain.Asset) map[string]string {
	clips := make(map[string]string)
	for _, asset := range catalogue {
		action, ok := strings.CutPrefix(string(asset.ID), "thrall_loop_")
		if ok && asset.URL != "" && media.BillboardActions(action) {
			clips[action] = asset.URL
		}
	}
	return clips
}

// reset starts a new run: loops of the previous run are cancelled and
// forgotten, and the per-run fal cap starts again. Prepared catalogue loops
// remain available for the matching hero class and battlefield.
func (h *billboardHub) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cancel != nil {
		h.cancel()
	}
	h.runCtx, h.cancel = context.WithCancel(context.Background())
	h.refs = make(map[domain.SeatID]media.ReferenceAssets)
	h.classes = make(map[domain.SeatID]string)
	h.started = make(map[domain.SeatID]domain.AssetID)
	h.clips = make(map[domain.SeatID]map[string]string)
	caps := map[vocab.VendorName]float64{}
	if h.perRunUSD > 0 {
		caps[vocab.VendorFal] = h.perRunUSD
	}
	h.ledger, _ = budget.NewLedger(caps)
}

// run returns the context and ledger of the current run.
func (h *billboardHub) run() (context.Context, *budget.Ledger) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.runCtx, h.ledger
}

// pcBillboardActions is the §0.21.5 slot rule: PC 1 idle, attack, hit;
// PC 2 idle, attack (no PC fall loop).
func pcBillboardActions(seat domain.SeatID) []string {
	if seat == 1 {
		return []string{media.BillboardIdle, media.BillboardAttack, media.BillboardHit}
	}
	return []string{media.BillboardIdle, media.BillboardAttack}
}

// observe records reference and billboard slots from an event and returns
// the loop effect to start when a seat's turnaround has just become Ready.
func (h *billboardHub) observe(event domain.Event, class func(domain.SeatID) string) (domain.GenerateBillboardLoops, bool) {
	ready, ok := event.(domain.AssetReady)
	if !ok {
		return domain.GenerateBillboardLoops{}, false
	}
	kind, seat, name, ok := parseSeatSlot(ready.Slot)
	if !ok {
		return domain.GenerateBillboardLoops{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if kind == "billboard" {
		if h.clips[seat] == nil {
			h.clips[seat] = make(map[string]string)
		}
		h.clips[seat][name] = assetURLOf(ready.Asset)
		return domain.GenerateBillboardLoops{}, false
	}
	refs := h.refs[seat]
	if name == "sheet" {
		refs.Sheet = ready.Asset.ID
	} else {
		if refs.Angles == nil {
			refs.Angles = make(map[vocab.ReferenceAngle]domain.AssetID)
		}
		refs.Angles[vocab.ReferenceAngle(name)] = ready.Asset.ID
	}
	h.refs[seat] = refs
	if !h.enabled || !refs.Ready() || h.started[seat] == refs.Sheet {
		return domain.GenerateBillboardLoops{}, false
	}
	h.started[seat] = refs.Sheet
	if class != nil {
		h.classes[seat] = class(seat)
	}
	return domain.GenerateBillboardLoops{Seat: seat, Clips: pcBillboardActions(seat)}, true
}

func parseSeatSlot(slot string) (string, domain.SeatID, string, bool) {
	parts := strings.Split(slot, ":")
	if len(parts) != 3 || (parts[0] != "reference" && parts[0] != "billboard") {
		return "", 0, "", false
	}
	var seat domain.SeatID
	if _, err := fmt.Sscanf(parts[1], "%d", &seat); err != nil || seat <= 0 {
		return "", 0, "", false
	}
	return parts[0], seat, parts[2], true
}

func assetURLOf(asset domain.Asset) string {
	if asset.URL != "" {
		return asset.URL
	}
	return "/assets/" + string(asset.ID)
}

// levelStill reads the scene's clean TACTICAL still from the build-time
// manifest; it conditions the loops' light and camera angle.
func (h *billboardHub) levelStill(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return h.still()
}

// subject returns the seat's identity references (front, three-quarter)
// and class captured when its loops started.
func (h *billboardHub) subject(seat domain.SeatID) (media.ReferenceAssets, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.refs[seat], h.classes[seat]
}

// decorate returns view with loop URLs on the combat tokens: PC token
// "pc-<seat>" gets that seat's loops, the thrall gets its build-time loops.
func (h *billboardHub) decorate(view domain.View) domain.View {
	h.mu.Lock()
	defer h.mu.Unlock()
	if view.Combat != nil {
		combat := *view.Combat
		combat.Tokens = h.decorateTokens(combat.Tokens)
		view.Combat = &combat
	}
	if view.Battlefield != nil {
		field := *view.Battlefield
		field.Tokens = h.decorateTokens(field.Tokens)
		view.Battlefield = &field
	}
	return view
}

func (h *billboardHub) decorateTokens(tokens []domain.TokenView) []domain.TokenView {
	if len(tokens) == 0 {
		return tokens
	}
	out := append([]domain.TokenView(nil), tokens...)
	for index, token := range out {
		clips := h.clipsFor(token)
		if len(clips) == 0 {
			continue
		}
		merged := make(map[string]domain.AssetID, len(token.Clips)+len(clips))
		for action, id := range token.Clips {
			merged[action] = id
		}
		for action, url := range clips {
			if _, exists := merged[action]; !exists {
				merged[action] = domain.AssetID(url)
			}
		}
		out[index].Clips = merged
	}
	return out
}

func (h *billboardHub) clipsFor(token domain.TokenView) map[string]string {
	if token.Kind == "thrall" || token.ID == "thrall" {
		return h.thrall
	}
	var seat domain.SeatID
	if _, err := fmt.Sscanf(string(token.ID), "pc-%d", &seat); err != nil {
		return nil
	}
	clips := make(map[string]string)
	for action, url := range h.preparedClipsFor(token) {
		clips[action] = url
	}
	for action, url := range h.clips[seat] {
		clips[action] = url
	}
	return clips
}
