package wire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	imageopenai "github.com/monstercameron/DungeonFlux/internal/adapters/image/openai"
	llmanthropic "github.com/monstercameron/DungeonFlux/internal/adapters/llm/anthropic"
	llmgemini "github.com/monstercameron/DungeonFlux/internal/adapters/llm/gemini"
	llmschema "github.com/monstercameron/DungeonFlux/internal/adapters/llm/schemaflux"
	soundeleven "github.com/monstercameron/DungeonFlux/internal/adapters/sound/elevenlabs"
	stteleven "github.com/monstercameron/DungeonFlux/internal/adapters/stt/elevenlabs"
	ttseleven "github.com/monstercameron/DungeonFlux/internal/adapters/tts/elevenlabs"
	ttsopenai "github.com/monstercameron/DungeonFlux/internal/adapters/tts/openai"
	videoevolink "github.com/monstercameron/DungeonFlux/internal/adapters/video/evolink"
	videofal "github.com/monstercameron/DungeonFlux/internal/adapters/video/fal"
	videosegmind "github.com/monstercameron/DungeonFlux/internal/adapters/video/segmind"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/modelchain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type adapterSet struct {
	llm   ports.LLM
	image ports.ImageGen
	video ports.VideoGen
	stt   ports.STT
	tts   ports.TTS
	sound ports.SoundGen
}

func buildAdapters(cfg config.Config, logger *slog.Logger) (adapterSet, error) {
	if err := requireLiveKeys(cfg); err != nil {
		return adapterSet{}, err
	}
	set := adapterSet{llm: newFakeLLM(), image: newFakeImage(), video: newFakeVideo(), stt: fakeSTT{}, tts: fakeTTS{}, sound: fakeSound{}}
	var err error
	if isLive(cfg, "llm") {
		set.llm, err = liveLLM(cfg, logger)
		if err != nil {
			return adapterSet{}, err
		}
	}
	if isLive(cfg, "image") {
		set.image, err = liveImage(cfg, logger)
		if err != nil {
			return adapterSet{}, err
		}
	}
	if isLive(cfg, "video") {
		set.video, err = liveVideo(cfg, logger)
		if err != nil {
			return adapterSet{}, err
		}
	}
	if isLive(cfg, "stt") {
		set.stt, err = liveSTT(cfg, logger)
		if err != nil {
			return adapterSet{}, err
		}
	}
	if isLive(cfg, "tts") {
		set.tts, err = liveTTS(cfg, logger)
		if err != nil {
			return adapterSet{}, err
		}
	}
	if isLive(cfg, "sound") {
		set.sound, err = liveSound(cfg, logger)
		if err != nil {
			return adapterSet{}, err
		}
	}
	return set, nil
}

func requireLiveKeys(cfg config.Config) error {
	for name, adapter := range cfg.Adapters {
		if adapter.Mode == "live" && strings.TrimSpace(adapter.APIKey) == "" {
			return fmt.Errorf("wire: adapter %q requires %s", name, envForVendor(adapter.Vendor))
		}
	}
	return nil
}

func envForVendor(vendor string) string {
	for _, pair := range []struct{ vendor, env string }{{"openai", "DF_OPENAI_API_KEY"}, {"gemini", "DF_GEMINI_API_KEY"}, {"anthropic", "DF_ANTHROPIC_API_KEY"}, {"elevenlabs", "DF_ELEVENLABS_API_KEY"}, {"segmind", "DF_SEGMIND_API_KEY"}, {"evolink", "DF_EVOLINK_API_KEY"}, {"fal", "DF_FAL_KEY"}, {"cerebras", "DF_CEREBRAS_API_KEY"}, {"typesafe", "DF_TYPESAFE_API_KEY"}} {
		if pair.vendor == vendor {
			return pair.env
		}
	}
	return "DF_" + strings.ToUpper(vendor) + "_API_KEY"
}

func isLive(cfg config.Config, name string) bool { return cfg.Adapters[name].Mode == "live" }

func liveLLM(cfg config.Config, logger *slog.Logger) (ports.LLM, error) {
	links := cfg.Models.Chains["npc_reply"]
	if len(links) == 0 {
		links = []string{"openai:gpt-6-luna"}
	}
	llms := make([]ports.LLM, 0, len(links))
	for _, link := range links {
		provider, _, ok := strings.Cut(link, ":")
		if !ok {
			provider = link
		}
		if provider == "" {
			return nil, errors.New("wire: empty LLM provider")
		}
		adapter, err := llmLink(cfg, provider, logger)
		if err != nil {
			return nil, err
		}
		llms = append(llms, adapter)
	}
	return modelchain.New(llms, modelchain.Config{FirstTokenDeadline: cfg.Timeouts.SpokenFirstToken, Deadline: cfg.Timeouts.Interpret}), nil
}

func llmLink(cfg config.Config, provider string, logger *slog.Logger) (ports.LLM, error) {
	key, endpoint := vendorConfig(cfg, provider)
	switch strings.ToLower(provider) {
	case "openai", "luna", "cerebras", "qwen", "local", "llama":
		model := "gpt-6-luna"
		if strings.EqualFold(provider, "cerebras") || strings.EqualFold(provider, "qwen") {
			model = "qwen-3-32b"
		}
		if strings.EqualFold(provider, "local") || strings.EqualFold(provider, "llama") {
			model = "llama"
		}
		return llmschema.New(llmschema.Config{Provider: provider, APIKey: key, BaseURL: endpoint, Model: model, ReasoningEffort: "none", Timeout: cfg.Timeouts.Interpret})
	case "gemini":
		return llmgemini.New(key, endpoint, cfg.Timeouts.Interpret), nil
	case "anthropic", "haiku":
		return llmanthropic.New(key, endpoint, cfg.Timeouts.Interpret, logger), nil
	default:
		return nil, fmt.Errorf("wire: unsupported live LLM provider %q", provider)
	}
}

func liveImage(cfg config.Config, logger *slog.Logger) (ports.ImageGen, error) {
	adapter := cfg.Adapters["image"]
	if adapter.Vendor != "openai" {
		return nil, fmt.Errorf("wire: unsupported live image vendor %q", adapter.Vendor)
	}
	return imageopenai.New(adapter.APIKey, adapter.BaseURL, cfg.Timeouts.Portrait, logger), nil
}

func liveVideo(cfg config.Config, logger *slog.Logger) (ports.VideoGen, error) {
	adapter := cfg.Adapters["video"]
	client := httpx.NewVendorClient(adapter.Vendor, cfg.Timeouts.Portrait, logger)
	switch strings.ToLower(adapter.Vendor) {
	case "segmind":
		return videosegmind.New(adapter.APIKey, adapter.BaseURL, client), nil
	case "evolink":
		return videoevolink.New(adapter.APIKey, adapter.BaseURL, client), nil
	case "fal":
		return videofal.New(adapter.APIKey, "", adapter.BaseURL, client), nil
	default:
		return nil, fmt.Errorf("wire: unsupported live video vendor %q", adapter.Vendor)
	}
}

func liveSTT(cfg config.Config, logger *slog.Logger) (ports.STT, error) {
	adapter := cfg.Adapters["stt"]
	if adapter.Vendor != "elevenlabs" {
		return nil, fmt.Errorf("wire: unsupported live STT vendor %q", adapter.Vendor)
	}
	return stteleven.New(adapter.APIKey, adapter.BaseURL, cfg.Timeouts.TTS, logger), nil
}

func liveTTS(cfg config.Config, logger *slog.Logger) (ports.TTS, error) {
	adapter := cfg.Adapters["tts"]
	switch strings.ToLower(adapter.Vendor) {
	case "elevenlabs":
		return ttseleven.New(adapter.APIKey, adapter.BaseURL, logger), nil
	case "openai":
		return ttsopenai.New(adapter.APIKey, adapter.BaseURL, cfg.Timeouts.TTS, logger), nil
	default:
		return nil, fmt.Errorf("wire: unsupported live TTS vendor %q", adapter.Vendor)
	}
}

func liveSound(cfg config.Config, logger *slog.Logger) (ports.SoundGen, error) {
	adapter := cfg.Adapters["sound"]
	if !strings.EqualFold(adapter.Vendor, "elevenlabs") {
		return nil, fmt.Errorf("wire: unsupported live sound vendor %q", adapter.Vendor)
	}
	client := httpx.NewVendorClient(adapter.Vendor, cfg.Timeouts.TTS, logger)
	return soundeleven.New(adapter.APIKey, adapter.BaseURL, client, logger), nil
}

type fakeSound struct{}

func (fakeSound) Generate(ctx context.Context, request ports.SoundRequest) (ports.Sound, error) {
	if err := ctx.Err(); err != nil {
		return ports.Sound{}, err
	}
	if request.Seconds <= 0 {
		return ports.Sound{}, errors.New("wire: fake sound duration must be positive")
	}
	return ports.Sound{Bytes: []byte("fake sound"), MIME: "audio/mpeg", DurationMS: int(request.Seconds * 1000)}, nil
}

func vendorConfig(cfg config.Config, vendor string) (string, string) {
	for _, adapter := range cfg.Adapters {
		if strings.EqualFold(adapter.Vendor, vendor) {
			return adapter.APIKey, adapter.BaseURL
		}
	}
	return "", ""
}
