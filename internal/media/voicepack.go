package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/content/prompts"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const voicePackCacheAdapter = "elevenlabs.voicepack"

// VoicePackRequest contains the character inputs needed for a sound pack.
type VoicePackRequest struct {
	Seat    domain.SeatID
	Species string
	Gender  string
	Class   string
	Flavor  string
	Scope   domain.Scope
}

// VoicePack is the completed set of named character sound assets.
type VoicePack map[prompts.VoiceCue]domain.Asset

// VoicePackConfig supplies generation, storage, cache, budget, and fallback services.
type VoicePackConfig struct {
	Sounds           ports.SoundGen
	Assets           ports.AssetWriter
	Cache            ports.Cache
	Pool             *Pool
	Budget           *budget.Ledger
	CostPerSecondUSD float64
	Normalize        func([]byte, string, int) ([]byte, error)
	Fallbacks        map[string][]byte
	Fake             bool
	Tone             []byte
}

// VoicePackExecutor generates and stores one pack per locked character.
type VoicePackExecutor struct {
	sounds        ports.SoundGen
	assets        ports.AssetWriter
	cache         ports.Cache
	pool          *Pool
	budget        *budget.Ledger
	costPerSecond float64
	normalize     func([]byte, string, int) ([]byte, error)
	fallbacks     map[string][]byte
	fake          bool
	tone          []byte
}

// NewVoicePackExecutor constructs a character sound-pack executor.
func NewVoicePackExecutor(config VoicePackConfig) *VoicePackExecutor {
	cost := config.CostPerSecondUSD
	if cost <= 0 {
		cost = 0.12 / 60
	}
	normalize := config.Normalize
	if normalize == nil {
		normalize = func(data []byte, _ string, _ int) ([]byte, error) {
			return append([]byte(nil), data...), nil
		}
	}
	return &VoicePackExecutor{
		sounds: config.Sounds, assets: config.Assets, cache: config.Cache,
		pool: config.Pool, budget: config.Budget, costPerSecond: cost,
		normalize: normalize, fallbacks: cloneVoiceFallbacks(config.Fallbacks),
		fake: config.Fake, tone: append([]byte(nil), config.Tone...),
	}
}

// Execute handles the reference effect emitted when a character is locked.
func (e *VoicePackExecutor) Execute(ctx context.Context, effect domain.GenerateCharacterReference, scope domain.Scope, in ports.Inbox) {
	if effect.Scope != (domain.Scope{}) {
		scope = effect.Scope
	}
	request := VoicePackRequest{Seat: effect.Seat, Species: effect.Species, Gender: effect.Gender, Class: effect.Class, Flavor: effect.Flavor, Scope: scope}
	pack, err := e.Generate(ctx, request)
	if err != nil {
		for _, cue := range prompts.VoicePackCues() {
			post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: VoicePackSlot(effect.Seat, cue), FailureKind: failureKind(err)}})
		}
		return
	}
	for _, cue := range prompts.VoicePackCues() {
		post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: VoicePackSlot(effect.Seat, cue), Asset: pack[cue]}})
	}
}

// Generate builds and stores a pack for a direct runtime trigger.
func (e *VoicePackExecutor) Generate(ctx context.Context, request VoicePackRequest) (VoicePack, error) {
	if e == nil || e.assets == nil {
		return nil, errors.New("voicepack: asset writer is not configured")
	}
	pack := make(VoicePack, len(prompts.VoicePackCues()))
	for _, cue := range prompts.VoicePackCues() {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("voicepack: generate %s: %w", cue, err)
		}
		data, mime, duration, err := e.resolve(ctx, request, cue)
		if err != nil {
			return nil, err
		}
		asset, err := e.assets.Write(ctx, vocab.AssetSFX, mime, data, ports.AssetMeta{InputHash: voicePromptHash(request, cue), DurationMS: duration})
		if err != nil {
			return nil, fmt.Errorf("voicepack: store %s: %w", cue, err)
		}
		pack[cue] = asset
	}
	return pack, nil
}

// VoicePackSlot returns the event slot and asset name for a cue.
func VoicePackSlot(seat domain.SeatID, cue prompts.VoiceCue) string {
	return "voicepack/" + strconv.Itoa(int(seat)) + "/" + string(cue)
}

func (e *VoicePackExecutor) resolve(ctx context.Context, request VoicePackRequest, cue prompts.VoiceCue) ([]byte, string, int, error) {
	duration := prompts.VoicePackDurationSeconds(cue)
	prompt := prompts.VoicePackPrompt(request.Species, request.Gender, request.Class, request.Flavor, cue)
	hash := voicePromptHash(request, cue)
	if e.cache != nil {
		if data, ok, err := e.cache.Get(ctx, voicePackCacheAdapter, hash); err == nil && ok && len(data) > 0 {
			return e.process(data, "audio/mpeg", duration)
		}
	}
	if e.fake {
		data := e.tone
		if len(data) == 0 {
			data = shortTone(duration)
		}
		return e.process(data, "audio/wav", duration)
	}
	if e.sounds == nil {
		return e.fallback(request, cue, duration)
	}
	amount := duration * e.costPerSecond
	reservation, err := e.reserve(amount)
	if err != nil {
		return e.fallback(request, cue, duration)
	}
	sound, err := e.generate(ctx, ports.SoundRequest{Kind: vocab.SoundSFX, Prompt: prompt, Seconds: duration})
	if err != nil {
		if reservation != nil {
			reservation.Release()
		}
		return e.fallback(request, cue, duration)
	}
	if reservation != nil {
		if _, settleErr := reservation.Settle(amount); settleErr != nil {
			return e.fallback(request, cue, duration)
		}
	}
	data, mime, durationMS, err := e.process(sound.Bytes, sound.MIME, duration)
	if err != nil {
		return e.fallback(request, cue, duration)
	}
	if e.cache != nil {
		_ = e.cache.Put(ctx, voicePackCacheAdapter, hash, data)
	}
	return data, mime, durationMS, nil
}

func (e *VoicePackExecutor) generate(ctx context.Context, request ports.SoundRequest) (ports.Sound, error) {
	var sound ports.Sound
	job := func(run context.Context) error {
		var err error
		sound, err = e.sounds.Generate(run, request)
		return err
	}
	if e.pool != nil {
		return sound, e.pool.Run(ctx, vocab.VendorElevenLabs, job)
	}
	return sound, job(ctx)
}

func (e *VoicePackExecutor) process(data []byte, mime string, duration float64) ([]byte, string, int, error) {
	if len(data) == 0 {
		return nil, "", 0, errors.New("voicepack: audio is empty")
	}
	if mime == "" {
		mime = "audio/mpeg"
	}
	durationMS := int(math.Round(duration * 1000))
	processed, err := e.normalize(data, mime, durationMS)
	if err != nil {
		return nil, "", 0, fmt.Errorf("voicepack: trim and normalize: %w", err)
	}
	if len(processed) == 0 {
		return nil, "", 0, errors.New("voicepack: normalized audio is empty")
	}
	return processed, mime, durationMS, nil
}

func (e *VoicePackExecutor) fallback(request VoicePackRequest, cue prompts.VoiceCue, duration float64) ([]byte, string, int, error) {
	data := e.fallbacks[request.Class+"/"+string(cue)]
	if len(data) == 0 {
		data = e.fallbacks[string(cue)]
	}
	if len(data) == 0 {
		return nil, "", 0, fmt.Errorf("voicepack: no fallback for class %q cue %q", request.Class, cue)
	}
	return e.process(data, "audio/mpeg", duration)
}

func (e *VoicePackExecutor) reserve(amount float64) (*budget.Reservation, error) {
	if e.budget == nil {
		return nil, nil
	}
	return e.budget.Reserve(vocab.VendorElevenLabs, amount)
}

func voicePromptHash(request VoicePackRequest, cue prompts.VoiceCue) string {
	prompt := prompts.VoicePackPrompt(request.Species, request.Gender, request.Class, request.Flavor, cue)
	hash := sha256.Sum256([]byte(prompt + "|" + strconv.FormatFloat(prompts.VoicePackDurationSeconds(cue), 'f', 2, 64)))
	return hex.EncodeToString(hash[:])
}

func cloneVoiceFallbacks(input map[string][]byte) map[string][]byte {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string][]byte, len(input))
	for key, data := range input {
		output[key] = append([]byte(nil), data...)
	}
	return output
}

func shortTone(duration float64) []byte {
	const sampleRate = 8000
	samples := int(duration * sampleRate)
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
	for index := 0; index < samples; index++ {
		phase := (index * 256 * 220 / sampleRate) % 256
		amplitude := phase
		if amplitude > 128 {
			amplitude = 256 - amplitude
		}
		sample := int16((amplitude - 64) * 400)
		putLE16(data[44+index*2:], uint16(sample))
	}
	return data
}
