package api

import (
	"net/http/httptest"
	"testing"
)

func TestSameRequestOrigin(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{name: "localhost", host: "localhost:8443", origin: "http://localhost:8443", want: true},
		{name: "lan ip", host: "192.168.1.27:8443", origin: "http://192.168.1.27:8443", want: true},
		{name: "foreign host", host: "192.168.1.27:8443", origin: "http://evil.example:8443", want: false},
		{name: "foreign port", host: "localhost:8443", origin: "http://localhost:18101", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://"+test.host+"/grpc", nil)
			r.Host = test.host
			r.Header.Set("Origin", test.origin)
			if got := sameRequestOrigin(r); got != test.want {
				t.Fatalf("sameRequestOrigin() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestOriginCheck_PreservesConfiguredOrigins(t *testing.T) {
	check := originCheck([]string{"https://allowed.example"})
	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{name: "configured extra", host: "localhost:8443", origin: "https://allowed.example", want: true},
		{name: "cross origin", host: "localhost:8443", origin: "https://blocked.example", want: false},
		{name: "non browser", host: "localhost:8443", origin: "", want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://"+test.host+"/grpc", nil)
			r.Host = test.host
			if test.origin != "" {
				r.Header.Set("Origin", test.origin)
			}
			if got := check(r); got != test.want {
				t.Fatalf("originCheck() = %v, want %v", got, test.want)
			}
		})
	}
}
