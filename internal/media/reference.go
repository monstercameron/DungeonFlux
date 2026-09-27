package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/content/prompts"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// ReferenceConfig supplies dependencies for the turnaround executor.
type ReferenceConfig struct {
	Images        ports.ImageGen
	Assets        ports.AssetWriter
	Pool          *Pool
	Budget        *budget.Ledger
	EstimateUSD   float64
	FallbackSheet []byte
	Fallbacks     map[vocab.ReferenceAngle][]byte
}

// ReferenceAssets identifies the immutable assets used to condition a visual.
type ReferenceAssets struct {
	Sheet  domain.AssetID
	Angles map[vocab.ReferenceAngle]domain.AssetID
}

// Ready reports whether all four canonical angle crops are available.
func (r ReferenceAssets) Ready() bool {
	if r.Sheet == "" {
		return false
	}
	for _, angle := range vocab.ReferenceAngles() {
		if r.Angles[angle] == "" {
			return false
		}
	}
	return true
}

// IDs returns the sheet and angle IDs in stable conditioning order.
func (r ReferenceAssets) IDs() []domain.AssetID {
	ids := make([]domain.AssetID, 0, 5)
	if r.Sheet != "" {
		ids = append(ids, r.Sheet)
	}
	for _, angle := range vocab.ReferenceAngles() {
		if id := r.Angles[angle]; id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// ReferenceSource loads a stored reference crop by immutable asset ID.
type ReferenceSource func(context.Context, domain.AssetID) ([]byte, error)

func referencePrompt(prompt string, references ReferenceAssets) string {
	if !references.Ready() {
		return prompt
	}
	ids := make([]string, 0, len(references.IDs()))
	for _, id := range references.IDs() {
		ids = append(ids, string(id))
	}
	return strings.TrimSpace(prompt) + " Identity reference assets: " + strings.Join(ids, ", ") + ". Preserve the same hero."
}

func loadReferenceImages(ctx context.Context, source ReferenceSource, references ReferenceAssets) ([][]byte, error) {
	if source == nil || !references.Ready() {
		return nil, nil
	}
	ids := references.IDs()
	images := make([][]byte, 0, len(ids))
	for _, id := range ids {
		data, err := source(ctx, id)
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("reference asset %q is empty", id)
		}
		images = append(images, append([]byte(nil), data...))
	}
	return images, nil
}

// ReferenceExecutor creates and persists a four-angle character turnaround.
type ReferenceExecutor struct {
	images      ports.ImageGen
	assets      ports.AssetWriter
	pool        *Pool
	ledger      *budget.Ledger
	estimateUSD float64
	fallback    []byte
	fallbacks   map[vocab.ReferenceAngle][]byte
}

// NewReferenceExecutor constructs a turnaround executor from injected services.
func NewReferenceExecutor(config ReferenceConfig) *ReferenceExecutor {
	amount := config.EstimateUSD
	if amount <= 0 {
		amount = 0.07
	}
	return &ReferenceExecutor{
		images: config.Images, assets: config.Assets, pool: config.Pool,
		ledger: config.Budget, estimateUSD: amount,
		fallback:  append([]byte(nil), config.FallbackSheet...),
		fallbacks: cloneReferenceFallbacks(config.Fallbacks),
	}
}

// Execute generates a sheet, crops its canonical angles, and posts one ready
// event for the sheet and every angle. A failure serves template placeholders.
func (e *ReferenceExecutor) Execute(ctx context.Context, effect domain.GenerateCharacterReference, scope domain.Scope, in ports.Inbox) {
	if effect.Scope != (domain.Scope{}) {
		scope = effect.Scope
	}
	if e == nil {
		postFailures(ctx, effect.Seat, scope, in, vocab.ErrUnavailable)
		return
	}
	if e.images == nil || e.assets == nil {
		e.postFallback(ctx, effect, scope, in, vocab.ErrUnavailable)
		return
	}
	reservation, err := e.reserve()
	if err != nil {
		e.postFallback(ctx, effect, scope, in, vocab.ErrRateLimited)
		return
	}
	data, err := e.generate(ctx, effect)
	if err != nil {
		if reservation != nil {
			reservation.Release()
		}
		e.postFallback(ctx, effect, scope, in, failureKind(err))
		return
	}
	if reservation != nil {
		_, _ = reservation.Settle(e.estimateUSD)
	}
	if err := e.publish(ctx, effect, scope, in, data); err != nil {
		e.postFallback(ctx, effect, scope, in, failureKind(err))
	}
}

func (e *ReferenceExecutor) generate(ctx context.Context, effect domain.GenerateCharacterReference) ([]byte, error) {
	request := ports.ImageRequest{Prompt: prompts.ReferencePrompt(effect.Species, effect.Gender, effect.Class, effect.Name, effect.Flavor), Size: "1536x1024"}
	var stream ports.ImageStream
	job := func(run context.Context) error {
		var err error
		stream, err = e.images.Generate(run, request)
		return err
	}
	var err error
	if e.pool != nil {
		err = e.pool.Run(ctx, vocab.VendorOpenAI, job)
	} else {
		err = job(ctx)
	}
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, errors.New("reference: image stream is nil")
	}
	defer stream.Close()
	var latest []byte
	for {
		event, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			if len(latest) == 0 {
				return nil, errors.New("reference: stream had no image")
			}
			return latest, nil
		}
		if recvErr != nil {
			return nil, recvErr
		}
		if len(event.PNG) > 0 {
			latest = append(latest[:0], event.PNG...)
		}
	}
}

func (e *ReferenceExecutor) publish(ctx context.Context, effect domain.GenerateCharacterReference, scope domain.Scope, in ports.Inbox, data []byte) error {
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("reference: decode sheet: %w", err)
	}
	hash := referenceHash(effect)
	sheet, err := e.assets.Write(ctx, vocab.AssetImage, "image/png", data, ports.AssetMeta{InputHash: hash})
	if err != nil {
		return err
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: referenceSlot(effect.Seat, "sheet"), Asset: sheet}})
	for index, angle := range vocab.ReferenceAngles() {
		crop, cropErr := cropPanel(decoded, index)
		if cropErr != nil {
			return cropErr
		}
		asset, writeErr := e.assets.Write(ctx, vocab.AssetImage, "image/png", crop, ports.AssetMeta{InputHash: hash})
		if writeErr != nil {
			return writeErr
		}
		post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: referenceSlot(effect.Seat, string(angle)), Asset: asset}})
	}
	return nil
}

func (e *ReferenceExecutor) postFallback(ctx context.Context, effect domain.GenerateCharacterReference, scope domain.Scope, in ports.Inbox, kind vocab.ErrKind) {
	if e == nil || e.assets == nil {
		postFailures(ctx, effect.Seat, scope, in, kind)
		return
	}
	sheetData := e.fallback
	if len(sheetData) == 0 {
		sheetData = placeholderSheet()
	}
	decoded, err := png.Decode(bytes.NewReader(sheetData))
	if err != nil {
		postFailures(ctx, effect.Seat, scope, in, kind)
		return
	}
	sheet, err := e.assets.Write(ctx, vocab.AssetImage, "image/png", sheetData, ports.AssetMeta{InputHash: "reference-fallback"})
	if err != nil {
		postFailures(ctx, effect.Seat, scope, in, failureKind(err))
		return
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: referenceSlot(effect.Seat, "sheet"), Asset: sheet}})
	for index, angle := range vocab.ReferenceAngles() {
		data := e.fallbacks[angle]
		if len(data) == 0 {
			data, err = cropPanel(decoded, index)
			if err != nil {
				postFailures(ctx, effect.Seat, scope, in, kind)
				return
			}
		}
		asset, writeErr := e.assets.Write(ctx, vocab.AssetImage, "image/png", data, ports.AssetMeta{InputHash: "reference-fallback"})
		if writeErr != nil {
			postFailures(ctx, effect.Seat, scope, in, failureKind(writeErr))
			return
		}
		post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: referenceSlot(effect.Seat, string(angle)), Asset: asset}})
	}
}

func cropPanel(source image.Image, index int) ([]byte, error) {
	width, height := source.Bounds().Dx(), source.Bounds().Dy()
	if width == 0 || height == 0 || index < 0 || index >= len(vocab.ReferenceAngles()) {
		return nil, errors.New("reference: invalid sheet dimensions")
	}
	start := width * index / len(vocab.ReferenceAngles())
	end := width * (index + 1) / len(vocab.ReferenceAngles())
	rect := image.Rect(start, 0, end, height)
	destination := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(destination, destination.Bounds(), source, rect.Min, draw.Src)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, destination); err != nil {
		return nil, fmt.Errorf("reference: encode crop: %w", err)
	}
	return encoded.Bytes(), nil
}

func placeholderSheet() []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{R: 15, G: 17, B: 23, A: 255}}, image.Point{}, draw.Src)
	for index := range vocab.ReferenceAngles() {
		left := index * 64
		body := image.Rect(left+22, 56, left+42, 210)
		draw.Draw(canvas, body, &image.Uniform{C: color.RGBA{R: 80, G: 104, B: 122, A: 255}}, image.Point{}, draw.Src)
		draw.Draw(canvas, image.Rect(left+25, 32, left+39, 56), &image.Uniform{C: color.RGBA{R: 173, G: 133, B: 78, A: 255}}, image.Point{}, draw.Src)
	}
	var encoded bytes.Buffer
	_ = png.Encode(&encoded, canvas)
	return encoded.Bytes()
}

func (e *ReferenceExecutor) reserve() (*budget.Reservation, error) {
	if e.ledger == nil {
		return nil, nil
	}
	return e.ledger.Reserve(vocab.VendorOpenAI, e.estimateUSD)
}

func referenceSlot(seat domain.SeatID, angle string) string {
	return fmt.Sprintf("reference:%d:%s", seat, angle)
}

func referenceHash(effect domain.GenerateCharacterReference) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%s|%s|%s", effect.Seat, effect.Species, effect.Gender, effect.Class, effect.Name, effect.Flavor)))
	return hex.EncodeToString(hash[:])
}

func cloneReferenceFallbacks(input map[vocab.ReferenceAngle][]byte) map[vocab.ReferenceAngle][]byte {
	if len(input) == 0 {
		return nil
	}
	output := make(map[vocab.ReferenceAngle][]byte, len(input))
	for angle, data := range input {
		output[angle] = append([]byte(nil), data...)
	}
	return output
}

func cloneReferenceAssets(input map[domain.SeatID]ReferenceAssets) map[domain.SeatID]ReferenceAssets {
	if len(input) == 0 {
		return nil
	}
	output := make(map[domain.SeatID]ReferenceAssets, len(input))
	for seat, references := range input {
		angles := make(map[vocab.ReferenceAngle]domain.AssetID, len(references.Angles))
		for angle, id := range references.Angles {
			angles[angle] = id
		}
		references.Angles = angles
		output[seat] = references
	}
	return output
}

func postFailures(ctx context.Context, seat domain.SeatID, scope domain.Scope, in ports.Inbox, kind vocab.ErrKind) {
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: referenceSlot(seat, "sheet"), FailureKind: kind}})
	for _, angle := range vocab.ReferenceAngles() {
		post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: referenceSlot(seat, string(angle)), FailureKind: kind}})
	}
}
