package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestAssetService_BufconnManifestAndChunkedGet(t *testing.T) {
	data := []byte(strings.Repeat("title-bg", 8193))
	digest := sha256.Sum256(data)
	sha := hex.EncodeToString(digest[:])
	source := &memoryAssetSource{assets: map[string]memoryAsset{
		"ui/title_bg": {info: AssetInfo{Name: "ui/title_bg", SHA256: sha, ContentType: "image/webp", Size: int64(len(data))}, data: data},
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
	stream, err := client.Get(context.Background(), &df.AssetRequest{Sha256: sha})
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

func TestAssetService_RuntimeSHAStreamsWithExtensionContentType(t *testing.T) {
	assetDir := t.TempDir()
	data := []byte("runtime lobby qr")
	digest := sha256.Sum256(data)
	sha := hex.EncodeToString(digest[:])
	if err := os.WriteFile(filepath.Join(assetDir, sha+".png"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	server, err := NewAssetServer(&memoryAssetSource{}, assetDir)
	if err != nil {
		t.Fatal(err)
	}
	stream := &recordingAssetStream{}
	if err := server.Get(&df.AssetRequest{Sha256: sha}, stream); err != nil {
		t.Fatalf("Get(runtime) error = %v", err)
	}
	if len(stream.chunks) < 1 || !bytes.Equal(streamData(stream.chunks), data) {
		t.Fatalf("runtime chunks = %#v", stream.chunks)
	}
	chunk := stream.chunks[0]
	if chunk.GetSha256() != sha || chunk.GetName() != sha || chunk.GetContentType() != "image/png" || !stream.chunks[len(stream.chunks)-1].GetFinal() {
		t.Fatalf("runtime metadata = %v", chunk)
	}
}

func streamData(chunks []*df.AssetChunk) []byte {
	var data []byte
	for _, chunk := range chunks {
		data = append(data, chunk.GetData()...)
	}
	return data
}

func TestAssetService_RuntimeSHAReportsMissAndRejectsBadSelector(t *testing.T) {
	server, err := NewAssetServer(&memoryAssetSource{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	missing := strings.Repeat("b", 64)
	if err := server.Get(&df.AssetRequest{Sha256: missing}, &recordingAssetStream{}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing runtime asset code = %v, want %v", status.Code(err), codes.NotFound)
	}
	if err := server.Get(&df.AssetRequest{Sha256: "../" + missing}, &recordingAssetStream{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad runtime selector code = %v, want %v", status.Code(err), codes.InvalidArgument)
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

type recordingAssetStream struct{ chunks []*df.AssetChunk }

func (*recordingAssetStream) SetHeader(metadata.MD) error  { return nil }
func (*recordingAssetStream) SendHeader(metadata.MD) error { return nil }
func (*recordingAssetStream) SetTrailer(metadata.MD)       {}
func (*recordingAssetStream) Context() context.Context     { return context.Background() }
func (*recordingAssetStream) SendMsg(any) error            { return nil }
func (*recordingAssetStream) RecvMsg(any) error            { return io.EOF }
func (s *recordingAssetStream) Send(chunk *df.AssetChunk) error {
	s.chunks = append(s.chunks, chunk)
	return nil
}
