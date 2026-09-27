package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var readVerbs = map[string]bool{
	"state": true, "view": true, "legal": true, "scopes": true,
	"assets": true, "events": true, "logs": true, "clients": true, "costs": true,
}

type readCommand func(context.Context, dungeonfluxv1.DebugServiceClient) (readResult, error)

type readResult interface {
	readResult()
}

type unaryResult struct{ message proto.Message }

func (unaryResult) readResult() {}

func runRead(ctx context.Context, args []string, stdout, stderr io.Writer, dial connectFunc) (int, bool) {
	verb := findReadVerb(args)
	if verb == "" {
		return 0, false
	}
	opts, command, err := parseReadOptions(args, verb, stderr)
	if err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	client, closer, err := dial(ctx, opts.address)
	if err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	defer closer.Close()
	result, err := command(withDebugToken(ctx, opts.token), client)
	if err != nil {
		writeError(stderr, readError(err))
		return exitTransport, true
	}
	if stream, ok := result.(readStream); ok {
		return receiveReadStream(stream, stdout, opts.pretty, stderr), true
	}
	message := result.(unaryResult).message
	if err := writeMessage(stdout, message, opts.pretty); err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	return exitOK, true
}

type readStream interface {
	readResult
	receive(io.Writer, bool) error
}

type eventStream struct {
	stream dungeonfluxv1.DebugService_EventsClient
}

func (eventStream) readResult() {}
func (s eventStream) receive(dst io.Writer, pretty bool) error {
	for {
		message, err := s.stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := writeMessage(dst, message, pretty); err != nil {
			return err
		}
	}
}

type logStream struct {
	stream dungeonfluxv1.DebugService_LogsClient
}

func (logStream) readResult() {}
func (s logStream) receive(dst io.Writer, pretty bool) error {
	for {
		message, err := s.stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := writeMessage(dst, message, pretty); err != nil {
			return err
		}
	}
}

func receiveReadStream(stream readStream, stdout io.Writer, pretty bool, stderr io.Writer) int {
	if err := stream.receive(stdout, pretty); err != nil {
		writeError(stderr, readError(err))
		return exitTransport
	}
	return exitOK
}

func readError(err error) error {
	if status.Code(err) == codes.Unimplemented {
		return errors.New("not supported by server")
	}
	return err
}

func findReadVerb(args []string) string {
	for _, arg := range args {
		if readVerbs[arg] {
			return arg
		}
	}
	return ""
}

func parseReadOptions(args []string, verb string, stderr io.Writer) (options, readCommand, error) {
	prefix, suffix := splitReadArgs(args, verb)
	opts := options{address: defaultAddress, token: envDebugToken()}
	flags := flag.NewFlagSet("dfctl", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.address, "addr", opts.address, "debug gRPC address")
	flags.StringVar(&opts.room, "room", "", "room code")
	flags.StringVar(&opts.token, "token", opts.token, "debug token")
	flags.BoolVar(&opts.pretty, "pretty", false, "indent JSON for people")
	if err := flags.Parse(prefix); err != nil {
		return options{}, nil, err
	}
	command, err := makeReadCommand(verb, opts.room, suffix)
	return opts, command, err
}

func splitReadArgs(args []string, verb string) ([]string, []string) {
	for i, arg := range args {
		if arg == verb {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func makeReadCommand(verb, room string, args []string) (readCommand, error) {
	switch verb {
	case "state":
		if len(args) != 0 {
			return nil, errors.New("usage: state")
		}
		return unaryRead(func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
			return c.State(ctx, &dungeonfluxv1.DebugRoom{Room: room})
		})
	case "view":
		return makeViewCommand(room, args)
	case "legal":
		if len(args) != 2 || args[0] != "--seat" {
			return nil, errors.New("usage: legal --seat N")
		}
		return unaryRead(func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
			return c.Legal(ctx, &dungeonfluxv1.SeatRequest{Room: room, Seat: args[1]})
		})
	case "scopes":
		if len(args) != 0 {
			return nil, errors.New("usage: scopes")
		}
		return unaryRead(func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
			return c.Scopes(ctx, &dungeonfluxv1.DebugRoom{Room: room})
		})
	case "assets":
		if len(args) != 0 {
			return nil, errors.New("usage: assets")
		}
		return unaryRead(func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
			return c.Assets(ctx, &dungeonfluxv1.DebugRoom{Room: room})
		})
	case "clients":
		if len(args) != 0 {
			return nil, errors.New("usage: clients")
		}
		return unaryRead(func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
			return c.Clients(ctx, &dungeonfluxv1.DebugRoom{Room: room})
		})
	case "costs":
		if len(args) != 0 {
			return nil, errors.New("usage: costs")
		}
		return unaryRead(func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
			return c.Costs(ctx, &dungeonfluxv1.DebugRoom{Room: room})
		})
	case "events":
		return makeEventsCommand(room, args)
	case "logs":
		return makeLogsCommand(room, args)
	default:
		return nil, fmt.Errorf("unknown read verb %q", verb)
	}
}

func unaryRead(call func(context.Context, dungeonfluxv1.DebugServiceClient) (proto.Message, error)) (readCommand, error) {
	return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (readResult, error) {
		message, err := call(ctx, client)
		if err != nil {
			return nil, err
		}
		return unaryResult{message: message}, nil
	}, nil
}

func makeViewCommand(room string, args []string) (readCommand, error) {
	if len(args) < 1 || (args[0] != "--seat" && args[0] != "--dm") {
		return nil, errors.New("usage: view --seat N | view --dm")
	}
	seat, view := "", ""
	if args[0] == "--seat" {
		if len(args) != 2 {
			return nil, errors.New("usage: view --seat N | view --dm")
		}
		seat = args[1]
	} else {
		if len(args) != 1 {
			return nil, errors.New("usage: view --dm")
		}
		view = "dm"
	}
	return unaryRead(func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
		return c.View(ctx, &dungeonfluxv1.ViewRequest{Room: room, Seat: seat, View: view})
	})
}

func makeEventsCommand(room string, args []string) (readCommand, error) {
	since, follow, err := parseStreamFlags(args, "events")
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (readResult, error) {
		stream, err := c.Events(ctx, &dungeonfluxv1.EventsRequest{Room: room, Since: since, Follow: follow})
		return eventStream{stream: stream}, err
	}, nil
}

func makeLogsCommand(room string, args []string) (readCommand, error) {
	level, trace, follow, err := parseLogFlags(args)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, c dungeonfluxv1.DebugServiceClient) (readResult, error) {
		stream, err := c.Logs(ctx, &dungeonfluxv1.LogsRequest{Room: room, Level: level, TraceId: trace, Follow: follow})
		return logStream{stream: stream}, err
	}, nil
}

func parseStreamFlags(args []string, verb string) (uint64, bool, error) {
	since, follow := uint64(0), false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--since":
			if i+1 >= len(args) {
				return 0, false, fmt.Errorf("usage: %s --since SEQ [--follow]", verb)
			}
			value, err := strconv.ParseUint(args[i+1], 10, 64)
			if err != nil {
				return 0, false, errors.New("since must be a non-negative integer")
			}
			since = value
			i++
		case "--follow":
			follow = true
		default:
			return 0, false, fmt.Errorf("usage: %s --since SEQ [--follow]", verb)
		}
	}
	return since, follow, nil
}

func parseLogFlags(args []string) (string, string, bool, error) {
	level, trace, follow := "", "", false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--level", "--trace-id":
			if i+1 >= len(args) {
				return "", "", false, errors.New("usage: logs [--level LEVEL] [--trace-id ID] [--follow]")
			}
			if args[i] == "--level" {
				level = args[i+1]
			} else {
				trace = args[i+1]
			}
			i++
		case "--follow":
			follow = true
		default:
			return "", "", false, errors.New("usage: logs [--level LEVEL] [--trace-id ID] [--follow]")
		}
	}
	return level, trace, follow, nil
}
