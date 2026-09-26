package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSplatJob_GeneratesConvertsAndRecordsMetadata(t *testing.T) {
	var requests []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/marble/v1/worlds:generate" {
			var req marbleRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.Model != "marble-1.1" || !req.SPZ {
				t.Fatalf("unexpected request: %#v", req)
			}
			json.NewEncoder(w).Encode(marbleOperation{Done: true, Response: marbleWorld{SPZURL: server.URL + "/asset.spz", MetricScaleFactor: 1.25, GroundPlaneOffset: -0.4}})
			return
		}
		if r.URL.Path == "/asset.spz" {
			w.Write([]byte("spz-data"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	converter := func(ctx context.Context, input, output string) error {
		data, err := os.ReadFile(input)
		if err != nil {
			return err
		}
		return os.WriteFile(output, append([]byte("sog:"), data...), 0o644)
	}
	err = RunSplatJob(context.Background(), writer, SplatOptions{Endpoint: server.URL + "/marble/v1/worlds:generate", APIKey: "world-key", Prompt: "tavern", Specs: []SplatSpec{{LogicalName: "battlefield_tavern_splat", Quality: "full", Take: 1}}, Converter: converter})
	if err != nil {
		t.Fatal(err)
	}
	asset := writer.manifest.Assets["battlefield_tavern_splat"]
	if asset.Kind != "SPLAT" || asset.Metadata["metric_scale_factor"] != "1.25" || asset.Metadata["ground_plane_offset"] != "-0.4" {
		t.Fatalf("unexpected splat asset: %#v", asset)
	}
	data, err := os.ReadFile(filepath.Join(writer.root, asset.Takes[0].Path))
	if err != nil || !strings.HasPrefix(string(data), "sog:spz-data") {
		t.Fatalf("unexpected SOG output: %q, %v", data, err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected generate and download requests, got %v", requests)
	}
}

func TestRunSplatJob_PollsOperation(t *testing.T) {
	polls := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/generate":
			json.NewEncoder(w).Encode(marbleOperation{OperationID: "op-1"})
		case "/generate/operations/op-1":
			polls++
			json.NewEncoder(w).Encode(marbleOperation{Done: true, Response: marbleWorld{SPZURL: server.URL + "/file"}})
		case "/file":
			w.Write([]byte("data"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	converter := func(ctx context.Context, input, output string) error {
		return os.WriteFile(output, []byte("sog"), 0o644)
	}
	// Polling derives its operation URL from the endpoint, so use a matching path.
	err = RunSplatJob(context.Background(), writer, SplatOptions{Endpoint: server.URL + "/generate", APIKey: "key", PollInterval: 1, Specs: []SplatSpec{{LogicalName: "lite", Quality: "lite", Take: 1}}, Converter: converter})
	if err != nil {
		t.Fatal(err)
	}
	if polls != 1 {
		t.Fatalf("expected one poll, got %d", polls)
	}
}

func TestRunSplatJob_DryRunAndValidation(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RunSplatJob(context.Background(), writer, SplatOptions{DryRun: true, Specs: []SplatSpec{{LogicalName: "full", Take: 1}}}); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != 0 {
		t.Fatal("dry-run wrote assets")
	}
	for _, spec := range []SplatSpec{{LogicalName: "../bad", Take: 1}, {LogicalName: "bad", Take: 0}} {
		if err := RunSplatJob(context.Background(), writer, SplatOptions{DryRun: true, Specs: []SplatSpec{spec}}); err == nil {
			t.Fatalf("accepted invalid spec %#v", spec)
		}
	}
}

func TestMarbleClient_RejectsMissingKey(t *testing.T) {
	_, err := (marbleClient{endpoint: "http://example.invalid", httpClient: &http.Client{}}).Generate(context.Background(), marbleRequest{})
	if err == nil {
		t.Fatal("Generate accepted missing API key")
	}
}

func TestSplatHelpersAndDownloadErrors(t *testing.T) {
	if effectiveQuality("") != "full" || effectiveQuality("lite") != "lite" {
		t.Fatal("quality defaults are incorrect")
	}
	if effectiveMarbleEndpoint("custom") != "custom" {
		t.Fatal("custom endpoint was not retained")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(marbleOperation{Error: &struct {
			Message string `json:"message"`
		}{Message: "upstream"}})
	}))
	defer server.Close()
	client := marbleClient{httpClient: &http.Client{}, endpoint: server.URL, apiKey: "key"}
	if _, err := client.Download(context.Background(), server.URL); err == nil {
		t.Fatal("Download accepted an error response")
	}
}

func TestRunSplatJobRejectsMissingWriterAndSpecs(t *testing.T) {
	if err := RunSplatJob(context.Background(), nil, SplatOptions{}); err == nil {
		t.Fatal("RunSplatJob accepted nil writer")
	}
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RunSplatJob(context.Background(), writer, SplatOptions{}); err == nil {
		t.Fatal("RunSplatJob accepted empty specs")
	}
}

func TestMarbleClient_GenerateRejectsMalformedAndMissingOperation(t *testing.T) {
	for _, body := range []string{"not-json", `{"done":false}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(body))
		}))
		_, err := (marbleClient{httpClient: &http.Client{}, endpoint: server.URL, apiKey: "key"}).Generate(context.Background(), marbleRequest{})
		server.Close()
		if err == nil {
			t.Fatalf("Generate accepted %q", body)
		}
	}
}
