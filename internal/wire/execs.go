package wire

import (
	"context"
	"errors"
	"io"
	"log/slog"

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

type emptyAssetReader struct{}

func (emptyAssetReader) Open(context.Context, domain.AssetID) (io.ReadCloser, error) {
	return nil, errors.New("wire: asset bytes are not available in the runtime store")
}

func newExecutors(cfg configForWire, audio ports.AudioOut) (*runtime.Runner, *roomInbox, error) {
	set, err := buildAdapters(cfg.config, cfg.logger)
	if err != nil {
		return nil, nil, err
	}
	inbox := &roomInbox{}
	runner := runtime.NewRunner(inbox, cfg.logger)
	assets := nullAssetWriter{}
	assembler := voicein.NewAssembler()
	transcriber, err := voicein.NewTranscriber(set.stt, assembler)
	if err != nil {
		return nil, nil, err
	}
	pcm := voiceout.NewPCMExecutor(set.tts, audio)
	canned := voiceout.NewCannedExecutor(emptyAssetReader{}, audio)
	interpret := llmexec.NewInterpretExecutor(llmexec.InterpretConfig{LLM: set.llm})
	portrait := media.NewPortraitExecutor(media.PortraitConfig{Images: set.image, Assets: assets})
	compose := media.NewComposeStillExecutor(media.ComposeStillConfig{Source: func(context.Context, domain.AssetID) ([]byte, error) {
		return nil, errors.New("wire: asset source unavailable")
	}, Assets: assets})
	clip := media.NewClipExecutor(media.ClipConfig{Videos: set.video, Assets: assets, Download: func(context.Context, string) ([]byte, error) {
		return nil, errors.New("wire: clip download unavailable")
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
	runtime.Handle(runner, func(context.Context, domain.GenerateBillboardLoops, domain.Scope, ports.Inbox) {})
	runtime.Handle(runner, func(context.Context, domain.ReleaseLine, domain.Scope, ports.Inbox) {})
	runtime.Handle(runner, func(context.Context, domain.DropLine, domain.Scope, ports.Inbox) {})
	runtime.Handle(runner, func(context.Context, domain.TalkStop, domain.Scope, ports.Inbox) {})
	return runner, inbox, nil
}

type configForWire struct {
	config config.Config
	logger *slog.Logger
}

type nullAssetWriter struct{}

func (nullAssetWriter) Write(context.Context, vocab.AssetKind, string, []byte, ports.AssetMeta) (domain.Asset, error) {
	return domain.Asset{}, errors.New("wire: asset writer stand-in")
}
