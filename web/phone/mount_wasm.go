//go:build js && wasm

package phone

import (
	"context"
	"errors"
	"strconv"
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
	return func(_ router.Attrs) *router.Element {
		if client == nil {
			return ui.CreateElement(func() ui.Node { return phoneError(locale, ClientUnavailable(locale)) })
		}
		props := phoneViewProps{
			client: client, seatToken: seatToken,
			creation: NewCreationModel(client, seatToken, 0),
			sheet:    NewSheetModel(), moves: NewMovesModel(client, seatToken),
			typed: NewTypedInputModel(client, seatToken), dice: NewDiceModel(client, seatToken),
			combat: NewCombatModel(client, seatToken), ptt: NewPTTModel(client, seatToken, 0), end: NewEndModel(), audio: newPhoneAudio(client, seatToken),
			journal: NewJournalLog(),
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
	snapshotVersion = state.Version
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

func renderPhoneScreen(kind ScreenKind, props phoneViewProps, locale string, view SeatView, activeTab PhoneTabID, taps map[PhoneTabID]ui.Handler, selectPlay ui.Handler) ui.Node {
	frame := NewFrameModel("Player", locale)
	frame.Screen = kind
	frame.Mode = modeForScreen(kind)
	frame.Connection = ConnectionOnline
	frame.ActiveTab = activeTab
	status := ComputeTurnStatus(view)
	frame.TurnLabel = TurnBannerText(locale, status)
	frame.TurnYours = status.Known && status.Yours
	tabBar := phoneTabBar(frame, taps)
	if activeTab != PhoneTabPlay {
		content := overlayScreen(activeTab, props, view, locale, selectPlay)
		return frameScreen(frame, content, props.audio, locale, tabBar)
	}
	switch kind {
	case ScreenCreate:
		return frameScreen(frame, ui.CreateElement(CreationScreen(props.creation)), props.audio, locale, tabBar)
	case ScreenDice:
		return frameScreen(frame, ui.CreateElement(DiceScreen(props.dice)), props.audio, locale, tabBar)
	case ScreenCombat:
		return frameScreen(frame, ui.CreateElement(func() ui.Node { return combatScreen(props.combat, locale) }), props.audio, locale, tabBar)
	case ScreenEnd:
		return frameScreen(frame, ui.CreateElement(EndScreen(props.end)), props.audio, locale, tabBar)
	case ScreenConversation:
		return frameScreen(frame, ui.CreateElement(func() ui.Node { return conversationScreen(props, locale) }), props.audio, locale, tabBar)
	case ScreenMoves:
		return frameScreen(frame, ui.CreateElement(MovesScreen(props.moves)), props.audio, locale, tabBar)
	case ScreenWaiting:
		return frameScreen(frame, ui.CreateElement(WaitingScreen(NewWaitingModel(view), locale, props.moves)), props.audio, locale, tabBar)
	default:
		return frameScreen(frame, ui.CreateElement(SheetScreen(props.sheet)), props.audio, locale, tabBar)
	}
}

// overlayScreen renders the Character, Journal, Map, or Menu tab in place of
// the current phase's Play content. The Character tab reuses the same
// server-authoritative sheet (with the real inventory) shown by the Play
// tab in non-combat phases; only the tab bar's Play state actually differs.
func overlayScreen(tab PhoneTabID, props phoneViewProps, view SeatView, locale string, selectPlay ui.Handler) ui.Node {
	switch tab {
	case PhoneTabCharacter:
		return ui.CreateElement(SheetScreen(props.sheet))
	case PhoneTabJournal:
		return ui.CreateElement(func() ui.Node { return journalScreen(props.journal, locale) })
	case PhoneTabMap:
		return ui.CreateElement(func() ui.Node { return mapScreen(view, locale, selectPlay) })
	case PhoneTabMenu:
		return ui.CreateElement(func() ui.Node { return menuScreen(props.audio, locale) })
	default:
		return ui.CreateElement(SheetScreen(props.sheet))
	}
}

// snapshotVersion is the server snapshot being rendered. Screens are prop-less
// closure components that the reconciler never re-renders from the parent, so
// screens without local input state are keyed by it and remount per snapshot
// (the creation screen otherwise kept its pickers after the hero was rolled).
// Conversation keeps its key so typed text and push-to-talk state survive.
var snapshotVersion uint64

// phoneEnter remembers the screen kind whose entry animation last started and
// the frame key it started on. The frame remounts on every snapshot, so the
// entry class rides only that first key: later snapshots of the same screen
// remount without it and never replay the animation.
var phoneEnter struct {
	kind ScreenKind
	key  string
}

func frameScreen(model FrameModel, content ui.Node, audio *PhoneAudio, locale string, tabBar ui.Node) ui.Node {
	key := string(model.Screen) + ":" + strconv.FormatUint(artRevision.Load(), 10)
	if model.Screen != ScreenConversation {
		key += ":" + strconv.FormatUint(snapshotVersion, 10)
	}
	if model.Screen != phoneEnter.kind {
		phoneEnter.kind, phoneEnter.key = model.Screen, key
	}
	model.Enter = key == phoneEnter.key
	return html.WithKey(ui.CreateElement(PhoneFrame(model, content, audioControls(audio, locale), tabBar)), key)
}

func conversationScreen(props phoneViewProps, locale string) ui.Node {
	return ui.CreateElement(TalkScreen(props, locale))
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
