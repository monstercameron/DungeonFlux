package phone

import (
	"context"
	"errors"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestTypedInputModel_SubmitsSayAndClosesOnAcceptedResponse(t *testing.T) {
	fake := &sayFake{result: SayResult{Value: &df.SayResponse{Accepted: true, UtteranceId: "utt-7"}}}
	model := NewTypedInputModel(fake, " seat-1 ")
	model.OpenFallback()
	if err := model.SetText("  Tell Mother Vell the truth  "); err != nil {
		t.Fatal(err)
	}
	got := model.ApplySay(<-model.Submit(context.Background()))
	if fake.request.GetSeatToken() != "seat-1" || fake.request.GetText() != "Tell Mother Vell the truth" {
		t.Fatalf("request = %+v", fake.request)
	}
	if got.Open || got.Sending || got.UtteranceID != "utt-7" || got.StatusText != "Message sent" {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestTypedInputModel_ValidatesAndReportsFailures(t *testing.T) {
	model := NewTypedInputModel(&sayFake{}, "seat")
	if err := model.SetText(strings.Repeat("x", 281)); err == nil {
		t.Fatal("long input accepted")
	}
	if err := model.SetText(" "); err != nil {
		t.Fatal(err)
	}
	if got := model.ApplySay(<-model.Submit(context.Background())); got.Error != "message is required" {
		t.Fatalf("empty error = %q", got.Error)
	}
	model.OpenFallback()
	model.SetText("hello")
	got := model.ApplySay(SayResult{Err: errors.New("offline")})
	if !got.Open || got.Error != "offline" || got.Sending {
		t.Fatalf("failure snapshot = %+v", got)
	}
	got = model.ApplySay(SayResult{Value: &df.SayResponse{Reason: "paused"}})
	if got.Error != "paused" {
		t.Fatalf("rejection snapshot = %+v", got)
	}
}

func TestTypedInputModel_NilAndFallbackStates(t *testing.T) {
	var model *TypedInputModel
	if got := model.OpenFallback(); got.Error == "" {
		t.Fatal("nil model did not report an error")
	}
	if got := NewTypedInputModel(nil, "").Snapshot(); got.Open || got.SeatToken != "" {
		t.Fatalf("initial snapshot = %+v", got)
	}
	if got := NewTypedInputModel(nil, "").Submit(context.Background()); (<-got).Err == nil {
		t.Fatal("missing client did not fail")
	}
}

type sayFake struct {
	request *df.SayRequest
	result  SayResult
}

func (f *sayFake) Say(_ context.Context, request *df.SayRequest) <-chan SayResult {
	f.request = request
	out := make(chan SayResult, 1)
	out <- f.result
	return out
}
