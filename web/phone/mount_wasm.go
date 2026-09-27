//go:build js && wasm

package phone

import (
	"context"
	"errors"
	"strings"
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
func Mount(client PhoneClient, seatToken, locale string) router.Component {
	if locale == "" {
		locale = "en"
	}
	props := phoneViewProps{
		client: client, seatToken: seatToken,
		creation: NewCreationModel(client, seatToken, 0),
		sheet:    NewSheetModel(), moves: NewMovesModel(client, seatToken),
		typed: NewTypedInputModel(client, seatToken), dice: NewDiceModel(client, seatToken),
		combat: NewCombatModel(client, seatToken), ptt: NewPTTModel(client, seatToken, 0), end: NewEndModel(), audio: newPhoneAudio(client, seatToken),
		journal: NewJournalLog(),
	}
	return func(_ router.Attrs) *router.Element {
		if client == nil {
			return ui.CreateElement(func() ui.Node { return phoneError(locale, ClientUnavailable(locale)) })
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
	end       *EndModel
	audio     *PhoneAudio
	journal   *JournalLog
}

func phoneError(locale, message string) ui.Node {
	if locale == "" {
		locale = "en"
	}
	return html.Main(html.Props{Class: "df-phone", Role: "main"}, html.H1(html.Props{}, html.Text(ErrorTitle(locale))), html.P(html.Props{Role: "alert"}, html.Text(message)))
}

func phoneView(props phoneViewProps) ui.Node {
	view := ui.UseState(SeatView{})
	activeTab := ui.UseState(PhoneTabPlay)
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
				props.end.ApplyScreenState(result.State)
				if narration := result.State.GetPhone().GetNarration(); narration.GetDone() {
					props.journal.Record(narration.GetLineId(), narration.GetSpeaker(), narration.GetTextSoFar())
				}
				view.Set(seatViewFromState(result.State))
			}
		}()
		return cancel
	}, props.client, props.seatToken)
	state := view.Get()
	locale := phoneLocale(state.Phone)
	props.typed.SetLocale(locale)
	ui.UseEffect(func() func() {
		return func() {
			if props.audio != nil {
				props.audio.CloseAudio()
			}
		}
	}, props.audio)
	kind := SelectScreen(state)
	// A tab switch is a client-only navigation choice, independent of the
	// server phase. It resets to Play whenever the underlying phase screen
	// changes, so the active player is never stuck looking at the Journal
	// while their own turn moves on.
	ui.UseEffect(func() func() { activeTab.Set(PhoneTabPlay); return nil }, kind)
	selectCharacter := ui.UseEvent(func() { activeTab.Set(PhoneTabCharacter) })
	selectJournal := ui.UseEvent(func() { activeTab.Set(PhoneTabJournal) })
	selectPlay := ui.UseEvent(func() { activeTab.Set(PhoneTabPlay) })
	selectMap := ui.UseEvent(func() { activeTab.Set(PhoneTabMap) })
	selectMenu := ui.UseEvent(func() { activeTab.Set(PhoneTabMenu) })
	taps := map[PhoneTabID]ui.Handler{
		PhoneTabCharacter: selectCharacter, PhoneTabJournal: selectJournal, PhoneTabPlay: selectPlay,
		PhoneTabMap: selectMap, PhoneTabMenu: selectMenu,
	}
	screen := renderPhoneScreen(kind, props, locale, state, activeTab.Get(), taps, selectPlay)
	if bubble := narrationBubble(state.Narration); bubble != nil {
		return html.Div(html.Props{Class: "df-phone-read-along-host"}, screen, bubble)
	}
	return screen
}

type pttProps struct {
	model  *PTTModel
	locale string
}

func pttScreen(props pttProps) ui.Node {
	locale := props.locale
	if locale == "" {
		locale = "en"
	}
	status := ui.UseState(PTTReady(locale))
	busy := ui.UseState(false)
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
		if busy.Get() {
			return
		}
		busy.Set(true)
		ctx, cancel := context.WithCancel(context.Background())
		cancelRecording.Set(cancel)
		go startPTT(ctx, cancel, props.model, locale, status.Set, busy.Set, recorder.Set, stream.Set)
	})
	stop := ui.UseEvent(func() {
		current := recorder.Get()
		if current == nil || !busy.Get() {
			return
		}
		status.Set(T(locale, "ptt.finishing", nil))
		go finishPTT(current, props.model, locale, stream.Get(), cancelRecording.Get, status.Set, busy.Set)
	})
	return html.Section(html.Props{Class: "df-phone-ptt"},
		html.Button(html.Props{Type: "button", OnClick: start, Disabled: busy.Get()}, html.Text(PTTStart(locale))),
		html.Button(html.Props{Type: "button", OnClick: stop, Disabled: !busy.Get()}, html.Text(PTTStop(locale))),
		html.P(html.Props{Role: "status"}, html.Text(status.Get())),
	)
}

func finishPTT(recorder *BrowserRecorder, model *PTTModel, locale string, stream js.Value, cancel func() context.CancelFunc, setStatus func(string), setBusy func(bool)) {
	if err := recorder.Stop(); err != nil {
		recorder.Dispose()
		stopTracks(stream)
		if cancelFn := cancel(); cancelFn != nil {
			cancelFn()
		}
		setStatus(err.Error())
		setBusy(false)
		return
	}
	<-recorder.Done()
	recorder.Dispose()
	stopTracks(stream)
	if err := recorder.Err(); err != nil {
		<-model.Stop(context.Background())
		setStatus(err.Error())
		setBusy(false)
		return
	}
	if err := <-model.Stop(context.Background()); err != nil {
		setStatus(err.Error())
		setBusy(false)
		return
	}
	if cancelFn := cancel(); cancelFn != nil {
		cancelFn()
	}
	setBusy(false)
	setStatus(PTTReady(locale))
}

func startPTT(ctx context.Context, cancel context.CancelFunc, model *PTTModel, locale string, setStatus func(string), setBusy func(bool), setRecorder func(*BrowserRecorder), setStream func(js.Value)) {
	if locale == "" {
		locale = "en"
	}
	started := false
	defer func() {
		if !started {
			setBusy(false)
		}
	}()
	if model == nil {
		setStatus(T(locale, "ptt.unavail", nil))
		return
	}
	mediaStream, err := requestMicrophone(ctx)
	if err != nil {
		setStatus(err.Error())
		return
	}
	setStream(mediaStream)
	mimeType := recorderMIME()
	if mimeType == "" {
		stopTracks(mediaStream)
		setStatus(PTTNoRecord(locale))
		return
	}
	if err := model.Start(ctx, mimeType); err != nil {
		stopTracks(mediaStream)
		setStatus(err.Error())
		return
	}
	recorder, err := newBrowserRecorder(mediaStream, mimeType, model.QueueChunk)
	if err != nil {
		stopTracks(mediaStream)
		<-model.Stop(context.Background())
		setStatus(err.Error())
		return
	}
	if err := recorder.Start(); err != nil {
		recorder.Dispose()
		stopTracks(mediaStream)
		<-model.Stop(context.Background())
		setStatus(err.Error())
		return
	}
	setRecorder(recorder)
	setStatus(T(locale, "ui.ptt.recording", nil))
	started = true
	go watchRecorderFailure(ctx, cancel, model, recorder, mediaStream, setStatus, setBusy)
}

func requestMicrophone(ctx context.Context) (js.Value, error) {
	navigator := js.Global().Get("navigator")
	if !navigator.Truthy() || !navigator.Get("mediaDevices").Truthy() {
		return js.Value{}, errors.New("microphone is unavailable")
	}
	promise := navigator.Get("mediaDevices").Call("getUserMedia", map[string]interface{}{"audio": true})
	resolved := make(chan js.Value, 1)
	rejected := make(chan error, 1)
	var then, catch js.Func
	release := func() { then.Release(); catch.Release() }
	then = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		defer release()
		if ctx.Err() != nil && len(args) > 0 {
			stopTracks(args[0])
			return nil
		}
		if len(args) == 0 {
			rejected <- errors.New("microphone permission returned no stream")
			return nil
		}
		resolved <- args[0]
		return nil
	})
	catch = js.FuncOf(func(js.Value, []js.Value) interface{} {
		defer release()
		rejected <- errors.New("microphone permission was denied")
		return nil
	})
	promise.Call("then", then).Call("catch", catch)
	select {
	case stream := <-resolved:
		if err := ctx.Err(); err != nil {
			stopTracks(stream)
			return js.Value{}, err
		}
		return stream, nil
	case err := <-rejected:
		return js.Value{}, err
	case <-ctx.Done():
		return js.Value{}, ctx.Err()
	}
}

func newBrowserRecorder(stream js.Value, mimeType string, queue func([]byte) bool) (recorder *BrowserRecorder, err error) {
	defer func() {
		if recover() != nil {
			recorder = nil
			err = errors.New("media recorder is unavailable")
		}
	}()
	return NewBrowserRecorder(stream, mimeType, queue)
}

func watchRecorderFailure(ctx context.Context, cancel context.CancelFunc, model *PTTModel, recorder *BrowserRecorder, stream js.Value, setStatus func(string), setBusy func(bool)) {
	select {
	case <-recorder.Done():
		if err := recorder.Err(); err != nil {
			recorder.Dispose()
			stopTracks(stream)
			<-model.Stop(context.Background())
			cancel()
			setStatus(err.Error())
			setBusy(false)
		}
	case <-ctx.Done():
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

func combatScreen(model *CombatModel, locale string) ui.Node {
	if locale == "" {
		locale = "en"
	}
	refresh := ui.UseState(0)
	snapshot := model.Snapshot()
	children := []ui.Node{html.H1(html.Props{}, html.Text(CombatTurnTitle(locale))), html.P(html.Props{Role: "status"}, html.Text(snapshot.StatusText))}
	for _, move := range snapshot.Moves {
		children = append(children, ui.CreateElement(combatMoveButton, combatMoveProps{model: model, move: move, refresh: refresh}))
	}
	return combatStyledScreen(model, locale, children...)
}

type combatMoveProps struct {
	model   *CombatModel
	move    *df.Move
	refresh ui.State[int]
}

func combatMoveButton(props combatMoveProps) ui.Node {
	tap := ui.UseEvent(func() {
		go func() {
			props.model.ApplyAct(<-props.model.Tap(context.Background(), props.move))
			props.refresh.Set(props.refresh.Get() + 1)
		}()
	})
	return html.Button(html.Props{Type: "button", OnClick: tap, Disabled: !props.move.GetEnabled()}, html.Text(props.move.GetLabel()))
}

// narrationBubble shows the line being spoken so players can read along. It
// lives outside the keyed screen frame so it updates as the text streams.
func narrationBubble(narration NarrationModel) ui.Node {
	text := strings.TrimSpace(narration.Text)
	if text == "" {
		return nil
	}
	speaker := strings.TrimSpace(narration.Speaker)
	if speaker == "" {
		speaker = "Dungeon Master"
	}
	class := "df-phone-read-along"
	if !narration.Done {
		class += " is-speaking"
	}
	return html.Aside(html.Props{Class: class, Role: "status", Aria: map[string]string{"live": "polite", "label": speaker}},
		html.Strong(html.Props{Class: "df-phone-read-along-speaker"}, ui.Text(speaker)),
		html.P(html.Props{Class: "df-phone-read-along-text"}, ui.Text(text)),
	)
}
