package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskGitOauth/infrastructure"
)

func seedAuditSiteProviders(app *App) {
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"github": {{
			Provider:        "github",
			ServiceProvider: "github-official",
			ProviderKey:     "github:github-official",
			Website:         "https://github.com",
		}},
		"gitlab": {{
			Provider:        "gitlab",
			ServiceProvider: "default",
			ProviderKey:     "gitlab:default",
			Website:         "http://localhost:8012",
		}},
	}
}

func TestInsertAccessAuditWritesSiteColumn(t *testing.T) {
	app := testApp(t)
	const userID = "877397583960502272"
	const site = "gitlab-tencent-sh-1.daydaymoney.com"
	if err := app.DB.InsertAccessAudit(site, userID, nil, nil, "oauth_callback_token_issued", "fp123", map[string]any{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	var gotSite, gotAction string
	err := app.DB.QueryRow(
		`SELECT site, action FROM git_oauth_appaccesstokenuseaudit WHERE task2app_user_id = ?`,
		userID,
	).Scan(&gotSite, &gotAction)
	if err != nil {
		t.Fatalf("SELECT site: %v", err)
	}
	if gotSite != site {
		t.Fatalf("site=%q want %q", gotSite, site)
	}
	if gotAction != "oauth_callback_token_issued" {
		t.Fatalf("action=%q", gotAction)
	}
}

func TestHandleTokenUseReportWritesHostPortSite(t *testing.T) {
	app := testApp(t)
	seedAuditSiteProviders(app)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","user_id":"42","audit":{"action":"clone","detail":{"repo":"test/repo"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-token-use-report/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var got string
	if err := app.DB.QueryRow(
		`SELECT site FROM git_oauth_appaccesstokenuseaudit WHERE action = ?`,
		"clone",
	).Scan(&got); err != nil {
		t.Fatalf("SELECT site: %v", err)
	}
	if got != "github.com" {
		t.Fatalf("site=%q want github.com", got)
	}

	body2 := `{"provider_key":"gitlab","user_id":"42","audit":{"action":"push"}}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/gitlab-token-use-report/", bytes.NewBufferString(body2))
	req2.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200 for gitlab, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	if err := app.DB.QueryRow(
		`SELECT site FROM git_oauth_appaccesstokenuseaudit WHERE action = ?`,
		"push",
	).Scan(&got); err != nil {
		t.Fatalf("SELECT gitlab site: %v", err)
	}
	if got != "localhost:8012" {
		t.Fatalf("gitlab site=%q want localhost:8012", got)
	}
}

func TestHandleTokenUseReportSiteUnresolved(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","user_id":"42","audit":{"action":"clone"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-token-use-report/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == 200 {
		t.Fatal("unresolved site must not succeed")
	}
	var n int
	if err := app.DB.QueryRow(`SELECT COUNT(*) FROM git_oauth_appaccesstokenuseaudit`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("must not write provider_key fallback, rows=%d", n)
	}
}

func TestAccessAuditSiteFromProviderKey(t *testing.T) {
	app := testApp(t)
	seedAuditSiteProviders(app)
	if got := app.accessAuditSite("github:github-official"); got != "github.com" {
		t.Fatalf("got %q", got)
	}
	if got := app.accessAuditSite("gitlab:default"); got != "localhost:8012" {
		t.Fatalf("got %q", got)
	}
}
