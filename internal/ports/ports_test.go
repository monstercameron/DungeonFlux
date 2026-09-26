package ports

import (
	"errors"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"testing"
)

func TestCallErrorWrapsVendorError(t *testing.T) {
	inner := errors.New("timeout")
	e := &CallError{Vendor: vocab.VendorOpenAI, Kind: vocab.ErrTimeout, Err: inner}
	if !errors.Is(e, inner) || e.Error() != "openai: timeout: timeout" {
		t.Fatalf("unexpected call error: %v", e)
	}
}

func TestCallErrorWithoutCauseAndNilReceiver(t *testing.T) {
	e := &CallError{Vendor: vocab.VendorGemini, Kind: vocab.ErrUnavailable}
	if e.Error() != "gemini: unavailable" || e.Unwrap() != nil {
		t.Fatalf("unexpected cause-free error: %v", e)
	}
	var nilError *CallError
	if nilError.Error() != "<nil>" || nilError.Unwrap() != nil {
		t.Fatal("nil CallError should be safe")
	}
}
