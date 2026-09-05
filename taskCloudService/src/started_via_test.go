package main

import (
	"net/http/httptest"
	"testing"
)

func TestResolveStartedViaTrustedQueued(t *testing.T) {
	prev := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Internal-Secret", "sec")
	via := resolveStartedVia(req, map[string]interface{}{"started_via": "queued_schedule"})
	if via != "queued_schedule" {
		t.Fatalf("got %q", via)
	}
}

func TestResolveStartedViaUntrustedQueuedBecomesManual(t *testing.T) {
	prev := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	req := httptest.NewRequest("POST", "/", nil)
	via := resolveStartedVia(req, map[string]interface{}{"started_via": "queued_schedule"})
	if via != "manual" {
		t.Fatalf("got %q", via)
	}
}
