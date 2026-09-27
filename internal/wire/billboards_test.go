package wire

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func liveBillboardConfig() config.Config {
	return config.Config{
		Adapters: map[string]config.AdapterConfig{"billboard": {Vendor: "fal", Mode: "live", APIKey: "k"}},
		Features: config.FeatureConfig{LivePCLoops: true},
		Budget:   config.BudgetConfig{PerRunUSD: 0.95, HardUSD: 8},
	}
}

func referenceEvents(seat domain.SeatID, sheet string) []domain.Event {
	events := []domain.Event{domain.AssetReady{Slot: "reference:" + string(rune('0'+seat)) + ":sheet", Asset: domain.Asset{ID: domain.AssetID(sheet)}}}
	for _, angle := range vocab.ReferenceAngles() {
		events = append(events, domain.AssetReady{Slot: "reference:" + string(rune('0'+seat)) + ":" + string(angle), Asset: domain.Asset{ID: domain.AssetID(string(angle) + ".png")}})
	}
	return events
}

func TestBillboardAdapter_selection(t *testing.T) {
	cases := []struct {
		name     string
		adapters map[string]config.AdapterConfig
		live     bool
	}{
		{"billboard entry", map[string]config.AdapterConfig{"billboard": {Vendor: "fal", Mode: "live", APIKey: "k"}}, true},
		{"fal video", map[string]config.AdapterConfig{"video": {Vendor: "fal", Mode: "live", APIKey: "k"}}, true},
		{"segmind video", map[string]config.AdapterConfig{"video": {Vendor: "segmind", Mode: "live", APIKey: "k"}}, false},
		{"no key", map[string]config.AdapterConfig{"billboard": {Vendor: "fal", Mode: "live"}}, false},
		{"fake", map[string]config.AdapterConfig{"billboard": {Vendor: "fal", Mode: "fake", APIKey: "k"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, live := billboardAdapter(config.Config{Adapters: tc.adapters}); live != tc.live {
				t.Fatalf("live=%v", live)
			}
		})
	}
}

func TestBillboardHub_startsLoopsOnceWhenTurnaroundReady(t *testing.T) {
	hub := newBillboardHub(liveBillboardConfig(), nil, "/splat/scenes/64bb46d5.json")
	if hub.sceneID != "64bb46d5" {
		t.Fatalf("scene=%q", hub.sceneID)
	}
	events := referenceEvents(1, "sheet-a.png")
	for index, event := range events {
		effect, ok := hub.observe(event, func(domain.SeatID) string { return "paladin" })
		if last := index == len(events)-1; ok != last {
			t.Fatalf("event %d: started=%v", index, ok)
		}
		if ok && (effect.Seat != 1 || len(effect.Clips) != 3 || effect.Clips[2] != media.BillboardHit) {
			t.Fatalf("effect=%+v", effect)
		}
	}
	if _, ok := hub.observe(events[1], nil); ok {
		t.Fatal("same turnaround started twice")
	}
	refs, class := hub.subject(1)
	if !refs.Ready() || class != "paladin" {
		t.Fatalf("refs=%+v class=%q", refs, class)
	}
	var seat2 domain.GenerateBillboardLoops
	for _, event := range referenceEvents(2, "sheet-b.png") {
		if effect, ok := hub.observe(event, nil); ok {
			seat2 = effect
		}
	}
	if len(seat2.Clips) != 2 {
		t.Fatalf("seat 2 loops=%v, want idle and attack", seat2.Clips)
	}
	if _, ok := hub.observe(domain.AssetReady{Slot: "portrait:1"}, nil); ok {
		t.Fatal("unrelated slot started loops")
	}
	if _, ok := hub.observe(domain.Join{}, nil); ok {
		t.Fatal("non-asset event started loops")
	}
}

func TestBillboardHub_disabledModesNeverStart(t *testing.T) {
	safe := liveBillboardConfig()
	safe.Features.SequenceMode = true
	off := liveBillboardConfig()
	off.Features.LivePCLoops = false
	for name, cfg := range map[string]config.Config{"safe": safe, "flag off": off, "fake": {}} {
		t.Run(name, func(t *testing.T) {
			hub := newBillboardHub(cfg, nil, "")
			for _, event := range referenceEvents(1, "s.png") {
				if _, ok := hub.observe(event, nil); ok {
					t.Fatal("loops started")
				}
			}
		})
	}
}

func TestBillboardEngine_decoratesTokensAndResetsPerRun(t *testing.T) {
	catalogue := []domain.Asset{{ID: "thrall_loop_idle", URL: "/assets/t-idle.mp4"}, {ID: "thrall_loop_attack"}, {ID: "battlefield_tavern_flat", URL: "/assets/x.png"}}
	hub := newBillboardHub(liveBillboardConfig(), catalogue, "")
	tokens := []domain.TokenView{{ID: "pc-1", Kind: "pc-paladin"}, {ID: "thrall", Kind: "thrall"}, {ID: "pc-2", Kind: "pc-rogue", Clips: map[string]domain.AssetID{"idle": "/assets/own.mp4"}}}
	inner := &fakes.FakeEngine{ViewValue: domain.View{Seats: []domain.SeatView{{Seat: 1, Character: &domain.Character{Class: "Paladin"}}}, Combat: &domain.CombatView{Tokens: tokens}, Battlefield: &domain.BattlefieldView{Tokens: tokens}}}
	engine := newBillboardEngine(inner, hub)
	var started []domain.Effect
	for _, event := range referenceEvents(1, "sheet.png") {
		started = append(started, engine.Step(domain.Envelope{Event: event}).Effects...)
	}
	if len(started) != 1 {
		t.Fatalf("effects=%v", started)
	}
	if _, class := hub.subject(1); class != "Paladin" {
		t.Fatalf("class=%q", class)
	}
	engine.Step(domain.Envelope{Event: domain.AssetReady{Slot: "billboard:1:idle", Asset: domain.Asset{ID: "abc.mp4"}}})
	engine.Step(domain.Envelope{Event: domain.AssetReady{Slot: "billboard:2:idle", Asset: domain.Asset{ID: "b.mp4", URL: "/assets/b.mp4"}}})
	view := engine.View()
	for _, list := range [][]domain.TokenView{view.Combat.Tokens, view.Battlefield.Tokens} {
		if list[0].Clips["idle"] != "/assets/abc.mp4" || list[1].Clips["idle"] != "/assets/t-idle.mp4" || len(list[1].Clips) != 1 || list[2].Clips["idle"] != "/assets/own.mp4" {
			t.Fatalf("tokens=%+v", list)
		}
	}
	if tokens[0].Clips != nil {
		t.Fatal("decoration mutated the engine's tokens")
	}
	newBillboardEngine(inner, hub)
	if view := hub.decorate(inner.View()); view.Combat.Tokens[0].Clips != nil {
		t.Fatal("a new run kept the previous run's loops")
	}
	if newBillboardEngine(inner, nil) != ports.Engine(inner) {
		t.Fatal("nil hub must return the engine unchanged")
	}
}

// countingReferenceVideo is a ports.ReferenceVideoGen that finishes at once.
type countingReferenceVideo struct {
	mu      sync.Mutex
	submits int
}

func (c *countingReferenceVideo) SubmitReference(context.Context, ports.ReferenceVideoRequest) (ports.VideoJob, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.submits++
	return ports.VideoJob{ID: "j"}, nil
}

func (c *countingReferenceVideo) Poll(context.Context, ports.VideoJob) (ports.VideoStatus, error) {
	return ports.VideoStatus{State: vocab.JobDone, URL: "u"}, nil
}

func (c *countingReferenceVideo) Download(context.Context, string) ([]byte, error) {
	return []byte("mp4 bytes"), nil
}

type eventInbox struct {
	mu     sync.Mutex
	events []domain.Event
}

func (e *eventInbox) Post(_ context.Context, env domain.Envelope) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, env.Event)
	return true
}

func TestBillboardLoopsExecutor_generatesThenHitsCache(t *testing.T) {
	dataDir := t.TempDir()
	assets := newAssetStore(dataDir)
	cache := &fakes.FakeCache{}
	base, err := billboardServices(config.Config{}, assets, cache, nil)
	if err != nil || base.Video != nil {
		t.Fatalf("base=%+v err=%v", base, err)
	}
	video := &countingReferenceVideo{}
	base.Video = video
	base.PollEvery = -1
	hub := newBillboardHub(liveBillboardConfig(), []domain.Asset{{ID: "thrall_loop_idle", URL: "/assets/t.mp4"}}, "")
	var refs []domain.AssetID
	for _, angle := range vocab.ReferenceAngles() {
		asset, writeErr := assets.Write(t.Context(), vocab.AssetImage, "image/png", []byte("img-"+string(angle)), ports.AssetMeta{})
		if writeErr != nil {
			t.Fatal(writeErr)
		}
		refs = append(refs, asset.ID)
		hub.observe(domain.AssetReady{Slot: "reference:1:" + string(angle), Asset: asset}, nil)
	}
	hub.observe(domain.AssetReady{Slot: "reference:1:sheet", Asset: domain.Asset{ID: refs[0]}}, func(domain.SeatID) string { return "ranger" })
	subject, err := hub.loadSubject(t.Context(), 1, assets.Read)
	if err != nil || len(subject.References) != 2 || subject.Weapon != "longbow" {
		t.Fatalf("subject=%+v err=%v", subject, err)
	}
	if _, err := hub.loadSubject(t.Context(), 2, assets.Read); err == nil {
		t.Fatal("seat without turnaround produced a subject")
	}
	in := &eventInbox{}
	billboardLoopsExecutor(hub, base, nil)(t.Context(), domain.GenerateBillboardLoops{Seat: 0, Clips: []string{"idle"}}, domain.Scope{}, in)
	if ready, ok := in.events[0].(domain.AssetReady); !ok || ready.Asset.URL != "/assets/t.mp4" || video.submits != 0 {
		t.Fatalf("thrall catalogue: %#v submits=%d", in.events, video.submits)
	}
	// Seat 1 has no level still in this test tree: the loop fails cleanly.
	in = &eventInbox{}
	billboardLoopsExecutor(hub, base, nil)(t.Context(), domain.GenerateBillboardLoops{Seat: 1, Clips: []string{"idle"}}, domain.Scope{}, in)
	if failed, ok := in.events[0].(domain.AssetFailed); !ok || failed.Slot != "billboard:1:idle" {
		t.Fatalf("events=%#v", in.events)
	}
	// With a level still: generate once, then serve the cache with no call.
	hub.still = func() ([]byte, error) { return []byte("level still"), nil }
	var ready []domain.AssetReady
	for range 2 {
		in = &eventInbox{}
		billboardLoopsExecutor(hub, base, discardLogger())(t.Context(), domain.GenerateBillboardLoops{Seat: 1, Clips: []string{"idle", "attack"}}, domain.Scope{}, in)
		for _, event := range in.events {
			value, ok := event.(domain.AssetReady)
			if !ok {
				t.Fatalf("event=%#v", event)
			}
			ready = append(ready, value)
		}
	}
	if video.submits != 2 || len(ready) != 4 {
		t.Fatalf("submits=%d ready=%d", video.submits, len(ready))
	}
	if _, ledger := hub.run(); ledger.Costs().Spent < 0.9 || ledger.Costs().Spent > 0.93 {
		t.Fatalf("spent=%+v", ledger.Costs())
	}
	// The per-run cap ($0.95) now refuses a third paid clip.
	in = &eventInbox{}
	billboardLoopsExecutor(hub, base, nil)(t.Context(), domain.GenerateBillboardLoops{Seat: 1, Clips: []string{"hit"}}, domain.Scope{}, in)
	if failed, ok := in.events[0].(domain.AssetFailed); !ok || failed.FailureKind != vocab.ErrRateLimited || video.submits != 2 {
		t.Fatalf("capped: %#v submits=%d", in.events, video.submits)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := hub.levelStill(cancelled); err == nil {
		t.Fatal("cancelled level still read succeeded")
	}
}

func TestBillboardLog_andLook(t *testing.T) {
	billboardLog(nil)("slot", media.BillboardResult{}, nil)
	billboardLog(discardLogger())("slot", media.BillboardResult{}, errors.New("x"))
	billboardLog(discardLogger())("slot", media.BillboardResult{Cached: true}, nil)
	if lookFor("") != "hero" || lookFor(" Rogue") != "rogue hero" {
		t.Fatal("look")
	}
	if levelStillName("64bb46d5", "tactical") != "level_still_64bb46d5_tactical" {
		t.Fatal("level still name")
	}
	if defaultBillboardHub(config.Config{}).enabled {
		t.Fatal("default hub must never generate")
	}
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestBillboardTool_cacheSurvivesReopen(t *testing.T) {
	dataDir := t.TempDir()
	spec := media.BillboardSpec{Action: media.BillboardIdle, Subject: media.BillboardSubject{References: [][]byte{[]byte("front")}}, LevelStill: []byte("level")}
	tool, err := NewBillboardToolForTest(t.Context(), dataDir, &countingReferenceVideo{})
	if err != nil {
		t.Fatal(err)
	}
	first, err := tool.Generate(t.Context(), spec)
	if err != nil || first.Cached || tool.VendorCalls() != 3 || tool.SpentUSD() < 0.45 {
		t.Fatalf("first=%+v err=%v calls=%d spent=%v", first, err, tool.VendorCalls(), tool.SpentUSD())
	}
	if tool.Key(spec) != first.Key || filepath.Base(tool.AssetPath(first.Asset)) != string(first.Asset.ID) {
		t.Fatal("key or asset path mismatch")
	}
	if err := tool.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenBillboardTool(t.Context(), dataDir, "", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	second, err := reopened.Generate(t.Context(), spec)
	if err != nil || !second.Cached || second.Asset.SHA256 != first.Asset.SHA256 || reopened.VendorCalls() != 0 {
		t.Fatalf("second=%+v err=%v calls=%d", second, err, reopened.VendorCalls())
	}
	other := spec
	other.Action = media.BillboardAttack
	if _, err := reopened.Generate(t.Context(), other); !errors.Is(err, media.ErrBillboardDisabled) {
		t.Fatalf("keyless miss err=%v", err)
	}
}
