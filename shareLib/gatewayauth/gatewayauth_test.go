package gatewayauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserFromGatewayHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/todos/", nil)
	req.Header.Set(HeaderGatewayVerified, "1")
	req.Header.Set(HeaderGatewaySecret, "gw-secret")
	req.Header.Set(HeaderUserID, "user-1")

	if got := UserFromGatewayHeaders(req, "gw-secret"); got != "user-1" {
		t.Fatalf("expected user-1, got %q", got)
	}
	req.Header.Set(HeaderGatewaySecret, "wrong")
	if got := UserFromGatewayHeaders(req, "gw-secret"); got != "" {
		t.Fatalf("expected empty for wrong secret, got %q", got)
	}
	req.Header.Del(HeaderGatewaySecret)
	if got := UserFromGatewayHeaders(req, "gw-secret"); got != "" {
		t.Fatalf("expected empty when APISIX omits secret header, got %q", got)
	}
}

func TestBearerOrTokenFromAuthHeader(t *testing.T) {
	if got := BearerOrTokenFromAuthHeader("Token abc"); got != "abc" {
		t.Fatalf("Token prefix: got %q", got)
	}
	if got := BearerOrTokenFromAuthHeader("Bearer xyz"); got != "xyz" {
		t.Fatalf("Bearer prefix: got %q", got)
	}
}

func TestVerifyWithTaskAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/verify" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Token test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user_id":"u1","tenant_id":"t1"}`))
	}))
	defer srv.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/todos/", nil)
	req.Header.Set("Authorization", "Token test-key")
	userID, tenantID, err := VerifyWithTaskAuth(req, srv.URL)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if userID != "u1" || tenantID != "t1" {
		t.Fatalf("got user=%s tenant=%s", userID, tenantID)
	}
}
