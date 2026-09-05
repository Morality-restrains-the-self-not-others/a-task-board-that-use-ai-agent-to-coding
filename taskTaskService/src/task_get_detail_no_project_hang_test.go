package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetTaskDetailDoesNotWaitForProjectHang(t *testing.T) {
	setupTestDB(t)

	var hangProjectGet atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		if pathIsWorkspaceList(r) {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "ws1", "name": "WS1", "company_id": "t1"},
			})
			return
		}
		if pathHasWorkspace(r, "ws1") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(path, "workspace-access") {
			_, _ = w.Write([]byte("[]"))
			return
		}
		if strings.Contains(path, "/projects/tenant_id/") {
			if hangProjectGet.Load() {
				time.Sleep(3 * time.Second)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"git_repos": []string{"https://github.com/ruandao/helloworld"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""

	taskID := createTaskWithRepo(t)
	hangProjectGet.Store(true)

	start := time.Now()
	row := getTaskProjects(t, taskID)[0]
	elapsed := time.Since(start)
	if elapsed > time.Second {
		t.Fatalf("GET task waited on project GET hang: elapsed=%s", elapsed)
	}
	if row["stored_repo_address"] != "https://github.com/ruandao/helloworld" {
		t.Fatalf("stored_repo_address=%v", row["stored_repo_address"])
	}
	if row["repo_address_mismatch"] != false {
		t.Fatalf("mismatch must fail-open on GET, got %#v", row["repo_address_mismatch"])
	}
	if url, _ := row["project_repo_url"].(string); strings.TrimSpace(url) != "" {
		t.Fatalf("project_repo_url must be empty on GET, got %#v", row["project_repo_url"])
	}
}
