package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRuntimeSessionOpen_CreatesConfigAndHistory(t *testing.T) {
	setupCloudTestDB(t)
	cfg.InternalSecret = ""

	body := `{"tenant_id":"t1","workspace_id":"w1","task_id":"task-open","runtime_source":"relay_local"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/runtime-session/open/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalRuntimeSessionOpen(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	sessionID, _ := out["session_id"].(string)
	if sessionID == "" {
		t.Fatal("expected session_id")
	}

	csc, err := loadCloudServerConfig("t1", "w1", "task-open")
	if err != nil || csc == nil {
		t.Fatalf("config err=%v cfg=%v", err, csc)
	}
	if csc.Platform != "relay-local" || csc.AuthorizationID != "relay-task-open" {
		t.Fatalf("cfg=%+v", csc)
	}
	h, err := loadCloudServerConfigHistoryByID(sessionID)
	if err != nil || h == nil {
		t.Fatalf("history err=%v", err)
	}
	if h.RuntimeSource != "relay_local" || h.Platform != "mock" || h.Region != "mock-local" {
		t.Fatalf("history=%+v", h)
	}
	if h.StoppedAt != nil {
		t.Fatalf("new session should be open, stopped_at=%v", h.StoppedAt)
	}
}

func TestRuntimeSessionOpen_SupersedesOpenHistory(t *testing.T) {
	setupCloudTestDB(t)
	cfg.InternalSecret = ""

	old := CloudServerConfigHistory{
		ID: "csh_old", CompanyID: "t1", WorkspaceID: "w1", TaskID: "task-open2",
		Platform: "mock", Region: "mock-local", ZoneID: "mock-local-a",
		AuthorizationID: "mock-auth", RuntimeSource: "relay_local",
	}
	if err := upsertCloudServerConfigHistory(old); err != nil {
		t.Fatal(err)
	}

	body := `{"tenant_id":"t1","workspace_id":"w1","task_id":"task-open2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/runtime-session/open", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalRuntimeSessionOpen(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	closed, err := loadCloudServerConfigHistoryByID("csh_old")
	if err != nil {
		t.Fatal(err)
	}
	if closed.StoppedAt == nil {
		t.Fatal("expected old session closed")
	}
	if closed.StopReason != "superseded_by_new_start" {
		t.Fatalf("stop_reason=%q", closed.StopReason)
	}
}

func TestRuntimeSessionOpen_MissingTenant(t *testing.T) {
	setupCloudTestDB(t)
	cfg.InternalSecret = ""
	req := httptest.NewRequest(http.MethodPost, "/api/internal/runtime-session/open/",
		strings.NewReader(`{"task_id":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalRuntimeSessionOpen(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}
