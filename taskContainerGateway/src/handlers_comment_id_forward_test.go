package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleContainerLayerGraph_CommentIDSelectsToken(t *testing.T) {
	var gotCommentID, gotToken string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Access-Token")
		if gotToken != "tok-a" {
			http.Error(w, "wrong token", http.StatusUnauthorized)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/layers") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"layers":             []any{map[string]any{"layer_id": "L1"}},
				"layers_root":        "/layers",
				"bootstrap_layer_id": "L1",
			})
		case strings.HasSuffix(r.URL.Path, "/jobs") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"jobs": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok-fallback")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			gotCommentID = r.URL.Query().Get("comment_id")
			token := "tok-b"
			if gotCommentID == "cmt-a" {
				token = "tok-a"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": token,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-graph/?comment_id=cmt-a"
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Token test-token-abc12345")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gotCommentID != "cmt-a" {
		t.Fatalf("container-target comment_id=%q want cmt-a", gotCommentID)
	}
	if gotToken != "tok-a" {
		t.Fatalf("upstream token=%q want tok-a (must not use tok-b)", gotToken)
	}
}

func TestHandleContainerLayerGitCommit_POSTCommentIDQuery(t *testing.T) {
	var gotCommentID, gotToken string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Access-Token")
		if gotToken != "tok-a" {
			http.Error(w, "wrong token", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok-fallback")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			gotCommentID = r.URL.Query().Get("comment_id")
			token := "tok-b"
			if gotCommentID == "cmt-a" {
				token = "tok-a"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": token,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-git-commit/?comment_id=cmt-a"
	body := `{"layer_id":"L1","message":"x","comment_id":"cmt-a"}`
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token test-token-abc12345")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gotCommentID != "cmt-a" {
		t.Fatalf("container-target comment_id=%q want cmt-a", gotCommentID)
	}
	if gotToken != "tok-a" {
		t.Fatalf("upstream token=%q want tok-a", gotToken)
	}
}
