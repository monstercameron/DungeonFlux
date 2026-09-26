package splat

import (
	"context"
	"errors"
	"testing"
)

type probeSource struct {
	events  chan Event
	init    Init
	initErr error
}

func (s *probeSource) Init(value Init) error {
	s.init = value
	return s.initErr
}

func (s *probeSource) Events() <-chan Event { return s.events }

type probeReporter struct {
	reports []Report
	err     error
}

func (r *probeReporter) Report(_ context.Context, report Report) error {
	r.reports = append(r.reports, report)
	return r.err
}

func TestProbe_RunReportsReadyAfterPassingStats(t *testing.T) {
	source := &probeSource{events: make(chan Event, 2)}
	reporter := &probeReporter{}
	probe := NewProbe(source, reporter)
	init := Init{CanvasID: "df-splat", SceneURL: "full.sog"}
	source.events <- Event{Type: "ready", Gaussians: 500000, Device: "webgl2"}
	source.events <- Event{Type: "stats", FPSP5: 45.5}

	got, err := probe.Run(context.Background(), init)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ReportReady || got.Gaussians != 500000 || got.FPSP5 != 45.5 {
		t.Fatalf("result = %+v", got)
	}
	if len(reporter.reports) != 1 || reporter.reports[0] != got {
		t.Fatalf("reports = %+v", reporter.reports)
	}
	if source.init.CanvasID != "df-splat" {
		t.Fatalf("init = %+v", source.init)
	}
}

func TestProbe_RunWaitsThroughLowFPSAndUsesLiteReady(t *testing.T) {
	source := &probeSource{events: make(chan Event, 4)}
	reporter := &probeReporter{}
	probe := NewProbe(source, reporter)
	source.events <- Event{Type: "ready", Gaussians: 500000}
	source.events <- Event{Type: "stats", FPSP5: 24}
	source.events <- Event{Type: "ready", Gaussians: 100000, Device: "webgl2"}
	source.events <- Event{Type: "stats", FPSP5: 38}

	got, err := probe.Run(context.Background(), Init{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ReportReady || got.Gaussians != 100000 || got.FPSP5 != 38 {
		t.Fatalf("result = %+v", got)
	}
}

func TestProbe_RunReportsFailure(t *testing.T) {
	source := &probeSource{events: make(chan Event, 1)}
	reporter := &probeReporter{}
	probe := NewProbe(source, reporter)
	source.events <- Event{Type: "error", Code: "LOW_FPS", Detail: "5th-percentile FPS stayed below 30"}

	got, err := probe.Run(context.Background(), Init{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ReportFailed || got.Code != "LOW_FPS" {
		t.Fatalf("result = %+v", got)
	}
}

func TestProbe_RunValidationAndCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source := &probeSource{events: make(chan Event)}
	reporter := &probeReporter{}
	if _, err := NewProbe(source, reporter).Run(ctx, Init{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
	if _, err := (&Probe{}).Run(context.Background(), Init{}); err == nil {
		t.Fatal("expected missing dependency error")
	}
	if _, err := NewProbe(&probeSource{events: make(chan Event), initErr: errors.New("init failed")}, reporter).Run(context.Background(), Init{}); err == nil {
		t.Fatal("expected init error")
	}
}

func TestProbe_RunReportsReporterError(t *testing.T) {
	source := &probeSource{events: make(chan Event, 1)}
	reporter := &probeReporter{err: errors.New("report failed")}
	source.events <- Event{Type: "error", Code: "LOAD_FAILED"}
	_, err := NewProbe(source, reporter).Run(context.Background(), Init{})
	if err == nil || err.Error() != "report failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestProbeEvent_ignoresUnknownAndTracksStats(t *testing.T) {
	candidate, done := probeEvent(Report{}, Event{Type: "unknown"})
	if done || candidate != (Report{}) {
		t.Fatalf("unknown = %+v, %v", candidate, done)
	}
	candidate, done = probeEvent(Report{Gaussians: 100000}, Event{Type: "stats", FPSP5: 29.9})
	if done || candidate.FPSP5 != 29.9 {
		t.Fatalf("low stats = %+v, %v", candidate, done)
	}
}
