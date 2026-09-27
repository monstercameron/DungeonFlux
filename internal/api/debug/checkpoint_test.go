package debug

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
)

type checkpointInbox struct{ called bool }

func (in *checkpointInbox) Post(context.Context, domain.Envelope) bool { return true }
func (in *checkpointInbox) DebugSnapshot(context.Context, string, []byte) ([]byte, error) {
	in.called = true
	return []byte("runtime point"), nil
}

func TestSnapshot_UsesRuntimeControllerBeforeEngine(t *testing.T) {
	in := &checkpointInbox{}
	engine := &controlEngine{FakeEngine: &fakes.FakeEngine{}}
	server, _ := NewServer(engine, in)
	response, err := server.Snapshot(context.Background(), &df.SnapshotRequest{Operation: "save", Data: []byte("before")})
	if err != nil || !response.GetOk() || string(response.GetData()) != "runtime point" || !in.called || engine.snapshotData != nil {
		t.Fatalf("runtime controller not used: %+v, %v", response, err)
	}
}
