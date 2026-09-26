package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const soundCacheAdapter = "elevenlabs.sound"

// SoundRequest describes one logical sound asset requested by the runtime.
type SoundRequest struct {
	Slot    string
	Logical string
	Kind    vocab.SoundKind
	Prompt  string
	Seconds float64
	Loop    bool
	Meta    ports.CallMeta
}

// SoundManifest contains build-time assets keyed by logical name.
type SoundManifest map[string]domain.Asset

// SoundConfig supplies the sound executor's generation and storage services.
type SoundConfig struct {
	Sounds       ports.SoundGen
	Assets       ports.AssetWriter
	Cache        ports.Cache
	Manifest     SoundManifest
	Pool         *Pool
	Timeout      time.Duration
	BudgetCapUSD float64
	Normalize    func([]byte, string) ([]byte, error)
	Silence      []byte
}

// SoundExecutor resolves, generates, stores, and publishes sound assets.
type SoundExecutor struct {
	sounds       ports.SoundGen
	assets       ports.AssetWriter
	cache        ports.Cache
	manifest     SoundManifest
	pool         *Pool
	timeout      time.Duration
	budgetCapUSD float64
	normalize    func([]byte, string) ([]byte, error)
	silence      []byte
	mu           sync.Mutex
	reservedUSD  float64
}

// NewSoundExecutor constructs a sound executor from injected dependencies.
func NewSoundExecutor(config SoundConfig) *SoundExecutor {
	normalize := config.Normalize
	if normalize == nil {
		normalize = func(data []byte, _ string) ([]byte, error) { return append([]byte(nil), data...), nil }
	}
	manifest := make(SoundManifest, len(config.Manifest))
	for key, asset := range config.Manifest {
		manifest[key] = asset
	}
	return &SoundExecutor{
		sounds: config.Sounds, assets: config.Assets, cache: config.Cache,
		manifest: manifest, pool: config.Pool, timeout: config.Timeout,
		budgetCapUSD: config.BudgetCapUSD, normalize: normalize,
		silence: append([]byte(nil), config.Silence...),
	}
}

// Execute resolves a sound request and posts asset_ready or asset_failed.
func (e *SoundExecutor) Execute(ctx context.Context, request SoundRequest, scope domain.Scope, in ports.Inbox) {
	if e == nil || e.assets == nil {
		post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: request.Slot, FailureKind: vocab.ErrUnavailable}})
		return
	}
	if asset, ok := e.manifest[request.Logical]; ok && asset.ID != "" {
		post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: request.Slot, Asset: asset}})
		return
	}
	data, mime, duration, err := e.resolve(ctx, request)
	if err == nil {
		asset, writeErr := e.assets.Write(ctx, assetKind(request.Kind), mime, data, ports.AssetMeta{InputHash: soundHash(request), DurationMS: duration})
		if writeErr == nil {
			post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: request.Slot, Asset: asset}})
			return
		}
		err = writeErr
	}
	if isTimeout(err) {
		e.postSilence(ctx, request, scope, in)
		return
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: request.Slot, FailureKind: failureKind(err)}})
}

func (e *SoundExecutor) resolve(ctx context.Context, request SoundRequest) ([]byte, string, int, error) {
	if e == nil || e.sounds == nil || e.assets == nil {
		return nil, "", 0, errors.New("sound: dependencies are incomplete")
	}
	hash := soundHash(request)
	if e.cache != nil {
		if data, ok, err := e.cache.Get(ctx, soundCacheAdapter, hash); err == nil && ok && len(data) > 0 {
			return e.normalized(data, "audio/mpeg", request)
		}
	}
	estimate := soundEstimate(request)
	if !e.reserve(estimate) {
		return nil, "", 0, errors.New("sound: budget cap reached")
	}
	defer e.release(estimate)
	runCtx := ctx
	var cancel context.CancelFunc
	if e.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}
	var sound ports.Sound
	job := func(callCtx context.Context) error {
		var err error
		sound, err = e.sounds.Generate(callCtx, ports.SoundRequest{Meta: request.Meta, Kind: request.Kind, Prompt: request.Prompt, Seconds: request.Seconds, Loop: request.Loop})
		return err
	}
	var err error
	if e.pool != nil {
		err = e.pool.Run(runCtx, vocab.VendorElevenLabs, job)
	} else {
		err = job(runCtx)
	}
	if err != nil {
		return nil, "", 0, err
	}
	data, mime, duration, err := e.normalized(sound.Bytes, sound.MIME, request)
	if err != nil {
		return nil, "", 0, err
	}
	if e.cache != nil {
		_ = e.cache.Put(ctx, soundCacheAdapter, hash, data)
	}
	return data, mime, duration, nil
}

func (e *SoundExecutor) normalized(data []byte, mime string, request SoundRequest) ([]byte, string, int, error) {
	if len(data) == 0 {
		return nil, "", 0, errors.New("sound: generated audio is empty")
	}
	if mime == "" {
		mime = "audio/mpeg"
	}
	value, err := e.normalize(data, mime)
	if err != nil {
		return nil, "", 0, fmt.Errorf("sound: normalize: %w", err)
	}
	if len(value) == 0 {
		return nil, "", 0, errors.New("sound: normalized audio is empty")
	}
	return value, mime, int(request.Seconds * 1000), nil
}

func (e *SoundExecutor) postSilence(ctx context.Context, request SoundRequest, scope domain.Scope, in ports.Inbox) {
	data := append([]byte(nil), e.silence...)
	if len(data) == 0 {
		data = silentWAV(1000)
	}
	asset, err := e.assets.Write(ctx, assetKind(request.Kind), "audio/wav", data, ports.AssetMeta{InputHash: soundHash(request), DurationMS: 1000})
	if err != nil {
		post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: request.Slot, FailureKind: failureKind(err)}})
		return
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: request.Slot, Asset: asset}})
}

func (e *SoundExecutor) reserve(amount float64) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.budgetCapUSD > 0 && e.reservedUSD+amount > e.budgetCapUSD {
		return false
	}
	e.reservedUSD += amount
	return true
}

func (e *SoundExecutor) release(amount float64) {
	e.mu.Lock()
	e.reservedUSD -= amount
	if e.reservedUSD < 0 {
		e.reservedUSD = 0
	}
	e.mu.Unlock()
}

func soundHash(request SoundRequest) string {
	data, _ := json.Marshal(struct {
		Kind    vocab.SoundKind `json:"kind"`
		Prompt  string          `json:"prompt"`
		Seconds float64         `json:"seconds"`
		Loop    bool            `json:"loop"`
	}{request.Kind, request.Prompt, request.Seconds, request.Loop})
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func soundEstimate(request SoundRequest) float64 {
	seconds := request.Seconds
	if seconds < 1 {
		seconds = 1
	}
	rate := 0.12
	if request.Kind == vocab.SoundMusic {
		rate = 0.15
	}
	return seconds / 60 * rate
}

func assetKind(kind vocab.SoundKind) vocab.AssetKind {
	if kind == vocab.SoundMusic {
		return vocab.AssetMusic
	}
	return vocab.AssetSFX
}

func silentWAV(durationMS int) []byte {
	const sampleRate = 8000
	samples := sampleRate * durationMS / 1000
	dataSize := samples * 2
	data := make([]byte, 44+dataSize)
	copy(data, []byte("RIFF"))
	putLE32(data[4:], uint32(36+dataSize))
	copy(data[8:], []byte("WAVEfmt "))
	putLE32(data[16:], 16)
	putLE16(data[20:], 1)
	putLE16(data[22:], 1)
	putLE32(data[24:], sampleRate)
	putLE32(data[28:], sampleRate*2)
	putLE16(data[32:], 2)
	putLE16(data[34:], 16)
	copy(data[36:], []byte("data"))
	putLE32(data[40:], uint32(dataSize))
	return data
}

func putLE16(dst []byte, value uint16) { dst[0] = byte(value); dst[1] = byte(value >> 8) }

func putLE32(dst []byte, value uint32) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
	dst[2] = byte(value >> 16)
	dst[3] = byte(value >> 24)
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var callErr *ports.CallError
	return errors.As(err, &callErr) && callErr.Kind == vocab.ErrTimeout
}
