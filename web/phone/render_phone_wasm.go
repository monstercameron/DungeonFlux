//go:build js && wasm

package phone

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// A content component has explicit snapshot props and a stable identity. Watch
// updates repaint it without discarding input, scroll, or in-flight callbacks.
type phoneContentProps struct {
	models     phoneViewProps
	kind       ScreenKind
	tab        PhoneTabID
	view       SeatView
	locale     string
	art        uint64
	selectPlay ui.Handler
}

func renderPhoneScreen(kind ScreenKind, props phoneViewProps, locale string, view SeatView, activeTab PhoneTabID, taps map[PhoneTabID]ui.Handler, selectPlay ui.Handler) ui.Node {
	frame := NewFrameModel("Player", locale)
	frame.Screen, frame.Mode = kind, modeForScreen(kind)
	frame.Connection, frame.ActiveTab = ConnectionOnline, activeTab
	status := ComputeTurnStatus(view)
	frame.TurnLabel = TurnBannerText(locale, status)
	frame.TurnYours = status.Known && status.Yours
	content := ui.CreateElement(phoneContent, phoneContentProps{
		models: props, kind: kind, tab: activeTab, view: view,
		locale: locale, art: artRevision.Load(), selectPlay: selectPlay,
	})
	content = html.WithKey(content, string(kind)+":"+string(activeTab))
	if activeTab == PhoneTabPlay && kind != ScreenEnd && kind != ScreenCreate {
		content = html.Div(html.Props{}, content, narrationBubble(view.Narration))
	}
	// The frame is pure markup. A prop-less closure here would hide tab and
	// snapshot changes from reconciliation even though its captured nodes changed.
	return PhoneFrame(frame, content, audioControls(props.audio, locale), phoneTabBar(frame, taps))(router.Attrs{})
}

func phoneContent(p phoneContentProps) ui.Node {
	if p.tab != PhoneTabPlay {
		return phoneOverlay(p)
	}
	attrs := router.Attrs{}
	switch p.kind {
	case ScreenCreate:
		return CreationScreen(p.models.creation)(attrs)
	case ScreenDice:
		return DiceScreen(p.models.dice)(attrs)
	case ScreenCombat:
		return combatScreen(p.models.combat, p.locale)
	case ScreenEnd:
		return EndScreen(p.models.end)(attrs)
	case ScreenConversation:
		return TalkScreen(p.models, p.locale)(attrs)
	case ScreenMoves:
		return MovesScreen(p.models.moves)(attrs)
	case ScreenWaiting:
		return WaitingScreen(NewWaitingModel(p.view), p.locale, p.models.moves)(attrs)
	default:
		return SheetScreen(p.models.sheet)(attrs)
	}
}

func phoneOverlay(p phoneContentProps) ui.Node {
	switch p.tab {
	case PhoneTabJournal:
		return journalScreen(p.models.journal, p.locale)
	case PhoneTabMap:
		return mapScreen(p.view, p.locale, p.selectPlay)
	case PhoneTabMenu:
		return menuScreen(p.models.audio, p.locale)
	default:
		return SheetScreen(p.models.sheet)(router.Attrs{})
	}
}
