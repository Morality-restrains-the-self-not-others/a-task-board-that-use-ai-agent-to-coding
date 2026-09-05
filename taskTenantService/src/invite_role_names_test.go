package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseInviteRoleNames_OK(t *testing.T) {
	body := map[string]interface{}{
		"role_names": []interface{}{"部署工程师", " 只读 ", "部署工程师"},
	}
	names, err := parseInviteRoleNames(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 deduped names, got %d: %v", len(names), names)
	}
	if names[0] != "部署工程师" || names[1] != "只读" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestParseInviteRoleNames_Missing(t *testing.T) {
	names, err := parseInviteRoleNames(map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Fatalf("expected no names, got %v", names)
	}
}

func TestParseInviteRoleNames_Invalid(t *testing.T) {
	if _, err := parseInviteRoleNames(map[string]interface{}{"role_names": "x"}); err == nil {
		t.Fatal("expected error for non-array role_names")
	}
	if _, err := parseInviteRoleNames(map[string]interface{}{"role_names": []interface{}{"ok", ""}}); err == nil {
		t.Fatal("expected error for empty role_names element")
	}
	if _, err := parseInviteRoleNames(map[string]interface{}{"role_names": []interface{}{"ok", 42}}); err == nil {
		t.Fatal("expected error for non-string role_names element")
	}
}

func TestInviteWithRoleNamesStoresJSON(t *testing.T) {
	mux := setupTestService(t)
	ensurePendingRoleNamesColumn(t)

	old := roleExistsInAuthFn
	roleExistsInAuthFn = func(companyID, roleName string) (bool, error) { return true, nil }
	t.Cleanup(func() { roleExistsInAuthFn = old })

	body := `{"invite_method":"link","company_member_name":"RoleGrantee","role":"member","workspace_id":"ws1","expiration_days":7,"role_names":["部署工程师","只读"]}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var raw string
	err := db.QueryRow(`SELECT COALESCE(CAST(pending_role_names AS CHAR),'') FROM tenant_invitation WHERE company_id='c1' AND company_member_name='RoleGrantee' ORDER BY created_at DESC LIMIT 1`).Scan(&raw)
	if err != nil {
		t.Fatalf("query pending_role_names: %v", err)
	}
	if !strings.Contains(raw, "部署工程师") || !strings.Contains(raw, "只读") {
		t.Fatalf("expected pending_role_names to contain both roles, got %q", raw)
	}
}

func TestInviteWithRoleNamesUnknownRole(t *testing.T) {
	mux := setupTestService(t)
	ensurePendingRoleNamesColumn(t)

	old := roleExistsInAuthFn
	roleExistsInAuthFn = func(companyID, roleName string) (bool, error) { return false, nil }
	t.Cleanup(func() { roleExistsInAuthFn = old })

	body := `{"invite_method":"link","company_member_name":"BadRole","role":"member","workspace_id":"ws1","expiration_days":7,"role_names":["不存在的角色"]}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for unknown role, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInviteAdminIgnoresRoleNames(t *testing.T) {
	mux := setupTestService(t)
	ensurePendingRoleNamesColumn(t)

	body := `{"invite_method":"link","company_member_name":"AdminRoleInvitee","role":"admin","workspace_id":"ws1","expiration_days":7,"role_names":["部署工程师"]}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var raw string
	err := db.QueryRow(`SELECT COALESCE(CAST(pending_role_names AS CHAR),'') FROM tenant_invitation WHERE company_id='c1' AND company_member_name='AdminRoleInvitee' ORDER BY created_at DESC LIMIT 1`).Scan(&raw)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if raw != "" && raw != "null" {
		t.Fatalf("admin invite should ignore role_names, got %q", raw)
	}
}

func TestJoinAppliesPendingRoleNames(t *testing.T) {
	mux := setupTestService(t)
	ensurePendingRoleNamesColumn(t)

	old := roleExistsInAuthFn
	roleExistsInAuthFn = func(companyID, roleName string) (bool, error) { return true, nil }
	t.Cleanup(func() { roleExistsInAuthFn = old })

	body := `{"invite_method":"link","company_member_name":"JoinRoleGrantee","role":"member","workspace_id":"ws1","expiration_days":7,"role_names":["部署工程师"]}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("invite expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var inviteOut map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &inviteOut)
	token, _ := inviteOut["invite_token"].(string)
	if token == "" {
		t.Fatalf("no token: %s", rec.Body.String())
	}

	joinBody := `{"token":"` + token + `"}`
	req3 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), "joinrolgrantee1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 201 {
		t.Fatalf("join expected 201, got %d body=%s", rec3.Code, rec3.Body.String())
	}
	var roleName string
	err := db.QueryRow(`SELECT role_name FROM tenant_member_role WHERE company_id='c1' AND member_id=(
		SELECT id FROM tenant_company_member WHERE user_id='joinrolgrantee1' AND company_id='c1' LIMIT 1)`).Scan(&roleName)
	if err != nil {
		t.Fatalf("member role: %v", err)
	}
	if roleName != "部署工程师" {
		t.Fatalf("expected bound role 部署工程师, got %q", roleName)
	}
}
