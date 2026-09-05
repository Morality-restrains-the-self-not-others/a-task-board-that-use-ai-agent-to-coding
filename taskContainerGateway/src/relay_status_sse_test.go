package main

import (
	"testing"
)

func TestBuildRelayToTraeStatusData_UsesErrorThenUIURL(t *testing.T) {
	got := buildRelayToTraeStatusData(map[string]any{
		"error":  "boom",
		"ui_url": "http://127.0.0.1/ui",
	})
	if got["status"] != relayToTraeSSEStatus {
		t.Fatalf("status=%v", got["status"])
	}
	if got["event_name"] != "server_status_update" {
		t.Fatalf("event_name=%v", got["event_name"])
	}
	if got["message"] != "boom" {
		t.Fatalf("message=%v want boom", got["message"])
	}
	payload, ok := got["relay_payload"].(map[string]any)
	if !ok || payload["error"] != "boom" {
		t.Fatalf("relay_payload=%#v", got["relay_payload"])
	}

	got = buildRelayToTraeStatusData(map[string]any{"ui_url": "http://x/ui"})
	if got["message"] != "http://x/ui" {
		t.Fatalf("message=%v want ui_url", got["message"])
	}
}

func TestPublishRelayStatusSSE_EmptyTaskIDNoPanic(t *testing.T) {
	publishRelayStatusSSE(nil, "", map[string]any{"running": false})
}
