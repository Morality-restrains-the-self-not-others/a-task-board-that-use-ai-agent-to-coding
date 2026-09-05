package clientip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newReq() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/", nil)
}

func TestResolve_XForwardedFor(t *testing.T) {
	r := newReq()
	r.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
	r.RemoteAddr = "127.0.0.1:12345"
	if got := Resolve(r); got != "203.0.113.10" {
		t.Fatalf("got %q", got)
	}
}

func TestResolve_SkipsDockerNATFirstHop(t *testing.T) {
	r := newReq()
	r.Header.Set("X-Forwarded-For", "172.26.0.1, 203.0.113.10")
	r.Header.Set("X-Real-IP", "172.26.0.1")
	r.RemoteAddr = "172.26.0.1:12345"
	if got := Resolve(r); got != "203.0.113.10" {
		t.Fatalf("got %q want public hop, not docker bridge", got)
	}
}

func TestResolve_RightmostPublicWhenSpoofedLeft(t *testing.T) {
	r := newReq()
	r.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.10, 172.25.0.1")
	r.RemoteAddr = "172.25.0.1:9080"
	if got := Resolve(r); got != "203.0.113.10" {
		t.Fatalf("got %q", got)
	}
}

func TestResolve_OnlyPrivateFallsBackToFirstHop(t *testing.T) {
	r := newReq()
	r.Header.Set("X-Forwarded-For", "172.26.0.1")
	r.RemoteAddr = "172.26.0.1:1"
	if got := Resolve(r); got != "172.26.0.1" {
		t.Fatalf("got %q", got)
	}
}

func TestResolve_XRealIP(t *testing.T) {
	r := newReq()
	r.Header.Set("X-Real-IP", "198.51.100.20")
	r.RemoteAddr = "127.0.0.1:12345"
	if got := Resolve(r); got != "198.51.100.20" {
		t.Fatalf("got %q", got)
	}
}

func TestResolve_RemoteAddr(t *testing.T) {
	r := newReq()
	r.RemoteAddr = "192.0.2.55:54321"
	if got := Resolve(r); got != "192.0.2.55" {
		t.Fatalf("got %q", got)
	}
}

func TestResolve_NilRequest(t *testing.T) {
	if got := Resolve(nil); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestCanonicalIPString(t *testing.T) {
	cases := map[string]string{
		" 203.0.113.5 ":     "203.0.113.5",
		"192.0.2.55:80":     "192.0.2.55",
		"[2001:db8::1]":     "[2001:db8::1]", // bare bracketed v6 without port stays verbatim (legacy behavior)
		"[2001:db8::1]:443": "2001:db8::1",
		"":                  "",
		"   ":               "",
		"not-an-ip":         "not-an-ip",
	}
	for in, want := range cases {
		if got := CanonicalIPString(in); got != want {
			t.Fatalf("CanonicalIPString(%q)=%q want %q", in, got, want)
		}
	}
}

func TestIsPublicIPAddr(t *testing.T) {
	public := []string{"203.0.113.10", "8.8.8.8", "2001:db8::1"}
	for _, s := range public {
		if !IsPublicIPAddr(s) {
			t.Fatalf("IsPublicIPAddr(%q)=false want true", s)
		}
	}
	private := []string{"127.0.0.1", "10.0.0.1", "172.26.0.1", "192.168.1.1", "169.254.1.1", "0.0.0.0", ""}
	for _, s := range private {
		if IsPublicIPAddr(s) {
			t.Fatalf("IsPublicIPAddr(%q)=true want false", s)
		}
	}
}
