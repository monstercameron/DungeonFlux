package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type faultLLMFake struct{}

func (faultLLMFake) JSON(context.Context, ports.TextRequest, ports.Schema) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
}
func (faultLLMFake) StreamText(context.Context, ports.TextRequest) (ports.TextStream, error) {
	return faultStream{}, nil
}

type faultStream struct{}

func (faultStream) Recv() (string, error) { return "token", nil }
func (faultStream) Close() error          { return nil }

func TestFaults_SetGetAndValidation(t *testing.T) {
	faults := NewFaults()
	if got := faults.Get("llm"); got.Mode != FaultOK {
		t.Fatalf("default fault = %#v", got)
	}
	for _, tc := range []struct {
		name  string
		mode  FaultMode
		delay time.Duration
	}{
		{"", FaultFail, 0}, {"llm", "unknown", 0}, {"llm", FaultSlow, -time.Millisecond},
	} {
		if err := faults.Set(tc.name, tc.mode, tc.delay); err == nil {
			t.Fatalf("accepted invalid fault %#v", tc)
		}
	}
	if err := faults.Set("llm", FaultFail, 0); err != nil || faults.Get("llm").Mode != FaultFail {
		t.Fatalf("set fail = %#v, err=%v", faults.Get("llm"), err)
	}
	if err := faults.Set("llm", FaultOK, 0); err != nil || faults.Get("llm").Mode != FaultOK {
		t.Fatalf("clear fault = %#v, err=%v", faults.Get("llm"), err)
	}
}

func TestWithFault_DelegatesAndFails(t *testing.T) {
	faults := NewFaults()
	decorated := WithFault(faultLLMFake{}, "llm", faults)
	value, err := decorated.JSON(context.Background(), ports.TextRequest{}, ports.Schema{})
	if err != nil || string(value) != `{"ok":true}` {
		t.Fatalf("normal JSON = %s, err=%v", value, err)
	}
	stream, err := decorated.StreamText(context.Background(), ports.TextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if token, err := stream.Recv(); err != nil || token != "token" {
		t.Fatalf("stream = %q, err=%v", token, err)
	}
	if err := faults.Set("llm", FaultFail, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := decorated.JSON(context.Background(), ports.TextRequest{}, ports.Schema{}); err == nil {
		t.Fatal("failure fault did not fail JSON")
	}
	if _, err := decorated.StreamText(context.Background(), ports.TextRequest{}); err == nil {
		t.Fatal("failure fault did not fail stream")
	}
}

func TestWithFault_SlowHonorsCancellationAndNilNext(t *testing.T) {
	faults := NewFaults()
	if err := faults.Set("llm", FaultSlow, time.Hour); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	decorated := WithFault(faultLLMFake{}, "llm", faults)
	if _, err := decorated.JSON(ctx, ports.TextRequest{}, ports.Schema{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("slow cancellation err = %v", err)
	}
	if _, err := WithFault(nil, "llm", NewFaults()).JSON(context.Background(), ports.TextRequest{}, ports.Schema{}); err == nil {
		t.Fatal("nil next accepted")
	}
}
