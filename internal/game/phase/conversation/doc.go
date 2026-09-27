// Package conversation owns the pure conversation-phase dispatch rules.
// It translates completed speech and interpretation callbacks into domain
// events and runtime effects; it performs no I/O and owns no goroutines.
package conversation
