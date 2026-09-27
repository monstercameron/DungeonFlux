//go:build js && wasm

package phone

import (
	"context"
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

// PhoneAudio owns cue policy and the browser's cancellable audio session.
type PhoneAudio struct {
	queue          *AudioQueue
	reducedMotion  bool
	vibrateEnabled bool
	localGate      localCueGate
	service        phoneAudioService
	seatToken      string
	seat           int32
	ctx            context.Context
	cancel         context.CancelFunc
	player         localCuePlayer
	tapInstalled   bool
}

type localCuePlayer interface{ playURL(string, string) error }

type phoneAudioService interface {
	Listen(context.Context, *df.ListenRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[df.AudioMessage], error)
}
