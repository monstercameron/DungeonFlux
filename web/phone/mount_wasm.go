//go:build js && wasm

package phone

import (
	"context"
	"errors"
	"syscall/js"

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
	TalkOpener
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
			combat: NewCombatModel(client, seatToken), ptt: NewPTTModel(client, seatToken, 0),
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
	ptt       *PTTModel
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
	}, props.seatToken)
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
		ui.CreateElement(pttScreen, pttProps{model: props.ptt}),
	)
}

type pttProps struct{ model *PTTModel }

func pttScreen(props pttProps) ui.Node {
	status := ui.UseState("Ready to talk")
	recorder := ui.UseState((*BrowserRecorder)(nil))
	stream := ui.UseState(js.Value{})
	cancelRecording := ui.UseState((context.CancelFunc)(nil))
	ui.UseEffect(func() func() {
		return func() {
			if cancel := cancelRecording.Get(); cancel != nil {
				cancel()
			}
			if current := recorder.Get(); current != nil {
				current.Dispose()
			}
			stopTracks(stream.Get())
		}
	}, props.model)
	start := ui.UseEvent(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancelRecording.Set(cancel)
		go startPTT(ctx, props.model, status.Set, recorder.Set, stream.Set)
	})
	stop := ui.UseEvent(func() {
		current := recorder.Get()
		if current == nil {
			return
		}
		status.Set("Finishing recording…")
		go func() {
			if err := current.Stop(); err != nil {
				status.Set(err.Error())
				return
			}
			<-current.Done()
			current.Dispose()
			stopTracks(stream.Get())
			if err := current.Err(); err != nil {
				_ = <-props.model.Stop(context.Background())
				status.Set(err.Error())
				return
			}
			if err := <-props.model.Stop(context.Background()); err != nil {
				status.Set(err.Error())
				return
			}
			status.Set("Ready to talk")
		}()
	})
	return html.Section(html.Props{Class: "df-phone-ptt"},
		html.Button(html.Props{Type: "button", OnClick: start}, html.Text("Start talking")),
		html.Button(html.Props{Type: "button", OnClick: stop}, html.Text("Stop talking")),
		html.P(html.Props{Role: "status"}, html.Text(status.Get())),
	)
}

func startPTT(ctx context.Context, model *PTTModel, setStatus func(string), setRecorder func(*BrowserRecorder), setStream func(js.Value)) {
	if model == nil {
		setStatus("Push-to-talk unavailable")
		return
	}
	promise := js.Global().Get("navigator").Get("mediaDevices").Call("getUserMedia", map[string]interface{}{"audio": true})
	resolved := make(chan js.Value, 1)
	rejected := make(chan error, 1)
	then := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			resolved <- args[0]
		}
		return nil
	})
	catch := js.FuncOf(func(js.Value, []js.Value) interface{} {
		rejected <- errors.New("microphone permission was denied")
		return nil
	})
	promise.Call("then", then).Call("catch", catch)
	select {
	case err := <-rejected:
		then.Release()
		catch.Release()
		setStatus(err.Error())
	case mediaStream := <-resolved:
		then.Release()
		catch.Release()
		setStream(mediaStream)
		mimeType := recorderMIME()
		if mimeType == "" {
			stopTracks(mediaStream)
			setStatus("This browser cannot record audio")
			return
		}
		if err := model.Start(ctx, mimeType); err != nil {
			stopTracks(mediaStream)
			setStatus(err.Error())
			return
		}
		recorder, err := NewBrowserRecorder(mediaStream, mimeType, model.QueueChunk)
		if err != nil {
			setStatus(err.Error())
			return
		}
		if err := recorder.Start(); err != nil {
			setStatus(err.Error())
			return
		}
		setRecorder(recorder)
		setStatus("Recording…")
	}
}

func recorderMIME() string {
	mediaRecorder := js.Global().Get("MediaRecorder")
	if !mediaRecorder.Truthy() {
		return ""
	}
	for _, mimeType := range []string{"audio/webm;codecs=opus", "audio/webm", "audio/mp4;codecs=mp4a.40.2", "audio/mp4"} {
		if mediaRecorder.Call("isTypeSupported", mimeType).Truthy() {
			return mimeType
		}
	}
	return ""
}

func stopTracks(stream js.Value) {
	if !stream.Truthy() {
		return
	}
	tracks := stream.Call("getTracks")
	for index := 0; index < tracks.Length(); index++ {
		tracks.Index(index).Call("stop")
	}
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
