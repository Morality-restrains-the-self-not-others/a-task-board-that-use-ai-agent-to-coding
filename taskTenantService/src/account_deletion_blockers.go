package main

import (
	"fmt"
	"net/http"
	"strings"
)

type tenantDeletionBlocker struct {
	Code       string `json:"code"`
	Blocking   bool   `json:"blocking"`
	Message    string `json:"message"`
	ActionURL  string `json:"action_url,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	TenantName string `json:"tenant_name,omitempty"`
}

func handleInternalTenantUsersRouter(w http.ResponseWriter, r *http.Request) {
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/tenant/users/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || (parts[1] != "account-deletion-blockers" && parts[1] != "personal-data") || r.Method != http.MethodGet {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	userID := strings.TrimSpace(parts[0])
	if userID == "" {
		writeError(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	if parts[1] == "personal-data" {
		handleInternalTenantPersonalData(w, r, userID)
		return
	}
	blockers, err := collectTenantDeletionBlockers(userID)
	if err != nil {
		logError("account_deletion_blockers collect failed user_id="+userID+" err="+err.Error(), traceIDForError(r))
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	codes := make([]string, 0, len(blockers))
	for _, b := range blockers {
		codes = append(codes, b.Code)
	}
	logInfo("account_deletion_blockers collected user_id="+userID+" count="+fmt.Sprintf("%d", len(blockers))+" codes="+strings.Join(codes, ","), traceIDForError(r))
	writeJSON(w, http.StatusOK, map[string]interface{}{"blockers": blockers})
}

func collectTenantDeletionBlockers(userID string) ([]tenantDeletionBlocker, error) {
	members, err := listMembersByUser(userID)
	if err != nil {
		return nil, err
	}
	byTenant := map[string][]map[string]interface{}{}
	for _, m := range members {
		cid, _ := m["company_id"].(string)
		cid = strings.TrimSpace(cid)
		if cid == "" {
			continue
		}
		byTenant[cid] = append(byTenant[cid], m)
	}
	var blockers []tenantDeletionBlocker
	for cid, rows := range byTenant {
		companyName, _ := rows[0]["company_name"].(string)
		adminCount, memberCount, err := countTenantAdminsAndMembers(cid)
		if err != nil {
			return nil, err
		}
		userIsAdmin := memberHasTenantAdminRole(userID, rows)
		if userIsAdmin && adminCount <= 1 && memberCount > 1 {
			blockers = append(blockers, tenantDeletionBlocker{
				Code: "TENANT_SOLE_ADMIN", Blocking: true,
				Message:    fmt.Sprintf("你是租户「%s」的唯一管理员，请先转让管理员权限", companyName),
				ActionURL:  tenantMembersActionURL(cid),
				TenantID:   cid,
				TenantName: companyName,
			})
		}
	}
	return blockers, nil
}

func memberHasTenantAdminRole(userID string, rows []map[string]interface{}) bool {
	for _, m := range rows {
		uid, _ := m["user_id"].(string)
		if uid != userID {
			continue
		}
		mid, _ := m["id"].(string)
		if mid == "" {
			continue
		}
		roles, err := listMemberRoleNames(mid)
		if err != nil {
			continue
		}
		for _, role := range roles {
			if role == "tenant_admin" {
				return true
			}
		}
	}
	return false
}

func listMemberRoleNames(memberID string) ([]string, error) {
	rows, err := db.Query(`SELECT role_name FROM tenant_member_role WHERE member_id=?`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err == nil && role != "" {
			roles = append(roles, role)
		}
	}
	return roles, nil
}

func countTenantAdminsAndMembers(companyID string) (adminCount, memberCount int, err error) {
	err = db.QueryRow(`SELECT COUNT(1) FROM tenant_company_member WHERE company_id=? AND is_active=1`, companyID).Scan(&memberCount)
	if err != nil {
		return 0, 0, err
	}
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT mr.member_id)
		FROM tenant_member_role mr
		INNER JOIN tenant_company_member m ON m.id = mr.member_id
		WHERE mr.company_id=? AND mr.role_name='tenant_admin' AND m.is_active=1`, companyID).Scan(&adminCount)
	return adminCount, memberCount, err
}
