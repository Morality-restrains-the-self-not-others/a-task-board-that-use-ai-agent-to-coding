package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleOidcEndSession_NoRedirectURI_FallsBackToLogin verifies that
// without post_logout_redirect_uri, the handler redirects to the default
// gateway login page (not GitLab sign_out — cookie clearing replaces redirect).
func TestHandleOidcEndSession_NoRedirectURI_FallsBackToLogin(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GitServicePublicBase = "http://127.0.0.1:8012"
	cfg.GatewayPublicBase = "http://127.0.0.1:8003"

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/endsession", nil)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	expected := "http://127.0.0.1:8003/auth/login/"
	if location != expected {
		t.Errorf("expected fallback to gateway login %q, got %q", expected, location)
	}
}

// TestHandleOidcEndSession_WithPostLogoutRedirect verifies that
// post_logout_redirect_uri is used directly as the redirect target
// when it passes allowlist validation.
func TestHandleOidcEndSession_WithPostLogoutRedirect(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GatewayPublicBase = "http://127.0.0.1:4000"

	req := httptest.NewRequest(http.MethodGet,
		"/api/oidc/endsession?post_logout_redirect_uri=http://127.0.0.1:4000/auth/login/", nil)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	expected := "http://127.0.0.1:4000/auth/login/"
	if location != expected {
		t.Errorf("expected direct post_logout_redirect_uri %q, got %q", expected, location)
	}
}

// TestHandleOidcEndSession_RejectsExternalRedirectURI verifies that
// an external/non-allowlisted post_logout_redirect_uri is rejected
// and the handler falls back to the default login page.
func TestHandleOidcEndSession_RejectsExternalRedirectURI(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GatewayPublicBase = "http://127.0.0.1:8003"

	req := httptest.NewRequest(http.MethodGet,
		"/api/oidc/endsession?post_logout_redirect_uri=https://evil.com/phish", nil)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found (even for rejected URI), got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location == "https://evil.com/phish" {
		t.Errorf("open redirect: rejected URI %q was NOT rejected", location)
	}
	expected := "http://127.0.0.1:8003/auth/login/"
	if location != expected {
		t.Errorf("expected fallback to gateway login %q, got %q", expected, location)
	}
}

// TestHandleOidcEndSession_ClearsUserIdCookie verifies that the userId cookie
// is cleared (Set-Cookie with Max-Age=-1, SameSite=Lax).
func TestHandleOidcEndSession_ClearsUserIdCookie(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GitServicePublicBase = "http://127.0.0.1:8012"
	cfg.GatewayPublicBase = "http://127.0.0.1:8003"

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/endsession", nil)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	cookies := resp.Cookies()
	var userIdCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "userId" {
			userIdCookie = c
			break
		}
	}
	if userIdCookie == nil {
		t.Error("expected userId cookie to be set (cleared)")
	} else if userIdCookie.MaxAge != -1 {
		t.Errorf("expected userId cookie MaxAge=-1 (deleted), got %d", userIdCookie.MaxAge)
	}
}

// TestHandleOidcEndSession_ClearsGitlabSessionCookie verifies that the
// _gitlab_session cookie is cleared (Set-Cookie with Max-Age=-1, SameSite=Lax).
func TestHandleOidcEndSession_ClearsGitlabSessionCookie(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GitServicePublicBase = "http://127.0.0.1:8012"
	cfg.GatewayPublicBase = "http://127.0.0.1:8003"

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/endsession", nil)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	cookies := resp.Cookies()
	var gitlabCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_gitlab_session" {
			gitlabCookie = c
			break
		}
	}
	if gitlabCookie == nil {
		t.Error("expected _gitlab_session cookie to be set (cleared)")
	} else if gitlabCookie.MaxAge != -1 {
		t.Errorf("expected _gitlab_session cookie MaxAge=-1 (deleted), got %d", gitlabCookie.MaxAge)
	}
}

// TestHandleOidcEndSession_WithPostLogoutRedirect_Passthrough verifies that
// when a valid post_logout_redirect_uri is provided, the handler passes it
// through directly as the redirect target.
func TestHandleOidcEndSession_WithPostLogoutRedirect_Passthrough(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GatewayPublicBase = "http://127.0.0.1:8003"

	req := httptest.NewRequest(http.MethodGet,
		"/api/oidc/endsession?post_logout_redirect_uri=http://127.0.0.1:8003/dashboard", nil)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "http://127.0.0.1:8003/dashboard" {
		t.Errorf("expected post_logout_redirect_uri passthrough %q, got %q",
			"http://127.0.0.1:8003/dashboard", location)
	}
}

// TestHandleOidcEndSession_NoConfigAtAll_FallsBackToGatewayLogin verifies that
// when no config is set and no post_logout_redirect_uri, the handler redirects
// to a safe relative fallback.
func TestHandleOidcEndSession_NoConfigAtAll_FallsBackToGatewayLogin(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GitServicePublicBase = ""
	cfg.GatewayPublicBase = ""

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/endsession", nil)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	expected := "/auth/login/"
	if location != expected {
		t.Errorf("expected relative fallback %q, got %q", expected, location)
	}
}

// TestHandleOidcEndSession_NoTokenNoError verifies the handler works even
// without any authentication token (unauthenticated end session is OK —
// the browser is already logging out).
func TestHandleOidcEndSession_NoTokenNoError(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GitServicePublicBase = "http://127.0.0.1:8012"
	cfg.GatewayPublicBase = "http://127.0.0.1:8003"

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/endsession", nil)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found even without token, got %d", resp.StatusCode)
	}
}

// TestHandleOidcEndSession_DeletesToken verifies that when a token is present
// (via Authorization header), it gets deleted from the database.
func TestHandleOidcEndSession_DeletesToken(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	cfg.GitServicePublicBase = "http://127.0.0.1:8012"
	cfg.GatewayPublicBase = "http://127.0.0.1:8003"

	token := oidcTestToken(t)

	uid, err := resolveTokenUserID(token)
	if err != nil || uid == "" {
		t.Fatalf("token should resolve before end session: uid=%q err=%v", uid, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/endsession", nil)
	req.Header.Set("Authorization", "Token "+token)
	w := httptest.NewRecorder()

	handleOidcEndSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", resp.StatusCode)
	}

	uid, err = resolveTokenUserID(token)
	if err == nil {
		t.Errorf("token should be deleted after end session, but still resolves to uid=%q", uid)
	}
}
