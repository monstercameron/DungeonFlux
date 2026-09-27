package wire

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/config"
)

var assetName = regexp.MustCompile(`^[0-9a-fA-F]{64}\.[A-Za-z0-9]+$`)

// Cache-Control values for the three resource classes served here. See
// plan.md §0.4 (static media over HTTPS) and the WEB-019/BASE-020 history:
// hashed content-addressed files never change, so they get the longest TTL
// browsers respect; splat scene chunks change only when a new scene id
// ships, so they get a long TTL plus a validator; everything the DM/phone
// shell loads by a fixed URL (index.html, the WASM bundle, wasm_exec.js, the
// splat JS modules and scene profile JSON) must revalidate on every load so
// a fresh deploy is never masked by a stale cached copy (a stale df-splat.mjs
// once shipped a broken battle view).
const (
	cacheControlImmutable  = "public, max-age=31536000, immutable"
	cacheControlSplatMedia = "public, max-age=2592000, must-revalidate"
	cacheControlNoCache    = "no-cache"
)

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
	mux.HandleFunc("/dm", loopbackTokenRedirect("token", cfg.Server.DMToken, pageHandler(staticRoot)))
	mux.HandleFunc("/p", pageHandler(staticRoot))
	mux.HandleFunc("/p/", pageHandler(staticRoot))
	mux.HandleFunc("/host", loopbackTokenRedirect("t", cfg.Server.HostToken, pageHandler(staticRoot)))
	mux.HandleFunc("/about", pageHandler(staticRoot))
	mux.HandleFunc("/preview", pageHandler(staticRoot))
	mux.HandleFunc("/app/dungeonflux.wasm", wasmHandler(filepath.Join(wasmRoot, "dungeonflux.wasm")))
	mux.HandleFunc("/wasm_exec.js", fileHandler(filepath.Join(wasmRoot, "wasm_exec.js"), "text/javascript; charset=utf-8"))
	mux.Handle("/splat/js/", staticHandler(filepath.Join(root, "splat", "js")))
	mux.Handle("/splat/vendor/", staticHandler(filepath.Join(root, "splat", "vendor")))
	mux.Handle("/splat/scenes/", staticHandler(filepath.Join(root, "splat", "scenes")))
	// Scene profiles in web/splat/scenes reference their LOD chunks and voxel
	// colliders as ../../../artifacts/media/supersplat/..., which resolves to
	// this path. These are the large streamed splat files: the stated HTTP
	// exception to the gRPC-only transport (SPLAT-023).
	supersplat, err := filepath.Abs(filepath.Join("artifacts", "media", "supersplat"))
	if err != nil {
		return fmt.Errorf("resolve splat media root: %w", err)
	}
	mux.Handle("/artifacts/media/supersplat/", http.StripPrefix("/artifacts/media/supersplat/", cachedTreeHandler(supersplat, cacheControlSplatMedia)))
	mux.HandleFunc("/assets/", assetHandler(filepath.Join(cfg.Server.DataDir, "assets")))
	// Self-hosted display fonts referenced by index.html's @font-face rules and
	// <link rel=preload>. Like /assets/, these are cached forever: a font
	// update ships under a new filename (rename-on-change) rather than
	// overwriting one of these four in place.
	mux.Handle("/fonts/", http.StripPrefix("/fonts/", fontHandler(filepath.Join(staticRoot, "fonts"))))
	return nil
}

// loopbackTokenRedirect sends a tokenless /dm or /host request from the host
// machine itself to the tokenized URL. The TV and host console run on the
// server's machine, so opening localhost:<port>/dm just works; requests from
// other devices still need the link with its token.
func loopbackTokenRedirect(param, token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.URL.Query().Get(param) == "" && isLoopbackRequest(r) && r.Method == http.MethodGet {
			query := r.URL.Query()
			query.Set(param, token)
			target := *r.URL
			target.RawQuery = query.Encode()
			http.Redirect(w, r, target.RequestURI(), http.StatusFound)
			return
		}
		next(w, r)
	}
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func pageHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		serveFileWithCache(w, r, filepath.Join(root, "index.html"), "text/html; charset=utf-8", cacheControlNoCache)
	}
}

func fileHandler(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		serveFileWithCache(w, r, name, contentType, cacheControlNoCache)
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
		serveFileWithCache(w, r, name, "application/wasm", cacheControlNoCache)
	}
}

// serveFileWithCache serves a single named file with a strong ETag (derived
// from its size and modification time) and the given Cache-Control value, so
// callers with cacheControlNoCache still get a cheap 304 on every load while
// callers with a long max-age skip revalidation entirely until it expires.
func serveFileWithCache(w http.ResponseWriter, r *http.Request, name, contentType, cacheControl string) {
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
	etag := fileETag(info)
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("ETag", etag)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if matchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	http.ServeContent(w, r, filepath.Base(name), info.ModTime(), file)
}

func fileETag(info os.FileInfo) string {
	return `"` + strconv.FormatInt(info.Size(), 16) + "-" + strconv.FormatInt(info.ModTime().UnixNano(), 16) + `"`
}

// cachedTreeHandler serves files under root with a strong ETag, Last-Modified
// (via http.ServeContent), and the given Cache-Control. It is used for
// content that changes as a whole directory (a new scene id) rather than per
// file, so a long max-age is safe as long as a validator is still present to
// catch a redeploy that reuses the same path.
func cachedTreeHandler(root, cacheControl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		rel := filepath.FromSlash(pathpkg.Clean("/" + r.URL.Path))
		full := filepath.Join(root, rel)
		if !isRegularFile(full) {
			http.NotFound(w, r)
			return
		}
		serveFileWithCache(w, r, full, "", cacheControl)
	}
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
		// Revalidate on every load: without this, browsers kept running a stale
		// battle module after a fix was deployed.
		w.Header().Set("Cache-Control", "no-cache")
		http.StripPrefix(filepath.ToSlash("/splat/"), http.FileServer(http.Dir(filepath.Dir(root)))).ServeHTTP(w, r)
	})
}

// assetHandler serves /assets/{sha256}.{ext}. The name is the content hash
// itself, so it is a strong ETag by construction and the response never
// changes for a given URL: Cache-Control is immutable with a one-year
// max-age (the longest browsers honor), and a client that already has the
// file need not even revalidate it.
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
		file, err := os.Open(path)
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
		etag := `"` + strings.ToLower(name) + `"`
		w.Header().Set("Cache-Control", cacheControlImmutable)
		w.Header().Set("ETag", etag)
		if matchesETag(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
	}
}

// fontHandler serves the self-hosted .woff2 files under root as
// font/woff2, cached like a content-addressed asset (immutable, one-year
// max-age): these ship under a fixed small set of filenames and are updated
// by renaming, never by overwriting bytes at an existing URL.
func fontHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		rel := filepath.FromSlash(pathpkg.Clean("/" + r.URL.Path))
		full := filepath.Join(root, rel)
		if !isRegularFile(full) || !strings.EqualFold(filepath.Ext(full), ".woff2") {
			http.NotFound(w, r)
			return
		}
		serveFileWithCache(w, r, full, "font/woff2", cacheControlImmutable)
	}
}

func isRegularFile(name string) bool {
	info, err := os.Lstat(name)
	return err == nil && info.Mode().IsRegular() && info.Mode()&fs.ModeSymlink == 0
}
