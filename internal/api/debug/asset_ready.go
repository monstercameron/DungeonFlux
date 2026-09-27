package debug

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// decodeBillboardAsset permits prepared local videos to use the same event
// path as generated loops. It cannot import URLs or alter arbitrary slots.
func decodeBillboardAsset(payload string) (domain.Event, error) {
	var ready domain.AssetReady
	if err := json.Unmarshal([]byte(payload), &ready); err != nil {
		return nil, err
	}
	parts := strings.Split(ready.Slot, ":")
	if len(parts) != 3 || parts[0] != "billboard" {
		return nil, errors.New("expected billboard:<seat>:<action> slot")
	}
	seat, err := strconv.Atoi(parts[1])
	if err != nil || seat < 1 || seat > 2 {
		return nil, errors.New("billboard seat must be 1 or 2")
	}
	switch parts[2] {
	case "idle", "attack", "hit", "fall":
	default:
		return nil, errors.New("unsupported billboard action")
	}
	asset := ready.Asset
	hash, err := hex.DecodeString(asset.SHA256)
	if err != nil || len(hash) != 32 || asset.URL != "/assets/"+asset.SHA256+".mp4" || string(asset.ID) != asset.SHA256 {
		return nil, errors.New("billboard must use its SHA256 local asset URL and ID")
	}
	if asset.Kind != "video" || asset.MIME != "video/mp4" || asset.DurationMS <= 0 {
		return nil, errors.New("expected a timed MP4 video asset")
	}
	return ready, nil
}
