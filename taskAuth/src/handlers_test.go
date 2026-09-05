package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	rec := httptest.NewRecorder()
	handleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// Go 1.22+ ServeMux rejects a method-less more-specific path under a
// method-restricted wildcard subtree (e.g. GET /users/{id}/ vs /users/me/...).
func TestMountRoutesDoesNotPanic(t *testing.T) {
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("mountRoutes panicked: %v", rec)
		}
	}()
	mountRoutes(http.NewServeMux())
}

func TestAccountDeletionRouteNotCapturedByGetUser(t *testing.T) {
	mux := http.NewServeMux()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("mountRoutes panicked: %v", rec)
		}
	}()
	mountRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/me/account-deletion/precheck", nil)
	_, pattern := mux.Handler(req)
	if pattern != "GET /api/accounts/users/me/account-deletion/" {
		t.Fatalf("account-deletion precheck matched %q, want deletion GET subtree", pattern)
	}

	postReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/me/account-deletion/request", nil)
	_, postPattern := mux.Handler(postReq)
	if postPattern != "POST /api/accounts/users/me/account-deletion/" {
		t.Fatalf("account-deletion request matched %q, want deletion POST subtree", postPattern)
	}

	userReq := httptest.NewRequest(http.MethodGet, "/api/accounts/users/123/", nil)
	_, userPattern := mux.Handler(userReq)
	if userPattern != "GET /api/accounts/users/{user_id}/" {
		t.Fatalf("get-user matched %q, want GET /users/{user_id}/", userPattern)
	}
}

func TestInboxRoutePatterns(t *testing.T) {
	mux := http.NewServeMux()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("mountRoutes panicked: %v", rec)
		}
	}()
	mountRoutes(mux)

	inboxReq := httptest.NewRequest(http.MethodGet, "/api/auth/inbox/", nil)
	_, inboxPattern := mux.Handler(inboxReq)
	if inboxPattern != "GET /api/auth/inbox/" {
		t.Fatalf("inbox list matched %q, want GET /api/auth/inbox/", inboxPattern)
	}

	readReq := httptest.NewRequest(http.MethodPost, "/api/auth/inbox/123/read/", nil)
	_, readPattern := mux.Handler(readReq)
	if readPattern != "POST /api/auth/inbox/{id}/read/" {
		t.Fatalf("inbox read matched %q, want POST /api/auth/inbox/{id}/read/", readPattern)
	}
}

func TestProfileBindPhoneRouteNotSwallowedByUpsertProfile(t *testing.T) {
	mux := http.NewServeMux()
	mountRoutes(mux)

	bindReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/bind-phone/", nil)
	_, bindPattern := mux.Handler(bindReq)
	if bindPattern != "POST /api/accounts/users/profile/bind-phone/" {
		t.Fatalf("bind-phone matched %q, want dedicated bind-phone pattern (not profile upsert subtree)", bindPattern)
	}

	replaceReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/replace-phone/", nil)
	_, replacePattern := mux.Handler(replaceReq)
	if replacePattern != "POST /api/accounts/users/profile/replace-phone/" {
		t.Fatalf("replace-phone matched %q, want dedicated replace-phone pattern", replacePattern)
	}

	upsertReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/", nil)
	_, upsertPattern := mux.Handler(upsertReq)
	if upsertPattern != "POST /api/accounts/users/profile/{$}" && upsertPattern != "POST /api/accounts/users/profile/" {
		t.Fatalf("profile POST matched %q, want exact profile upsert", upsertPattern)
	}
}

func TestStrField(t *testing.T) {
	body := map[string]interface{}{"email": "a@b.com", "n": float64(1)}
	if got := strField(body, "email"); got != "a@b.com" {
		t.Fatalf("email: %q", got)
	}
	if got := strField(body, "missing"); got != "" {
		t.Fatalf("missing: %q", got)
	}
}
