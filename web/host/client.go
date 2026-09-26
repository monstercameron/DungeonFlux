package host

import (
	"context"
	"errors"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

type hostCommandClient interface {
	Command(context.Context, *df.HostCommand, ...grpc.CallOption) (*df.HostAck, error)
}

type hostClient struct {
	conn    *grpc.ClientConn
	service hostCommandClient
}

func newHostClient(endpoint string) (*hostClient, error) {
	if endpoint == "" {
		return nil, errors.New("host: endpoint is required")
	}
	conn, err := grpc.NewClient(endpoint, transportDialOption(endpoint))
	if err != nil {
		return nil, err
	}
	return &hostClient{conn: conn, service: df.NewHostServiceClient(conn)}, nil
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
