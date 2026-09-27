package debug

import (
	"context"
	"errors"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
)

type controlEngine struct {
	*fakes.FakeEngine
	snapshotData []byte
	vendorResult string
	clientPath   string
	err          error
}

func (e *controlEngine) DebugSnapshot(_ context.Context, _ string, data []byte) ([]byte, error) {
	if e.err != nil {
		return nil, e.err
	}
	e.snapshotData = append([]byte(nil), data...)
	return append([]byte(nil), data...), nil
}
func (e *controlEngine) DebugVendor(context.Context, string, string) (string, error) {
	return e.vendorResult, e.err
}
func (e *controlEngine) DebugClient(context.Context, string, string, string) (string, error) {
	return e.clientPath, e.err
}

func TestControlRPCs_DelegateAndReturnResults(t *testing.T) {
	engine := &controlEngine{FakeEngine: &fakes.FakeEngine{}, vendorResult: `{"mode":"fail"}`, clientPath: "shot.png"}
	server, err := NewServer(engine, &fakes.FakeInbox{PostResult: true})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := server.Snapshot(context.Background(), &df.SnapshotRequest{Operation: "save", Data: []byte("snap")})
	if err != nil || !snapshot.GetOk() || string(snapshot.GetData()) != "snap" {
		t.Fatalf("snapshot = %+v, err=%v", snapshot, err)
	}
	vendor, err := server.Vendor(context.Background(), &df.VendorRequest{Operation: "set", PayloadJson: "{}"})
	if err != nil || !vendor.GetOk() || vendor.GetResultJson() == "" {
		t.Fatalf("vendor = %+v, err=%v", vendor, err)
	}
	client, err := server.Client(context.Background(), &df.ClientRequest{ClientId: "DM", Verb: "screenshot", Arg: "out.png"})
	if err != nil || !client.GetOk() || client.GetPath() != "shot.png" {
		t.Fatalf("client = %+v, err=%v", client, err)
	}
	if string(engine.snapshotData) != "snap" {
		t.Fatalf("snapshot data = %q", engine.snapshotData)
	}
}

func TestControlRPCs_ValidateAndReportControllerErrors(t *testing.T) {
	server, _ := NewServer(&controlEngine{FakeEngine: &fakes.FakeEngine{}, err: errors.New("broken")}, &fakes.FakeInbox{PostResult: true})
	if _, err := server.Snapshot(context.Background(), nil); err == nil {
		t.Fatal("nil snapshot accepted")
	}
	if _, err := server.Vendor(context.Background(), &df.VendorRequest{}); err == nil {
		t.Fatal("empty vendor operation accepted")
	}
	if _, err := server.Client(context.Background(), &df.ClientRequest{ClientId: "DM"}); err == nil {
		t.Fatal("empty client verb accepted")
	}
	if response, err := server.Snapshot(context.Background(), &df.SnapshotRequest{Operation: "load"}); err != nil || response.GetOk() || response.GetReason() != "broken" {
		t.Fatalf("snapshot error = %+v, err=%v", response, err)
	}
	if response, err := server.Vendor(context.Background(), &df.VendorRequest{Operation: "set"}); err != nil || response.GetOk() || response.GetReason() != "broken" {
		t.Fatalf("vendor error = %+v, err=%v", response, err)
	}
	if response, err := server.Client(context.Background(), &df.ClientRequest{ClientId: "DM", Verb: "reload"}); err != nil || response.GetOk() || response.GetReason() != "broken" {
		t.Fatalf("client error = %+v, err=%v", response, err)
	}
}

func TestControlRPCs_UnimplementedWithoutOptionalControllers(t *testing.T) {
	server, _ := NewServer(&fakes.FakeEngine{}, &fakes.FakeInbox{PostResult: true})
	if _, err := server.Snapshot(context.Background(), &df.SnapshotRequest{Operation: "save"}); err == nil {
		t.Fatal("snapshot unexpectedly implemented")
	}
	if _, err := server.Vendor(context.Background(), &df.VendorRequest{Operation: "set"}); err == nil {
		t.Fatal("vendor unexpectedly implemented")
	}
	if _, err := server.Client(context.Background(), &df.ClientRequest{ClientId: "DM", Verb: "reload"}); err == nil {
		t.Fatal("client unexpectedly implemented")
	}
}

func TestDecodeEvent_DebugControlPayloads(t *testing.T) {
	for _, tc := range []struct {
		kind string
		want any
	}{
		{"debug_patch", domain.DebugPatch{}},
		{"debug_timer", domain.DebugTimer{}},
		{"debug_force_dice", domain.DebugForceDice{}},
		{"host_timers_on", domain.HostCmd{}},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			got, err := decodeEvent(tc.kind, "{}")
			if err != nil {
				t.Fatal(err)
			}
			switch tc.want.(type) {
			case domain.DebugPatch:
				if _, ok := got.(domain.DebugPatch); !ok {
					t.Fatalf("got %T", got)
				}
			case domain.DebugTimer:
				if _, ok := got.(domain.DebugTimer); !ok {
					t.Fatalf("got %T", got)
				}
			case domain.DebugForceDice:
				if _, ok := got.(domain.DebugForceDice); !ok {
					t.Fatalf("got %T", got)
				}
			case domain.HostCmd:
				if _, ok := got.(domain.HostCmd); !ok {
					t.Fatalf("got %T", got)
				}
			}
		})
	}
}
