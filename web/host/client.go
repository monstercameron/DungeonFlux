package host

import (
	"context"
	"errors"
	"github.com/monstercameron/DungeonFlux/web/shell/watch"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

type hostCommandClient interface {
	Command(context.Context, *df.HostCommand, ...grpc.CallOption) (*df.HostAck, error)
}

type hostSessionClient interface {
	Watch(context.Context, *df.WatchRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[df.WatchMessage], error)
}

type hostClient struct {
	recovery watch.Controller
	conn     *grpc.ClientConn
	service  hostCommandClient
	session  hostSessionClient
}

func newHostClient(endpoint string) (*hostClient, error) {
	if endpoint == "" {
		return nil, errors.New("host: endpoint is required")
	}
	conn, err := grpc.NewClient("passthrough:///dungeonflux", transportDialOptions(endpoint)...)
	if err != nil {
		return nil, err
	}
	return &hostClient{conn: conn, service: df.NewHostServiceClient(conn), session: df.NewSessionServiceClient(conn)}, nil
}

func (c *hostClient) close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *hostClient) command(ctx context.Context, request *df.HostCommand) (*df.HostAck, error) {
	return c.service.Command(ctx, request)
}

func (c *hostClient) watch(ctx context.Context, token string) <-chan *df.WatchMessage {
	if c == nil || c.session == nil {
		out := make(chan *df.WatchMessage)
		close(out)
		return out
	}
	return c.recovery.Messages(ctx, c.session, token)
}
