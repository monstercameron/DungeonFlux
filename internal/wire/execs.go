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
	"sync"
	"time"

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

func (i *roomInbox) PostAndWait(ctx context.Context, env domain.Envelope) (domain.Ack, bool) {
	if i == nil || i.room == nil {
		return domain.Ack{}, false
	}
	reply := make(chan domain.Ack, 1)
	env.Reply = reply
	if !i.room.Post(ctx, env) {
		return domain.Ack{}, false
	}
	select {
	case ack := <-reply:
		return ack, true
	case <-ctx.Done():
		return domain.Ack{}, false
	}
}

func newExecutors(cfg configForWire, audio ports.AudioOut) (*runtime.Runner, *roomInbox, error) {
	ledger, err := newBudget(cfg.config)
	if err != nil {
		return nil, nil, err
	}
	set, err := buildAdapters(cfg.config, cfg.logger)
	if err != nil {
		return nil, nil, err
	}
	set.llm = sequenceLLM(cfg.config, set.llm, cfg.recordings)
	inbox := &roomInbox{}
	runner := runtime.NewRunner(inbox, cfg.logger)
	assets := newAssetStore(cfg.config.Server.DataDir)
	fakeMode := isFakeMode(cfg.config)
	assembler := voicein.NewAssembler()
	transcriber, err := voicein.NewTranscriber(set.stt, assembler)
	if err != nil {
		return nil, nil, err
	}
	if _, ok := set.tts.(fakeTTS); ok {
		set.tts = fakeTTS{read: assets.Read}
	}
	pcm := voiceout.NewPCMExecutor(set.tts, audio)
	canned := voiceout.NewCannedExecutor(assets, audio)
	interpret := llmexec.NewInterpretExecutor(llmexec.InterpretConfig{LLM: set.llm})
	composeSource := assets.Read
	if fakeMode {
		composeSource = func(ctx context.Context, id domain.AssetID) ([]byte, error) {
			data, err := assets.Read(ctx, id)
			if err == nil {
				return data, nil
			}
			return fakePNG(), nil
		}
	}
	references := &referenceRegistry{assets: make(map[domain.SeatID]media.ReferenceAssets)}
	referenceInbox := func(in ports.Inbox) ports.Inbox {
		return &referenceCaptureInbox{next: in, registry: references}
	}
	voicePack := media.NewVoicePackExecutor(media.VoicePackConfig{
		Sounds: set.sound, Assets: assets, Budget: ledger, Fake: fakeMode,
		Fallbacks: loadVoiceFallbacks(filepath.Join("artifacts", "runtime", "buildtime", "manifest.json")),
	})
	runtime.Handle(runner, loggedExecutor(cfg.logger, transcriber.Execute))
	runtime.Handle(runner, loggedExecutor(cfg.logger, interpret.Execute))
	runtime.Handle(runner, loggedExecutor(cfg.logger, llmexec.NewCharacterFlavorExecutor(set.llm).Execute))
	runtime.Handle(runner, loggedExecutor(cfg.logger, pcm.StartLine))
	runtime.Handle(runner, loggedExecutor(cfg.logger, canned.PlayCanned))
	runtime.Handle(runner, loggedExecutor(cfg.logger, llmexec.NewPrerenderTextExecutor(set.llm).Execute))
	runtime.Handle(runner, loggedExecutor(cfg.logger, voiceout.NewRenderLinesExecutor(set.tts, assets).Execute))
	runtime.Handle(runner, loggedExecutor(cfg.logger, func(ctx context.Context, effect domain.GenerateImage, scope domain.Scope, in ports.Inbox) {
		config := media.PortraitConfig{Images: set.image, Assets: assets, References: references.snapshot(), ReferenceSource: assets.Read}
		media.NewPortraitExecutor(config).Execute(ctx, effect, scope, in)
	}))
	runtime.Handle(runner, loggedExecutor(cfg.logger, func(ctx context.Context, effect domain.GenerateCharacterReference, scope domain.Scope, in ports.Inbox) {
		fallbackSheet, fallbacks := referenceFallback(effect)
		config := media.ReferenceConfig{Images: set.image, Assets: assets, Budget: ledger, FallbackSheet: fallbackSheet, Fallbacks: fallbacks}
		ref := media.NewReferenceExecutor(config)
		captured := referenceInbox(in)
		var group sync.WaitGroup
		group.Add(2)
		go func() { defer group.Done(); ref.Execute(ctx, effect, scope, captured) }()
		go func() { defer group.Done(); voicePack.Execute(ctx, effect, scope, in) }()
		group.Wait()
	}))
	runtime.Handle(runner, loggedExecutor(cfg.logger, func(ctx context.Context, effect domain.ComposeStill, scope domain.Scope, in ports.Inbox) {
		source := media.ReferenceSource(assets.Read)
		config := media.ComposeStillConfig{Source: composeSource, Assets: assets, References: references.forEffect(effect.Slot), ReferenceInput: referenceInput(source)}
		media.NewComposeStillExecutor(config).Execute(ctx, effect, scope, in)
	}))
	runtime.Handle(runner, loggedExecutor(cfg.logger, func(ctx context.Context, effect domain.GenerateClip, scope domain.Scope, in ports.Inbox) {
		source := media.ReferenceSource(assets.Read)
		config := media.ClipConfig{Videos: set.video, Assets: assets, References: references.forEffect(effect.Slot), ReferenceFrame: referenceFrame(source)}
		config.Download = func(downloadCtx context.Context, url string) ([]byte, error) {
			downloader, ok := set.video.(interface {
				Download(context.Context, string) ([]byte, error)
			})
			if !ok {
				return nil, errors.New("wire: video adapter does not support downloads")
			}
			return downloader.Download(downloadCtx, url)
		}
		media.NewClipExecutor(config).Execute(ctx, effect, scope, in)
	}))
	runtime.Handle(runner, loggedExecutor(cfg.logger, billboardExecutor(cfg.config.Server.DataDir, fakeMode)))
	runtime.Handle(runner, loggedExecutor(cfg.logger, func(_ context.Context, effect domain.ReleaseLine, _ domain.Scope, _ ports.Inbox) {
		pcm.Cancel(effect.UtteranceID)
		canned.Cancel(effect.UtteranceID)
	}))
	runtime.Handle(runner, loggedExecutor(cfg.logger, func(_ context.Context, effect domain.DropLine, _ domain.Scope, _ ports.Inbox) {
		pcm.Cancel(effect.UtteranceID)
		canned.Cancel(effect.UtteranceID)
	}))
	runtime.Handle(runner, loggedExecutor(cfg.logger, func(_ context.Context, effect domain.TalkStop, _ domain.Scope, _ ports.Inbox) {
		cfg.logger.Debug("talk stop requested", "seat", effect.Seat, "reason", effect.Reason)
	}))
	return runner, inbox, nil
}

func loggedExecutor[E domain.Effect](logger *slog.Logger, next runtime.Executor[E]) runtime.Executor[E] {
	return func(ctx context.Context, effect E, scope domain.Scope, in ports.Inbox) {
		started := time.Now()
		outcome := "completed"
		if next != nil {
			next(ctx, effect, scope, in)
		} else {
			outcome = "missing"
		}
		if ctx.Err() != nil {
			outcome = "canceled"
		}
		if logger != nil {
			logger.Info("effect executed", "effect", effect.Kind(), "scope", scope, "dur_ms", time.Since(started).Milliseconds(), "outcome", outcome)
		}
	}
}

type configForWire struct {
	config     config.Config
	logger     *slog.Logger
	recordings ports.Recordings
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

func billboardExecutor(root string, fake bool) runtime.Executor[domain.GenerateBillboardLoops] {
	_ = root
	return func(ctx context.Context, effect domain.GenerateBillboardLoops, scope domain.Scope, in ports.Inbox) {
		manifest, err := LoadManifest(filepath.Join("artifacts", "runtime", "buildtime", "manifest.json"), nil)
		if err != nil {
			if fake && in != nil {
				for index, clip := range effect.Clips {
					in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: fmt.Sprintf("billboard:%d:%d", effect.Seat, index), Asset: domain.Asset{ID: domain.AssetID(clip), Kind: string(vocab.AssetVideo), MIME: "video/mp4"}}})
				}
				return
			}
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
