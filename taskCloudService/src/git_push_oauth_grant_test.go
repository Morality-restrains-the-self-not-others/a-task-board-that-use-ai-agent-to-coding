package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func startTaskCommentGrantServer(t *testing.T, userID, gitsite string, extra map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{
			"id": "task1",
			"comments": []any{
				map[string]any{
					"created_by": map[string]any{"id": userID},
					"repo_identities": []any{
						map[string]any{
							"oauth_gitsite": gitsite,
							"repo_url":      "https://" + gitsite + "/org/repo.git",
						},
					},
				},
			},
		}
		for k, v := range extra {
			body[k] = v
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCommentOAuthGrantsFromTaskBody(t *testing.T) {
	grants := commentOAuthGrantsFromTaskBody(map[string]any{
		"comments": []any{
			map[string]any{
				"created_by": map[string]any{"id": "42"},
				"repo_identities": []any{
					map[string]any{"oauth_gitsite": "GitHub.com", "repo_url": "https://github.com/a/b.git"},
				},
			},
		},
	})
	if !userHasCommentOAuthGrant(grants, "42", "github.com") {
		t.Fatal("expected grant for comment author")
	}
	if userHasCommentOAuthGrant(grants, "99", "github.com") {
		t.Fatal("other user must not inherit comment L2")
	}
}

func TestResolveLayerGitPushOauthMaps_RefusesWithoutCommentGrant(t *testing.T) {
	prevTask := cfg.TaskServiceURL
	prevOauth := cfg.GitOauthBaseURL
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.GitOauthBaseURL = prevOauth
	})
	hits := 0
	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ghu_must_not"})
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "task1",
			"projects": []any{
				map[string]any{
					"project_id":          "p1",
					"stored_repo_address": "https://github.com/acme/demo.git",
				},
			},
			"comments": []any{},
		})
	}))
	defer taskSrv.Close()
	cfg.TaskServiceURL = taskSrv.URL

	_, _, errDetail := resolveLayerGitPushOauthMaps(context.Background(), "t1", "w1", "task1", "42", "")
	if hits != 0 {
		t.Fatalf("access-for-user hits=%d want 0", hits)
	}
	if errDetail == "" || !strings.Contains(errDetail, "使用授权") {
		t.Fatalf("errDetail=%q", errDetail)
	}
}

func TestCommentOAuthGrantsFromTaskBody_PrefersCompactField(t *testing.T) {
	grants := commentOAuthGrantsFromTaskBody(map[string]any{
		"comment_oauth_grants": []any{
			map[string]any{"user_id": "42", "gitsite": "github.com"},
		},
		"comments": []any{},
	})
	if !userHasCommentOAuthGrant(grants, "42", "github.com") {
		t.Fatal("expected compact grant")
	}
}

func TestCommentOAuthGrantsFromTaskBody_EmptyCompactWinsOverComments(t *testing.T) {
	grants := commentOAuthGrantsFromTaskBody(map[string]any{
		"comment_oauth_grants": []any{},
		"comments": []any{
			map[string]any{
				"created_by": map[string]any{"id": "42"},
				"repo_identities": []any{
					map[string]any{"oauth_gitsite": "github.com"},
				},
			},
		},
	})
	if userHasCommentOAuthGrant(grants, "42", "github.com") {
		t.Fatal("empty compact field must be authoritative")
	}
}
