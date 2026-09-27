// Command devserver supervises the human DungeonFlux test server.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type configuration struct {
	port                                    int
	phase, devlogPath, statusPath           string
	repoRoot, buildDir, dataDir, configPath string
	interval                                time.Duration
	skipGate                                bool
	liveReload                              bool
	runtimeDir                              string
}

type status struct {
	Commit  string `json:"commit"`
	BuildAt string `json:"build_time"`
	Mode    string `json:"mode"`
	Uptime  string `json:"uptime"`
	LastErr string `json:"last_error"`
}
type statusWriter struct {
	mu      sync.Mutex
	path    string
	started time.Time
	current status
}
type devlogEntry struct{ Title, Body string }

func main() {
	cfg := configuration{}
	flag.IntVar(&cfg.port, "port", 8443, "HTTP port")
	flag.StringVar(&cfg.phase, "phase", "Build in progress", "placeholder phase")
	flag.StringVar(&cfg.devlogPath, "devlog", "docs/devlog.html", "HTML devlog")
	flag.StringVar(&cfg.statusPath, "status", "artifacts/logs/devserver/status.json", "status JSON")
	flag.StringVar(&cfg.repoRoot, "repo", ".", "repository root")
	flag.StringVar(&cfg.buildDir, "build-dir", "artifacts/build/human", "binary directory")
	flag.StringVar(&cfg.dataDir, "data-dir", "artifacts/runtime/human", "runtime data directory")
	flag.StringVar(&cfg.configPath, "config", "", "server config")
	flag.DurationVar(&cfg.interval, "interval", 30*time.Minute, "rebuild interval")
	flag.BoolVar(&cfg.skipGate, "skip-gate", false, "skip full gate for local development")
	flag.BoolVar(&cfg.liveReload, "live-reload", false, "watch source files and reload clients after successful builds")
	flag.StringVar(&cfg.runtimeDir, "runtime-dir", "", "isolated server working directory (defaults to repo)")
	flag.Parse()
	if err := runSupervisor(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "supervisor: %v\n", err)
	}
}

type listenFunc func(string, http.Handler) error

// run retains the placeholder HTTP harness used by unit tests.
func run(cfg configuration, listen listenFunc) error {
	started := time.Now()
	w := &statusWriter{path: cfg.statusPath, started: started, current: status{BuildAt: started.UTC().Format(time.RFC3339), Mode: "placeholder"}}
	_ = w.write("")
	server := newPlaceholderServer(cfg, w)
	addr := fmt.Sprintf(":%d", cfg.port)
	if err := listen(addr, server.Handler); err != nil {
		_ = w.write(err.Error())
		return fmt.Errorf("serve %s: %w", addr, err)
	}
	return nil
}

func newPlaceholderServer(cfg configuration, w *statusWriter) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(rw, r)
			return
		}
		servePlaceholder(rw, cfg, w)
	})
	return &http.Server{Handler: mux}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func servePlaceholder(w http.ResponseWriter, cfg configuration, writer *statusWriter) {
	entries, err := readDevlog(cfg.devlogPath)
	lastErr := ""
	if err != nil {
		lastErr = err.Error()
	}
	_ = writer.write(lastErr)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, placeholderHTML(cfg.phase, entries, writer.started))
}

func readDevlog(path string) ([]devlogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseDevlog(string(data)), nil
}

func parseDevlog(source string) []devlogEntry {
	entryRE := regexp.MustCompile(`(?is)<li\b[^>]*class=["'][^"']*\bentry\b[^"']*["'][^>]*>(.*?)</li>`)
	titleRE := regexp.MustCompile(`(?is)<h2\b[^>]*>(.*?)</h2>`)
	bodyRE := regexp.MustCompile(`(?is)<p\b[^>]*>(.*?)</p>`)
	tagRE := regexp.MustCompile(`(?is)<[^>]+>`)
	matches := entryRE.FindAllStringSubmatch(source, 5)
	entries := make([]devlogEntry, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		title := firstText(titleRE.FindStringSubmatch(match[1]), tagRE)
		body := firstText(bodyRE.FindStringSubmatch(match[1]), tagRE)
		if title != "" || body != "" {
			entries = append(entries, devlogEntry{title, body})
		}
	}
	return entries
}

func firstText(match []string, tagRE *regexp.Regexp) string {
	if len(match) < 2 {
		return ""
	}
	text := tagRE.ReplaceAllString(match[1], " ")
	return strings.Join(strings.Fields(html.UnescapeString(text)), " ")
}

func placeholderHTML(phase string, entries []devlogEntry, started time.Time) string {
	var logHTML strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&logHTML, "<article><h3>%s</h3><p>%s</p></article>", html.EscapeString(entry.Title), html.EscapeString(entry.Body))
	}
	if logHTML.Len() == 0 {
		logHTML.WriteString("<p class=muted>The first devlog entry will appear here shortly.</p>")
	}
	return fmt.Sprintf(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>DungeonFlux — build in progress</title><style>:root{color-scheme:dark}body{margin:0;background:#0b1528;color:#e8edf5;font:16px system-ui,sans-serif}main{max-width:760px;margin:10vh auto;padding:2rem}h1{font-size:clamp(2rem,6vw,4rem);color:#e0b061}.card{border:1px solid #30425e;border-radius:12px;padding:1rem;margin:1.5rem 0;background:#111f36}article{border-top:1px solid #30425e;padding:.8rem 0}h3{color:#e0b061}p{line-height:1.5}.muted{color:#9aa9bd}small{color:#8494ab}</style></head><body><main><small>DUNGEONFLUX · HOUR ZERO</small><h1>%s</h1><h2>The human test server is online.</h2><div class="card"><strong>Current phase</strong><p>%s</p><p class="muted">The playable dungeon will appear here as the build passes its gates. Keep this page open; the supervisor will replace it with the latest good build.</p></div><section><h2>Latest devlog entries</h2>%s</section><small>Placeholder started %s · <a href="/healthz">health check</a></small></main></body></html>`, html.EscapeString(phase), html.EscapeString(phase), logHTML.String(), started.UTC().Format(time.RFC3339))
}

func (s *statusWriter) write(lastErr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current.LastErr = lastErr
	s.current.Uptime = time.Since(s.started).Round(time.Second).String()
	data, err := json.MarshalIndent(s.current, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
