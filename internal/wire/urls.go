package wire

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// buildURLs returns DM, host, and phone URLs for localhost and each
// non-loopback IPv4 address on the machine.
func buildURLs(port int, room, dmToken, hostToken string) ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("wire: find interface addresses: %w", err)
	}
	return buildURLsForAddrs(port, room, dmToken, hostToken, addrs), nil
}

func buildURLsForAddrs(port int, room, dmToken, hostToken string, addrs []net.Addr) []string {
	hosts := []string{"localhost"}
	seen := map[string]bool{"localhost": true}
	for _, addr := range addrs {
		ip := addressIPv4(addr)
		if ip == "" || seen[ip] {
			continue
		}
		seen[ip] = true
		hosts = append(hosts, ip)
	}
	sort.Strings(hosts[1:])
	urls := make([]string, 0, len(hosts)*3)
	for _, host := range hosts {
		base := fmt.Sprintf("http://%s:%d", host, port)
		urls = append(urls,
			base+"/dm?token="+url.QueryEscape(dmToken),
			base+"/host?t="+url.QueryEscape(hostToken),
			base+"/p?room="+url.QueryEscape(room),
		)
	}
	return urls
}

func addressIPv4(addr net.Addr) string {
	var value string
	switch typed := addr.(type) {
	case *net.IPNet:
		value = typed.IP.String()
	case *net.IPAddr:
		value = typed.IP.String()
	default:
		return ""
	}
	ip := net.ParseIP(value)
	if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.To4() == nil {
		return ""
	}
	return ip.To4().String()
}

func writeURLs(dataDir string, urls []string) error {
	path := filepath.Join(dataDir, "urls.txt")
	if err := os.WriteFile(path, []byte(strings.Join(urls, "\n")+"\n"), 0o600); err != nil {
		return fmt.Errorf("wire: write tester URLs: %w", err)
	}
	return nil
}

func printURLs(out io.Writer, urls []string) {
	if out == nil {
		return
	}
	_, _ = io.WriteString(out, "DungeonFlux tester URLs:\n")
	for _, line := range urls {
		_, _ = io.WriteString(out, line+"\n")
	}
}
