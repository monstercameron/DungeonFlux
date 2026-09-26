package main

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

const (
	watchBufferSize = 32
	minWatchRetry   = 100 * time.Millisecond
	maxWatchRetry   = 2 * time.Second
)

// UnaryResult carries either a response or the error from an asynchronous RPC.
type UnaryResult[T any] struct {
	Value T
	Err   error
}

// WatchResult carries one server message or a terminal watch error.
type WatchResult struct {
	Message *dungeonfluxv1.WatchMessage
	Err     error
}

// Client is the shell's asynchronous gRPC client over the bridge tunnel.
type Client struct {
	conn    *grpc.ClientConn
	session sessionClient
}

type sessionClient interface {
	Join(context.Context, *dungeonfluxv1.JoinRequest, ...grpc.CallOption) (*dungeonfluxv1.JoinResponse, error)
	Act(context.Context, *dungeonfluxv1.ActRequest, ...grpc.CallOption) (*dungeonfluxv1.ActResponse, error)
	Say(context.Context, *dungeonfluxv1.SayRequest, ...grpc.CallOption) (*dungeonfluxv1.SayResponse, error)
	Watch(context.Context, *dungeonfluxv1.WatchRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.WatchMessage], error)
}

// NewClient opens a non-blocking gRPC connection to the bridge endpoint.
func NewClient(ctx context.Context, endpoint string) (*Client, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, errors.New("shell client: endpoint is required")
	}
	bridgeURL, err := BridgeWebSocketURL(endpoint)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient(bridgeURL, transportDialOption(bridgeURL))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, session: dungeonfluxv1.NewSessionServiceClient(conn)}, nil
}

// Close releases the underlying bridge connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Join starts Join without blocking the caller, which keeps UI callbacks short.
func (c *Client) Join(ctx context.Context, request *dungeonfluxv1.JoinRequest) <-chan UnaryResult[*dungeonfluxv1.JoinResponse] {
	result := make(chan UnaryResult[*dungeonfluxv1.JoinResponse], 1)
	go func() {
		value, err := c.session.Join(ctx, request)
		result <- UnaryResult[*dungeonfluxv1.JoinResponse]{Value: value, Err: err}
	}()
	return result
}

// Act starts Act without blocking the caller, which keeps UI callbacks short.
func (c *Client) Act(ctx context.Context, request *dungeonfluxv1.ActRequest) <-chan UnaryResult[*dungeonfluxv1.ActResponse] {
	result := make(chan UnaryResult[*dungeonfluxv1.ActResponse], 1)
	go func() {
		value, err := c.session.Act(ctx, request)
		result <- UnaryResult[*dungeonfluxv1.ActResponse]{Value: value, Err: err}
	}()
	return result
}

// Say starts Say without blocking the caller, which keeps UI callbacks short.
func (c *Client) Say(ctx context.Context, request *dungeonfluxv1.SayRequest) <-chan UnaryResult[*dungeonfluxv1.SayResponse] {
	result := make(chan UnaryResult[*dungeonfluxv1.SayResponse], 1)
	go func() {
		value, err := c.session.Say(ctx, request)
		result <- UnaryResult[*dungeonfluxv1.SayResponse]{Value: value, Err: err}
	}()
	return result
}

// Watch starts a reconnecting watch stream. The returned channel is closed when ctx ends.
func (c *Client) Watch(ctx context.Context, request *dungeonfluxv1.WatchRequest) <-chan WatchResult {
	results := make(chan WatchResult, watchBufferSize)
	go c.watchLoop(ctx, request, results)
	return results
}

func (c *Client) watchLoop(ctx context.Context, request *dungeonfluxv1.WatchRequest, results chan<- WatchResult) {
	defer close(results)
	delay := minWatchRetry
	for ctx.Err() == nil {
		stream, err := c.session.Watch(ctx, request)
		if err == nil {
			delay = minWatchRetry
			err = c.receiveWatch(ctx, stream, results)
		}
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			if !sendWatchResult(ctx, results, WatchResult{Err: err}) {
				return
			}
		}
		if !waitWatchRetry(ctx, delay) {
			return
		}
		delay *= 2
		if delay > maxWatchRetry {
			delay = maxWatchRetry
		}
	}
}

func (c *Client) receiveWatch(ctx context.Context, stream grpc.ServerStreamingClient[dungeonfluxv1.WatchMessage], results chan<- WatchResult) error {
	for ctx.Err() == nil {
		message, err := stream.Recv()
		if err != nil {
			return err
		}
		if !sendWatchResult(ctx, results, WatchResult{Message: message}) {
			return ctx.Err()
		}
	}
	return ctx.Err()
}

func sendWatchResult(ctx context.Context, results chan<- WatchResult, result WatchResult) bool {
	select {
	case results <- result:
		return true
	case <-ctx.Done():
		return false
	}
}

func waitWatchRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// BridgeWebSocketURL converts an HTTP bridge URL into the browser WebSocket URL.
func BridgeWebSocketURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", errors.New("shell client: bridge URL must use http(s) or ws(s)")
	}
	if parsed.Path == "" {
		parsed.Path = "/grpc"
	}
	return parsed.String(), nil
}
