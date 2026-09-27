package dm

import (
	"reflect"
	"testing"
)

func TestRevealArt_WaitsForEveryLobbyPiece(t *testing.T) {
	loaded := map[string]string{
		titleBackgroundAsset: "blob:bg", titleWordmarkAsset: "blob:wm", titlePanelFrameAsset: "blob:frame", crestPosterAsset: "blob:crest",
	}
	resolve := func(name string) string { return loaded[name] }
	urls, ok := revealArt("lobby", "", "", "", resolve)
	if !ok || !reflect.DeepEqual(urls, []string{"blob:bg", "blob:wm", "blob:frame", "blob:crest"}) {
		t.Fatalf("lobby art = %v, %v", urls, ok)
	}
	if _, ok := revealArt("", "", "ui/qr", "", resolve); ok {
		t.Fatal("a join QR that is still loading must hold the splash")
	}
	loaded["ui/qr"] = "blob:qr"
	if urls, ok := revealArt("", "", "ui/qr", "", resolve); !ok || urls[len(urls)-1] != "blob:qr" {
		t.Fatalf("resolved QR = %v, %v", urls, ok)
	}
	if urls, ok := revealArt("lobby", "", "https://example.test/qr.png", "", resolve); !ok || urls[len(urls)-1] != "https://example.test/qr.png" {
		t.Fatalf("direct QR URL = %v, %v", urls, ok)
	}
	delete(loaded, titleWordmarkAsset)
	if _, ok := revealArt("lobby", "", "", "", resolve); ok {
		t.Fatal("a missing wordmark must hold the splash")
	}
}

func TestRevealArt_PerPhase(t *testing.T) {
	resolve := func(name string) string { return "blob:" + name }
	cases := []struct {
		phase, scene string
		want         []string
	}{
		{"creation", "", []string{"blob:" + titleBackgroundWideAsset}},
		{"END", "", []string{"blob:ui/end_bg"}},
		{"combat", "", nil},
		{"cliffhanger", "", nil},
		{"exploration", "blob:tavern", []string{"blob:tavern"}},
		{"conversation", "", nil},
	}
	for _, c := range cases {
		urls, ok := revealArt(c.phase, "", "", c.scene, resolve)
		if !ok || !reflect.DeepEqual(urls, c.want) {
			t.Fatalf("%s: %v, %v; want %v", c.phase, urls, ok, c.want)
		}
	}
	if _, ok := revealArt("creation", "", "", "", func(string) string { return "" }); ok {
		t.Fatal("creation background still loading must hold the splash")
	}
	if _, ok := revealArt("lobby", "", "", "", nil); ok {
		t.Fatal("no art source yet must hold the splash")
	}
}
