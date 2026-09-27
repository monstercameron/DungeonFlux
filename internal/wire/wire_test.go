package wire

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
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

func TestBuild_TwoStartsShareDataDirWithUniqueRunIDs(t *testing.T) {
	cfg := testConfig(t)
	first, err := Build(context.Background(), cfg, []byte("rehearsal"))
	if err != nil {
		t.Fatalf("first Build() error = %v", err)
	}
	firstID := first.runID
	if err := first.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}

	second, err := Build(context.Background(), cfg, []byte("rehearsal"))
	if err != nil {
		t.Fatalf("second Build() error = %v", err)
	}
	defer func() { _ = second.Close() }()
	if second.runID == firstID {
		t.Fatalf("second run ID = %q, reused first run ID", firstID)
	}
}

func TestBuild_ResetPublishesLobbyFromFreshRun(t *testing.T) {
	app, err := Build(context.Background(), testConfig(t), []byte("rehearsal"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() { _ = app.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub := app.watch.Subscribe(ctx, df.ClientKind_CLIENT_KIND_DM, 0)
	defer sub.Close()
	ack := make(chan domain.Ack, 1)
	if !app.room.Post(ctx, domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostReset}, Reply: ack}) {
		t.Fatal("reset was not accepted by room")
	}
	if response := <-ack; !response.Accepted {
		t.Fatalf("reset ack = %+v", response)
	}
	select {
	case message := <-sub.Messages():
		if got := message.GetState().GetPhase(); got != string(vocab.StateLobby) {
			t.Fatalf("reset phase = %q, want lobby", got)
		}
	case <-ctx.Done():
		t.Fatal("reset view was not published")
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
