//go:build js && wasm

package phone

import (
	"context"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// WatchResult carries one seat view from the shell watch subscription.
type WatchResult struct {
	State *df.ScreenState
	Err   error
}

// PhoneClient is the small shell client surface needed by the phone screens.
type PhoneClient interface {
	Act(context.Context, *df.ActRequest) <-chan ActResult
	Say(context.Context, *df.SayRequest) <-chan SayResult
	Watch(context.Context, *df.WatchRequest) <-chan WatchResult
}

// Mount returns the stateful player phone screen for the shared shell router.
func Mount(client PhoneClient, seatToken string) router.Component {
	return func(_ router.Attrs) *router.Element {
		if client == nil {
			return ui.CreateElement(phoneError, "Player client unavailable")
		}
		props := phoneViewProps{
			client: client, seatToken: seatToken,
			creation: NewCreationModel(client, seatToken, 0),
			sheet:    NewSheetModel(), moves: NewMovesModel(client, seatToken),
			typed: NewTypedInputModel(client, seatToken), dice: NewDiceModel(client, seatToken),
			combat: NewCombatModel(client, seatToken),
		}
		return ui.CreateElement(phoneView, props)
	}
}

type phoneViewProps struct {
	client    PhoneClient
	seatToken string
	creation  *CreationModel
	sheet     *SheetModel
	moves     *MovesModel
	typed     *TypedInputModel
	dice      *DiceModel
	combat    *CombatModel
}

func phoneError(message string) ui.Node {
	return html.Main(html.Props{Class: "df-phone", Role: "main"}, html.H1(html.Props{}, html.Text("DungeonFlux")), html.P(html.Props{Role: "alert"}, html.Text(message)))
}

func phoneView(props phoneViewProps) ui.Node {
	view := ui.UseState(SeatView{})
	ui.UseEffect(func() func() {
		ctx, cancel := context.WithCancel(context.Background())
		updates := props.client.Watch(ctx, &df.WatchRequest{SeatToken: props.seatToken})
		go func() {
			for result := range updates {
				if result.State == nil {
					continue
				}
				props.creation.ApplyScreenState(result.State)
				props.sheet.ApplyScreenState(result.State)
				props.moves.ApplyScreenState(result.State)
				props.dice.ApplyScreenState(result.State)
				props.combat.ApplyScreenState(result.State)
				view.Set(SeatView{Phase: result.State.GetPhase(), Phone: result.State.GetPhone()})
			}
		}()
		return cancel
	})
	state := view.Get()
	return renderPhoneScreen(SelectScreen(state), props)
}

func renderPhoneScreen(kind ScreenKind, props phoneViewProps) ui.Node {
	switch kind {
	case ScreenCreate:
		return ui.CreateElement(CreationScreen(props.creation))
	case ScreenDice:
		return ui.CreateElement(DiceScreen(props.dice))
	case ScreenCombat:
		return ui.CreateElement(combatScreen, props.combat)
	case ScreenConversation:
		return ui.CreateElement(conversationScreen, props)
	case ScreenMoves:
		return ui.CreateElement(MovesScreen(props.moves))
	default:
		return ui.CreateElement(SheetScreen(props.sheet))
	}
}

func conversationScreen(props phoneViewProps) ui.Node {
	return html.Main(html.Props{Class: "df-phone df-phone-conversation"},
		ui.CreateElement(MovesScreen(props.moves)),
		ui.CreateElement(TypedInputScreen(props.typed)),
	)
}

func combatScreen(model *CombatModel) ui.Node {
	state := ui.UseState(model.Snapshot())
	snapshot := state.Get()
	children := []ui.Node{html.H1(html.Props{}, html.Text("Your turn")), html.P(html.Props{Role: "status"}, html.Text(snapshot.StatusText))}
	for _, move := range snapshot.Moves {
		item := move
		tap := ui.UseEvent(func() { go func() { state.Set(model.ApplyAct(<-model.Tap(context.Background(), item))) }() })
		children = append(children, html.Button(html.Props{Type: "button", OnClick: tap, Disabled: !item.GetEnabled()}, html.Text(item.GetLabel())))
	}
	return html.Main(html.Props{Class: "df-phone df-phone-combat"}, children...)
}
