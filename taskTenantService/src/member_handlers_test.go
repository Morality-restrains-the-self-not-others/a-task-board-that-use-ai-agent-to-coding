package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompanyMembersList(t *testing.T) {
	mux := setupTestService(t)

	// Admin sees full member list with permission
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/company_members/", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	members, _ := out["members"].([]interface{})
	if len(members) < 1 {
		t.Fatalf("expected members, got %v", out)
	}
	meta, _ := out["meta"].(map[string]interface{})
	if meta["has_permission"] != true {
		t.Fatalf("expected has_permission=true for admin, got %v", meta)
	}

	// Non-admin/non-member gets members list with has_permission=false
	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/company_members/", nil), "stranger")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec2.Code, rec.Body.String())
	}
	var out2 map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out2)
	meta2, _ := out2["meta"].(map[string]interface{})
	if meta2["has_permission"] != false {
		t.Fatalf("expected has_permission=false for stranger, got %v", meta2)
	}

	// Unauthenticated user
	req3 := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/company_members/", nil)
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 401 {
		t.Fatalf("expected 401 for unauthenticated, got %d", rec3.Code)
	}
}

func TestUpdateRole(t *testing.T) {
	mux := setupTestService(t)
	// Add a non-admin member
	_, _ = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active) VALUES ('m3','u2','c1',0,1)`)

	// Promote to admin
	body := `{"role":"admin"}`
	req := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m3/update_role/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	msg, _ := out["message"].(string)
	if !strings.Contains(msg, "角色已更新") {
		t.Fatalf("expected success message, got %v", out)
	}
	member, _ := out["member"].(map[string]interface{})
	if member["is_admin"] != true {
		t.Fatalf("expected is_admin=true, got %v", member)
	}

	// Demote to member
	body2 := `{"role":"member"}`
	req2 := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m3/update_role/", bytes.NewBufferString(body2)), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200 for demotion, got %d body=%s", rec2.Code, rec.Body.String())
	}

	// Cannot change creator's role
	body3 := `{"role":"member"}`
	req3 := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m1/update_role/", bytes.NewBufferString(body3)), "admin1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 400 {
		t.Fatalf("expected 400 for creator role change, got %d body=%s", rec3.Code, rec3.Body.String())
	}

	// Non-admin cannot update role
	req4 := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m3/update_role/", bytes.NewBufferString(body)), "u2")
	rec4 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec4, req4)
	if rec4.Code != 403 {
		t.Fatalf("expected 403 for non-admin, got %d", rec4.Code)
	}

	// Update with member_name (creator name update)
	body5 := `{"member_name":"New Creator Name"}`
	req5 := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m1/update_role/", bytes.NewBufferString(body5)), "admin1")
	rec5 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec5, req5)
	if rec5.Code != 200 {
		t.Fatalf("expected 200 for creator name update, got %d body=%s", rec5.Code, rec5.Body.String())
	}
}

func TestToggleStatus(t *testing.T) {
	mux := setupTestService(t)
	_, _ = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active) VALUES ('m4','u3','c1',0,1)`)

	// Deactivate member
	req := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m4/toggle_status/", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["status"] != "inactive" {
		t.Fatalf("expected status=inactive, got %v", out)
	}

	// Reactivate
	req2 := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m4/toggle_status/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var out2 map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out2)
	if out2["status"] != "active" {
		t.Fatalf("expected status=active, got %v", out2)
	}

	// Cannot toggle creator
	req3 := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m1/toggle_status/", nil), "admin1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 400 {
		t.Fatalf("expected 400 for creator toggle, got %d body=%s", rec3.Code, rec3.Body.String())
	}

	// Non-admin cannot toggle
	req4 := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m4/toggle_status/", nil), "u3")
	rec4 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec4, req4)
	if rec4.Code != 403 {
		t.Fatalf("expected 403 for non-admin, got %d", rec4.Code)
	}

	// Non-existent member
	req5 := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/nonexist/toggle_status/", nil), "admin1")
	rec5 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec5, req5)
	if rec5.Code != 404 {
		t.Fatalf("expected 404 for non-existent, got %d", rec5.Code)
	}
}

func TestDeleteMember(t *testing.T) {
	mux := setupTestService(t)
	_, _ = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active) VALUES ('m5','u4','c1',0,1)`)

	// Delete member
	req := withUser(httptest.NewRequest(http.MethodDelete, "/api/tenant/c1/accounts/members/m5/", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Cannot delete creator
	req2 := withUser(httptest.NewRequest(http.MethodDelete, "/api/tenant/c1/accounts/members/m1/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 400 {
		t.Fatalf("expected 400 for creator delete, got %d body=%s", rec2.Code, rec2.Body.String())
	}

	// Non-admin cannot delete
	_, _ = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active) VALUES ('m6','u5','c1',0,1)`)
	req3 := withUser(httptest.NewRequest(http.MethodDelete, "/api/tenant/c1/accounts/members/m6/", nil), "u5")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 403 {
		t.Fatalf("expected 403 for non-admin delete, got %d", rec3.Code)
	}

	// Non-existent member
	req4 := withUser(httptest.NewRequest(http.MethodDelete, "/api/tenant/c1/accounts/members/nonexist/", nil), "admin1")
	rec4 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec4, req4)
	if rec4.Code != 404 {
		t.Fatalf("expected 404 for non-existent, got %d", rec4.Code)
	}
}

func TestMemberRoutesMethodNotAllowed(t *testing.T) {
	mux := setupTestService(t)

	// company_members - only GET allowed
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/company_members/", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405 for POST company_members, got %d", rec.Code)
	}

	// update_role - only PATCH allowed
	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/m1/update_role/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 405 {
		t.Fatalf("expected 405 for GET update_role, got %d", rec2.Code)
	}
}

func TestUpdateRoleWithName(t *testing.T) {
	mux := setupTestService(t)
	_, _ = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active, member_name) VALUES ('m7','u6','c1',0,1,'OldName')`)

	// Update both role and name
	body := `{"role":"admin","member_name":"NewName"}`
	req := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/members/m7/update_role/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Verify name was updated by listing members
	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/company_members/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	var out map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out)
	members, _ := out["members"].([]interface{})
	found := false
	for _, m := range members {
		mm, _ := m.(map[string]interface{})
		if mm["id"] == "m7" && mm["member_name"] == "NewName" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("member name not updated, members=%v", members)
	}
}
