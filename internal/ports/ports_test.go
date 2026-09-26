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
