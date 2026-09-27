package wire

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// withPublicURLs puts the DM, host, and phone URLs for publicURL ahead of the
// local tester URLs and returns the public phone URL as the join URL. With no
// public URL it returns urls unchanged and the preferred LAN join URL.
func withPublicURLs(publicURL string, urls []string, port int, room, dmToken, hostToken string) ([]string, string) {
	if publicURL == "" {
		return urls, preferredLANJoinURL(urls, port, room)
	}
	base := strings.TrimSuffix(publicURL, "/")
	public := []string{
		base + "/dm?token=" + url.QueryEscape(dmToken),
		base + "/host?t=" + url.QueryEscape(hostToken),
		base + "/p?room=" + url.QueryEscape(room),
	}
	return append(public, urls...), public[2]
}

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
