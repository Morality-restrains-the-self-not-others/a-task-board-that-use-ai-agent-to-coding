package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

// Path A 凭据 provider_key=gitlab:tenant-{company} 只在 DB，不在 YAML。
// refreshAccessToken 若只扫 GetProviderConfigs 会「未找到 provider_key」→ 项目页授权异常。
func TestRefreshAccessTokenUsesTenantPathAConfig(t *testing.T) {
	app := testApp(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "grant_type=refresh_token") {
			t.Errorf("expected refresh_token grant, got %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"glpat-from-path-a","refresh_token":"rt-rotated","token_type":"Bearer"}`))
	}))
	t.Cleanup(ts.Close)

	tid := "877397588196749312"
	cipher, err := app.Fernet.Encrypt("path-a-secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID:       tid,
		BaseURL:         ts.URL,
		ClientID:        "path-a-client",
		ClientSecretEnc: cipher,
		RedirectURI:     domain.DefaultRedirectURI(app.Cfg.PublicBaseURL, tid),
		Scope:           domain.DefaultTenantGitLabScope,
		Active:          true,
		UpdatedAt:       time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := app.refreshAccessToken(domain.ProviderKeyForCompany(tid), "old-refresh")
	if err != nil {
		t.Fatalf("refreshAccessToken: %v", err)
	}
	if got := strings.TrimSpace(fmt.Sprint(out["access_token"])); got != "glpat-from-path-a" {
		t.Fatalf("access_token=%q", got)
	}
}
