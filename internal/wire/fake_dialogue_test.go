package wire

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestFakeDialogue_CaptionMatchesTheCachedNPCRecording(t *testing.T) {
	line, found := content.CannedLineByID(content.CannedNPCReplyID)
	if !found {
		t.Fatal("missing authored reply")
	}
	for _, question := range []string{"Where is the lamplighter?", "Did anyone follow him?"} {
		t.Run(question, func(t *testing.T) {
			request := ports.TextRequest{Meta: ports.CallMeta{Role: vocab.RoleNPCReply}, Messages: []ports.Message{{Role: vocab.MsgUser, Text: question}}}
			stream, err := newFakeLLM().StreamText(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Close()
			caption, err := stream.Recv()
			if err != nil || caption != line.Text {
				t.Fatalf("caption=%q err=%v; want cached transcript %q", caption, err, line.Text)
			}
			if _, err := stream.Recv(); err != io.EOF {
				t.Fatalf("reply did not end: %v", err)
			}
			pcm := []byte{1, 0, 2, 0}
			var requested domain.AssetID
			tts := fakeTTS{read: func(_ context.Context, id domain.AssetID) ([]byte, error) {
				requested = id
				return pcm, nil
			}}
			audio, err := tts.Stream(t.Context(), ports.TTSRequest{Meta: request.Meta}, &fakeTextStream{ctx: t.Context(), text: caption})
			if err != nil {
				t.Fatal(err)
			}
			defer audio.Close()
			chunk, err := audio.Recv()
			if err != nil || requested != domain.AssetID(line.ID) || !bytes.Equal(chunk.S16LE, pcm) {
				t.Fatalf("speech asset=%q chunk=%+v err=%v; want matching recording %q", requested, chunk, err, line.ID)
			}
		})
	}
}
