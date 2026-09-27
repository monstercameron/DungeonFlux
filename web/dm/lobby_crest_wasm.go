//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// lobbyCrestCSS positions the animated lantern-crest loop over the still crest
// baked into the ui/logo_wordmark art, in the same box the wordmark's own
// background-image mask uses (df-lobby-title-plate at 740px wide, dmLobbyFinishCSS).
// The box below is the crest's source region (x520-981, y75-585 of the 1536x1024
// wordmark) scaled by 740/1536. The clip has an opaque black background, so the
// box blends with lighten (black drops out, the lantern and embers stay) and a
// feathered mask removes the rectangle's edges; the still crest underneath
// fills in anything the mask fades.
const lobbyCrestCSS = `
.df-lobby-crest{position:absolute;left:250px;top:36px;width:222px;height:246px;overflow:hidden;pointer-events:none;z-index:1;mix-blend-mode:lighten;-webkit-mask-image:radial-gradient(ellipse 50% 50% at 50% 50%,#000 62%,transparent 100%);mask-image:radial-gradient(ellipse 50% 50% at 50% 50%,#000 62%,transparent 100%)}
.df-lobby-crest-video{display:block;width:100%;height:100%;object-fit:cover;object-position:center;filter:brightness(1.3) contrast(1.18) saturate(1.1)}
`

// lobbyCrestLoop renders the looping lantern-crest video over the lobby title
// plate's still crest art. It is muted, autoplaying, and loops silently; a
// stable Key keeps GWC from tearing the element down (and restarting playback)
// on unrelated lobby re-renders. It plays even when the OS asks for reduced
// motion: the TV is a shared display, and the laptop driving it should not
// silently strip the lobby's one piece of ambient motion (developer decision,
// 2026-09-27).
func lobbyCrestLoop() ui.Node {
	injectStyleSheet("df-dm-lobby-crest", lobbyCrestCSS)
	videoURL := ArtURL("ui/logo_crest_loop")
	webmURL := ArtURL("ui/logo_crest_loop_webm")
	if videoURL == "" && webmURL == "" {
		return html.Div(html.Props{Hidden: true})
	}
	posterURL := ArtURL("ui/logo_crest_poster")
	sources := make([]ui.Node, 0, 2)
	if webmURL != "" {
		sources = append(sources, html.Source(html.Props{Src: webmURL, Type: "video/webm"}))
	}
	if videoURL != "" {
		sources = append(sources, html.Source(html.Props{Src: videoURL, Type: "video/mp4"}))
	}
	return html.Div(html.Props{Class: "df-lobby-crest", Aria: map[string]string{"hidden": "true"}},
		html.Video(html.Props{Key: "lobby-crest-video", Class: "df-lobby-crest-video", Raw: map[string]any{
			"autoplay": true, "muted": true, "loop": true, "playsinline": true, "poster": posterURL,
		}}, sources...),
	)
}
