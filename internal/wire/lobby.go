package wire

import (
	"fmt"
	"net"
	"net/url"
)

// preferredLANJoinURL returns the first phone URL whose host is a usable
// non-loopback, non-link-local IPv4 address. The URL list is ordered by
// buildURLs, so this preserves the advertised preference deterministically.
func preferredLANJoinURL(urls []string, port int, room string) string {
	for _, raw := range urls {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Path != "/p" {
			continue
		}
		host, _, err := net.SplitHostPort(parsed.Host)
		if err != nil {
			continue
		}
		ip := net.ParseIP(host)
		if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		return raw
	}
	return fmt.Sprintf("http://localhost:%d/p?room=%s", port, url.QueryEscape(room))
}
