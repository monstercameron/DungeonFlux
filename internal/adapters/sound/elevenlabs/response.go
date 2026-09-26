package elevenlabs

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func parseResponse(response *http.Response, kind vocab.SoundKind) (ports.Sound, error) {
	if response == nil || response.Body == nil {
		return ports.Sound{}, errors.New("sound: empty response")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ports.Sound{}, fmt.Errorf("sound: ElevenLabs returned %s", response.Status)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return ports.Sound{}, fmt.Errorf("sound: read response: %w", err)
	}
	if len(data) == 0 {
		return ports.Sound{}, errors.New("sound: response audio is empty")
	}
	mime := "audio/mpeg"
	if kind == vocab.SoundMusic {
		mime = "audio/mpeg"
	}
	return ports.Sound{Bytes: data, MIME: mime}, nil
}
