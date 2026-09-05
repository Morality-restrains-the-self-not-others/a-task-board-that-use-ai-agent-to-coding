package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"taskGitOauth/infrastructure"
)

func TestGetCredentialByIDForUpdateBlocksUntilCommit(t *testing.T) {
	app := testApp(t)
	const userID = "lock-user"
	insertActiveCredential(t, app, "gitlab:tencent-sh-1", userID, "cipher")

	var id int64
	if err := app.DB.QueryRow(
		`SELECT id FROM git_oauth_appusercredential WHERE task2app_user_id=?`,
		userID,
	).Scan(&id); err != nil {
		t.Fatal(err)
	}

	tx1, err := infrastructure.BeginImmediate(app.DB.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx1.Rollback() }()
	if _, err := app.DB.GetCredentialByIDForUpdate(tx1, id); err != nil {
		t.Fatalf("first lock: %v", err)
	}

	started := make(chan struct{})
	gotSecond := make(chan error, 1)
	go func() {
		tx2, err := infrastructure.BeginImmediate(app.DB.DB)
		if err != nil {
			gotSecond <- err
			return
		}
		defer func() { _ = tx2.Rollback() }()
		close(started)
		_, err = app.DB.GetCredentialByIDForUpdate(tx2, id)
		gotSecond <- err
		if err == nil {
			_ = tx2.Commit()
		}
	}()

	select {
	case <-started:
	case err := <-gotSecond:
		t.Fatalf("second transaction failed to start: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("second Begin timed out")
	}

	select {
	case err := <-gotSecond:
		t.Fatalf("FOR UPDATE must block while first tx holds the row, got err=%v", err)
	case <-time.After(200 * time.Millisecond):
	}

	if err := tx1.Commit(); err != nil {
		t.Fatalf("commit first: %v", err)
	}
	select {
	case err := <-gotSecond:
		if err != nil {
			t.Fatalf("second lock after commit: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second FOR UPDATE did not unblock after commit")
	}
}

func TestConcurrentProbeAndAccessForUserRefreshOnce(t *testing.T) {
	app := providerTestApp(t)
	const userID = "877397583960502272"
	cipher, err := app.Fernet.Encrypt("glrt_shared")
	if err != nil {
		t.Fatal(err)
	}
	insertActiveCredential(t, app, "gitlab:gitlab-local", userID, cipher)

	var refreshCalls atomic.Int32
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		refreshCalls.Add(1)
		time.Sleep(150 * time.Millisecond)
		return map[string]any{
			"access_token":  "glpat-once",
			"refresh_token": "glrt_rotated",
			"expires_in":    3600,
		}, nil
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		code, _, raw := getUserAppConnection(t, app,
			"/api/git-oauth/user-app-connection/?repo_url=http%3A%2F%2Flocalhost%3A8012%2Fgroup%2Frepo&probe_access_token=1",
			userID)
		if code != 200 {
			t.Errorf("probe status=%d body=%s", code, raw)
		}
	}()
	go func() {
		defer wg.Done()
		body := `{"user_id":"` + userID + `","provider_key":"gitlab:gitlab-local"}`
		req := httptest.NewRequest(http.MethodPost, "/api/internal/gitlab/oauth/access-for-user/", bytes.NewBufferString(body))
		req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("access-for-user status=%d body=%s", rec.Code, rec.Body.String())
		}
	}()
	wg.Wait()

	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("serialized GitLab refresh must run once, calls=%d", got)
	}
}

func TestIssueAccessTokenGitLabRefreshRequiresRotatedRefresh(t *testing.T) {
	app := providerTestApp(t)
	const userID = "877397583960502273"
	cipher, err := app.Fernet.Encrypt("glrt_old")
	if err != nil {
		t.Fatal(err)
	}
	insertActiveCredential(t, app, "gitlab:gitlab-local", userID, cipher)
	app.RefreshAccessTokenFn = func(providerKey, refreshPlain string) (map[string]any, error) {
		return map[string]any{"access_token": "glpat-only", "expires_in": 3600}, nil
	}
	row, err := app.DB.FindActiveCredential("gitlab:gitlab-local", userID, "")
	if err != nil || row == nil {
		t.Fatalf("cred: %v %v", err, row)
	}
	_, err = app.issueAccessTokenFromCredential(userID, row)
	if err == nil || !strings.Contains(err.Error(), "missing refresh_token") {
		t.Fatalf("want missing refresh_token, got %v", err)
	}
}
