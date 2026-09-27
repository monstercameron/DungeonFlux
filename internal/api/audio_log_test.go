package api

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
)

func captureLogger() (*slog.Logger, *bytes.Buffer) {
	var buffer bytes.Buffer
	return slog.New(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug})), &buffer
}

func talkServerWithSeat(t *testing.T) (*TalkServer, string) {
	t.Helper()
	inbox := &fakes.FakeInbox{PostResult: true}
	session, err := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	join, err := session.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewTalkServer(inbox, session, &talkSink{})
	if err != nil {
		t.Fatal(err)
	}
	return server, join.GetSeatToken()
}

func TestTalkServer_logsEachRecording(t *testing.T) {
	cases := []struct {
		name     string
		tail     []*df.TalkRequest
		cancel   bool
		wantLine string
		wantMore []string
	}{
		{
			name:     "sent recording",
			tail:     []*df.TalkRequest{chunkRequest(0, 3), chunkRequest(1, 5), {Message: &df.TalkRequest_End{End: &df.TalkEnd{}}}},
			wantLine: `msg="talk ended"`,
			wantMore: []string{"chunks=2", "bytes=8", "audio_ms=200", `msg="talk audio receiving"`, `msg="talk chunk received"`, "seq=1", "total_chunks=2", "total_bytes=8"},
		},
		{
			name:     "stream dropped before TalkEnd",
			tail:     []*df.TalkRequest{chunkRequest(0, 4)},
			cancel:   true,
			wantLine: `msg="talk cancelled"`,
			wantMore: []string{"chunks=1", "bytes=4", `reason="stream closed before TalkEnd"`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, token := talkServerWithSeat(t)
			logger, buffer := captureLogger()
			server.SetLogger(logger)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			requests := append([]*df.TalkRequest{{Message: &df.TalkRequest_Start{Start: &df.TalkStart{SeatToken: token, MimeType: "audio/webm"}}}}, tc.tail...)
			stream := &talkStream{ctx: ctx, requests: requests}
			if tc.cancel {
				stream.recvErr, stream.onEmpty = context.Canceled, cancel
			}
			if err := server.Talk(stream); err != nil {
				t.Fatal(err)
			}
			logs := buffer.String()
			for _, want := range append([]string{`msg="talk started"`, "mime=audio/webm", tc.wantLine, "component=talk"}, tc.wantMore...) {
				if !strings.Contains(logs, want) {
					t.Fatalf("logs lack %q:\n%s", want, logs)
				}
			}
		})
	}
}

func chunkRequest(seq uint64, size int) *df.TalkRequest {
	return &df.TalkRequest{Message: &df.TalkRequest_Chunk{Chunk: &df.AudioChunk{Seq: seq, Data: bytes.Repeat([]byte{1}, size)}}}
}

func TestListenHub_logsSubscriptionsAndVoiceLines(t *testing.T) {
	hub := NewListenHub()
	logger, buffer := captureLogger()
	hub.SetLogger(logger)
	ctx, cancel := context.WithCancel(context.Background())
	sub := hub.SubscribeTarget(ctx, AudioTarget{Kind: TargetDM}, false)
	hub.Frame(domain.AudioFrame{UtteranceID: "vell-1", Speaker: "npc_reply", Seq: 0, SampleRate: 24000, PCMS16LE: make([]byte, 480)})
	hub.Frame(domain.AudioFrame{UtteranceID: "vell-1", Speaker: "npc_reply", Seq: 1, SampleRate: 24000, PCMS16LE: make([]byte, 240), Final: true})
	cancel()
	sub.Close()
	hub.remove(1, sub)
	logs := buffer.String()
	for _, want := range []string{
		`msg="listen subscribed"`, "target=" + TargetDM, "component=listen",
		`msg="voice line sent"`, "utterance=vell-1", "frames=2", "bytes=720", "listeners=1",
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs lack %q:\n%s", want, logs)
		}
	}
	if strings.Count(logs, `msg="voice line sent"`) != 1 {
		t.Fatalf("a line must be summarised once, at its final frame:\n%s", logs)
	}
}

func TestAudioLogging_withoutLoggerIsSilent(t *testing.T) {
	var server *TalkServer
	server.SetLogger(nil)
	server.log().Info("nothing")
	var hub *ListenHub
	hub.SetLogger(nil)
	hub.log().Info("nothing")
}
