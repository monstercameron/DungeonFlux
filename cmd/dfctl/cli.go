package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	defaultAddress = "127.0.0.1:19101"
	exitOK         = 0
	exitRejected   = 1
	exitTransport  = 2
)

type options struct {
	address string
	room    string
	token   string
	pretty  bool
}

type connectFunc func(context.Context, string) (dungeonfluxv1.DebugServiceClient, io.Closer, error)

func connect(_ context.Context, address string) (dungeonfluxv1.DebugServiceClient, io.Closer, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return dungeonfluxv1.NewDebugServiceClient(conn), conn, nil
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, dial connectFunc) int {
	opts, verb, err := parseOptions(args, stderr)
	if err != nil {
		return exitTransport
	}
	if verb != "state" {
		writeError(stderr, fmt.Errorf("unknown or missing verb %q (available: state)", verb))
		return exitTransport
	}

	client, closer, err := dial(ctx, opts.address)
	if err != nil {
		writeError(stderr, err)
		return exitTransport
	}
	defer closer.Close()

	callCtx := metadata.AppendToOutgoingContext(ctx, "x-df-debug-token", opts.token)
	response, err := client.State(callCtx, &dungeonfluxv1.DebugRoom{Room: opts.room})
	if err != nil {
		writeError(stderr, err)
		return exitTransport
	}
	if err := writeMessage(stdout, response, opts.pretty); err != nil {
		writeError(stderr, err)
		return exitTransport
	}
	if rejected(response) {
		return exitRejected
	}
	return exitOK
}

func parseOptions(args []string, stderr io.Writer) (options, string, error) {
	opts := options{address: defaultAddress, token: os.Getenv("DF_DEBUG_TOKEN")}
	flags := flag.NewFlagSet("dfctl", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.address, "addr", opts.address, "debug gRPC address")
	flags.StringVar(&opts.room, "room", "", "room code")
	flags.StringVar(&opts.token, "token", opts.token, "debug token (defaults to DF_DEBUG_TOKEN)")
	flags.BoolVar(&opts.pretty, "pretty", false, "indent JSON for people")
	if err := flags.Parse(args); err != nil {
		return options{}, "", err
	}
	remaining := flags.Args()
	if len(remaining) != 1 {
		return options{}, "", errors.New("exactly one verb is required")
	}
	return opts, remaining[0], nil
}

func writeMessage(dst io.Writer, message proto.Message, pretty bool) error {
	marshal := protojson.MarshalOptions{UseProtoNames: true}
	if pretty {
		marshal.Indent = "  "
	}
	data, err := marshal.Marshal(message)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(dst, "%s\n", data)
	return err
}

func writeError(dst io.Writer, err error) {
	data, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		_, _ = fmt.Fprintf(dst, "{\"error\":%q}\n", err.Error())
		return
	}
	_, _ = fmt.Fprintf(dst, "%s\n", data)
}

func rejected(message proto.Message) bool {
	switch value := message.(type) {
	case *dungeonfluxv1.SendResponse:
		return !value.GetAccepted()
	case *dungeonfluxv1.SnapshotResponse:
		return !value.GetOk()
	case *dungeonfluxv1.VendorResponse:
		return !value.GetOk()
	case *dungeonfluxv1.ClientResponse:
		return !value.GetOk()
	default:
		return false
	}
}
