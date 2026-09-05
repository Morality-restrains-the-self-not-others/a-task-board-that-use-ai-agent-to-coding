package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"taskGitOauth/infrastructure"
)

// TestGitlabCallbackExchangeErrorClassification — 回归 2026-08-08（OPT-20260807-029）：
// GitLab 换票失败分类。明确拒绝（HTTP 4xx，或 200 但 body 携带 error，如 invalid_grant /
// redirect_uri_mismatch）→ 302 带 gitlab=exchange_rejected；网络/配置类失败 →
// gitlab=exchange_failed。与 GitHub 侧分类对齐，前端据此分级提示。
func TestGitlabCallbackExchangeErrorClassification(t *testing.T) {
	type serverCase struct {
		status int
		body   string
	}
	mkApp := func(t *testing.T, gitlabWebsite string, clientID, clientSecret string) *App {
		t.Helper()
		app := providerTestApp(t)
		row := &app.Cfg.Providers["gitlab"][0]
		row.Website = gitlabWebsite
		row.ClientID = clientID
		row.ClientSecret = clientSecret
		return app
	}
	stateFor := func(t *testing.T, app *App) string {
		t.Helper()
		state, err := app.Sess.EncodeOAuthState(infrastructure.OAuthBrowserState{
			CSRF:            "csrf-gitlab-x",
			UID:             "873093522473906176",
			Next:            "/profile/git-site-oauth/",
			Feb:             "https://www.daydaymoney.com",
			RedirectURI:     "https://gitoauth_api.daydaymoney.com/api/accounts/gitlab/oauth/callback/",
			ServiceProvider: "gitlab-local",
		})
		if err != nil {
			t.Fatal(err)
		}
		return state
	}
	runCase := func(t *testing.T, app *App, state, wantKey string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet,
			"/api/accounts/gitlab/oauth/callback/?code=fake&state="+url.QueryEscape(state), nil)
		rec := httptest.NewRecorder()
		app.handleGitlabCallback(rec, req)
		loc := rec.Header().Get("Location")
		if !strings.Contains(loc, "gitlab="+wantKey) {
			t.Fatalf("want gitlab=%s in redirect, got %q", wantKey, loc)
		}
	}

	t.Run("rejected_4xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid_grant","error_description":"The provided authorization grant is invalid"}`))
		}))
		defer srv.Close()
		app := mkApp(t, srv.URL, "cid", "sec")
		runCase(t, app, stateFor(t, app), "exchange_rejected")
	})

	t.Run("rejected_200_with_error_body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"error":"invalid_grant"}`))
		}))
		defer srv.Close()
		app := mkApp(t, srv.URL, "cid", "sec")
		runCase(t, app, stateFor(t, app), "exchange_rejected")
	})

	t.Run("network_failure", func(t *testing.T) {
		// 关闭本地端口 → 直连 connection refused（非拒绝语义）→ exchange_failed
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		deadURL := "http://" + l.Addr().String()
		l.Close()
		app := mkApp(t, deadURL, "cid", "sec")
		runCase(t, app, stateFor(t, app), "exchange_failed")
	})

	t.Run("missing_credentials", func(t *testing.T) {
		app := mkApp(t, "http://gitlab.example", "", "")
		runCase(t, app, stateFor(t, app), "exchange_failed")
	})
}
