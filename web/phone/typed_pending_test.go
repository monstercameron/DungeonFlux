package phone

import (
	"context"
	"errors"
	"sync"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestTypedInput_PendingSendRejectsDuplicatesWithoutReleasingOwner(t *testing.T) {
	client := &pendingSay{result: make(chan SayResult, 1)}
	model := NewTypedInputModel(client, "seat")
	model.SetText("Where is the lamplighter?")
	first := model.Submit(context.Background())
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			duplicate := <-model.Submit(context.Background())
			if duplicate.Err == nil {
				t.Error("duplicate send was accepted")
			}
			model.ApplySay(duplicate)
			if got := model.Snapshot(); !got.Sending || got.CanSubmit {
				t.Error("duplicate released pending request")
			}
		})
	}
	group.Wait()
	if client.calls != 1 {
		t.Fatalf("sent %d requests", client.calls)
	}
	if err := model.SetText("replacement"); err == nil {
		t.Fatal("pending draft was overwritten")
	}
	if got := model.Snapshot(); got.StatusText != "Sending…" {
		t.Fatalf("pending status = %+v", got)
	}
	client.result <- SayResult{Value: &df.SayResponse{Accepted: true, UtteranceId: "reply"}}
	if got := model.ApplySay(<-first); got.Sending || got.Text != "" || got.UtteranceID != "reply" {
		t.Fatalf("completion = %+v", got)
	}
}

func TestTypedInput_FailureRetainsDraftForRetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result SayResult
	}{
		{"network", SayResult{Err: errors.New("connection interrupted")}},
		{"server", SayResult{Value: &df.SayResponse{Reason: "Mother Vell is replying"}}},
		{"empty response", SayResult{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &pendingSay{result: make(chan SayResult, 1)}
			model := NewTypedInputModel(client, "seat")
			model.SetText("Where is he?")
			pending := model.Submit(context.Background())
			client.result <- tc.result
			got := model.ApplySay(<-pending)
			if got.Text != "Where is he?" || got.Error == "" || !got.CanSubmit {
				t.Fatalf("unretryable failure: %+v", got)
			}
			model.Submit(context.Background())
			if client.calls != 2 || model.Snapshot().Error != "" {
				t.Fatal("retry did not start cleanly")
			}
		})
	}
}

func TestTypedInput_AuthoritativeAvailabilityKeepsDraft(t *testing.T) {
	for _, tc := range []struct {
		name, phase, spotlight string
		paused                 bool
		key                    string
	}{
		{"waiting", "Conversation", "2", false, "typed.waiting"},
		{"paused", "Conversation", "1", true, "typed.paused"},
		{"wrong phase", "Combat", "1", false, "typed.unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &pendingSay{result: make(chan SayResult, 1)}
			model := NewTypedInputModel(client, "seat")
			state := &df.ScreenState{Phase: tc.phase, SpotlightSeat: tc.spotlight, Paused: tc.paused, View: &df.ScreenState_Phone{Phone: &df.PhoneView{PlayerNumber: 1, Locale: "es"}}}
			model.ApplyScreenState(state)
			model.SetText("¿Dónde está?")
			got := model.Snapshot()
			if got.CanSubmit || got.BlockedReason != T("es", tc.key, nil) {
				t.Fatalf("availability: %+v", got)
			}
			if result := <-model.Submit(context.Background()); result.Err == nil || client.calls != 0 {
				t.Fatal("blocked message reached server")
			}
			state.Phase, state.SpotlightSeat, state.Paused = "Conversation", "1", false
			model.ApplyScreenState(state)
			if got := model.Snapshot(); !got.CanSubmit || got.Text != "¿Dónde está?" {
				t.Fatalf("draft lost after resume: %+v", got)
			}
		})
	}
}

type pendingSay struct {
	calls  int
	result chan SayResult
}

func (p *pendingSay) Say(context.Context, *df.SayRequest) <-chan SayResult {
	p.calls++
	return p.result
}
