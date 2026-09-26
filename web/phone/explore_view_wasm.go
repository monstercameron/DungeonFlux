//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// ExploreScreen renders the scene still, DM narration, and legal exploration moves.
func ExploreScreen(model *MovesModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		snapshot := NewExploreSnapshot(model.Snapshot())
		rows := make([]ui.Node, 0, len(snapshot.Moves))
		for _, row := range snapshot.Moves {
			item := row
			tap := ui.UseEvent(func() {
				go func() {
					model.ApplyAct(<-model.Tap(context.Background(), item.ID))
					refresh.Set(refresh.Get() + 1)
				}()
			})
			rows = append(rows, ChoiceRow(item, tap))
		}
		if len(rows) == 0 {
			rows = append(rows, html.P(html.Props{Style: exploreMutedStyle()}, html.Text("The room offers no clear path yet.")))
		}
		if errorText := model.Snapshot().Error; errorText != "" {
			rows = append(rows, html.P(html.Props{Role: "alert", Style: exploreErrorStyle()}, html.Text(errorText)))
		}
		return html.Main(html.Props{Class: "df-phone df-phone-moves df-phone-explore", Role: "main", Style: explorePageStyle()},
			html.Div(html.Props{Class: "df-phone-explore-scene", Style: exploreSceneStyle(snapshot.SceneArt)},
				html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "14px", "right": "14px", "top": "14px", "color": DefaultPhoneTheme().GoldBright, "font-family": DefaultPhoneTheme().Serif, "font-size": "13px", "letter-spacing": ".08em", "text-transform": "uppercase"}}, html.Text(snapshot.Location)),
			),
			html.Div(html.Props{Class: "df-phone-explore-narration", Style: exploreNarrationStyle()}, NarrationCard("Dungeon Master", snapshot.Narration, ArtURL("ui/dm_speaker"))),
			html.H1(html.Props{Style: map[string]string{"margin": "0", "color": DefaultPhoneTheme().Parchment, "font-family": DefaultPhoneTheme().Serif, "font-size": "25px"}}, html.Text(snapshot.Title)),
			html.Section(html.Props{Class: "df-phone-explore-choices", Aria: map[string]string{"label": "Exploration choices"}, Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "8px"}}, rows...),
		)
	}
}

func explorePageStyle() map[string]string {
	theme := DefaultPhoneTheme()
	return map[string]string{"display": "flex", "flex-direction": "column", "gap": "10px", "min-height": "100%", "color": theme.Parchment, "font-family": theme.Sans}
}

func exploreSceneStyle(asset string) map[string]string {
	theme := DefaultPhoneTheme()
	style := map[string]string{"position": "relative", "height": "270px", "overflow": "hidden", "border": "1px solid rgba(217,164,65,.62)", "border-radius": theme.BorderRadius, "background": "linear-gradient(180deg,rgba(6,10,15,.08),rgba(6,10,15,.84)), radial-gradient(circle at 50% 35%,#3a4550,#10141c 72%)", "background-size": "cover", "background-position": "center"}
	if url := ArtURL(asset); url != "" {
		style["background-image"] = "linear-gradient(180deg,rgba(6,10,15,.08),rgba(6,10,15,.84)), url(\"" + url + "\")"
	}
	return style
}

func exploreNarrationStyle() map[string]string {
	return map[string]string{"margin-top": "-34px", "position": "relative", "z-index": "1"}
}

func exploreMutedStyle() map[string]string {
	theme := DefaultPhoneTheme()
	return map[string]string{"margin": "0", "padding": "12px", "border": "1px solid rgba(168,159,140,.3)", "border-radius": theme.BorderRadius, "color": theme.Muted, "font-family": theme.Serif, "font-style": "italic"}
}

func exploreErrorStyle() map[string]string {
	theme := DefaultPhoneTheme()
	return map[string]string{"margin": "0", "padding": "10px", "border": "1px solid " + theme.Blood, "border-radius": "8px", "color": "#ffd8d2", "font-size": "13px"}
}
