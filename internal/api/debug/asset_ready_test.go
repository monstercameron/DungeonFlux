package debug

import (
	"context"
	"encoding/json"
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"strings"
	"testing"
)

func TestSend_PreparedBillboardUsesEngineInbox(t *testing.T) {
	hash := strings.Repeat("a", 64)
	for _, tc := range []struct {
		name     string
		change   func(*domain.AssetReady)
		rejected bool
	}{
		{"valid", func(*domain.AssetReady) {}, false},
		{"wrong slot", func(a *domain.AssetReady) { a.Slot = "portrait:1" }, true},
		{"wrong seat", func(a *domain.AssetReady) { a.Slot = "billboard:9:idle" }, true},
		{"wrong action", func(a *domain.AssetReady) { a.Slot = "billboard:1:unknown" }, true},
		{"remote URL", func(a *domain.AssetReady) { a.Asset.URL = "https://example.com/hero.mp4" }, true},
		{"bad hash", func(a *domain.AssetReady) { a.Asset.SHA256 = "bad" }, true},
		{"wrong mime", func(a *domain.AssetReady) { a.Asset.MIME = "image/png" }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ready := domain.AssetReady{Slot: "billboard:1:idle", Asset: domain.Asset{ID: domain.AssetID(hash), SHA256: hash, Kind: "video", MIME: "video/mp4", URL: "/assets/" + hash + ".mp4", DurationMS: 4000}}
			tc.change(&ready)
			payload, err := json.Marshal(ready)
			if err != nil {
				t.Fatal(err)
			}
			inbox := &fakes.FakeInbox{PostResult: true}
			server, _ := NewServer(&fakes.FakeEngine{}, inbox)
			got, err := server.Send(context.Background(), &df.SendRequest{Event: "asset_ready", PayloadJson: string(payload)})
			if tc.rejected {
				if err == nil || len(inbox.Calls) != 0 {
					t.Fatal("invalid asset reached engine")
				}
				return
			}
			if err != nil || !got.GetAccepted() || len(inbox.Calls) != 1 {
				t.Fatalf("send: %v, %v", got, err)
			}
			event, ok := inbox.Calls[0].Envelope.Event.(domain.AssetReady)
			if !ok || event.Slot != ready.Slot || event.Asset.URL != ready.Asset.URL || event.Asset.DurationMS != 4000 {
				t.Fatalf("lost asset data: %#v", event)
			}
		})
	}
	if _, err := decodeBillboardAsset("{"); err == nil {
		t.Fatal("invalid JSON accepted")
	}
}
