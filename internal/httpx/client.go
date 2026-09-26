package httpx

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"
)

// CallStats contains transport timing measurements for one HTTP call.
type CallStats struct {
	Vendor    string
	URL       string
	Status    int
	Bytes     int64
	DNSMS     float64
	ConnectMS float64
	TLSMS     float64
	TTFTMS    float64
	DurMS     float64
	RequestID string
}

// Recorder receives one record for each completed or failed HTTP call.
type Recorder func(CallStats)

// Client is a vendor-scoped HTTP client with trace timing and call logging.
type Client struct {
	vendor   string
	http     *http.Client
	logger   *slog.Logger
	recorder Recorder
}

// NewClient returns a keep-alive HTTP/2-capable client for vendor.
func NewClient(vendor string, timeout time.Duration, logger *slog.Logger) *Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{vendor: vendor, http: &http.Client{Transport: transport, Timeout: timeout}, logger: logger}
}

// SetRecorder installs a callback for one-call records. It is intended for
// composition roots and tests, before the client is used concurrently.
func (c *Client) SetRecorder(recorder Recorder) { c.recorder = recorder }

// HTTPClient returns the underlying client for SDKs that accept *http.Client.
func (c *Client) HTTPClient() *http.Client { return c.http }

// Do executes req and returns its response and transport statistics. The
// response body must be closed by the caller; closing it records completion.
func (c *Client) Do(req *http.Request) (*http.Response, CallStats, error) {
	return c.do(req)
}

func (c *Client) do(req *http.Request) (*http.Response, CallStats, error) {
	started := time.Now()
	traceState := newTraceState(started)
	request := req.WithContext(httptrace.WithClientTrace(req.Context(), traceState.trace()))
	response, err := c.http.Do(request)
	if err != nil {
		stats := traceState.stats(c.vendor, req.URL.String(), 0, 0, time.Since(started))
		c.record(stats, err)
		return nil, stats, err
	}
	stats := traceState.stats(c.vendor, req.URL.String(), response.StatusCode, 0, time.Since(started))
	stats.RequestID = response.Header.Get("x-request-id")
	response.Body = &recordingBody{ReadCloser: response.Body, client: c, stats: stats, start: started}
	return response, stats, nil
}

func (c *Client) record(stats CallStats, err error) {
	if c.recorder != nil {
		c.recorder(stats)
	}
	if c.logger == nil {
		return
	}
	attrs := []any{"vendor", stats.Vendor, "url", stats.URL, "status", stats.Status, "ttft_ms", stats.TTFTMS, "dur_ms", stats.DurMS, "bytes", stats.Bytes}
	if stats.RequestID != "" {
		attrs = append(attrs, "request_id", stats.RequestID)
	}
	if err != nil {
		attrs = append(attrs, "err", err)
	}
	c.logger.Info("call", attrs...)
}

type recordingBody struct {
	io.ReadCloser
	client *Client
	stats  CallStats
	start  time.Time
	once   sync.Once
}

func (b *recordingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.stats.Bytes += int64(n)
	if err == io.EOF {
		b.finish()
	}
	return n, err
}

func (b *recordingBody) Close() error {
	err := b.ReadCloser.Close()
	b.finish()
	return err
}

func (b *recordingBody) finish() {
	b.once.Do(func() {
		b.stats.DurMS = durationMS(time.Since(b.start))
		b.client.record(b.stats, nil)
	})
}

func durationMS(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

type traceState struct {
	started, dnsStart, dnsDone, connectStart, connectDone, tlsStart, tlsDone, firstByte time.Time
}

func newTraceState(start time.Time) *traceState { return &traceState{started: start} }

func (s *traceState) trace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { s.dnsStart = time.Now() },
		DNSDone:              func(httptrace.DNSDoneInfo) { s.dnsDone = time.Now() },
		ConnectStart:         func(_, _ string) { s.connectStart = time.Now() },
		ConnectDone:          func(_, _ string, _ error) { s.connectDone = time.Now() },
		TLSHandshakeStart:    func() { s.tlsStart = time.Now() },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { s.tlsDone = time.Now() },
		GotFirstResponseByte: func() { s.firstByte = time.Now() },
	}
}

func (s *traceState) stats(vendor, url string, status int, bytes int64, duration time.Duration) CallStats {
	return CallStats{Vendor: vendor, URL: url, Status: status, Bytes: bytes,
		DNSMS: pairMS(s.dnsStart, s.dnsDone), ConnectMS: pairMS(s.connectStart, s.connectDone),
		TLSMS: pairMS(s.tlsStart, s.tlsDone), TTFTMS: pairMS(s.started, s.firstByte), DurMS: durationMS(duration)}
}

func pairMS(start, end time.Time) float64 {
	if start.IsZero() || end.IsZero() {
		return 0
	}
	return durationMS(end.Sub(start))
}

// NewVendorClient is an explicit alias for composition code that names the
// vendor timeout table at the call site.
func NewVendorClient(vendor string, timeout time.Duration, logger *slog.Logger) *Client {
	return NewClient(vendor, timeout, logger)
}

// ContextRequest creates an HTTP request with ctx and method/url.
func ContextRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}
