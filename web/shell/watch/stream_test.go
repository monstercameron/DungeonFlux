package watch

import (
	"context"
	"errors"
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"testing"
	"testing/synctest"
	"time"
)

type fakeService struct {
	calls int
	fail  bool
	eof   bool
}

func (f *fakeService) Watch(ctx context.Context, req *df.WatchRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[df.WatchMessage], error) {
	f.calls++
	if req.GetSeatToken() != "seat" {
		return nil, errors.New("wrong token")
	}
	if f.fail && f.calls == 1 {
		return nil, errors.New("offline")
	}
	return &fakeStream{ctx: ctx, version: uint64(f.calls), eof: f.eof && f.calls == 1}, nil
}

type fakeStream struct {
	grpc.ServerStreamingClient[df.WatchMessage]
	ctx     context.Context
	version uint64
	sent    bool
	eof     bool
}

func (f *fakeStream) Recv() (*df.WatchMessage, error) {
	if !f.sent {
		f.sent = true
		return &df.WatchMessage{Message: &df.WatchMessage_State{State: &df.ScreenState{Version: f.version}}}, nil
	}
	if f.eof {
		return nil, errors.New("disconnected")
	}
	<-f.ctx.Done()
	return nil, f.ctx.Err()
}
func TestMessages_recoversAndStops(t *testing.T) {
	for _, tc := range []struct {
		name string
		idle bool
		fail bool
		eof  bool
	}{
		{name: "explicit wake"}, {name: "silent socket", idle: true}, {name: "open fails", fail: true}, {name: "stream fails", eof: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := &Controller{}
				service := &fakeService{fail: tc.fail, eof: tc.eof}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				out := c.Messages(ctx, service, "seat")
				first := <-out
				if tc.fail {
					if first.GetState().GetVersion() != 2 {
						t.Fatal("failed subscription did not retry")
					}
				} else {
					if first.GetState().GetVersion() != 1 {
						t.Fatal("first snapshot missing")
					}
					if !tc.idle && !tc.eof {
						c.Resync()
					}
					next := <-out
					if next.GetState().GetVersion() != 2 {
						t.Fatal("did not replay after recovery")
					}
				}
				cancel()
				for range out {
				}
				synctest.Wait()
			})
		})
	}
}
func TestSupervise_heartbeatExtendsWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := &Controller{Idle: time.Second}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		received := make(chan struct{}, 1)
		kicked := c.Supervise(ctx, cancel, received)
		synctest.Wait()
		timer := time.NewTimer(750 * time.Millisecond)
		<-timer.C
		received <- struct{}{}
		synctest.Wait()
		timer.Reset(750 * time.Millisecond)
		<-timer.C
		if kicked.Load() || ctx.Err() != nil {
			t.Fatal("heartbeat did not extend deadline")
		}
		<-ctx.Done()
		if !kicked.Load() {
			t.Fatal("idle stream was not restarted")
		}
	})
}
func TestController_resyncBeforeWatchAndNil(t *testing.T) {
	var nilController *Controller
	nilController.Resync()
	c := &Controller{}
	c.Resync()
	old := c.Signal()
	c.Resync()
	select {
	case <-old:
	default:
		t.Fatal("old stream not notified")
	}
	select {
	case <-c.Signal():
		t.Fatal("new stream was canceled")
	default:
	}
}
func TestMessages_cancelBlockedConsumer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		c := &Controller{Idle: time.Hour}
		service := &fakeService{eof: true}
		out := c.Messages(ctx, service, "seat")
		synctest.Wait()
		cancel()
		synctest.Wait()
		for range out {
		}
	})
}
