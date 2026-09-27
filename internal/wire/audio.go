package wire

import (
	"context"
	"io"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/runtime"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// audioStreamService adapts the target-aware listen hub to gRPC. The legacy
// PCM service remains in api_services.go for wire compatibility; this service
// is the registered endpoint for new DM and phone streams.
type audioStreamService struct {
	df.UnimplementedAudioServiceServer
	hub      *api.ListenHub
	sessions *api.SessionServer
}

// Keep the compatibility implementation build-checked while the registered
// service uses the target-aware implementation below.
var _ df.AudioServiceServer = (*audioService)(nil)

type audioHubSink struct{ hub *api.ListenHub }

func (s audioHubSink) PublishChunk(chunk media.AudioChunk) {
	s.hub.Publish(api.AudioMessage{Channel: apiChannel(chunk.Channel), Target: apiTarget(chunk.Target), Chunk: &api.EncodedAudioChunk{CodecMIME: chunk.CodecMIME, Seq: chunk.Seq, Data: chunk.Data, Final: chunk.Final, DurationMS: chunk.DurationMS}})
}

func (s audioHubSink) PublishMix(mix media.MixCommand) {
	s.hub.Publish(api.AudioMessage{Channel: apiChannel(mix.Channel), Target: apiTarget(mix.Target), Mix: &api.MixCommand{Command: mix.Command, TrackID: mix.TrackID, StartAtMS: mix.StartAtMS, DurationMS: mix.DurationMS, Loop: mix.Loop, Gain: mix.Gain, Duck: mix.Duck}})
}

func newAudioRouter(hub *api.ListenHub) (*media.AudioRouter, error) {
	return media.NewAudioRouter(audioHubSink{hub: hub}, 64*1024)
}

func registerAudioExecutor(runner *runtime.Runner, router *media.AudioRouter, assets *assetCatalog) {
	runtime.Handle(runner, func(ctx context.Context, effect domain.PlaySound, _ domain.Scope, _ ports.Inbox) {
		if router == nil || assets == nil || effect.Name == "" {
			return
		}
		info, reader, err := assets.Open(ctx, effect.Name)
		if err != nil {
			return
		}
		data, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil {
			return
		}
		_ = router.RouteSound(effect, domain.Asset{ID: domain.AssetID(info.Name), MIME: info.ContentType}, data)
	})
}

func (s *audioStreamService) Listen(req *df.ListenRequest, stream df.AudioService_ListenServer) error {
	if req == nil || stream == nil || s == nil || s.hub == nil || s.sessions == nil {
		return status.Error(codes.InvalidArgument, "listen request is required")
	}
	client, ok := s.sessions.AudioClientForToken(req.GetSeatToken())
	if !ok {
		return status.Error(codes.PermissionDenied, "audio token is invalid")
	}
	target, phone := listenTarget(client)
	sub := s.hub.SubscribeTarget(stream.Context(), target, phone)
	defer sub.Close()
	if err := stream.SendHeader(metadata.MD{}); err != nil {
		return err
	}
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-sub.Done():
			stream.SetTrailer(metadata.Pairs("dungeonflux-listen", "replaced"))
			return io.EOF
		case message, ok := <-sub.Messages():
			if !ok {
				return nil
			}
			if phone && message.Chunk != nil && message.Chunk.DurationMS > 10000 {
				continue
			}
			if err := stream.Send(toProtoAudioMessage(message)); err != nil {
				return err
			}
		}
	}
}

func listenTarget(client api.AudioClient) (api.AudioTarget, bool) {
	if client.Kind == df.ClientKind_CLIENT_KIND_PHONE {
		return api.AudioTarget{Kind: api.TargetSeat, Seat: client.PlayerNumber}, true
	}
	return api.AudioTarget{Kind: api.TargetDM}, false
}

func toProtoAudioMessage(message api.AudioMessage) *df.AudioMessage {
	result := &df.AudioMessage{Channel: protoChannel(message.Channel), Target: protoTarget(message.Target)}
	switch {
	case message.Frame != nil:
		frame := message.Frame
		result.Message = &df.AudioMessage_Frame{Frame: &df.AudioFrame{UtteranceId: string(frame.UtteranceID), Speaker: frame.Speaker, Seq: uint64(frame.Seq), SampleRate: int32(frame.SampleRate), PcmS16Le: frame.PCMS16LE, Final: frame.Final}}
	case message.Chunk != nil:
		chunk := message.Chunk
		result.Message = &df.AudioMessage_Chunk{Chunk: &df.EncodedAudioChunk{CodecMime: chunk.CodecMIME, Seq: chunk.Seq, Data: chunk.Data, Final: chunk.Final, DurationMs: int32(chunk.DurationMS)}}
	case message.Mix != nil:
		mix := message.Mix
		result.Message = &df.AudioMessage_Mix{Mix: &df.AudioMixCommand{Kind: protoMixKind(mix.Command), TrackId: mix.TrackID, StartAtMs: mix.StartAtMS, DurationMs: int32(mix.DurationMS), Loop: mix.Loop, Gain: mix.Gain, Duck: mix.Duck}}
	default:
		result.Message = &df.AudioMessage_Cancel{Cancel: &df.AudioCancel{UtteranceId: message.CancelID, All: message.CancelAll}}
	}
	return result
}

func apiChannel(channel vocab.SoundKind) api.AudioChannel {
	switch channel {
	case vocab.SoundMusic:
		return api.AudioMusic
	case vocab.SoundAmbience:
		return api.AudioAmbience
	case vocab.SoundSFX:
		return api.AudioSFX
	default:
		return api.AudioVoice
	}
}

func apiTarget(target media.AudioTarget) api.AudioTarget {
	return api.AudioTarget{Kind: target.Kind, Seat: int(target.Seat)}
}

func protoChannel(channel api.AudioChannel) df.AudioChannel {
	switch channel {
	case api.AudioVoice:
		return df.AudioChannel_AUDIO_CHANNEL_VOICE
	case api.AudioMusic:
		return df.AudioChannel_AUDIO_CHANNEL_MUSIC
	case api.AudioAmbience:
		return df.AudioChannel_AUDIO_CHANNEL_AMBIENCE
	case api.AudioSFX:
		return df.AudioChannel_AUDIO_CHANNEL_SFX
	default:
		return df.AudioChannel_AUDIO_CHANNEL_UNSPECIFIED
	}
}

func protoTarget(target api.AudioTarget) *df.AudioTarget {
	result := &df.AudioTarget{}
	switch target.Kind {
	case api.TargetDM:
		result.Kind = df.AudioTargetKind_AUDIO_TARGET_KIND_DM
	case api.TargetSeat:
		result.Kind = df.AudioTargetKind_AUDIO_TARGET_KIND_SEAT
		result.Seat = int32(target.Seat)
	case api.TargetAllPhones:
		result.Kind = df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES
	}
	return result
}

func protoMixKind(command string) df.AudioMixCommandKind {
	switch command {
	case "play":
		return df.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_PLAY
	case "stop":
		return df.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_STOP
	case "crossfade":
		return df.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_CROSSFADE
	case "duck":
		return df.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_DUCK
	default:
		return df.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_UNSPECIFIED
	}
}
