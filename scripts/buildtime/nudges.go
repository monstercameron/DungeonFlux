package main

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
)

// NudgeLine is a name-free turn-timer reminder.
type NudgeLine struct {
	ID    string
	Voice string
	Text  string
}

// NudgeLines returns the two fixed turn-timer reminders.
func NudgeLines() []NudgeLine {
	return []NudgeLine{
		{ID: "nudge_exploration", Voice: "dm", Text: "The rain won't wait."},
		{ID: "nudge_conversation", Voice: "mother_vell", Text: "Well? Speak or drink."},
	}
}

// RenderNudgeLine requests one nudge and records it in the manifest.
func RenderNudgeLine(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, line NudgeLine, take int) error {
	if line.ID == "" || line.Voice == "" || line.Text == "" {
		return errors.New("buildtime: nudge requires id, voice, and text")
	}
	if err := RenderCannedLine(ctx, client, endpoint, outputDir, writer, CannedLine{ID: line.ID, Voice: line.Voice, Text: line.Text}, take); err != nil {
		return err
	}
	return writer.SetMetadata(line.ID, 0, 0, map[string]string{
		"voice":     line.Voice,
		"name_free": "true",
	})
}

// NudgeJob returns a job that renders both turn-timer reminders.
func NudgeJob(client *http.Client, endpoint, outputDir string, take int) Job {
	return Job{Name: "turn-nudges", Run: func(ctx context.Context, writer *ManifestWriter) error {
		for _, line := range NudgeLines() {
			if err := RenderNudgeLine(ctx, client, endpoint, filepath.Clean(outputDir), writer, line, take); err != nil {
				return err
			}
		}
		return nil
	}}
}
