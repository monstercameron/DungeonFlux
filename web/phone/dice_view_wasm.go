//go:build js && wasm

package phone

import "github.com/monstercameron/GoWebComponents/v6/router"

// DiceScreen renders the touch-first persuasion roll card.
func DiceScreen(model *DiceModel) router.Component {
	return CheckScreen(model)
}
