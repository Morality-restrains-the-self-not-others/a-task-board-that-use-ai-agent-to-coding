package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"authz"
	"gatewayauth"
)


// 平台超管（X-User-Roles 含 super_admin，网关 forward-auth 注入）可访问任意租户
// 云资源 —— 与 taskProjectService / 系统管理页授权模型一致。
func TestEnsureTenantMemberAllowsPlatformSuperAdmin(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	startSaasInternalMock(t, store)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t9/installed-images/", nil)
	req.Header.Set(gatewayauth.HeaderAuthUserID, "bootstrap-admin")
	req.Header.Set(authz.HeaderUserRoles, "super_admin")
	rec := httptest.NewRecorder()
	if !ensureTenantMember(rec, req, "t9") {
		t.Fatalf("super_admin denied: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// 非超管平台角色（employee）不享受租户资源绕行 —— 仍需成员校验。
func TestEnsureTenantMemberDeniesPlatformEmployee(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	startSaasInternalMock(t, store)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t9/installed-images/", nil)
	req.Header.Set(gatewayauth.HeaderAuthUserID, "staff")
	req.Header.Set(authz.HeaderUserRoles, "employee")
	rec := httptest.NewRecorder()
	if ensureTenantMember(rec, req, "t9") {
		t.Fatal("employee must not bypass tenant membership")
	}
	if rec.Code != 403 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
