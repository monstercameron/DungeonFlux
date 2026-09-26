package segmind

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type submitPayload struct {
	Prompt        string `json:"prompt"`
	ImageURL      string `json:"first_frame_url"`
	LastFrameURL  string `json:"last_frame_url,omitempty"`
	Duration      int    `json:"duration"`
	Resolution    string `json:"resolution"`
	GenerateAudio bool   `json:"generate_audio"`
}

func buildSubmitPayload(req ports.VideoRequest) ([]byte, error) {
	if len(req.FirstFrame) == 0 {
		return nil, fmt.Errorf("first frame is required")
	}
	if req.Seconds <= 0 || req.Resolution == "" {
		return nil, fmt.Errorf("duration and resolution are required")
	}
	payload := submitPayload{
		Prompt: req.Prompt, ImageURL: dataURL(req.FirstFrame), Duration: req.Seconds,
		Resolution: req.Resolution, GenerateAudio: false,
	}
	if len(req.LastFrame) != 0 {
		payload.LastFrameURL = dataURL(req.LastFrame)
	}
	return json.Marshal(payload)
}

func dataURL(data []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}
