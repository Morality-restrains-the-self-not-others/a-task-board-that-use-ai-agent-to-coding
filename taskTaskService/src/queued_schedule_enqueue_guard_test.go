package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHttpStatusOfWorkspaceAutoScheduleDisabled(t *testing.T) {
	err := errWorkspaceAutoScheduleDisabled()
	if httpStatusOf(err, http.StatusBadRequest) != http.StatusConflict {
		t.Fatalf("status=%d", httpStatusOf(err, http.StatusBadRequest))
	}
	if !strings.Contains(err.Error(), "尚未启用自动调度") {
		t.Fatalf("msg=%q", err.Error())
	}
	if httpStatusOf(errors.New("plain"), http.StatusBadRequest) != http.StatusBadRequest {
		t.Fatal("fallback")
	}
}

func TestEnqueueQueuedAutoRunRejectsWhenWorkspaceScheduleDisabled(t *testing.T) {
	setupTestDB(t)
	setupWorkspaceRhythmAllDay(t, "t1", "ws1", false, 1)
	id := insertQueuedTestTask(t, "t1", "ws1", "disabled-queue", "")
	rec, err := loadTask(id)
	if err != nil || rec == nil {
		t.Fatalf("loadTask: %v", err)
	}
	_, err = enqueueQueuedAutoRun(rec, "u-test")
	if err == nil {
		t.Fatal("want 409 error")
	}
	if httpStatusOf(err, 0) != http.StatusConflict {
		t.Fatalf("status=%d err=%v", httpStatusOf(err, 0), err)
	}
	m, mErr := loadMembership(id)
	if mErr != nil {
		t.Fatal(mErr)
	}
	if m != nil {
		t.Fatalf("membership must not be inserted: %+v", m)
	}
}

func TestEnqueueQueuedAutoRunAllowsWhenWorkspaceScheduleEnabled(t *testing.T) {
	setupTestDB(t)
	setupWorkspaceRhythmAllDay(t, "t1", "ws1", true, 1)
	id := insertQueuedTestTask(t, "t1", "ws1", "enabled-queue", "")
	rec, err := loadTask(id)
	if err != nil || rec == nil {
		t.Fatalf("loadTask: %v", err)
	}
	m, err := enqueueQueuedAutoRun(rec, "u-test")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil || m.TaskID != id {
		t.Fatalf("membership=%+v", m)
	}
}

func TestEnqueueQueuedAutoRunLegacyWithoutWorkspaceRhythm(t *testing.T) {
	setupTestDB(t)
	id := insertQueuedTestTask(t, "t1", "ws-legacy", "legacy-queue", "")
	rec, err := loadTask(id)
	if err != nil || rec == nil {
		t.Fatalf("loadTask: %v", err)
	}
	m, err := enqueueQueuedAutoRun(rec, "u-test")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil || m.Status != "deferred" {
		t.Fatalf("legacy enqueue want deferred, got %+v", m)
	}
}

func TestCreateTaskQueuedAutoRunConflictWhenWorkspaceScheduleDisabled(t *testing.T) {
	setupTestDB(t)
	startAutoRunMockServices(t, true)
	setupWorkspaceRhythmAllDay(t, "t1", "ws1", false, 1)

	body := `{
		"title":"Queued while schedule off",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"queued_auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("create: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&payload)
	msg, _ := payload["error"].(string)
	if msg == "" {
		msg, _ = payload["message"].(string)
	}
	if !strings.Contains(rec.Body.String(), "尚未启用自动调度") && !strings.Contains(msg, "尚未启用自动调度") {
		t.Fatalf("body=%s", rec.Body.String())
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM task_queued_auto_run_memberships`).Scan(&n)
	if n != 0 {
		t.Fatalf("membership rows=%d", n)
	}
}
