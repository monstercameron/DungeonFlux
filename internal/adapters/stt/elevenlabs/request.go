package elevenlabs

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

const scribeModel = "scribe_v2"

func buildRequest(req ports.STTRequest) (*bytes.Buffer, string, error) {
	if len(req.Audio) == 0 {
		return nil, "", fmt.Errorf("audio is required")
	}
	mimeType := strings.TrimSpace(req.MIME)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	filename := filenameForMIME(mimeType)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	header.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, "", fmt.Errorf("create audio part: %w", err)
	}
	if _, err := part.Write(req.Audio); err != nil {
		return nil, "", fmt.Errorf("write audio part: %w", err)
	}
	if err := writer.WriteField("model_id", scribeModel); err != nil {
		return nil, "", fmt.Errorf("write model field: %w", err)
	}
	for _, keyterm := range req.Keyterms {
		if strings.TrimSpace(keyterm) == "" {
			continue
		}
		if err := writer.WriteField("keyterms", keyterm); err != nil {
			return nil, "", fmt.Errorf("write keyterm field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart request: %w", err)
	}
	return body, writer.FormDataContentType(), nil
}

func filenameForMIME(mimeType string) string {
	extension := "bin"
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(mimeType, ";", 2)[0])) {
	case "audio/webm":
		extension = "webm"
	case "audio/ogg":
		extension = "ogg"
	case "audio/mpeg":
		extension = "mp3"
	case "audio/wav", "audio/x-wav":
		extension = "wav"
	case "audio/mp4", "video/mp4":
		extension = "mp4"
	}
	return "audio." + extension
}
