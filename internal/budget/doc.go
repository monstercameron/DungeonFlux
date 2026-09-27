// Package budget reserves and settles estimated vendor spend.
//
// The ledger is intentionally independent of adapters. Callers reserve an
// estimate before starting work, then settle the reservation with actual
// usage or release it when the call does not happen.
package budget
