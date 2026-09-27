package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func TestRoom_RunLog(t *testing.T) {
	tests := []struct {
		name     string
		startErr error
		wantRuns []domain.RunID
	}{
		{name: "stamps records and replays seats into the new run on reset", wantRuns: []domain.RunID{"run-1", "run-2", "run-2"}},
		{name: "keeps the previous run when starting a run fails", startErr: errors.New("disk full"), wantRuns: []domain.RunID{"run-1", "run-1", "run-1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state, err := NewRoomState([]byte("seed"))
			if err != nil {
				t.Fatal(err)
			}
			state.Join(domain.Seat{ID: 1, PlayerNumber: 1})
			log := &roomLog{}
			published := make(chan domain.View, 8)
			var startSeeds [][]byte
			start := func(_ context.Context, seed []byte) (domain.RunID, error) {
				startSeeds = append(startSeeds, append([]byte(nil), seed...))
				return "run-2", tc.startErr
			}
			old := &roomEngine{effects: []domain.Effect{domain.NewRun{Seed: []byte("ignored")}}}
			room := NewRoom(old, clock.NewFake(time.Unix(0, 0)), log, nil, func(view domain.View) { published <- view },
				WithRoomState(state), WithRunLog("run-1", start),
				WithNewGame(func([]byte) ports.Engine { return &roomEngine{} }))
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- room.Run(ctx) }()
			<-published
			room.Post(ctx, domain.Envelope{Event: domain.Join{Seat: 1}})
			<-published
			room.Post(ctx, domain.Envelope{Event: domain.Join{Seat: 1}})
			<-published
			cancel()
			<-done
			if len(log.records) != len(tc.wantRuns) {
				t.Fatalf("records = %d, want %d", len(log.records), len(tc.wantRuns))
			}
			for i, want := range tc.wantRuns {
				if log.records[i].Run != want {
					t.Fatalf("record %d run = %q, want %q", i, log.records[i].Run, want)
				}
			}
			if len(startSeeds) != 1 || string(startSeeds[0]) != string(deriveSeed([]byte("seed"))) {
				t.Fatalf("start seeds = %x, want the new engine's seed once", startSeeds)
			}
		})
	}
}
