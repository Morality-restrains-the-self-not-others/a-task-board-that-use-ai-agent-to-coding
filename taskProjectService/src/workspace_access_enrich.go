package main

import (
	"fmt"
	"strings"
)

func displayName(memberName, userID string) string {
	if n := strings.TrimSpace(memberName); n != "" {
		return n
	}
	uid := strings.TrimSpace(userID)
	if len(uid) > 8 {
		return uid[:8]
	}
	if uid != "" {
		return uid
	}
	return "未设置"
}

func collectAccessUserIDs(accessRows []map[string]interface{}) (map[string]struct{}, error) {
	userIDs := map[string]struct{}{}
	for _, row := range accessRows {
		uid := strings.TrimSpace(strField(row, "user_id"))
		gid := strings.TrimSpace(strField(row, "group_id"))
		if uid != "" {
			userIDs[uid] = struct{}{}
			continue
		}
		if gid == "" {
			continue
		}
		ids, err := tenantGroupMemberUserIDs(gid)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			userIDs[id] = struct{}{}
		}
	}
	return userIDs, nil
}

func memberPermissionsForUser(tenantID, userID string) []map[string]interface{} {
	rows, err := db.Query(
		`SELECT wa.workspace_id, wa.permission, COALESCE(w.name,'')
		 FROM project_workspace_accesses wa
		 LEFT JOIN project_workspace_entries w ON w.id = wa.workspace_id
		 WHERE wa.user_id=? AND w.company_id=?`,
		userID, tenantID,
	)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var wsID, perm, wsName string
		if err := rows.Scan(&wsID, &perm, &wsName); err != nil {
			continue
		}
		if wsName == "" {
			wsName = wsID
		}
		out = append(out, map[string]interface{}{
			"workspace_id":   wsID,
			"workspace_name": wsName,
			"role":           perm,
		})
	}
	return out
}

func serializeCollaboratorMember(tenantID string, m map[string]interface{}) map[string]interface{} {
	userID := strings.TrimSpace(strField(m, "user_id"))
	memberName := strings.TrimSpace(strField(m, "member_name"))
	avatarURL := strings.TrimSpace(strField(m, "member_avatar_url"))
	if avatarURL == "" {
		// Fallback when internal payload has stored path but no public URL.
		stored := strings.TrimSpace(strField(m, "member_avatar"))
		mid := strings.TrimSpace(strField(m, "id"))
		cid := strings.TrimSpace(strField(m, "company_id"))
		if cid == "" {
			cid = strings.TrimSpace(tenantID)
		}
		if stored != "" && mid != "" && cid != "" {
			avatarURL = "/api/tenant/" + cid + "/accounts/members/" + mid + "/avatar"
		}
	}
	return map[string]interface{}{
		"id":                 strField(m, "id"),
		"user":               userID,
		"username":           displayName(memberName, userID),
		"member_name":        memberName,
		"member_avatar_url":  avatarURL,
		"company":            strField(m, "company_id"),
		"is_admin":           m["is_admin"],
		"permissions":        memberPermissionsForUser(tenantID, userID),
		"created_at":         m["created_at"],
	}
}

// buildWorkspaceCollaborators mirrors former Django workspace_collaborators_internal.
func buildWorkspaceCollaborators(tenantID, callerUserID string, accessRows []map[string]interface{}) ([]map[string]interface{}, int, error) {
	callerUserID = strings.TrimSpace(callerUserID)
	if callerUserID == "" {
		return nil, 400, fmt.Errorf("tenant_id and user_id required")
	}
	caller, err := tenantResolveMember(tenantID, callerUserID)
	if err != nil {
		return nil, 502, err
	}
	if caller == nil {
		return nil, 400, fmt.Errorf("您不是任何公司的成员")
	}

	wanted, err := collectAccessUserIDs(accessRows)
	if err != nil {
		return nil, 502, err
	}
	members, err := tenantListMembers(tenantID)
	if err != nil {
		return nil, 502, err
	}
	if len(wanted) == 0 {
		for _, m := range members {
			if uid := strings.TrimSpace(strField(m, "user_id")); uid != "" {
				wanted[uid] = struct{}{}
			}
		}
	}

	payload := make([]map[string]interface{}, 0)
	for _, m := range members {
		uid := strings.TrimSpace(strField(m, "user_id"))
		if uid == "" {
			continue
		}
		if _, ok := wanted[uid]; !ok {
			continue
		}
		payload = append(payload, serializeCollaboratorMember(tenantID, m))
	}
	return payload, 200, nil
}

func enrichWorkspaceAccessRows(tenantID string, accessRows []map[string]interface{}) ([]map[string]interface{}, error) {
	creatorID, err := tenantCompanyCreatorID(tenantID)
	if err != nil {
		// Soft-fail: keep enrich usable if tenant company-creator is briefly unavailable.
		creatorID = ""
	}
	emailCache := map[string]string{}
	out := make([]map[string]interface{}, 0, len(accessRows))
	for _, row := range accessRows {
		enriched := map[string]interface{}{
			"id":         row["id"],
			"workspace":  strField(row, "workspace_id"),
			"user":       nil,
			"group":      nil,
			"role":       strField(row, "permission"),
			"user_info":  nil,
			"group_info": nil,
			"created_at": row["created_at"],
		}
		if role := strField(row, "role"); role != "" && enriched["role"] == "" {
			enriched["role"] = role
		}
		if enriched["role"] == "" {
			enriched["role"] = "viewer"
		}

		uid := strings.TrimSpace(strField(row, "user_id"))
		gid := strings.TrimSpace(strField(row, "group_id"))
		if uid != "" {
			enriched["user"] = uid
			member, err := tenantResolveMember(tenantID, uid)
			if err != nil {
				return nil, err
			}
			memberID := ""
			memberName := ""
			isTenant := false
			if member != nil {
				memberID = strField(member, "id")
				memberName = strings.TrimSpace(strField(member, "member_name"))
				isTenant = creatorID != "" && creatorID == uid
			}
			email, ok := emailCache[uid]
			if !ok {
				email = authGetUserEmail(uid)
				emailCache[uid] = email
			}
			enriched["user_info"] = map[string]interface{}{
				"id":                uid,
				"company_member_id": memberID,
				"member_name":       memberName,
				"username":          displayName(memberName, uid),
				"email":             email,
				"is_tenant":         isTenant,
			}
		} else if gid != "" {
			enriched["group"] = gid
			group, err := tenantGetGroup(gid, tenantID)
			if err != nil {
				return nil, err
			}
			name := ""
			if group != nil {
				name = strField(group, "name")
			}
			ids, err := tenantGroupMemberUserIDs(gid)
			if err != nil {
				return nil, err
			}
			enriched["group_info"] = map[string]interface{}{
				"id":          gid,
				"name":        name,
				"memberCount": len(ids),
			}
		}
		out = append(out, enriched)
	}
	return out, nil
}
