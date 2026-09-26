// Command server starts the DungeonFlux HTTP and gRPC-over-WebSocket server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/wire"
)

func main() {
	configPath := flag.String("config", "config/fake.json", "configuration file")
	port := flag.Int("port", 0, "HTTP port override")
	dataDir := flag.String("data-dir", "", "runtime data directory override")
	seedText := flag.String("seed", "", "hex rehearsal seed")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fatal(err)
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *dataDir != "" {
		cfg.Server.DataDir = *dataDir
	}
	seed, err := wire.ParseSeed(*seedText)
	if err != nil {
		fatal(err)
	}
	app, err := wire.Build(context.Background(), cfg, seed)
	if err != nil {
		fatal(err)
	}
	defer app.Close()

	server := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Server.Port), Handler: app.Handler()}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-stopCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fatal(err)
		}
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			fatal(err)
		}
	}
}

const shutdownTimeout = 5 * time.Second

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
