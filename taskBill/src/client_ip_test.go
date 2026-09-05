package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveClientIPFromXFF(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")
	if got := resolveClientIP(r); got != "203.0.113.50" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClientIPSkipsDockerNATFirstHop(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Forwarded-For", "172.26.0.1, 203.0.113.50")
	r.Header.Set("X-Real-IP", "172.26.0.1")
	r.RemoteAddr = "172.26.0.1:12345"
	if got := resolveClientIP(r); got != "203.0.113.50" {
		t.Fatalf("got %q", got)
	}
}
