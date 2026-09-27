package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/png"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// scriptedReferenceVideo is a hand-written ports.ReferenceVideoGen double.
type scriptedReferenceVideo struct {
	mu        sync.Mutex
	submits   []ports.ReferenceVideoRequest
	polls     int
	pendingN  int // polls answered "running" before "done"
	submitErr error
	pollErr   error
	download  []byte
	onPoll    func()
}

func (s *scriptedReferenceVideo) SubmitReference(_ context.Context, req ports.ReferenceVideoRequest) (ports.VideoJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.submits = append(s.submits, req)
	if s.submitErr != nil {
		return ports.VideoJob{}, s.submitErr
	}
	return ports.VideoJob{Vendor: "fal", ID: fmt.Sprintf("job-%d", len(s.submits))}, nil
}

func (s *scriptedReferenceVideo) Poll(context.Context, ports.VideoJob) (ports.VideoStatus, error) {
	s.mu.Lock()
	s.polls++
	polls, onPoll := s.polls, s.onPoll
	s.mu.Unlock()
	if onPoll != nil {
		onPoll()
	}
	if s.pollErr != nil {
		return ports.VideoStatus{}, s.pollErr
	}
	if s.pendingN >= 0 && polls <= s.pendingN {
		return ports.VideoStatus{State: vocab.JobRunning}, nil
	}
	if s.pendingN < 0 {
		return ports.VideoStatus{State: vocab.JobQueued}, nil
	}
	return ports.VideoStatus{State: vocab.JobDone, URL: "https://cdn.test/clip.mp4"}, nil
}

func (s *scriptedReferenceVideo) Download(context.Context, string) ([]byte, error) {
	if s.download == nil {
		return []byte("mp4:" + fmt.Sprint(len(s.submits))), nil
	}
	return s.download, nil
}

func (s *scriptedReferenceVideo) submitCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.submits)
}

// memoryAssets is a content-addressed ports.AssetWriter plus a reader.
type memoryAssets struct {
	mu    sync.Mutex
	files map[domain.AssetID][]byte
}

func (m *memoryAssets) Write(_ context.Context, kind vocab.AssetKind, mime string, data []byte, meta ports.AssetMeta) (domain.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.files == nil {
		m.files = make(map[domain.AssetID][]byte)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(data))
	id := domain.AssetID(sum + ".mp4")
	m.files[id] = append([]byte(nil), data...)
	return domain.Asset{ID: id, SHA256: sum, URL: "/assets/" + string(id), Kind: string(kind), MIME: mime, DurationMS: meta.DurationMS}, nil
}

func (m *memoryAssets) read(_ context.Context, id domain.AssetID) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.files[id]
	if !ok {
		return nil, errors.New("missing")
	}
	return data, nil
}

func heroSpec(action string) BillboardSpec {
	return BillboardSpec{Action: action, Subject: BillboardSubject{Look: "paladin in plate", Weapon: "longsword", References: [][]byte{[]byte("front"), []byte("three")}}, LevelStill: []byte("level")}
}

func newTestGenerator(video ports.ReferenceVideoGen, assets *memoryAssets, cache ports.Cache, ledger *budget.Ledger) *BillboardGenerator {
	return NewBillboardGenerator(BillboardGeneratorConfig{Video: video, Model: "m", Assets: assets, Cache: cache, Source: assets.read, Budget: ledger, PollEvery: -1})
}

func TestBillboardGenerator_generatesThenServesCacheWithoutVendorCalls(t *testing.T) {
	video := &scriptedReferenceVideo{pendingN: 2}
	assets := &memoryAssets{}
	cache := &fakes.FakeCache{}
	ledger, _ := budget.NewLedger(nil, 5)
	generator := newTestGenerator(video, assets, cache, ledger)
	first, err := generator.Generate(t.Context(), heroSpec(BillboardAttack))
	if err != nil || first.Cached || first.Asset.ID == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if first.Asset.Meta["contact_ms"] != "1200" || first.Asset.DurationMS != 4000 || first.Asset.Meta["prompt_version"] != BillboardPromptVersion {
		t.Fatalf("meta=%+v duration=%d", first.Asset.Meta, first.Asset.DurationMS)
	}
	request := video.submits[0]
	if len(request.References) != 3 || string(request.References[2]) != "level" || request.Aspect != "9:16" || request.Resolution != "480p" || request.Seconds != 4 {
		t.Fatalf("request=%+v", request)
	}
	if video.polls != 3 {
		t.Fatalf("polls=%d, want 3", video.polls)
	}
	if spent := ledger.Costs().Spent; spent < 0.45 || spent > 0.47 {
		t.Fatalf("spent=%v", spent)
	}
	// A new generator over the same cache (a restart) makes zero vendor calls.
	again := &scriptedReferenceVideo{}
	second, err := newTestGenerator(again, assets, cache, ledger).Generate(t.Context(), heroSpec(BillboardAttack))
	if err != nil || !second.Cached || second.Asset.SHA256 != first.Asset.SHA256 || again.submitCount() != 0 {
		t.Fatalf("second=%+v err=%v submits=%d", second, err, again.submitCount())
	}
	if spent := ledger.Costs().Spent; spent > 0.47 {
		t.Fatalf("cache hit was charged: %v", spent)
	}
}

func TestBillboardGenerator_missingFileIsAMiss(t *testing.T) {
	video := &scriptedReferenceVideo{}
	assets := &memoryAssets{}
	cache := &fakes.FakeCache{}
	generator := newTestGenerator(video, assets, cache, nil)
	if _, err := generator.Generate(t.Context(), heroSpec(BillboardIdle)); err != nil {
		t.Fatal(err)
	}
	assets.files = nil
	if _, ok, err := generator.Lookup(t.Context(), heroSpec(BillboardIdle)); ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	result, err := generator.Generate(t.Context(), heroSpec(BillboardIdle))
	if err != nil || result.Cached || video.submitCount() != 2 {
		t.Fatalf("result=%+v err=%v submits=%d", result, err, video.submitCount())
	}
}

func TestBillboardGenerator_failures(t *testing.T) {
	capped, _ := budget.NewLedger(nil, 0.1)
	cases := []struct {
		name   string
		video  *scriptedReferenceVideo
		ledger *budget.Ledger
		spec   BillboardSpec
		want   error
	}{
		{"budget cap", &scriptedReferenceVideo{}, capped, heroSpec(BillboardIdle), budget.ErrCapReached},
		{"bad action", &scriptedReferenceVideo{}, nil, heroSpec("dance"), nil},
		{"no level", &scriptedReferenceVideo{}, nil, BillboardSpec{Action: BillboardIdle, Subject: BillboardSubject{References: [][]byte{[]byte("x")}}}, nil},
		{"empty ref", &scriptedReferenceVideo{}, nil, BillboardSpec{Action: BillboardIdle, Subject: BillboardSubject{References: [][]byte{nil}}, LevelStill: []byte("l")}, nil},
		{"submit error", &scriptedReferenceVideo{submitErr: errors.New("boom")}, nil, heroSpec(BillboardIdle), nil},
		{"poll error", &scriptedReferenceVideo{pollErr: errors.New("boom")}, nil, heroSpec(BillboardIdle), nil},
		{"empty download", &scriptedReferenceVideo{download: []byte{}}, nil, heroSpec(BillboardIdle), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ledger := tc.ledger
			generator := newTestGenerator(tc.video, &memoryAssets{}, &fakes.FakeCache{}, ledger)
			_, err := generator.Generate(t.Context(), tc.spec)
			if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
			if ledger != nil && ledger.Costs().Reserved != 0 {
				t.Fatalf("reservation leaked: %+v", ledger.Costs())
			}
		})
	}
}

func TestBillboardGenerator_deadlineAndDisabled(t *testing.T) {
	fake := clock.NewFake(time.Unix(0, 0))
	video := &scriptedReferenceVideo{pendingN: -1}
	video.onPoll = func() { fake.Advance(time.Minute) }
	assets := &memoryAssets{}
	generator := NewBillboardGenerator(BillboardGeneratorConfig{Video: video, Assets: assets, Cache: &fakes.FakeCache{}, Clock: fake, PollEvery: -1, Deadline: 3 * time.Minute})
	if _, err := generator.Generate(t.Context(), heroSpec(BillboardHit)); !errors.Is(err, ErrBillboardDeadline) {
		t.Fatalf("err=%v", err)
	}
	cacheOnly := NewBillboardGenerator(BillboardGeneratorConfig{Cache: &fakes.FakeCache{}})
	if cacheOnly.CanGenerate() {
		t.Fatal("cache-only generator claims it can generate")
	}
	if _, err := cacheOnly.Generate(t.Context(), heroSpec(BillboardIdle)); !errors.Is(err, ErrBillboardDisabled) {
		t.Fatalf("err=%v", err)
	}
}

func TestBillboardGenerator_waitUsesClockAndContext(t *testing.T) {
	fake := clock.NewFake(time.Unix(0, 0))
	generator := NewBillboardGenerator(BillboardGeneratorConfig{Clock: fake, PollEvery: time.Second})
	done := make(chan error, 1)
	go func() { done <- generator.wait(t.Context()) }()
	for {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if err := generator.wait(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("err=%v", err)
			}
			return
		default:
			fake.Advance(time.Second)
		}
	}
}

func TestBillboardCacheKey_changesWithEveryInput(t *testing.T) {
	base := heroSpec(BillboardIdle)
	key := BillboardCacheKey("m", base)
	variants := map[string]string{
		"model":  BillboardCacheKey("other", base),
		"action": BillboardCacheKey("m", heroSpec(BillboardAttack)),
		"ref":    BillboardCacheKey("m", BillboardSpec{Action: BillboardIdle, Subject: BillboardSubject{References: [][]byte{[]byte("front2"), []byte("three")}}, LevelStill: []byte("level")}),
		"level":  BillboardCacheKey("m", BillboardSpec{Action: BillboardIdle, Subject: base.Subject, LevelStill: []byte("level2")}),
	}
	for name, variant := range variants {
		if variant == key {
			t.Errorf("%s did not change the key", name)
		}
	}
	lookOnly := base
	lookOnly.Subject.Look = "other look"
	if BillboardCacheKey("m", lookOnly) != key || len(key) != 64 {
		t.Fatal("look text must not change the key")
	}
}

func TestBillboardPrompt_slots(t *testing.T) {
	cases := []struct {
		action, weapon string
		want           []string
	}{
		{BillboardIdle, "", []string{"@Image1 and @Image2", "breathing slowly", "@Image3 is a lighting reference only", "#00B140", "no shadows on the background", "One continuous shot, no cuts"}},
		{BillboardAttack, "Longsword", []string{"swings a longsword once toward it"}},
		{BillboardAttack, "Longbow", []string{"looses one shot from a longbow"}},
		{BillboardAttack, "", []string{"swings a weapon"}},
		{BillboardHit, "", []string{"recoils from a blow"}},
		{BillboardFall, "", []string{"collapses"}},
	}
	for _, tc := range cases {
		t.Run(tc.action+tc.weapon, func(t *testing.T) {
			spec := heroSpec(tc.action)
			spec.Subject.Weapon = tc.weapon
			prompt := BillboardPrompt(spec)
			for _, want := range tc.want {
				if !strings.Contains(prompt, want) {
					t.Errorf("prompt missing %q:\n%s", want, prompt)
				}
			}
			for _, banned := range []string{"realistic", "photo", "4K", "handheld", "zoom", "fast"} {
				if strings.Contains(strings.ToLower(prompt), strings.ToLower(banned)) {
					t.Errorf("prompt has banned word %q", banned)
				}
			}
		})
	}
	single := BillboardPrompt(BillboardSpec{Action: BillboardIdle, Subject: BillboardSubject{References: [][]byte{[]byte("x")}}, LevelStill: []byte("l")})
	if !strings.Contains(single, "The character from @Image1,") || !strings.Contains(single, "like @Image2") {
		t.Fatalf("single-ref prompt: %s", single)
	}
}

func TestBillboardWeapon_classes(t *testing.T) {
	cases := map[string]string{"Paladin": "longsword", "ranger": "longbow", "Warlock": "light crossbow", "wizard": "dagger", "barbarian": "greataxe", "cleric": "mace", "druid": "scimitar", "monk": "quarterstaff", "rogue": "shortsword", "unknown": ""}
	for class, want := range cases {
		if got := BillboardWeapon(class); got != want {
			t.Errorf("%s: got %q want %q", class, got, want)
		}
	}
}

func TestCompactReference_downscalesAndFlattens(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2048, 1024))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	out := compactReference(encoded.Bytes())
	decoded, format, err := image.Decode(bytes.NewReader(out))
	if err != nil || format != "jpeg" || decoded.Bounds().Dx() != 1024 || decoded.Bounds().Dy() != 512 {
		t.Fatalf("format=%s bounds=%v err=%v", format, decoded.Bounds(), err)
	}
	if r, g, b, _ := decoded.At(10, 10).RGBA(); r>>8 < 190 || g>>8 < 190 || b>>8 < 190 {
		t.Fatalf("transparent pixels not flattened to grey: %d %d %d", r>>8, g>>8, b>>8)
	}
	if !bytes.Equal(compactReference([]byte("not an image")), []byte("not an image")) {
		t.Fatal("undecodable input must pass through")
	}
	if !bytes.Equal(compactReference(encoded.Bytes()), out) {
		t.Fatal("compaction must be deterministic")
	}
}
