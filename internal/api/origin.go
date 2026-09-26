package api

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
)

// originCheck allows same-origin browser upgrades and configured extra origins.
func originCheck(allowedOrigins []string) func(*http.Request) bool {
	configured := grpctunnel.BuildOriginAllowlistCheck(allowedOrigins...)
	return func(r *http.Request) bool {
		if sameRequestOrigin(r) {
			return true
		}
		return configured(r)
	}
}

func sameRequestOrigin(r *http.Request) bool {
	if r == nil {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || r.Host == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}
