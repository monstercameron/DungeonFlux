package main

import "github.com/monstercameron/DungeonFlux/internal/i18n"

// localeCatalog is the committed client catalog for shell screens.
var localeCatalog = i18n.Default()

// LocaleModel owns client locale detection and the language switcher. The
// settled tag travels in JoinRequest so the server projects views in it.
type LocaleModel struct {
	switcher i18n.Switcher
}

// NewLocaleModel settles the client locale from browser language tags in
// priority order, falling back to English.
func NewLocaleModel(tags []string) *LocaleModel {
	return &LocaleModel{switcher: i18n.NewSwitcher(i18n.DetectLocale(tags))}
}

// Active returns the settled locale tag.
func (m *LocaleModel) Active() string {
	if m == nil {
		return i18n.DefaultLocale
	}
	if m.switcher.Active == "" {
		return i18n.DefaultLocale
	}
	return m.switcher.Active
}

// Set switches the active locale, falling back for unsupported tags.
func (m *LocaleModel) Set(locale string) string {
	if m == nil {
		return i18n.DefaultLocale
	}
	return m.switcher.Set(locale)
}

// Options returns the ordered locale options for the switcher control.
func (m *LocaleModel) Options() []string {
	if m == nil {
		return append([]string(nil), i18n.SupportedLocales...)
	}
	return append([]string(nil), m.switcher.Options...)
}

// T renders a catalog key for the active locale.
func (m *LocaleModel) T(key string, args map[string]string) string {
	return localeCatalog.T(m.Active(), key, args, 0, "")
}

// BrowserLocales reports the browser language tags in priority order.
// The platform files provide the implementation.
func BrowserLocales() []string { return browserLocales() }
