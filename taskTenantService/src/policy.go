package main

import (
	"authz"
	"database/sql"
	"net/http"
	"strings"
)

type memberRow struct {
	ID           string
	UserID       string
	CompanyID    string
	IsAdmin      bool
	IsActive     bool
	WorkspaceID  string
	MemberName   string
	MemberAvatar string
	CreatedAt    string
}

const memberSelectCols = `id, user_id, company_id, is_admin, is_active,
		COALESCE(workspace_id,''), COALESCE(member_name,''), COALESCE(member_avatar,''), created_at`

func scanMember(row interface{ Scan(dest ...any) error }) (*memberRow, error) {
	var m memberRow
	var isAdmin, isActive int
	err := row.Scan(
		&m.ID, &m.UserID, &m.CompanyID, &isAdmin, &isActive,
		&m.WorkspaceID, &m.MemberName, &m.MemberAvatar, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.IsAdmin = isAdmin == 1
	m.IsActive = isActive == 1
	return &m, nil
}

// memberHasRole 判断成员是否持有指定角色（tenant_member_role 行，v63 角色模型）。
func memberHasRole(memberID, roleName string) bool {
	if memberID == "" || roleName == "" {
		return false
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(1) FROM tenant_member_role WHERE member_id=? AND role_name=?`, memberID, roleName).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

func memberToJSON(m *memberRow) map[string]interface{} {
	if m == nil {
		return nil
	}
	// v63 硬切换: is_admin 由角色行推导（tenant_admin 角色行存在即管理员），
	// 不再以 is_admin 列作为判定/回显来源（列保留仅作写入兼容）。
	isAdmin := memberHasRole(m.ID, "tenant_admin")
	return map[string]interface{}{
		"id": m.ID, "user_id": m.UserID, "company_id": m.CompanyID,
		"is_admin": isAdmin, "is_active": m.IsActive,
		"workspace_id": m.WorkspaceID, "member_name": m.MemberName,
		"member_avatar":     m.MemberAvatar,
		"member_avatar_url": memberAvatarPublicURL(m.CompanyID, m.ID, m.MemberAvatar),
		"created_at":        m.CreatedAt,
	}
}

func getMember(userID, companyID string) (*memberRow, error) {
	row := db.QueryRow(`SELECT `+memberSelectCols+`
		FROM tenant_company_member
		WHERE user_id=? AND company_id=?`, userID, companyID)
	return scanMember(row)
}

func getMemberByID(id, companyID string) (*memberRow, error) {
	row := db.QueryRow(`SELECT `+memberSelectCols+`
		FROM tenant_company_member WHERE id=? AND company_id=?`, id, companyID)
	return scanMember(row)
}

// requireCompanyAdmin returns nil if OK, or writes response and returns false.
// v63: 判定迁移至 authz 权限码（member:manage）；is_admin/creator 由 PDP 过渡期兼容。
func requireCompanyAdmin(w http.ResponseWriter, r *http.Request, userID, companyID string) bool {
	return authz.RequirePerm(w, r, authz.PermMemberManage, companyID)
}

func checkCompanyAdmin(r *http.Request, userID, companyID string) (ok bool, msg string) {
	if !authz.HasPerm(r, companyID, authz.PermMemberManage) {
		return false, "您没有该公司的人员管理权限"
	}
	return true, ""
}

// requireCompanyMember returns the member row if OK.
// v63: 判定迁移至 authz（TenantPerms 非空 = 有效成员）；返回 nil 表示无成员权限。
func requireCompanyMember(w http.ResponseWriter, r *http.Request, userID, companyID string) (*memberRow, bool) {
	if !authz.RequireTenantMember(w, r, companyID) {
		return nil, false
	}
	m, err := getMember(userID, companyID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "内部错误")
		return nil, false
	}
	if m == nil {
		// creator 无成员行时仍视为成员（PDP 已含 creator 回填）
		return &memberRow{UserID: userID, CompanyID: companyID, IsAdmin: true, IsActive: true}, true
	}
	return m, true
}

// isCreator 保留用于数据回显（member_handlers is_creator 标志），非鉴权用途。
func isCreator(companyID, userID string) bool {
	creatorID, found, err := fetchCompanyCreator(companyID)
	if err != nil || !found {
		return false
	}
	return creatorID != "" && strings.TrimSpace(creatorID) == strings.TrimSpace(userID)
}
