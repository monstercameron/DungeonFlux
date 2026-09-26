package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad_validatesCommittedConfigs(t *testing.T) {
	for _, name := range []string{"fake.json", "demo.json"} {
		t.Run(name, func(t *testing.T) {
			cfg, err := Load(filepath.Join("..", "..", "config", name))
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.Server.Port == 0 || len(cfg.Adapters) == 0 || len(cfg.Models.Chains) == 0 {
				t.Fatal("loaded config is incomplete")
			}
		})
	}
}

func TestLoad_rejectsUnknownFields(t *testing.T) {
	path := writeConfig(t, `{"server":{},"unknown":true}`)
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load() error = %v, want unknown-field error", err)
	}
}

func TestLoad_appliesSecretFromEnvironment(t *testing.T) {
	t.Setenv("DF_OPENAI_API_KEY", "test-secret")
	path := writeConfig(t, validJSON("openai"))
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Adapters["llm"].APIKey; got != "test-secret" {
		t.Fatalf("APIKey = %q, want environment value", got)
	}
}

func TestLoad_rejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name string
		json string
		want string
	}{
		{"bad port", strings.Replace(validJSON("local"), `"port":18101`, `"port":0`, 1), "server.port"},
		{"bad timeout", strings.Replace(validJSON("local"), `"interpret":"2.5s"`, `"interpret":"0s"`, 1), "timeouts"},
		{"bad mode", strings.Replace(validJSON("local"), `"mode":"fake"`, `"mode":"other"`, 1), "mode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Load(writeConfig(t, tc.json)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Load() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidate_rejectsBudgetAndNav(t *testing.T) {
	cfg := Config{Server: ServerConfig{Port: 1, LogLevel: "info", DataDir: "x"}, Timeouts: TimeoutConfig{
		CharacterFlavor: time.Second, Interpret: time.Second, SpokenFirstToken: time.Second,
		Prerender: time.Second, Portrait: time.Second, TTS: time.Second,
	}, Battlefield: BattlefieldConfig{NavPath: "nav"}, Budget: BudgetConfig{PerRunUSD: 2, HardUSD: 1}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("Validate() error = %v, want budget error", err)
	}
	cfg.Budget = BudgetConfig{}
	cfg.Battlefield.NavPath = ""
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "nav_path") {
		t.Fatalf("Validate() error = %v, want nav error", err)
	}
}

func writeConfig(t *testing.T, value string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func validJSON(vendor string) string {
	return `{"server":{"port":18101,"debug":true,"log_level":"info","data_dir":"artifacts/runtime/test"},"adapters":{"llm":{"vendor":"` + vendor + `","mode":"fake"}},"models":{"chains":{"opening":["llm"]}},"timeouts":{"character_flavor":"10s","interpret":"2.5s","spoken_first_token":"3s","prerender":"20s","portrait":"22s","tts":"5s"},"features":{},"battlefield":{"nav_path":"internal/content/battlefield_tavern.json"},"budget":{"per_run_usd":0.62,"hard_usd":8},"debug_start":""}`
}
