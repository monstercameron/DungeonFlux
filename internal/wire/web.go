package wire

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/config"
)

var assetName = regexp.MustCompile(`^[0-9a-fA-F]{64}\.[A-Za-z0-9]+$`)

func mountWeb(mux *http.ServeMux, cfg config.Config) error {
	root, err := filepath.Abs("web")
	if err != nil {
		return fmt.Errorf("resolve web root: %w", err)
	}
	wasmRoot, err := filepath.Abs("artifacts/wasm")
	if err != nil {
		return fmt.Errorf("resolve wasm root: %w", err)
	}
	staticRoot := filepath.Join(root, "shell", "static")
	mux.HandleFunc("/dm", pageHandler(staticRoot))
	mux.HandleFunc("/p", pageHandler(staticRoot))
	mux.HandleFunc("/p/", pageHandler(staticRoot))
	mux.HandleFunc("/host", pageHandler(staticRoot))
	mux.HandleFunc("/about", pageHandler(staticRoot))
	mux.HandleFunc("/app/dungeonflux.wasm", wasmHandler(filepath.Join(wasmRoot, "dungeonflux.wasm")))
	mux.HandleFunc("/wasm_exec.js", fileHandler(filepath.Join(wasmRoot, "wasm_exec.js"), "text/javascript; charset=utf-8"))
	mux.Handle("/splat/js/", staticHandler(filepath.Join(root, "splat", "js")))
	mux.Handle("/splat/vendor/", staticHandler(filepath.Join(root, "splat", "vendor")))
	mux.HandleFunc("/assets/", assetHandler(filepath.Join(cfg.Server.DataDir, "assets")))
	return nil
}

func pageHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		serveNoCacheFile(w, r, filepath.Join(root, "index.html"), "text/html; charset=utf-8")
	}
}

func fileHandler(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		serveNoCacheFile(w, r, name, contentType)
	}
}

func wasmHandler(uncompressed string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := uncompressed
		compressed := uncompressed + ".br"
		if acceptsBrotli(r.Header.Get("Accept-Encoding")) && isRegularFile(compressed) {
			name = compressed
			w.Header().Set("Content-Encoding", "br")
			w.Header().Set("Vary", "Accept-Encoding")
		} else if acceptsGzip(r.Header.Get("Accept-Encoding")) && isRegularFile(uncompressed+".gz") {
			name = uncompressed + ".gz"
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")
		}
		serveNoCacheFile(w, r, name, "application/wasm")
	}
}

func serveNoCacheFile(w http.ResponseWriter, r *http.Request, name, contentType string) {
	file, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	etag := `"` + strconv.FormatInt(info.Size(), 16) + "-" + strconv.FormatInt(info.ModTime().UnixNano(), 16) + `"`
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", contentType)
	if matchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	http.ServeContent(w, r, filepath.Base(name), info.ModTime(), file)
}

func matchesETag(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

func acceptsBrotli(value string) bool {
	return acceptsEncoding(value, "br")
}

func acceptsGzip(value string) bool {
	return acceptsEncoding(value, "gzip")
}

func acceptsEncoding(value, encoding string) bool {
	for _, part := range strings.Split(value, ",") {
		bits := strings.Split(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(bits[0]), encoding) {
			for _, option := range bits[1:] {
				if strings.EqualFold(strings.TrimSpace(option), "q=0") || strings.EqualFold(strings.TrimSpace(option), "q=0.0") {
					return false
				}
			}
			return true
		}
	}
	return false
}

func staticHandler(root string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".mjs") {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		}
		http.StripPrefix(filepath.ToSlash("/splat/"), http.FileServer(http.Dir(filepath.Dir(root)))).ServeHTTP(w, r)
	})
}

func assetHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/assets/")
		if strings.Contains(name, "/") || !assetName.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		path := filepath.Join(root, name)
		if !isRegularFile(path) {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	}
}

func isRegularFile(name string) bool {
	info, err := os.Lstat(name)
	return err == nil && info.Mode().IsRegular() && info.Mode()&fs.ModeSymlink == 0
}
