// Package splat is the typed Go boundary for the PlayCanvas battlefield.
//
// Protocol values are plain data so they can be tested without a browser. The
// js/wasm build provides the syscall/js transport; native builds provide a
// disabled transport for server-side compilation and unit tests.
package splat
