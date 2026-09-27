package wire

import (
	"context"
	"encoding/json"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type positionedLLM struct{ next ports.LLM }

func (p positionedLLM) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	req.Meta = ports.ResolveCallMeta(ctx, req.Meta)
	return p.next.JSON(ctx, req, schema)
}

func (p positionedLLM) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	req.Meta = ports.ResolveCallMeta(ctx, req.Meta)
	return p.next.StreamText(ctx, req)
}

var _ ports.LLM = positionedLLM{}
