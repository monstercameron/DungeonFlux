package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func TestBuildReferenceEditRequest_IncludesEveryCrop(t *testing.T) {
	body, contentType, err := buildReferenceEditRequest(ports.ImageRequest{Prompt: "keep identity", Size: "1536x1024"}, [][]byte{{1, 2}, {3, 4}})
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(bytes.NewReader(body), strings.TrimPrefix(contentType, "multipart/form-data; boundary="))
	count := 0
	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}
		if part.FormName() == "image[]" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("reference parts=%d", count)
	}
}

func TestBuildReferenceEditRequest_RejectsMissingInputs(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  ports.ImageRequest
		refs [][]byte
	}{
		{name: "prompt", req: ports.ImageRequest{Size: "1024x1024"}, refs: [][]byte{{1}}},
		{name: "references", req: ports.ImageRequest{Prompt: "hero"}},
		{name: "empty image", req: ports.ImageRequest{Prompt: "hero"}, refs: [][]byte{{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := buildReferenceEditRequest(tc.req, tc.refs); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}

func TestAdapter_GenerateWithReferencesPostsEditAndParsesImage(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Errorf("content type=%q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if len(r.MultipartForm.File["image[]"]) != 1 || r.FormValue("prompt") != "hero" {
			t.Fatalf("form=%v files=%v", r.MultipartForm.Value, r.MultipartForm.File)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64Fixture()}}})
	}))
	defer server.Close()
	stream, err := New("key", server.URL+"/generations", time.Second, nil).GenerateWithReferences(context.Background(), ports.ImageRequest{Prompt: "hero"}, [][]byte{{1}})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/edits" {
		t.Fatalf("path=%q", gotPath)
	}
	if event, err := stream.Recv(); err != nil || len(event.PNG) == 0 {
		t.Fatalf("event=%+v err=%v", event, err)
	}
}

func base64Fixture() string {
	return "iVBORw0KGgo="
}

func TestBuildReferenceEditRequest_LabelsEachReferenceWithItsImageType(t *testing.T) {
	pngData := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpegData := []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	webpData := []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
	body, contentType, err := buildReferenceEditRequest(ports.ImageRequest{Prompt: "keep outfit"}, [][]byte{pngData, jpegData, webpData})
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(bytes.NewReader(body), strings.TrimPrefix(contentType, "multipart/form-data; boundary="))
	var got []string
	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}
		if part.FormName() == "image[]" {
			got = append(got, part.Header.Get("Content-Type")+" "+part.FileName())
		}
	}
	want := []string{"image/png reference-0.png", "image/jpeg reference-1.jpg", "image/webp reference-2.webp"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("reference parts = %v, want %v (the edits endpoint rejects application/octet-stream)", got, want)
	}
}
