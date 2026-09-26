package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestSoundExecutor_ManifestWinsWithoutGeneration(t *testing.T) {
	asset := domain.Asset{ID: "manifest-asset", SHA256: "abc", MIME: "audio/mpeg"}
	gen := &fakes.FakeSoundGen{}
	in := &fakes.FakeInbox{PostResult: true}
	e := NewSoundExecutor(SoundConfig{Sounds: gen, Assets: &fakes.FakeAssetWriter{}, Manifest: SoundManifest{"door": asset}})
	e.Execute(context.Background(), SoundRequest{Slot: "sfx", Logical: "door", Kind: vocab.SoundSFX, Prompt: "door", Seconds: 2}, domain.Scope{}, in)
	if len(gen.Calls) != 0 || len(in.Calls) != 1 {
		t.Fatalf("calls=%d inbox=%d", len(gen.Calls), len(in.Calls))
	}
	ready, ok := in.Calls[0].Envelope.Event.(domain.AssetReady)
	if !ok || ready.Asset.ID != asset.ID {
		t.Fatalf("event=%#v", in.Calls[0].Envelope.Event)
	}
}

func TestSoundExecutor_CacheHitStoresAndPublishes(t *testing.T) {
	asset := domain.Asset{ID: "cached-asset", MIME: "audio/mpeg"}
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: asset}}}
	request := SoundRequest{Slot: "ambience", Logical: "rain", Kind: vocab.SoundSFX, Prompt: "rain", Seconds: 4, Loop: true}
	cache := &fakes.FakeCache{Values: map[string][]byte{soundCacheAdapter + "\x00" + soundHash(request): []byte("cached")}}
	gen := &fakes.FakeSoundGen{}
	in := &fakes.FakeInbox{PostResult: true}
	e := NewSoundExecutor(SoundConfig{Sounds: gen, Assets: writer, Cache: cache})
	e.Execute(context.Background(), request, domain.Scope{}, in)
	if len(gen.Calls) != 0 || len(writer.Calls) != 1 || len(in.Calls) != 1 {
		t.Fatalf("gen=%d writes=%d posts=%d", len(gen.Calls), len(writer.Calls), len(in.Calls))
	}
	if string(writer.Calls[0].Data) != "cached" {
		t.Fatalf("cached data=%q", writer.Calls[0].Data)
	}
}

func TestSoundExecutor_GeneratesNormalizesStoresAndUsesPool(t *testing.T) {
	asset := domain.Asset{ID: "generated", MIME: "audio/mpeg"}
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: asset}}}
	cache := &fakes.FakeCache{}
	gen := &fakes.FakeSoundGen{Script: []fakes.SoundResult{{Sound: ports.Sound{Bytes: []byte("raw"), MIME: "audio/mpeg"}}}}
	pool, err := NewPool(map[vocab.VendorName]int{vocab.VendorElevenLabs: 1})
	if err != nil {
		t.Fatal(err)
	}
	in := &fakes.FakeInbox{PostResult: true}
	e := NewSoundExecutor(SoundConfig{Sounds: gen, Assets: writer, Cache: cache, Pool: pool, Normalize: func(data []byte, mime string) ([]byte, error) {
		if mime != "audio/mpeg" {
			t.Fatalf("mime=%q", mime)
		}
		return append(data, '!'), nil
	}})
	request := SoundRequest{Slot: "roll", Logical: "roll", Kind: vocab.SoundSFX, Prompt: "dice", Seconds: 2}
	e.Execute(context.Background(), request, domain.Scope{}, in)
	if len(gen.Calls) != 1 || string(writer.Calls[0].Data) != "raw!" {
		t.Fatalf("calls=%d data=%q", len(gen.Calls), writer.Calls[0].Data)
	}
	if len(cache.Values) != 1 {
		t.Fatalf("cache=%v", cache.Values)
	}
}

func TestSoundExecutor_TimeoutFallsBackToSilence(t *testing.T) {
	asset := domain.Asset{ID: "silence", MIME: "audio/wav"}
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: asset}}}
	gen := &fakes.FakeSoundGen{Script: []fakes.SoundResult{{Err: context.DeadlineExceeded}}}
	in := &fakes.FakeInbox{PostResult: true}
	e := NewSoundExecutor(SoundConfig{Sounds: gen, Assets: writer, Timeout: time.Second})
	e.Execute(context.Background(), SoundRequest{Slot: "dice", Kind: vocab.SoundSFX, Prompt: "dice", Seconds: 2}, domain.Scope{}, in)
	if len(writer.Calls) != 1 || writer.Calls[0].MIME != "audio/wav" || len(writer.Calls[0].Data) < 44 {
		t.Fatalf("writes=%+v", writer.Calls)
	}
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetReady); !ok {
		t.Fatalf("event=%#v", in.Calls[0].Envelope.Event)
	}
}

func TestSoundExecutor_FailuresAndBudgetCap(t *testing.T) {
	for name, config := range map[string]SoundConfig{
		"missing dependencies": {},
		"budget":               {Sounds: &fakes.FakeSoundGen{}, Assets: &fakes.FakeAssetWriter{}, BudgetCapUSD: .001},
		"generator":            {Sounds: &fakes.FakeSoundGen{Script: []fakes.SoundResult{{Err: errors.New("down")}}}, Assets: &fakes.FakeAssetWriter{}},
		"empty audio":          {Sounds: &fakes.FakeSoundGen{Script: []fakes.SoundResult{{Sound: ports.Sound{}}}}, Assets: &fakes.FakeAssetWriter{}},
	} {
		t.Run(name, func(t *testing.T) {
			in := &fakes.FakeInbox{PostResult: true}
			e := NewSoundExecutor(config)
			e.Execute(context.Background(), SoundRequest{Slot: "x", Kind: vocab.SoundSFX, Prompt: "x", Seconds: 2}, domain.Scope{}, in)
			if len(in.Calls) != 1 {
				t.Fatal("expected terminal event")
			}
			if _, ok := in.Calls[0].Envelope.Event.(domain.AssetFailed); !ok {
				t.Fatalf("event=%#v", in.Calls[0].Envelope.Event)
			}
		})
	}
}

func TestSoundExecutor_RejectsNormalizeAndWriteErrors(t *testing.T) {
	gen := &fakes.FakeSoundGen{Script: []fakes.SoundResult{{Sound: ports.Sound{Bytes: []byte("x")}}}}
	in := &fakes.FakeInbox{PostResult: true}
	e := NewSoundExecutor(SoundConfig{Sounds: gen, Assets: &fakes.FakeAssetWriter{}, Normalize: func([]byte, string) ([]byte, error) { return nil, errors.New("bad audio") }})
	e.Execute(context.Background(), SoundRequest{Slot: "x", Kind: vocab.SoundSFX, Prompt: "x", Seconds: 1}, domain.Scope{}, in)
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetFailed); !ok {
		t.Fatalf("event=%#v", in.Calls[0].Envelope.Event)
	}
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Err: errors.New("disk")}}}
	e = NewSoundExecutor(SoundConfig{Sounds: &fakes.FakeSoundGen{Script: []fakes.SoundResult{{Sound: ports.Sound{Bytes: []byte("x")}}}}, Assets: writer})
	in = &fakes.FakeInbox{PostResult: true}
	e.Execute(context.Background(), SoundRequest{Slot: "x", Kind: vocab.SoundSFX, Prompt: "x", Seconds: 1}, domain.Scope{}, in)
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetFailed); !ok {
		t.Fatalf("event=%#v", in.Calls[0].Envelope.Event)
	}
}
