package api

import (
	"context"
	"errors"
	"io"
	"sort"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const assetChunkSize = 64 * 1024

// AssetInfo describes one immutable asset available to a client.
type AssetInfo struct {
	Name        string
	SHA256      string
	ContentType string
	Size        int64
}

// AssetSource provides metadata and readers for the AssetService.
type AssetSource interface {
	Manifest(context.Context) ([]AssetInfo, error)
	Open(context.Context, string) (AssetInfo, io.ReadCloser, error)
}

// AssetServer implements the gRPC asset transfer and preload manifest APIs.
type AssetServer struct {
	df.UnimplementedAssetServiceServer
	source AssetSource
}

// NewAssetServer creates an AssetService backed by source.
func NewAssetServer(source AssetSource) (*AssetServer, error) {
	if source == nil {
		return nil, errors.New("api: asset source is required")
	}
	return &AssetServer{source: source}, nil
}

// Manifest returns the complete logical-name manifest in stable order.
func (s *AssetServer) Manifest(ctx context.Context, _ *df.AssetManifestRequest) (*df.AssetManifestResponse, error) {
	if s == nil || s.source == nil {
		return nil, status.Error(codes.FailedPrecondition, "asset source is unavailable")
	}
	assets, err := s.source.Manifest(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read asset manifest: %v", err)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Name < assets[j].Name })
	response := &df.AssetManifestResponse{Assets: make([]*df.AssetManifestEntry, 0, len(assets))}
	for _, asset := range assets {
		response.Assets = append(response.Assets, manifestEntry(asset))
	}
	return response, nil
}

// Get streams an asset in bounded 64 KiB chunks selected by name or SHA-256.
func (s *AssetServer) Get(req *df.AssetRequest, stream df.AssetService_GetServer) error {
	if s == nil || s.source == nil {
		return status.Error(codes.FailedPrecondition, "asset source is unavailable")
	}
	if req == nil || (req.GetName() == "" && req.GetSha256() == "") || (req.GetName() != "" && req.GetSha256() != "") {
		return status.Error(codes.InvalidArgument, "exactly one asset selector is required")
	}
	selector := req.GetName()
	if selector == "" {
		selector = req.GetSha256()
	}
	info, reader, err := s.source.Open(stream.Context(), selector)
	if err != nil {
		return status.Errorf(codes.NotFound, "open asset %q: %v", selector, err)
	}
	defer func() { _ = reader.Close() }()
	return sendAsset(stream, info, reader)
}

func sendAsset(stream df.AssetService_GetServer, info AssetInfo, reader io.Reader) error {
	buffer := make([]byte, assetChunkSize)
	var offset int64
	for {
		read, err := reader.Read(buffer)
		if read > 0 {
			chunk := &df.AssetChunk{Name: info.Name, Sha256: info.SHA256, ContentType: info.ContentType,
				Size: uint64(maxInt64(info.Size)), Offset: uint64(offset), Data: append([]byte(nil), buffer[:read]...), Final: err == io.EOF}
			if sendErr := stream.Send(chunk); sendErr != nil {
				return sendErr
			}
			offset += int64(read)
		}
		if err == io.EOF {
			if read == 0 {
				return stream.Send(&df.AssetChunk{Name: info.Name, Sha256: info.SHA256, ContentType: info.ContentType, Size: uint64(maxInt64(info.Size)), Offset: uint64(offset), Final: true})
			}
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "read asset %q: %v", info.Name, err)
		}
	}
}

func manifestEntry(asset AssetInfo) *df.AssetManifestEntry {
	return &df.AssetManifestEntry{Name: asset.Name, Sha256: asset.SHA256, ContentType: asset.ContentType, Size: uint64(maxInt64(asset.Size))}
}

func maxInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
