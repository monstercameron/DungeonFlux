package wire

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildURLsForAddrs_includesLocalAndIPv4Hosts(t *testing.T) {
	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("192.168.1.20")},
		&net.IPNet{IP: net.ParseIP("127.0.0.1")},
		&net.IPNet{IP: net.ParseIP("2001:db8::1")},
		&net.IPNet{IP: net.ParseIP("169.254.1.20")},
		&net.IPNet{IP: net.ParseIP("10.0.0.4")},
		&net.IPNet{IP: net.ParseIP("192.168.1.20")},
	}
	got := buildURLsForAddrs(8443, "ROOM & 1", "dm/token", "host token", addrs)
	want := []string{
		"http://localhost:8443/dm?token=dm%2Ftoken",
		"http://localhost:8443/host?t=host+token",
		"http://localhost:8443/p?room=ROOM+%26+1",
		"http://10.0.0.4:8443/dm?token=dm%2Ftoken",
		"http://10.0.0.4:8443/host?t=host+token",
		"http://10.0.0.4:8443/p?room=ROOM+%26+1",
		"http://192.168.1.20:8443/dm?token=dm%2Ftoken",
		"http://192.168.1.20:8443/host?t=host+token",
		"http://192.168.1.20:8443/p?room=ROOM+%26+1",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("URLs = %#v, want %#v", got, want)
	}
}

func TestWriteURLs_writesOneURLPerLine(t *testing.T) {
	dir := t.TempDir()
	urls := []string{"http://localhost:1/dm?token=dm", "http://localhost:1/host?t=host"}
	if err := writeURLs(dir, urls); err != nil {
		t.Fatalf("writeURLs() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "urls.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got, want := string(data), strings.Join(urls, "\n")+"\n"; got != want {
		t.Fatalf("urls.txt = %q, want %q", got, want)
	}
}
