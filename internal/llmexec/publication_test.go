package llmexec

import (
	"context"
	"errors"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"io"
	"strings"
	"testing"
)

type guardedLLM struct {
	fakes.FakeLLM
	stream *guardedStream
}

func (g *guardedLLM) StreamText(context.Context, ports.TextRequest) (ports.TextStream, error) {
	return g.stream, nil
}

type guardedStream struct {
	chunks []string
	before func()
	err    error
	closed bool
}

func (g *guardedStream) Recv() (string, error) {
	g.before()
	if len(g.chunks) == 0 {
		if g.err != nil {
			return "", g.err
		}
		return "", io.EOF
	}
	chunk := g.chunks[0]
	g.chunks = g.chunks[1:]
	return chunk, nil
}
func (g *guardedStream) Close() error { g.closed = true; return nil }
func TestSpokenPublication_validatesBeforeAnyConsumerSeesText(t *testing.T) {
	cases := []struct {
		name   string
		role   vocab.Role
		chunks []string
		want   string
		err    error
	}{
		{name: "split clue", role: vocab.RoleNPCReply, chunks: []string{"Listen, ", "the to", "wer."}},
		{name: "refusal cannot leak", role: vocab.RoleNPCRefuse, chunks: []string{"The bell tower."}},
		{name: "clean reply", role: vocab.RoleNPCReply, chunks: []string{"**sigh", "s** Drink", " ", "up."}, want: "Drink up."},
		{name: "opening stage directions", role: vocab.RoleOpening, chunks: []string{"*Rain swells* ", "Welcome, heroes."}, want: "Welcome, heroes."},
		{name: "reveal is allowed", role: vocab.RoleNPCReveal, chunks: []string{"*whispers* The bell ", "tower."}, want: "The bell tower."},
		{name: "unclosed action", role: vocab.RoleNPCReply, chunks: []string{"*Vell sighs"}},
		{name: "structured reply", role: vocab.RoleNPCReply, chunks: []string{`{"answer":"Drink up."}`}, want: "Drink up."},
		{name: "partial stream failure", role: vocab.RoleNPCReply, chunks: []string{"Fine, love."}, err: errors.New("disconnected")},
	}
	for _, role := range []vocab.Role{vocab.RoleOpening, vocab.RoleNPCReply, vocab.RoleNPCRefuse} {
		cases = append(cases, struct {
			name   string
			role   vocab.Role
			chunks []string
			want   string
			err    error
		}{name: "bounded " + string(role), role: role, chunks: []string{strings.Repeat("a", maxSpokenBytes+1)}})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &fakes.FakeInbox{}
			stream := &guardedStream{chunks: tc.chunks, err: tc.err, before: func() {
				if len(in.Calls) != 0 {
					t.Fatal("text escaped before stream validation")
				}
			}}
			llm := &guardedLLM{stream: stream}
			effect := domain.StartLine{Role: tc.role, UtteranceID: "line", Input: "A patron asks a question."}
			switch tc.role {
			case vocab.RoleOpening:
				NewOpeningExecutor(llm).Execute(context.Background(), effect, domain.Scope{}, in)
			case vocab.RoleNPCReply:
				NewNPCReplyExecutor(llm).Execute(context.Background(), effect, domain.Scope{}, in)
			default:
				NewOutcomeExecutor(llm, testClue).Execute(context.Background(), effect, domain.Scope{}, in)
			}
			if !stream.closed {
				t.Fatal("stream was not released")
			}
			if tc.want == "" {
				if len(in.Calls) != 1 {
					t.Fatalf("rejected output emitted %d events", len(in.Calls))
				}
				if _, ok := in.Calls[0].Envelope.Event.(domain.LineFailed); !ok {
					t.Fatalf("rejected output escaped: %#v", in.Calls)
				}
				return
			}
			if len(in.Calls) != 2 {
				t.Fatalf("accepted line events=%#v", in.Calls)
			}
			delta, ok := in.Calls[0].Envelope.Event.(domain.NarrationDelta)
			if !ok || !delta.Final || delta.Text != tc.want || delta.TextSoFar != tc.want {
				t.Fatalf("published %#v, want %q", delta, tc.want)
			}
			if _, ok := in.Calls[1].Envelope.Event.(domain.LineDone); !ok {
				t.Fatal("accepted line did not complete")
			}
		})
	}
}
