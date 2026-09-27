package media

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestNewPool_rejectsInvalidLimits(t *testing.T) {
	tests := []struct {
		name   string
		limits map[vocab.VendorName]int
	}{
		{name: "empty", limits: nil},
		{name: "zero", limits: map[vocab.VendorName]int{vocab.VendorOpenAI: 0}},
		{name: "negative", limits: map[vocab.VendorName]int{vocab.VendorFal: -1}},
		{name: "empty vendor", limits: map[vocab.VendorName]int{"": 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewPool(test.limits); !errors.Is(err, ErrInvalidLimit) {
				t.Fatalf("error = %v, want ErrInvalidLimit", err)
			}
		})
	}
}

func TestPool_Run_capsEachVendorIndependently(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool, err := NewPool(map[vocab.VendorName]int{vocab.VendorOpenAI: 1, vocab.VendorFal: 1})
		if err != nil {
			t.Fatal(err)
		}
		started := make(chan vocab.VendorName, 3)
		release := make(chan struct{})
		job := func(vendor vocab.VendorName) Job {
			return func(context.Context) error {
				started <- vendor
				<-release
				return nil
			}
		}
		go func() { _ = pool.Run(context.Background(), vocab.VendorOpenAI, job(vocab.VendorOpenAI)) }()
		go func() { _ = pool.Run(context.Background(), vocab.VendorOpenAI, job(vocab.VendorOpenAI)) }()
		go func() { _ = pool.Run(context.Background(), vocab.VendorFal, job(vocab.VendorFal)) }()
		synctest.Wait()
		if got := len(started); got != 2 {
			t.Fatalf("started jobs = %d, want one per vendor", got)
		}
		close(release)
		synctest.Wait()
		if got := len(started); got != 3 {
			t.Fatalf("started jobs after release = %d, want 3", got)
		}
	})
}

func TestPool_Run_cancelledWaitDoesNotRunJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool, err := NewPool(map[vocab.VendorName]int{vocab.VendorGemini: 1})
		if err != nil {
			t.Fatal(err)
		}
		block := make(chan struct{})
		started := make(chan struct{}, 1)
		go func() {
			_ = pool.Run(context.Background(), vocab.VendorGemini, func(context.Context) error {
				started <- struct{}{}
				<-block
				return nil
			})
		}()
		synctest.Wait()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		run := pool.Run(ctx, vocab.VendorGemini, func(context.Context) error {
			t.Fatal("cancelled job ran")
			return nil
		})
		if !errors.Is(run, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", run)
		}
		close(block)
	})
}

func TestPool_Run_validatesVendorAndJob(t *testing.T) {
	pool, err := NewPool(map[vocab.VendorName]int{vocab.VendorOpenAI: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(pool.Run(context.Background(), vocab.VendorGemini, func(context.Context) error { return nil }), ErrUnknownVendor) {
		t.Fatal("unknown vendor was accepted")
	}
	if err := pool.Run(context.Background(), vocab.VendorOpenAI, nil); err == nil {
		t.Fatal("nil job was accepted")
	}
}
