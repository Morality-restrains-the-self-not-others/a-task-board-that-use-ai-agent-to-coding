package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gatewayauth"
)

func TestResolveUserMemberUsesDjangoHTTP(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t-local", "u-local", "m-local", true)
	startSaasInternalMock(t, store)

	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	mid, isAdmin, err := resolveUserMember(r, "t-local", "u-local")
	if err != nil {
		t.Fatalf("resolveUserMember err=%v", err)
	}
	if mid != "m-local" || !isAdmin {
		t.Fatalf("got mid=%q isAdmin=%v", mid, isAdmin)
	}
}

func TestEnsureTenantMemberAllowsDjangoMember(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t2", "u2", "m2", false)
	startSaasInternalMock(t, store)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t2/installed-images/", nil)
	req.Header.Set(gatewayauth.HeaderAuthUserID, "u2")
	rec := httptest.NewRecorder()
	if !ensureTenantMember(rec, req, "t2") {
		t.Fatalf("ensureTenantMember denied: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEnsureTenantMemberDeniesUnknownUser(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t3/installed-images/", nil)
	req.Header.Set(gatewayauth.HeaderAuthUserID, "nobody")
	rec := httptest.NewRecorder()
	if !ensureTenantMember(rec, req, "t3") {
		t.Fatalf("no saas mock should skip membership gate, got %d %s", rec.Code, rec.Body.String())
	}

	store := newSaasHTTPStore()
	startSaasInternalMock(t, store)

	req = httptest.NewRequest(http.MethodGet, "/api/tenant/t3/installed-images/", nil)
	req.Header.Set(gatewayauth.HeaderAuthUserID, "nobody")
	rec = httptest.NewRecorder()
	if ensureTenantMember(rec, req, "t3") {
		t.Fatal("expected deny for unknown user")
	}
	if rec.Code != 403 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["message"] != "无权访问该租户资源" {
		t.Fatalf("body=%v", body)
	}
}
