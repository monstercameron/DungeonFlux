package main

import (
	"context"
	"errors"
	"net/http"
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

// NudgeTTSPlan returns the request and character estimate for timer nudges.
func NudgeTTSPlan() TTSPlan {
	lines := NudgeLines()
	canned := make([]CannedLine, 0, len(lines))
	for _, line := range lines {
		canned = append(canned, CannedLine(line))
	}
	return planCannedLines(canned)
}

// RenderNudgeLine requests one nudge and records it in the manifest.
func RenderNudgeLine(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, line NudgeLine, take int) error {
	if line.ID == "" || line.Voice == "" || line.Text == "" {
		return errors.New("buildtime: nudge requires id, voice, and text")
	}
	if err := RenderCannedLine(ctx, client, endpoint, outputDir, writer, CannedLine(line), take); err != nil {
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
		return withManifestLock(ctx, writer, func(writer *ManifestWriter) error {
			files := make([]string, 0, len(NudgeLines()))
			for _, line := range NudgeLines() {
				canned := CannedLine(line)
				result, err := renderCannedLine(ctx, client, endpoint, audioOutputDir(outputDir), writer, canned, take, true)
				if err != nil {
					return err
				}
				metadata := audioMetadata(canned, result)
				metadata["name_free"] = "true"
				if err := writer.SetMetadata(line.ID, result.DurationMS, 0, metadata); err != nil {
					return err
				}
				files = append(files, summaryFile(result, audioOutputDir(outputDir)))
			}
			logTTSSummary("turn-nudges", files)
			return nil
		})
	}}
}
