package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseDevlog_ReturnsNewestEntries(t *testing.T) {
	source := `<ol><li class="entry" id="new"><h2><a>Newest &amp; useful</a></h2><p>First <em>note</em>.</p></li><li class="entry"><h2>Older</h2><p>Second</p></li></ol>`
	entries := parseDevlog(source)
	if len(entries) != 2 || entries[0].Title != "Newest & useful" || entries[0].Body != "First note ." {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestParseDevlog_IgnoresNonEntries(t *testing.T) {
	if got := parseDevlog(`<li><h2>not a devlog item</h2></li>`); len(got) != 0 {
		t.Fatalf("got non-entry: %#v", got)
	}
}

func TestReadDevlog_ReturnsFileError(t *testing.T) {
	if _, err := readDevlog(filepath.Join(t.TempDir(), "missing.html")); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestHealthHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	healthHandler(recorder, httptest.NewRequest("GET", "/healthz", nil))
	if recorder.Code != 200 || recorder.Body.String() != "ok\n" {
		t.Fatalf("unexpected response: %d %q", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	healthHandler(recorder, httptest.NewRequest("POST", "/healthz", nil))
	if recorder.Code != 405 {
		t.Fatalf("unexpected method response: %d", recorder.Code)
	}
}

func TestRun_RecordsListenError(t *testing.T) {
	dir := t.TempDir()
	listenErr := errors.New("listen failed")
	err := run(configuration{port: 18199, devlogPath: filepath.Join(dir, "missing.html"), statusPath: filepath.Join(dir, "status.json")}, func(_ string, _ http.Handler) error {
		return listenErr
	})
	if err == nil || !strings.Contains(err.Error(), "listen failed") {
		t.Fatalf("unexpected run error: %v", err)
	}
}

func TestPlaceholderHTML_EscapesContent(t *testing.T) {
	page := placeholderHTML(`<phase>`, []devlogEntry{{Title: `<title>`, Body: `body & text`}}, time.Unix(0, 0))
	if !strings.Contains(page, "&lt;phase&gt;") || strings.Contains(page, "<h3><title>") || !strings.Contains(page, "body &amp; text") {
		t.Fatalf("content was not escaped: %s", page)
	}
}

func TestServePlaceholder_ReadsDevlogAndWritesStatus(t *testing.T) {
	dir := t.TempDir()
	devlog := filepath.Join(dir, "devlog.html")
	if err := os.WriteFile(devlog, []byte(`<li class="entry"><h2>Ready</h2><p>Started.</p></li>`), 0o644); err != nil {
		t.Fatal(err)
	}
	writer := &statusWriter{path: filepath.Join(dir, "status.json"), started: time.Now()}
	recorder := httptest.NewRecorder()
	servePlaceholder(recorder, configuration{phase: "Hour 0", devlogPath: devlog}, writer)
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "Ready") {
		t.Fatalf("unexpected placeholder: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestServePlaceholder_ReportsMissingDevlog(t *testing.T) {
	dir := t.TempDir()
	writer := &statusWriter{path: filepath.Join(dir, "status.json"), started: time.Now()}
	recorder := httptest.NewRecorder()
	servePlaceholder(recorder, configuration{phase: "Hour 0", devlogPath: filepath.Join(dir, "missing.html")}, writer)
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "first devlog entry") {
		t.Fatalf("unexpected fallback: %d %s", recorder.Code, recorder.Body.String())
	}
	data, err := os.ReadFile(writer.path)
	if err != nil || !strings.Contains(string(data), "missing.html") {
		t.Fatalf("missing error was not recorded: %v %s", err, data)
	}
}

func TestStatusWriter_WritesStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "status.json")
	writer := &statusWriter{path: path, started: time.Now()}
	if err := writer.write("problem"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), `"last_error": "problem"`) {
		t.Fatalf("status file: %v %s", err, data)
	}
}

func TestStatusWriter_ReturnsPathError(t *testing.T) {
	writer := &statusWriter{path: string([]byte{0}), started: time.Now()}
	if err := writer.write(""); err == nil {
		t.Fatal("expected invalid path error")
	}
}

func TestStatusWriter_ReturnsRenameError(t *testing.T) {
	writer := &statusWriter{path: t.TempDir(), started: time.Now()}
	if err := writer.write(""); err == nil {
		t.Fatal("expected rename error when target is a directory")
	}
}
