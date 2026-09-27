package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const shutdownDeadline = 5 * time.Second

// ShutdownHooks describes the owned resources that must be closed in order.
// Each hook must honor ctx and return after its resource is quiescent.
type ShutdownHooks struct {
	CancelScopes func()
	StopStreams  func(context.Context) error
	DrainHubs    func(context.Context) error
	FlushStore   func(context.Context) error
	CloseVendors func(context.Context) error
}

// Shutdown cancels work first, then stops streams, drains hubs, flushes the
// event store, and finally closes vendor clients. The whole sequence has a
// five-second deadline.
func (h ShutdownHooks) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownDeadline)
	defer cancel()
	if h.CancelScopes != nil {
		h.CancelScopes()
	}
	var errs []error
	for _, hook := range []struct {
		name string
		fn   func(context.Context) error
	}{
		{"stop streams", h.StopStreams},
		{"drain hubs", h.DrainHubs},
		{"flush store", h.FlushStore},
		{"close vendors", h.CloseVendors},
	} {
		if hook.fn == nil {
			continue
		}
		if err := shutdownHook(shutdownCtx, hook.name, hook.fn); err != nil {
			errs = append(errs, err)
		}
		if shutdownCtx.Err() != nil {
			errs = append(errs, shutdownCtx.Err())
			break
		}
	}
	return errors.Join(errs...)
}

func shutdownHook(ctx context.Context, name string, hook func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := hook(ctx); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
