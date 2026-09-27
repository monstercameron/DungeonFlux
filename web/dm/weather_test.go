package dm

import (
	"math/rand"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBoltDataURI_IsAnEscapedSVGForEveryVariant(t *testing.T) {
	for i := -1; i <= len(boltPaths); i++ {
		uri := boltDataURI(i)
		if !strings.HasPrefix(uri, "data:image/svg+xml,") {
			t.Fatalf("variant %d: prefix %q", i, uri[:24])
		}
		if strings.ContainsAny(strings.TrimPrefix(uri, "data:image/svg+xml,"), "#\" <>") {
			t.Fatalf("variant %d: unescaped characters would break the CSS url()", i)
		}
		svg, err := url.PathUnescape(strings.TrimPrefix(uri, "data:image/svg+xml,"))
		if err != nil {
			t.Fatalf("variant %d: %v", i, err)
		}
		want := len(boltPaths[((i%len(boltPaths))+len(boltPaths))%len(boltPaths)]) * 3
		if got := strings.Count(svg, "<path "); got != want {
			t.Fatalf("variant %d: %d paths, want %d (three glow layers per branch)", i, got, want)
		}
	}
}

func TestNextStrike_StaysInsideTheSafeEnvelope(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	seen := map[int]bool{}
	doubles := 0
	for range 2000 {
		s := nextStrike(r)
		if s.Delay < 3*time.Second || s.Delay >= 9*time.Second {
			t.Fatalf("delay %v outside 3s..9s", s.Delay)
		}
		if s.Burst < 1 || s.Burst > 3 {
			t.Fatalf("burst %d outside 1..3", s.Burst)
		}
		if g := burstGap(r); g < 600*time.Millisecond || g >= 1500*time.Millisecond {
			t.Fatalf("burst gap %v would allow more than three flashes a second", g)
		}
		if s.LeftPct < 4 || s.LeftPct > 48 || s.TopPct < -6 || s.TopPct > 2 {
			t.Fatalf("placement %.1f%%, %.1f%% outside the open sky", s.LeftPct, s.TopPct)
		}
		if s.Scale < 0.75 || s.Scale > 1.2 {
			t.Fatalf("scale %.2f", s.Scale)
		}
		seen[s.Variant] = true
		if s.Double {
			doubles++
		}
	}
	if len(seen) != len(boltPaths) {
		t.Fatalf("only %d of %d bolt variants ever chosen", len(seen), len(boltPaths))
	}
	if doubles < 600 || doubles > 1000 {
		t.Fatalf("%d double flickers in 2000 strikes, want about 40%%", doubles)
	}
}
