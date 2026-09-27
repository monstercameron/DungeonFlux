package media

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type recordingInbox struct {
	mu     sync.Mutex
	events []domain.Event
}

func (r *recordingInbox) Post(_ context.Context, env domain.Envelope) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, env.Event)
	return true
}

func (r *recordingInbox) bySlot() map[string]domain.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]domain.Event)
	for _, event := range r.events {
		switch value := event.(type) {
		case domain.AssetReady:
			out[value.Slot] = value
		case domain.AssetFailed:
			out[value.Slot] = value
		}
	}
	return out
}

func loopsConfig(generator *BillboardGenerator, generate bool) BillboardLoopsConfig {
	return BillboardLoopsConfig{
		Generator: generator, Generate: generate,
		Subject: func(_ context.Context, seat domain.SeatID) (BillboardSubject, error) {
			if seat == 9 {
				return BillboardSubject{}, errors.New("no references yet")
			}
			return heroSpec(BillboardIdle).Subject, nil
		},
		LevelStill: func(context.Context) ([]byte, error) { return []byte("level"), nil },
	}
}

func TestBillboardLoopsExecutor_catalogueCacheGenerate(t *testing.T) {
	video := &scriptedReferenceVideo{}
	assets := &memoryAssets{}
	cache := &fakes.FakeCache{}
	generator := newTestGenerator(video, assets, cache, nil)
	config := loopsConfig(generator, true)
	config.Catalogue = func(seat domain.SeatID, action string) (domain.Asset, bool) {
		return domain.Asset{ID: "thrall_loop_fall"}, seat == 0 && action == BillboardFall
	}
	var results []string
	var mu sync.Mutex
	config.OnResult = func(slot string, result BillboardResult, err error) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, slot)
	}
	in := &recordingInbox{}
	NewBillboardLoopsExecutor(config).Execute(t.Context(), domain.GenerateBillboardLoops{Seat: 1, Clips: []string{BillboardIdle, BillboardAttack}}, domain.Scope{}, in)
	got := in.bySlot()
	for _, slot := range []string{"billboard:1:idle", "billboard:1:attack"} {
		if _, ok := got[slot].(domain.AssetReady); !ok {
			t.Fatalf("%s: %#v", slot, got[slot])
		}
	}
	if video.submitCount() != 2 {
		t.Fatalf("submits=%d", video.submitCount())
	}
	sort.Strings(results)
	if len(results) != 2 || results[0] != "billboard:1:attack" {
		t.Fatalf("results=%v", results)
	}
	// The thrall's fall loop comes from the build-time catalogue, no call.
	in = &recordingInbox{}
	NewBillboardLoopsExecutor(config).Execute(t.Context(), domain.GenerateBillboardLoops{Seat: 0, Clips: []string{BillboardFall}}, domain.Scope{}, in)
	if ready, ok := in.bySlot()["billboard:0:fall"].(domain.AssetReady); !ok || ready.Asset.ID != "thrall_loop_fall" || video.submitCount() != 2 {
		t.Fatalf("catalogue: %#v submits=%d", in.bySlot(), video.submitCount())
	}
	// Generation off: a cached loop is still served, a miss fails.
	cacheOnly := loopsConfig(generator, false)
	in = &recordingInbox{}
	NewBillboardLoopsExecutor(cacheOnly).Execute(t.Context(), domain.GenerateBillboardLoops{Seat: 2, Clips: []string{BillboardIdle, BillboardHit}}, domain.Scope{}, in)
	got = in.bySlot()
	if _, ok := got["billboard:2:idle"].(domain.AssetReady); !ok {
		t.Fatalf("cached idle not served: %#v", got["billboard:2:idle"])
	}
	if failed, ok := got["billboard:2:hit"].(domain.AssetFailed); !ok || failed.FailureKind != vocab.ErrUnavailable {
		t.Fatalf("uncached hit: %#v", got["billboard:2:hit"])
	}
	if video.submitCount() != 2 {
		t.Fatalf("cache-only mode called the vendor: %d", video.submitCount())
	}
}

func TestBillboardLoopsExecutor_failures(t *testing.T) {
	capped, _ := budget.NewLedger(nil, 0.1)
	cases := []struct {
		name   string
		config BillboardLoopsConfig
		seat   domain.SeatID
		action string
		want   vocab.ErrKind
	}{
		{"no generator", BillboardLoopsConfig{}, 1, BillboardIdle, vocab.ErrUnavailable},
		{"unknown action", loopsConfig(newTestGenerator(&scriptedReferenceVideo{}, &memoryAssets{}, &fakes.FakeCache{}, nil), true), 1, "dance", vocab.ErrUnavailable},
		{"no references", loopsConfig(newTestGenerator(&scriptedReferenceVideo{}, &memoryAssets{}, &fakes.FakeCache{}, nil), true), 9, BillboardIdle, vocab.ErrUnavailable},
		{"budget", loopsConfig(newTestGenerator(&scriptedReferenceVideo{}, &memoryAssets{}, &fakes.FakeCache{}, capped), true), 1, BillboardIdle, vocab.ErrRateLimited},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &recordingInbox{}
			NewBillboardLoopsExecutor(tc.config).Execute(t.Context(), domain.GenerateBillboardLoops{Seat: tc.seat, Clips: []string{tc.action}}, domain.Scope{}, in)
			failed, ok := in.bySlot()[BillboardSlot(tc.seat, tc.action)].(domain.AssetFailed)
			if !ok || failed.FailureKind != tc.want {
				t.Fatalf("event=%#v", in.bySlot())
			}
		})
	}
	level := loopsConfig(newTestGenerator(&scriptedReferenceVideo{}, &memoryAssets{}, &fakes.FakeCache{}, nil), true)
	level.LevelStill = func(context.Context) ([]byte, error) { return nil, context.DeadlineExceeded }
	in := &recordingInbox{}
	NewBillboardLoopsExecutor(level).Execute(t.Context(), domain.GenerateBillboardLoops{Seat: 1, Clips: []string{BillboardIdle}}, domain.Scope{}, in)
	if failed, ok := in.bySlot()["billboard:1:idle"].(domain.AssetFailed); !ok || failed.FailureKind != vocab.ErrTimeout {
		t.Fatalf("level failure: %#v", in.bySlot())
	}
	if billboardFailure(ErrBillboardDeadline) != vocab.ErrTimeout {
		t.Fatal("deadline kind")
	}
}

func TestBillboardLoopsExecutor_capAdmitsLoopsInActionOrder(t *testing.T) {
	for range 20 {
		ledger, _ := budget.NewLedger(map[vocab.VendorName]float64{vocab.VendorFal: 0.95})
		video := &scriptedReferenceVideo{}
		config := loopsConfig(newTestGenerator(video, &memoryAssets{}, &fakes.FakeCache{}, ledger), true)
		in := &recordingInbox{}
		NewBillboardLoopsExecutor(config).Execute(t.Context(), domain.GenerateBillboardLoops{Seat: 1, Clips: []string{BillboardIdle, BillboardAttack, BillboardHit}}, domain.Scope{}, in)
		got := in.bySlot()
		_, idle := got["billboard:1:idle"].(domain.AssetReady)
		_, attack := got["billboard:1:attack"].(domain.AssetReady)
		hit, capped := got["billboard:1:hit"].(domain.AssetFailed)
		if !idle || !attack || !capped || hit.FailureKind != vocab.ErrRateLimited || video.submitCount() != 2 {
			t.Fatalf("events=%#v submits=%d", got, video.submitCount())
		}
	}
}
