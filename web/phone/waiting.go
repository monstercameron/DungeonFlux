package phone

import (
	"fmt"
	"strings"
)

// WaitingSeat is the stable, display-safe lobby summary for one seat.
type WaitingSeat struct {
	Number int32
	Name   string
	Joined bool
}

// WaitingModel contains the content needed by the lobby waiting screen.
type WaitingModel struct {
	PlayerName   string
	PlayerNumber int32
	Joined       []WaitingSeat
}

// NewWaitingModel projects lobby metadata from a seat view.
func NewWaitingModel(view SeatView) WaitingModel {
	model := WaitingModel{PlayerName: strings.TrimSpace(view.PlayerName), PlayerNumber: view.PlayerNumber}
	if model.PlayerName == "" && view.Phone != nil && view.Phone.GetCharacter() != nil {
		model.PlayerName = strings.TrimSpace(view.Phone.GetCharacter().GetName())
	}
	if model.PlayerName == "" {
		model.PlayerName = "Player"
	}
	if model.PlayerNumber < 1 {
		model.PlayerNumber = 1
	}
	for index, seat := range view.LobbySeats {
		if seat == nil || !seat.GetJoined() {
			continue
		}
		number := seat.GetPlayerNumber()
		if number < 1 {
			number = int32(index + 1)
		}
		name := strings.TrimSpace(seat.GetName())
		if name == "" {
			name = fmt.Sprintf("Player %d", number)
		}
		model.Joined = append(model.Joined, WaitingSeat{Number: number, Name: name, Joined: true})
	}
	return model
}

// WaitingStatus returns the concise status line shown beneath the seat card.
func WaitingStatus(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "es") {
		return "Esperando a que el anfitrión comience"
	}
	return "Waiting for the host to start"
}

// WaitingSeatLabel formats a seat number consistently across locales.
func WaitingSeatLabel(locale string, number int32) string {
	if strings.EqualFold(strings.TrimSpace(locale), "es") {
		return fmt.Sprintf("Asiento %d", number)
	}
	return fmt.Sprintf("Seat %d", number)
}

func joinedSummary(locale string, count int) string {
	if strings.EqualFold(strings.TrimSpace(locale), "es") {
		return plural(count, "jugador unido", "jugadores unidos")
	}
	return plural(count, "player joined", "players joined")
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", count, plural)
}
