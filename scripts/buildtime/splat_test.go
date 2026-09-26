package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunSplatJob_ExportsPLYAndRecordsMetadata(t *testing.T) {
	var paths []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch {
		case r.URL.Path == "/marble/v1/worlds:generate":
			var req marbleRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.Model != "marble-1.1" || strings.Contains(string(mustJSON(req)), "return_spz") {
				t.Fatalf("unexpected request: %#v", req)
			}
			json.NewEncoder(w).Encode(marbleOperation{Done: true, Response: marbleWorldWithMetadata("world-1", 1.25, -0.4)})
		case r.URL.Path == "/marble/v1/worlds/world-1:export":
			var req exportRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.AssetType != "splats" || req.Format != "ply" || req.Resolution != "full_res" {
				t.Fatalf("unexpected export: %#v", req)
			}
			json.NewEncoder(w).Encode(exportResponse{Done: true, Response: struct {
				URL string `json:"url"`
			}{URL: server.URL + "/asset.ply"}})
		case r.URL.Path == "/asset.ply":
			w.Write([]byte("ply\nformat binary_little_endian 1.0\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = RunSplatJob(context.Background(), writer, SplatOptions{Endpoint: server.URL + "/marble/v1/worlds:generate", APIKey: "world-key", Specs: []SplatSpec{{LogicalName: "battlefield", Quality: "full", Take: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	asset := writer.manifest.Assets["battlefield"]
	if asset.Metadata["metric_scale_factor"] != "1.25" || asset.Metadata["ground_plane_offset"] != "-0.4" {
		t.Fatalf("metadata: %#v", asset.Metadata)
	}
	if !strings.HasSuffix(asset.Takes[0].Path, ".ply") {
		t.Fatalf("asset path: %#v", asset.Takes[0])
	}
	if len(paths) != 3 {
		t.Fatalf("requests: %v", paths)
	}
}

func TestRunSplatJob_LiteRequests100k(t *testing.T) {
	var resolution string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "generate") {
			json.NewEncoder(w).Encode(marbleOperation{Done: true, Response: marbleWorldWithMetadata("w", 1, 0)})
			return
		}
		if strings.HasSuffix(r.URL.Path, ":export") {
			var req exportRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			resolution = req.Resolution
			json.NewEncoder(w).Encode(exportResponse{Done: true, Response: struct {
				URL string `json:"url"`
			}{URL: server.URL + "/ply"}})
			return
		}
		w.Write([]byte("ply"))
	}))
	defer server.Close()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RunSplatJob(context.Background(), writer, SplatOptions{Endpoint: server.URL + "/generate", APIKey: "key", Specs: []SplatSpec{{LogicalName: "lite", Quality: "lite", Take: 1}}}); err != nil {
		t.Fatal(err)
	}
	if resolution != "100k" {
		t.Fatalf("resolution %q", resolution)
	}
}

func TestRunSplatJob_PollsOperation(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/generate":
			json.NewEncoder(w).Encode(marbleOperation{OperationID: "op-1"})
		case "/generate/operations/op-1":
			json.NewEncoder(w).Encode(marbleOperation{Done: true, Response: marbleWorldWithMetadata("w", 1, 0)})
		case "/generate/worlds/w:export":
			json.NewEncoder(w).Encode(exportResponse{Done: true, Response: struct {
				URL string `json:"url"`
			}{URL: server.URL + "/ply"}})
		default:
			w.Write([]byte("ply"))
		}
	}))
	defer server.Close()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RunSplatJob(context.Background(), writer, SplatOptions{Endpoint: server.URL + "/generate", APIKey: "key", PollInterval: 1, Specs: []SplatSpec{{LogicalName: "polled", Take: 1}}}); err != nil {
		t.Fatal(err)
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
	if err := RunSplatJob(context.Background(), nil, SplatOptions{}); err == nil {
		t.Fatal("accepted nil writer")
	}
	if err := RunSplatJob(context.Background(), writer, SplatOptions{}); err == nil {
		t.Fatal("accepted empty specs")
	}
}

func TestHelpersAndDownloadErrors(t *testing.T) {
	if effectiveQuality("") != "full" || exportResolution("lite") != "100k" || exportResolution("full") != "full_res" {
		t.Fatal("quality defaults are incorrect")
	}
	if effectiveMarbleEndpoint("custom") != "custom" {
		t.Fatal("custom endpoint was not retained")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer server.Close()
	client := marbleClient{httpClient: &http.Client{}, endpoint: server.URL, apiKey: "key"}
	if _, err := client.Download(context.Background(), server.URL); err == nil {
		t.Fatal("Download accepted an error response")
	}
	if _, err := client.Generate(context.Background(), marbleRequest{}); err == nil {
		t.Fatal("Generate accepted missing key")
	}
	if _, err := client.Export(context.Background(), "w", exportRequest{}); err == nil {
		t.Fatal("Export accepted an error response")
	}
}

func marbleWorldWithMetadata(id string, scale, offset float64) marbleWorld {
	var world marbleWorld
	world.ID = id
	world.Assets.Splats.SemanticsMetadata.MetricScaleFactor = scale
	world.Assets.Splats.SemanticsMetadata.GroundPlaneOffset = offset
	return world
}
func mustJSON(value any) []byte { data, _ := json.Marshal(value); return data }
