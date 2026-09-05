package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSOVendorStatusSkipsForwardAuthWithoutCookie(t *testing.T) {
	called := false
	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(authSrv.Close)

	app := testApp(t)
	app.Cfg.TaskAuthBaseURL = authSrv.URL
	app.Cfg.TaskAuthInternalSecret = "test-secret"
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rec := vendorStatusRequest(app, mux, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without credentials, got %d body=%s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("forward-auth must not be called when request has no session cookie")
	}
}

func TestSSOVendorStatusFromSessionCookie(t *testing.T) {
	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/gateway/forward-auth/" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-TaskAuth-Internal-Secret") != "test-secret" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if !strings.Contains(r.Header.Get("Cookie"), "token=sess-cookie") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-User-Id", "88001")
		w.Header().Set("X-User-Email", "cookie-user@example.com")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(authSrv.Close)

	app := testApp(t)
	app.Cfg.TaskAuthBaseURL = authSrv.URL
	app.Cfg.TaskAuthInternalSecret = "test-secret"
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(88001), "cookie-user@example.com", true)

	req := httptest.NewRequest(http.MethodGet, "/api/ai-provider/vendor-status/", nil)
	req.Header.Set("Cookie", "token=sess-cookie")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from cookie session, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeVendorStatus(t, rec)
	if body["status"] != "qualified" || body["is_vendor"] != true {
		t.Fatalf("expected qualified vendor from cookie, got %v", body)
	}
}

func TestHasMainSiteSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if hasMainSiteSessionCookie(req) {
		t.Fatal("empty request should have no session cookie")
	}
	req.AddCookie(&http.Cookie{Name: "token", Value: "abc"})
	if !hasMainSiteSessionCookie(req) {
		t.Fatal("token cookie should count as session")
	}
}

func TestApplicantHasBindableEmail(t *testing.T) {
	if applicantHasBindableEmail("") {
		t.Fatal("empty email is not bindable")
	}
	if applicantHasBindableEmail("sso-1@sso.invalid") {
		t.Fatal("synthetic email is not bindable")
	}
	if !applicantHasBindableEmail("vendor@example.com") {
		t.Fatal("real email should be bindable")
	}
}
