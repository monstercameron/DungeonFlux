package dm

import "sync/atomic"

// ArtSource resolves a logical UI art name, as registered in the build-time
// manifest (for example "ui/title_bg" or "tavern_interior"), to a URL the
// browser can load. The shell installs the gRPC-backed asset loader (WEB-015);
// until then every name resolves to "" and screens must degrade gracefully.
type ArtSource interface {
	ArtURL(name string) string
}

// artSourceHolder keeps the mount-time art source. It is written once by the
// shell at boot and read by every render, so it uses an atomic value.
type artSourceHolder struct{ value atomic.Value }

var artSources artSourceHolder

// SetArtSource installs the art resolver used by every DM layer. The shell
// calls it once at boot; a nil source restores the empty resolver.
func SetArtSource(source ArtSource) {
	if source == nil {
		source = noArt{}
	}
	artSources.value.Store(artBox{source: source})
}

// ArtURL resolves a logical art name through the installed source, or returns
// "" when no source is installed or the name is unknown.
func ArtURL(name string) string {
	box, ok := artSources.value.Load().(artBox)
	if !ok || box.source == nil {
		return ""
	}
	return box.source.ArtURL(name)
}

// artBox gives atomic.Value one concrete type for every stored source.
type artBox struct{ source ArtSource }

type noArt struct{}

func (noArt) ArtURL(string) string { return "" }
