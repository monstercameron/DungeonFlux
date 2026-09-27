package audio

import (
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func frameMessage(id string, size int, final bool) *dungeonfluxv1.AudioMessage {
	return &dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Frame{Frame: &dungeonfluxv1.AudioFrame{UtteranceId: id, Speaker: "npc_reply", SampleRate: 24000, PcmS16Le: make([]byte, size), Final: final}}}
}

func TestReceiveLog_Observe(t *testing.T) {
	var log ReceiveLog
	if line := log.Observe(frameMessage("vell-1", 480, false)); line != "" {
		t.Fatalf("mid-line frame reported %q, want nothing until the final frame", line)
	}
	if pending := log.Pending(); len(pending) != 1 || pending[0] != "vell-1" {
		t.Fatalf("pending = %v, want the unfinished line", pending)
	}
	line := log.Observe(frameMessage("vell-1", 240, true))
	for _, want := range []string{"vell-1", "npc_reply", "2 frames", "720 bytes", "24000 Hz"} {
		if !strings.Contains(line, want) {
			t.Fatalf("summary %q lacks %q", line, want)
		}
	}
	if len(log.Pending()) != 0 {
		t.Fatalf("pending after final = %v, want none", log.Pending())
	}
	cancel := &dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Cancel{Cancel: &dungeonfluxv1.AudioCancel{UtteranceId: "vell-2"}}}
	if line := log.Observe(cancel); !strings.Contains(line, "cancel received for vell-2") {
		t.Fatalf("cancel line = %q", line)
	}
	if line := log.Observe(&dungeonfluxv1.AudioMessage{}); line != "" {
		t.Fatalf("empty message reported %q", line)
	}
}

func TestListenClient_WithDebugNilIsSafe(t *testing.T) {
	var client *ListenClient
	if client.WithDebug(func(string) {}) != nil {
		t.Fatal("nil client returned a client")
	}
	NewListenClient(nil).WithDebug(nil).report("ignored")
}
