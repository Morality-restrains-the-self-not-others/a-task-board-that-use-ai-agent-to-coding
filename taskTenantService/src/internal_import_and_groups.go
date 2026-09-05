package main

import (
	"database/sql"
	"log"
	"net/http"
	"snowflake"
	"strings"
)

func handleImportMembers(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid json")
		return
	}
	raw, _ := body["items"].([]interface{})
	n := 0
	for _, it := range raw {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		id := strField(m, "id")
		if id == "" {
			id = snowflake.GenerateIDString()
		}
		isAdmin, isActive := 0, 1
		if boolField(m, "is_admin") {
			isAdmin = 1
		}
		if v, ok := m["is_active"]; ok {
			if b, ok := v.(bool); ok && !b {
				isActive = 0
			}
		}
		_, err := db.Exec(`REPLACE INTO tenant_company_member
			(id, user_id, company_id, is_admin, is_active, workspace_id, member_name, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),CURRENT_TIMESTAMP)`,
			id, strField(m, "user_id"), strField(m, "company_id"), isAdmin, isActive,
			strField(m, "workspace_id"), strField(m, "member_name"), normalizeDatetimeString(strField(m, "created_at")))
		if err == nil {
			n++
		}
	}
	writeJSON(w, 200, map[string]int{"imported": n})
}

func handleImportInvitations(w http.ResponseWriter, r *http.Request) {
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, _ := readJSONBody(r)
	raw, _ := body["items"].([]interface{})
	n := 0
	for _, it := range raw {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		id := strField(m, "id")
		if id == "" {
			id = snowflake.GenerateIDString()
		}
		isAdmin, accepted := 0, 0
		if boolField(m, "is_admin") {
			isAdmin = 1
		}
		if boolField(m, "is_accepted") {
			accepted = 1
		}
		token := strField(m, "invitation_token")
		var tokenArg interface{} = token
		if token == "" {
			tokenArg = nil
		}
		// 导入时按渠道回填投递状态：email/phone 视作待异步投递，link 无投递动作
		importMethod := strField(m, "invite_method")
		importDeliveryStatus := "none"
		if importMethod == "email" || importMethod == "phone" {
			importDeliveryStatus = "queued"
		}
		_, err := db.Exec(`REPLACE INTO tenant_invitation
			(id, company_id, is_admin, workspace_id, invite_method, invite_target,
			 invitation_token, invitation_token_expires_at, is_accepted, company_member_name,
			 delivery_status, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),CURRENT_TIMESTAMP)`,
			id, strField(m, "company_id"), isAdmin, strField(m, "workspace_id"),
			importMethod, strField(m, "invite_target"),
			tokenArg, nullStr(normalizeDatetimeString(strField(m, "invitation_token_expires_at"))), accepted,
			strField(m, "company_member_name"), importDeliveryStatus,
			normalizeDatetimeString(strField(m, "created_at")))
		if err == nil {
			n++
		}
	}
	writeJSON(w, 200, map[string]int{"imported": n})
}

func handleImportGroups(w http.ResponseWriter, r *http.Request) {
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, _ := readJSONBody(r)
	groups, _ := body["groups"].([]interface{})
	members, _ := body["members"].([]interface{})
	ng, nm := 0, 0
	for _, it := range groups {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		id := strField(m, "id")
		if id == "" {
			id = snowflake.GenerateIDString()
		}
		_, err := db.Exec(`REPLACE INTO tenant_company_group
			(id, name, description, company_id, created_by_id, created_at, updated_at)
			VALUES (?,?,?,?,?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),CURRENT_TIMESTAMP)`,
			id, strField(m, "name"), strField(m, "description"), strField(m, "company_id"),
			strField(m, "created_by_id"), normalizeDatetimeString(strField(m, "created_at")))
		if err == nil {
			ng++
		}
	}
	for _, it := range members {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		id := strField(m, "id")
		if id == "" {
			id = snowflake.GenerateIDString()
		}
		_, err := db.Exec(`REPLACE INTO tenant_company_group_member
			(id, group_id, user_id, created_at) VALUES (?,?,?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP))`,
			id, strField(m, "group_id"), strField(m, "user_id"), normalizeDatetimeString(strField(m, "created_at")))
		if err == nil {
			nm++
		}
	}
	writeJSON(w, 200, map[string]int{"groups": ng, "members": nm})
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func handleInternalGroups(w http.ResponseWriter, r *http.Request) {
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/tenant/groups")
	path = strings.Trim(path, "/")

	switch {
	case path == "user-in-group" && r.Method == http.MethodGet:
		uid := r.URL.Query().Get("user_id")
		gid := r.URL.Query().Get("group_id")
		if uid == "" || gid == "" {
			writeError(w, r, 400, "user_id and group_id required")
			return
		}
		var n int
		err := db.QueryRow(`SELECT COUNT(1) FROM tenant_company_group_member WHERE group_id=? AND user_id=?`, gid, uid).Scan(&n)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]bool{"in_group": n > 0})
	case path == "members" && r.Method == http.MethodGet:
		gid := r.URL.Query().Get("group_id")
		if gid == "" {
			writeError(w, r, 400, "group_id required")
			return
		}
		rows, err := db.Query(`SELECT user_id FROM tenant_company_group_member WHERE group_id=?`, gid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		defer rows.Close()
		userIDs := make([]string, 0)
		for rows.Next() {
			var uid string
			if err := rows.Scan(&uid); err == nil && uid != "" {
				userIDs = append(userIDs, uid)
			}
		}
		writeJSON(w, 200, map[string]interface{}{"user_ids": userIDs})
	case path == "user-in-admin-groups" && r.Method == http.MethodGet:
		// 组管理员作用域判定（OPT-20260812-029）：目标用户是否在 actor 所管小组内。
		adminUID := r.URL.Query().Get("admin_user_id")
		uid := r.URL.Query().Get("user_id")
		cid := r.URL.Query().Get("company_id")
		if adminUID == "" || uid == "" || cid == "" {
			writeError(w, r, 400, "admin_user_id, user_id and company_id required")
			return
		}
		var n int
		err := db.QueryRow(`
			SELECT COUNT(1)
			FROM tenant_company_group_member m
			JOIN tenant_group_admin ga ON ga.group_id = m.group_id
			WHERE ga.user_id = ? AND ga.company_id = ? AND m.user_id = ?`,
			adminUID, cid, uid).Scan(&n)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]bool{"in_admin_group": n > 0})
	case path == "by-id" && r.Method == http.MethodGet:
		gid := r.URL.Query().Get("group_id")
		cid := r.URL.Query().Get("company_id")
		if gid == "" {
			writeError(w, r, 400, "group_id required")
			return
		}
		var id, name, desc, company, createdBy, created string
		var err error
		if cid != "" {
			err = db.QueryRow(`SELECT id, name, COALESCE(description,''), company_id, COALESCE(created_by_id,''), created_at
				FROM tenant_company_group WHERE id=? AND company_id=?`, gid, cid).Scan(&id, &name, &desc, &company, &createdBy, &created)
		} else {
			err = db.QueryRow(`SELECT id, name, COALESCE(description,''), company_id, COALESCE(created_by_id,''), created_at
				FROM tenant_company_group WHERE id=?`, gid).Scan(&id, &name, &desc, &company, &createdBy, &created)
		}
		if err == sql.ErrNoRows {
			writeError(w, r, 404, "not found")
			return
		}
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{
			"id": id, "name": name, "description": desc, "company_id": company,
			"created_by_id": createdBy, "created_at": created,
		})
	default:
		writeError(w, r, 404, "not found")
	}
}

// handleInvitationDeliveryCallback receives delivery confirmation from the
// taskEvents INVITATION_CREATED consumer after it sends the invitation
// email/SMS. Closes the async delivery loop for tenant invitations:
//
//	delivered → email_sent_at stamped; failed → delivery_error recorded.
//
// POST /api/internal/tenant/invitations/delivery-callback/{id}/
func handleInvitationDeliveryCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 6 || parts[3] != "invitations" || parts[4] != "delivery-callback" {
		writeError(w, r, http.StatusBadRequest, "invalid path")
		return
	}
	inviteID := parts[len(parts)-1]
	if inviteID == "" {
		writeError(w, r, http.StatusBadRequest, "缺少邀请ID")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	status := strings.TrimSpace(strField(body, "status"))
	errMsg := strings.TrimSpace(strField(body, "error_message"))
	if status != "delivered" && status != "failed" && status != "skipped_unsubscribed" {
		writeError(w, r, http.StatusBadRequest, "status must be delivered, failed, or skipped_unsubscribed")
		return
	}
	var companyID string
	err = db.QueryRow(`SELECT company_id FROM tenant_invitation WHERE id=?`, inviteID).Scan(&companyID)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "邀请记录不存在")
		return
	}
	if status == "delivered" {
		_, err = db.Exec(`
			UPDATE tenant_invitation
			SET delivery_status='delivered', delivery_error=NULL, email_sent_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP
			WHERE id=?`, inviteID)
	} else if status == "skipped_unsubscribed" {
		_, err = db.Exec(`
			UPDATE tenant_invitation
			SET delivery_status='skipped_unsubscribed', delivery_error=NULL, updated_at=CURRENT_TIMESTAMP
			WHERE id=?`, inviteID)
	} else {
		_, err = db.Exec(`
			UPDATE tenant_invitation
			SET delivery_status='failed', delivery_error=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=?`, errMsg, inviteID)
	}
	if err != nil {
		log.Printf("[taskTenantService] delivery callback db update failed invite=%s: %v", inviteID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "delivery status updated", "invitation_id": inviteID, "status": status,
	})
}
