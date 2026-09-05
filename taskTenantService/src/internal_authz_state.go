package main

import (
	"fmt"
	"net/http"
)

// AuthzState 是 taskAuth PDP 计算权限码集合所需的全部租户侧数据。
// GET /api/internal/tenant/authz-state?user_id=X
// 由 taskAuth forward-auth 调用（internal secret 校验）。
type AuthzStateCompany struct {
	CompanyID     string           `json:"company_id"`
	MemberID      string           `json:"member_id"`
	IsActive      bool             `json:"is_active"`
	IsAdmin       bool             `json:"is_admin"` // 硬切换过渡期保留
	DirectRoles   []string         `json:"direct_roles"`
	GroupRoles    []GroupRoleEntry `json:"group_roles"`
	GroupAdmins   []GroupAdminEty  `json:"group_admins"`
	GroupResource []GroupResEntry  `json:"group_resources"`
}

type GroupRoleEntry struct {
	GroupID string `json:"group_id"`
	Role    string `json:"role"`
}

type GroupAdminEty struct {
	GroupID string `json:"group_id"`
}

type GroupResEntry struct {
	GroupID      string `json:"group_id"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Permission   string `json:"permission"`
}

// handleInternalAuthzState 返回用户在全部公司中的授权状态（角色 + 组 + 资源）。
func handleInternalAuthzState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, r, http.StatusBadRequest, "user_id required")
		return
	}

	// 1. 成员列表（含 is_admin 过渡字段）
	members, err := fetchMemberRowsByUser(userID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "load members failed")
		return
	}
	if len(members) == 0 {
		writeJSON(w, http.StatusOK, []AuthzStateCompany{})
		return
	}

	result := make([]AuthzStateCompany, 0, len(members))
	for _, m := range members {
		entry := AuthzStateCompany{
			CompanyID: m.CompanyID,
			MemberID:  m.ID,
			IsActive:  m.IsActive,
		}

		// 2. 直接角色；IsAdmin 由角色行推导（v63 硬切换，不再读 is_admin 列）
		if roles, err := fetchMemberRoles(m.ID); err == nil {
			entry.DirectRoles = roles
			for _, r := range roles {
				if r == "tenant_admin" {
					entry.IsAdmin = true
					break
				}
			}
		}

		// 3. 该成员所属的组
		groupIDs, err := fetchGroupIDsByUser(userID, m.CompanyID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "load groups failed")
			return
		}
		if len(groupIDs) > 0 {
			// 组→角色继承
			if gr, err := fetchGroupRoles(groupIDs); err == nil {
				entry.GroupRoles = gr
			}
			// 组管理员指派（该成员为哪些组的管理员）
			if ga, err := fetchGroupAdminsByUser(userID, m.CompanyID); err == nil {
				entry.GroupAdmins = ga
			}
			// 组资源分配（组成员 → 组 → 资源）
			if res, err := fetchGroupResources(groupIDs, m.CompanyID); err == nil {
				entry.GroupResource = res
			}
		}

		result = append(result, entry)
	}
	writeJSON(w, http.StatusOK, result)
}

// buildInQuery 生成 `col IN (?,?,...)` 占位符查询（q 须含 %s 一次）。
func buildInQuery(q string, n int) string {
	if n <= 0 {
		return q
	}
	placeholders := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
	}
	return fmt.Sprintf(q, placeholders)
}

func fetchMemberRowsByUser(userID string) ([]memberRow, error) {
	rows, err := db.Query(`SELECT `+memberSelectCols+`
		FROM tenant_company_member WHERE user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []memberRow
	for rows.Next() {
		var m memberRow
		var isAdmin, isActive int
		if err := rows.Scan(&m.ID, &m.UserID, &m.CompanyID, &isAdmin, &isActive,
			&m.WorkspaceID, &m.MemberName, &m.MemberAvatar, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.IsAdmin = isAdmin == 1
		m.IsActive = isActive == 1
		out = append(out, m)
	}
	return out, rows.Err()
}

func fetchMemberRoles(memberID string) ([]string, error) {
	rows, err := db.Query(`SELECT role_name FROM tenant_member_role WHERE member_id=?`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err == nil {
			out = append(out, r)
		}
	}
	return out, rows.Err()
}

func fetchGroupIDsByUser(userID, companyID string) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT g.id
		FROM tenant_company_group_member gm
		JOIN tenant_company_group g ON g.id = gm.group_id
		WHERE gm.user_id=? AND g.company_id=?`, userID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

func fetchGroupRoles(groupIDs []string) ([]GroupRoleEntry, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	q := buildInQuery("SELECT group_id, role_name FROM tenant_group_role WHERE group_id IN (%s)", len(groupIDs))
	args := make([]any, len(groupIDs))
	for i, g := range groupIDs {
		args[i] = g
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupRoleEntry
	for rows.Next() {
		var e GroupRoleEntry
		if err := rows.Scan(&e.GroupID, &e.Role); err == nil {
			out = append(out, e)
		}
	}
	return out, rows.Err()
}

func fetchGroupAdminsByUser(userID, companyID string) ([]GroupAdminEty, error) {
	rows, err := db.Query(`SELECT group_id FROM tenant_group_admin WHERE user_id=? AND company_id=?`, userID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupAdminEty
	for rows.Next() {
		var e GroupAdminEty
		if err := rows.Scan(&e.GroupID); err == nil {
			out = append(out, e)
		}
	}
	return out, rows.Err()
}

func fetchGroupResources(groupIDs []string, companyID string) ([]GroupResEntry, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	q := buildInQuery(`SELECT group_id, resource_type, resource_id, permission
		FROM tenant_resource_group_assignment
		WHERE company_id=? AND group_id IN (%s)`, len(groupIDs))
	args := []any{companyID}
	for _, g := range groupIDs {
		args = append(args, g)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupResEntry
	for rows.Next() {
		var e GroupResEntry
		if err := rows.Scan(&e.GroupID, &e.ResourceType, &e.ResourceID, &e.Permission); err == nil {
			out = append(out, e)
		}
	}
	return out, rows.Err()
}
