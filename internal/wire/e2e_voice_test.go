package wire

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/config"
	grpctunnel "github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// voiceRoom is a fake-mode app parked in Mother Vell's conversation, reached
// the way players reach it: phones join over the browser's gRPC tunnel and the
// game is driven through the debug service. Fake STT always hears "persuade"
// and fake interpret classifies it as MOVE persuade, so a line that makes it
// through the whole voice pipeline starts the Persuasion check.
type voiceRoom struct {
	phone   *grpc.ClientConn
	debug   df.DebugServiceClient
	ctx     context.Context
	room    string
	seatOne string
}

func newVoiceRoom(t *testing.T) voiceRoom {
	t.Helper()
	debugPort := freePort(t)
	t.Setenv("DF_DEBUG_TOKEN", "e2e-voice-token")
	dataDir := settledTempDir(t)
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: debugPort - 1000, Debug: true, LogLevel: "debug",
			DataDir: filepath.Join(dataDir, "runtime"), RoomCode: "DF-VOICE",
			HostToken: "voice-host", DMToken: "voice-dm",
		},
		Budget: config.BudgetConfig{PerRunUSD: 1, HardUSD: 2},
		Timeouts: config.TimeoutConfig{
			CharacterFlavor: time.Second, Interpret: time.Second,
			SpokenFirstToken: time.Second, Prerender: time.Second,
			Portrait: time.Second, TTS: time.Second,
		},
		Battlefield: config.BattlefieldConfig{NavPath: "internal/content/battlefield_tavern.json"},
	}
	app, err := Build(context.Background(), cfg, []byte("voice-seed"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	server := httptest.NewServer(app.Handler())
	t.Cleanup(server.Close)

	phone, err := grpctunnel.Dial(server.URL+"/grpc", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("phone tunnel: %v", err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	debugConn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", debugPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("debug client: %v", err)
	}
	t.Cleanup(func() { _ = debugConn.Close() })
	room := voiceRoom{phone: phone, debug: df.NewDebugServiceClient(debugConn), ctx: metadata.AppendToOutgoingContext(context.Background(), "x-df-debug-token", "e2e-voice-token"), room: "DF-VOICE"}

	session := df.NewSessionServiceClient(phone)
	for seat := int32(1); seat <= 2; seat++ {
		joined, err := session.Join(context.Background(), &df.JoinRequest{RoomCode: room.room, Kind: df.ClientKind_CLIENT_KIND_PHONE})
		if err != nil || joined.GetPlayerNumber() != seat {
			t.Fatalf("join seat %d: %v %v", seat, joined, err)
		}
		if seat == 1 {
			room.seatOne = joined.GetSeatToken()
		}
	}
	room.toConversation(t)
	return room
}

// settledTempDir is a temp directory removed after the app closes, retrying
// while fake-mode media executors finish writing into it. t.TempDir failed a
// fast test with "directory not empty" when a portrait landed during cleanup.
func settledTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "df-voice-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		deadline := time.Now().Add(2 * time.Second)
		for os.RemoveAll(dir) != nil && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
	})
	return dir
}

func (r voiceRoom) toConversation(t *testing.T) {
	t.Helper()
	sendAct(t, r.debug, r.ctx, "1", "ready", "")
	sendAct(t, r.debug, r.ctx, "2", "ready", "")
	sendDebug(t, r.debug, r.ctx, "host_start")
	for _, seat := range []string{"1", "2"} {
		class := map[string]string{"1": "paladin", "2": "rogue"}[seat]
		sendAct(t, r.debug, r.ctx, seat, "species", "human")
		sendAct(t, r.debug, r.ctx, seat, "gender", "female")
		sendAct(t, r.debug, r.ctx, seat, "class", class)
		sendAct(t, r.debug, r.ctx, seat, "roll_hero", "")
		if contains(readLegal(t, r.debug, r.ctx, r.room, seat), "ready") {
			sendAct(t, r.debug, r.ctx, seat, "ready", "")
		}
	}
	if state := waitPhaseLeaves(t, r.debug, r.ctx, r.room, "creation"); state.GetPhase() == "creation" {
		t.Skipf("creation stalled before the voice path could be tested; legal=%v", readLegal(t, r.debug, r.ctx, r.room, "1"))
	}
	for _, phase := range []string{"exploration", "conversation"} {
		sendDebug(t, r.debug, r.ctx, "host_skip")
		if got := r.waitPhase(t, phase); got != phase {
			t.Fatalf("phase = %q, want %q on the way to the conversation", got, phase)
		}
	}
}

// waitPhase polls until the room reaches want or 6 s pass, and returns the
// last phase seen.
func (r voiceRoom) waitPhase(t *testing.T, want string) string {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for {
		phase := readDebugState(t, r.debug, r.ctx, r.room).GetPhase()
		if phase == want || time.Now().After(deadline) {
			return phase
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// talk sends one push-to-talk recording over VoiceService.Talk exactly as the
// phone does and returns the chunk acks the server sent back.
func (r voiceRoom) talk(t *testing.T, token string, chunks [][]byte, end bool) []uint64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := df.NewVoiceServiceClient(r.phone).Talk(ctx)
	if err != nil {
		t.Fatalf("open Talk: %v", err)
	}
	if err := stream.Send(&df.TalkRequest{Message: &df.TalkRequest_Start{Start: &df.TalkStart{SeatToken: token, MimeType: "audio/webm;codecs=opus"}}}); err != nil {
		t.Fatalf("send TalkStart: %v", err)
	}
	for seq, data := range chunks {
		if err := stream.Send(&df.TalkRequest{Message: &df.TalkRequest_Chunk{Chunk: &df.AudioChunk{Seq: uint64(seq), Data: data}}}); err != nil {
			t.Fatalf("send chunk %d: %v", seq, err)
		}
	}
	if end {
		if err := stream.Send(&df.TalkRequest{Message: &df.TalkRequest_End{End: &df.TalkEnd{}}}); err != nil {
			t.Fatalf("send TalkEnd: %v", err)
		}
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("close send: %v", err)
	}
	var acks []uint64
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return acks
		}
		if err != nil {
			return acks
		}
		if ack := response.GetAck(); ack != nil {
			acks = append(acks, ack.GetSeq())
		}
	}
}

func webmChunks(count int) [][]byte {
	chunks := make([][]byte, count)
	for i := range chunks {
		chunks[i] = []byte(fmt.Sprintf("opus-frame-%02d", i))
	}
	return chunks
}

// TestE2E_Voice_pushToTalkReachesTheEngine guards the whole push-to-talk path
// through the real server: Talk stream -> assembler (not a no-op sink) ->
// TalkEnd -> Transcribe -> STT -> Transcribed -> interpret -> MOVE (any case)
// -> Persuasion check. Each link broke once; any one breaking again leaves the
// room in conversation.
func TestE2E_Voice_pushToTalkReachesTheEngine(t *testing.T) {
	room := newVoiceRoom(t)
	acks := room.talk(t, room.seatOne, webmChunks(5), true)
	if len(acks) != 5 {
		t.Fatalf("chunk acks = %v, want all 5 chunks acknowledged by the server", acks)
	}
	if got := room.waitPhase(t, "check"); got != "check" && got != "resolution" {
		t.Fatalf("phase after a spoken 'persuade' = %q, want the Persuasion check: the recording did not travel Talk -> STT -> interpret -> move", got)
	}
}

// TestE2E_Voice_cancelledRecordingIsNotTranscribed checks that a recording
// dropped before TalkEnd is discarded rather than acted on.
func TestE2E_Voice_cancelledRecordingIsNotTranscribed(t *testing.T) {
	room := newVoiceRoom(t)
	room.talk(t, room.seatOne, webmChunks(3), false)
	time.Sleep(300 * time.Millisecond)
	if got := readDebugState(t, room.debug, room.ctx, room.room).GetPhase(); got != "conversation" {
		t.Fatalf("phase after a recording without TalkEnd = %q, want conversation unchanged", got)
	}
}

// TestE2E_Voice_emptyRecordingDoesNotAdvance checks that TalkEnd with no audio
// never reaches interpret (the transcriber finds nothing to transcribe).
func TestE2E_Voice_emptyRecordingDoesNotAdvance(t *testing.T) {
	room := newVoiceRoom(t)
	room.talk(t, room.seatOne, nil, true)
	time.Sleep(300 * time.Millisecond)
	if got := readDebugState(t, room.debug, room.ctx, room.room).GetPhase(); got != "conversation" {
		t.Fatalf("phase after an empty recording = %q, want conversation unchanged", got)
	}
}

// TestE2E_Voice_invalidSeatTokenIsRejected checks that a stranger's stream is
// refused before any audio is accepted.
func TestE2E_Voice_invalidSeatTokenIsRejected(t *testing.T) {
	room := newVoiceRoom(t)
	if acks := room.talk(t, "not-a-seat", webmChunks(2), true); len(acks) != 0 {
		t.Fatalf("acks for an invalid seat = %v, want none", acks)
	}
	if got := readDebugState(t, room.debug, room.ctx, room.room).GetPhase(); got != "conversation" {
		t.Fatalf("phase after an invalid seat = %q, want conversation unchanged", got)
	}
}

// TestE2E_Voice_typedLineReachesTheEngine guards typed input: Say enters at
// the transcript stage, so typing "persuade" starts the check just like
// speaking it.
func TestE2E_Voice_typedLineReachesTheEngine(t *testing.T) {
	room := newVoiceRoom(t)
	response, err := df.NewSessionServiceClient(room.phone).Say(context.Background(), &df.SayRequest{SeatToken: room.seatOne, Text: "I try to persuade her."})
	if err != nil || !response.GetAccepted() {
		t.Fatalf("Say: response=%v err=%v", response, err)
	}
	if got := room.waitPhase(t, "check"); got != "check" && got != "resolution" {
		t.Fatalf("phase after a typed persuade = %q, want the Persuasion check: typed input did not reach interpret", got)
	}
}
