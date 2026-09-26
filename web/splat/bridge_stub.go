//go:build !js || !wasm

package splat

import "errors"

// ErrUnavailable reports that the browser bridge is not available in a native build.
var ErrUnavailable = errors.New("splat bridge requires js/wasm")

// Bridge is the disabled native implementation of the browser bridge.
type Bridge struct{}

// New returns a bridge that reports ErrUnavailable when used natively.
func New(_ int) *Bridge { return &Bridge{} }

// Init reports that no browser is available.
func (b *Bridge) Init(_ Init) error { return ErrUnavailable }

// Scene reports that no browser is available.
func (b *Bridge) Scene(_ Scene) error { return ErrUnavailable }

// Pause reports that no browser is available.
func (b *Bridge) Pause(_ Pause) error { return ErrUnavailable }

// Dispose reports that no browser is available.
func (b *Bridge) Dispose() error { return ErrUnavailable }

// Events returns an empty event stream for native callers.
func (b *Bridge) Events() <-chan Event { return nil }

// Close releases native bridge resources.
func (b *Bridge) Close() {}
