package dm

const (
	titleBackgroundAsset     = "ui/title_bg"
	titleBackgroundWideAsset = "ui/title_bg_wide"
	titleWordmarkAsset       = "ui/logo_wordmark"
	titleQRFrameAsset        = "ui/qr_frame"
	titlePanelFrameAsset     = "ui/panel_frame"
	titleDividerAsset        = "ui/divider"
)

// titleArt contains the logical UI art URLs used by the title and lobby.
// Empty URLs are intentional: the browser renderer supplies a CSS fallback
// until the shell's gRPC asset loader has installed a source.
type titleArt struct {
	Background string
	Wordmark   string
	QRFrame    string
	PanelFrame string
	Divider    string
}

// titleArtFor resolves the title art for an aspect class. Keeping this
// projection in plain Go makes asset selection testable without a browser.
func titleArtFor(aspect string, resolve func(string) string) titleArt {
	if resolve == nil {
		return titleArt{}
	}
	background := titleBackgroundAsset
	if aspect == aspectUltrawide {
		background = titleBackgroundWideAsset
	}
	return titleArt{
		Background: resolve(background),
		Wordmark:   resolve(titleWordmarkAsset),
		QRFrame:    resolve(titleQRFrameAsset),
		PanelFrame: resolve(titlePanelFrameAsset),
		Divider:    resolve(titleDividerAsset),
	}
}
