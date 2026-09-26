package in

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	voiceout "github.com/monstercameron/DungeonFlux/internal/voice/out"
)

// TestSpeakHearAnswerLoop_roundTripsPCMToTranscript drives one full voice
// turn through the production executors: the NPC line renders via voice/out,
// the bytes travel the assembler as the player's recorded answer, and
// voice/in transcribes them back into a Transcribed event. It fails if any
// stage drops, rewrites, or misroutes a byte, a keyterm, or the utterance
// identity.
func TestSpeakHearAnswerLoop_roundTripsPCMToTranscript(t *testing.T) {
	tests := []struct {
		name    string
		chunks  int
		reverse bool
	}{
		{name: "single chunk", chunks: 1},
		{name: "three chunks in order", chunks: 3},
		{name: "three chunks out of order", chunks: 3, reverse: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			scope := domain.Scope{Key: "utterance/u-7"}
			session := Session{Seat: 1, UtteranceID: "u-7", MIME: "audio/webm"}

			tts := &fakes.FakeTTS{Script: []fakes.TTSResult{{Chunks: []ports.PCMChunk{
				{SampleRate: 24000, S16LE: []byte{0, 1, 2, 3}},
				{SampleRate: 24000, S16LE: []byte{4, 5, 6, 7}},
			}}}}
			writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: domain.Asset{ID: "vell-q"}}}}
			playInbox := &fakes.FakeInbox{}
			voiceout.NewRenderLinesExecutor(tts, writer).Execute(ctx, domain.RenderLines{
				Set:    "vell",
				Texts:  []string{"You here to drink, or hunt for the lamplighter?"},
				Voices: []string{"onyx"},
			}, scope, playInbox)

			if len(playInbox.Calls) != 1 {
				t.Fatalf("play posts = %d", len(playInbox.Calls))
			}
			done, ok := playInbox.Calls[0].Envelope.Event.(domain.PrerenderDone)
			if !ok || done.Set != "vell" || len(done.Assets) != 1 || done.Assets[0].ID != "vell-q" {
				t.Fatalf("play event = %#v", playInbox.Calls[0].Envelope.Event)
			}
			if len(writer.Calls) != 1 {
				t.Fatalf("asset writes = %d", len(writer.Calls))
			}
			spoken := writer.Calls[0].Data
			if string(spoken) != string([]byte{0, 1, 2, 3, 4, 5, 6, 7}) {
				t.Fatalf("rendered PCM = %v", spoken)
			}
			if tts.Calls[0].Request.VoiceID != "onyx" {
				t.Fatalf("voice = %+v", tts.Calls[0].Request)
			}

			assembler := NewAssembler()
			if err := assembler.Start(ctx, session); err != nil {
				t.Fatal(err)
			}
			order := make([]int, tc.chunks)
			for i := range order {
				order[i] = i
			}
			if tc.reverse {
				for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
					order[i], order[j] = order[j], order[i]
				}
			}
			for _, seq := range order {
				lo := seq * len(spoken) / tc.chunks
				hi := (seq + 1) * len(spoken) / tc.chunks
				if err := assembler.Chunk(ctx, Chunk{Session: session, Seq: uint64(seq), Data: spoken[lo:hi]}); err != nil {
					t.Fatal(err)
				}
			}
			recording, err := assembler.End(ctx, session)
			if err != nil {
				t.Fatal(err)
			}
			if string(recording.Audio) != string(spoken) {
				t.Fatalf("assembled = %v, want %v", recording.Audio, spoken)
			}

			stt := &fakes.FakeSTT{Script: []fakes.STTResult{{Transcript: ports.Transcript{Text: "I hunt the lamplighter"}}}}
			executor, err := NewTranscriber(stt, assembler)
			if err != nil {
				t.Fatal(err)
			}
			hearInbox := &fakes.FakeInbox{}
			executor.Execute(ctx, domain.Transcribe{Seat: 1, UtteranceID: "u-7", Keyterms: []string{"the lamplighter"}}, scope, hearInbox)

			if len(hearInbox.Calls) != 1 {
				t.Fatalf("hear posts = %d", len(hearInbox.Calls))
			}
			heard, ok := hearInbox.Calls[0].Envelope.Event.(domain.Transcribed)
			if !ok || heard.UtteranceID != "u-7" || heard.Text != "I hunt the lamplighter" {
				t.Fatalf("hear event = %#v", hearInbox.Calls[0].Envelope.Event)
			}
			if hearInbox.Calls[0].Envelope.Scope != scope {
				t.Fatalf("scope = %#v", hearInbox.Calls[0].Envelope.Scope)
			}
			if len(stt.Calls) != 1 {
				t.Fatalf("stt calls = %d", len(stt.Calls))
			}
			got := stt.Calls[0]
			if string(got.Audio) != string(spoken) {
				t.Fatalf("stt audio = %v, want %v", got.Audio, spoken)
			}
			if len(got.Keyterms) != 1 || got.Keyterms[0] != "the lamplighter" || got.Language != "en-US" || got.Meta.Seat != 1 {
				t.Fatalf("stt request = %#v", got)
			}
		})
	}
}
