package api

import (
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/i18n"
)

// LocalizeScreen renders a projected screen in a seat locale. Move labels
// come from the catalog by move ID; reasons and status lines are matched
// back to catalog keys from their engine English, with the engine string as
// fallback. The default locale keeps engine strings byte-identical.
func LocalizeScreen(state *df.ScreenState, locale string) *df.ScreenState {
	if state == nil {
		return nil
	}
	tag := i18n.Settle(locale, i18n.DefaultLocale)
	catalog := i18n.Default()
	state.Locale = tag
	switch view := state.View.(type) {
	case *df.ScreenState_Phone:
		localizePhone(view.Phone, catalog, tag)
	case *df.ScreenState_Dm:
		localizeDM(view.Dm, catalog, tag)
	case *df.ScreenState_Host:
		if view.Host != nil {
			view.Host.Locale = tag
			localizeDM(view.Host.Dm, catalog, tag)
		}
	}
	return state
}

// LocalizeDomainView stamps a domain view with its locale and fills the
// key-plus-arguments notice and seat status messages.
func LocalizeDomainView(view domain.View, locale string) domain.View {
	tag := i18n.Settle(locale, i18n.DefaultLocale)
	view = view.DeepCopy()
	view.Locale = tag
	for i, seat := range view.Seats {
		if key, args, ok := i18n.MatchEnglish(seat.StatusText); ok {
			seat.StatusMsg = domain.LocalizedMessage{Key: key, Args: args, Fallback: seat.StatusText}
		}
		view.Seats[i] = seat
	}
	return view
}

// ProjectLocalized projects a domain view and localizes it for one locale.
func ProjectLocalized(view domain.View, kind df.ClientKind, seat domain.SeatID, locale string) *df.ScreenState {
	return LocalizeScreen(Project(LocalizeDomainView(view, locale), kind, seat), locale)
}

func localizePhone(phone *df.PhoneView, catalog i18n.Catalog, locale string) {
	if phone == nil {
		return
	}
	phone.Locale = locale
	for _, move := range phone.Moves {
		if move == nil {
			continue
		}
		key := i18n.MoveKey(move.GetMoveId())
		if catalog.Has(locale, key) || catalog.Has(i18n.DefaultLocale, key) {
			move.LabelKey = key
			move.Label = catalog.T(locale, key, nil, 0, move.GetLabel())
		}
		if reason := move.GetReason(); reason != "" {
			if matched, args, ok := i18n.MatchEnglish(reason); ok {
				move.ReasonMsg = &df.Text{Key: matched, Args: args, Fallback: reason}
				move.Reason = catalog.T(locale, matched, args, 0, reason)
			}
		}
	}
	if status := phone.GetStatusText(); status != "" {
		if matched, args, ok := i18n.MatchEnglish(status); ok {
			phone.StatusMsg = &df.Text{Key: matched, Args: args, Fallback: status}
			phone.StatusText = catalog.T(locale, matched, args, 0, status)
		}
	}
}

func localizeDM(dm *df.DMView, catalog i18n.Catalog, locale string) {
	if dm == nil {
		return
	}
	dm.Locale = locale
}
