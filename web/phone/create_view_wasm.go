//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CreationScreen renders the touch-first character creation card.
func CreationScreen(model *CreationModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		chooseSpecies := ui.UseEvent(func() {
			_ = model.SelectSpecies("human")
			refresh.Set(refresh.Get() + 1)
		})
		chooseGender := ui.UseEvent(func() {
			_ = model.SelectGender("nonbinary")
			refresh.Set(refresh.Get() + 1)
		})
		roll := ui.UseEvent(func() {
			go func() { model.ApplyAct(<-model.RollHero(context.Background())); refresh.Set(refresh.Get() + 1) }()
		})
		snapshot := model.Snapshot()
		build := "Choose a species and gender to roll your hero."
		if snapshot.Build != nil {
			build = snapshot.Build.GetName() + " · " + snapshot.Build.GetClassName()
		}
		return html.Main(html.Props{Class: "df-phone df-phone-create"},
			html.H1(html.Props{}, html.Text("Create your hero")),
			html.P(html.Props{Role: "status"}, html.Text(build)),
			html.Button(html.Props{Type: "button", OnClick: chooseSpecies}, html.Text("Human")),
			html.Button(html.Props{Type: "button", OnClick: chooseGender}, html.Text("Nonbinary")),
			html.Button(html.Props{Type: "button", OnClick: roll, Disabled: snapshot.Phase == CreationRolling || snapshot.Phase == CreationLocked}, html.Text("Roll my hero")),
		)
	}
}
