package api

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestAssetService_BufconnManifestAndChunkedGet(t *testing.T) {
	data := []byte(strings.Repeat("title-bg", 8193))
	source := &memoryAssetSource{assets: map[string]memoryAsset{
		"ui/title_bg": {info: AssetInfo{Name: "ui/title_bg", SHA256: "abc123", ContentType: "image/webp", Size: int64(len(data))}, data: data},
	}}
	server, err := NewAssetServer(source)
	if err != nil {
		t.Fatalf("NewAssetServer() error = %v", err)
	}
	listener := bufconn.Listen(1 << 20)
	grpcServer := grpc.NewServer()
	df.RegisterAssetServiceServer(grpcServer, server)
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(func() { grpcServer.Stop(); _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer conn.Close()
	client := df.NewAssetServiceClient(conn)
	manifest, err := client.Manifest(context.Background(), &df.AssetManifestRequest{})
	if err != nil || len(manifest.GetAssets()) != 1 {
		t.Fatalf("Manifest() = %v, error = %v", manifest, err)
	}
	if manifest.GetAssets()[0].GetName() != "ui/title_bg" || manifest.GetAssets()[0].GetSize() != uint64(len(data)) {
		t.Fatalf("manifest entry = %v", manifest.GetAssets()[0])
	}
	stream, err := client.Get(context.Background(), &df.AssetRequest{Sha256: "abc123"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	var got []byte
	var chunks int
	var final bool
	var offset uint64
	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatalf("Recv() error = %v", recvErr)
		}
		if chunk.GetOffset() != offset || chunk.GetName() != "ui/title_bg" || chunk.GetContentType() != "image/webp" {
			t.Fatalf("chunk metadata = %v, offset=%d", chunk, offset)
		}
		got = append(got, chunk.GetData()...)
		offset += uint64(len(chunk.GetData()))
		chunks++
		final = chunk.GetFinal()
	}
	if chunks < 2 || !final || !bytes.Equal(got, data) {
		t.Fatalf("stream chunks=%d final=%v bytes=%d want=%d", chunks, final, len(got), len(data))
	}
}

func TestAssetService_RejectsAmbiguousSelector(t *testing.T) {
	server, err := NewAssetServer(&memoryAssetSource{})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Get(&df.AssetRequest{Name: "name", Sha256: "sha"}, &recordingAssetStream{}); err == nil {
		t.Fatal("Get() accepted two selectors")
	}
	if _, err := NewAssetServer(nil); err == nil {
		t.Fatal("NewAssetServer(nil) accepted nil source")
	}
}

type memoryAsset struct {
	info AssetInfo
	data []byte
}

type memoryAssetSource struct{ assets map[string]memoryAsset }

func (s *memoryAssetSource) Manifest(context.Context) ([]AssetInfo, error) {
	assets := make([]AssetInfo, 0, len(s.assets))
	for _, asset := range s.assets {
		assets = append(assets, asset.info)
	}
	return assets, nil
}

func (s *memoryAssetSource) Open(_ context.Context, selector string) (AssetInfo, io.ReadCloser, error) {
	for name, asset := range s.assets {
		if selector == name || selector == asset.info.SHA256 {
			return asset.info, io.NopCloser(bytes.NewReader(asset.data)), nil
		}
	}
	return AssetInfo{}, nil, io.ErrUnexpectedEOF
}

type recordingAssetStream struct{}

func (*recordingAssetStream) SetHeader(metadata.MD) error  { return nil }
func (*recordingAssetStream) SendHeader(metadata.MD) error { return nil }
func (*recordingAssetStream) SetTrailer(metadata.MD)       {}
func (*recordingAssetStream) Context() context.Context     { return context.Background() }
func (*recordingAssetStream) SendMsg(any) error            { return nil }
func (*recordingAssetStream) RecvMsg(any) error            { return io.EOF }
func (*recordingAssetStream) Send(*df.AssetChunk) error    { return nil }
