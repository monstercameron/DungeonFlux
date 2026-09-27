package sqlite

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestAssets_putGetAndUpdate(t *testing.T) {
	store := testStore(t)
	assets := NewAssets(store)
	asset := domain.Asset{ID: "asset-1", SHA256: "sha-1", Kind: "audio", MIME: "audio/ogg", DurationMS: 1200, Meta: map[string]string{"voice": "vell"}}
	if err := assets.Put(context.Background(), asset); err != nil {
		t.Fatal(err)
	}
	got, ok, err := assets.Get(context.Background(), asset.SHA256)
	if err != nil || !ok || got.Meta["voice"] != "vell" {
		t.Fatalf("asset = %#v, ok=%v, err=%v", got, ok, err)
	}
	asset.MIME = "audio/wav"
	if err := assets.Put(context.Background(), asset); err != nil {
		t.Fatal(err)
	}
	got, _, err = assets.Get(context.Background(), asset.SHA256)
	if err != nil || got.MIME != "audio/wav" {
		t.Fatalf("updated asset = %#v, err=%v", got, err)
	}
	if _, ok, err := assets.Get(context.Background(), "missing"); err != nil || ok {
		t.Fatalf("missing asset ok=%v, err=%v", ok, err)
	}
}
