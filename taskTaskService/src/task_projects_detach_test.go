package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetachTaskProjectsByProjectIDRemovesRows(t *testing.T) {
	setupTestDB(t)
	projectID := "proj_deleted_1"
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_a", "task_a", projectID, "main", "feat", "https://example.com/a.git",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_b", "task_b", projectID, "main", "feat", "https://example.com/b.git",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_keep", "task_c", "proj_keep", "main", "feat", "https://example.com/c.git",
	); err != nil {
		t.Fatal(err)
	}

	n, err := detachTaskProjectsByProjectID(projectID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("deleted=%d want 2", n)
	}
	var leftover int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_projects WHERE project_id=?`, projectID).Scan(&leftover); err != nil {
		t.Fatal(err)
	}
	if leftover != 0 {
		t.Fatalf("leftover=%d want 0", leftover)
	}
	var kept int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_projects WHERE project_id=?`, "proj_keep").Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if kept != 1 {
		t.Fatalf("kept=%d want 1", kept)
	}

	n2, err := detachTaskProjectsByProjectID(projectID)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 0 {
		t.Fatalf("replay deleted=%d want 0", n2)
	}
}

func TestHandleInternalDetachTaskProjects(t *testing.T) {
	setupTestDB(t)
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_http", "task_http", "proj_http", "main", "feat", "",
	); err != nil {
		t.Fatal(err)
	}
	body := `{"project_id":"proj_http"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-projects/detach-by-project/", strings.NewReader(body))
	req.Header.Set("X-Auth-User-Id", "internal")
	rec := httptest.NewRecorder()
	handleInternalDetachTaskProjects(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["project_id"] != "proj_http" {
		t.Fatalf("payload=%#v", payload)
	}
	var leftover int
	_ = db.QueryRow(`SELECT COUNT(*) FROM task_projects WHERE project_id=?`, "proj_http").Scan(&leftover)
	if leftover != 0 {
		t.Fatalf("leftover=%d", leftover)
	}
}
