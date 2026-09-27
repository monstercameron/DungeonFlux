package in

import (
	"context"
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestAssembler_End_preservesHeaderAndOrdersChunks(t *testing.T) {
	a := NewAssembler()
	session := Session{Seat: 1, UtteranceID: "u-1", MIME: "audio/webm;codecs=opus"}
	if err := a.Start(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	chunks := []Chunk{
		{Session: Session{Seat: 1, UtteranceID: "u-1", MIME: "audio/webm"}, Seq: 2, Data: []byte("tail")},
		{Session: Session{Seat: 1, UtteranceID: "u-1", MIME: "audio/webm"}, Seq: 0, Data: []byte{0x1a, 0x45, 0xdf, 0xa3}},
		{Session: Session{Seat: 1, UtteranceID: "u-1", MIME: "audio/webm"}, Seq: 1, Data: []byte("body")},
	}
	for _, chunk := range chunks {
		if err := a.Chunk(context.Background(), chunk); err != nil {
			t.Fatal(err)
		}
	}
	recording, err := a.End(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	if recording.MIME != "audio/webm" || string(recording.Audio) != "\x1aE\xdf\xa3bodytail" {
		t.Fatalf("recording = %#v", recording)
	}
}

func TestAssembler_End_acceptsIOSContainersAndTakeClones(t *testing.T) {
	tests := []struct {
		name string
		mime string
		want string
	}{
		{name: "mp4", mime: "audio/mp4", want: "audio/mp4"},
		{name: "m4a", mime: "audio/m4a; codecs=mp4a.40.2", want: "audio/mp4"},
		{name: "aac", mime: "audio/aac", want: "audio/aac"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAssembler()
			session := Session{UtteranceID: domain.UtteranceID(tc.name), MIME: tc.mime}
			if err := a.Start(context.Background(), session); err != nil {
				t.Fatal(err)
			}
			if err := a.Chunk(context.Background(), Chunk{Session: Session{UtteranceID: domain.UtteranceID(tc.name), MIME: tc.mime}, Data: []byte("ftyp-or-aac")}); err != nil {
				t.Fatal(err)
			}
			recording, err := a.End(context.Background(), session)
			if err != nil || recording.MIME != tc.want {
				t.Fatalf("recording=%+v err=%v", recording, err)
			}
			got, ok, err := a.Take(context.Background(), session.UtteranceID)
			if err != nil || !ok {
				t.Fatalf("take ok=%v err=%v", ok, err)
			}
			got.Audio[0] = 'X'
			if next, ok, _ := a.Take(context.Background(), session.UtteranceID); ok || next.Audio != nil {
				t.Fatal("take did not remove recording")
			}
		})
	}
}

func TestAssembler_rejectsInvalidLifecycle(t *testing.T) {
	a := NewAssembler()
	ctx := context.Background()
	if err := a.Start(ctx, Session{UtteranceID: "u", MIME: "audio/ogg"}); !errors.Is(err, ErrUnsupportedMIME) {
		t.Fatalf("unsupported MIME error = %v", err)
	}
	session := Session{UtteranceID: "u", MIME: "audio/mp4"}
	if err := a.Start(ctx, session); err != nil {
		t.Fatal(err)
	}
	if err := a.Start(ctx, session); !errors.Is(err, ErrSessionExists) {
		t.Fatalf("duplicate start error = %v", err)
	}
	if err := a.Chunk(ctx, Chunk{Session: session, Seq: 1, Data: []byte("gap")}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.End(ctx, session); !errors.Is(err, ErrChunkSequence) {
		t.Fatalf("gap error = %v", err)
	}
	if err := a.Cancel(ctx, session.UtteranceID); err != nil {
		t.Fatal(err)
	}
	if err := a.Chunk(ctx, Chunk{Session: session, Data: []byte("late")}); !errors.Is(err, ErrSessionMissing) {
		t.Fatalf("late chunk error = %v", err)
	}
}

func TestAssembler_rejectsMismatchedAndCancelledOperations(t *testing.T) {
	a := NewAssembler()
	ctx := context.Background()
	session := Session{Seat: 1, UtteranceID: "u", MIME: "audio/webm"}
	if err := a.Start(ctx, session); err != nil {
		t.Fatal(err)
	}
	if err := a.Chunk(ctx, Chunk{Session: Session{Seat: 2, UtteranceID: "u", MIME: session.MIME}, Data: []byte("bad")}); err == nil {
		t.Fatal("mismatched chunk accepted")
	}
	if err := a.Chunk(ctx, Chunk{Session: session, Data: []byte("header")}); err != nil {
		t.Fatal(err)
	}
	if err := a.Chunk(ctx, Chunk{Session: session, Data: []byte("duplicate")}); !errors.Is(err, ErrChunkSequence) {
		t.Fatalf("duplicate error = %v", err)
	}
	if err := a.Cancel(ctx, session.UtteranceID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := a.Take(ctx, session.UtteranceID); err != nil || ok {
		t.Fatalf("cancelled recording ok=%v err=%v", ok, err)
	}
}

func TestAssembler_contextCancellation(t *testing.T) {
	a := NewAssembler()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	session := Session{UtteranceID: "u", MIME: "audio/mp4"}
	if err := a.Start(ctx, session); !errors.Is(err, context.Canceled) {
		t.Fatalf("start error = %v", err)
	}
}
