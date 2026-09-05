package main

import (
	"net/http"
	"testing"
	"time"
)

func transportProxyIsNil(t *testing.T, client *http.Client) {
	t.Helper()

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatalf("expected transport proxy to be nil, got %p", transport.Proxy)
	}
}

func TestNewBackendHTTPClientDisablesProxy(t *testing.T) {
	t.Setenv("RELAY_TO_TRAE_BACKEND_PROXY_MODE", "")
	client := newBackendHTTPClient(0)
	transportProxyIsNil(t, client)
}

func TestNewBackendHTTPClientUsesSystemProxyWhenConfigured(t *testing.T) {
	t.Setenv("RELAY_TO_TRAE_BACKEND_PROXY_MODE", backendProxyModeSystem)

	client := newBackendHTTPClient(250 * time.Millisecond)
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("expected non-nil transport proxy in system mode")
	}
	if client.Timeout != 250*time.Millisecond {
		t.Fatalf("expected timeout 250ms, got %v", client.Timeout)
	}
}
