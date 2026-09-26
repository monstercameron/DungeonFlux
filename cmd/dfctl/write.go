package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func runWrite(ctx context.Context, args []string, stdout, stderr io.Writer, dial connectFunc) (int, bool) {
	verb := writeVerb(args)
	if verb == "" {
		return 0, false
	}
	if containsArg(args, "--dry-run") {
		return runWriteDryRun(args, verb, stdout, stderr), true
	}
	opts, command, err := parseWriteOptions(args, stderr)
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
	callCtx := withDebugToken(ctx, opts.token)
	response, err := command(callCtx, client)
	if err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	if err := writeMessage(stdout, response, opts.pretty); err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	if rejected(response) {
		return exitRejected, true
	}
	return exitOK, true
}

func runWriteDryRun(args []string, verb string, stdout, stderr io.Writer) int {
	clean := make([]string, 0, len(args)-1)
	for _, arg := range args {
		if arg != "--dry-run" {
			clean = append(clean, arg)
		}
	}
	opts, _, err := parseWriteOptions(clean, stderr)
	if err != nil {
		writeError(stderr, err)
		return exitTransport
	}
	_, suffix := splitWriteArgs(clean, verb)
	if _, err := makeWriteCommand(verb, opts.room, suffix); err != nil {
		writeError(stderr, err)
		return exitTransport
	}
	request, err := previewWriteRequest(opts.room, verb, suffix)
	if err != nil {
		writeError(stderr, err)
		return exitTransport
	}
	if err := writeDryRun(stdout, request, opts.pretty); err != nil {
		writeError(stderr, err)
		return exitTransport
	}
	return exitOK
}

func previewWriteRequest(room, verb string, args []string) (proto.Message, error) {
	switch verb {
	case "send":
		payload := "{}"
		if len(args) == 2 {
			payload = args[1]
		}
		return &dungeonfluxv1.SendRequest{Room: room, Event: args[0], PayloadJson: payload}, nil
	case "act":
		seat, move := "", ""
		payload := map[string]string{}
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--seat":
				seat, i = args[i+1], i+1
			case "--arg":
				payload["arg"], i = args[i+1], i+1
			case "--target":
				payload["target_id"], i = args[i+1], i+1
			case "--cell":
				payload["cell"], i = args[i+1], i+1
			default:
				move = args[i]
			}
		}
		payload["seat"], payload["move_id"] = seat, move
		return &dungeonfluxv1.SendRequest{Room: room, Event: "act", PayloadJson: marshalJSON(payload)}, nil
	case "say":
		return &dungeonfluxv1.SendRequest{Room: room, Event: "say", PayloadJson: marshalJSON(map[string]string{"seat": args[1], "text": strings.Join(args[2:], " ")})}, nil
	case "reset":
		seed := ""
		if len(args) == 2 {
			seed = args[1]
		}
		return &dungeonfluxv1.SendRequest{Room: room, Event: "debug_reset", PayloadJson: marshalJSON(map[string]string{"seed": seed})}, nil
	case "dice":
		return &dungeonfluxv1.SendRequest{Room: room, Event: "host_force_d20", PayloadJson: marshalJSON(map[string]string{"d20": strings.TrimPrefix(args[1], "d20=")})}, nil
	default:
		return nil, fmt.Errorf("unknown write verb %q", verb)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

type writeCommand func(context.Context, dungeonfluxv1.DebugServiceClient) (*dungeonfluxv1.SendResponse, error)

func parseWriteOptions(args []string, stderr io.Writer) (options, writeCommand, error) {
	verb := writeVerb(args)
	if verb == "" {
		return options{}, nil, errors.New("unknown write verb")
	}
	global := options{address: defaultAddress}
	globalFlags := flag.NewFlagSet("dfctl", flag.ContinueOnError)
	globalFlags.SetOutput(stderr)
	globalFlags.StringVar(&global.address, "addr", global.address, "debug gRPC address")
	globalFlags.StringVar(&global.room, "room", "", "room code")
	globalFlags.StringVar(&global.token, "token", "", "debug token")
	globalFlags.BoolVar(&global.pretty, "pretty", false, "indent JSON for people")
	prefix, suffix := splitWriteArgs(args, verb)
	if err := globalFlags.Parse(prefix); err != nil {
		return options{}, nil, err
	}
	if global.token == "" {
		global.token = envDebugToken()
	}
	command, err := makeWriteCommand(verb, global.room, suffix)
	return global, command, err
}

func envDebugToken() string {
	return os.Getenv("DF_DEBUG_TOKEN")
}

func withDebugToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-df-debug-token", token)
}

func makeWriteCommand(verb, room string, args []string) (writeCommand, error) {
	switch verb {
	case "send":
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("usage: send <event> [json]")
		}
		payload := "{}"
		if len(args) == 2 {
			if !json.Valid([]byte(args[1])) {
				return nil, errors.New("payload must be valid JSON")
			}
			payload = args[1]
		}
		request := &dungeonfluxv1.SendRequest{Room: room, Event: args[0], PayloadJson: payload}
		return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (*dungeonfluxv1.SendResponse, error) {
			return client.Send(ctx, request)
		}, nil
	case "act":
		return makeActCommand(room, args)
	case "say":
		if len(args) < 3 || args[0] != "--seat" {
			return nil, errors.New("usage: say --seat N text")
		}
		request := &dungeonfluxv1.DebugSayRequest{Room: room, Seat: args[1], Text: strings.Join(args[2:], " ")}
		return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (*dungeonfluxv1.SendResponse, error) {
			return client.Say(ctx, request)
		}, nil
	case "reset":
		if len(args) != 0 && (len(args) != 2 || args[0] != "--seed") {
			return nil, errors.New("usage: reset [--seed HEX]")
		}
		seed := ""
		if len(args) == 2 {
			seed = args[1]
		}
		request := &dungeonfluxv1.ResetRequest{Room: room, Seed: seed}
		return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (*dungeonfluxv1.SendResponse, error) {
			return client.Reset(ctx, request)
		}, nil
	case "dice":
		if len(args) != 2 || args[0] != "force" || !strings.HasPrefix(args[1], "d20=") {
			return nil, errors.New("usage: dice force d20=N")
		}
		value, err := strconv.Atoi(strings.TrimPrefix(args[1], "d20="))
		if err != nil || value < 1 || value > 20 {
			return nil, errors.New("d20 must be between 1 and 20")
		}
		request := &dungeonfluxv1.DiceForceRequest{Room: room, D20: int32(value)}
		return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (*dungeonfluxv1.SendResponse, error) {
			return client.DiceForce(ctx, request)
		}, nil
	default:
		return nil, fmt.Errorf("unknown write verb %q", verb)
	}
}

func makeActCommand(room string, args []string) (writeCommand, error) {
	seat, arg, target, cell := "", "", "", ""
	move := ""
	for i := 0; i < len(args); i++ {
		value := func() string {
			i++
			if i >= len(args) {
				return ""
			}
			return args[i]
		}
		switch args[i] {
		case "--seat":
			seat = value()
		case "--arg":
			arg = value()
		case "--target":
			target = value()
		case "--cell":
			cell = value()
		default:
			if strings.HasPrefix(args[i], "--") || move != "" {
				return nil, errors.New("usage: act --seat N <move_id> [--arg A] [--target T] [--cell c,r]")
			}
			move = args[i]
		}
	}
	if seat == "" || move == "" {
		return nil, errors.New("usage: act --seat N <move_id> [--arg A] [--target T] [--cell c,r]")
	}
	request := &dungeonfluxv1.DebugActRequest{Room: room, Seat: seat, MoveId: move, Arg: arg, TargetId: target}
	if cell != "" {
		parts := strings.Split(cell, ",")
		if len(parts) != 2 {
			return nil, errors.New("cell must be c,r")
		}
		c, errC := strconv.ParseInt(parts[0], 10, 32)
		r, errR := strconv.ParseInt(parts[1], 10, 32)
		if errC != nil || errR != nil {
			return nil, errors.New("cell must be c,r")
		}
		request.Cell = &dungeonfluxv1.Cell{C: int32(c), R: int32(r)}
	}
	return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (*dungeonfluxv1.SendResponse, error) {
		return client.Act(ctx, request)
	}, nil
}

func writeVerb(args []string) string {
	for _, arg := range args {
		switch arg {
		case "send", "act", "say", "dice", "reset":
			return arg
		}
	}
	return ""
}

func splitWriteArgs(args []string, verb string) ([]string, []string) {
	for i, arg := range args {
		if arg == verb {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}
