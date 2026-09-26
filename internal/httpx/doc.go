// Package httpx provides shared instrumented HTTP clients for vendor
// adapters. It uses keep-alive transports and records httptrace timings.
//
// The package imports only the standard library. CallStats is intentionally
// local until the orchestrator adds the shared ports.CallStats contract.
package httpx
