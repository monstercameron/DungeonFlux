package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testTransparentPNG(t *testing.T) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 1, color.NRGBA{B: 255, A: 64})
	var data strings.Builder
	if err := png.Encode(&stringWriter{Builder: &data}, img); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString([]byte(data.String()))
}

type stringWriter struct{ Builder *strings.Builder }

func (w *stringWriter) Write(p []byte) (int, error) { return w.Builder.Write(p) }

func TestImagesClient_GenerateBuildsTransparentRequest(t *testing.T) {
	encoded := testTransparentPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected request: %s %s", r.Method, r.Header.Get("Authorization"))
		}
		var req imagesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Background != "transparent" || req.OutputFormat != "png" || req.Model == "" {
			t.Errorf("unexpected image request: %#v", req)
		}
		json.NewEncoder(w).Encode(imagesResponse{Data: []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		}{{B64JSON: encoded}}})
	}))
	defer server.Close()

	data, err := (ImagesClient{Endpoint: server.URL, APIKey: "test-key"}).Generate(context.Background(), "a full-body tavern NPC")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.At(1, 1).(color.NRGBA).A != 64 {
		t.Fatalf("alpha was not preserved: %#v", decoded.At(1, 1))
	}
}

func TestImagesClient_GenerateRejectsErrorsAndOpaqueImages(t *testing.T) {
	opaque := image.NewRGBA(image.Rect(0, 0, 1, 1))
	opaque.Set(0, 0, color.White)
	var raw strings.Builder
	if err := png.Encode(&stringWriter{Builder: &raw}, opaque); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		body imagesResponse
		code int
	}{
		{name: "opaque", body: imagesResponse{Data: []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		}{{B64JSON: base64.StdEncoding.EncodeToString([]byte(raw.String()))}}}, code: http.StatusOK},
		{name: "server error", body: imagesResponse{Error: &struct {
			Message string `json:"message"`
		}{Message: "bad request"}}, code: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				json.NewEncoder(w).Encode(tc.body)
			}))
			defer server.Close()
			if _, err := (ImagesClient{Endpoint: server.URL, APIKey: "key"}).Generate(context.Background(), "prompt"); err == nil {
				t.Fatal("Generate accepted invalid response")
			}
		})
	}
}

func TestRunCutoutJob_DryRunAndManifest(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	spec := CutoutSpec{LogicalName: "npc_cutout", Prompt: "full body", Take: 1}
	if err := RunCutoutJob(context.Background(), writer, CutoutOptions{DryRun: true, Specs: []CutoutSpec{spec}}); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != 0 {
		t.Fatal("dry-run wrote a manifest asset")
	}

	encoded := testTransparentPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(imagesResponse{Data: []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		}{{B64JSON: encoded}}})
	}))
	defer server.Close()
	if err := RunCutoutJob(context.Background(), writer, CutoutOptions{Endpoint: server.URL, APIKey: "key", Specs: []CutoutSpec{spec}}); err != nil {
		t.Fatal(err)
	}
	asset := writer.manifest.Assets["npc_cutout"]
	if asset.Kind != "IMAGE_CUTOUT" || len(asset.Takes) != 1 {
		t.Fatalf("unexpected manifest asset: %#v", asset)
	}
	if _, err := os.Stat(filepath.Join(writer.root, asset.Takes[0].Path)); err != nil {
		t.Fatal(err)
	}
}

func TestRunCutoutJob_ValidatesSpecs(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range []CutoutSpec{{LogicalName: "../x", Prompt: "x", Take: 1}, {LogicalName: "x", Take: 1}, {LogicalName: "x", Prompt: "x"}} {
		if err := RunCutoutJob(context.Background(), writer, CutoutOptions{DryRun: true, Specs: []CutoutSpec{spec}}); err == nil {
			t.Fatalf("accepted invalid spec: %#v", spec)
		}
	}
}
