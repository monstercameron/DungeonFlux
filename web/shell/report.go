package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

const clientReportInterval = 2 * time.Second

// ClientState is the diagnostic snapshot sent by a shell client.
type ClientState struct {
	Route           string  `json:"route,omitempty"`
	Seat            string  `json:"seat,omitempty"`
	ViewVersion     uint64  `json:"view_version,omitempty"`
	AudioQueueDepth int     `json:"audio_queue_depth,omitempty"`
	SplatMode       string  `json:"splat_mode,omitempty"`
	FPSP5           float64 `json:"fps_p5,omitempty"`
	LastError       string  `json:"last_error,omitempty"`
	Active          bool    `json:"active"`
}

// MarshalClientState encodes a diagnostic snapshot for ReportRequest.
func MarshalClientState(state ClientState) (string, error) {
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", errors.Join(errors.New("shell report: encode client state"), err)
	}
	return string(encoded), nil
}

type reportTransport interface {
	report(context.Context, *dungeonfluxv1.ReportRequest) error
}

type clientReportTransport struct{ client *Client }

func (t clientReportTransport) report(ctx context.Context, request *dungeonfluxv1.ReportRequest) error {
	if t.client == nil || t.client.session == nil {
		return errors.New("shell report: client is not connected")
	}
	reporter, ok := t.client.session.(interface {
		Report(context.Context, *dungeonfluxv1.ReportRequest, ...grpc.CallOption) (*dungeonfluxv1.ReportResponse, error)
	})
	if !ok {
		return errors.New("shell report: session does not support reports")
	}
	_, err := reporter.Report(ctx, request)
	return err
}

// ClientReporter sends client health snapshots to the server.
type ClientReporter struct {
	transport reportTransport
	seatToken string
	clientID  string

	mu     sync.RWMutex
	state  ClientState
	active bool
	events chan struct{}
}

// NewClientReporter creates a reporter associated with a shell client.
func NewClientReporter(client *Client, seatToken, clientID string) *ClientReporter {
	return &ClientReporter{
		transport: clientReportTransport{client: client},
		seatToken: seatToken,
		clientID:  clientID,
		events:    make(chan struct{}, 8),
	}
}

// Start runs periodic reporting until ctx is cancelled.
func (r *ClientReporter) Start(ctx context.Context) {
	if r == nil {
		return
	}
	go r.loop(ctx)
}

// SetActive enables or disables the two-second reporting interval.
func (r *ClientReporter) SetActive(active bool) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.active = active
	r.state.Active = active
	r.mu.Unlock()
}

// Update replaces the snapshot used by the next periodic report.
func (r *ClientReporter) Update(state ClientState) {
	if r == nil {
		return
	}
	r.mu.Lock()
	state.Active = r.active
	r.state = state
	r.mu.Unlock()
}

// ReportError records an error and schedules an immediate client-state report.
func (r *ClientReporter) ReportError(err error) {
	if r == nil || err == nil {
		return
	}
	r.mu.Lock()
	r.state.LastError = err.Error()
	r.state.Active = r.active
	r.mu.Unlock()
	r.enqueue()
}

// ReportScreenshot sends a best-effort PNG report for a debug client command.
// The caller owns capture and supplies the command ID so the API can match the
// response to dfctl client screenshot.
func (r *ClientReporter) ReportScreenshot(ctx context.Context, commandID string, png []byte) error {
	if r == nil || commandID == "" {
		return errors.New("shell report: screenshot command ID is required")
	}
	return r.transport.report(ctx, &dungeonfluxv1.ReportRequest{
		SeatToken: r.seatToken,
		Kind:      dungeonfluxv1.ReportKind_REPORT_KIND_CLIENT_SCREENSHOT,
		Id:        commandID,
		Png:       append([]byte(nil), png...),
	})
}

func (r *ClientReporter) loop(ctx context.Context) {
	ticker := time.NewTicker(clientReportInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if r.isActive() {
				_ = r.send(ctx)
			}
		case <-r.events:
			_ = r.send(ctx)
		}
	}
}

func (r *ClientReporter) enqueue() {
	select {
	case r.events <- struct{}{}:
	default:
	}
}

func (r *ClientReporter) isActive() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active
}

func (r *ClientReporter) send(ctx context.Context) error {
	r.mu.RLock()
	state := r.state
	r.mu.RUnlock()
	encoded, err := MarshalClientState(state)
	if err != nil {
		return err
	}
	return r.transport.report(ctx, &dungeonfluxv1.ReportRequest{
		SeatToken:   r.seatToken,
		Kind:        dungeonfluxv1.ReportKind_REPORT_KIND_CLIENT_STATE,
		Id:          r.clientID,
		ClientState: encoded,
	})
}
