package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func startForkProgressMockServer(t *testing.T, respond func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(respond))
	t.Cleanup(srv.Close)
	oldURL := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = oldURL })
	return srv
}

func TestApplyForkProgressColumnOverrideForcesFirstColumn(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	startForkProgressMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"progress_column_id":"col-first"}`))
	})

	body := map[string]interface{}{"fork_from": "task_src", "progress_column_id": "col-client"}
	applyForkProgressColumnOverride(body, "t1", "ws1")

	if body["progress_column_id"] != "col-first" {
		t.Fatalf("fork must force first column, got %v", body["progress_column_id"])
	}
	if gotPath != "/api/internal/progress-columns/first" {
		t.Fatalf("expected path /api/internal/progress-columns/first, got %s", gotPath)
	}
	if gotBody["tenant_id"] != "t1" || gotBody["workspace_id"] != "ws1" {
		t.Fatalf("unexpected body %v", gotBody)
	}
}

func TestApplyForkProgressColumnOverrideNonForkKeepsClientColumn(t *testing.T) {
	called := false
	startForkProgressMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	})

	body := map[string]interface{}{"progress_column_id": "col-client"}
	applyForkProgressColumnOverride(body, "t1", "ws1")

	if called {
		t.Fatal("non-fork must not call project service")
	}
	if body["progress_column_id"] != "col-client" {
		t.Fatalf("non-fork must keep client column, got %v", body["progress_column_id"])
	}
}

func TestApplyForkProgressColumnOverrideResolveErrorKeepsClientColumn(t *testing.T) {
	startForkProgressMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	body := map[string]interface{}{"fork_from": "task_src", "progress_column_id": "col-client"}
	applyForkProgressColumnOverride(body, "t1", "ws1")

	// 解析失败非致命：保留客户端列，不阻断创建。
	if body["progress_column_id"] != "col-client" {
		t.Fatalf("on resolve error must keep client column, got %v", body["progress_column_id"])
	}
}
