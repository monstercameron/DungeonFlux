package dm

import "testing"

func TestTitleArtFor_UsesWideBackgroundForUltrawide(t *testing.T) {
	var names []string
	art := titleArtFor(aspectUltrawide, func(name string) string {
		names = append(names, name)
		return "blob:" + name
	})
	if art.Background != "blob:ui/title_bg_wide" {
		t.Fatalf("background = %q", art.Background)
	}
	if art.Wordmark != "blob:ui/logo_wordmark" || art.QRFrame != "blob:ui/qr_frame" {
		t.Fatalf("resolved art = %+v", art)
	}
	if len(names) != 5 {
		t.Fatalf("resolved %d assets, want 5", len(names))
	}
}

func TestTitleArtFor_UsesStandardBackgroundForOtherAspects(t *testing.T) {
	art := titleArtFor(aspectWide, func(name string) string { return name })
	if art.Background != titleBackgroundAsset {
		t.Fatalf("background = %q, want %q", art.Background, titleBackgroundAsset)
	}
}

func TestTitleArtFor_NilResolverReturnsFallbackArt(t *testing.T) {
	if got := titleArtFor(aspectWide, nil); got != (titleArt{}) {
		t.Fatalf("nil resolver = %+v, want empty art", got)
	}
}
