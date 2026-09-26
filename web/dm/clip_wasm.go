//go:build js && wasm

package dm

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// ClipComponent renders a playable clip and its still fallback for the DM TV.
func ClipComponent(model ClipModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		if model.UseFallback || model.VideoURL == "" {
			return html.Div(html.Props{Class: "df-dm-clip df-dm-clip-fallback", Role: "img", Aria: map[string]string{"label": "Animated scene still"}},
				html.Img(html.Props{Class: "df-dm-clip-still", Src: model.StillURL, Alt: ""}),
			)
		}
		return html.Div(html.Props{Class: "df-dm-clip", Role: "img", Aria: map[string]string{"label": "DungeonFlux scene clip"}},
			html.Video(html.Props{Class: "df-dm-clip-video", Src: model.VideoURL, Raw: map[string]any{
				"autoplay": model.Playing, "muted": true, "playsinline": true, "poster": model.StillURL,
				"data-offset-ms": strconv.FormatInt(model.OffsetMS, 10),
			}}, ui.Text("Your browser cannot play this scene clip.")),
		)
	}
}
