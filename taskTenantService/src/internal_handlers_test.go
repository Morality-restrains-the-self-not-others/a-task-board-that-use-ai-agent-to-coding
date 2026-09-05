package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalMembersResolve(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/resolve?user_id=admin1&company_id=c1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["user_id"] != "admin1" {
		t.Fatalf("expected user_id=admin1, got %v", out)
	}
}

func TestInternalMembersResolveNotFound(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/resolve?user_id=nonexist&company_id=c1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestInternalMembersExists(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/exists?user_id=admin1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["exists"] != true {
		t.Fatalf("expected exists=true, got %v", out)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/exists?user_id=nonexist", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	var out2 map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out2)
	if out2["exists"] != false {
		t.Fatalf("expected exists=false, got %v", out2)
	}
}

func TestInternalMembersExistsMissingUser(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/exists", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestInternalMembersByID(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/by-id?member_id=m1&company_id=c1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["id"] != "m1" {
		t.Fatalf("expected id=m1, got %v", out)
	}
}

func TestInternalMembersWorkspaceUpdate(t *testing.T) {
	mux := setupTestService(t)

	body := `{"user_id":"admin1","company_id":"c1","workspace_id":"ws2"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/tenant/members/workspace", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != true || out["workspace_id"] != "ws2" {
		t.Fatalf("expected ok=true workspace=ws2, got %v", out)
	}
}

func TestInternalMembersUpdate(t *testing.T) {
	mux := setupTestService(t)

	body := `{"user_id":"admin1","company_id":"c1","workspace_id":"ws3","member_name":"Updated Admin"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/tenant/members/update", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["member_name"] != "Updated Admin" {
		t.Fatalf("expected member_name='Updated Admin', got %v", out)
	}
}

func TestInternalMembersCreate(t *testing.T) {
	mux := setupTestService(t)

	body := `{"user_id":"newuser","company_id":"c1","is_admin":false,"workspace_id":"ws1","member_name":"New User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/members", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalMembersListByCompany(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members?company_id=c1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) < 1 {
		t.Fatalf("expected members, got %v", string(rec.Body.Bytes()))
	}
}

func TestInternalMembersListByUser(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members?user_id=admin1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) < 1 {
		t.Fatalf("expected members for admin1, got %v", string(rec.Body.Bytes()))
	}
}

func TestInternalMembersImport(t *testing.T) {
	mux := setupTestService(t)

	body := `{"items":[{"user_id":"imp1","company_id":"c1","is_admin":false,"workspace_id":"ws1","member_name":"Imp1"},{"user_id":"imp2","company_id":"c1","is_admin":true,"workspace_id":"ws1","member_name":"Imp2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/members/import", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	imp, _ := out["imported"].(float64)
	if int(imp) != 2 {
		t.Fatalf("expected imported=2, got %v", out)
	}
}

func TestInternalGroupsUserInGroup(t *testing.T) {
	mux := setupTestService(t)
	// Create group and add member
	body := `{"name":"G1","description":"test"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	var g map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	gid, _ := g["id"].(string)

	add := `{"user_id":"admin1"}`
	req2 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/"+gid+"/add_member/", bytes.NewBufferString(add)), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)

	// Check user-in-group
	req3 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/groups/user-in-group?user_id=admin1&group_id="+gid, nil)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec3.Code, rec3.Body.String())
	}
	var ug map[string]interface{}
	_ = json.Unmarshal(rec3.Body.Bytes(), &ug)
	if ug["in_group"] != true {
		t.Fatalf("expected in_group=true, got %v", ug)
	}

	// Check user not in group
	req4 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/groups/user-in-group?user_id=nonexist&group_id="+gid, nil)
	rec4 := httptest.NewRecorder()
	mux.ServeHTTP(rec4, req4)
	var ug4 map[string]interface{}
	_ = json.Unmarshal(rec4.Body.Bytes(), &ug4)
	if ug4["in_group"] != false {
		t.Fatalf("expected in_group=false, got %v", ug4)
	}
}

func TestInternalGroupsMembers(t *testing.T) {
	mux := setupTestService(t)
	body := `{"name":"G2","description":"test"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	var g map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	gid, _ := g["id"].(string)

	// add member
	add := `{"user_id":"admin1"}`
	req2 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/"+gid+"/add_member/", bytes.NewBufferString(add)), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)

	req3 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/groups/members?group_id="+gid, nil)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec3.Code, rec3.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec3.Body.Bytes(), &out)
	ids, _ := out["user_ids"].([]interface{})
	found := false
	for _, id := range ids {
		if id.(string) == "admin1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected admin1 in group members, got %v", out)
	}
}

func TestInternalGroupsByID(t *testing.T) {
	mux := setupTestService(t)
	body := `{"name":"G3","description":"desc3"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	var g map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	gid, _ := g["id"].(string)

	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/groups/by-id?group_id="+gid, nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out)
	if out["name"] != "G3" {
		t.Fatalf("expected name=G3, got %v", out)
	}

	// Non-existent group
	req3 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/groups/by-id?group_id=nonexist", nil)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != 404 {
		t.Fatalf("expected 404 for non-existent group, got %d", rec3.Code)
	}
}

func TestInternalInvitationsImport(t *testing.T) {
	mux := setupTestService(t)

	body := `{"items":[{"company_id":"c1","is_admin":false,"workspace_id":"ws1","invite_method":"link","invite_target":"","company_member_name":"Inv1"},{"company_id":"c1","is_admin":true,"workspace_id":"ws1","invite_method":"email","invite_target":"e@test.com","company_member_name":"Inv2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/invitations/import", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	imp, _ := out["imported"].(float64)
	if int(imp) != 2 {
		t.Fatalf("expected imported=2, got %v", out)
	}
}

func TestInternalGroupsImport(t *testing.T) {
	mux := setupTestService(t)

	body := `{"groups":[{"id":"g-imp-1","name":"Group A","description":"desc","company_id":"c1","created_by_id":"admin1"}],"members":[{"group_id":"g-imp-1","user_id":"admin1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/groups/import", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	g, _ := out["groups"].(float64)
	m, _ := out["members"].(float64)
	if int(g) != 1 || int(m) != 1 {
		t.Fatalf("expected groups=1 members=1, got %v", out)
	}
}

func TestInternalCompaniesUpsert(t *testing.T) {
	mux := setupTestService(t)

	body := `{"id":"c-new","name":"New Company","creator_id":"creator1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/companies/upsert", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["name"] != "New Company" {
		t.Fatalf("expected name='New Company', got %v", out)
	}

	// Verify via by-id
	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/companies/by-id?company_id=c-new", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200 for by-id, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestInternalCompaniesBatch(t *testing.T) {
	mux := setupTestService(t)
	// c1 already exists, upsert a new one too
	_, _ = db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES ('c-b1','Batch1','cr1')`)
	_, _ = db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES ('c-b2','Batch2','cr1')`)

	body := `{"ids":["c-b1","c-b2","c1"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/companies/batch", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var list []interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 3 {
		t.Fatalf("expected 3 companies, got %d: %s", len(list), rec.Body.String())
	}
}

func TestInternalCompaniesNameTaken(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/companies/name-taken?name=Test+Co", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["taken"] != true {
		t.Fatalf("expected taken=true for existing name, got %v", out)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/companies/name-taken?name=UniqueName123", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	var out2 map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out2)
	if out2["taken"] != false {
		t.Fatalf("expected taken=false for unique name, got %v", out2)
	}

	// With exclude_id
	req3 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/companies/name-taken?name=Test+Co&exclude_id=c1", nil)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	var out3 map[string]interface{}
	_ = json.Unmarshal(rec3.Body.Bytes(), &out3)
	if out3["taken"] != false {
		t.Fatalf("expected taken=false when excluding self c1, got %v", out3)
	}
}

func TestInternalCompaniesImport(t *testing.T) {
	mux := setupTestService(t)

	body := `{"items":[{"id":"c-imp-1","name":"Import Co","creator_id":"cr1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/companies/import", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	imp, _ := out["imported"].(float64)
	if int(imp) < 1 {
		t.Fatalf("expected imported>=1, got %v", out)
	}
}

func TestInternalGroupsUserInAdminGroups(t *testing.T) {
	mux := setupTestService(t)

	// admin1 creates group G1
	body := `{"name":"G-ADMIN","description":"test"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("create group: got %d body=%s", rec.Code, rec.Body.String())
	}
	var g map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	gid, _ := g["id"].(string)
	if gid == "" {
		t.Fatalf("group id empty: %s", rec.Body.String())
	}

	// admin1 加入该组
	add := `{"user_id":"admin1"}`
	req2 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/"+gid+"/add_member/", bytes.NewBufferString(add)), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 201 && rec2.Code != 200 {
		t.Fatalf("add member: got %d body=%s", rec2.Code, rec2.Body.String())
	}

	// 指派 admin1 为 G1 组长
	req3 := withUser(httptest.NewRequest(http.MethodPut, "/api/tenant/group-admin/company_id/c1/group_id/"+gid+"/user_id/admin1/", nil), "admin1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("assign group admin: got %d body=%s", rec3.Code, rec3.Body.String())
	}

	// 目标在组长所管组内 → in_admin_group=true
	qIn := "/api/internal/tenant/groups/user-in-admin-groups?admin_user_id=admin1&user_id=admin1&company_id=c1"
	recIn := httptest.NewRecorder()
	mux.ServeHTTP(recIn, httptest.NewRequest(http.MethodGet, qIn, nil))
	if recIn.Code != 200 {
		t.Fatalf("in-group query: got %d body=%s", recIn.Code, recIn.Body.String())
	}
	var in map[string]interface{}
	_ = json.Unmarshal(recIn.Body.Bytes(), &in)
	if in["in_admin_group"] != true {
		t.Fatalf("expected in_admin_group=true, got %v", in)
	}

	// 目标不在组长所管组内 → in_admin_group=false
	qOut := "/api/internal/tenant/groups/user-in-admin-groups?admin_user_id=admin1&user_id=member1&company_id=c1"
	recOut := httptest.NewRecorder()
	mux.ServeHTTP(recOut, httptest.NewRequest(http.MethodGet, qOut, nil))
	var out map[string]interface{}
	_ = json.Unmarshal(recOut.Body.Bytes(), &out)
	if out["in_admin_group"] != false {
		t.Fatalf("expected in_admin_group=false, got %v", out)
	}

	// 缺参 → 400
	recBad := httptest.NewRecorder()
	mux.ServeHTTP(recBad, httptest.NewRequest(http.MethodGet, "/api/internal/tenant/groups/user-in-admin-groups?admin_user_id=admin1", nil))
	if recBad.Code != 400 {
		t.Fatalf("expected 400 for missing params, got %d", recBad.Code)
	}
}

func TestInternalMembersSyncNickname(t *testing.T) {
	mux := setupTestService(t)
	// u-nick 三公司：c2 误种子（'我的公司'）、c3 空、c4 已设真实名
	if _, err := db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES ('c2','我的公司','u-nick')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tenant_company_member (id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m2','u-nick','c2',0,1,'ws1','我的公司')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES ('c3','Real Co','u-nick')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tenant_company_member (id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m3','u-nick','c3',0,1,'ws1','')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES ('c4','Real Co4','u-nick')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tenant_company_member (id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m4','u-nick','c4',0,1,'ws1','真实名')`); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/internal/tenant/members/sync-nickname",
		bytes.NewBufferString(`{"user_id":"u-nick","nickname":"微信昵称X"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("sync-nickname: got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	updated, _ := out["updated"].(float64)
	if int(updated) != 2 {
		t.Fatalf("expected updated=2, got %v", out)
	}

	var m2, m3, m4 string
	_ = db.QueryRow(`SELECT member_name FROM tenant_company_member WHERE id='m2'`).Scan(&m2)
	_ = db.QueryRow(`SELECT member_name FROM tenant_company_member WHERE id='m3'`).Scan(&m3)
	_ = db.QueryRow(`SELECT member_name FROM tenant_company_member WHERE id='m4'`).Scan(&m4)
	if m2 != "微信昵称X" || m3 != "微信昵称X" {
		t.Fatalf("mis-seeded rows should be healed: m2=%q m3=%q", m2, m3)
	}
	if m4 != "真实名" {
		t.Fatalf("user-set member_name must not be overwritten: m4=%q", m4)
	}

	recBad := httptest.NewRecorder()
	mux.ServeHTTP(recBad, httptest.NewRequest(http.MethodPatch, "/api/internal/tenant/members/sync-nickname",
		bytes.NewBufferString(`{"user_id":"u-nick"}`)))
	if recBad.Code != 400 {
		t.Fatalf("expected 400 for missing nickname, got %d", recBad.Code)
	}
}

// TestInternalMembersDisplayName — 合并审计等跨服务场景的展示名解析：
// 误种子 member_name（=公司名）回退个人昵称；正常 member_name 不被覆盖。
func TestInternalMembersDisplayName(t *testing.T) {
	mux := setupTestService(t)

	prev := fetchPersonalNicknamesFn
	fetchPersonalNicknamesFn = func(userIDs []string) map[string]string {
		out := map[string]string{}
		for _, id := range userIDs {
			out[id] = "软刀"
		}
		return out
	}
	t.Cleanup(func() { fetchPersonalNicknamesFn = prev })

	// 误种子：member_name = 公司名 → display_name 应回退个人昵称
	if _, err := db.Exec(`UPDATE tenant_company SET name=? WHERE id='c1'`, "Test Co"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE tenant_company_member SET member_name=? WHERE id='m1'`, "Test Co"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/display-name?user_id=admin1&company_id=c1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["display_name"] != "软刀" {
		t.Fatalf("expected display_name=软刀 (personal nickname), got %v", out)
	}

	// 正常 member_name 不被覆盖
	if _, err := db.Exec(`UPDATE tenant_company_member SET member_name=? WHERE id='m1'`, "真名"); err != nil {
		t.Fatal(err)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/display-name?user_id=admin1&company_id=c1", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	var out2 map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out2)
	if out2["display_name"] != "真名" {
		t.Fatalf("expected display_name=真名, got %v", out2)
	}

	// 缺参 → 400；不存在 → 404
	recBad := httptest.NewRecorder()
	mux.ServeHTTP(recBad, httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/display-name?user_id=admin1", nil))
	if recBad.Code != 400 {
		t.Fatalf("expected 400 for missing params, got %d", recBad.Code)
	}
	rec404 := httptest.NewRecorder()
	mux.ServeHTTP(rec404, httptest.NewRequest(http.MethodGet, "/api/internal/tenant/members/display-name?user_id=nonexist&company_id=c1", nil))
	if rec404.Code != 404 {
		t.Fatalf("expected 404 for unknown user, got %d", rec404.Code)
	}
}
