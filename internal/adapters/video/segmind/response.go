package segmind

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type jobResponse struct {
	RequestID string `json:"request_id"`
	ID        string `json:"id"`
	Status    string `json:"status"`
	VideoURL  string `json:"video_url"`
	URL       string `json:"url"`
	Output    string `json:"output"`
	Error     string `json:"error"`
	QueuePos  int    `json:"queue_position"`
}

func parseSubmit(data []byte) (string, error) {
	var response jobResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("decode submit response: %w", err)
	}
	id := response.RequestID
	if id == "" {
		id = response.ID
	}
	if id == "" {
		return "", fmt.Errorf("submit response has no request_id")
	}
	return id, nil
}

func parseStatus(data []byte) (ports.VideoStatus, error) {
	var response jobResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return ports.VideoStatus{}, fmt.Errorf("decode status response: %w", err)
	}
	state, err := mapState(response.Status)
	if err != nil {
		return ports.VideoStatus{}, err
	}
	url := response.VideoURL
	if url == "" {
		url = response.URL
	}
	if url == "" {
		url = response.Output
	}
	if state == vocab.JobDone && url == "" {
		return ports.VideoStatus{}, fmt.Errorf("completed response has no video URL")
	}
	if state == vocab.JobFailed && response.Error == "" {
		response.Error = "provider reported failure"
	}
	if state == vocab.JobFailed {
		return ports.VideoStatus{}, fmt.Errorf("video job failed: %s", response.Error)
	}
	return ports.VideoStatus{State: state, QueuePos: response.QueuePos, URL: url}, nil
}

func mapState(status string) (vocab.JobState, error) {
	switch strings.ToLower(status) {
	case "queued", "pending", "submitted":
		return vocab.JobQueued, nil
	case "running", "processing", "in_progress":
		return vocab.JobRunning, nil
	case "done", "completed", "success", "succeeded":
		return vocab.JobDone, nil
	case "failed", "error", "canceled", "cancelled":
		return vocab.JobFailed, nil
	default:
		return "", fmt.Errorf("unknown video job status %q", status)
	}
}
