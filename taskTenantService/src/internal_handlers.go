package main

import (
	"fmt"
	"net/http"
	"snowflake"
	"strings"
)

func checkInternalSecret(r *http.Request) bool {
	if cfg.InternalSecret == "" {
		return true
	}
	return r.Header.Get("X-Internal-Secret") == cfg.InternalSecret
}

func getMemberByIDOnly(id, companyID string) (*memberRow, error) {
	if companyID != "" {
		return getMemberByID(id, companyID)
	}
	row := db.QueryRow(`SELECT `+memberSelectCols+` FROM tenant_company_member WHERE id=?`, id)
	return scanMember(row)
}

func handleInternalMembers(w http.ResponseWriter, r *http.Request) {
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/tenant/members")
	path = strings.Trim(path, "/")

	switch {
	case path == "resolve" && r.Method == http.MethodGet:
		uid := r.URL.Query().Get("user_id")
		cid := r.URL.Query().Get("company_id")
		m, err := getMember(uid, cid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if m == nil {
			writeError(w, r, 404, "not found")
			return
		}
		writeJSON(w, 200, memberToJSON(m))
	case path == "display-name" && r.Method == http.MethodGet:
		// 合并审计等跨服务场景需要「对外展示的公司成员名」（误种子自愈逻辑与
		// 人员页一致，见 resolveMemberDisplayName / healMisSeededMemberName）。
		uid := r.URL.Query().Get("user_id")
		cid := r.URL.Query().Get("company_id")
		if uid == "" || cid == "" {
			writeError(w, r, 400, "user_id and company_id required")
			return
		}
		m, err := getMember(uid, cid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if m == nil {
			writeError(w, r, 404, "not found")
			return
		}
		var companyName string
		_ = db.QueryRow(`SELECT COALESCE(name,'') FROM tenant_company WHERE id=?`, cid).Scan(&companyName)
		nick := fetchPersonalNicknamesFn([]string{uid})[uid]
		display := resolveMemberDisplayName(m.MemberName, companyName, nick, uid)
		if isMisSeededMemberName(m.MemberName, companyName) && strings.TrimSpace(nick) != "" {
			healMisSeededMemberName(m.ID, nick)
		}
		writeJSON(w, 200, map[string]interface{}{
			"user_id":      uid,
			"company_id":   cid,
			"member_id":    m.ID,
			"display_name": display,
		})
	case path == "by-id" && r.Method == http.MethodGet:
		mid := r.URL.Query().Get("member_id")
		cid := r.URL.Query().Get("company_id")
		if mid == "" {
			writeError(w, r, 400, "member_id required")
			return
		}
		m, err := getMemberByIDOnly(mid, cid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if m == nil {
			writeError(w, r, 404, "not found")
			return
		}
		writeJSON(w, 200, memberToJSON(m))
	case path == "exists" && r.Method == http.MethodGet:
		uid := r.URL.Query().Get("user_id")
		if uid == "" {
			writeError(w, r, 400, "user_id required")
			return
		}
		var n int
		_ = db.QueryRow(`SELECT COUNT(1) FROM tenant_company_member WHERE user_id=?`, uid).Scan(&n)
		writeJSON(w, 200, map[string]bool{"exists": n > 0})
	case path == "workspace" && r.Method == http.MethodPatch:
		body, _ := readJSONBody(r)
		uid := strField(body, "user_id")
		cid := strField(body, "company_id")
		ws := strField(body, "workspace_id")
		if uid == "" || cid == "" || ws == "" {
			writeError(w, r, 400, "user_id, company_id, workspace_id required")
			return
		}
		res, err := db.Exec(`UPDATE tenant_company_member SET workspace_id=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND company_id=?`, ws, uid, cid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			writeError(w, r, 404, "not found")
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true, "workspace_id": ws})
	case path == "batch-get" && r.Method == http.MethodPost:
		handleInternalMembersBatchGet(w, r)
	case path == "avatar" && (r.Method == http.MethodPost || r.Method == http.MethodDelete):
		handleInternalMemberAvatar(w, r)
	case path == "update" && r.Method == http.MethodPatch:
		body, _ := readJSONBody(r)
		uid := strField(body, "user_id")
		cid := strField(body, "company_id")
		if uid == "" || cid == "" {
			writeError(w, r, 400, "user_id, company_id required")
			return
		}
		ws := strField(body, "workspace_id")
		name := strField(body, "member_name")
		if ws == "" && name == "" {
			writeError(w, r, 400, "nothing to update")
			return
		}
		if ws != "" && name != "" {
			_, err := db.Exec(`UPDATE tenant_company_member SET workspace_id=?, member_name=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND company_id=?`, ws, name, uid, cid)
			if err != nil {
				writeError(w, r, 500, err.Error())
				return
			}
		} else if ws != "" {
			_, err := db.Exec(`UPDATE tenant_company_member SET workspace_id=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND company_id=?`, ws, uid, cid)
			if err != nil {
				writeError(w, r, 500, err.Error())
				return
			}
		} else {
			_, err := db.Exec(`UPDATE tenant_company_member SET member_name=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND company_id=?`, name, uid, cid)
			if err != nil {
				writeError(w, r, 500, err.Error())
				return
			}
		}
		m, _ := getMember(uid, cid)
		if m == nil {
			writeError(w, r, 404, "not found")
			return
		}
		writeJSON(w, 200, memberToJSON(m))
	case path == "sync-nickname" && r.Method == http.MethodPatch:
		// 写路径回填误种子/空 member_name（OPT-20260812-038）：taskAuth 在个人
		// 昵称写入后同步创建者成员名，不依赖读路径自愈。仅更新误种子行，不覆盖
		// 用户已设置的真实成员名。
		body, _ := readJSONBody(r)
		uid := strField(body, "user_id")
		nickname := strings.TrimSpace(strField(body, "nickname"))
		if uid == "" || nickname == "" {
			writeError(w, r, 400, "user_id and nickname required")
			return
		}
		res, err := db.Exec(`
			UPDATE tenant_company_member m
			JOIN tenant_company c ON c.id = m.company_id
			SET m.member_name = ?, m.updated_at = CURRENT_TIMESTAMP
			WHERE m.user_id = ?
			  AND (TRIM(COALESCE(m.member_name, '')) = ''
			       OR m.member_name = ?
			       OR m.member_name = c.name)`, nickname, uid, defaultPersonalCompanyName)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			// 成员名变化 → PDP 缓存失效（与读路径 heal 一致）
			incrMembershipRev(uid)
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true, "updated": n})
	case path == "" && r.Method == http.MethodGet:
		uid := r.URL.Query().Get("user_id")
		cid := r.URL.Query().Get("company_id")
		if uid != "" {
			list, err := listMembersByUser(uid)
			if err != nil {
				writeError(w, r, 500, err.Error())
				return
			}
			writeJSON(w, 200, list)
			return
		}
		if cid == "" {
			writeError(w, r, 400, "company_id or user_id required")
			return
		}
		rows, err := db.Query(`SELECT `+memberSelectCols+`
			FROM tenant_company_member WHERE company_id=? ORDER BY created_at ASC`, cid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		defer rows.Close()
		list := make([]map[string]interface{}, 0)
		for rows.Next() {
			m, err := scanMember(rows)
			if err != nil || m == nil {
				continue
			}
			list = append(list, memberToJSON(m))
		}
		writeJSON(w, 200, list)
	case path == "" && r.Method == http.MethodPost:
		body, _ := readJSONBody(r)
		id := strField(body, "id")
		if id == "" {
			id = snowflake.GenerateIDString()
		}
		uid := strField(body, "user_id")
		cid := strField(body, "company_id")
		isAdmin := 0
		if boolField(body, "is_admin") {
			isAdmin = 1
		}
		isActive := 1
		if v, ok := body["is_active"]; ok {
			if b, ok := v.(bool); ok && !b {
				isActive = 0
			}
		}
		memberName := strField(body, "member_name")
		workspaceID := strField(body, "workspace_id")
		// Detect new vs replace so MEMBER_JOINED only fires on first create.
		var existingID string
		_ = db.QueryRow(`SELECT id FROM tenant_company_member WHERE user_id=? AND company_id=? LIMIT 1`, uid, cid).Scan(&existingID)
		isNew := existingID == ""
		if existingID != "" {
			id = existingID
		}
		_, err := db.Exec(`REPLACE INTO tenant_company_member
			(id, user_id, company_id, is_admin, is_active, workspace_id, member_name, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
			id, uid, cid, isAdmin, isActive, workspaceID, memberName)
		if err != nil {
			writeError(w, r, 400, err.Error())
			return
		}
		// v63 硬切换: 内部创建管理员成员 → 同步落 tenant_admin 角色行
		if isAdmin == 1 {
			ensureMemberRole(id, "tenant_admin", cid, "system")
		}
		// 事件失效: 内部建成员（事件驱动公司创建/修复）→ incr rev 使 PDP 缓存失效
		incrMembershipRev(uid)
		if isNew {
			role := "member"
			if isAdmin == 1 {
				role = "admin"
			}
			go publishMemberJoined(id, uid, cid, memberName, role, workspaceID, "internal_upsert", nil)
		}
		writeJSON(w, 201, map[string]string{"id": id})
	case path == "import" && r.Method == http.MethodPost:
		handleImportMembers(w, r)
	default:
		writeError(w, r, 404, "not found")
	}
}

// handleInternalMembersBatchGet — POST /api/internal/tenant/members/batch-get/
// Body: {"member_ids": ["id1","id2",...]} or {"user_ids": ["uid1","uid2",...]}
// member_ids queries by tenant_company_member.id (member PK); user_ids queries by m.user_id (user account ID).
// Returns: {"members": [{id, user_id, company_id, is_admin, is_active, workspace_id, member_name, member_avatar, member_avatar_url, created_at, company_name}, ...]}
// Max 500 ids per request.
func handleInternalMembersBatchGet(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}

	// Support both member_ids (query by m.id) and user_ids (query by m.user_id)
	rawIDs, _ := body["user_ids"].([]interface{})
	queryCol := "m.user_id"
	if len(rawIDs) == 0 {
		rawIDs, _ = body["member_ids"].([]interface{})
		queryCol = "m.id"
	}
	if len(rawIDs) == 0 {
		writeJSON(w, 200, map[string]interface{}{"members": []interface{}{}})
		return
	}

	// Deduplicate and collect
	ids := make([]string, 0, len(rawIDs))
	seen := map[string]bool{}
	for _, raw := range rawIDs {
		mid := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if mid == "" || mid == "<nil>" || seen[mid] {
			continue
		}
		seen[mid] = true
		ids = append(ids, mid)
		if len(ids) >= 500 {
			break
		}
	}
	if len(ids) == 0 {
		writeJSON(w, 200, map[string]interface{}{"members": []interface{}{}})
		return
	}

	// Build IN query
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	rows, err := db.Query(`SELECT m.id, m.user_id, m.company_id, m.is_admin, m.is_active,
			COALESCE(m.workspace_id,''), COALESCE(m.member_name,''), COALESCE(m.member_avatar,''), m.created_at,
			COALESCE(c.name, '') AS company_name
		FROM tenant_company_member m
		LEFT JOIN tenant_company c ON c.id = m.company_id
		WHERE `+queryCol+` IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer rows.Close()

	members := make([]map[string]interface{}, 0, len(ids))
	for rows.Next() {
		var id, uid, cid, wsID, memberName, memberAvatar, createdAt, companyName string
		var isAdmin, isActive int
		if err := rows.Scan(&id, &uid, &cid, &isAdmin, &isActive, &wsID, &memberName, &memberAvatar, &createdAt, &companyName); err != nil {
			continue
		}
		members = append(members, map[string]interface{}{
			"id":                id,
			"user_id":           uid,
			"company_id":        cid,
			"is_admin":          isAdmin == 1,
			"is_active":         isActive == 1,
			"workspace_id":      wsID,
			"member_name":       memberName,
			"member_avatar":     memberAvatar,
			"member_avatar_url": memberAvatarPublicURL(cid, id, memberAvatar),
			"created_at":        createdAt,
			"company_name":      companyName,
		})
	}
	if members == nil {
		members = []map[string]interface{}{}
	}
	writeJSON(w, 200, map[string]interface{}{"members": members})
}
