package wire

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
)

func TestMountWeb_ServesPagesAndWasm(t *testing.T) {
	root := t.TempDir()
	wasm := filepath.Join(root, "artifacts", "wasm")
	if err := os.MkdirAll(wasm, 0o700); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "web", "shell", "static"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web", "shell", "static", "index.html"), []byte("host"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wasm, "dungeonflux.wasm"), []byte("wasm"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wasm, "wasm_exec.js"), []byte("js"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "web", "splat", "scenes"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web", "splat", "scenes", "64bb46d5.json"), []byte("scene"), 0o600); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	if err := mountWeb(mux, config.Config{Server: config.ServerConfig{DataDir: filepath.Join(root, "runtime")}}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/dm", "/p", "/p/seat", "/host", "/about"} {
		res := request(mux, path, "")
		if res.Code != http.StatusOK || res.Body.String() != "host" {
			t.Errorf("GET %s = %d %q", path, res.Code, res.Body.String())
		}
	}
	res := request(mux, "/app/dungeonflux.wasm", "")
	if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "application/wasm" || res.Header().Get("Cache-Control") != "no-cache" || res.Header().Get("ETag") == "" {
		t.Fatalf("wasm = %d %q %q", res.Code, res.Body.String(), res.Header().Get("Content-Type"))
	}
	res = request(mux, "/wasm_exec.js", "")
	if res.Code != http.StatusOK || !strings.HasPrefix(res.Header().Get("Content-Type"), "text/javascript") || res.Header().Get("Cache-Control") != "no-cache" || res.Header().Get("ETag") == "" {
		t.Fatalf("loader = %d %q", res.Code, res.Header().Get("Content-Type"))
	}
	res = request(mux, "/splat/scenes/64bb46d5.json", "")
	if res.Code != http.StatusOK || res.Body.String() != "scene" {
		t.Fatalf("splat scene = %d %q", res.Code, res.Body.String())
	}
	for _, path := range []string{"/dm", "/p", "/host"} {
		res := request(mux, path, "")
		if res.Header().Get("Cache-Control") != "no-cache" || res.Header().Get("ETag") == "" {
			t.Errorf("page %s headers = %q, %q", path, res.Header().Get("Cache-Control"), res.Header().Get("ETag"))
		}
	}
}

func TestMountWeb_ServesBrotliAndValidatesAssets(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "web", "shell", "static"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "artifacts", "wasm"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web", "shell", "static", "index.html"), []byte("host"), 0o600); err != nil {
		t.Fatal(err)
	}
	wasm := filepath.Join(root, "artifacts", "wasm", "dungeonflux.wasm")
	if err := os.WriteFile(wasm+".br", []byte("compressed"), 0o600); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	assets := filepath.Join(root, "runtime", "assets")
	if err := os.MkdirAll(assets, 0o700); err != nil {
		t.Fatal(err)
	}
	name := strings.Repeat("a", 64) + ".png"
	if err := os.WriteFile(filepath.Join(assets, name), []byte("asset"), 0o600); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	if err := mountWeb(mux, config.Config{Server: config.ServerConfig{DataDir: filepath.Join(root, "runtime")}}); err != nil {
		t.Fatal(err)
	}
	res := request(mux, "/app/dungeonflux.wasm", "gzip, br")
	if res.Code != http.StatusOK || res.Header().Get("Content-Encoding") != "br" || res.Body.String() != "compressed" {
		t.Fatalf("brotli = %d %q %q", res.Code, res.Header(), res.Body.String())
	}
	match := requestWithETag(mux, "/app/dungeonflux.wasm", "gzip, br", res.Header().Get("ETag"))
	if match.Code != http.StatusNotModified || match.Body.Len() != 0 {
		t.Fatalf("brotli conditional = %d %q", match.Code, match.Body.String())
	}
	res = request(mux, "/assets/"+name, "")
	if res.Code != http.StatusOK || res.Body.String() != "asset" {
		t.Fatalf("asset = %d %q", res.Code, res.Body.String())
	}
	for _, path := range []string{"/assets/not-a-hash.png", "/assets/" + strings.Repeat("a", 64) + ".png/extra", "/assets/secret/extra"} {
		if res := request(mux, path, ""); res.Code != http.StatusNotFound {
			t.Errorf("invalid asset %s = %d", path, res.Code)
		}
	}
}

func TestMountWeb_ServesGzipWhenBrotliIsUnavailable(t *testing.T) {
	root := t.TempDir()
	wasm := filepath.Join(root, "artifacts", "wasm")
	if err := os.MkdirAll(filepath.Join(root, "web", "shell", "static"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wasm, 0o700); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string]string{
		filepath.Join(root, "web", "shell", "static", "index.html"): "host",
		filepath.Join(wasm, "dungeonflux.wasm"):                     "wasm",
		filepath.Join(wasm, "dungeonflux.wasm.gz"):                  "compressed",
	} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	if err := mountWeb(mux, config.Config{}); err != nil {
		t.Fatal(err)
	}
	res := request(mux, "/app/dungeonflux.wasm", "gzip, br; q=0")
	if res.Code != http.StatusOK || res.Header().Get("Content-Encoding") != "gzip" || res.Body.String() != "compressed" {
		t.Fatalf("gzip = %d %q %q", res.Code, res.Header(), res.Body.String())
	}
	if res.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("gzip Vary = %q", res.Header().Get("Vary"))
	}
	match := requestWithETag(mux, "/app/dungeonflux.wasm", "gzip", res.Header().Get("ETag"))
	if match.Code != http.StatusNotModified || match.Body.Len() != 0 {
		t.Fatalf("gzip conditional = %d %q", match.Code, match.Body.String())
	}
}

func request(handler http.Handler, path, encoding string) *httptest.ResponseRecorder {
	return requestWithETag(handler, path, encoding, "")
}

func requestWithETag(handler http.Handler, path, encoding, etag string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept-Encoding", encoding)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

// TestMountWeb_CacheHeaderPolicy checks the three cache classes from plan.md
// §0.4: content-addressed assets get a one-year immutable Cache-Control,
// splat scene media under artifacts/media/supersplat gets a long TTL plus a
// validator so a 304 is still possible, and the mutable app shell (pages,
// wasm, wasm_exec.js, splat JS modules) keeps revalidating on every load so a
// fresh deploy is never masked by a stale cached copy.
func TestMountWeb_CacheHeaderPolicy(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "web", "shell", "static", "index.html"), "host")
	mustWriteFile(t, filepath.Join(root, "artifacts", "wasm", "dungeonflux.wasm"), "wasm")
	mustWriteFile(t, filepath.Join(root, "artifacts", "wasm", "wasm_exec.js"), "js")
	mustWriteFile(t, filepath.Join(root, "web", "splat", "js", "df-splat.mjs"), "module")
	mustWriteFile(t, filepath.Join(root, "web", "splat", "scenes", "scene.json"), "scene")
	mustWriteFile(t, filepath.Join(root, "artifacts", "media", "supersplat", "scene-1", "chunk.webp"), "chunk")

	assetsDir := filepath.Join(root, "runtime", "assets")
	assetFileName := strings.Repeat("b", 64) + ".png"
	mustWriteFile(t, filepath.Join(assetsDir, assetFileName), "image")

	fontFile := filepath.Join(root, "web", "shell", "static", "fonts", "Inter-normal-400-700.woff2")
	mustWriteFile(t, fontFile, "font-bytes")

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	if err := mountWeb(mux, config.Config{Server: config.ServerConfig{DataDir: filepath.Join(root, "runtime")}}); err != nil {
		t.Fatal(err)
	}

	immutable := []string{"/assets/" + assetFileName, "/fonts/Inter-normal-400-700.woff2"}
	for _, path := range immutable {
		res := request(mux, path, "")
		if res.Code != http.StatusOK {
			t.Fatalf("%s = %d", path, res.Code)
		}
		if got := res.Header().Get("Cache-Control"); got != cacheControlImmutable {
			t.Errorf("%s Cache-Control = %q, want %q", path, got, cacheControlImmutable)
		}
		if res.Header().Get("ETag") == "" {
			t.Errorf("%s missing ETag", path)
		}
		match := requestWithETag(mux, path, "", res.Header().Get("ETag"))
		if match.Code != http.StatusNotModified {
			t.Errorf("%s conditional = %d, want 304", path, match.Code)
		}
	}
	if got := request(mux, "/fonts/Inter-normal-400-700.woff2", "").Header().Get("Content-Type"); got != "font/woff2" {
		t.Errorf("font Content-Type = %q, want font/woff2", got)
	}
	mustWriteFile(t, filepath.Join(root, "web", "shell", "static", "fonts", "notes.txt"), "not a font")
	if res := request(mux, "/fonts/notes.txt", ""); res.Code != http.StatusNotFound {
		t.Errorf("fonts non-woff2 file = %d, want 404", res.Code)
	}

	splatMedia := "/artifacts/media/supersplat/scene-1/chunk.webp"
	res := request(mux, splatMedia, "")
	if res.Code != http.StatusOK || res.Body.String() != "chunk" {
		t.Fatalf("%s = %d %q", splatMedia, res.Code, res.Body.String())
	}
	if got := res.Header().Get("Cache-Control"); got != cacheControlSplatMedia {
		t.Errorf("%s Cache-Control = %q, want %q", splatMedia, got, cacheControlSplatMedia)
	}
	if res.Header().Get("ETag") == "" {
		t.Errorf("%s missing ETag", splatMedia)
	}
	match := requestWithETag(mux, splatMedia, "", res.Header().Get("ETag"))
	if match.Code != http.StatusNotModified || match.Body.Len() != 0 {
		t.Errorf("%s conditional = %d %q, want 304", splatMedia, match.Code, match.Body.String())
	}
	if res := request(mux, "/artifacts/media/supersplat/../../../etc/passwd", ""); res.Code == http.StatusOK {
		t.Errorf("supersplat path traversal served: %d", res.Code)
	}

	mustRevalidate := map[string]string{
		"/dm":                      "/dm",
		"/app/dungeonflux.wasm":    "/app/dungeonflux.wasm",
		"/wasm_exec.js":            "/wasm_exec.js",
		"/splat/js/df-splat.mjs":   "/splat/js/df-splat.mjs",
		"/splat/scenes/scene.json": "/splat/scenes/scene.json",
	}
	for name, path := range mustRevalidate {
		res := request(mux, path, "")
		if res.Code != http.StatusOK {
			t.Fatalf("%s (%s) = %d", name, path, res.Code)
		}
		if got := res.Header().Get("Cache-Control"); got != cacheControlNoCache {
			t.Errorf("%s (%s) Cache-Control = %q, want %q", name, path, got, cacheControlNoCache)
		}
	}

	rejectedMethods := []string{
		"/dm",
		"/wasm_exec.js",
		"/app/dungeonflux.wasm",
		"/assets/" + assetFileName,
		"/fonts/Inter-normal-400-700.woff2",
		splatMedia,
	}
	for _, path := range rejectedMethods {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s = %d, want 405", path, rec.Code)
		}
	}

	if res := request(mux, "/assets/"+strings.Repeat("c", 64)+".png", ""); res.Code != http.StatusNotFound {
		t.Errorf("missing asset = %d, want 404", res.Code)
	}
	if res := request(mux, "/artifacts/media/supersplat/scene-1/missing.webp", ""); res.Code != http.StatusNotFound {
		t.Errorf("missing splat media = %d, want 404", res.Code)
	}
}

func TestServeFileWithCache_RejectsDirectoriesAndMissingFiles(t *testing.T) {
	dir := t.TempDir()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	serveFileWithCache(rec, req, dir, "text/plain", cacheControlNoCache)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("directory = %d, want 404", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	rec = httptest.NewRecorder()
	serveFileWithCache(rec, req, filepath.Join(dir, "missing"), "text/plain", cacheControlNoCache)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing file = %d, want 404", rec.Code)
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoopbackTokenRedirect(t *testing.T) {
	next := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	handler := loopbackTokenRedirect("token", "dm-secret", next)
	cases := []struct {
		name, remote, target string
		want                 int
		location             string
	}{
		{"loopback without token", "127.0.0.1:5000", "/dm", http.StatusFound, "/dm?token=dm-secret"},
		{"ipv6 loopback", "[::1]:5000", "/dm", http.StatusFound, "/dm?token=dm-secret"},
		{"loopback with token", "127.0.0.1:5000", "/dm?token=x", http.StatusOK, ""},
		{"lan device", "192.168.1.40:5000", "/dm", http.StatusOK, ""},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodGet, tc.target, nil)
		request.RemoteAddr = tc.remote
		recorder := httptest.NewRecorder()
		handler(recorder, request)
		if recorder.Code != tc.want || recorder.Header().Get("Location") != tc.location {
			t.Fatalf("%s: code %d location %q, want %d %q", tc.name, recorder.Code, recorder.Header().Get("Location"), tc.want, tc.location)
		}
	}
}
