// Package host provides the minimal DungeonFlux host control surface.
//
// The package owns only the /host WASM entry point and its view-model helpers.
// Browser transport and DOM integration are isolated behind js/wasm build tags;
// command construction remains native-testable.
package host
