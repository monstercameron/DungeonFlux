package wire

import "testing"

func TestPreferredLANJoinURL_SelectsUsableIPv4(t *testing.T) {
	urls := []string{
		"http://localhost:18197/p?room=DF-ROOM",
		"http://169.254.10.4:18197/p?room=DF-ROOM",
		"http://192.168.1.7:18197/p?room=DF-ROOM",
		"http://10.0.0.9:18197/p?room=DF-ROOM",
	}
	if got := preferredLANJoinURL(urls, 18197, "DF-ROOM"); got != urls[2] {
		t.Fatalf("preferredLANJoinURL() = %q, want %q", got, urls[2])
	}
}

func TestPreferredLANJoinURL_FallsBackToLocalhost(t *testing.T) {
	got := preferredLANJoinURL([]string{"http://localhost:18197/p?room=DF-ROOM"}, 18197, "DF-ROOM")
	if got != "http://localhost:18197/p?room=DF-ROOM" {
		t.Fatalf("preferredLANJoinURL() = %q", got)
	}
}
