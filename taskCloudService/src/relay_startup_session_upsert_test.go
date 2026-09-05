package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUpsertRelayStartupSessionTokenInitWritesKeys(t *testing.T) {
	client := setupRelayMemoryStore(t)
	now := time.Date(2026, 7, 10, 2, 0, 0, 0, time.UTC)
	session, err := upsertRelayStartupSession(context.Background(), relaySessions, upsertRelayStartupSessionRequest{
		WorkflowID:       "wf-token-1",
		TenantID:         "t1",
		WorkspaceID:      "w1",
		TaskID:           "task1",
		Phase:            relayPhaseTokenInitSucceeded,
		TokenInitialized: true,
	}, now)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if session.Phase != relayPhaseTokenInitSucceeded || !session.TokenInitialized {
		t.Fatalf("session=%+v", session)
	}

	raw, ok, err := client.Get(context.Background(), workflowKey("wf-token-1"))
	if err != nil || !ok {
		t.Fatalf("wf key: ok=%v err=%v", ok, err)
	}
	loaded, err := deserializeRelaySession(raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if loaded.Phase != relayPhaseTokenInitSucceeded || !loaded.TokenInitialized {
		t.Fatalf("loaded=%+v", loaded)
	}
	if loaded.CreatedAtRaw == "" || loaded.UpdatedAtRaw == "" {
		t.Fatalf("timestamps missing: %+v", loaded)
	}

	scopePtr, ok, err := client.Get(context.Background(), scopeKey(RelayTaskScope{
		TenantID: "t1", WorkspaceID: "w1", TaskID: "task1",
	}))
	if err != nil || !ok || scopePtr != "wf-token-1" {
		t.Fatalf("scope ptr=%q ok=%v err=%v", scopePtr, ok, err)
	}
}

func TestUpsertRelayStartupSessionStartAcceptedMerges(t *testing.T) {
	client := setupRelayMemoryStore(t)
	now := time.Date(2026, 7, 10, 2, 0, 0, 0, time.UTC)
	_, err := upsertRelayStartupSession(context.Background(), relaySessions, upsertRelayStartupSessionRequest{
		WorkflowID:       "wf-start-1",
		TenantID:         "t1",
		WorkspaceID:      "w1",
		TaskID:           "task1",
		Phase:            relayPhaseTokenInitSucceeded,
		TokenInitialized: true,
	}, now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("token upsert: %v", err)
	}

	reqID := "req-start-1"
	session, err := upsertRelayStartupSession(context.Background(), relaySessions, upsertRelayStartupSessionRequest{
		WorkflowID:       "wf-start-1",
		TenantID:         "t1",
		WorkspaceID:      "w1",
		TaskID:           "task1",
		Phase:            relayPhaseStartAccepted,
		TokenInitialized: true,
		RequestID:        reqID,
	}, now)
	if err != nil {
		t.Fatalf("start upsert: %v", err)
	}
	if session.Phase != relayPhaseStartAccepted {
		t.Fatalf("phase=%q", session.Phase)
	}
	if session.RequestID == nil || *session.RequestID != reqID {
		t.Fatalf("request_id=%v", session.RequestID)
	}
	if !session.TokenInitialized {
		t.Fatal("token_initialized cleared")
	}

	raw, ok, _ := client.Get(context.Background(), workflowKey("wf-start-1"))
	if !ok {
		t.Fatal("missing wf key")
	}
	loaded, err := deserializeRelaySession(raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if loaded.Phase != relayPhaseStartAccepted || loaded.RequestID == nil || *loaded.RequestID != reqID {
		t.Fatalf("loaded=%+v", loaded)
	}
}

func TestHandleInternalRelayStartupSessionUpsert(t *testing.T) {
	_ = setupRelayMemoryStore(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "cloud-secret"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	body, _ := json.Marshal(map[string]any{
		"workflow_id":       "wf-http-1",
		"tenant_id":         "t1",
		"workspace_id":      "w1",
		"task_id":           "task1",
		"phase":             relayPhaseTokenInitSucceeded,
		"token_initialized": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/internal/relay-startup-session/upsert/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "cloud-secret")
	rec := httptest.NewRecorder()
	handleInternalRelayStartupSessionUpsert(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "ok" || resp["workflow_id"] != "wf-http-1" {
		t.Fatalf("resp=%v", resp)
	}
	if resp["phase"] != relayPhaseTokenInitSucceeded || resp["token_initialized"] != true {
		t.Fatalf("resp=%v", resp)
	}
}

func TestHandleInternalRelayStartupSessionUpsertForbidden(t *testing.T) {
	_ = setupRelayMemoryStore(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "cloud-secret"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	body, _ := json.Marshal(map[string]any{
		"workflow_id": "wf-x", "tenant_id": "t", "workspace_id": "w", "task_id": "task",
		"phase": relayPhaseTokenInitSucceeded, "token_initialized": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/internal/relay-startup-session/upsert/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalRelayStartupSessionUpsert(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
}
