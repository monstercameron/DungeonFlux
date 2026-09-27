package wire

import (
	"context"
	"encoding/json"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/modelchain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// sequenceLLM applies the rehearsal recording layer to the LLM. Recording is
// enabled for every run; sequence mode changes the layer to replay-only so no
// network-backed provider can be reached during a show.
func sequenceLLM(cfg config.Config, next ports.LLM, recordings ports.Recordings) ports.LLM {
	if next == nil {
		return next
	}
	if recordings == nil {
		return positionedLLM{next: next}
	}
	decorated := modelchain.RecordReplay(next, recordings, "sequence")
	if !cfg.Features.SequenceMode {
		return positionedLLM{next: decorated}
	}
	return positionedLLM{next: replayOnlyLLM{next: decorated}}
}

type replayOnlyLLM struct{ next ports.LLM }

func (r replayOnlyLLM) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	req.Meta.ForceReplay = true
	return r.next.JSON(ctx, req, schema)
}

func (r replayOnlyLLM) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	req.Meta.ForceReplay = true
	return r.next.StreamText(ctx, req)
}

var _ ports.LLM = replayOnlyLLM{}
