package media

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/content/prompts"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestVoicePackExecutor_GeneratesCachesAndRecordsBudget(t *testing.T) {
	writer := scriptedVoiceAssets()
	cache := &fakes.FakeCache{}
	generator := &fakes.FakeSoundGen{Script: soundResults()}
	ledger, err := budget.NewLedger(map[vocab.VendorName]float64{vocab.VendorElevenLabs: 1})
	if err != nil {
		t.Fatal(err)
	}
	var processed int
	executor := NewVoicePackExecutor(VoicePackConfig{
		Sounds: generator, Assets: writer, Cache: cache, Budget: ledger,
		Normalize: func(data []byte, _ string, durationMS int) ([]byte, error) {
			processed++
			if durationMS < 400 || durationMS > 1500 {
				t.Fatalf("duration=%d", durationMS)
			}
			return append(data, '!'), nil
		},
	})
	request := VoicePackRequest{Seat: 1, Species: "elf", Gender: "female", Class: "wizard", Flavor: "river debt"}
	pack, err := executor.Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(pack) != 6 || len(generator.Calls) != 6 || len(writer.Calls) != 6 || processed != 6 {
		t.Fatalf("pack=%d generated=%d writes=%d processed=%d", len(pack), len(generator.Calls), len(writer.Calls), processed)
	}
	if !strings.Contains(generator.Calls[2].Prompt, "wizard") || generator.Calls[2].Seconds != 1.2 {
		t.Fatalf("spell request=%#v", generator.Calls[2])
	}
	for _, cue := range prompts.VoicePackCues() {
		if got := pack[cue].ID; got == "" {
			t.Fatalf("missing asset for %q", cue)
		}
	}
	spent := ledger.Costs().ByVendor[vocab.VendorElevenLabs].Spent
	if spent < 0.0115 || spent > 0.0117 {
		t.Fatalf("spent=%v", spent)
	}

	second, err := executor.Generate(context.Background(), request)
	if err != nil || len(second) != 6 || len(generator.Calls) != 6 {
		t.Fatalf("cached pack=%d err=%v calls=%d", len(second), err, len(generator.Calls))
	}
	if processed != 12 {
		t.Fatalf("cached assets should still be processed for storage, processed=%d", processed)
	}
}

func TestVoicePackExecutor_ReferenceEffectPublishesNamedSlots(t *testing.T) {
	writer := scriptedVoiceAssets()
	in := &fakes.FakeInbox{PostResult: true}
	executor := NewVoicePackExecutor(VoicePackConfig{Fake: true, Assets: writer})
	effect := domain.GenerateCharacterReference{Seat: 2, Species: "dwarf", Gender: "male", Class: "fighter", Flavor: "bell tower", Scope: domain.Scope{Key: "creation"}}
	executor.Execute(context.Background(), effect, domain.Scope{}, in)
	if len(in.Calls) != 6 || len(writer.Calls) != 6 {
		t.Fatalf("events=%d writes=%d", len(in.Calls), len(writer.Calls))
	}
	for index, cue := range prompts.VoicePackCues() {
		event, ok := in.Calls[index].Envelope.Event.(domain.AssetReady)
		if !ok || event.Slot != VoicePackSlot(2, cue) {
			t.Fatalf("event[%d]=%#v", index, in.Calls[index].Envelope.Event)
		}
		if len(writer.Calls[index].Data) <= 44 || writer.Calls[index].MIME != "audio/wav" {
			t.Fatalf("tone[%d] mime=%q bytes=%d", index, writer.Calls[index].MIME, len(writer.Calls[index].Data))
		}
	}
}

func TestVoicePackExecutor_UsesClassFallbackWhenVendorFails(t *testing.T) {
	fallbacks := map[string][]byte{
		"wizard/spell_cast": []byte("class-spell"),
	}
	for _, cue := range prompts.VoicePackCues() {
		fallbacks[string(cue)] = []byte("generic-" + string(cue))
	}
	writer := scriptedVoiceAssets()
	generator := &fakes.FakeSoundGen{Script: []fakes.SoundResult{{Err: errors.New("vendor down")}}}
	executor := NewVoicePackExecutor(VoicePackConfig{Sounds: generator, Assets: writer, Fallbacks: fallbacks})
	pack, err := executor.Generate(context.Background(), VoicePackRequest{Class: "wizard"})
	if err != nil {
		t.Fatal(err)
	}
	if string(writer.Calls[0].Data) != "generic-attack_effort" {
		t.Fatalf("unexpected first fallback=%q", writer.Calls[0].Data)
	}
	if pack[prompts.VoiceCueSpellCast].ID == "" || string(writer.Calls[2].Data) != "class-spell" {
		t.Fatal("spell fallback asset missing")
	}
}

func TestVoicePackExecutor_FakeModeUsesShortTonesWithoutGenerator(t *testing.T) {
	writer := scriptedVoiceAssets()
	executor := NewVoicePackExecutor(VoicePackConfig{Fake: true, Assets: writer})
	pack, err := executor.Generate(context.Background(), VoicePackRequest{Seat: 1})
	if err != nil || len(pack) != 6 || len(writer.Calls) != 6 {
		t.Fatalf("pack=%d writes=%d err=%v", len(pack), len(writer.Calls), err)
	}
	if writer.Calls[0].Kind != vocab.AssetSFX || writer.Calls[0].Meta.DurationMS != 600 {
		t.Fatalf("write=%+v", writer.Calls[0])
	}
}

func TestVoicePackExecutor_BudgetCapAndMissingFallbackFailClearly(t *testing.T) {
	ledger, err := budget.NewLedger(map[vocab.VendorName]float64{vocab.VendorElevenLabs: 0.0001})
	if err != nil {
		t.Fatal(err)
	}
	executor := NewVoicePackExecutor(VoicePackConfig{Sounds: &fakes.FakeSoundGen{}, Assets: scriptedVoiceAssets(), Budget: ledger})
	_, err = executor.Generate(context.Background(), VoicePackRequest{Class: "wizard"})
	if err == nil || !strings.Contains(err.Error(), "no fallback") {
		t.Fatalf("err=%v", err)
	}

	in := &fakes.FakeInbox{PostResult: true}
	NewVoicePackExecutor(VoicePackConfig{}).Execute(context.Background(), domain.GenerateCharacterReference{Seat: 1}, domain.Scope{}, in)
	if len(in.Calls) != 6 {
		t.Fatalf("failure events=%d", len(in.Calls))
	}
	for _, call := range in.Calls {
		if _, ok := call.Envelope.Event.(domain.AssetFailed); !ok {
			t.Fatalf("event=%T", call.Envelope.Event)
		}
	}
}

func TestVoicePackExecutor_UsesHTTPFixtureThroughSoundPort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), "no words") {
			t.Fatalf("request body=%q err=%v", body, err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fixture-audio"))
	}))
	defer server.Close()
	generator := &httpFixtureSoundGen{URL: server.URL, Client: server.Client()}
	writer := scriptedVoiceAssets()
	executor := NewVoicePackExecutor(VoicePackConfig{Sounds: generator, Assets: writer})
	if _, err := executor.Generate(context.Background(), VoicePackRequest{Class: "wizard"}); err != nil {
		t.Fatal(err)
	}
	if len(generator.Calls) != 6 {
		t.Fatalf("calls=%d", len(generator.Calls))
	}
}

func scriptedVoiceAssets() *fakes.FakeAssetWriter {
	results := make([]fakes.AssetWriteResult, 6)
	for index := range results {
		results[index].Asset = domain.Asset{ID: domain.AssetID("voice-" + string(rune('a'+index)))}
	}
	return &fakes.FakeAssetWriter{Script: results}
}

func soundResults() []fakes.SoundResult {
	results := make([]fakes.SoundResult, 6)
	for index := range results {
		results[index].Sound = ports.Sound{Bytes: []byte{byte(index + 1)}, MIME: "audio/mpeg"}
	}
	return results
}

type httpFixtureSoundGen struct {
	URL    string
	Client *http.Client
	Calls  []ports.SoundRequest
}

func (g *httpFixtureSoundGen) Generate(ctx context.Context, request ports.SoundRequest) (ports.Sound, error) {
	g.Calls = append(g.Calls, request)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, g.URL, strings.NewReader(request.Prompt))
	if err != nil {
		return ports.Sound{}, err
	}
	response, err := g.Client.Do(httpRequest)
	if err != nil {
		return ports.Sound{}, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return ports.Sound{}, err
	}
	return ports.Sound{Bytes: data, MIME: "audio/mpeg"}, nil
}
