package media

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// ClipDownloader retrieves a completed video URL while it is still valid.
type ClipDownloader func(context.Context, string) ([]byte, error)

// ClipConfig supplies video generation and persistence dependencies.
type ClipConfig struct {
	Videos         ports.VideoGen
	Assets         ports.AssetWriter
	Download       ClipDownloader
	Clock          clock.Clock
	PollInterval   time.Duration
	References     map[domain.SeatID]ReferenceAssets
	ReferenceFrame func(context.Context, []domain.AssetID) ([]byte, error)
}

// ClipExecutor runs GenerateClip effects until a video is ready or its
// context deadline expires.
type ClipExecutor struct {
	videos         ports.VideoGen
	assets         ports.AssetWriter
	download       ClipDownloader
	clock          clock.Clock
	pollInterval   time.Duration
	references     map[domain.SeatID]ReferenceAssets
	referenceFrame func(context.Context, []domain.AssetID) ([]byte, error)
}

// NewClipExecutor constructs a deadline-aware clip executor.
func NewClipExecutor(config ClipConfig) *ClipExecutor {
	interval := config.PollInterval
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}
	clk := config.Clock
	if clk == nil {
		clk = clock.Real{}
	}
	return &ClipExecutor{videos: config.Videos, assets: config.Assets, download: config.Download, clock: clk, pollInterval: interval, references: cloneReferenceAssets(config.References), referenceFrame: config.ReferenceFrame}
}

// Execute submits, polls, downloads, and stores one generated video.
func (e *ClipExecutor) Execute(ctx context.Context, effect domain.GenerateClip, scope domain.Scope, in ports.Inbox) {
	data, err := e.generate(ctx, effect)
	if err == nil {
		var asset domain.Asset
		asset, err = e.assets.Write(ctx, vocab.AssetVideo, "video/mp4", data, ports.AssetMeta{DurationMS: 5000})
		if err == nil {
			post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: effect.Slot, Asset: asset}})
			return
		}
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: effect.Slot, FailureKind: failureKind(err)}})
}

func (e *ClipExecutor) generate(ctx context.Context, effect domain.GenerateClip) ([]byte, error) {
	if e == nil || e.videos == nil || e.assets == nil || e.download == nil {
		return nil, errors.New("clip: dependencies are incomplete")
	}
	firstFrame := append([]byte(nil), effect.FirstFrame...)
	if references := e.references[seatFromMediaSlot(effect.Slot)]; references.Ready() {
		requestIDs := references.IDs()
		if e.referenceFrame != nil {
			frame, err := e.referenceFrame(ctx, requestIDs)
			if err != nil {
				return nil, fmt.Errorf("clip reference frame: %w", err)
			}
			firstFrame = frame
		}
		effect.Shot = referencePrompt(effect.Shot, references)
	}
	request := ports.VideoRequest{FirstFrame: firstFrame, LastFrame: effect.LastFrame, Prompt: effect.Shot, Seconds: 5, Resolution: effect.Resolution}
	job, err := e.videos.Submit(ctx, request)
	if err != nil {
		return nil, err
	}
	if job.ID == "" {
		return nil, errors.New("clip: video vendor returned an empty job ID")
	}
	for {
		status, pollErr := e.videos.Poll(ctx, job)
		if pollErr != nil {
			return nil, pollErr
		}
		switch status.State {
		case vocab.JobDone:
			if status.URL == "" {
				return nil, errors.New("clip: completed job has no URL")
			}
			data, downloadErr := e.download(ctx, status.URL)
			if downloadErr != nil {
				return nil, downloadErr
			}
			if len(data) == 0 {
				return nil, errors.New("clip: downloaded video is empty")
			}
			return data, nil
		case vocab.JobFailed:
			return nil, errors.New("clip: video generation failed")
		}
		if err := e.wait(ctx); err != nil {
			return nil, err
		}
	}
}

func (e *ClipExecutor) wait(ctx context.Context) error {
	timer := e.clock.NewTimer(e.pollInterval)
	defer timer.Stop()
	select {
	case <-timer.C():
		return nil
	case <-ctx.Done():
		return fmt.Errorf("clip deadline: %w", ctx.Err())
	}
}
