package main

import (
	"net/http"
	"strings"
	"time"
)

func handleCompanyMembers(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	hasPerm, msg := checkCompanyAdmin(r, userID, tenantID)
	if !hasPerm {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"members": []interface{}{},
			"meta": map[string]interface{}{
				"has_permission": false, "permission_message": msg,
				"can_manage_budget_permissions": false, "llm_budget_enabled": false,
				"budget_raise_by_member_id": map[string]bool{},
			},
		})
		return
	}

	creatorID, _, _ := fetchCompanyCreator(tenantID)
	companyName := ""
	if c, err := getCompanyByID(tenantID); err == nil && c != nil {
		companyName = c.Name
	}
	rows, err := db.Query(`
		SELECT id, user_id, is_admin, is_active, created_at, COALESCE(member_name,''), COALESCE(member_avatar,'')
		FROM tenant_company_member WHERE company_id=? ORDER BY created_at ASC`, tenantID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "查询失败")
		return
	}
	defer rows.Close()

	type memberScan struct {
		id, uid, created, name, avatar string
		isAdmin, isActive              int
	}
	scanned := make([]memberScan, 0)
	userIDs := make([]string, 0)
	for rows.Next() {
		var m memberScan
		if err := rows.Scan(&m.id, &m.uid, &m.isAdmin, &m.isActive, &m.created, &m.name, &m.avatar); err != nil {
			continue
		}
		scanned = append(scanned, m)
		userIDs = append(userIDs, m.uid)
	}

	nicknames := fetchPersonalNicknamesFn(userIDs)
	members := make([]map[string]interface{}, 0, len(scanned))
	oneMonthAgo := time.Now().UTC().AddDate(0, -1, 0)
	for _, row := range scanned {
		isCreator := creatorID != "" && creatorID == row.uid
		role := "member"
		if isCreator {
			role = "创建者"
		} else if row.isAdmin == 1 {
			role = "admin"
		}
		status := "inactive"
		if row.isActive == 1 {
			if t, err := time.Parse(time.RFC3339, row.created); err == nil && !t.Before(oneMonthAgo) {
				status = "active"
			} else if len(row.created) >= 10 {
				// sqlite CURRENT_TIMESTAMP often "2006-01-02 15:04:05"
				if t, err := time.Parse("2006-01-02 15:04:05", row.created); err == nil && !t.Before(oneMonthAgo) {
					status = "active"
				} else if row.isActive == 1 {
					status = "active"
				}
			} else {
				status = "active"
			}
		}
		personalNick := nicknames[row.uid]
		display := resolveMemberDisplayName(row.name, companyName, personalNick, row.uid)
		if isMisSeededMemberName(row.name, companyName) && personalNick != "" {
			healMisSeededMemberName(row.id, personalNick)
		}
		joinDate := row.created
		if len(joinDate) >= 10 {
			joinDate = joinDate[:10]
		}
		members = append(members, map[string]interface{}{
			"id": row.id, "company_member_id": row.id, "member_name": display,
			"member_avatar_url": memberAvatarPublicURL(tenantID, row.id, row.avatar),
			"email":             "", "role": role, "joinDate": joinDate,
			"status": status, "is_creator": isCreator, "can_raise_task_budget": false,
			"cannot_remove": isCreator, "cannot_disable": isCreator, "cannot_change_role": isCreator,
		})
	}

	llmEnabled, raiseByMember := fetchLLMBudgetMeta(tenantID)
	for _, m := range members {
		mid, _ := m["id"].(string)
		if raiseByMember[mid] {
			m["can_raise_task_budget"] = true
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"members": members,
		"meta": map[string]interface{}{
			"has_permission": true, "permission_message": "",
			"can_manage_budget_permissions": true, "llm_budget_enabled": llmEnabled,
			"budget_raise_by_member_id": raiseByMember,
		},
	})
}

func handleUpdateRole(w http.ResponseWriter, r *http.Request, tenantID, memberID string) {
	if r.Method != http.MethodPatch {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if !requireCompanyAdmin(w, r, userID, tenantID) {
		return
	}
	m, err := getMemberByID(memberID, tenantID)
	if err != nil || m == nil {
		writeError(w, r, http.StatusNotFound, "成员不存在")
		return
	}
	body, _ := readJSONBody(r)
	role := strField(body, "role")
	memberName := strField(body, "member_name")
	if isCreator(tenantID, m.UserID) {
		if role != "" && role != "创建者" {
			writeError(w, r, http.StatusBadRequest, "公司创建者的角色不可被修改")
			return
		}
		if memberName == "" {
			writeError(w, r, http.StatusBadRequest, "请提供要更新的名称")
			return
		}
		_, _ = db.Exec(`UPDATE tenant_company_member SET member_name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, memberName, memberID)
		writeJSON(w, http.StatusOK, map[string]interface{}{"message": "名称已更新", "member": map[string]string{"id": memberID, "member_name": memberName}})
		return
	}
	if role == "" {
		writeError(w, r, http.StatusBadRequest, "角色不能为空")
		return
	}
	isAdmin := 0
	if role == "admin" {
		isAdmin = 1
	}
	if memberName != "" {
		_, _ = db.Exec(`UPDATE tenant_company_member SET is_admin=?, member_name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, isAdmin, memberName, memberID)
	} else {
		_, _ = db.Exec(`UPDATE tenant_company_member SET is_admin=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, isAdmin, memberID)
	}
	incrMembershipRev(m.UserID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "角色已更新", "member": map[string]interface{}{"id": memberID, "is_admin": isAdmin == 1}})
}

func handleToggleStatus(w http.ResponseWriter, r *http.Request, tenantID, memberID string) {
	if r.Method != http.MethodPatch {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if !requireCompanyAdmin(w, r, userID, tenantID) {
		return
	}
	m, err := getMemberByID(memberID, tenantID)
	if err != nil || m == nil {
		writeError(w, r, http.StatusNotFound, "成员不存在")
		return
	}
	if isCreator(tenantID, m.UserID) {
		writeError(w, r, http.StatusBadRequest, "公司创建者不可被禁用")
		return
	}
	newActive := 1
	status := "active"
	if m.IsActive {
		newActive = 0
		status = "inactive"
	}
	_, _ = db.Exec(`UPDATE tenant_company_member SET is_active=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, newActive, memberID)
	incrMembershipRev(m.UserID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "状态已更新", "status": status})
}

func handleDeleteMember(w http.ResponseWriter, r *http.Request, tenantID, memberID string) {
	if r.Method != http.MethodDelete {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if !requireCompanyAdmin(w, r, userID, tenantID) {
		return
	}
	m, err := getMemberByID(memberID, tenantID)
	if err != nil || m == nil {
		writeError(w, r, http.StatusNotFound, "成员不存在")
		return
	}
	if isCreator(tenantID, m.UserID) {
		writeError(w, r, http.StatusBadRequest, "公司创建者不可被移除")
		return
	}
	incrMembershipRev(m.UserID)
	_, _ = db.Exec(`DELETE FROM tenant_company_member WHERE id=?`, memberID)
	w.WriteHeader(http.StatusNoContent)
}

func handleMembersRoute(w http.ResponseWriter, r *http.Request, tenantID string, parts []string) {
	// parts after "members"
	if len(parts) == 0 {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	switch parts[0] {
	case "invite":
		handleInvite(w, r, tenantID)
	case "validate-invite":
		handleValidateInvite(w, r, tenantID)
	case "join":
		handleJoin(w, r, tenantID)
	case "company_members":
		handleCompanyMembers(w, r, tenantID)
	case "pending-invitations":
		handlePendingInvitations(w, r, tenantID)
	default:
		id := parts[0]
		if len(parts) == 1 {
			handleDeleteMember(w, r, tenantID, id)
			return
		}
		switch parts[1] {
		case "avatar":
			handleMemberAvatarGET(w, r, tenantID, id)
		case "update_role":
			handleUpdateRole(w, r, tenantID, id)
		case "toggle_status":
			handleToggleStatus(w, r, tenantID, id)
		case "revoke-invitation":
			handleRevokeInvitation(w, r, tenantID, id)
		case "resend-invitation-link":
			handleResendInvitation(w, r, tenantID, id)
		default:
			writeError(w, r, http.StatusNotFound, "not found")
		}
	}
}

func displayNameFallback(name, userID string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	if len(userID) > 8 {
		return userID[:8]
	}
	return userID
}
