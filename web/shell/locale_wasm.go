//go:build js && wasm

package main

import "syscall/js"

// browserLocales reads navigator.languages in priority order.
func browserLocales() []string {
	navigator := js.Global().Get("navigator")
	languages := navigator.Get("languages")
	if !languages.Truthy() || languages.Get("length").Int() == 0 {
		if single := navigator.Get("language"); single.Truthy() {
			return []string{single.String()}
		}
		return nil
	}
	tags := make([]string, 0, languages.Get("length").Int())
	for index := 0; index < languages.Get("length").Int(); index++ {
		tags = append(tags, languages.Index(index).String())
	}
	return tags
}
