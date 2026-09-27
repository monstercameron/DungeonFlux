package openai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// GenerateWithReferences submits an edit request using the supplied crop
// bytes as identity references. It is an additive capability; callers can
// continue using ports.ImageGen when no references are ready.
func (a *Adapter) GenerateWithReferences(ctx context.Context, req ports.ImageRequest, references [][]byte) (ports.ImageStream, error) {
	if len(references) == 0 {
		return a.Generate(ctx, req)
	}
	body, contentType, err := buildReferenceEditRequest(req, references)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimSuffix(a.endpoint, "/generations") + "/edits"
	httpReq, err := httpx.ContextRequest(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, callError(vocab.ErrBadOutput, false, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.key)
	httpReq.Header.Set("Content-Type", contentType)
	response, _, err := a.client.Do(httpReq)
	if err != nil {
		return nil, classifyTransport(ctx, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, statusError(response)
	}
	events, err := parseResponse(response.Body)
	if err != nil {
		return nil, callError(vocab.ErrBadOutput, false, err)
	}
	return &stream{events: events}, nil
}

func buildReferenceEditRequest(req ports.ImageRequest, references [][]byte) ([]byte, string, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, "", callError(vocab.ErrBadOutput, false, errors.New("image prompt is empty"))
	}
	if len(references) == 0 {
		return nil, "", callError(vocab.ErrBadOutput, false, errors.New("reference image list is empty"))
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{"model": model, "prompt": req.Prompt, "size": req.Size, "background": background(req.Transparent), "output_format": "png"}
	if fields["size"] == "" {
		fields["size"] = "1024x1024"
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", fmt.Errorf("write image field %q: %w", name, err)
		}
	}
	for index, data := range references {
		if len(data) == 0 {
			return nil, "", fmt.Errorf("reference image %d is empty", index)
		}
		// CreateFormFile labels every part application/octet-stream, which the
		// edits endpoint rejects; it accepts only image/png, jpeg and webp.
		kind := http.DetectContentType(data)
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image[]"; filename="reference-%d%s"`, index, referenceExtension(kind)))
		header.Set("Content-Type", kind)
		part, err := writer.CreatePart(header)
		if err != nil {
			return nil, "", fmt.Errorf("create reference image %d: %w", index, err)
		}
		if _, err := part.Write(data); err != nil {
			return nil, "", fmt.Errorf("write reference image %d: %w", index, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close reference request: %w", err)
	}
	return body.Bytes(), writer.FormDataContentType(), nil
}

func referenceExtension(kind string) string {
	switch kind {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	}
	return ".png"
}

var _ interface {
	GenerateWithReferences(context.Context, ports.ImageRequest, [][]byte) (ports.ImageStream, error)
} = (*Adapter)(nil)
