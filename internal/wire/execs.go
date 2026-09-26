package wire

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/llmexec"
	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/runtime"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	voicein "github.com/monstercameron/DungeonFlux/internal/voice/in"
	voiceout "github.com/monstercameron/DungeonFlux/internal/voice/out"
)

type roomInbox struct{ room *runtime.Room }

func (i *roomInbox) Post(ctx context.Context, env domain.Envelope) bool {
	if i == nil || i.room == nil {
		return false
	}
	return i.room.Post(ctx, env)
}

func newExecutors(cfg configForWire, audio ports.AudioOut) (*runtime.Runner, *roomInbox, error) {
	if _, err := newBudget(cfg.config); err != nil {
		return nil, nil, err
	}
	set, err := buildAdapters(cfg.config, cfg.logger)
	if err != nil {
		return nil, nil, err
	}
	inbox := &roomInbox{}
	runner := runtime.NewRunner(inbox, cfg.logger)
	assets := newAssetStore(cfg.config.Server.DataDir)
	assembler := voicein.NewAssembler()
	transcriber, err := voicein.NewTranscriber(set.stt, assembler)
	if err != nil {
		return nil, nil, err
	}
	pcm := voiceout.NewPCMExecutor(set.tts, audio)
	canned := voiceout.NewCannedExecutor(assets, audio)
	interpret := llmexec.NewInterpretExecutor(llmexec.InterpretConfig{LLM: set.llm})
	portrait := media.NewPortraitExecutor(media.PortraitConfig{Images: set.image, Assets: assets})
	compose := media.NewComposeStillExecutor(media.ComposeStillConfig{Source: assets.Read, Assets: assets})
	clip := media.NewClipExecutor(media.ClipConfig{Videos: set.video, Assets: assets, Download: func(ctx context.Context, url string) ([]byte, error) {
		downloader, ok := set.video.(interface {
			Download(context.Context, string) ([]byte, error)
		})
		if !ok {
			return nil, errors.New("wire: video adapter does not support downloads")
		}
		return downloader.Download(ctx, url)
	}})
	runtime.Handle(runner, transcriber.Execute)
	runtime.Handle(runner, interpret.Execute)
	runtime.Handle(runner, llmexec.NewCharacterFlavorExecutor(set.llm).Execute)
	runtime.Handle(runner, pcm.StartLine)
	runtime.Handle(runner, canned.PlayCanned)
	runtime.Handle(runner, llmexec.NewPrerenderTextExecutor(set.llm).Execute)
	runtime.Handle(runner, voiceout.NewRenderLinesExecutor(set.tts, assets).Execute)
	runtime.Handle(runner, portrait.Execute)
	runtime.Handle(runner, compose.Execute)
	runtime.Handle(runner, clip.Execute)
	runtime.Handle(runner, billboardExecutor(cfg.config.Server.DataDir))
	runtime.Handle(runner, func(_ context.Context, effect domain.ReleaseLine, _ domain.Scope, _ ports.Inbox) {
		pcm.Cancel(effect.UtteranceID)
		canned.Cancel(effect.UtteranceID)
	})
	runtime.Handle(runner, func(_ context.Context, effect domain.DropLine, _ domain.Scope, _ ports.Inbox) {
		pcm.Cancel(effect.UtteranceID)
		canned.Cancel(effect.UtteranceID)
	})
	runtime.Handle(runner, func(_ context.Context, effect domain.TalkStop, _ domain.Scope, _ ports.Inbox) {
		cfg.logger.Debug("talk stop requested", "seat", effect.Seat, "reason", effect.Reason)
	})
	return runner, inbox, nil
}

type configForWire struct {
	config config.Config
	logger *slog.Logger
}

type assetStore struct{ root, buildtime string }

func newAssetStore(root string) *assetStore {
	return &assetStore{root: filepath.Join(root, "assets"), buildtime: filepath.Join("artifacts", "runtime", "buildtime")}
}

func (s *assetStore) Write(_ context.Context, kind vocab.AssetKind, mime string, data []byte, meta ports.AssetMeta) (domain.Asset, error) {
	if len(data) == 0 {
		return domain.Asset{}, errors.New("wire: cannot store empty asset")
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return domain.Asset{}, fmt.Errorf("wire: create asset directory: %w", err)
	}
	digest := sha256.Sum256(data)
	ext := extension(kind, mime)
	name := fmt.Sprintf("%x.%s", digest, ext)
	if err := os.WriteFile(filepath.Join(s.root, name), data, 0o600); err != nil {
		return domain.Asset{}, fmt.Errorf("wire: write asset: %w", err)
	}
	return domain.Asset{ID: domain.AssetID(name), SHA256: fmt.Sprintf("%x", digest), URL: "/assets/" + name, Kind: string(kind), MIME: mime, DurationMS: meta.DurationMS}, nil
}

func (s *assetStore) Read(ctx context.Context, id domain.AssetID) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	name := filepath.Base(string(id))
	if name == "." || name == "" || strings.Contains(name, "..") {
		return nil, errors.New("wire: invalid asset ID")
	}
	data, err := os.ReadFile(filepath.Join(s.root, name))
	if err == nil {
		return data, nil
	}
	manifest, manifestErr := LoadManifest(filepath.Join(s.buildtime, "manifest.json"), nil)
	if manifestErr != nil {
		return nil, err
	}
	for _, asset := range manifest.OneShot.Catalogue {
		if string(asset.ID) == string(id) && asset.URL != "" {
			return os.ReadFile(filepath.Join(s.buildtime, "assets", filepath.Base(asset.URL)))
		}
	}
	return nil, err
}

func (s *assetStore) Open(ctx context.Context, id domain.AssetID) (io.ReadCloser, error) {
	data, err := s.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func extension(kind vocab.AssetKind, mime string) string {
	if strings.Contains(mime, "wav") {
		return "wav"
	}
	if strings.Contains(mime, "png") {
		return "png"
	}
	if strings.Contains(mime, "mp4") {
		return "mp4"
	}
	if strings.Contains(strings.ToLower(string(kind)), "audio") {
		return "wav"
	}
	return "bin"
}

func billboardExecutor(root string) runtime.Executor[domain.GenerateBillboardLoops] {
	_ = root
	return func(ctx context.Context, effect domain.GenerateBillboardLoops, scope domain.Scope, in ports.Inbox) {
		manifest, err := LoadManifest(filepath.Join("artifacts", "runtime", "buildtime", "manifest.json"), nil)
		if err != nil {
			postAssetFailure(ctx, scope, in, vocab.ErrUnavailable)
			return
		}
		for index, clip := range effect.Clips {
			for _, asset := range manifest.OneShot.Catalogue {
				if string(asset.ID) == clip && in != nil {
					in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: fmt.Sprintf("billboard:%d:%d", effect.Seat, index), Asset: asset}})
				}
			}
		}
	}
}

func postAssetFailure(ctx context.Context, scope domain.Scope, in ports.Inbox, kind vocab.ErrKind) {
	if in != nil {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.AssetFailed{FailureKind: kind}})
	}
}
