package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGrantTicketsFromCreateBody(t *testing.T) {
	t.Parallel()
	got := grantTicketsFromCreateBody(map[string]interface{}{
		"grant_ticket":  " tkt-a ",
		"grant_tickets": []interface{}{"tkt-a", "tkt-b", ""},
	})
	if len(got) != 2 || got[0] != "tkt-a" || got[1] != "tkt-b" {
		t.Fatalf("tickets=%v", got)
	}
}

func TestApplyCreateProjectGrantTicketsWritesL2(t *testing.T) {
	setupTestDB(t)
	orig := consumeGitOAuthGrantTicketFn
	t.Cleanup(func() { consumeGitOAuthGrantTicketFn = orig })
	var sawUser, sawSite, sawTicket string
	consumeGitOAuthGrantTicketFn = func(userID, gitsite, ticketID string) (string, bool) {
		sawUser, sawSite, sawTicket = userID, gitsite, ticketID
		return "42", true
	}
	entries := []gitRepoEntry{{
		URL: "https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git",
	}}
	applyCreateProjectGrantTickets("proj_1", "t1", "u1", entries, []string{"tkt-1"}, "trace-1")
	if sawUser != "u1" || sawSite != "gitlab-tencent-sh-1.daydaymoney.com" || sawTicket != "tkt-1" {
		t.Fatalf("consume args user=%s site=%s ticket=%s", sawUser, sawSite, sawTicket)
	}
	if !hasProjectGitOAuthGrant("proj_1", "u1", "gitlab-tencent-sh-1.daydaymoney.com") {
		t.Fatal("expected project L2 grant")
	}
	remote, ok := lookupProjectGitOAuthGrant("proj_1", "u1", "gitlab-tencent-sh-1.daydaymoney.com")
	if !ok || remote != "42" {
		t.Fatalf("lookup ok=%v remote=%q", ok, remote)
	}
}

func TestHandleCreateProjectConsumesGrantTicketIntoL2(t *testing.T) {
	setupTestDB(t)
	orig := consumeGitOAuthGrantTicketFn
	t.Cleanup(func() { consumeGitOAuthGrantTicketFn = orig })
	consumeGitOAuthGrantTicketFn = func(userID, gitsite, ticketID string) (string, bool) {
		if userID != "u1" || ticketID != "tkt-create-1" {
			t.Fatalf("unexpected consume user=%s ticket=%s", userID, ticketID)
		}
		if gitsite != "gitlab-tencent-sh-1.daydaymoney.com" {
			t.Fatalf("gitsite=%s", gitsite)
		}
		return "99", true
	}

	mux := http.NewServeMux()
	mountRoutes(mux)
	body := `{"name":"OAuthL2","description":"from create oauth","git_repos":["https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git"],"grant_ticket":"tkt-create-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Trace-Id", "trace-create-oauth")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatal("missing project id")
	}
	if !hasProjectGitOAuthGrant(pid, "u1", "gitlab-tencent-sh-1.daydaymoney.com") {
		t.Fatal("create should persist project L2 from grant_ticket")
	}
}

func TestHandleCreateProjectGrantTicketFailureStillCreates(t *testing.T) {
	setupTestDB(t)
	orig := consumeGitOAuthGrantTicketFn
	t.Cleanup(func() { consumeGitOAuthGrantTicketFn = orig })
	consumeGitOAuthGrantTicketFn = func(userID, gitsite, ticketID string) (string, bool) {
		return "", false
	}

	mux := http.NewServeMux()
	mountRoutes(mux)
	body := `{"name":"OAuthFail","description":"ticket unusable","git_repos":["https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git"],"grant_ticket":"tkt-dead"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	pid, _ := created["id"].(string)
	if hasProjectGitOAuthGrant(pid, "u1", "gitlab-tencent-sh-1.daydaymoney.com") {
		t.Fatal("unusable ticket must not write L2")
	}
}
