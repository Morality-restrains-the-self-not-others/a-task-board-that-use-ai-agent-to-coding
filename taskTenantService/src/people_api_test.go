package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dbload"
	"gatewayauth"
)

func setupTestService(t *testing.T) *http.ServeMux {
	t.Helper()
	dir := t.TempDir()
	// 独立测试库自举（与 taskAuth/taskBill/taskReferral 一致）：dbload 创建唯一测试库、
	// 退出即 DROP。此前硬编码 test_task_tenant，夜间 mysql_reset 会清掉该静态库，
	// 导致 openDB 直接失败（Error 1049: Unknown database 'test_task_tenant'）。
	testDSN, cleanup, err := dbload.OpenTestMySQLClonedFromDir(
		"task-tenant", repoRoot(), "dataMigrate/taskTenantService",
		func(dsn string) error {
			if err := openDB(dsn); err != nil {
				return err
			}
			defer db.Close()
			return runDataMigrate(repoRoot())
		},
	)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	cfg = Config{
		Host:         "0.0.0.0",
		Port:         8020,
		DBPath:       testDSN,
		FrontendBase: "http://example.test",
	}
	if err := openDB(cfg.DBPath); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	ensureInvitationDeliveryColumns(t)
	ensurePendingGrantsColumn(t)
	ensurePendingRoleNamesColumn(t)
	ensureOpenInviteColumns(t)
	initMemberAvatarMediaRoot(dir)
	t.Cleanup(func() {
		// Clean all test data
		db.Exec(`DELETE FROM tenant_member_role`)
		db.Exec(`DELETE FROM tenant_company_group_member`)
		db.Exec(`DELETE FROM tenant_company_group`)
		db.Exec(`DELETE FROM tenant_invitation_redemption`)
		db.Exec(`DELETE FROM tenant_invitation`)
		db.Exec(`DELETE FROM tenant_company_member`)
		db.Exec(`DELETE FROM tenant_company`)
		if db != nil {
			_ = db.Close()
			db = nil
		}
	})
	// Clean existing data before seeding
	db.Exec(`DELETE FROM tenant_member_role`)
	db.Exec(`DELETE FROM tenant_company_group_member`)
	db.Exec(`DELETE FROM tenant_company_group`)
	db.Exec(`DELETE FROM tenant_invitation_redemption`)
	db.Exec(`DELETE FROM tenant_invitation`)
	db.Exec(`DELETE FROM tenant_company_member`)
	db.Exec(`DELETE FROM tenant_company`)
	// seed company + admin member for company c1 (tenant_company is creator SSOT)
	_, err = db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES ('c1','Test Co','admin1')`)
	if err != nil {
		t.Fatalf("seed company: %v", err)
	}
	_, err = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m1','admin1','c1',1,1,'ws1','Admin')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	// v63: 管理员由角色行定义（is_admin 列已停用判定）
	_, err = db.Exec(`INSERT INTO tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
		VALUES ('r-m1','m1','tenant_admin','c1','system',NOW())`)
	if err != nil {
		t.Fatalf("seed role: %v", err)
	}
	mux := http.NewServeMux()
	mountRoutes(mux)
	return mux
}

// 模拟 APISIX forward-auth 注入头（PDP 已展开权限码）:
// admin1 = 租户管理员（tenant_admin 全量租户码）；已播种成员 = member 码（view 级，无 manage）；
// 未知用户 = 无头（网关不会为无成员关系用户注入权限码）。
func tenantPermsFor(userID string) string {
	if userID == "admin1" {
		return "c1:member:manage,member:view,company:manage,company:view,group:manage,group-members:manage,group-resources:manage,group-resources:view,project:manage,project:view,task:manage,task:view,cloud:manage,cloud:view,billing:manage,billing:view,workspace:manage,resources:manage,resources:view"
	}
	switch userID {
	case "member1", "member2", "u2", "u3", "u5", "u7":
		return "c1:member:view,company:view,project:view,task:manage,task:view,cloud:view,billing:view,workspace:view,resources:view,group-resources:view"
	}
	return ""
}

func withUser(req *http.Request, userID string) *http.Request {
	req.Header.Set(gatewayauth.HeaderAuthUserID, userID)
	req.Header.Set("X-User-Id", userID)
	req.Header.Set("X-Tenant-Perms", tenantPermsFor(userID))
	return req
}

func TestInviteLinkAndValidate(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","company_member_name":"Bob","role":"member","workspace_id":"ws1","expiration_days":7}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("invite status %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	token, _ := out["invite_token"].(string)
	if token == "" {
		t.Fatal("missing token")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/validate-invite/?token="+token, nil)
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("validate %d %s", rec2.Code, rec2.Body.String())
	}
}

func TestCompanyMembersPermissionDenied(t *testing.T) {
	mux := setupTestService(t)
	_, _ = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active) VALUES ('m2','member1','c1',0,1)`)
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/company_members/", nil), "member1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	meta, _ := out["meta"].(map[string]interface{})
	if meta["has_permission"] != false {
		t.Fatalf("expected no permission, got %#v", meta)
	}
}

func TestCreateGroupAndMembers(t *testing.T) {
	mux := setupTestService(t)
	body := `{"name":"G1","description":"d"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create group %d %s", rec.Code, rec.Body.String())
	}
	var g map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	gid, _ := g["id"].(string)

	_, _ = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active) VALUES ('m3','u2','c1',0,1)`)
	add := `{"user_id":"u2"}`
	req2 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/"+gid+"/add_member/", bytes.NewBufferString(add)), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 201 {
		t.Fatalf("add member %d %s", rec2.Code, rec2.Body.String())
	}

	req3 := withUser(httptest.NewRequest(http.MethodDelete, "/api/tenant/c1/accounts/groups/"+gid+"/remove_member/", bytes.NewBufferString(add)), "admin1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("remove %d %s", rec3.Code, rec3.Body.String())
	}
}

// TestUpdateGroup — PUT /accounts/groups/{id}/ 更新名称与描述（编辑按钮依赖此接口）
func TestUpdateGroup(t *testing.T) {
	mux := setupTestService(t)
	body := `{"name":"G-old","description":"old-desc"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create group %d %s", rec.Code, rec.Body.String())
	}
	var g map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	gid, _ := g["id"].(string)
	if gid == "" {
		t.Fatal("missing group id")
	}

	upd := `{"name":"G-new","description":"new-desc"}`
	req2 := withUser(httptest.NewRequest(http.MethodPut, "/api/tenant/c1/accounts/groups/"+gid+"/", bytes.NewBufferString(upd)), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("update group %d %s", rec2.Code, rec2.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out)
	if out["name"] != "G-new" || out["description"] != "new-desc" {
		t.Fatalf("unexpected update body: %#v", out)
	}

	req3 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/groups/", nil), "admin1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("list groups %d %s", rec3.Code, rec3.Body.String())
	}
	var list []map[string]interface{}
	_ = json.Unmarshal(rec3.Body.Bytes(), &list)
	found := false
	for _, item := range list {
		if item["id"] == gid {
			found = true
			if item["name"] != "G-new" || item["description"] != "new-desc" {
				t.Fatalf("list not updated: %#v", item)
			}
		}
	}
	if !found {
		t.Fatalf("updated group not in list: %#v", list)
	}
}

// TestGroupManagePermissionDenied — 分组创建/改名/删除需 group:manage（或 member:manage / group-members:manage），
// 与 FE PeopleGroups 页门禁一致（OPT-20260811-077）。view-only 成员应 403。
func TestGroupManagePermissionDenied(t *testing.T) {
	mux := setupTestService(t)
	// 管理员建组作为后续删除/改名目标
	body := `{"name":"G-perm","description":"d"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create group %d %s", rec.Code, rec.Body.String())
	}
	var g map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	gid, _ := g["id"].(string)
	if gid == "" {
		t.Fatal("missing group id")
	}

	// view-only 成员（无 group:manage / group-members:manage / member:manage）
	postReq := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/groups/", bytes.NewBufferString(`{"name":"X"}`)), "u5")
	postRec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusForbidden {
		t.Fatalf("view-only create group status %d body=%s", postRec.Code, postRec.Body.String())
	}

	putReq := withUser(httptest.NewRequest(http.MethodPut, "/api/tenant/c1/accounts/groups/"+gid+"/", bytes.NewBufferString(`{"name":"X2","description":"d"}`)), "u5")
	putRec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusForbidden {
		t.Fatalf("view-only update group status %d body=%s", putRec.Code, putRec.Body.String())
	}

	delReq := withUser(httptest.NewRequest(http.MethodDelete, "/api/tenant/c1/accounts/groups/"+gid+"/", nil), "u5")
	delRec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusForbidden {
		t.Fatalf("view-only delete group status %d body=%s", delRec.Code, delRec.Body.String())
	}

	// 管理员仍可更新/删除（回归）
	updReq := withUser(httptest.NewRequest(http.MethodPut, "/api/tenant/c1/accounts/groups/"+gid+"/", bytes.NewBufferString(`{"name":"G-renamed","description":"d"}`)), "admin1")
	updRec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(updRec, updReq)
	if updRec.Code != http.StatusOK {
		t.Fatalf("admin update group status %d body=%s", updRec.Code, updRec.Body.String())
	}
	delReq2 := withUser(httptest.NewRequest(http.MethodDelete, "/api/tenant/c1/accounts/groups/"+gid+"/", nil), "admin1")
	delRec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(delRec2, delReq2)
	if delRec2.Code != http.StatusNoContent {
		t.Fatalf("admin delete group status %d body=%s", delRec2.Code, delRec2.Body.String())
	}
}

// ensureInvitationDeliveryColumns 幂等应用 dataMigrate 008 投递状态列，
// 使测试库与迁移后 schema 一致（新代码 SELECT 含 delivery_status）。
func ensureInvitationDeliveryColumns(t *testing.T) {
	t.Helper()
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tenant_invitation' AND COLUMN_NAME = 'delivery_status'`).Scan(&n)
	if n > 0 {
		return
	}
	if _, err := db.Exec(`ALTER TABLE tenant_invitation
		ADD COLUMN delivery_status VARCHAR(32) NOT NULL DEFAULT 'none',
		ADD COLUMN delivery_error VARCHAR(512) DEFAULT NULL,
		ADD COLUMN email_sent_at DATETIME DEFAULT NULL`); err != nil {
		t.Fatalf("apply delivery columns: %v", err)
	}
}

func ensurePendingGrantsColumn(t *testing.T) {
	t.Helper()
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tenant_invitation' AND COLUMN_NAME = 'pending_grants'`).Scan(&n)
	if n > 0 {
		return
	}
	if _, err := db.Exec(`ALTER TABLE tenant_invitation ADD COLUMN pending_grants JSON NULL`); err != nil {
		t.Fatalf("apply pending_grants column: %v", err)
	}
}

// ensurePendingRoleNamesColumn 幂等应用 dataMigrate 010 可复用角色预绑列，
// 使测试库与迁移后 schema 一致（新代码 SELECT/INSERT 含 pending_role_names）。
func ensurePendingRoleNamesColumn(t *testing.T) {
	t.Helper()
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tenant_invitation' AND COLUMN_NAME = 'pending_role_names'`).Scan(&n)
	if n > 0 {
		return
	}
	if _, err := db.Exec(`ALTER TABLE tenant_invitation ADD COLUMN pending_role_names JSON NULL`); err != nil {
		t.Fatalf("apply pending_role_names column: %v", err)
	}
}

func ensureOpenInviteColumns(t *testing.T) {
	t.Helper()
	adds := []struct{ col, def string }{
		{"link_kind", "VARCHAR(16) NOT NULL DEFAULT 'single'"},
		{"max_uses", "INT NOT NULL DEFAULT 1"},
		{"use_count", "INT NOT NULL DEFAULT 0"},
	}
	for _, a := range adds {
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tenant_invitation' AND COLUMN_NAME = ?`, a.col).Scan(&n)
		if n > 0 {
			continue
		}
		if _, err := db.Exec("ALTER TABLE tenant_invitation ADD COLUMN " + a.col + " " + a.def); err != nil {
			t.Fatalf("apply %s: %v", a.col, err)
		}
	}
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS tenant_invitation_redemption (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		invitation_id VARCHAR(64) NOT NULL,
		company_id VARCHAR(64) NOT NULL,
		user_id VARCHAR(64) NOT NULL,
		member_id VARCHAR(64) NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY uq_tenant_invitation_redemption_invite_user (invitation_id, user_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
}
