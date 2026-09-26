package splat

import "errors"

const (
	// ModeSplat selects the PlayCanvas battlefield renderer.
	ModeSplat = "SPLAT"
	// ModeFlat selects the portable fallback renderer.
	ModeFlat = "FLAT"
)

// ViewBattlefield is the part of a DM view needed by the splat client.
// The DM adapter should populate Init and Scene from View.Battlefield.
type ViewBattlefield struct {
	Mode  string
	Init  Init
	Scene Scene
}

// Renderer is the bridge surface used by ModeController.
type Renderer interface {
	Init(Init) error
	Scene(Scene) error
	Dispose() error
}

// ModeController keeps the browser renderer aligned with the server view.
// It never changes the server-authoritative mode; browser failures only change
// the local renderer to FLAT until a later view starts a fresh SPLAT load.
type ModeController struct {
	renderer Renderer
	mode     string
	failed   bool
	lastErr  error
}

// NewModeController creates a mode controller for a browser renderer.
func NewModeController(renderer Renderer) *ModeController {
	return &ModeController{renderer: renderer, mode: ModeFlat}
}

// Mode returns the currently selected local renderer mode.
func (c *ModeController) Mode() string {
	if c == nil || c.mode == "" {
		return ModeFlat
	}
	return c.mode
}

// Failed reports whether the current local SPLAT load has failed.
func (c *ModeController) Failed() bool {
	return c != nil && c.failed
}

// LastError returns the most recent browser failure, if any.
func (c *ModeController) LastError() error {
	if c == nil {
		return nil
	}
	return c.lastErr
}

// Apply follows a server battlefield snapshot. FLAT does not load JavaScript;
// SPLAT initializes the bridge and then applies the full scene snapshot.
func (c *ModeController) Apply(view ViewBattlefield) error {
	if c == nil || c.renderer == nil {
		return errors.New("splat mode controller requires a renderer")
	}
	if view.Mode != ModeSplat {
		return c.switchFlat(nil)
	}
	c.mode = ModeSplat
	c.failed = false
	c.lastErr = nil
	if err := c.renderer.Init(view.Init); err != nil {
		return c.switchFlat(err)
	}
	if err := c.renderer.Scene(view.Scene); err != nil {
		return c.switchFlat(err)
	}
	return nil
}

// HandleEvent applies a browser event. Terminal error events switch locally to
// FLAT; readiness and statistics remain informational to the caller.
func (c *ModeController) HandleEvent(event Event) error {
	if c == nil {
		return errors.New("splat mode controller is nil")
	}
	if event.Type != "error" {
		return nil
	}
	if event.Code == "" && event.Detail == "" {
		return c.switchFlat(errors.New("splat renderer failed"))
	}
	message := event.Code
	if event.Detail != "" {
		message = event.Detail
	}
	return c.switchFlat(errors.New(message))
}

func (c *ModeController) switchFlat(cause error) error {
	c.mode = ModeFlat
	c.failed = cause != nil
	c.lastErr = cause
	if c.renderer == nil {
		return cause
	}
	if err := c.renderer.Dispose(); err != nil && cause == nil {
		return err
	}
	return cause
}
