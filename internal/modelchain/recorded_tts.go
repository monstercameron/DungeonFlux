package modelchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

const pcmRecordingMIME = "application/vnd.dungeonflux.pcm-recording+json;v=1"

// RecordReplayTTS stores complete PCM streams and serves them when ForceReplay
// is set, or when the provider fails before returning a stream. A replay miss
// returns an error so the engine can select its existing canned speech policy.
func RecordReplayTTS(next ports.TTS, store ports.Recordings, adapter string) ports.TTS {
	return recordedTTS{next: next, store: store, adapter: adapter}
}

type recordedTTS struct {
	next    ports.TTS
	store   ports.Recordings
	adapter string
}

func (r recordedTTS) Stream(ctx context.Context, req ports.TTSRequest, text ports.TextStream) (ports.PCMStream, error) {
	if err := ctx.Err(); err != nil {
		closeText(text)
		return nil, err
	}
	key := ttsRecordingKey(r.adapter, req)
	if req.Meta.ForceReplay {
		closeText(text)
		return r.replay(ctx, key)
	}
	if r.next == nil {
		closeText(text)
		return r.replay(ctx, key)
	}
	stream, err := r.next.Stream(ctx, req, text)
	if canceled := cancellationError(ctx, err); canceled != nil {
		if stream != nil {
			_ = stream.Close()
		}
		closeText(text)
		return nil, canceled
	}
	if err != nil || stream == nil {
		if stream != nil {
			_ = stream.Close()
		}
		closeText(text)
		if replay, replayErr := r.replay(ctx, key); replayErr == nil {
			return replay, nil
		}
		if err == nil {
			err = errors.New("modelchain: TTS returned no stream")
		}
		return nil, err
	}
	return newRecordedPCM(ctx, stream, req.SampleRate, func(ctx context.Context, chunks []ports.PCMChunk) error {
		if r.store == nil {
			return nil
		}
		data, err := json.Marshal(chunks)
		if err != nil {
			return err
		}
		return r.store.Put(ctx, key, domain.Recording{Audio: data, MIME: pcmRecordingMIME})
	}), nil
}

func (r recordedTTS) replay(ctx context.Context, key ports.RecKey) (ports.PCMStream, error) {
	if r.store == nil {
		return nil, errors.New("modelchain: speech recordings unavailable")
	}
	record, ok, err := r.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if !ok || record.MIME != pcmRecordingMIME || len(record.Audio) > 2*maxRecordedPCMBytes {
		return nil, errors.New("modelchain: speech recording unavailable")
	}
	var chunks []ports.PCMChunk
	if err := json.Unmarshal(record.Audio, &chunks); err != nil {
		return nil, fmt.Errorf("decode speech recording: %w", err)
	}
	if len(chunks) == 0 || len(chunks) > maxRecordedPCMChunks {
		return nil, errors.New("modelchain: invalid speech recording")
	}
	size := 0
	for _, chunk := range chunks {
		size += len(chunk.S16LE)
		if chunk.SampleRate <= 0 || len(chunk.S16LE) == 0 || len(chunk.S16LE)%2 != 0 || size > maxRecordedPCMBytes {
			return nil, errors.New("modelchain: invalid speech recording PCM")
		}
	}
	return &replayedPCM{ctx: ctx, chunks: chunks}, nil
}

func ttsRecordingKey(adapter string, req ports.TTSRequest) ports.RecKey {
	// Spoken roles use the room's position, independent of generated line IDs.
	// Pre-rendered lines carry a stable content ID to distinguish each asset.
	id := domain.UtteranceID("")
	if req.Meta.Role == "" {
		id = req.Meta.UtteranceID
	}
	data, _ := json.Marshal([]any{adapter, req.Meta.Role, req.Meta.Locale, req.VoiceID, req.SampleRate, id})
	digest := sha256.Sum256(data)
	return ports.RecKey{Adapter: "tts-v1:" + hex.EncodeToString(digest[:]), Phase: req.Meta.Phase, Seat: req.Meta.Seat, Index: req.Meta.Index}
}

func closeText(text ports.TextStream) {
	if text != nil {
		_ = text.Close()
	}
}

type replayedPCM struct {
	mu     sync.Mutex
	ctx    context.Context
	chunks []ports.PCMChunk
}

func (r *replayedPCM) Recv() (ports.PCMChunk, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ctx.Err(); err != nil {
		return ports.PCMChunk{}, err
	}
	if len(r.chunks) == 0 {
		return ports.PCMChunk{}, io.EOF
	}
	chunk := r.chunks[0]
	r.chunks = r.chunks[1:]
	return chunk, nil
}

func (r *replayedPCM) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chunks = nil
	return nil
}
