package wire

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/runtime"
)

// errExecutorHookMissing records the runtime contract required before wire
// can install the configured effect registry. Runtime.NewRoom currently
// installs an unexported noop runner and exposes no replacement hook.
var errExecutorHookMissing = errors.New("wire: runtime room has no executor registration hook")

// executorHookAvailable is kept as a narrow seam for the pending runtime
// contract. It intentionally does not use reflection or unsafe access to the
// room's private runner.
func executorHookAvailable() bool {
	var _ = runtime.NewRunner
	return false
}
