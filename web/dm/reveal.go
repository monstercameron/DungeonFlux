package dm

import (
	"slices"
	"strings"
)

// First impression: the shell's loading splash stays up until the screen it
// covers can appear whole. revealArt names the art that must be downloaded
// and decoded first, so the TV never fades in on a half-painted lobby with
// the background, wordmark or panel frames popping in one by one.

// crestPosterAsset is the still that stands in for the crest loop until the
// video's first frame is ready.
const crestPosterAsset = "ui/logo_crest_poster"

// revealArt returns the resolved art URLs the phase needs before it is shown,
// and false while any of them is still loading (resolve returned ""). Phases
// whose stage is a video, a splat or a clip need nothing: they have their own
// loading states. qr is the lobby's join code image, a logical name or a URL.
func revealArt(phase, aspect, qr, sceneBackground string, resolve func(string) string) ([]string, bool) {
	if resolve == nil {
		return nil, false
	}
	var names []string
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "", "lobby":
		art := titleArtFor(aspect, resolve)
		urls := []string{art.Background, art.Wordmark, art.PanelFrame, resolve(crestPosterAsset)}
		if qr = strings.TrimSpace(qr); qr != "" {
			// A URL is usable as is; a logical asset name must finish loading.
			resolved := resolve(qr)
			if resolved == "" && isURL(qr) {
				resolved = qr
			}
			urls = append(urls, resolved)
		}
		return complete(urls)
	case "creation":
		names = []string{titleBackgroundWideAsset}
	case "end":
		names = []string{"ui/end_bg"}
	case "combat", "cliffhanger":
		return nil, true
	default:
		if sceneBackground == "" {
			return nil, true
		}
		return []string{sceneBackground}, true
	}
	urls := make([]string, 0, len(names))
	for _, name := range names {
		urls = append(urls, resolve(name))
	}
	return complete(urls)
}

func complete(urls []string) ([]string, bool) {
	return urls, !slices.Contains(urls, "")
}

func isURL(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "://") || strings.HasPrefix(lower, "/") || strings.HasPrefix(lower, "blob:") || strings.HasPrefix(lower, "data:")
}
