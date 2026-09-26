package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type reportInbox struct {
	accepted bool
	events   []domain.Event
}

func (r *reportInbox) Post(_ context.Context, env domain.Envelope) bool {
	if !r.accepted {
		return false
	}
	r.events = append(r.events, env.Event)
	return true
}

func TestReportServer_ReportPostsGameplayEvents(t *testing.T) {
	inbox := &reportInbox{accepted: true}
	server, err := NewReportServer(inbox)
	if err != nil {
		t.Fatalf("NewReportServer() error = %v", err)
	}
	for _, kind := range []df.ReportKind{df.ReportKind_REPORT_KIND_PLAYBACK_DONE, df.ReportKind_REPORT_KIND_CLIP_ENDED, df.ReportKind_REPORT_KIND_SPLAT_READY, df.ReportKind_REPORT_KIND_SPLAT_FAILED} {
		_, err = server.Report(context.Background(), &df.ReportRequest{Kind: kind, Id: "asset-1"})
		if err != nil {
			t.Fatalf("Report(%v) error = %v", kind, err)
		}
	}
	if len(inbox.events) != 4 {
		t.Fatalf("event count = %d, want 4", len(inbox.events))
	}
	if event, ok := inbox.events[2].(domain.Report); !ok || event.ReportKind != vocab.ReportSplatReady || event.ID != "asset-1" {
		t.Fatalf("event = %#v, want splat ready report", inbox.events[2])
	}
}

func TestReportServer_ClientDiagnosticsStayLocal(t *testing.T) {
	server, err := NewReportServer(&reportInbox{accepted: true})
	if err != nil {
		t.Fatalf("NewReportServer() error = %v", err)
	}
	_, err = server.Report(context.Background(), &df.ReportRequest{SeatToken: "seat", Kind: df.ReportKind_REPORT_KIND_CLIENT_SCREENSHOT, Id: "shot", ClientState: "ready", Png: []byte{1, 2}})
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	reports := server.ClientReports()
	if len(reports) != 1 || reports[0].Kind != vocab.ReportClientScreenshot || string(reports[0].PNG) != string([]byte{1, 2}) {
		t.Fatalf("reports = %#v", reports)
	}
	reports[0].PNG[0] = 9
	if server.ClientReports()[0].PNG[0] != 1 {
		t.Fatal("ClientReports() did not copy PNG")
	}
}

func TestReportServer_RejectsInvalidAndFullInbox(t *testing.T) {
	server, err := NewReportServer(&reportInbox{})
	if err != nil {
		t.Fatalf("NewReportServer() error = %v", err)
	}
	if _, err := server.Report(context.Background(), nil); err == nil {
		t.Fatal("nil request error = nil")
	}
	if _, err := server.Report(context.Background(), &df.ReportRequest{}); err == nil {
		t.Fatal("invalid kind error = nil")
	}
	if _, err := server.Report(context.Background(), &df.ReportRequest{Kind: df.ReportKind_REPORT_KIND_SPLAT_READY}); err == nil {
		t.Fatal("full inbox error = nil")
	}
	if _, err := NewReportServer(nil); err == nil {
		t.Fatal("nil inbox error = nil")
	}
}
