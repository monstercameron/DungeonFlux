// Package media coordinates bounded access to external media vendors.
package media

import (
	"context"
	"errors"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// ErrUnknownVendor reports that no semaphore was configured for a vendor.
var ErrUnknownVendor = errors.New("media: unknown vendor")

// ErrInvalidLimit reports a non-positive vendor concurrency limit.
var ErrInvalidLimit = errors.New("media: invalid vendor limit")

// Job is one vendor operation admitted by a Pool.
type Job func(context.Context) error

// PoolConfig contains the per-vendor concurrency caps. A cap is required for
// every vendor used by a job; omitting a vendor is treated as a configuration
// error at execution time rather than silently allowing unbounded work.
type PoolConfig struct {
	Limits map[vocab.VendorName]int
}

// Pool limits concurrent jobs independently for each configured vendor.
type Pool struct {
	semaphores map[vocab.VendorName]chan struct{}
}

// NewPool constructs a pool from vendor concurrency caps. The input map is
// copied, so later configuration changes cannot alter a running pool.
func NewPool(limits map[vocab.VendorName]int) (*Pool, error) {
	return newPool(PoolConfig{Limits: limits})
}

// NewPoolFromConfig constructs a pool from its explicit configuration.
func NewPoolFromConfig(config PoolConfig) (*Pool, error) {
	return newPool(config)
}

func newPool(config PoolConfig) (*Pool, error) {
	if len(config.Limits) == 0 {
		return nil, fmt.Errorf("%w: no vendors configured", ErrInvalidLimit)
	}
	semaphores := make(map[vocab.VendorName]chan struct{}, len(config.Limits))
	for vendor, limit := range config.Limits {
		if vendor == "" || limit <= 0 {
			return nil, fmt.Errorf("%w: vendor %q has limit %d", ErrInvalidLimit, vendor, limit)
		}
		semaphores[vendor] = make(chan struct{}, limit)
	}
	return &Pool{semaphores: semaphores}, nil
}

// Run waits for the vendor's semaphore, runs job, and releases the slot. The
// wait is context-aware so a cancelled scope never leaves a queued job stuck.
func (p *Pool) Run(ctx context.Context, vendor vocab.VendorName, job Job) error {
	if p == nil {
		return ErrUnknownVendor
	}
	semaphore, ok := p.semaphores[vendor]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownVendor, vendor)
	}
	if job == nil {
		return errors.New("media: nil job")
	}
	select {
	case semaphore <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-semaphore }()
	return job(ctx)
}
