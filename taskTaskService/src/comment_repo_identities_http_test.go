package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func insertTaskProjectRepo(t *testing.T, taskID, repoURL string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_"+taskID, taskID, "proj-1", "main", "work", repoURL,
	)
	if err != nil {
		t.Fatalf("insert task_projects: %v", err)
	}
}

func TestCreateRunCommentRequiresRepoIdentitiesWhenLinkedRepo(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	taskID := createTestTaskForComments(t)
	insertTaskProjectRepo(t, taskID, "https://github.com/acme/demo.git")

	body := `{"content":"@My Image","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "repo_identities") && !strings.Contains(rec.Body.String(), errMsgRepoIdentitiesRequired) {
		t.Fatalf("expected identities error, got: %s", rec.Body.String())
	}
}

func TestCreateRunCommentPersistsRepoIdentitiesAndList(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	taskID := createTestTaskForComments(t)
	repoURL := "https://github.com/acme/demo.git"
	insertTaskProjectRepo(t, taskID, repoURL)

	body := `{
		"content":"@My Image run",
		"mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}],
		"repo_identities":[{"repo_url":"` + repoURL + `","git_identity_id":"gid-1","github_user_id":"1321779"}]
	}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	commentID, _ := resp["id"].(string)
	if commentID == "" {
		t.Fatal("comment id")
	}
	idents, _ := resp["repo_identities"].([]interface{})
	if len(idents) != 1 {
		t.Fatalf("resp identities=%v", resp["repo_identities"])
	}

	var stored string
	if err := db.QueryRow(`SELECT repo_identities_json FROM task_comments WHERE id=?`, commentID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stored, "gid-1") || !strings.Contains(stored, "1321779") {
		t.Fatalf("stored=%s", stored)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/"+taskID+"/comments/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	listRec := httptest.NewRecorder()
	handleCommentRoutes(listRec, req)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listRec.Code, listRec.Body.String())
	}
	var listed []map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) == 0 {
		t.Fatal("empty list")
	}
	got, _ := listed[0]["repo_identities"].([]interface{})
	if len(got) != 1 {
		t.Fatalf("list identities=%v", listed[0]["repo_identities"])
	}
}

func TestCreateCommentWithoutMentionDoesNotRequireIdentities(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)
	taskID := createTestTaskForComments(t)
	insertTaskProjectRepo(t, taskID, "https://github.com/acme/demo.git")
	rec := postComment(t, taskID, `{"content":"plain"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContainerSnapshotPrefersCommentRepoIdentities(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	taskID := createTestTaskForComments(t)
	repoURL := "https://github.com/acme/demo.git"
	insertTaskProjectRepo(t, taskID, repoURL)
	_, err := db.Exec(
		`INSERT INTO task_repo_identities(id,task_id,repo_url,git_identity_id) VALUES(?,?,?,?)`,
		"tri_"+taskID, taskID, repoURL, "gid-task",
	)
	if err != nil {
		t.Fatalf("task identities: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_git_identities(id,user_id,git_user_name,git_user_email) VALUES(?,?,?,?)`,
		"gid-comment", "99", "Ann Comment", "ann@comment.example",
	); err != nil {
		t.Fatalf("git identity: %v", err)
	}
	body := `{
		"content":"@My Image",
		"mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}],
		"repo_identities":[{"repo_url":"` + repoURL + `","git_identity_id":"gid-comment","github_user_id":"99"}]
	}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create comment: %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	commentID, _ := resp["id"].(string)
	if commentID == "" {
		t.Fatal("comment id")
	}

	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })
	req := httptest.NewRequest(http.MethodGet, "/api/internal/tasks/"+taskID+"/container-snapshot?comment_id="+commentID, nil)
	req.Header.Set("X-Task-Id", taskID)
	snapRec := httptest.NewRecorder()
	handleInternalContainerSnapshot(snapRec, req)
	if snapRec.Code != http.StatusOK {
		t.Fatalf("snapshot: %d %s", snapRec.Code, snapRec.Body.String())
	}
	var snap map[string]interface{}
	json.NewDecoder(snapRec.Body).Decode(&snap)
	idents, _ := snap["repo_identities"].([]interface{})
	if len(idents) != 1 {
		t.Fatalf("idents=%v", snap["repo_identities"])
	}
	row := idents[0].(map[string]interface{})
	if row["git_identity_id"] != "gid-comment" {
		t.Fatalf("want comment identity, got %v", row)
	}
	if row["user_name"] != "Ann Comment" || row["user_email"] != "ann@comment.example" {
		t.Fatalf("snapshot must include resolved git author, got %v", row)
	}
}

func TestContainerSnapshotFallsBackToTaskIdentities(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)
	taskID := createTestTaskForComments(t)
	repoURL := "https://git.example/a.git"
	insertTaskProjectRepo(t, taskID, repoURL)
	_, err := db.Exec(
		`INSERT INTO task_repo_identities(id,task_id,repo_url,git_identity_id) VALUES(?,?,?,?)`,
		"tri_fb_"+taskID, taskID, repoURL, "gid-task",
	)
	if err != nil {
		t.Fatalf("task identities: %v", err)
	}
	rec := postComment(t, taskID, `{"content":"legacy"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("comment: %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	commentID, _ := resp["id"].(string)
	if commentID == "" {
		t.Fatal("comment id")
	}

	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })
	req := httptest.NewRequest(http.MethodGet, "/api/internal/tasks/"+taskID+"/container-snapshot?comment_id="+commentID, nil)
	req.Header.Set("X-Task-Id", taskID)
	snapRec := httptest.NewRecorder()
	handleInternalContainerSnapshot(snapRec, req)
	if snapRec.Code != http.StatusOK {
		t.Fatalf("snapshot: %d %s", snapRec.Code, snapRec.Body.String())
	}
	var snap map[string]interface{}
	json.NewDecoder(snapRec.Body).Decode(&snap)
	idents, _ := snap["repo_identities"].([]interface{})
	if len(idents) != 1 {
		t.Fatalf("idents=%v", snap["repo_identities"])
	}
	row := idents[0].(map[string]interface{})
	if row["git_identity_id"] != "gid-task" {
		t.Fatalf("want task fallback, got %v", row)
	}
}
