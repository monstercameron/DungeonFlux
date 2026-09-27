package modelchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// Cached returns an LLM decorator that stores successful JSON and text results
// under a stable hash of the adapter, request, and schema.
func Cached(next ports.LLM, store ports.Cache, adapter string) ports.LLM {
	return cachedLLM{next: next, store: store, adapter: adapter}
}

type cachedLLM struct {
	next    ports.LLM
	store   ports.Cache
	adapter string
}

func (c cachedLLM) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, err := inputHash(c.adapter, req, schema)
	if err != nil {
		return nil, err
	}
	if value, ok, getErr := c.store.Get(ctx, c.adapter, key); getErr != nil {
		return nil, getErr
	} else if ok {
		return json.RawMessage(append([]byte(nil), value...)), nil
	}
	value, err := c.next.JSON(ctx, req, schema)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.store.Put(ctx, c.adapter, key, append([]byte(nil), value...)); err != nil {
		return nil, err
	}
	return value, nil
}

func (c cachedLLM) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, err := inputHash(c.adapter, req, ports.Schema{})
	if err != nil {
		return nil, err
	}
	if value, ok, getErr := c.store.Get(ctx, c.adapter, key); getErr != nil {
		return nil, getErr
	} else if ok {
		return newReplayStream(ctx, string(value)), nil
	}
	stream, err := c.next.StreamText(ctx, req)
	if err != nil {
		return nil, err
	}
	return newStoredStream(ctx, stream, func(ctx context.Context, value []byte) error {
		return c.store.Put(ctx, c.adapter, key, value)
	}), nil
}

func inputHash(adapter string, req ports.TextRequest, schema ports.Schema) (string, error) {
	payload := struct {
		Adapter string            `json:"adapter"`
		Request ports.TextRequest `json:"request"`
		Schema  ports.Schema      `json:"schema"`
	}{adapter, req, schema}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

var _ ports.LLM = cachedLLM{}
