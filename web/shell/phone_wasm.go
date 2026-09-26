//go:build js && wasm

package main

import (
	"context"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/phone"
)

// phoneClientAdapter preserves the joined shell connection and seat identity.
type phoneClientAdapter struct{ client *Client }

func (a phoneClientAdapter) Act(ctx context.Context, request *df.ActRequest) <-chan phone.ActResult {
	results := make(chan phone.ActResult, 1)
	go func() {
		defer close(results)
		select {
		case result := <-a.client.Act(ctx, request):
			results <- phone.ActResult{Value: result.Value, Err: result.Err}
		case <-ctx.Done():
			results <- phone.ActResult{Err: ctx.Err()}
		}
	}()
	return results
}

func (a phoneClientAdapter) Say(ctx context.Context, request *df.SayRequest) <-chan phone.SayResult {
	results := make(chan phone.SayResult, 1)
	go func() {
		defer close(results)
		select {
		case result := <-a.client.Say(ctx, request):
			results <- phone.SayResult{Value: result.Value, Err: result.Err}
		case <-ctx.Done():
			results <- phone.SayResult{Err: ctx.Err()}
		}
	}()
	return results
}

func (a phoneClientAdapter) Watch(ctx context.Context, request *df.WatchRequest) <-chan phone.WatchResult {
	// A slow screen consumes only the newest server snapshot.
	results := make(chan phone.WatchResult, 1)
	go func() {
		defer close(results)
		for result := range a.client.Watch(ctx, request) {
			state := result.Message.GetState()
			if state == nil && result.Err == nil {
				continue
			}
			next := phone.WatchResult{State: state, Err: result.Err}
			select {
			case results <- next:
			case <-ctx.Done():
				return
			default:
				select {
				case <-results:
				default:
				}
				select {
				case results <- next:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return results
}

func (a phoneClientAdapter) OpenTalk(ctx context.Context) (phone.TalkStream, error) {
	stream, err := df.NewVoiceServiceClient(a.client.conn).Talk(ctx)
	if err != nil {
		return nil, err
	}
	// Drain server acknowledgements so its bidi stream cannot stall. The phone
	// recording context owns this goroutine; cancellation or EOF ends it.
	go func() {
		for {
			if _, err := stream.Recv(); err != nil {
				return
			}
		}
	}()
	return stream, nil
}
