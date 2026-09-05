package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func startMutableProjectService(t *testing.T, gitRepos *[]string, mu *sync.Mutex) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathIsWorkspaceList(r) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "ws1", "name": "WS1", "company_id": "t1"},
			})
			return
		}
		if pathHasWorkspace(r, "ws1") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			w.Write([]byte("[]"))
			return
		}
		if strings.Contains(r.URL.Path, "/projects/tenant_id/") {
			mu.Lock()
			repos := append([]string(nil), *gitRepos...)
			mu.Unlock()
			json.NewEncoder(w).Encode(map[string]interface{}{"git_repos": repos})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""
}

func createTaskWithRepo(t *testing.T) string {
	t.Helper()
	body := `{
		"title":"hello world",
		"workspace_id":"ws1",
		"projects":[
			{"project_id":"p1","repo_index":0,"base_branch":"master","target_branch":"feature/x"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id")
	}
	return taskID
}

func getTaskProjects(t *testing.T, taskID string) []map[string]interface{} {
	t.Helper()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Auth-User-Id", "u1")
	recGet := httptest.NewRecorder()
	handleTaskRoutes(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get task: expected 200, got %d: %s", recGet.Code, recGet.Body.String())
	}
	var detail map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&detail)
	raw, ok := detail["projects"].([]interface{})
	if !ok || len(raw) == 0 {
		t.Fatalf("expected projects, got %#v", detail["projects"])
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("project row not object: %#v", item)
		}
		out = append(out, m)
	}
	return out
}

func TestGetTaskRepoAddressMismatchWhenProjectUrlChanged(t *testing.T) {
	setupTestDB(t)
	repos := []string{"https://github.com/ruandao/helloworld"}
	var mu sync.Mutex
	startMutableProjectService(t, &repos, &mu)

	taskID := createTaskWithRepo(t)

	mu.Lock()
	repos[0] = "https://github.com/test-ruandao/helloworld.git"
	mu.Unlock()

	row := getTaskProjects(t, taskID)[0]
	if row["stored_repo_address"] != "https://github.com/ruandao/helloworld" {
		t.Fatalf("stored_repo_address=%v", row["stored_repo_address"])
	}
	// B-086: GET task is fail-open; mismatch is computed from workspace catalog on the client.
	if row["repo_address_mismatch"] != false {
		t.Fatalf("GET task must not block on live project URL, mismatch=%#v", row["repo_address_mismatch"])
	}
	if url, _ := row["project_repo_url"].(string); strings.TrimSpace(url) != "" {
		t.Fatalf("project_repo_url must be empty on GET, got %#v", row["project_repo_url"])
	}
}

func TestGetTaskRepoAddressNotMismatchWhenOnlyGitSuffixDiffers(t *testing.T) {
	setupTestDB(t)
	repos := []string{"https://github.com/acme/demo"}
	var mu sync.Mutex
	startMutableProjectService(t, &repos, &mu)

	taskID := createTaskWithRepo(t)

	mu.Lock()
	repos[0] = "https://github.com/acme/demo.git"
	mu.Unlock()

	row := getTaskProjects(t, taskID)[0]
	if row["repo_address_mismatch"] != false {
		t.Fatalf("expected repo_address_mismatch=false for .git suffix, got %#v", row)
	}
}
