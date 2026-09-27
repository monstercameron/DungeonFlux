// Command server starts the DungeonFlux HTTP and gRPC-over-WebSocket server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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

	server := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Server.Port), Handler: app.Handler()}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-stopCtx.Done():
		if err := shutdown(server, app, shutdownTimeout, os.Stderr); err != nil {
			fatal(err)
		}
	case err := <-serveErr:
		closeErr := app.Close()
		if !errors.Is(err, http.ErrServerClosed) {
			fatal(errors.Join(err, closeErr))
		}
	}
}

// shutdown drains server for up to timeout, then force-closes connections
// still busy (for example a phone mid-download of the WASM bundle or a clip),
// and always closes app so queued events are flushed to the store. A forced
// close after the grace period is expected on a live table, so it is reported
// on stderr rather than returned. app is closed exactly once.
func shutdown(server *http.Server, app io.Closer, timeout time.Duration, stderr io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var closeErr error
	if err := server.Shutdown(ctx); err != nil {
		_, _ = fmt.Fprintf(stderr, "shutdown: %v after %s; closing remaining connections\n", err, timeout)
		closeErr = server.Close()
	}
	return errors.Join(closeErr, app.Close())
}

const shutdownTimeout = 5 * time.Second

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
