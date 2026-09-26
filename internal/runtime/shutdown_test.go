package runtime

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"testing/synctest"
)

func TestShutdown_ordersCancellationAndResourceDrain(t *testing.T) {
	var order []string
	add := func(name string) func(context.Context) error {
		return func(context.Context) error {
			order = append(order, name)
			return nil
		}
	}
	hooks := ShutdownHooks{
		CancelScopes: func() { order = append(order, "cancel scopes") },
		StopStreams:  add("stop streams"),
		DrainHubs:    add("drain hubs"),
		FlushStore:   add("flush store"),
		CloseVendors: add("close vendors"),
	}
	if err := hooks.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"cancel scopes", "stop streams", "drain hubs", "flush store", "close vendors"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("shutdown order = %v, want %v", order, want)
	}
}

func TestShutdown_stopsAfterContextDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		called := false
		hooks := ShutdownHooks{
			StopStreams: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
			CloseVendors: func(context.Context) error { called = true; return nil },
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := hooks.Shutdown(ctx)
		if !errors.Is(err, context.Canceled) || called {
			t.Fatalf("err = %v, vendors called = %v", err, called)
		}
	})
}

func TestShutdown_returnsHookError(t *testing.T) {
	want := errors.New("store unavailable")
	err := (ShutdownHooks{FlushStore: func(context.Context) error { return want }}).Shutdown(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}
