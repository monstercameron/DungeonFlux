package splat

import (
	"context"
	"errors"
)

const (
	// ReportReady identifies a probe that found a usable splat renderer.
	ReportReady = "SPLAT_READY"
	// ReportFailed identifies a probe that must fall back to the flat renderer.
	ReportFailed = "SPLAT_FAILED"
)

var errProbeClosed = errors.New("splat probe event stream closed")

// EventSource is the browser-facing subset of Bridge used by Probe.
type EventSource interface {
	Init(Init) error
	Events() <-chan Event
}

// Report contains the result sent to the session report RPC.
type Report struct {
	Kind      string
	FPSP5     float64
	Gaussians int
	Device    string
	Code      string
	Detail    string
}

// Reporter receives one terminal result for a probe run.
type Reporter interface {
	Report(context.Context, Report) error
}

// Probe coordinates the hidden battlefield render and its readiness report.
type Probe struct {
	source   EventSource
	reporter Reporter
}

// NewProbe creates a probe using a browser event source and session reporter.
func NewProbe(source EventSource, reporter Reporter) *Probe {
	return &Probe{source: source, reporter: reporter}
}

// Run initializes the hidden renderer and waits for a passing FPS sample or a
// terminal browser error. The JavaScript module performs the 500k-to-100k
// downgrade before emitting a second ready event.
func (p *Probe) Run(ctx context.Context, init Init) (Report, error) {
	if p == nil || p.source == nil || p.reporter == nil {
		return Report{}, errors.New("splat probe dependencies are required")
	}
	if err := p.source.Init(init); err != nil {
		return Report{}, err
	}
	var candidate Report
	for {
		select {
		case <-ctx.Done():
			return Report{}, ctx.Err()
		case event, ok := <-p.source.Events():
			if !ok {
				return Report{}, errProbeClosed
			}
			result, done := probeEvent(candidate, event)
			if !done {
				candidate = result
				continue
			}
			if err := p.reporter.Report(ctx, result); err != nil {
				return result, err
			}
			return result, nil
		}
	}
}

func probeEvent(candidate Report, event Event) (Report, bool) {
	switch event.Type {
	case "ready":
		return Report{Gaussians: event.Gaussians, Device: event.Device}, false
	case "stats":
		candidate.FPSP5 = event.FPSP5
		if candidate.FPSP5 >= 30 {
			candidate.Kind = ReportReady
			return candidate, true
		}
		return candidate, false
	case "error":
		return Report{Kind: ReportFailed, Code: event.Code, Detail: event.Detail}, true
	default:
		return candidate, false
	}
}
