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
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/api/debug"
	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game"
	"github.com/monstercameron/DungeonFlux/internal/logx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/runtime"
	"github.com/monstercameron/DungeonFlux/internal/store/sqlite"
	voicein "github.com/monstercameron/DungeonFlux/internal/voice/in"
	"google.golang.org/grpc"
)

// App is the assembled server and its owned resources.
type App struct {
	handler   http.Handler
	room      *runtime.Room
	roomDone  chan error
	roomStop  context.CancelFunc
	store     *sqlite.Store
	logFile   io.Closer
	logger    *slog.Logger
	watch     *api.WatchHub
	runID     domain.RunID
	debugStop context.CancelFunc
}

// Build assembles an application from cfg. A non-empty seed is expanded into
// the deterministic rehearsal seed; an empty seed uses the system CSPRNG.
func Build(ctx context.Context, cfg config.Config, seed []byte) (*App, error) {
	return BuildWithWriter(ctx, cfg, seed, os.Stdout)
}

// BuildWithWriter assembles an application and writes its tester URLs to out.
// A nil writer suppresses the console presentation while retaining urls.txt.
func BuildWithWriter(ctx context.Context, cfg config.Config, seed []byte, out io.Writer) (*App, error) {
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
	manifestPath := filepath.Join("artifacts", "runtime", "buildtime", "manifest.json")
	manifest, err := LoadManifest(manifestPath, logger)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: load build-time manifest: %w", err)
	}
	assetCatalog, err := loadAssetCatalog(ctx, cfg.Server.DataDir, manifestPath, sqlite.NewAssets(store), logger)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: load asset store: %w", err)
	}
	oneShot := manifest.OneShot.OneShot
	roomState, err := runtime.NewRoomState(nil)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create room state: %w", err)
	}
	roomID := cfg.Server.RoomCode
	if roomID == "" {
		roomID = "default"
	}
	hostToken, err := tokenOrGenerate(cfg.Server.HostToken)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create host token: %w", err)
	}
	dmToken, err := tokenOrGenerate(cfg.Server.DMToken)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create dm token: %w", err)
	}
	urls, err := buildURLs(cfg.Server.Port, roomID, dmToken, hostToken)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, err
	}
	if err := writeURLs(cfg.Server.DataDir, urls); err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, err
	}
	joinURL := preferredLANJoinURL(urls, cfg.Server.Port, roomID)
	qrURL, err := writeJoinQR(cfg.Server.DataDir, joinURL)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, err
	}
	logger.Info("room ready", "room_code", roomID, "join_url", joinURL, "join_qr", qrURL)
	logger.Info("tester URLs ready", "url_count", len(urls))
	printURLs(out, urls)
	lobbyProjection := api.LobbyProjection{RoomCode: roomID, JoinURL: joinURL, QRURL: qrURL}
	lobbyOption := game.Lobby{RoomCode: roomID, JoinURL: joinURL, QRAsset: domain.AssetID(qrURL)}
	gameOptions := []game.Option{game.WithLobby(lobbyOption), game.WithTurnTimers(cfg.Features.TurnTimers), game.WithCombatMoveUI(cfg.Features.CombatMoveUI)}
	// server.debug enables the live dfctl control events (goto, seat, timer,
	// combat); the engine rejects them otherwise (ENG-033).
	billboards := billboardHubFor(cfg, manifest)
	eng := newBillboardEngine(newLobbyEngine(game.NewWithDebug(oneShot, seed, cfg.Server.Debug, cfg.DebugStart, gameOptions...), lobbyProjection), billboards)
	roomEngine, err := newSynchronizedEngine(eng)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, err
	}
	run, err := domainRun(roomID, seed)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create run: %w", err)
	}
	if err := store.Start(ctx, run); err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: start run: %w", err)
	}
	watch := api.NewWatchHub()
	listen := api.NewListenHub()
	listen.SetLogger(logger)
	assembler := voicein.NewAssembler()
	runner, inbox, err := newExecutors(configForWire{config: cfg, logger: logger, recordings: sqlite.NewRecordings(store), cache: sqlite.NewCache(store), billboards: billboards, assembler: assembler}, listen)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create executors: %w", err)
	}
	audioRouter, err := newAudioRouter(listen)
	if err != nil {
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create audio router: %w", err)
	}
	registerAudioExecutor(runner, audioRouter, assetCatalog)
	room := runtime.NewRoom(roomEngine, clock.Real{}, store, logger, watch.Publish,
		runtime.WithRunner(runner), runtime.WithRoomState(roomState),
		runtime.WithTurnTimersEnabled(cfg.Features.TurnTimers),
		runtime.WithNewGame(func(runSeed []byte) ports.Engine {
			roomEngine.replace(newBillboardEngine(newLobbyEngine(game.NewWithDebug(oneShot, runSeed, cfg.Server.Debug, "", gameOptions...), lobbyProjection), billboards))
			return roomEngine
		}))
	inbox.room = room
	roomCtx, cancel := context.WithCancel(context.Background())
	roomDone := make(chan error, 1)
	go func() { roomDone <- room.Run(roomCtx) }()

	grpcServer := grpc.NewServer()
	assetServer, err := api.NewAssetServer(assetCatalog, filepath.Join(cfg.Server.DataDir, "assets"))
	if err != nil {
		cancel()
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create asset server: %w", err)
	}
	df.RegisterAssetServiceServer(grpcServer, assetServer)
	report, err := api.NewReportServer(room)
	if err != nil {
		cancel()
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create report server: %w", err)
	}
	session, err := api.NewSessionServer(room, roomID, hostToken, dmToken)
	if err != nil {
		cancel()
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create session server: %w", err)
	}
	df.RegisterSessionServiceServer(grpcServer, &sessionService{SessionServer: session, ReportServer: report,
		watch: watch, room: roomID, dm: dmToken, host: hostToken})
	host, err := api.NewHostServer(room, hostToken)
	if err != nil {
		cancel()
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create host server: %w", err)
	}
	host.SetRoomLocales(session)
	df.RegisterHostServiceServer(grpcServer, host)
	talk, err := api.NewTalkServer(room, session, assemblerTalkSink{assembler: assembler})
	if err != nil {
		cancel()
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: create talk server: %w", err)
	}
	talk.SetLogger(logger)
	df.RegisterVoiceServiceServer(grpcServer, talk)
	df.RegisterAudioServiceServer(grpcServer, &audioStreamService{hub: listen, sessions: session})
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
	webCfg := cfg
	webCfg.Server.DMToken, webCfg.Server.HostToken = dmToken, hostToken
	if err := mountWeb(mux, webCfg); err != nil {
		cancel()
		_ = store.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("wire: mount web: %w", err)
	}
	app := &App{handler: mux, room: room, roomDone: roomDone, roomStop: cancel,
		store: store, logFile: logFile, logger: logger, watch: watch, runID: run.ID}
	if cfg.Server.Debug {
		if err := startDebug(ctx, app, roomEngine, inbox, cfg.Server.Port+1000, cfg.Server.DataDir); err != nil {
			_ = app.Close()
			return nil, err
		}
	}
	return app, nil
}

// lobbyEngine keeps room metadata on every projected engine view while the
// shared domain View contract still has no room-level Lobby field.
type lobbyEngine struct {
	ports.Engine
	lobby api.LobbyProjection
}

func newLobbyEngine(engine ports.Engine, lobby api.LobbyProjection) ports.Engine {
	return &lobbyEngine{Engine: engine, lobby: lobby}
}

func (e *lobbyEngine) View() domain.View {
	if e == nil || e.Engine == nil {
		return domain.View{}
	}
	return api.AttachLobbyMetadata(e.Engine.View(), e.lobby)
}

func startDebug(ctx context.Context, app *App, eng ports.Engine, inbox ports.Inbox, port int, dataDir string) error {
	token, generatedPath, err := resolveDebugToken(dataDir, os.Getenv)
	if err != nil {
		return err
	}
	if generatedPath != "" {
		app.logger.Info("debug token generated", "path", generatedPath)
	}
	service, err := debug.NewServer(eng, inbox)
	if err != nil {
		return fmt.Errorf("wire: create debug service: %w", err)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("wire: listen debug port: %w", err)
	}
	debugCtx, stop := context.WithCancel(ctx)
	app.debugStop = stop
	go func() {
		if err := debug.Serve(debugCtx, listener, service, token); err != nil && !errors.Is(err, context.Canceled) {
			app.logger.Error("debug listener stopped", "err", err)
		}
	}()
	return nil
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
	if a.debugStop != nil {
		a.debugStop()
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

func tokenOrGenerate(value string) (string, error) {
	if value != "" {
		return value, nil
	}
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func domainRun(room string, seed []byte) (domain.Run, error) {
	runID, err := newRunID(seed)
	if err != nil {
		return domain.Run{}, err
	}
	return domain.Run{ID: runID, Room: domain.RoomID(room), Seed: append([]byte(nil), seed...), Mode: "demo"}, nil
}

func newRunID(seed []byte) (domain.RunID, error) {
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate run ID nonce: %w", err)
	}
	digest := sha256.Sum256(seed)
	id := fmt.Sprintf("run-%020d-%x-%x", time.Now().UTC().UnixNano(), digest[:8], nonce)
	return domain.RunID(id), nil
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
