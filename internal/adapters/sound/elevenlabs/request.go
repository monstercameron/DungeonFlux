package elevenlabs

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const musicModel = "music_v2_5"

type sfxRequest struct {
	Text            string  `json:"text"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	PromptInfluence float64 `json:"prompt_influence,omitempty"`
}

type musicRequest struct {
	ModelID         string `json:"model_id"`
	Seed            uint32 `json:"seed,omitempty"`
	StoreForInpaint bool   `json:"store_for_inpainting"`
	CompositionPlan struct {
		Chunks []musicChunk `json:"chunks"`
	} `json:"composition_plan"`
}

type musicChunk struct {
	Text          string   `json:"text"`
	DurationMS    int      `json:"duration_ms"`
	PositiveStyle []string `json:"positive_styles,omitempty"`
	NegativeStyle []string `json:"negative_styles,omitempty"`
}

func buildRequest(request ports.SoundRequest) (string, []byte, error) {
	if request.Prompt == "" {
		return "", nil, errors.New("sound: prompt is required")
	}
	if request.Seconds <= 0 {
		return "", nil, errors.New("sound: duration must be positive")
	}
	switch request.Kind {
	case vocab.SoundSFX:
		body, err := json.Marshal(sfxRequest{Text: request.Prompt, DurationSeconds: request.Seconds, PromptInfluence: 0.3})
		return "/sound-generation", body, err
	case vocab.SoundMusic:
		return buildMusicRequest(request)
	default:
		return "", nil, fmt.Errorf("sound: unsupported kind %q", request.Kind)
	}
}

func buildMusicRequest(request ports.SoundRequest) (string, []byte, error) {
	seconds := request.Seconds
	if seconds > 600 {
		return "", nil, errors.New("sound: music duration exceeds ten minutes")
	}
	body := musicRequest{ModelID: musicModel, StoreForInpaint: true}
	body.CompositionPlan.Chunks = []musicChunk{{
		Text: request.Prompt + "\n{instrumental}", DurationMS: int(seconds * 1000),
		PositiveStyle: []string{"instrumental only"},
		NegativeStyle: []string{"vocals", "lyrics", "fade out"},
	}}
	data, err := json.Marshal(body)
	return "/music/detailed?output_format=mp3_44100_192", data, err
}
