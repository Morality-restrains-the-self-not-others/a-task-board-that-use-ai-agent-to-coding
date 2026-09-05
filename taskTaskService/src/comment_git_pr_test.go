package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildGitPrReplySSEStatusData(t *testing.T) {
	got := buildGitPrReplySSEStatusData("cmt_1", "cmt_parent", "https://gl.example/a/b/-/merge_requests/1", "gitlab", "t1", "ws1", "u1")
	if got["event_name"] != "task_git_pr_reply_created" {
		t.Fatalf("event_name=%v", got["event_name"])
	}
	if got["comment_id"] != "cmt_1" || got["parent_comment_id"] != "cmt_parent" {
		t.Fatalf("ids=%v", got)
	}
	if got["git_pr_html_url"] != "https://gl.example/a/b/-/merge_requests/1" {
		t.Fatalf("url=%v", got["git_pr_html_url"])
	}
}

func TestCreateCommentGitPRPublishesDomainAndSSE(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)
	taskID := createTestTaskForComments(t)

	parentRec := postComment(t, taskID, `{"content":"parent exec","execution_mode":"independent"}`)
	if parentRec.Code != http.StatusCreated {
		t.Fatalf("parent: %d %s", parentRec.Code, parentRec.Body.String())
	}
	var parent map[string]interface{}
	_ = json.NewDecoder(parentRec.Body).Decode(&parent)
	parentID, _ := parent["id"].(string)

	type pub struct {
		typ  string
		data map[string]interface{}
	}
	var pubs []pub
	prev := publishDomainEventFn
	publishDomainEventFn = func(_ context.Context, eventType string, data map[string]interface{}, _ string) error {
		pubs = append(pubs, pub{typ: eventType, data: data})
		return nil
	}
	t.Cleanup(func() { publishDomainEventFn = prev })

	htmlURL := "https://gitlab.example/a/b/-/merge_requests/99"
	body := `{"content":"` + htmlURL + `","parent_comment_id":"` + parentID + `","execution_mode":"independent","git_pr":{"html_url":"` + htmlURL + `","provider":"gitlab"}}`
	first := postComment(t, taskID, body)
	if first.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", first.Code, first.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(first.Body).Decode(&created)
	id1, _ := created["id"].(string)

	var sawRecorded, sawSSE bool
	for _, p := range pubs {
		switch p.typ {
		case "TASK_GIT_PULL_REQUEST_RECORDED":
			sawRecorded = true
			if p.data["comment_id"] != id1 || p.data["git_pr_html_url"] != htmlURL {
				t.Fatalf("recorded data=%v", p.data)
			}
		case "SSE_MESSAGE":
			sawSSE = true
			sd, _ := p.data["status_data"].(map[string]interface{})
			if sd == nil || sd["event_name"] != "task_git_pr_reply_created" {
				t.Fatalf("sse status_data=%v", p.data["status_data"])
			}
			if sd["comment_id"] != id1 {
				t.Fatalf("sse comment_id=%v", sd["comment_id"])
			}
		}
	}
	if !sawRecorded || !sawSSE {
		t.Fatalf("want TASK_GIT_PULL_REQUEST_RECORDED+SSE_MESSAGE, got %#v", pubs)
	}

	pubs = nil
	second := postComment(t, taskID, body)
	if second.Code != http.StatusOK {
		t.Fatalf("idempotent: %d", second.Code)
	}
	if len(pubs) != 0 {
		t.Fatalf("idempotent skip must not re-publish, got %#v", pubs)
	}
}

func TestCreateCommentGitPRIdempotentAndNested(t *testing.T) {
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
	body := `{"content":"` + htmlURL + `","parent_comment_id":"` + parentID + `","execution_mode":"independent","git_pr":{"html_url":"` + htmlURL + `","provider":"gitlab"}}`
	first := postComment(t, taskID, body)
	if first.Code != http.StatusCreated {
		t.Fatalf("first git_pr comment: %d %s", first.Code, first.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(first.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	id1, _ := created["id"].(string)
	if id1 == "" {
		t.Fatal("missing comment id")
	}
	if created["parent_comment_id"] != parentID {
		t.Fatalf("parent=%v", created["parent_comment_id"])
	}
	gp, _ := created["git_pr"].(map[string]interface{})
	if gp == nil || gp["html_url"] != htmlURL {
		t.Fatalf("git_pr=%v", created["git_pr"])
	}

	second := postComment(t, taskID, body)
	if second.Code != http.StatusOK {
		t.Fatalf("idempotent git_pr comment: %d %s", second.Code, second.Body.String())
	}
	var again map[string]interface{}
	if err := json.NewDecoder(second.Body).Decode(&again); err != nil {
		t.Fatal(err)
	}
	if again["id"] != id1 {
		t.Fatalf("idempotent id=%v want %s", again["id"], id1)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/"+taskID+"/comments/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listReq.Header.Set("X-Auth-User-Id", "u1")
	listRec := httptest.NewRecorder()
	handleCommentRoutes(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listRec.Code, listRec.Body.String())
	}
	var rows []map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, row := range rows {
		if row["id"] == id1 {
			found++
			if row["parent_comment_id"] != parentID {
				t.Fatalf("listed parent=%v", row["parent_comment_id"])
			}
		}
	}
	if found != 1 {
		t.Fatalf("expected one git_pr row, found %d in %d comments", found, len(rows))
	}
}
