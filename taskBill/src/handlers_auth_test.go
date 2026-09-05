package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func resetCfgForAuthTest(t *testing.T, internalSecret, gatewaySecret string) {
	t.Helper()
	origInternal := cfg.InternalSecret
	origGateway := cfg.GatewayInternalSecret
	cfg.InternalSecret = internalSecret
	cfg.GatewayInternalSecret = gatewaySecret
	t.Cleanup(func() {
		cfg.InternalSecret = origInternal
		cfg.GatewayInternalSecret = origGateway
	})
}

// ── requireInternalSecret ──

func TestRequireInternalSecret_NoSecretsConfigured(t *testing.T) {
	resetCfgForAuthTest(t, "", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	if !requireInternalSecret(req) {
		t.Fatal("expected pass when no secrets are configured")
	}
}

func TestRequireInternalSecret_TaskBillSecretMatch(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-TaskBill-Internal-Secret", "bill-secret")
	if !requireInternalSecret(req) {
		t.Fatal("expected pass when X-TaskBill-Internal-Secret matches")
	}
}

func TestRequireInternalSecret_TaskBillSecretMismatch(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-TaskBill-Internal-Secret", "wrong-secret")
	if requireInternalSecret(req) {
		t.Fatal("expected fail when X-TaskBill-Internal-Secret does not match")
	}
}

func TestRequireInternalSecret_GatewaySecretMatch(t *testing.T) {
	resetCfgForAuthTest(t, "", "gw-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-TaskGateway-Internal-Secret", "gw-secret")
	if !requireInternalSecret(req) {
		t.Fatal("expected pass when X-TaskGateway-Internal-Secret matches")
	}
}

func TestRequireInternalSecret_GatewaySecretMismatch(t *testing.T) {
	resetCfgForAuthTest(t, "", "gw-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-TaskGateway-Internal-Secret", "wrong-gw-secret")
	if requireInternalSecret(req) {
		t.Fatal("expected fail when X-TaskGateway-Internal-Secret does not match")
	}
}

func TestRequireInternalSecret_BothSecretsEitherWorks(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "gw-secret")

	// Bill secret match
	req1 := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req1.Header.Set("X-TaskBill-Internal-Secret", "bill-secret")
	if !requireInternalSecret(req1) {
		t.Fatal("expected pass with matching X-TaskBill-Internal-Secret")
	}

	// Gateway secret match
	req2 := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req2.Header.Set("X-TaskGateway-Internal-Secret", "gw-secret")
	if !requireInternalSecret(req2) {
		t.Fatal("expected pass with matching X-TaskGateway-Internal-Secret")
	}

	// Neither matches
	req3 := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	if requireInternalSecret(req3) {
		t.Fatal("expected fail when no matching headers")
	}
}

func TestRequireInternalSecret_NoHeaderWithSecretConfigured(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	if requireInternalSecret(req) {
		t.Fatal("expected fail when secret is configured but no header is present")
	}
}

// ── requireInternalOrGatewayAuth ──

func TestRequireInternalOrGatewayAuth_InternalSecretPass(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-TaskBill-Internal-Secret", "bill-secret")
	rec := httptest.NewRecorder()
	if !requireInternalOrGatewayAuth(rec, req) {
		t.Fatalf("expected pass with internal secret, got body=%s", rec.Body.String())
	}
}

func TestRequireInternalOrGatewayAuth_GatewaySecretPass(t *testing.T) {
	resetCfgForAuthTest(t, "", "gw-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-TaskGateway-Internal-Secret", "gw-secret")
	rec := httptest.NewRecorder()
	if !requireInternalOrGatewayAuth(rec, req) {
		t.Fatalf("expected pass with gateway secret, got body=%s", rec.Body.String())
	}
}

func TestRequireInternalOrGatewayAuth_GatewayAuthSuperuserPass(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	// v63: 平台角色经 X-User-Roles 注入（super_admin/employee）
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	if !requireInternalOrGatewayAuth(rec, req) {
		t.Fatalf("expected pass with superuser auth, got body=%s", rec.Body.String())
	}
}

func TestRequireInternalOrGatewayAuth_GatewayAuthStaffPass(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "employee")
	rec := httptest.NewRecorder()
	if !requireInternalOrGatewayAuth(rec, req) {
		t.Fatalf("expected pass with staff auth, got body=%s", rec.Body.String())
	}
}

func TestRequireInternalOrGatewayAuth_NotAdminReturns403(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	// Neither superuser nor staff
	rec := httptest.NewRecorder()
	if requireInternalOrGatewayAuth(rec, req) {
		t.Fatal("expected fail when verified but not admin")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected response body")
	}
}

func TestRequireInternalOrGatewayAuth_NoAuthReturns403(t *testing.T) {
	resetCfgForAuthTest(t, "bill-secret", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	// No auth headers at all
	rec := httptest.NewRecorder()
	if requireInternalOrGatewayAuth(rec, req) {
		t.Fatal("expected fail when no auth headers")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

// ── requireInternalOrGatewayAuth: gateway secret fallback ──
// When X-TaskGateway-Internal-Secret passes requireInternalSecret,
// requireInternalOrGatewayAuth should pass (delegating to requireInternalSecret).

func TestRequireInternalOrGatewayAuth_GatewaySecretFallbackPass(t *testing.T) {
	resetCfgForAuthTest(t, "", "gw-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)
	req.Header.Set("X-TaskGateway-Internal-Secret", "gw-secret")
	rec := httptest.NewRecorder()
	if !requireInternalOrGatewayAuth(rec, req) {
		t.Fatalf("expected pass via gateway secret fallback, got body=%s", rec.Body.String())
	}
}
