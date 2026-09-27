package wire

import (
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game"
)

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

func TestWithPublicURLs(t *testing.T) {
	local := []string{"http://localhost:8444/p?room=DF-ROOM", "http://10.0.0.9:8444/p?room=DF-ROOM"}
	tests := []struct {
		name      string
		publicURL string
		wantURLs  []string
		wantJoin  string
	}{
		{name: "no public url keeps the LAN join url", wantURLs: local, wantJoin: local[1]},
		{name: "public url leads, drops unreachable LAN urls, and becomes the join url", publicURL: "https://play.example.com/",
			wantURLs: []string{"https://play.example.com/dm?token=dm+1", "https://play.example.com/host?t=host", "https://play.example.com/p?room=DF-ROOM", local[0]},
			wantJoin: "https://play.example.com/p?room=DF-ROOM"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			urls, join := withPublicURLs(tc.publicURL, append([]string(nil), local...), 8444, "DF-ROOM", "dm 1", "host")
			if join != tc.wantJoin || strings.Join(urls, " ") != strings.Join(tc.wantURLs, " ") {
				t.Fatalf("withPublicURLs() = %q, %q; want %q, %q", urls, join, tc.wantURLs, tc.wantJoin)
			}
		})
	}
}

func TestLobbyEngine_AttachesMetadataToProjectedView(t *testing.T) {
	engine := newLobbyEngine(game.New(domain.OneShot{}, nil), api.LobbyProjection{
		RoomCode: "DF-ROOM", JoinURL: "https://dm.test/p?room=DF-ROOM", QRURL: "/assets/qr.png",
	})
	lobby := api.ProjectDM(engine.View()).GetLobby()
	if lobby == nil || lobby.GetRoomCode() != "DF-ROOM" || lobby.GetJoinUrl() == "" || lobby.GetQrUrl() != "/assets/qr.png" {
		t.Fatalf("projected lobby = %#v", lobby)
	}
}
