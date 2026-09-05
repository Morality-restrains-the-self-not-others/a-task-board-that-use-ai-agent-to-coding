package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/client-ip/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
	req.RemoteAddr = "127.0.0.1:12345"
	if got := resolveClientIP(req); got != "203.0.113.10" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClientIP_SkipsDockerNATFirstHop(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/client-ip/", nil)
	req.Header.Set("X-Forwarded-For", "172.26.0.1, 203.0.113.10")
	req.Header.Set("X-Real-IP", "172.26.0.1")
	req.RemoteAddr = "172.26.0.1:12345"
	if got := resolveClientIP(req); got != "203.0.113.10" {
		t.Fatalf("got %q want public hop, not docker bridge", got)
	}
}

func TestResolveClientIP_RightmostPublicWhenSpoofedLeft(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/client-ip/", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.10, 172.25.0.1")
	req.RemoteAddr = "172.25.0.1:9080"
	if got := resolveClientIP(req); got != "203.0.113.10" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClientIP_OnlyPrivateFallsBackToFirstHop(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/client-ip/", nil)
	req.Header.Set("X-Forwarded-For", "172.26.0.1")
	req.RemoteAddr = "172.26.0.1:1"
	if got := resolveClientIP(req); got != "172.26.0.1" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/client-ip/", nil)
	req.Header.Set("X-Real-IP", "198.51.100.20")
	req.RemoteAddr = "127.0.0.1:12345"
	if got := resolveClientIP(req); got != "198.51.100.20" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/client-ip/", nil)
	req.RemoteAddr = "192.0.2.55:54321"
	if got := resolveClientIP(req); got != "192.0.2.55" {
		t.Fatalf("got %q", got)
	}
}

func TestHandleClientIP_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/client-ip/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	rec := httptest.NewRecorder()
	handleClientIP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["ip"] != "203.0.113.99" {
		t.Fatalf("body=%v", body)
	}
}

func TestHandleClientIP_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/client-ip/", nil)
	rec := httptest.NewRecorder()
	handleClientIP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rec.Code)
	}
}
