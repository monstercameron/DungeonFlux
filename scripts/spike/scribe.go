package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"
)

// scribeTranscriber posts the complete MediaRecorder container to ElevenLabs.
type scribeTranscriber struct {
	client *http.Client
	key    string
}

func newScribeTranscriber(key string) scribeTranscriber {
	return scribeTranscriber{client: &http.Client{Timeout: 5 * time.Second}, key: key}
}

func (t scribeTranscriber) Transcribe(ctx context.Context, mediaType string, audio []byte) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	field, err := writer.CreateFormFile("file", filename(mediaType))
	if err != nil {
		return "", fmt.Errorf("create audio form: %w", err)
	}
	if _, err := field.Write(audio); err != nil {
		return "", fmt.Errorf("write audio form: %w", err)
	}
	if err := writer.WriteField("model_id", "scribe_v2"); err != nil {
		return "", fmt.Errorf("write model field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close audio form: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.elevenlabs.io/v1/speech-to-text", &body)
	if err != nil {
		return "", fmt.Errorf("build scribe request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("xi-api-key", t.key)
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("scribe request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("scribe status: %s", resp.Status)
	}
	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode scribe response: %w", err)
	}
	return result.Text, nil
}

func filename(mediaType string) string {
	if mediaType == "audio/mp4" {
		return "talk.mp4"
	}
	return "talk.webm"
}
