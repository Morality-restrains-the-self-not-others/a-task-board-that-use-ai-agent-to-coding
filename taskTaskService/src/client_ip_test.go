package main

import (
	"net/http"
	"testing"
)

func TestResolveClientIPPrefersXFF(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")
	r.Header.Set("X-Real-IP", "198.51.100.7")
	r.RemoteAddr = "127.0.0.1:12345"
	if got := resolveClientIP(r); got != "203.0.113.50" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClientIPSkipsDockerNATFirstHop(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Forwarded-For", "172.26.0.1, 203.0.113.50")
	r.Header.Set("X-Real-IP", "172.26.0.1")
	r.RemoteAddr = "172.26.0.1:12345"
	if got := resolveClientIP(r); got != "203.0.113.50" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClientIPFallsBackRealIP(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Real-IP", "198.51.100.7")
	r.RemoteAddr = "127.0.0.1:12345"
	if got := resolveClientIP(r); got != "198.51.100.7" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAutoRunClientPublicIPBodyOverride(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	body := map[string]interface{}{"client_public_ip": "198.51.100.9"}
	if got := resolveAutoRunClientPublicIP(r, body); got != "198.51.100.9" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAutoRunClientPublicIPFromHeaders(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.77")
	if got := resolveAutoRunClientPublicIP(r, map[string]interface{}{}); got != "203.0.113.77" {
		t.Fatalf("got %q", got)
	}
}
