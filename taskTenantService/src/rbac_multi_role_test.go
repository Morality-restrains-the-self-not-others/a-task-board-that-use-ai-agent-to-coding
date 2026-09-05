package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseRoleNamesBody_LegacyAndMulti(t *testing.T) {
	t.Parallel()
	mk := func(raw string) *http.Request {
		return httptest.NewRequest(http.MethodPut, "/", strings.NewReader(raw))
	}
	names, err := parseRoleNamesBody(mk(`{"role_name":"member"}`))
	if err != nil || len(names) != 1 || names[0] != "member" {
		t.Fatalf("legacy: got %#v err=%v", names, err)
	}
	names, err = parseRoleNamesBody(mk(`{"role_names":["custom_a","custom_b","custom_a"]}`))
	if err != nil || len(names) != 2 {
		t.Fatalf("multi dedupe: got %#v err=%v", names, err)
	}
	names, err = parseRoleNamesBody(mk(`{"role_names":["a"],"role_name":"b"}`))
	if err != nil || len(names) != 2 {
		t.Fatalf("merge: got %#v err=%v", names, err)
	}
	names, err = parseRoleNamesBody(mk(`{"role_names":[]}`))
	if err != nil || len(names) != 0 {
		t.Fatalf("empty replace-all: got %#v err=%v", names, err)
	}
}

func TestMemberRoleReplaceAllAndDelete(t *testing.T) {
	mux := setupTestService(t)
	roleExistsInAuthFn = func(companyID, roleName string) (bool, error) {
		return roleName == "member" || roleName == "custom_fin" || roleName == "custom_ops", nil
	}
	t.Cleanup(func() { roleExistsInAuthFn = roleExistsInAuth })

	_, err := db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m2','u2','c1',0,1,'ws1','Bob')`)
	if err != nil {
		t.Fatalf("seed member: %v", err)
	}

	body := `{"role_names":["member","custom_fin"]}`
	req := withUser(httptest.NewRequest(http.MethodPut, "/api/tenant/member-role/company_id/c1/member_id/m2/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("replace-all status=%d body=%s", rec.Code, rec.Body.String())
	}
	var putResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &putResp)
	gotNames, _ := putResp["role_names"].([]any)
	if len(gotNames) != 2 {
		t.Fatalf("expected 2 role_names, got %#v", putResp)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tenant_member_role WHERE member_id='m2' AND company_id='c1'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("expected 2 rows, n=%d err=%v", n, err)
	}

	// replace with single role
	req2 := withUser(httptest.NewRequest(http.MethodPut, "/api/tenant/member-role/company_id/c1/member_id/m2/",
		bytes.NewBufferString(`{"role_names":["custom_ops"]}`)), "admin1")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("replace single status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM tenant_member_role WHERE member_id='m2'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("after replace expected 1 row, n=%d err=%v", n, err)
	}

	// DELETE single
	req3 := withUser(httptest.NewRequest(http.MethodDelete, "/api/tenant/member-role/company_id/c1/member_id/m2/role_name/custom_ops/", nil), "admin1")
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec3.Code, rec3.Body.String())
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM tenant_member_role WHERE member_id='m2'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("after delete expected 0 rows, n=%d err=%v", n, err)
	}

	// list endpoint (3-part path)
	req4 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/member-role/company_id/c1/", nil), "admin1")
	rec4 := httptest.NewRecorder()
	mux.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec4.Code, rec4.Body.String())
	}
}

func TestMemberRoleLegacyRoleNameStillWorks(t *testing.T) {
	mux := setupTestService(t)
	roleExistsInAuthFn = func(companyID, roleName string) (bool, error) {
		return roleName == "member", nil
	}
	t.Cleanup(func() { roleExistsInAuthFn = roleExistsInAuth })

	_, err := db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m3','u3','c1',0,1,'ws1','Carol')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	req := withUser(httptest.NewRequest(http.MethodPut, "/api/tenant/member-role/company_id/c1/member_id/m3/",
		bytes.NewBufferString(`{"role_name":"member"}`)), "admin1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy put status=%d body=%s", rec.Code, rec.Body.String())
	}
	var role string
	if err := db.QueryRow(`SELECT role_name FROM tenant_member_role WHERE member_id='m3'`).Scan(&role); err != nil || role != "member" {
		t.Fatalf("role=%q err=%v", role, err)
	}
}
