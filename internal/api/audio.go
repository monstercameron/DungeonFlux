package api

import df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

// AudioClient describes the authenticated audience for an AudioService stream.
type AudioClient struct {
	Kind         df.ClientKind
	PlayerNumber int
}

// AudioClientForToken authenticates a DM or phone token for Listen. Host
// tokens are intentionally excluded because hosts do not receive table audio.
func (s *SessionServer) AudioClientForToken(token string) (AudioClient, bool) {
	if s == nil || token == "" {
		return AudioClient{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if token == s.dmToken {
		return AudioClient{Kind: df.ClientKind_CLIENT_KIND_DM}, true
	}
	seat, ok := s.seats[token]
	if !ok {
		return AudioClient{}, false
	}
	return AudioClient{Kind: df.ClientKind_CLIENT_KIND_PHONE, PlayerNumber: seat.playerNumber}, true
}
