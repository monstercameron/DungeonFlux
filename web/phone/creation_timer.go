package phone

import (
	"strconv"
	"strings"
)

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
