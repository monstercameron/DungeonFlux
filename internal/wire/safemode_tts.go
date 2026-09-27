package wire

import (
	"context"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/modelchain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func sequenceTTS(cfg config.Config, next ports.TTS, recordings ports.Recordings) ports.TTS {
	return positionedTTS{next: modelchain.RecordReplayTTS(next, recordings, "sequence"), force: cfg.Features.SequenceMode}
}

type positionedTTS struct {
	next  ports.TTS
	force bool
}

func (p positionedTTS) Stream(ctx context.Context, req ports.TTSRequest, text ports.TextStream) (ports.PCMStream, error) {
	req.Meta = ports.ResolveCallMeta(ctx, req.Meta)
	req.Meta.ForceReplay = req.Meta.ForceReplay || p.force
	return p.next.Stream(ctx, req, text)
}
