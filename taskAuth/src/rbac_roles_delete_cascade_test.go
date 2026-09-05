package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleDeleteRoleCascadesResourceGroupsAndPermissions — 删除自定义角色时必须
// 先清 auth_role_resource_group / auth_role_permission，否则 FK 或残留授权会导致失败。
func TestHandleDeleteRoleCascadesResourceGroupsAndPermissions(t *testing.T) {
	setupAuthTestDB(t)

	const (
		companyID = "c-del-cascade-1"
		roleID    = "role-custom-del-cascade"
		rgBindID  = "rrg-custom-del-cascade"
		rpID      = "rp-custom-del-cascade"
	)

	if _, err := db.Exec(`
INSERT INTO auth_role (id, name, display_name, level, priority, is_system, company_id, description)
VALUES (?, 'custom_del_cascade', '删除级联测', 'tenant', 10, 0, ?, 'test')`,
		roleID, companyID); err != nil {
		t.Fatalf("insert role: %v", err)
	}
	if _, err := db.Exec(`
INSERT INTO auth_role_permission (id, role_id, permission_id) VALUES (?, ?, 'perm-member-view')`,
		rpID, roleID); err != nil {
		t.Fatalf("insert role_permission: %v", err)
	}
	if _, err := db.Exec(`
INSERT INTO auth_role_resource_group (id, role_id, resource_group_id, effect)
VALUES (?, ?, 'rg-page-people-access', 'view')`,
		rgBindID, roleID); err != nil {
		t.Fatalf("insert role_resource_group: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/roles/role_id/"+roleID+"/", nil)
	req.SetPathValue("rid", roleID)
	req.Header.Set("X-User-Id", "u-admin")
	req.Header.Set("X-Tenant-Perms", companyID+":member:manage")
	rec := httptest.NewRecorder()
	handleDeleteRole(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_role WHERE id=?`, roleID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("auth_role should be gone: n=%d err=%v", n, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_role_permission WHERE role_id=?`, roleID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("auth_role_permission should cascade: n=%d err=%v", n, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_role_resource_group WHERE role_id=?`, roleID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("auth_role_resource_group should cascade: n=%d err=%v", n, err)
	}
}
