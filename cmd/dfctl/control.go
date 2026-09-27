package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/protobuf/proto"
)

var controlVerbs = map[string]bool{
	"goto": true, "seat": true, "timers": true, "timer": true,
	"pause": true, "resume": true, "combat": true, "snapshot": true,
	"vendor": true, "client": true,
}

type controlCommand func(context.Context, dungeonfluxv1.DebugServiceClient) (proto.Message, error)

func runControl(ctx context.Context, args []string, stdout, stderr io.Writer, dial connectFunc) (int, bool) {
	verb := findControlVerb(args)
	if verb == "" {
		return 0, false
	}
	opts, command, event, err := parseControlOptions(args, verb, stderr)
	if err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	if opts.dryRun {
		if event == nil {
			writeError(stderr, errors.New("--dry-run requires a write command"))
			return exitTransport, true
		}
		if err := writeDryRun(stdout, event, opts.pretty); err != nil {
			writeError(stderr, err)
			return exitTransport, true
		}
		return exitOK, true
	}
	client, closer, err := dial(ctx, opts.address)
	if err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	defer closer.Close()
	response, err := command(withDebugToken(ctx, opts.token), client)
	if err != nil {
		writeError(stderr, readError(err))
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

func parseControlOptions(args []string, verb string, stderr io.Writer) (options, controlCommand, proto.Message, error) {
	index := indexOf(args, verb)
	prefix, suffix := args[:index], args[index+1:]
	opts := options{address: defaultAddress, token: envDebugToken()}
	flags := flag.NewFlagSet("dfctl "+verb, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.address, "addr", opts.address, "debug gRPC address")
	flags.StringVar(&opts.room, "room", "", "room code")
	bindTokenFlag(flags, &opts.token)
	flags.BoolVar(&opts.pretty, "pretty", false, "indent JSON for people")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "print the event without posting it")
	if err := flags.Parse(prefix); err != nil {
		return options{}, nil, nil, err
	}
	command, event, err := makeControlCommand(verb, opts.room, suffix)
	return opts, command, event, err
}

func makeControlCommand(verb, room string, args []string) (controlCommand, proto.Message, error) {
	switch verb {
	case "goto":
		return makeGotoCommand(room, args)
	case "seat":
		return makeSeatCommand(room, args)
	case "timers":
		return makeTimersCommand(room, args)
	case "timer":
		return makeTimerCommand(room, args)
	case "pause", "resume":
		if len(args) != 0 {
			return nil, nil, fmt.Errorf("usage: %s", verb)
		}
		event := &dungeonfluxv1.SendRequest{Room: room, Event: "host_" + verb}
		return sendCommand(event), event, nil
	case "combat":
		return makeCombatCommand(room, args)
	case "snapshot":
		return makeSnapshotCommand(room, args)
	case "vendor":
		return makeVendorCommand(room, args)
	case "client":
		return makeClientCommand(room, args)
	default:
		return nil, nil, fmt.Errorf("unknown control verb %q", verb)
	}
}

func makeGotoCommand(room string, args []string) (controlCommand, proto.Message, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, nil, errors.New("usage: goto PHASE [--turn pc1|thrall|pc2]")
	}
	phase := args[0]
	if !validPhase(phase) {
		return nil, nil, fmt.Errorf("unknown phase %q", phase)
	}
	turn := ""
	if len(args) == 3 && args[1] == "--turn" {
		turn = args[2]
	} else if len(args) != 1 {
		return nil, nil, errors.New("usage: goto PHASE [--turn pc1|thrall|pc2]")
	}
	if turn != "" && (phase != "combat" || (turn != "pc1" && turn != "thrall" && turn != "pc2")) {
		return nil, nil, errors.New("--turn is only valid for combat: pc1, thrall, or pc2")
	}
	payload := map[string]string{"phase": phase}
	if turn != "" {
		payload["turn"] = turn
	}
	return sendJSONCommand(room, "debug_goto", payload)
}

func makeSeatCommand(room string, args []string) (controlCommand, proto.Message, error) {
	if len(args) < 2 {
		return nil, nil, errors.New("usage: seat set N [fields] | seat ready N")
	}
	if args[0] == "ready" {
		if len(args) != 2 {
			return nil, nil, errors.New("usage: seat ready N")
		}
		request := &dungeonfluxv1.DebugActRequest{Room: room, Seat: args[1], MoveId: "ready"}
		return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
			return client.Act(ctx, request)
		}, request, nil
	}
	if args[0] != "set" || len(args) < 3 {
		return nil, nil, errors.New("usage: seat set N --name NAME --species SPECIES --gender GENDER --class CLASS [--hp N] [--cell c,r]")
	}
	fields := map[string]string{}
	for i := 2; i < len(args); i += 2 {
		if i+1 >= len(args) || !strings.HasPrefix(args[i], "--") {
			return nil, nil, errors.New("seat fields require --name VALUE style arguments")
		}
		key := strings.TrimPrefix(args[i], "--")
		if !validSeatField(key) {
			return nil, nil, fmt.Errorf("unknown seat field %q", key)
		}
		fields[key], _ = valuePair(args, i)
	}
	if len(fields) == 0 {
		return nil, nil, errors.New("seat set requires at least one field")
	}
	return sendJSONCommand(room, "debug_patch", map[string]any{"target": "seat:" + args[1], "fields": fields})
}

func makeTimersCommand(room string, args []string) (controlCommand, proto.Message, error) {
	if len(args) == 0 {
		request := &dungeonfluxv1.DebugRoom{Room: room}
		return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
			return client.State(ctx, request)
		}, request, nil
	}
	if len(args) != 1 || (args[0] != "off" && args[0] != "on") {
		return nil, nil, errors.New("usage: timers | timers off | timers on")
	}
	event := "host_timers_" + args[0]
	request := &dungeonfluxv1.SendRequest{Room: room, Event: event}
	return sendCommand(request), request, nil
}

func makeTimerCommand(room string, args []string) (controlCommand, proto.Message, error) {
	if len(args) != 2 && len(args) != 3 {
		return nil, nil, errors.New("usage: timer fire|cancel NAME | timer set NAME MS")
	}
	op := args[0]
	if op != "fire" && op != "cancel" && op != "set" {
		return nil, nil, errors.New("timer operation must be fire, cancel, or set")
	}
	ms := 0
	if op == "set" {
		if len(args) != 3 {
			return nil, nil, errors.New("usage: timer set NAME MS")
		}
		value, err := strconv.Atoi(args[2])
		if err != nil || value < 0 {
			return nil, nil, errors.New("timer milliseconds must be non-negative")
		}
		ms = value
	} else if len(args) != 2 {
		return nil, nil, fmt.Errorf("usage: timer %s NAME", op)
	}
	return sendJSONCommand(room, "debug_timer", map[string]any{"name": args[1], "op": op, "ms": ms})
}

func makeCombatCommand(room string, args []string) (controlCommand, proto.Message, error) {
	if len(args) > 0 && args[0] == "end" {
		return makeCombatEndCommand(room, args[1:])
	}
	if len(args) != 3 || (args[0] != "hp" && args[0] != "move") {
		return nil, nil, errors.New("usage: combat hp TOKEN N | combat move TOKEN c,r | combat end slain|fled")
	}
	fields := map[string]string{}
	switch args[0] {
	case "hp":
		if _, err := strconv.Atoi(args[2]); err != nil {
			return nil, nil, errors.New("combat HP must be numeric")
		}
		fields["hp"] = args[2]
	case "move":
		if _, _, err := parseCell(args[2]); err != nil {
			return nil, nil, err
		}
		fields["cell"] = args[2]
	}
	return sendJSONCommand(room, "debug_patch", map[string]any{"target": "token:" + args[1], "fields": fields})
}

func makeCombatEndCommand(room string, args []string) (controlCommand, proto.Message, error) {
	// Keep the previously accepted explicit thrall form for existing scripts.
	if len(args) == 2 && args[0] == "thrall" {
		args = args[1:]
	}
	if len(args) != 1 || (args[0] != "slain" && args[0] != "fled") {
		return nil, nil, errors.New("usage: combat end slain|fled")
	}
	return sendJSONCommand(room, "debug_patch", map[string]any{
		"target": "token:thrall", "fields": map[string]string{"outcome": args[0]},
	})
}

func makeSnapshotCommand(room string, args []string) (controlCommand, proto.Message, error) {
	if len(args) != 2 || (args[0] != "save" && args[0] != "load") {
		return nil, nil, errors.New("usage: snapshot save|load NAME")
	}
	request := &dungeonfluxv1.SnapshotRequest{Room: room, Operation: args[0], Data: []byte(args[1])}
	return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
		return client.Snapshot(ctx, request)
	}, request, nil
}

func makeVendorCommand(room string, args []string) (controlCommand, proto.Message, error) {
	if len(args) < 2 || len(args) > 3 || (args[1] != "ok" && args[1] != "fail" && args[1] != "slow") {
		return nil, nil, errors.New("usage: vendor NAME ok|fail|slow MS")
	}
	payload := map[string]any{"vendor": args[0], "mode": args[1]}
	if args[1] == "slow" {
		if len(args) != 3 {
			return nil, nil, errors.New("usage: vendor NAME slow MS")
		}
		ms, err := strconv.Atoi(args[2])
		if err != nil || ms < 0 {
			return nil, nil, errors.New("vendor delay must be non-negative")
		}
		payload["ms"] = ms
	} else if len(args) != 2 {
		return nil, nil, errors.New("ok and fail do not take milliseconds")
	}
	request := &dungeonfluxv1.VendorRequest{Room: room, Operation: "set", PayloadJson: marshalJSON(payload)}
	return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
		return client.Vendor(ctx, request)
	}, request, nil
}

func makeClientCommand(room string, args []string) (controlCommand, proto.Message, error) {
	if len(args) < 2 || len(args) > 3 || (args[1] != "reload" && args[1] != "screenshot") {
		return nil, nil, errors.New("usage: client DM|seatN reload | client DM screenshot [PATH]")
	}
	if args[1] == "reload" && len(args) != 2 || args[1] == "screenshot" && len(args) == 2 {
		return nil, nil, errors.New("screenshot requires an output path")
	}
	arg := ""
	if len(args) == 3 {
		arg = args[2]
	}
	request := &dungeonfluxv1.ClientRequest{Room: room, ClientId: args[0], Verb: args[1], Arg: arg}
	return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
		return client.Client(ctx, request)
	}, request, nil
}

func sendCommand(request *dungeonfluxv1.SendRequest) controlCommand {
	return func(ctx context.Context, client dungeonfluxv1.DebugServiceClient) (proto.Message, error) {
		return client.Send(ctx, request)
	}
}

func sendJSONCommand(room, event string, payload any) (controlCommand, proto.Message, error) {
	request := &dungeonfluxv1.SendRequest{Room: room, Event: event, PayloadJson: marshalJSON(payload)}
	return sendCommand(request), request, nil
}

func writeDryRun(dst io.Writer, request proto.Message, pretty bool) error {
	value := map[string]any{"dry_run": true}
	switch request := request.(type) {
	case *dungeonfluxv1.SendRequest:
		value["event"], value["payload"] = request.GetEvent(), json.RawMessage(request.GetPayloadJson())
	case *dungeonfluxv1.DebugActRequest:
		value["event"], value["payload"] = "act", request
	case *dungeonfluxv1.SnapshotRequest:
		value["request"] = map[string]any{"room": request.GetRoom(), "operation": request.GetOperation(), "data": string(request.GetData())}
	case *dungeonfluxv1.VendorRequest:
		value["request"] = map[string]any{"room": request.GetRoom(), "operation": request.GetOperation(), "payload_json": request.GetPayloadJson()}
	case *dungeonfluxv1.ClientRequest:
		value["request"] = map[string]any{"room": request.GetRoom(), "client_id": request.GetClientId(), "verb": request.GetVerb(), "arg": request.GetArg()}
	default:
		value["request"] = request
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if pretty {
		data, err = json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(dst, "%s\n", data)
	return err
}

func findControlVerb(args []string) string {
	for _, arg := range args {
		if controlVerbs[arg] {
			return arg
		}
	}
	return ""
}

func indexOf(args []string, value string) int {
	for i, arg := range args {
		if arg == value {
			return i
		}
	}
	return -1
}

func validPhase(value string) bool {
	switch value {
	case "lobby", "creation", "opening", "exploration", "conversation", "check", "resolution", "hook_event", "combat", "cliffhanger", "end":
		return true
	default:
		return false
	}
}

func validSeatField(value string) bool {
	switch value {
	case "name", "species", "gender", "class", "hp", "cell":
		return true
	default:
		return false
	}
}

func valuePair(args []string, index int) (string, bool) {
	if index+1 >= len(args) {
		return "", false
	}
	return args[index+1], true
}

func parseCell(value string) (int, int, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return 0, 0, errors.New("cell must be c,r")
	}
	c, errC := strconv.Atoi(parts[0])
	r, errR := strconv.Atoi(parts[1])
	if errC != nil || errR != nil {
		return 0, 0, errors.New("cell must be c,r")
	}
	return c, r, nil
}

func marshalJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}
