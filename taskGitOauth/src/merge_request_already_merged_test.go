package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"taskGitOauth/domain"
)

func TestParseMergeRequestStateMergedAtOverridesOpened(t *testing.T) {
	ref, err := domain.ParseMergeRequestURL(gitlabHTMLURL())
	if err != nil {
		t.Fatal(err)
	}
	st := parseMergeRequestState(ref, map[string]any{
		"state":     "opened",
		"title":     "already done",
		"merged_at": "2026-08-23T17:17:11.000+08:00",
	})
	if st.State != "merged" {
		t.Fatalf("merged_at must count as merged, got %q", st.State)
	}
	if st.Title != "already done" {
		t.Fatalf("title=%q", st.Title)
	}
}

func TestParseMergeRequestStateOpenedWithoutMergedAtStaysOpen(t *testing.T) {
	ref, err := domain.ParseMergeRequestURL(gitlabHTMLURL())
	if err != nil {
		t.Fatal(err)
	}
	st := parseMergeRequestState(ref, map[string]any{
		"state":     "opened",
		"merged_at": nil,
	})
	if st.State != "open" {
		t.Fatalf("state=%q", st.State)
	}
}

func TestMergeRequestMergeReconciles405WhenGitLabAlreadyMerged(t *testing.T) {
	app := mergeTestApp(t)
	gets, puts := 0, 0
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		switch req.Method {
		case http.MethodGet:
			gets++
			if gets == 1 {
				return jsonHTTPResponse(http.StatusOK, `{"state":"opened","title":"x"}`), nil
			}
			return jsonHTTPResponse(http.StatusOK, `{"state":"merged","merged_at":"2026-08-23T17:17:11+08:00"}`), nil
		case http.MethodPut:
			puts++
			return jsonHTTPResponse(http.StatusMethodNotAllowed, `{"message":"405 Method Not Allowed"}`), nil
		default:
			t.Fatalf("unexpected %s %s", req.Method, req.URL)
			return nil, nil
		}
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["merged"] != true || out["noop"] != true {
		t.Fatalf("want noop already merged, body=%v", out)
	}
	if puts != 1 || gets != 2 {
		t.Fatalf("gets=%d puts=%d", gets, puts)
	}
}

func TestMergeRequestMergeReconcilesTimeoutWhenGitLabAlreadyMerged(t *testing.T) {
	app := mergeTestApp(t)
	gets := 0
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		switch req.Method {
		case http.MethodGet:
			gets++
			if gets == 1 {
				return jsonHTTPResponse(http.StatusOK, `{"state":"opened"}`), nil
			}
			return jsonHTTPResponse(http.StatusOK, `{"state":"merged","merged_at":"2026-08-23T17:17:11+08:00"}`), nil
		case http.MethodPut:
			return nil, fmt.Errorf(
				`Put "https://gitlab-tencent-sh-1.daydaymoney.com/api/v4/projects/x/merge": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`,
			)
		default:
			t.Fatalf("unexpected %s %s", req.Method, req.URL)
			return nil, nil
		}
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"noop":true`) {
		t.Fatalf("timeout then already-merged must be noop success: %s", rr.Body.String())
	}
}

func TestMergeRequestMerge405StillOpenRemainsFailure(t *testing.T) {
	app := mergeTestApp(t)
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return jsonHTTPResponse(http.StatusOK, `{"state":"opened"}`), nil
		}
		if req.Method == http.MethodPut {
			return jsonHTTPResponse(http.StatusMethodNotAllowed, `{"message":"405 Method Not Allowed"}`), nil
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL)
		return nil, nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+gitlabHTMLURL()+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code == http.StatusOK {
		t.Fatalf("unmergeable open MR must not succeed: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "git merge http 405") {
		t.Fatalf("want original merge error, got %s", rr.Body.String())
	}
}
