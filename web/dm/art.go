package dm

import (
	"hash/fnv"
	"strings"
	"sync/atomic"
)

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

// artSrc resolves a wire art selector without allowing an unresolved view URL
// to reach the DOM. Blob and data URLs are already browser-local and are
// passed through; logical names, SHA-256 selectors, and /assets paths go
// through the gRPC-backed ArtURL source and remain empty while loading.
func artSrc(selector string) string {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return ""
	}
	lower := strings.ToLower(selector)
	if strings.HasPrefix(lower, "blob:") || strings.HasPrefix(lower, "data:") {
		return selector
	}
	return ArtURL(selector)
}

// artBox gives atomic.Value one concrete type for every stored source.
type artBox struct{ source ArtSource }

type noArt struct{}

func (noArt) ArtURL(string) string { return "" }

var proxySpecies = []string{"human", "elf", "dwarf", "halfling", "orc", "tiefling", "dragonborn", "gnome", "goliath"}

// heroProxyArt is the stand-in portrait while a hero's generated image is not
// ready: the rolled species art, else the class crest, else a seeded random
// species (the same seed always rolls the same proxy). "" only while the
// proxy art itself is still loading.
func heroProxyArt(species, class, seed string) string {
	if species = strings.ToLower(strings.TrimSpace(species)); species != "" {
		if url := ArtURL("ui/species_" + species); url != "" {
			return url
		}
	}
	if class = strings.ToLower(strings.TrimSpace(class)); class != "" {
		if url := ArtURL("ui/class_" + class); url != "" {
			return url
		}
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	return ArtURL("ui/species_" + proxySpecies[hash.Sum32()%uint32(len(proxySpecies))])
}

// heroProxyArtGendered is heroProxyArt, but tries the gendered species
// stand-in first (OPS-028: ui/species_<species>_<gender>, registered for the
// nine species x three genders creation offers). Callers that already know a
// seat's chosen gender (for example the TV creation build card) should use
// this instead of heroProxyArt so two heroes of different genders no longer
// share one stand-in portrait.
func heroProxyArtGendered(species, gender, class, seed string) string {
	species = strings.ToLower(strings.TrimSpace(species))
	gender = strings.ToLower(strings.TrimSpace(gender))
	if species != "" && gender != "" {
		if url := ArtURL("ui/species_" + species + "_" + gender); url != "" {
			return url
		}
		return ArtURL("ui/class_" + strings.ToLower(strings.TrimSpace(class)))
	}
	return heroProxyArt(species, class, seed)
}
