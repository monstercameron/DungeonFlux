package wire

import (
	"io"
	"log/slog"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
)

func TestBuildAdapters_liveLLM(t *testing.T) {
	demoChain := []string{"openai:gpt-6-luna/none", "anthropic:claude-haiku-4-5", "recording:npc_reply"}
	tests := []struct {
		name     string
		adapters map[string]config.AdapterConfig
		chain    []string
		wantLive bool
		wantErr  bool
	}{
		{name: "per-vendor names turn live text on and skip recording and unkeyed vendors",
			adapters: map[string]config.AdapterConfig{"llm_openai": {Vendor: "openai", Mode: "live", APIKey: "k"}},
			chain:    demoChain, wantLive: true},
		{name: "a single llm adapter still turns live text on",
			adapters: map[string]config.AdapterConfig{"llm": {Vendor: "openai", Mode: "live", APIKey: "k"}},
			chain:    []string{"openai:gpt-6-luna"}, wantLive: true},
		{name: "fake llm adapters keep the fake",
			adapters: map[string]config.AdapterConfig{"llm": {Vendor: "local", Mode: "fake"}},
			chain:    []string{"local:fake"}},
		{name: "a live chain with no callable vendor fails start-up",
			adapters: map[string]config.AdapterConfig{"llm_openai": {Vendor: "openai", Mode: "live", APIKey: "k"}},
			chain:    []string{"recording:npc_reply"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{Adapters: tc.adapters, Models: config.ModelConfig{Chains: map[string][]string{"npc_reply": tc.chain}}}
			set, err := buildAdapters(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if (err != nil) != tc.wantErr {
				t.Fatalf("buildAdapters() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if _, fake := set.llm.(fakeLLM); fake == tc.wantLive {
				t.Fatalf("llm is fake = %v, want live = %v (%T)", fake, tc.wantLive, set.llm)
			}
		})
	}
}
