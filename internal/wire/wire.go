package wire

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game"
	"github.com/monstercameron/DungeonFlux/internal/logx"
	"github.com/monstercameron/DungeonFlux/internal/runtime"
	"github.com/monstercameron/DungeonFlux/internal/store/sqlite"
	"google.golang.org/grpc"
)

// App is the assembled server and its owned resources.
type App struct {
	handler  http.Handler
	room     *runtime.Room
	roomDone chan error
	roomStop context.CancelFunc
	store    *sqlite.Store
	logFile  io.Closer
	logger   *slog.Logger
}

// Build assembles an application from cfg. A non-empty seed is expanded into
// the deterministic rehearsal seed; an empty seed uses the system CSPRNG.
func Build(ctx context.Context, cfg config.Config, seed []byte) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("wire: validate config: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := os.MkdirAll(filepath.Join(cfg.Server.DataDir, "logs"), 0o700); err != nil {
		return nil, fmt.Errorf("wire: create data directory: %w", err)
	}
	logger, logFile, err := newLogger(cfg)
	if err != nil {
		return nil, err
	}
	store, err := sqlite.Open(ctx, filepath.Join(cfg.Server.DataDir, "dungeonflux.db"), logger)
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: open store: %w", err)
	}
	seed, err = makeSeed(seed)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: make seed: %w", err)
	}
	eng := game.New(content.DefaultOneShot().OneShot, seed)
	roomID := cfg.Server.RoomCode
	if roomID == "" {
		roomID = "default"
	}
	run := domainRun(roomID, seed)
	if err := store.Start(ctx, run); err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: start run: %w", err)
	}
	room := runtime.NewRoom(eng, clock.Real{}, store, logger, nil)
	roomCtx, cancel := context.WithCancel(context.Background())
	roomDone := make(chan error, 1)
	go func() { roomDone <- room.Run(roomCtx) }()

	grpcServer := grpc.NewServer()
	report, err := api.NewReportServer(room)
	if err != nil {
		cancel()
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create report server: %w", err)
	}
	df.RegisterSessionServiceServer(grpcServer, report)
	apiServer, err := api.NewServer(grpcServer, cfg.Server.AllowedOrigins)
	if err != nil {
		cancel()
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create api server: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health)
	mux.Handle("/grpc", apiServer.Handler())
	return &App{handler: mux, room: room, roomDone: roomDone, roomStop: cancel,
		store: store, logFile: logFile, logger: logger}, nil
}

// Handler returns the HTTP handler for the assembled application.
func (a *App) Handler() http.Handler {
	if a == nil || a.handler == nil {
		return http.NotFoundHandler()
	}
	return a.handler
}

// Close stops the room and releases storage and logging resources.
func (a *App) Close() error {
	if a == nil {
		return nil
	}
	if a.roomStop != nil {
		a.roomStop()
	}
	var errs []error
	if a.roomDone != nil {
		if err := <-a.roomDone; err != nil && !errors.Is(err, context.Canceled) {
			errs = append(errs, err)
		}
	}
	if a.store != nil {
		if err := a.store.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if a.logFile != nil {
		if err := a.logFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

func makeSeed(input []byte) ([]byte, error) {
	if len(input) == 0 {
		seed := make([]byte, 32)
		if _, err := rand.Read(seed); err != nil {
			return nil, err
		}
		return seed, nil
	}
	h := sha256.New()
	_, _ = h.Write(input)
	var zero [4]byte
	_, _ = h.Write(zero[:])
	return h.Sum(nil), nil
}

func domainRun(room string, seed []byte) domain.Run {
	return domain.Run{ID: domain.RunID("run-0"), Room: domain.RoomID(room), Seed: append([]byte(nil), seed...), Mode: "demo"}
}

func newLogger(cfg config.Config) (*slog.Logger, io.Closer, error) {
	level := parseLevel(cfg.Server.LogLevel)
	path := filepath.Join(cfg.Server.DataDir, "logs", "server.jsonl")
	jsonHandler, file, err := logx.NewJSONLHandler(path, level)
	if err != nil {
		return nil, nil, fmt.Errorf("wire: create log: %w", err)
	}
	return slog.New(logx.NewRedactingHandler(jsonHandler)), file, nil
}

func parseLevel(value string) slog.Level {
	switch value {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ParseSeed decodes the optional command-line rehearsal seed.
func ParseSeed(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	seed, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode seed: %w", err)
	}
	if len(seed) == 0 {
		return nil, errors.New("decode seed: empty value")
	}
	return seed, nil
}
