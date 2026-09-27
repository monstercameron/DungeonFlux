package phone

import (
	"context"
	"errors"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestToggleControl_Tap(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	later := func(step int) time.Time { return base.Add(time.Duration(step) * time.Second) }
	cases := []struct {
		name       string
		run        func(c *ToggleControl) []ToggleAction
		want       []ToggleAction
		wantPhase  TogglePhase
		wantNotice string
	}{
		{
			name:      "first tap starts",
			run:       func(c *ToggleControl) []ToggleAction { return []ToggleAction{c.Tap(later(0))} },
			want:      []ToggleAction{ToggleStart},
			wantPhase: ToggleStarting,
		},
		{
			name: "second tap sends",
			run: func(c *ToggleControl) []ToggleAction {
				return []ToggleAction{c.Tap(later(0)), c.Started(nil), c.Tap(later(2))}
			},
			want:      []ToggleAction{ToggleStart, ToggleNone, ToggleFinish},
			wantPhase: ToggleFinishing,
		},
		{
			name: "double-fired tap does not stop the recording",
			run: func(c *ToggleControl) []ToggleAction {
				return []ToggleAction{c.Tap(later(0)), c.Started(nil), c.Tap(later(0).Add(5 * time.Millisecond))}
			},
			want:      []ToggleAction{ToggleStart, ToggleNone, ToggleNone},
			wantPhase: ToggleRecording,
		},
		{
			name: "tap while the mic is opening sends once it opens",
			run: func(c *ToggleControl) []ToggleAction {
				return []ToggleAction{c.Tap(later(0)), c.Tap(later(1)), c.Started(nil)}
			},
			want:      []ToggleAction{ToggleStart, ToggleNone, ToggleFinish},
			wantPhase: ToggleFinishing,
		},
		{
			name: "microphone failure returns to idle with a visible notice",
			run: func(c *ToggleControl) []ToggleAction {
				return []ToggleAction{c.Tap(later(0)), c.Started(errors.New("Permission denied"))}
			},
			want:       []ToggleAction{ToggleStart, ToggleNone},
			wantPhase:  ToggleIdle,
			wantNotice: "Permission denied",
		},
		{
			name: "tap while finishing is ignored",
			run: func(c *ToggleControl) []ToggleAction {
				return []ToggleAction{c.Tap(later(0)), c.Started(nil), c.Tap(later(1)), c.Tap(later(2))}
			},
			want:      []ToggleAction{ToggleStart, ToggleNone, ToggleFinish, ToggleNone},
			wantPhase: ToggleFinishing,
		},
		{
			name: "finished control starts again",
			run: func(c *ToggleControl) []ToggleAction {
				out := []ToggleAction{c.Tap(later(0)), c.Started(nil), c.Tap(later(1))}
				c.Finished("")
				return append(out, c.Tap(later(2)))
			},
			want:      []ToggleAction{ToggleStart, ToggleNone, ToggleFinish, ToggleStart},
			wantPhase: ToggleStarting,
		},
		{
			name: "finished with an error keeps the notice until the next start",
			run: func(c *ToggleControl) []ToggleAction {
				out := []ToggleAction{c.Tap(later(0)), c.Started(nil), c.Tap(later(1))}
				c.Finished("upload failed")
				return out
			},
			want:       []ToggleAction{ToggleStart, ToggleNone, ToggleFinish},
			wantPhase:  ToggleIdle,
			wantNotice: "upload failed",
		},
		{
			name:      "started report without a start is ignored",
			run:       func(c *ToggleControl) []ToggleAction { return []ToggleAction{c.Started(nil)} },
			want:      []ToggleAction{ToggleNone},
			wantPhase: ToggleIdle,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &ToggleControl{}
			got := tc.run(c)
			if len(got) != len(tc.want) {
				t.Fatalf("actions = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("actions = %v, want %v", got, tc.want)
				}
			}
			if phase := c.Phase(); phase != tc.wantPhase {
				t.Fatalf("phase = %q, want %q", phase, tc.wantPhase)
			}
			if notice := c.Notice(); notice != tc.wantNotice {
				t.Fatalf("notice = %q, want %q", notice, tc.wantNotice)
			}
		})
	}
}

func TestPTTModel_ToggleOutlivesRenders(t *testing.T) {
	model := NewPTTModel(nil, "seat", 1)
	first := model.Toggle()
	first.Tap(time.Unix(1_700_000_000, 0))
	if second := model.Toggle(); second != first || second.Phase() != ToggleStarting {
		t.Fatalf("second render got %p in %q, want the same control %p still starting", second, second.Phase(), first)
	}
	var nilModel *PTTModel
	if nilModel.Toggle() != nil {
		t.Fatal("nil model returned a control")
	}
}

func TestTalkStatus(t *testing.T) {
	cases := []struct {
		name       string
		phase      TogglePhase
		notice     string
		wantStatus string
		wantLabel  string
		wantAria   string
	}{
		{name: "idle", phase: ToggleIdle, wantStatus: PTTReady("en"), wantLabel: "●", wantAria: PTTStart("en")},
		{name: "starting shows listening", phase: ToggleStarting, wantStatus: T("en", "ui.ptt.recording", nil), wantLabel: "■", wantAria: PTTStop("en")},
		{name: "recording", phase: ToggleRecording, wantStatus: T("en", "ui.ptt.recording", nil), wantLabel: "■", wantAria: PTTStop("en")},
		{name: "finishing", phase: ToggleFinishing, wantStatus: T("en", "ptt.finishing", nil), wantLabel: "…", wantAria: PTTStop("en")},
		{name: "error is visible when idle", phase: ToggleIdle, notice: "Permission denied", wantStatus: "Permission denied", wantLabel: "●", wantAria: PTTStart("en")},
		{name: "stale notice hidden while recording", phase: ToggleRecording, notice: "old error", wantStatus: T("en", "ui.ptt.recording", nil), wantLabel: "■", wantAria: PTTStop("en")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, label, aria := talkStatus("en", tc.phase, tc.notice)
			if status != tc.wantStatus || label != tc.wantLabel || aria != tc.wantAria {
				t.Fatalf("talkStatus = (%q, %q, %q), want (%q, %q, %q)", status, label, aria, tc.wantStatus, tc.wantLabel, tc.wantAria)
			}
		})
	}
}

type countingTalkStream struct{ sent int }

func (s *countingTalkStream) Send(*df.TalkRequest) error { s.sent++; return nil }
func (s *countingTalkStream) CloseSend() error           { return nil }

type countingOpener struct{ stream *countingTalkStream }

func (o countingOpener) OpenTalk(context.Context) (TalkStream, error) { return o.stream, nil }

func TestPTTModel_SentCountsChunksPerRecording(t *testing.T) {
	stream := &countingTalkStream{}
	model := NewPTTModel(countingOpener{stream: stream}, "seat", 4)
	for round, sizes := range [][]int{{3, 5}, {7}} {
		if err := model.Start(context.Background(), "audio/webm"); err != nil {
			t.Fatal(err)
		}
		want := SentAudio{}
		for _, size := range sizes {
			if !model.QueueChunk(make([]byte, size)) {
				t.Fatalf("round %d: chunk rejected", round)
			}
			want.Chunks++
			want.Bytes += size
		}
		if err := <-model.Stop(context.Background()); err != nil {
			t.Fatalf("round %d: stop: %v", round, err)
		}
		if got := model.Sent(); got != want {
			t.Fatalf("round %d: sent = %+v, want %+v (counts reset per recording)", round, got, want)
		}
	}
	var nilModel *PTTModel
	if nilModel.Sent() != (SentAudio{}) {
		t.Fatal("nil model reported sent audio")
	}
}
