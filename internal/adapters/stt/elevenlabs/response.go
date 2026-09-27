package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type transcriptResponse struct {
	Text string `json:"text"`
}

func parseResponse(reader io.Reader) (ports.Transcript, error) {
	var response transcriptResponse
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&response); err != nil {
		return ports.Transcript{}, fmt.Errorf("decode Scribe response: %w", err)
	}
	if strings.TrimSpace(response.Text) == "" {
		return ports.Transcript{}, fmt.Errorf("scribe response has empty text")
	}
	return ports.Transcript{Text: response.Text}, nil
}

func parseResponseBytes(data []byte) (ports.Transcript, error) {
	return parseResponse(bytes.NewReader(data))
}
