package main

import (
	"log/slog"
	"net/http"
	"os"

	spikev1 "github.com/monstercameron/DungeonFlux/scripts/spike/gen"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	grpcServer := grpc.NewServer()
	spikev1.RegisterEchoServer(grpcServer, echoServer{})

	mux := http.NewServeMux()
	mux.Handle("/grpc", grpctunnel.Wrap(grpcServer))
	logger.Info("audio tunnel spike listening", "addr", ":18120")
	if err := http.ListenAndServe(":18120", mux); err != nil {
		logger.Error("audio tunnel spike stopped", "err", err)
	}
}
