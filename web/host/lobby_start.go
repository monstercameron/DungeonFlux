package host

func canStart(snapshot hostSnapshot) bool {
	return snapshot.Connected && isLobbyPhase(snapshot.Phase) && !snapshot.Paused &&
		snapshot.SeatsJoined == 2 && snapshot.SeatsReady == 2
}

func lobbyStartHint(snapshot hostSnapshot) string {
	if !isLobbyPhase(snapshot.Phase) {
		return ""
	}
	en, es := "Both players are ready. Start the adventure.", "Ambos jugadores están listos. Inicia la aventura."
	switch {
	case !snapshot.Connected:
		en, es = "Connecting to the table…", "Conectando con la mesa…"
	case snapshot.Paused:
		en, es = "Resume the game before starting.", "Reanuda la partida antes de iniciar."
	case snapshot.SeatsJoined != 2:
		en, es = "Waiting for both players to join. Use Skip to rehearse.", "Esperando a que se unan ambos jugadores. Usa Saltar para ensayar."
	case snapshot.SeatsReady != 2:
		en, es = "Ask both players to choose Ready. Use Skip to rehearse.", "Pide a ambos jugadores que pulsen Listo. Usa Saltar para ensayar."
	}
	if snapshot.Locale == "es" {
		return es
	}
	return en
}
