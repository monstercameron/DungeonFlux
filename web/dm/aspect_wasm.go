//go:build js && wasm

package dm

import "syscall/js"

func currentAspectClass() string {
	return AspectClass(browserQuery("aspect"), viewportWidth(), viewportHeight())
}

func viewportWidth() float64 {
	return browserDimension("innerWidth")
}

func viewportHeight() float64 {
	return browserDimension("innerHeight")
}

func browserDimension(name string) float64 {
	value := js.Global().Get(name)
	if !value.Truthy() {
		return 0
	}
	return value.Float()
}
