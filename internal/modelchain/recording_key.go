package modelchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// A recording is a position in a rehearsal, independent of the live prompt
// and run ID. Its namespace must still identify the exact response contract.
// V2 intentionally does not read legacy entries, which may already contain
// replies overwritten by another role, language or output format.
func recordingKey(adapter string, req ports.TextRequest, format string, schema ports.Schema) (ports.RecKey, error) {
	contract := struct {
		Adapter string
		Role    vocab.Role
		Locale  string
		Format  string
		Schema  ports.Schema
	}{adapter, req.Meta.Role, req.Meta.Locale, format, schema}
	raw, err := json.Marshal(contract)
	if err != nil {
		return ports.RecKey{}, fmt.Errorf("modelchain: encode recording contract: %w", err)
	}
	digest := sha256.Sum256(raw)
	return ports.RecKey{
		Adapter: "recording-v2:" + hex.EncodeToString(digest[:]),
		Phase:   req.Meta.Phase, Seat: req.Meta.Seat, Index: req.Meta.Index,
	}, nil
}
