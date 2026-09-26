package debug

import (
	"context"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// snapshotController is the optional engine capability used by Snapshot.
// Keeping it separate from ports.Engine preserves the read-only engine port.
type snapshotController interface {
	DebugSnapshot(context.Context, string, []byte) ([]byte, error)
}

// vendorController is the optional runtime capability used by Vendor.
type vendorController interface {
	DebugVendor(context.Context, string, string) (string, error)
}

// clientController is the optional API capability used by Client.
type clientController interface {
	DebugClient(context.Context, string, string, string) (string, error)
}

// Snapshot saves or loads a deterministic engine snapshot through the debug
// controller installed by the runtime. The request data is the snapshot name
// for the CLI's save/load operations.
func (s *Server) Snapshot(ctx context.Context, request *df.SnapshotRequest) (*df.SnapshotResponse, error) {
	if request == nil || request.GetOperation() == "" {
		return nil, status.Error(codes.InvalidArgument, "snapshot operation is required")
	}
	controller, ok := s.engine.(snapshotController)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "snapshot controller is not configured")
	}
	data, err := controller.DebugSnapshot(ctx, request.GetOperation(), request.GetData())
	if err != nil {
		return &df.SnapshotResponse{Reason: err.Error()}, nil
	}
	return &df.SnapshotResponse{Ok: true, Data: data}, nil
}

// Vendor changes a fake vendor fault through the optional runtime controller.
func (s *Server) Vendor(ctx context.Context, request *df.VendorRequest) (*df.VendorResponse, error) {
	if request == nil || request.GetOperation() == "" {
		return nil, status.Error(codes.InvalidArgument, "vendor operation is required")
	}
	controller, ok := s.engine.(vendorController)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "vendor controller is not configured")
	}
	result, err := controller.DebugVendor(ctx, request.GetOperation(), request.GetPayloadJson())
	if err != nil {
		return &df.VendorResponse{Reason: err.Error()}, nil
	}
	return &df.VendorResponse{Ok: true, ResultJson: result}, nil
}

// Client routes a command to a connected debug client through the optional
// client registry. Screenshot paths are returned by that registry.
func (s *Server) Client(ctx context.Context, request *df.ClientRequest) (*df.ClientResponse, error) {
	if request == nil || request.GetClientId() == "" || request.GetVerb() == "" {
		return nil, status.Error(codes.InvalidArgument, "client id and verb are required")
	}
	controller, ok := s.engine.(clientController)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "client controller is not configured")
	}
	path, err := controller.DebugClient(ctx, request.GetClientId(), request.GetVerb(), request.GetArg())
	if err != nil {
		return &df.ClientResponse{Reason: err.Error()}, nil
	}
	return &df.ClientResponse{Ok: true, Path: path}, nil
}
