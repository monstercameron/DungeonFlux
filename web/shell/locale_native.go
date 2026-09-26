//go:build !js || !wasm

package main

// browserLocales has no browser outside wasm; native callers get English.
func browserLocales() []string { return []string{"en"} }
