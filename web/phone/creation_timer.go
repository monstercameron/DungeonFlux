package phone

import (
	"strconv"
	"strings"
)

// creationTimerRemainingAt projects a server timer onto a local monotonic
// clock without changing the authoritative value stored in the model.
func creationTimerRemainingAt(remainingMS int64, anchorMS, nowMS float64) int64 {
	if nowMS <= anchorMS {
		return remainingMS
	}
	remainingMS -= int64(nowMS - anchorMS)
	if remainingMS < 0 {
		return 0
	}
	return remainingMS
}

func creationTimerSeconds(remainingMS int64) int64 {
	if remainingMS <= 0 {
		return 0
	}
	return (remainingMS + 999) / 1000
}

func creationTimerLabel(locale string, remainingMS int64, frozen bool) string {
	seconds := strconv.FormatInt(creationTimerSeconds(remainingMS), 10)
	if strings.HasPrefix(strings.ToLower(locale), "es") {
		if frozen {
			return "Creación pausada · " + seconds + " s"
		}
		return "Tiempo para elegir · " + seconds + " s"
	}
	if frozen {
		return "Creation paused · " + seconds + "s"
	}
	return "Time to choose · " + seconds + "s"
}
