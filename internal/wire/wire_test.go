package wire

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
)

func TestBuild_HealthAndClose(t *testing.T) {
	cfg := testConfig(t)
	app, err := Build(context.Background(), cfg, []byte("rehearsal"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	app.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK || strings.TrimSpace(res.Body.String()) != "ok" {
		t.Fatalf("health = %d %q", res.Code, res.Body.String())
	}
	if err := app.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestParseSeed_HexAndEmpty(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "hex", in: "00ff", want: "00ff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSeed(tc.in)
			if err != nil {
				t.Fatalf("ParseSeed() error = %v", err)
			}
			if hex.EncodeToString(got) != tc.want {
				t.Fatalf("ParseSeed() = %x, want %s", got, tc.want)
			}
		})
	}
	if _, err := ParseSeed("not-hex"); err == nil {
		t.Fatal("ParseSeed() accepted malformed hex")
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{Server: config.ServerConfig{
		Port: 18199, LogLevel: "debug", DataDir: filepath.Join(t.TempDir(), "runtime"),
	}, Budget: config.BudgetConfig{PerRunUSD: 1, HardUSD: 2},
		Timeouts:    config.TimeoutConfig{CharacterFlavor: 1, Interpret: 1, SpokenFirstToken: 1, Prerender: 1, Portrait: 1, TTS: 1},
		Battlefield: config.BattlefieldConfig{NavPath: "internal/content/battlefield_tavern.json"}}
}
