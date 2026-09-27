package phone

import "testing"

func TestPreviewPTT_ReflectsDistinctRecorderStates(t *testing.T) {
	for name, want := range map[string]PTTState{"ptt-idle": PTTIdle, "ptt-recording": PTTRecording, "ptt-sending": PTTTranscribing, "typed-input": PTTFailed} {
		t.Run(name, func(t *testing.T) {
			fixture, _ := Preview(name)
			model := previewPTT(fixture.View)
			got := model.ControlSnapshot("en")
			if got.State != want || got.StatusText == "" || model.opener != nil {
				t.Fatalf("preview state = %#v", got)
			}
		})
	}
}
