package main

import (
	"authz"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"snowflake"
)

// ═══════════════════════════════════════════════════════════════
// 租户成员角色 / 组角色 API (v63 + v75 multi-role replace-all / DELETE)
// ═══════════════════════════════════════════════════════════════

var httpClientRoleCheck = &http.Client{Timeout: 4 * time.Second}

// ensureMemberRole 幂等写入成员角色行（v63 硬切换：成员创建时若携带管理员标记，
// 同步落 tenant_member_role，避免依赖 is_admin 列判定）。
func ensureMemberRole(memberID, roleName, companyID, assignedBy string) {
	if memberID == "" || roleName == "" {
		return
	}
	if _, err := db.Exec(`INSERT INTO tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
		VALUES (?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE role_name = VALUES(role_name), assigned_by = VALUES(assigned_by)`,
		snowflake.GenerateIDString(), memberID, roleName, companyID, assignedBy); err != nil {
		logWarn("ensureMemberRole failed: "+err.Error(), "member_id="+memberID+" role="+roleName)
	}
}

func authServiceURL() string {
	if v := strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_URL")); v != "" {
		return v
	}
	if cfg.TaskAuthURL != "" {
		return cfg.TaskAuthURL
	}
	return "http://taskAuth:8003"
}

// roleExistsInAuthFn 接缝：单测可替换，避免依赖真实 taskAuth。
var roleExistsInAuthFn = roleExistsInAuth

func roleExistsInAuth(companyID, roleName string) (bool, error) {
	base := strings.TrimRight(authServiceURL(), "/")
	if base == "" {
		return false, fmt.Errorf("auth service not configured")
	}
	url := fmt.Sprintf("%s/api/internal/authz/role-exists?company_id=%s&role_name=%s", base, companyID, roleName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := httpClientRoleCheck.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("auth returned %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Exists bool `json:"exists"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return false, err
	}
	return out.Exists, nil
}

// parseRoleNamesBody accepts v75 {"role_names":[...]} or legacy {"role_name":"..."}.
func parseRoleNamesBody(r *http.Request) ([]string, error) {
	var body struct {
		RoleName  string   `json:"role_name"`
		RoleNames []string `json:"role_names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(body.RoleNames)+1)
	seen := map[string]bool{}
	add := func(n string) {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			return
		}
		seen[n] = true
		names = append(names, n)
	}
	for _, n := range body.RoleNames {
		add(n)
	}
	add(body.RoleName)
	return names, nil
}

func validateRolesExist(companyID string, names []string) error {
	for _, n := range names {
		exists, err := roleExistsInAuthFn(companyID, n)
		if err != nil {
			return fmt.Errorf("角色校验服务不可用")
		}
		if !exists {
			return fmt.Errorf("角色不存在: %s", n)
		}
	}
	return nil
}

// handleSetMemberRole — PUT …/member_id/{mid}/
// body: {"role_names":[...]} replace-all；兼容 {"role_name":"..."}
func handleSetMemberRole(w http.ResponseWriter, r *http.Request, cid, mid string) {
	if !authz.RequirePerm(w, r, authz.PermMemberManage, cid) {
		return
	}
	names, err := parseRoleNamesBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRolesExist(cid, names); err != nil {
		msg := err.Error()
		code := http.StatusBadRequest
		if strings.Contains(msg, "不可用") {
			code = http.StatusBadGateway
		}
		writeError(w, r, code, msg)
		return
	}
	var memberCompany, memberUser string
	if err := db.QueryRow(`SELECT company_id, user_id FROM tenant_company_member WHERE id=?`, mid).Scan(&memberCompany, &memberUser); err != nil {
		writeError(w, r, http.StatusNotFound, "成员不存在")
		return
	}
	if memberCompany != cid {
		writeError(w, r, http.StatusForbidden, "成员不属于该公司")
		return
	}
	tx, err := db.Begin()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM tenant_member_role WHERE member_id=? AND company_id=?`, mid, cid); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	assignedBy := getAuthUser(r)
	for _, roleName := range names {
		if _, err := tx.Exec(`INSERT INTO tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
			VALUES (?, ?, ?, ?, ?, NOW())`,
			snowflake.GenerateIDString(), mid, roleName, cid, assignedBy); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	incrMembershipRev(memberUser)
	publishTenantRoleChanged(cid)
	logInfo("member roles replaced", fmt.Sprintf("company_id=%s member_id=%s count=%d", cid, mid, len(names)))
	writeJSON(w, http.StatusOK, map[string]any{"detail": "ok", "member_id": mid, "role_names": names})
}

// handleDeleteMemberRole — DELETE …/member_id/{mid}/role_name/{name}/
func handleDeleteMemberRole(w http.ResponseWriter, r *http.Request, cid, mid, roleName string) {
	if !authz.RequirePerm(w, r, authz.PermMemberManage, cid) {
		return
	}
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		writeError(w, r, http.StatusBadRequest, "role_name required")
		return
	}
	var memberCompany, memberUser string
	if err := db.QueryRow(`SELECT company_id, user_id FROM tenant_company_member WHERE id=?`, mid).Scan(&memberCompany, &memberUser); err != nil {
		writeError(w, r, http.StatusNotFound, "成员不存在")
		return
	}
	if memberCompany != cid {
		writeError(w, r, http.StatusForbidden, "成员不属于该公司")
		return
	}
	res, err := db.Exec(`DELETE FROM tenant_member_role WHERE member_id=? AND company_id=? AND role_name=?`, mid, cid, roleName)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, r, http.StatusNotFound, "角色绑定不存在")
		return
	}
	incrMembershipRev(memberUser)
	publishTenantRoleChanged(cid)
	logInfo("member role removed", fmt.Sprintf("company_id=%s member_id=%s role=%s", cid, mid, roleName))
	writeJSON(w, http.StatusOK, map[string]any{"detail": "ok", "member_id": mid, "role_name": roleName})
}

func handleSetGroupRole(w http.ResponseWriter, r *http.Request, cid, gid string) {
	if !authz.RequirePerm(w, r, authz.PermGroupManage, cid) {
		return
	}
	names, err := parseRoleNamesBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRolesExist(cid, names); err != nil {
		msg := err.Error()
		code := http.StatusBadRequest
		if strings.Contains(msg, "不可用") {
			code = http.StatusBadGateway
		}
		writeError(w, r, code, msg)
		return
	}
	var groupCompany string
	if err := db.QueryRow(`SELECT company_id FROM tenant_company_group WHERE id=?`, gid).Scan(&groupCompany); err != nil {
		writeError(w, r, http.StatusNotFound, "小组不存在")
		return
	}
	if groupCompany != cid {
		writeError(w, r, http.StatusForbidden, "小组不属于该公司")
		return
	}
	tx, err := db.Begin()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM tenant_group_role WHERE group_id=? AND company_id=?`, gid, cid); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	assignedBy := getAuthUser(r)
	for _, roleName := range names {
		if _, err := tx.Exec(`INSERT INTO tenant_group_role (id, group_id, role_name, company_id, assigned_by, assigned_at)
			VALUES (?, ?, ?, ?, ?, NOW())`,
			snowflake.GenerateIDString(), gid, roleName, cid, assignedBy); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	incrGroupMembersRev(gid)
	publishTenantRoleChanged(cid)
	logInfo("group roles replaced", fmt.Sprintf("company_id=%s group_id=%s count=%d", cid, gid, len(names)))
	writeJSON(w, http.StatusOK, map[string]any{"detail": "ok", "group_id": gid, "role_names": names})
}

func handleDeleteGroupRole(w http.ResponseWriter, r *http.Request, cid, gid, roleName string) {
	if !authz.RequirePerm(w, r, authz.PermGroupManage, cid) {
		return
	}
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		writeError(w, r, http.StatusBadRequest, "role_name required")
		return
	}
	var groupCompany string
	if err := db.QueryRow(`SELECT company_id FROM tenant_company_group WHERE id=?`, gid).Scan(&groupCompany); err != nil {
		writeError(w, r, http.StatusNotFound, "小组不存在")
		return
	}
	if groupCompany != cid {
		writeError(w, r, http.StatusForbidden, "小组不属于该公司")
		return
	}
	res, err := db.Exec(`DELETE FROM tenant_group_role WHERE group_id=? AND company_id=? AND role_name=?`, gid, cid, roleName)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, r, http.StatusNotFound, "角色绑定不存在")
		return
	}
	incrGroupMembersRev(gid)
	publishTenantRoleChanged(cid)
	logInfo("group role removed", fmt.Sprintf("company_id=%s group_id=%s role=%s", cid, gid, roleName))
	writeJSON(w, http.StatusOK, map[string]any{"detail": "ok", "group_id": gid, "role_name": roleName})
}

func handleListMemberRoles(w http.ResponseWriter, r *http.Request, cid string) {
	if !authz.RequireTenantMember(w, r, cid) {
		return
	}
	rows, err := db.Query(`SELECT mr.id, mr.member_id, m.member_name, mr.role_name, mr.company_id, mr.assigned_at
		FROM tenant_member_role mr
		LEFT JOIN tenant_company_member m ON m.id = mr.member_id
		WHERE mr.company_id=? ORDER BY mr.assigned_at DESC`, cid)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var id, mid, role, c, at string
		var mnameNull sql.NullString
		if err := rows.Scan(&id, &mid, &mnameNull, &role, &c, &at); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, map[string]any{
			"id": id, "member_id": mid, "member_name": mnameNull.String, "role_name": role, "company_id": c, "assigned_at": at,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func handleListGroupRoles(w http.ResponseWriter, r *http.Request, cid string) {
	if !authz.RequireTenantMember(w, r, cid) {
		return
	}
	rows, err := db.Query(`SELECT gr.id, gr.group_id, g.name, gr.role_name, gr.assigned_at
		FROM tenant_group_role gr
		LEFT JOIN tenant_company_group g ON g.id = gr.group_id
		WHERE gr.company_id=? ORDER BY gr.assigned_at DESC`, cid)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var id, gid, role, at string
		var gnameNull sql.NullString
		if err := rows.Scan(&id, &gid, &gnameNull, &role, &at); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, map[string]any{
			"id": id, "group_id": gid, "group_name": gnameNull.String, "role_name": role, "assigned_at": at,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func publishTenantRoleChanged(companyID string) {
	publishEvent("TenantRoleChanged", map[string]interface{}{
		"company_id": companyID,
		"at":         time.Now().Format(time.RFC3339),
	})
}

// handleMemberRoleRoute 分发:
//
//	GET  /member-role/company_id/{cid}/
//	PUT  /member-role/company_id/{cid}/member_id/{mid}/
//	DELETE /member-role/company_id/{cid}/member_id/{mid}/role_name/{name}/
func handleMemberRoleRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 3 || parts[1] != "company_id" {
		writeError(w, r, 404, "not found")
		return
	}
	cid := parts[2]
	if len(parts) == 3 {
		if r.Method == http.MethodGet {
			handleListMemberRoles(w, r, cid)
			return
		}
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(parts) >= 5 && parts[3] == "member_id" {
		mid := parts[4]
		if len(parts) == 5 {
			switch r.Method {
			case http.MethodPut:
				handleSetMemberRole(w, r, cid, mid)
			case http.MethodGet:
				handleListMemberRoles(w, r, cid)
			default:
				writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if len(parts) >= 7 && parts[5] == "role_name" && r.Method == http.MethodDelete {
			handleDeleteMemberRole(w, r, cid, mid, parts[6])
			return
		}
	}
	writeError(w, r, 404, "not found")
}

func handleGroupRoleRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 3 || parts[1] != "company_id" {
		writeError(w, r, 404, "not found")
		return
	}
	cid := parts[2]
	if len(parts) == 3 {
		if r.Method == http.MethodGet {
			handleListGroupRoles(w, r, cid)
			return
		}
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(parts) >= 5 && parts[3] == "group_id" {
		gid := parts[4]
		if len(parts) == 5 {
			switch r.Method {
			case http.MethodPut:
				handleSetGroupRole(w, r, cid, gid)
			case http.MethodGet:
				handleListGroupRoles(w, r, cid)
			default:
				writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if len(parts) >= 7 && parts[5] == "role_name" && r.Method == http.MethodDelete {
			handleDeleteGroupRole(w, r, cid, gid, parts[6])
			return
		}
	}
	writeError(w, r, 404, "not found")
}
