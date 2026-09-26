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
		locale := localeOrDefault(model.Locale)
		mediaStyle := map[string]string{"width": "100%", "height": "100%", "object-fit": "cover", "object-position": "center 42%", "filter": "saturate(.9) contrast(1.04)"}
		if model.Opening {
			mediaStyle["opacity"] = ".12"
		}
		if model.UseFallback || model.VideoURL == "" {
			return html.Div(html.Props{Class: "df-dm-clip df-dm-clip-fallback", Role: "img", Aria: map[string]string{"label": T(locale, "dm.clip_still", nil)}, Style: map[string]string{"position": "absolute", "left": "0", "top": "0", "width": "1920px", "height": "1080px", "overflow": "hidden", "background": "linear-gradient(180deg, rgba(13,17,24,.06), rgba(8,10,15,.68))"}},
				html.Img(html.Props{Class: "df-dm-clip-still", Src: model.StillURL, Alt: "", Style: mediaStyle}),
			)
		}
		return html.Div(html.Props{Class: "df-dm-clip", Role: "img", Aria: map[string]string{"label": T(locale, "dm.clip_label", nil)}, Style: map[string]string{"position": "absolute", "left": "0", "top": "0", "width": "1920px", "height": "1080px", "overflow": "hidden", "background": "#0b0d12"}},
			html.Img(html.Props{Class: "df-dm-clip-still", Src: model.StillURL, Alt: "", Style: cloneMediaStyle(mediaStyle, true), Raw: map[string]any{"aria-hidden": "true"}}),
			html.Video(html.Props{Class: "df-dm-clip-video", Src: model.VideoURL, Style: cloneMediaStyle(mediaStyle, true), Raw: map[string]any{
				"autoplay": model.Playing, "muted": true, "playsinline": true, "poster": model.StillURL,
				"data-offset-ms": strconv.FormatInt(model.OffsetMS, 10), "data-start-seconds": strconv.FormatFloat(model.ClipStartSeconds(), 'f', 3, 64),
			}}, ui.Text(T(locale, "dm.clip_fallback", nil))),
		)
	}
}

func cloneMediaStyle(style map[string]string, absolute bool) map[string]string {
	result := make(map[string]string, len(style)+2)
	for key, value := range style {
		result[key] = value
	}
	if absolute {
		result["position"], result["inset"] = "absolute", "0"
	}
	return result
}
