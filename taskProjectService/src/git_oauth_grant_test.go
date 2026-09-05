package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplyResourceGrantGate(t *testing.T) {
	t.Parallel()
	if got := applyResourceGrantGate(tokenStatusAvailable, false); got != tokenStatusNotBound {
		t.Fatalf("L1 without L2: %s", got)
	}
	if got := applyResourceGrantGate(tokenStatusAvailable, true); got != tokenStatusAvailable {
		t.Fatalf("L2+L1: %s", got)
	}
	if got := applyResourceGrantGate(tokenStatusTokenError, true); got != tokenStatusTokenError {
		t.Fatalf("L2 probe fail: %s", got)
	}
}

func TestUpsertAndHasProjectGitOAuthGrant(t *testing.T) {
	setupTestDB(t)
	if err := upsertProjectGitOAuthGrant("t1", "p1", "u1", "GitHub.com", "99"); err != nil {
		t.Fatal(err)
	}
	if !hasProjectGitOAuthGrant("p1", "u1", "github.com") {
		t.Fatal("expected grant")
	}
	if hasProjectGitOAuthGrant("p1", "u2", "github.com") {
		t.Fatal("other user must not inherit")
	}
	if hasProjectGitOAuthGrant("p2", "u1", "github.com") {
		t.Fatal("other project must not inherit")
	}
	remote, ok := lookupProjectGitOAuthGrant("p1", "u1", "github.com")
	if !ok || remote != "99" {
		t.Fatalf("lookup got ok=%v remote=%q", ok, remote)
	}
	if _, ok := lookupProjectGitOAuthGrant("p1", "u2", "github.com"); ok {
		t.Fatal("other user lookup must miss")
	}
	if _, ok := lookupProjectGitOAuthGrant("p2", "u1", "github.com"); ok {
		t.Fatal("other project lookup must miss")
	}
}

func TestHandleInternalLookupProjectGitOAuthGrant(t *testing.T) {
	setupTestDB(t)
	if err := upsertProjectGitOAuthGrant("t1", "p1", "u1", "github.com", "gh-99"); err != nil {
		t.Fatal(err)
	}
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	req := httptest.NewRequest(http.MethodGet, "/api/internal/projects/git-oauth-grant/?project_id=p1&user_id=u1&gitsite=GitHub.com", nil)
	req.Header.Set("X-Internal-Secret", "sec")
	rec := httptest.NewRecorder()
	handleInternalMarkProjectGitOAuthGrant(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["has_grant"] != true {
		t.Fatalf("has_grant=%v", out["has_grant"])
	}
	if strings.TrimSpace(fmt.Sprintf("%v", out["remote_user_id"])) != "gh-99" {
		t.Fatalf("remote_user_id=%v", out["remote_user_id"])
	}

	miss := httptest.NewRequest(http.MethodGet, "/api/internal/projects/git-oauth-grant/?project_id=p1&user_id=u2&gitsite=github.com", nil)
	miss.Header.Set("X-Internal-Secret", "sec")
	missRec := httptest.NewRecorder()
	handleInternalMarkProjectGitOAuthGrant(missRec, miss)
	if missRec.Code != http.StatusOK {
		t.Fatalf("miss status=%d", missRec.Code)
	}
	var missOut map[string]interface{}
	_ = json.Unmarshal(missRec.Body.Bytes(), &missOut)
	if missOut["has_grant"] != false {
		t.Fatalf("other user has_grant=%v", missOut["has_grant"])
	}

	bad := httptest.NewRequest(http.MethodGet, "/api/internal/projects/git-oauth-grant/?project_id=p1", nil)
	bad.Header.Set("X-Internal-Secret", "sec")
	badRec := httptest.NewRecorder()
	handleInternalMarkProjectGitOAuthGrant(badRec, bad)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("missing params status=%d", badRec.Code)
	}

	noSec := httptest.NewRequest(http.MethodGet, "/api/internal/projects/git-oauth-grant/?project_id=p1&user_id=u1&gitsite=github.com", nil)
	noSecRec := httptest.NewRecorder()
	handleInternalMarkProjectGitOAuthGrant(noSecRec, noSec)
	if noSecRec.Code != http.StatusForbidden {
		t.Fatalf("no secret status=%d", noSecRec.Code)
	}
}

func TestMountRoutesGitOAuthGrantGET(t *testing.T) {
	setupTestDB(t)
	if err := upsertProjectGitOAuthGrant("t1", "p1", "u1", "gitlab-tencent-sh-1.daydaymoney.com", "3"); err != nil {
		t.Fatal(err)
	}
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/projects/git-oauth-grant/?project_id=p1&user_id=u1&gitsite=gitlab-tencent-sh-1.daydaymoney.com", nil)
	req.Header.Set("X-Internal-Secret", "sec")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["has_grant"] != true {
		t.Fatalf("has_grant=%v body=%s", out["has_grant"], rec.Body.String())
	}
}

func TestValidateGitRepoForUserSkipsExchangeWithoutProjectGrant(t *testing.T) {
	setupTestDB(t)
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	hits := 0
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		writeJSON(w, 200, map[string]interface{}{"access_token": "ghu_must_not"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	got := validateGitRepoForUser("1001", "https://github.com/acme/demo.git", false, "", "proj-no-grant")
	if got.TokenStatus != tokenStatusNotBound {
		t.Fatalf("token_status=%q want not_bound", got.TokenStatus)
	}
	if hits != 0 {
		t.Fatalf("access-for-user hits=%d want 0", hits)
	}
}
