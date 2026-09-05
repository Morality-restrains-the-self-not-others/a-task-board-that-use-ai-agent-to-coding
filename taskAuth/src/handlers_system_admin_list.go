package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// handleSystemAdminListUsers returns a paginated list of all users.
// Same implementation as internal handleListUsers but uses superuser auth.
func handleSystemAdminListUsers(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	limit := parseIntParam(r, "limit", 50, 200)
	offset := parseIntParam(r, "offset", 0, 100000)
	searchQ := strings.TrimSpace(r.URL.Query().Get("q"))
	archivedFilter := strings.TrimSpace(r.URL.Query().Get("is_archived"))
	colFilters := parseSystemAdminUserListFilters(r)

	var matchedIDs []string
	if searchQ != "" {
		matchedIDs = searchUserIDs(searchQ)
		if len(matchedIDs) == 0 {
			writeJSON(w, 200, map[string]interface{}{"users": []interface{}{}, "total": 0})
			return
		}
	}

	identifierLookup := searchQ != "" || colFilters.phone != "" || colFilters.email != ""
	whereClauses, whereArgs := colFilters.localWhere(matchedIDs, archivedFilter, identifierLookup)
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	var total int64
	var query string
	var queryArgs []interface{}
	if colFilters.needsEnrichment() {
		ids, err := querySystemAdminUserIDs(whereSQL, whereArgs, maxSystemAdminListEnrichmentScan)
		if err != nil {
			writeErrorDetail(w, r, 500, "db error")
			return
		}
		ids = applySystemAdminListEnrichment(r.Context(), ids, colFilters)
		total = int64(len(ids))
		pageIDs := paginateIDs(ids, offset, limit)
		if len(pageIDs) == 0 {
			slog.InfoContext(r.Context(), "system_admin_users_list",
				"filter_keys", colFilters.keys(), "total", total,
				"duration_ms", time.Since(started).Milliseconds())
			writeJSON(w, 200, map[string]interface{}{"users": []interface{}{}, "total": total})
			return
		}
		placeholders := make([]string, len(pageIDs))
		queryArgs = make([]interface{}, len(pageIDs))
		for i, id := range pageIDs {
			placeholders[i] = "?"
			queryArgs[i] = id
		}
		query = `SELECT id, is_active, is_superuser, is_staff, COALESCE(is_tenant, 0), COALESCE(is_tester, 0), date_joined, COALESCE(is_archived, 0), last_login
			FROM auth_user WHERE id IN (` + strings.Join(placeholders, ",") + `)
			ORDER BY date_joined DESC`
	} else {
		countQuery := "SELECT COUNT(*) FROM auth_user " + whereSQL
		if err := db.QueryRow(countQuery, whereArgs...).Scan(&total); err != nil {
			writeErrorDetail(w, r, 500, "db error")
			return
		}
		queryArgs = append(append([]interface{}{}, whereArgs...), limit, offset)
		query = `SELECT id, is_active, is_superuser, is_staff, COALESCE(is_tenant, 0), COALESCE(is_tester, 0), date_joined, COALESCE(is_archived, 0), last_login
			FROM auth_user ` + whereSQL + `
			ORDER BY date_joined DESC LIMIT ? OFFSET ?`
	}
	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		writeErrorDetail(w, r, 500, "db error")
		return
	}
	defer rows.Close()

	type userRow struct {
		id          string
		isActive    bool
		isSuperuser bool
		isStaff     bool
		isTenant    bool
		isTester    bool
		dateJoined  string
		isArchived  bool
		lastLogin   string
	}
	var userRows []userRow
	for rows.Next() {
		var u userRow
		var dj, ll sql.NullString
		if err := rows.Scan(&u.id, &u.isActive, &u.isSuperuser, &u.isStaff, &u.isTenant, &u.isTester, &dj, &u.isArchived, &ll); err != nil {
			writeErrorDetail(w, r, 500, "db error")
			return
		}
		if dj.Valid {
			u.dateJoined = dj.String
		}
		if ll.Valid {
			u.lastLogin = ll.String
		}
		userRows = append(userRows, u)
	}

	userIDs := make([]string, len(userRows))
	for i, u := range userRows {
		userIDs[i] = u.id
	}
	loginMethodsByUser := batchLoginMethods(userIDs)
	profileUsernames := batchProfileUsernames(userIDs)

	users := make([]map[string]interface{}, 0, len(userRows))
	for _, u := range userRows {
		lms := loginMethodsByUser[u.id]
		userMap := map[string]interface{}{
			"id":            u.id,
			"is_active":     u.isActive,
			"is_superuser":  u.isSuperuser,
			"is_staff":      u.isStaff,
			"is_tenant":     u.isTenant,
			"is_tester":     u.isTester,
			"is_archived":   u.isArchived,
			"date_joined":   u.dateJoined,
			"last_login":    u.lastLogin,
			"login_methods": lms,
			"referrer_code": "",
		}
		// Lift email/phone/username so the admin UI 邮箱/手机号 columns render
		// real identifiers instead of falling back to the raw user id
		// (regression: the seeded superadmin id "bootstrap-admin" was shown
		// in place of author@example.com).
		for k, v := range loginMethodProfileFields(lms) {
			userMap[k] = v
		}
		// username 统一取 auth_user_profile.username（与 buildUserDetailJSON 一致，
		// OPT-20260807-010）；profile 未设置时回退登录方式 identifier。
		if uname := profileUsernames[u.id]; uname != "" {
			userMap["username"] = uname
		}
		users = append(users, userMap)
	}

	// OPT-20260821-031：批量回填「推荐人」列。best-effort 走 taskReferral 内部接口
	// （禁止 taskAuth 直连 task_bill）；服务不可达时 referrer_code 保持空串，前端显示 —。
	if len(userIDs) > 0 {
		refs := fetchReferrersBatch(r.Context(), userIDs)
		if len(refs) > 0 {
			referrerIDs := make([]string, 0, len(refs))
			seen := map[string]bool{}
			for _, e := range refs {
				rid := strings.TrimSpace(e["referrer_user_id"])
				if rid != "" && !seen[rid] {
					seen[rid] = true
					referrerIDs = append(referrerIDs, rid)
				}
			}
			referrerProfiles := batchProfileUsernames(referrerIDs)
			referrerLogins := batchLoginMethods(referrerIDs)
			for _, u := range users {
				uid, _ := u["id"].(string)
				e, ok := refs[uid]
				if !ok {
					continue
				}
				u["referrer_code"] = referrerDisplayName(e["referrer_user_id"], referrerProfiles, referrerLogins)
			}
		}
	}

	// 批量回填「所属租户公司」。best-effort 走 taskTenantService members/batch-get
	// （禁止 taskAuth 直连 tenant_* 表）；服务不可达时 tenant_companies 为空数组。
	if len(userIDs) > 0 {
		attachTenantCompanies(users, fetchTenantCompaniesBatch(r.Context(), userIDs))
		quals, ok := fetchQualificationsBatch(r.Context(), userIDs)
		attachProfitSharingQualifications(users, quals, ok)
	} else {
		attachTenantCompanies(users, nil)
		attachProfitSharingQualifications(users, nil, true)
	}

	slog.InfoContext(r.Context(), "system_admin_users_list",
		"filter_keys", colFilters.keys(), "total", total,
		"duration_ms", time.Since(started).Milliseconds())
	writeJSON(w, 200, map[string]interface{}{"users": users, "total": total})
}

// loginMethodProfileFields lifts email/phone/username identifiers out of a
// user's login methods into the top-level fields the admin UI expects
// (UserListRow renders user.email first, then phone/username, then id).
// Unknown method types are ignored so no spurious JSON keys leak through.
func loginMethodProfileFields(loginMethods []map[string]interface{}) map[string]interface{} {
	fields := map[string]interface{}{
		"email":    "",
		"phone":    "",
		"username": "",
	}
	for _, lm := range loginMethods {
		methodType, _ := lm["method_type"].(string)
		identifier, _ := lm["identifier"].(string)
		if identifier == "" {
			continue
		}
		switch methodType {
		case "email", "phone", "username":
			if fields[methodType] == "" {
				fields[methodType] = identifier
			}
		}
	}
	return fields
}
