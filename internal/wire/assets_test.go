package wire

import (
	"context"
	"io"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestAssetService_LiveServerFetchesUITitleBackground(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	buildtime := filepath.Join(root, "artifacts", "runtime", "buildtime")
	assetDir := filepath.Join(buildtime, "assets")
	if err := os.MkdirAll(assetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("c", 64)
	fileData := []byte("live title background")
	if err := os.WriteFile(filepath.Join(assetDir, sha+".webp"), fileData, 0o600); err != nil {
		t.Fatal(err)
	}
	manifestJSON := `{"version":1,"assets":{"ui/title_bg":{"kind":"IMAGE","selected":1,"takes":[{"number":1,"sha256":"` + sha + `","path":"assets/` + sha + `.webp"}]}}}`
	if err := os.WriteFile(filepath.Join(buildtime, "manifest.json"), []byte(manifestJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := Build(context.Background(), testConfig(t), []byte("asset-e2e"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() { _ = app.Close() }()
	httpServer := httptest.NewServer(app.Handler())
	defer httpServer.Close()
	conn, err := grpctunnel.Dial(httpServer.URL+"/grpc", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpctunnel.Dial() error = %v", err)
	}
	defer conn.Close()
	client := df.NewAssetServiceClient(conn)
	manifest, err := client.Manifest(context.Background(), &df.AssetManifestRequest{})
	if err != nil {
		t.Fatalf("Manifest() error = %v", err)
	}
	var title *df.AssetManifestEntry
	for _, entry := range manifest.GetAssets() {
		if entry.GetName() == "ui/title_bg" {
			title = entry
			break
		}
	}
	if title == nil || title.GetSha256() == "" || title.GetSize() == 0 {
		t.Fatalf("title background missing from manifest: %v", manifest.GetAssets())
	}
	stream, err := client.Get(context.Background(), &df.AssetRequest{Sha256: title.GetSha256()})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	var data []byte
	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatalf("Get().Recv() error = %v", recvErr)
		}
		data = append(data, chunk.GetData()...)
	}
	if uint64(len(data)) != title.GetSize() {
		t.Fatalf("title background bytes = %d, manifest size = %d", len(data), title.GetSize())
	}
}

func TestAssetCatalog_BufconnClientFetchesUITitleBackground(t *testing.T) {
	root := t.TempDir()
	buildtime := filepath.Join(root, "buildtime")
	assetDir := filepath.Join(buildtime, "assets")
	if err := os.MkdirAll(assetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 64)
	data := []byte("title background")
	if err := os.WriteFile(filepath.Join(assetDir, sha+".webp"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":1,"assets":{"ui/title_bg":{"kind":"IMAGE","selected":1,"takes":[{"number":1,"sha256":"` + sha + `","path":"assets/` + sha + `.webp"}]}}}`
	manifestPath := filepath.Join(buildtime, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	index := &recordingAssetIndex{}
	catalog, err := loadAssetCatalog(context.Background(), filepath.Join(root, "runtime"), manifestPath, index, nil)
	if err != nil {
		t.Fatalf("loadAssetCatalog() error = %v", err)
	}
	info, reader, err := catalog.Open(context.Background(), "ui/title_bg")
	if err != nil {
		t.Fatalf("catalog.Open() error = %v", err)
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(got) != string(data) || info.ContentType != "image/webp" {
		t.Fatalf("catalog asset = %#v, data=%q, error=%v", info, got, err)
	}
	if len(index.assets) != 1 || index.assets[0].ID != "ui/title_bg" || index.assets[0].SHA256 != sha {
		t.Fatalf("indexed assets = %#v", index.assets)
	}
	if _, err := os.Stat(filepath.Join(root, "runtime", "assets", sha+".webp")); err != nil {
		t.Fatalf("runtime asset was not copied: %v", err)
	}
	service, err := api.NewAssetServer(catalog)
	if err != nil {
		t.Fatal(err)
	}
	listener := bufconn.Listen(1 << 20)
	grpcServer := grpc.NewServer()
	df.RegisterAssetServiceServer(grpcServer, service)
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(func() { grpcServer.Stop(); _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer conn.Close()
	stream, err := df.NewAssetServiceClient(conn).Get(context.Background(), &df.AssetRequest{Name: "ui/title_bg"})
	if err != nil {
		t.Fatalf("AssetService.Get() error = %v", err)
	}
	var streamed []byte
	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatalf("AssetService.Get().Recv() error = %v", recvErr)
		}
		streamed = append(streamed, chunk.GetData()...)
	}
	if string(streamed) != string(data) {
		t.Fatalf("streamed asset = %q, want %q", streamed, data)
	}
}

type recordingAssetIndex struct{ assets []domain.Asset }

func (i *recordingAssetIndex) Put(_ context.Context, asset domain.Asset) error {
	i.assets = append(i.assets, asset)
	return nil
}

func (*recordingAssetIndex) Get(context.Context, string) (domain.Asset, bool, error) {
	return domain.Asset{}, false, nil
}

func TestAssetCatalog_MissingManifestIsEmpty(t *testing.T) {
	catalog, err := loadAssetCatalog(context.Background(), t.TempDir(), filepath.Join(t.TempDir(), "missing.json"), nil, nil)
	if err != nil {
		t.Fatalf("loadAssetCatalog() error = %v", err)
	}
	assets, err := catalog.Manifest(context.Background())
	if err != nil || len(assets) != 0 {
		t.Fatalf("Manifest() = %#v, error=%v", assets, err)
	}
}

func TestAssetCatalog_RejectsManifestPathEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside.bin")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "manifest.json")
	manifest := `{"version":1,"assets":{"bad":{"kind":"BIN","selected":1,"takes":[{"number":1,"sha256":"` + strings.Repeat("b", 64) + `","path":"../outside.bin"}]}}}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAssetCatalog(context.Background(), filepath.Join(root, "runtime"), manifestPath, nil, nil); err == nil {
		t.Fatal("loadAssetCatalog() accepted an escaping path")
	}
}
