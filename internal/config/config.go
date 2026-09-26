package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Config is the complete validated configuration for one DungeonFlux process.
type Config struct {
	Server      ServerConfig             `json:"server"`
	Adapters    map[string]AdapterConfig `json:"adapters"`
	Models      ModelConfig              `json:"models"`
	Timeouts    TimeoutConfig            `json:"timeouts"`
	Features    FeatureConfig            `json:"features"`
	Battlefield BattlefieldConfig        `json:"battlefield"`
	Budget      BudgetConfig             `json:"budget"`
	DebugStart  string                   `json:"debug_start"`
}

// ServerConfig controls listeners, storage, room identity, and logging.
type ServerConfig struct {
	Port           int      `json:"port"`
	Debug          bool     `json:"debug"`
	LogLevel       string   `json:"log_level"`
	DataDir        string   `json:"data_dir"`
	RoomCode       string   `json:"room_code"`
	HostToken      string   `json:"host_token"`
	DMToken        string   `json:"dm_token"`
	AllowedOrigins []string `json:"allowed_origins"`
}

// AdapterConfig identifies one vendor adapter and its optional endpoint.
type AdapterConfig struct {
	Vendor  string `json:"vendor"`
	Mode    string `json:"mode"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"-"`
}

// ModelConfig defines role chains, keyed by the role names in vocab.
type ModelConfig struct {
	Chains map[string][]string `json:"chains"`
}

// TimeoutConfig contains deadlines used by model and media executors.
type TimeoutConfig struct {
	CharacterFlavor  time.Duration `json:"character_flavor"`
	Interpret        time.Duration `json:"interpret"`
	SpokenFirstToken time.Duration `json:"spoken_first_token"`
	Prerender        time.Duration `json:"prerender"`
	Portrait         time.Duration `json:"portrait"`
	TTS              time.Duration `json:"tts"`
}

// UnmarshalJSON accepts human-readable duration strings such as "3s".
func (t *TimeoutConfig) UnmarshalJSON(data []byte) error {
	var raw struct {
		CharacterFlavor  string `json:"character_flavor"`
		Interpret        string `json:"interpret"`
		SpokenFirstToken string `json:"spoken_first_token"`
		Prerender        string `json:"prerender"`
		Portrait         string `json:"portrait"`
		TTS              string `json:"tts"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	values := []*time.Duration{&t.CharacterFlavor, &t.Interpret, &t.SpokenFirstToken, &t.Prerender, &t.Portrait, &t.TTS}
	texts := []string{raw.CharacterFlavor, raw.Interpret, raw.SpokenFirstToken, raw.Prerender, raw.Portrait, raw.TTS}
	for i, text := range texts {
		value, err := time.ParseDuration(text)
		if err != nil {
			return fmt.Errorf("timeout %q: %w", text, err)
		}
		*values[i] = value
	}
	return nil
}

// FeatureConfig controls the demo cut-order switches.
type FeatureConfig struct {
	CombatMoveUI  bool `json:"combat_move_ui"`
	TurnTimers    bool `json:"turn_timers"`
	LivePCLoops   bool `json:"live_pc_loops"`
	LiveVideo     bool `json:"live_video"`
	SequenceMode  bool `json:"sequence_mode"`
	Splat         bool `json:"splat"`
	Music         bool `json:"music"`
	BudgetEnforce bool `json:"budget_enforce"`
}

// BattlefieldConfig identifies the authored battlefield data.
type BattlefieldConfig struct {
	NavPath  string `json:"nav_path"`
	SceneURL string `json:"scene_url"`
	LiteURL  string `json:"lite_url"`
	FlatURL  string `json:"flat_url"`
}

// BudgetConfig contains the per-run and hard budget ceilings in USD.
type BudgetConfig struct {
	PerRunUSD float64 `json:"per_run_usd"`
	HardUSD   float64 `json:"hard_usd"`
}

// Load reads, strictly decodes, environment-expands secrets, and validates a
// JSON configuration file.
func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	var cfg Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Config{}, fmt.Errorf("decode config %q: multiple JSON values", path)
		}
		return Config{}, fmt.Errorf("decode config %q: trailing data: %w", path, err)
	}
	applyEnvironment(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %q: %w", path, err)
	}
	return cfg, nil
}

// Validate checks values that would otherwise produce a partially configured
// server or an unusable run.
func (c Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.Server.DataDir) == "" {
		return fmt.Errorf("server.data_dir is required")
	}
	if c.Server.LogLevel != "debug" && c.Server.LogLevel != "info" && c.Server.LogLevel != "warn" && c.Server.LogLevel != "error" {
		return fmt.Errorf("server.log_level must be debug, info, warn, or error")
	}
	if c.Timeouts.CharacterFlavor <= 0 || c.Timeouts.Interpret <= 0 || c.Timeouts.SpokenFirstToken <= 0 || c.Timeouts.Prerender <= 0 || c.Timeouts.Portrait <= 0 || c.Timeouts.TTS <= 0 {
		return fmt.Errorf("all timeouts must be positive")
	}
	if c.Budget.PerRunUSD < 0 || c.Budget.HardUSD < 0 || c.Budget.HardUSD < c.Budget.PerRunUSD {
		return fmt.Errorf("budget values must be non-negative and hard_usd must cover per_run_usd")
	}
	if strings.TrimSpace(c.Battlefield.NavPath) == "" {
		return fmt.Errorf("battlefield.nav_path is required")
	}
	for name, adapter := range c.Adapters {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(adapter.Vendor) == "" {
			return fmt.Errorf("adapter names and vendors are required")
		}
		if adapter.Mode != "fake" && adapter.Mode != "live" {
			return fmt.Errorf("adapter %q mode must be fake or live", name)
		}
	}
	for role, chain := range c.Models.Chains {
		if strings.TrimSpace(role) == "" || len(chain) == 0 {
			return fmt.Errorf("model chain %q must not be empty", role)
		}
	}
	return nil
}

func applyEnvironment(c *Config) {
	for name, adapter := range c.Adapters {
		if envName := secretEnv(adapter.Vendor); envName != "" {
			if value := os.Getenv(envName); value != "" {
				adapter.APIKey = value
				c.Adapters[name] = adapter
			}
		}
	}
}

func secretEnv(vendor string) string {
	switch vendor {
	case "openai":
		return "DF_OPENAI_API_KEY"
	case "gemini":
		return "DF_GEMINI_API_KEY"
	case "anthropic":
		return "DF_ANTHROPIC_API_KEY"
	case "elevenlabs":
		return "DF_ELEVENLABS_API_KEY"
	case "segmind":
		return "DF_SEGMIND_API_KEY"
	case "evolink":
		return "DF_EVOLINK_API_KEY"
	case "fal":
		return "DF_FAL_KEY"
	case "cerebras":
		return "DF_CEREBRAS_API_KEY"
	case "typesafe":
		return "DF_TYPESAFE_API_KEY"
	case "worldlabs":
		return "DF_WORLDLABS_API_KEY"
	default:
		return ""
	}
}
