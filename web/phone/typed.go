package phone

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

const typedInputLimit = 280

// SayResult is the result of an asynchronous typed-input Say call.
type SayResult struct {
	Value *df.SayResponse
	Err   error
}

// SayClient is the smallest client surface required by typed input.
type SayClient interface {
	Say(context.Context, *df.SayRequest) <-chan SayResult
}

// TypedInputSnapshot is the render-safe state of the typed-input fallback.
type TypedInputSnapshot struct {
	SeatToken   string
	Text        string
	StatusText  string
	Error       string
	UtteranceID string
	Open        bool
	Sending     bool
	Locale      string
	Characters  int
	CanSubmit   bool
}

// TypedInputModel owns the fallback text box and Say request construction.
type TypedInputModel struct {
	client SayClient
	state  TypedInputSnapshot
	locale string
}

// SetLocale settles the render locale for typed-input statuses.
func (m *TypedInputModel) SetLocale(locale string) {
	if m == nil {
		return
	}
	if locale == "" {
		locale = "en"
	}
	m.locale = locale
	m.state.Locale = locale
}

func (m *TypedInputModel) renderLocale() string {
	if m == nil || m.locale == "" {
		return "en"
	}
	return m.locale
}

// NewTypedInputModel creates a typed-input model for one seat.
func NewTypedInputModel(client SayClient, seatToken string) *TypedInputModel {
	return &TypedInputModel{client: client, state: TypedInputSnapshot{
		SeatToken: strings.TrimSpace(seatToken),
	}}
}

// Snapshot returns a copy of the current typed-input state.
func (m *TypedInputModel) Snapshot() TypedInputSnapshot {
	if m == nil {
		return TypedInputSnapshot{Error: "typed input is unavailable"}
	}
	snapshot := m.state
	snapshot.Characters = utf8.RuneCountInString(snapshot.Text)
	snapshot.CanSubmit = snapshot.Open && !snapshot.Sending && strings.TrimSpace(snapshot.Text) != "" && snapshot.Characters <= typedInputLimit
	return snapshot
}

// SetText records text from the phone input without sending it.
func (m *TypedInputModel) SetText(text string) error {
	if m == nil {
		return errors.New("typed input is unavailable")
	}
	if utf8.RuneCountInString(text) > typedInputLimit {
		return errors.New("message must be 280 characters or fewer")
	}
	m.state.Text = text
	m.state.Error = ""
	m.state.Open = true
	return nil
}

// OpenFallback opens the text box after a speech-to-text failure.
func (m *TypedInputModel) OpenFallback() TypedInputSnapshot {
	if m == nil {
		return TypedInputSnapshot{Error: "typed input is unavailable"}
	}
	m.state.Open = true
	m.state.StatusText = T(m.renderLocale(), "ui.ptt.failed", nil)
	m.state.Error = ""
	return m.Snapshot()
}

// CloseFallback hides the typed input after speech becomes available again.
func (m *TypedInputModel) CloseFallback() TypedInputSnapshot {
	if m == nil {
		return TypedInputSnapshot{Error: "typed input is unavailable"}
	}
	m.state.Open = false
	m.state.Error = ""
	return m.Snapshot()
}

// Submit sends the current text through SessionService.Say.
func (m *TypedInputModel) Submit(ctx context.Context) <-chan SayResult {
	result := make(chan SayResult, 1)
	if m == nil || m.client == nil {
		result <- SayResult{Err: errors.New("typed input client is unavailable")}
		return result
	}
	text := strings.TrimSpace(m.state.Text)
	if text == "" {
		result <- SayResult{Err: errors.New("message is required")}
		return result
	}
	if utf8.RuneCountInString(text) > typedInputLimit {
		result <- SayResult{Err: errors.New("message must be 280 characters or fewer")}
		return result
	}
	m.state.Text, m.state.Sending, m.state.Error = text, true, ""
	return m.client.Say(ctx, &df.SayRequest{SeatToken: m.state.SeatToken, Text: text})
}

// ApplySay updates the model after a Say response arrives.
func (m *TypedInputModel) ApplySay(result SayResult) TypedInputSnapshot {
	if m == nil {
		return TypedInputSnapshot{Error: "typed input is unavailable"}
	}
	m.state.Sending = false
	if result.Err != nil {
		m.state.Error = result.Err.Error()
		return m.Snapshot()
	}
	if result.Value == nil {
		m.state.Error = "typed message returned no response"
		return m.Snapshot()
	}
	if !result.Value.GetAccepted() {
		m.state.Error = result.Value.GetReason()
		if m.state.Error == "" {
			m.state.Error = "server rejected typed message"
		}
		return m.Snapshot()
	}
	m.state.Error, m.state.StatusText = "", TypedSent(m.renderLocale())
	m.state.UtteranceID = result.Value.GetUtteranceId()
	m.state.Text, m.state.Open = "", false
	return m.Snapshot()
}
