package main

import (
	"log/slog"
	"net/http"
	"snowflake"
	"strings"
)

func handleJoin(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	token := strField(body, "token")
	if token == "" {
		writeError(w, r, http.StatusBadRequest, "邀请链接无效")
		return
	}

	var invID, workspaceID, memberName, pendingRaw, pendingRoleRaw, linkKind string
	var isAdmin, isAccepted, maxUses, useCount int
	err = db.QueryRow(`
		SELECT id, is_admin, workspace_id, COALESCE(company_member_name,''), is_accepted,
		       COALESCE(CAST(pending_grants AS CHAR), ''), COALESCE(CAST(pending_role_names AS CHAR), ''),
		       COALESCE(link_kind,'single'), max_uses, use_count
		FROM tenant_invitation
		WHERE invitation_token=? AND company_id=? AND invitation_token_expires_at > CURRENT_TIMESTAMP`,
		token, tenantID,
	).Scan(&invID, &isAdmin, &workspaceID, &memberName, &isAccepted, &pendingRaw, &pendingRoleRaw, &linkKind, &maxUses, &useCount)
	if err != nil || isAccepted == 1 || !inviteHasRemainingUses(maxUses, useCount) {
		writeError(w, r, http.StatusBadRequest, "邀请链接无效或已过期")
		return
	}

	existing, _ := getMember(userID, tenantID)
	if existing != nil {
		writeError(w, r, http.StatusBadRequest, "您已在公司中")
		return
	}

	memberName = resolveJoinMemberName(body, memberName)
	memberID := snowflake.GenerateIDString()
	tx, err := db.Begin()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "加入失败")
		return
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.Exec(`
		INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active, workspace_id, member_name, created_at, updated_at)
		VALUES (?,?,?,?,1,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		memberID, userID, tenantID, isAdmin, workspaceID, memberName,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			writeError(w, r, http.StatusBadRequest, "您已在公司中")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "加入失败")
		return
	}
	_, err = tx.Exec(`
		INSERT INTO tenant_invitation_redemption
		(id, invitation_id, company_id, user_id, member_id, created_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)`,
		snowflake.GenerateIDString(), invID, tenantID, userID, memberID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			writeError(w, r, http.StatusBadRequest, "您已在公司中")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "加入失败")
		return
	}
	res, err := tx.Exec(`
		UPDATE tenant_invitation
		SET use_count = use_count + 1, updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND company_id=? AND invitation_token=? AND invitation_token != ''
		  AND (max_uses = 0 OR use_count < max_uses)`,
		invID, tenantID, token,
	)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "加入失败")
		return
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		writeError(w, r, http.StatusBadRequest, "邀请链接无效或已过期")
		return
	}
	var newCount int
	if err := tx.QueryRow(`SELECT use_count FROM tenant_invitation WHERE id=?`, invID).Scan(&newCount); err != nil {
		writeError(w, r, http.StatusInternalServerError, "加入失败")
		return
	}
	exhausted := maxUses > 0 && newCount >= maxUses
	if exhausted {
		_, err = tx.Exec(`
			UPDATE tenant_invitation
			SET is_accepted=1, invitation_token=NULL, invitation_token_expires_at=NULL, updated_at=CURRENT_TIMESTAMP
			WHERE id=?`, invID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "加入失败")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, r, http.StatusInternalServerError, "加入失败")
		return
	}

	applyInviteJoinRoles(memberID, tenantID, invID, memberName, isAdmin, pendingRaw, pendingRoleRaw)
	incrMembershipRev(userID)
	ensureWorkspaceAccess(r.Context(), tenantID, workspaceID, memberID, isAdmin == 1)

	role := "member"
	if isAdmin == 1 {
		role = "admin"
	}
	extra := map[string]interface{}{
		"invitation_id": invID, "link_kind": linkKind,
		"use_count": newCount, "invitation_exhausted": exhausted,
	}
	if grants, gErr := decodePendingGrantsJSON(pendingRaw); gErr == nil && len(grants) > 0 && isAdmin == 0 {
		extra["grants"] = grants
	}
	if roleNames, rErr := decodePendingRoleNamesJSON(pendingRoleRaw); rErr == nil && len(roleNames) > 0 && isAdmin == 0 {
		extra["role_names"] = roleNames
	}
	slog.Info("invite_join",
		"level", "info",
		"link_kind", linkKind,
		"use_count", newCount,
		"invitation_exhausted", exhausted,
		"token_fp", inviteTokenFingerprint(token),
	)
	publishMemberJoined(memberID, userID, tenantID, memberName, role, workspaceID, "invite_join", extra)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "成功加入公司",
		"member": map[string]interface{}{
			"id": memberID, "user_id": userID, "company_id": tenantID,
			"is_admin": isAdmin == 1, "member_name": memberName, "workspace_id": workspaceID,
		},
	})
}

func applyInviteJoinRoles(memberID, tenantID, invID, memberName string, isAdmin int, pendingRaw, pendingRoleRaw string) {
	if isAdmin == 1 {
		ensureMemberRole(memberID, "tenant_admin", tenantID, "system")
		return
	}
	roleNames, rErr := decodePendingRoleNamesJSON(pendingRoleRaw)
	if rErr != nil {
		logWarn("decode pending_role_names failed: "+rErr.Error(), "invitation_id="+invID)
	}
	for _, rn := range roleNames {
		ensureMemberRole(memberID, rn, tenantID, "system")
	}
	grants, gErr := decodePendingGrantsJSON(pendingRaw)
	if gErr != nil {
		logWarn("decode pending_grants failed: "+gErr.Error(), "invitation_id="+invID)
		return
	}
	if len(grants) == 0 {
		return
	}
	roleName, aErr := applyPendingGrantsViaAuth(tenantID, accessRoleDisplayNameForInvite(memberName), grants)
	if aErr != nil {
		logWarn("apply pending grants failed: "+aErr.Error(), "invitation_id="+invID+" member_id="+memberID)
		return
	}
	if roleName != "" {
		ensureMemberRole(memberID, roleName, tenantID, "system")
	}
}
