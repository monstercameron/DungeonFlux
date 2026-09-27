package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestClipExecutor_DownloadsCompletedJob(t *testing.T) {
	videos := &fakes.FakeVideoGen{Submits: []fakes.VideoResult{{Job: ports.VideoJob{ID: "job"}}}, Polls: []fakes.VideoResult{{Status: ports.VideoStatus{State: vocab.JobDone, URL: "video-url"}}}}
	assets := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: domain.Asset{ID: "clip"}}}}
	in := &fakes.FakeInbox{PostResult: true}
	e := NewClipExecutor(ClipConfig{Videos: videos, Assets: assets, Download: func(_ context.Context, url string) ([]byte, error) {
		if url != "video-url" {
			t.Fatal(url)
		}
		return []byte("mp4"), nil
	}})
	e.Execute(context.Background(), domain.GenerateClip{Slot: "shot", Shot: "slow push", FirstFrame: []byte("png"), Resolution: "480p"}, domain.Scope{}, in)
	if len(assets.Calls) != 1 || string(assets.Calls[0].Data) != "mp4" {
		t.Fatalf("asset writes = %+v", assets.Calls)
	}
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetReady); !ok {
		t.Fatalf("event = %T", in.Calls[0].Envelope.Event)
	}
	if videos.Calls[0].Request.Seconds != 5 {
		t.Fatalf("seconds = %d", videos.Calls[0].Request.Seconds)
	}
}

func TestClipExecutor_DeadlinePostsFailure(t *testing.T) {
	videos := &fakes.FakeVideoGen{Submits: []fakes.VideoResult{{Job: ports.VideoJob{ID: "job"}}}, Polls: []fakes.VideoResult{{Status: ports.VideoStatus{State: vocab.JobRunning}}}}
	in := &fakes.FakeInbox{PostResult: true}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	NewClipExecutor(ClipConfig{Videos: videos, Assets: &fakes.FakeAssetWriter{}, Download: func(context.Context, string) ([]byte, error) { return nil, nil }, PollInterval: time.Hour}).Execute(ctx, domain.GenerateClip{Slot: "shot"}, domain.Scope{}, in)
	event, ok := in.Calls[0].Envelope.Event.(domain.AssetFailed)
	if !ok || event.FailureKind != vocab.ErrCanceled {
		t.Fatalf("event = %#v", in.Calls[0].Envelope.Event)
	}
}

func TestClipExecutor_MapsVendorAndDownloadFailures(t *testing.T) {
	for _, test := range []struct {
		name     string
		videos   *fakes.FakeVideoGen
		download ClipDownloader
		want     vocab.ErrKind
	}{
		{name: "submit", videos: &fakes.FakeVideoGen{Submits: []fakes.VideoResult{{Err: &ports.CallError{Kind: vocab.ErrRateLimited}}}}, download: func(context.Context, string) ([]byte, error) { return nil, nil }, want: vocab.ErrRateLimited},
		{name: "download", videos: &fakes.FakeVideoGen{Submits: []fakes.VideoResult{{Job: ports.VideoJob{ID: "job"}}}, Polls: []fakes.VideoResult{{Status: ports.VideoStatus{State: vocab.JobDone, URL: "url"}}}}, download: func(context.Context, string) ([]byte, error) { return nil, errors.New("expired") }, want: vocab.ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			in := &fakes.FakeInbox{PostResult: true}
			NewClipExecutor(ClipConfig{Videos: test.videos, Assets: &fakes.FakeAssetWriter{}, Download: test.download}).Execute(context.Background(), domain.GenerateClip{Slot: "shot"}, domain.Scope{}, in)
			event := in.Calls[0].Envelope.Event.(domain.AssetFailed)
			if event.FailureKind != test.want {
				t.Fatalf("kind = %s, want %s", event.FailureKind, test.want)
			}
		})
	}
}
