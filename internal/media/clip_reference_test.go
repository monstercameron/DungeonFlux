package media

import (
	"context"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestClipExecutor_ComposesReferenceFrameBeforeVideo(t *testing.T) {
	refs := completeReferences()
	var got []domain.AssetID
	videos := &fakes.FakeVideoGen{Submits: []fakes.VideoResult{{Job: ports.VideoJob{ID: "job"}}}, Polls: []fakes.VideoResult{{Status: ports.VideoStatus{State: vocab.JobDone, URL: "url"}}}}
	assets := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: domain.Asset{ID: "clip"}}}}
	in := &fakes.FakeInbox{PostResult: true}
	NewClipExecutor(ClipConfig{
		Videos: videos, Assets: assets, Download: func(context.Context, string) ([]byte, error) { return []byte("mp4"), nil },
		References: map[domain.SeatID]ReferenceAssets{1: refs}, ReferenceFrame: func(_ context.Context, ids []domain.AssetID) ([]byte, error) {
			got = ids
			return []byte("composed"), nil
		},
	}).Execute(context.Background(), domain.GenerateClip{Slot: "clip:1", Shot: "attack"}, domain.Scope{}, in)
	if len(got) != 5 || string(videos.Calls[0].Request.FirstFrame) != "composed" || !strings.Contains(videos.Calls[0].Request.Prompt, "side") {
		t.Fatalf("ids=%v request=%#v", got, videos.Calls[0].Request)
	}
}
