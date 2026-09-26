package main

import (
	"context"
	"errors"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestMarshalClientState_UsesStableDiagnosticFields(t *testing.T) {
	encoded, err := MarshalClientState(ClientState{Route: "/dm", FPSP5: 47.5, Active: true})
	if err != nil {
		t.Fatalf("MarshalClientState() error = %v", err)
	}
	for _, want := range []string{`"route":"/dm"`, `"fps_p5":47.5`, `"active":true`} {
		if !contains(encoded, want) {
			t.Errorf("encoded state %q does not contain %q", encoded, want)
		}
	}
}

func TestClientReporter_ReportErrorSendsClientState(t *testing.T) {
	fake := &fakeReportTransport{requests: make(chan *dungeonfluxv1.ReportRequest, 1)}
	reporter := newTestReporter(fake)
	reporter.SetActive(true)
	reporter.Update(ClientState{Route: "/dm", ViewVersion: 4})
	reporter.ReportError(errors.New("audio failed"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reporter.Start(ctx)
	request := <-fake.requests
	if request.GetKind() != dungeonfluxv1.ReportKind_REPORT_KIND_CLIENT_STATE {
		t.Fatalf("kind = %v", request.GetKind())
	}
	if request.GetId() != "dm-1" || request.GetSeatToken() != "seat-1" {
		t.Fatalf("request identity = %q/%q", request.GetId(), request.GetSeatToken())
	}
	if !contains(request.GetClientState(), "audio failed") {
		t.Fatalf("client state = %q", request.GetClientState())
	}
}

func TestClientReporter_ReportErrorIgnoresNil(t *testing.T) {
	fake := &fakeReportTransport{requests: make(chan *dungeonfluxv1.ReportRequest, 1)}
	reporter := newTestReporter(fake)
	reporter.ReportError(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reporter.Start(ctx)
	select {
	case request := <-fake.requests:
		t.Fatalf("unexpected request: %v", request)
	default:
	}
}

func TestClientReporter_ReportScreenshotCopiesPNG(t *testing.T) {
	fake := &fakeReportTransport{requests: make(chan *dungeonfluxv1.ReportRequest, 1)}
	reporter := newTestReporter(fake)
	png := []byte{1, 2, 3}
	if err := reporter.ReportScreenshot(context.Background(), "command-1", png); err != nil {
		t.Fatal(err)
	}
	png[0] = 9
	request := <-fake.requests
	if request.GetKind() != dungeonfluxv1.ReportKind_REPORT_KIND_CLIENT_SCREENSHOT || request.GetId() != "command-1" || request.GetPng()[0] != 1 {
		t.Fatalf("screenshot request = %+v", request)
	}
	if err := reporter.ReportScreenshot(context.Background(), "", nil); err == nil {
		t.Fatal("empty command ID accepted")
	}
}

type fakeReportTransport struct {
	requests chan *dungeonfluxv1.ReportRequest
}

func (f *fakeReportTransport) report(_ context.Context, request *dungeonfluxv1.ReportRequest) error {
	f.requests <- request
	return nil
}

func newTestReporter(transport reportTransport) *ClientReporter {
	return &ClientReporter{transport: transport, seatToken: "seat-1", clientID: "dm-1", events: make(chan struct{}, 8)}
}

func contains(value, fragment string) bool {
	for offset := 0; offset+len(fragment) <= len(value); offset++ {
		if value[offset:offset+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
