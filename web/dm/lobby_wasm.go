//go:build js && wasm

package dm

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// LobbyComponent is the concept-matched title and lobby surface for the DM.
// It renders live room data over the generated harbor art and keeps a CSS
// fallback when the gRPC asset loader is not installed yet.
func LobbyComponent(model LobbyModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(model.Locale)
		aspect := currentAspectClass()
		art := titleArtFor(aspect, ArtURL)
		return html.Main(html.Props{Class: "df-dm-lobby " + aspect, Role: "main", Style: titleSurfaceStyle(art.Background)},
			html.Div(html.Props{Class: "df-title-vignette", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"position": "absolute", "inset": "0", "z-index": "0", "pointer-events": "none", "box-shadow": "inset 0 0 12vw rgba(0, 0, 0, .72), inset 0 -16vh 18vh rgba(0, 0, 0, .48)"}}),
			titleResponsiveStyles(),
			titleBrand(art.Wordmark, locale),
			html.Div(html.Props{Class: "df-lobby-layout", Style: lobbyLayoutStyle()},
				joinPanel(locale, model, art),
				partyPanel(locale, model, art),
				audioPanel(locale, model, art),
			),
		)
	}
}

func titleResponsiveStyles() ui.Node {
	return html.Tag("style", html.Props{ID: "df-dm-title-styles"}, html.Text(`
.df-dm-lobby .df-title-brand,.df-dm-lobby .df-lobby-layout{z-index:2}
.df-dm-lobby .df-lobby-code{min-width:0}
.df-dm-lobby .df-lobby-qr-fallback{position:absolute;inset:22%;background:repeating-conic-gradient(#17202b 0 25%,#efe6d2 0 50%) 50%/18px 18px;opacity:.7;z-index:0}
.df-dm-lobby .df-lobby-qr{position:relative;z-index:1}
.df-dm-lobby .df-lobby-seat-number,.df-dm-lobby .df-lobby-seat-name,.df-dm-lobby .df-lobby-seat-status{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.df-dm-lobby.df-aspect-ultrawide .df-title-brand{left:6%;width:min(43vw,58rem)}
.df-dm-lobby.df-aspect-ultrawide .df-lobby-layout{left:6%!important;right:6%!important;max-height:43%}
.df-dm-lobby.df-aspect-laptop .df-title-brand{width:48vw}
.df-dm-lobby.df-aspect-projector .df-lobby-layout{max-height:49%;gap:.55rem}
.df-dm-lobby.df-aspect-projector .df-lobby-qr-frame{max-width:9rem}
.df-dm-lobby.df-aspect-projector .df-lobby-intro{display:none}
.df-dm-lobby.df-aspect-portrait .df-title-brand{top:1.25rem;left:8%;width:84%}
.df-dm-lobby.df-aspect-portrait .df-lobby-layout{inset:35% 1rem 1rem!important;display:grid;grid-template-columns:1fr!important;max-height:none;overflow:auto;padding-right:.2rem}
.df-dm-lobby.df-aspect-portrait .df-title-panel{padding:.8rem}
.df-dm-lobby.df-aspect-portrait .df-lobby-qr-frame{max-width:10rem}
@media (max-width:900px){.df-dm-lobby .df-lobby-layout{grid-template-columns:1fr 1.35fr!important}.df-dm-lobby .df-lobby-audio{display:none}.df-dm-lobby .df-title-brand{width:58vw}}
@media (prefers-reduced-motion:reduce){.df-dm-lobby *{transition:none!important}}
`))
}

func titleBrand(wordmark, locale string) ui.Node {
	if wordmark != "" {
		return html.Header(html.Props{Class: "df-title-brand", Style: titleBrandStyle()},
			html.Img(html.Props{Src: wordmark, Alt: "DungeonFlux", Class: "df-title-wordmark", Style: map[string]string{"display": "block", "width": "100%", "max-height": "22vh", "object-fit": "contain"}}),
			html.P(html.Props{Class: "df-title-tagline", Style: titleTaglineStyle()}, ui.Text(LobbyIntro(locale))),
		)
	}
	return html.Header(html.Props{Class: "df-title-brand df-title-brand-fallback", Style: titleBrandStyle()},
		html.H1(html.Props{Class: "df-title-wordmark-text", Style: map[string]string{"margin": "0", "color": "#e6c276", "font-family": "Georgia, 'Times New Roman', serif", "font-size": "clamp(3rem, 7vw, 7rem)", "line-height": "1", "text-shadow": "0 4px 18px #000"}}, ui.Text("DungeonFlux")),
		html.P(html.Props{Class: "df-title-tagline", Style: titleTaglineStyle()}, ui.Text(LobbyIntro(locale))),
	)
}

func titleBrandStyle() map[string]string {
	return map[string]string{"position": "absolute", "top": "clamp(1.5rem, 4vh, 3.8rem)", "left": "clamp(1rem, 3vw, 4rem)", "z-index": "2", "width": "min(49vw, 58rem)", "text-align": "center", "filter": "drop-shadow(0 8px 14px rgba(0, 0, 0, .7))"}
}

func titleTaglineStyle() map[string]string {
	return map[string]string{"margin": ".25rem auto 0", "max-width": "40rem", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(.82rem, 1.25vw, 1.35rem)", "letter-spacing": ".08em", "text-shadow": "0 2px 5px #000"}
}

func titleSurfaceStyle(background string) map[string]string {
	image := "linear-gradient(180deg, rgba(7, 11, 18, .38), rgba(7, 10, 15, .76)), linear-gradient(135deg, #13283c, #0f1117 70%)"
	if background != "" {
		image = "linear-gradient(180deg, rgba(7, 11, 18, .25), rgba(7, 10, 15, .72)), url(\"" + background + "\")"
	}
	return map[string]string{
		"position": "relative", "width": "100%", "height": "100%", "min-height": "100%",
		"overflow": "hidden", "padding": "clamp(1rem, 3.4vw, 4rem)", "box-sizing": "border-box",
		"background-color": "#0f1117", "background-image": image, "background-size": "cover", "background-position": "center",
		"color": "#efe6d2", "font-family": "Inter, ui-sans-serif, system-ui, sans-serif",
	}
}

func lobbyLayoutStyle() map[string]string {
	return map[string]string{
		"position": "absolute", "inset": "auto clamp(1rem, 2.2vw, 3rem) clamp(1rem, 2.7vw, 3rem)",
		"display": "grid", "grid-template-columns": "minmax(15rem, .95fr) minmax(24rem, 1.45fr) minmax(13rem, .72fr)",
		"gap": "clamp(.65rem, 1.4vw, 1.8rem)", "align-items": "stretch", "max-height": "44%",
	}
}

func titlePanelStyle(frame string) map[string]string {
	image := "linear-gradient(145deg, rgba(19, 28, 38, .96), rgba(10, 15, 22, .96))"
	if frame != "" {
		image = "linear-gradient(145deg, rgba(19, 28, 38, .78), rgba(10, 15, 22, .9)), url(\"" + frame + "\")"
	}
	return map[string]string{
		"position": "relative", "min-width": "0", "padding": "clamp(.8rem, 1.4vw, 1.5rem)",
		"border": "1px solid rgba(217, 164, 65, .42)", "border-radius": "10px", "background-color": "rgba(10, 15, 22, .94)",
		"background-image": image, "background-size": "100% 100%", "box-shadow": "0 12px 38px rgba(0, 0, 0, .38), inset 0 0 24px rgba(0, 0, 0, .25)",
		"overflow": "hidden", "color": "#efe6d2",
	}
}

func panelHeadingStyle() map[string]string {
	return map[string]string{"margin": "0 0 .7rem", "color": "#d9a441", "font-family": "Georgia, serif", "font-size": "clamp(.78rem, 1vw, 1.05rem)", "font-weight": "700", "letter-spacing": ".12em", "text-transform": "uppercase"}
}

func joinPanel(locale string, model LobbyModel, art titleArt) ui.Node {
	return html.Section(html.Props{Class: "df-lobby-join df-title-panel", Aria: map[string]string{"label": RoomCodeLabel(locale)}, Style: titlePanelStyle(art.PanelFrame)},
		html.H2(html.Props{Style: panelHeadingStyle()}, ui.Text(RoomCodeLabel(locale))),
		html.Div(html.Props{Class: "df-lobby-qr-frame", Style: qrFrameStyle(art.QRFrame)},
			html.Div(html.Props{Class: "df-lobby-qr-fallback", Aria: map[string]string{"hidden": "true"}}),
			html.Img(html.Props{Src: model.QRURL, Alt: T(locale, "dm.qr_alt", nil), Class: "df-lobby-qr", Style: map[string]string{"display": "block", "width": "100%", "height": "100%", "min-width": "0", "object-fit": "contain", "background": "#f4ead5"}}),
		),
		html.Div(html.Props{Class: "df-lobby-code"},
			html.Code(html.Props{Class: "df-lobby-code-value", Style: map[string]string{"display": "block", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(1.45rem, 2.8vw, 3rem)", "font-weight": "700", "letter-spacing": ".16em", "overflow-wrap": "anywhere", "text-align": "center"}}, ui.Text(model.RoomCode)),
			html.A(html.Props{Href: model.JoinURL, Class: "df-lobby-url", Aria: map[string]string{"label": "Player join URL"}, Style: map[string]string{"display": "block", "margin-top": ".45rem", "color": "#a89f8c", "font-size": "clamp(.72rem, .95vw, 1rem)", "overflow-wrap": "anywhere", "text-align": "center"}}, ui.Text(model.JoinURL)),
		),
	)
}

func qrFrameStyle(frame string) map[string]string {
	style := map[string]string{"position": "relative", "display": "grid", "place-items": "center", "width": "100%", "max-width": "13rem", "margin": "0 auto .7rem", "aspect-ratio": "1", "padding": "18%", "box-sizing": "border-box", "background-color": "#11151b", "background-size": "100% 100%"}
	if frame != "" {
		style["background-image"] = "url(\"" + frame + "\")"
	} else {
		style["border"] = "1px solid rgba(217, 164, 65, .52)"
	}
	return style
}

func partyPanel(locale string, model LobbyModel, art titleArt) ui.Node {
	return html.Section(html.Props{Class: "df-lobby-party df-title-panel", Aria: map[string]string{"label": SeatsLabel(locale)}, Style: titlePanelStyle(art.PanelFrame)},
		html.H2(html.Props{Style: panelHeadingStyle()}, ui.Text(SeatsLabel(locale))),
		html.Div(html.Props{Class: "df-lobby-seats", Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(2, minmax(0, 1fr))", "gap": "clamp(.45rem, .8vw, .9rem)"}}, seatCard(locale, model.Seats[0]), seatCard(locale, model.Seats[1])),
		html.Div(html.Props{Class: "df-lobby-divider", Style: dividerStyle(art.Divider)}),
		html.P(html.Props{Class: "df-lobby-intro"}, ui.Text(LobbyIntro(locale))),
	)
}

func audioPanel(locale string, model LobbyModel, art titleArt) ui.Node {
	return html.Section(html.Props{Class: "df-lobby-audio df-title-panel", Aria: map[string]string{"label": ListenLabel(locale)}, Style: titlePanelStyle(art.PanelFrame)},
		html.H2(html.Props{Style: panelHeadingStyle()}, ui.Text(ListenLabel(locale))),
		html.Div(html.Props{Class: "df-lobby-audio-mark", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"margin": "1rem 0", "color": "#d9a441", "font-size": "clamp(2rem, 4vw, 4rem)", "text-align": "center", "text-shadow": "0 0 18px rgba(217, 164, 65, .45)"}}, html.Text("◈")),
		html.P(html.Props{Class: "df-lobby-audio-state", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"margin": "0", "color": "#a89f8c", "font-size": "clamp(.8rem, 1.1vw, 1.2rem)", "text-align": "center"}}, ui.Text(model.AudioState)),
		html.Audio(html.Props{ID: "df-opening-audio", Hidden: true, Src: model.OpeningAudio, Raw: map[string]any{"controls": true}}),
	)
}

func dividerStyle(divider string) map[string]string {
	style := map[string]string{"height": "1.5rem", "margin": ".2rem 0 .5rem", "background-position": "center", "background-repeat": "no-repeat", "background-size": "100% 100%", "opacity": ".85"}
	if divider != "" {
		style["background-image"] = "url(\"" + divider + "\")"
	}
	return style
}

func seatCard(locale string, seat Seat) ui.Node {
	status := SeatStatus(locale, seat.Joined, seat.Ready)
	border := "rgba(168, 159, 140, .38)"
	if seat.Joined {
		border = "rgba(58, 163, 154, .72)"
	}
	return html.Div(html.Props{Class: "df-lobby-seat", Role: "listitem", Style: map[string]string{"min-width": "0", "padding": "clamp(.6rem, 1vw, 1rem)", "border": "1px solid " + border, "border-radius": "7px", "background": "rgba(7, 11, 17, .58)"}, Raw: map[string]any{"data-seat": seat.Number, "data-joined": seat.Joined, "data-ready": seat.Ready}},
		html.P(html.Props{Class: "df-lobby-seat-number", Style: map[string]string{"margin": "0 0 .35rem", "color": "#a89f8c", "font-size": "clamp(.6rem, .75vw, .8rem)", "font-weight": "700", "letter-spacing": ".1em"}}, ui.Text("SEAT "+strconv.Itoa(seat.Number))),
		html.H3(html.Props{Class: "df-lobby-seat-name", Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(1rem, 1.5vw, 1.7rem)", "overflow-wrap": "anywhere"}}, ui.Text(SeatName(locale, seat.Name, seat.Number))),
		html.P(html.Props{Class: "df-lobby-seat-status", Aria: map[string]string{"label": "Seat status"}, Style: map[string]string{"margin": ".45rem 0 0", "color": "#3aa39a", "font-size": "clamp(.7rem, .9vw, 1rem)", "font-weight": "700"}}, ui.Text(status)),
	)
}
