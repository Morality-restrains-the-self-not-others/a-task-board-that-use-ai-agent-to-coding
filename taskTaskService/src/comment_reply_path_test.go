package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseTenantIDCommentReplyPath(t *testing.T) {
	tid, wid, taskID, parentID, ok := parseTenantIDCommentReplyPath(
		"/api/tenant_id/t1/workspaceId/ws1/tasks/task_9/comments/cmt-exec/",
	)
	if !ok || tid != "t1" || wid != "ws1" || taskID != "task_9" || parentID != "cmt-exec" {
		t.Fatalf("got tid=%q wid=%q task=%q parent=%q ok=%v", tid, wid, taskID, parentID, ok)
	}
	if _, _, _, _, ok := parseTenantIDCommentReplyPath("/api/tasks/task_9/comments/tenant_id/t1/"); ok {
		t.Fatal("old comment create path must not parse as reply path")
	}
	if _, _, _, _, ok := parseTenantIDCommentReplyPath("/api/tenant_id/t1/workspaceId/ws1/tasks/task_9/comments/"); ok {
		t.Fatal("missing parent_comment_id must not parse")
	}
}

func TestCreateCommentReplyUsesPathParentAndWorkspace(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)
	taskID := createTestTaskForComments(t)

	parentRec := postComment(t, taskID, `{"content":"parent exec","execution_mode":"independent"}`)
	if parentRec.Code != http.StatusCreated {
		t.Fatalf("parent comment: %d %s", parentRec.Code, parentRec.Body.String())
	}
	var parent map[string]interface{}
	if err := json.NewDecoder(parentRec.Body).Decode(&parent); err != nil {
		t.Fatal(err)
	}
	parentID, _ := parent["id"].(string)
	if parentID == "" {
		t.Fatal("missing parent id")
	}

	htmlURL := "https://gitlab-tencent-sh-1.daydaymoney.com/ljy/somanyad/-/merge_requests/12"
	body := `{"content":"` + htmlURL + `","execution_mode":"independent","git_pr":{"html_url":"` + htmlURL + `","provider":"gitlab"}}`
	path := "/api/tenant_id/t1/workspaceId/ws1/tasks/" + taskID + "/comments/" + parentID + "/"
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("Content-Type", "application/json")
	rec := serveMux(req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("reply: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created["parent_comment_id"] != parentID {
		t.Fatalf("parent=%v want %s", created["parent_comment_id"], parentID)
	}

	again := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	again.Header.Set("X-Auth-Tenant-Id", "t1")
	again.Header.Set("X-Auth-User-Id", "u1")
	again.Header.Set("Content-Type", "application/json")
	againRec := serveMux(again)
	if againRec.Code != http.StatusOK {
		t.Fatalf("idempotent reply: %d %s", againRec.Code, againRec.Body.String())
	}

	wrongWS := httptest.NewRequest(http.MethodPost,
		"/api/tenant_id/t1/workspaceId/other-ws/tasks/"+taskID+"/comments/"+parentID+"/",
		strings.NewReader(body))
	wrongWS.Header.Set("X-Auth-Tenant-Id", "t1")
	wrongWS.Header.Set("X-Auth-User-Id", "u1")
	wrongWS.Header.Set("Content-Type", "application/json")
	if rec := serveMux(wrongWS); rec.Code != http.StatusNotFound {
		t.Fatalf("workspace mismatch: %d %s", rec.Code, rec.Body.String())
	}

	missingParent := httptest.NewRequest(http.MethodPost,
		"/api/tenant_id/t1/workspaceId/ws1/tasks/"+taskID+"/comments/cmt-missing/",
		strings.NewReader(body))
	missingParent.Header.Set("X-Auth-Tenant-Id", "t1")
	missingParent.Header.Set("X-Auth-User-Id", "u1")
	missingParent.Header.Set("Content-Type", "application/json")
	if rec := serveMux(missingParent); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing parent: %d %s", rec.Code, rec.Body.String())
	}
}
