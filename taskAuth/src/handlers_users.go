package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// handleCountUsers returns the total number of users in auth_user.
// Internal endpoint — requires X-TaskAuth-Internal-Secret.
func handleCountUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}

	var count int64
	err := db.QueryRow(`SELECT COUNT(*) FROM auth_user`).Scan(&count)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": count})
}

// handleListUsers returns a paginated list of all users with their login methods.
// Internal endpoint — requires X-TaskAuth-Internal-Secret.
//
// Query params: limit (default 50, max 200), offset (default 0),
// q (optional search by email/ID/phone), is_archived (optional filter: "true"/"false").
func handleListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}

	limit := parseIntParam(r, "limit", 50, 200)
	offset := parseIntParam(r, "offset", 0, 100000)
	searchQ := strings.TrimSpace(r.URL.Query().Get("q"))
	archivedFilter := strings.TrimSpace(r.URL.Query().Get("is_archived"))

	// Search: find user IDs matching email/ID/phone
	var matchedIDs []string
	if searchQ != "" {
		matchedIDs = searchUserIDs(searchQ)
		if len(matchedIDs) == 0 {
			// No matches — return empty list with total=0
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"users": []interface{}{},
				"total": 0,
			})
			return
		}
	}

	// Build WHERE clause
	whereClauses := []string{}
	whereArgs := []interface{}{}
	if len(matchedIDs) > 0 {
		placeholders := make([]string, len(matchedIDs))
		for i, id := range matchedIDs {
			placeholders[i] = "?"
			whereArgs = append(whereArgs, id)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ",")))
	}
	if archivedFilter == "true" {
		whereClauses = append(whereClauses, "COALESCE(is_archived, 0) = 1")
	} else if archivedFilter == "false" {
		whereClauses = append(whereClauses, "COALESCE(is_archived, 0) = 0")
	}
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Total count
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM auth_user %s", whereSQL)
	if err := db.QueryRow(countQuery, whereArgs...).Scan(&total); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	// Paginated users
	queryArgs := append(whereArgs, limit, offset)
	rows, err := db.Query(fmt.Sprintf(`
		SELECT id, is_active, is_superuser, is_staff, COALESCE(is_tenant, 0), COALESCE(is_tester, 0), date_joined, COALESCE(is_archived, 0), last_login
		FROM auth_user
		%s
		ORDER BY date_joined DESC
		LIMIT ? OFFSET ?`, whereSQL), queryArgs...)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
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
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
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

	// Batch fetch login methods + profile usernames for all returned users
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
		}
		// 与系统管理列表一致：提升 email/phone/username 顶层字段（OPT-20260807-009），
		// username 优先取 auth_user_profile.username（与 buildUserDetailJSON 一致，OPT-20260807-010）。
		for k, v := range loginMethodProfileFields(lms) {
			userMap[k] = v
		}
		if uname := profileUsernames[u.id]; uname != "" {
			userMap["username"] = uname
		}
		users = append(users, userMap)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": total,
	})
}

// searchUserIDs finds user IDs whose email, username, ID, or phone matches the query string.
func searchUserIDs(q string) []string {
	lowerQ := strings.ToLower(q)
	seen := make(map[string]bool)
	var ids []string

	appendMatches := func(query string, args ...interface{}) {
		rows, err := db.Query(query, args...)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if rows.Scan(&id) != nil {
				continue
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}

	// Search by user ID (exact prefix match)
	appendMatches(`SELECT id FROM auth_user WHERE id LIKE ? LIMIT 50`, q+"%")

	// Search by email or phone identifier and by username — prefix match first
	// (uses index where available), fallback to infix (full scan) only when the
	// prefix passes return too few results.
	searchPattern := lowerQ + "%"
	appendMatches(`
		SELECT DISTINCT object_id FROM auth_login_method
		WHERE LOWER(identifier) LIKE ? AND binding_voided_at IS NULL
		LIMIT 50`, searchPattern)
	appendMatches(`
		SELECT user_id FROM auth_user_profile
		WHERE LOWER(username) LIKE ? LIMIT 50`, searchPattern)

	// Infix fallback: if prefix search returned nothing but the user might be
	// searching by a substring (e.g. domain part of email), run the full
	// LIKE '%q%' scan across login identifiers and usernames. Limit to 20
	// results per table to cap scan cost.
	if len(ids) == 0 && len(q) >= 3 {
		infixPattern := "%" + lowerQ + "%"
		appendMatches(`
			SELECT DISTINCT object_id FROM auth_login_method
			WHERE LOWER(identifier) LIKE ? AND binding_voided_at IS NULL
			LIMIT 20`, infixPattern)
		appendMatches(`
			SELECT user_id FROM auth_user_profile
			WHERE LOWER(username) LIKE ? LIMIT 20`, infixPattern)
	}

	return ids
}

// handlePatchUser updates whitelisted fields on a user.
// Internal endpoint — requires X-TaskAuth-Internal-Secret.
//
// Allowed fields: is_active, is_superuser, is_staff.
func handlePatchUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}

	userID := strings.Trim(r.PathValue("user_id"), "/")
	if userID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	// Whitelist: only these fields may be updated
	allowedFields := map[string]bool{
		"is_active":    true,
		"is_superuser": true,
		"is_staff":     true,
		"is_tenant":    true,
		"is_tester":    true,
		"is_archived":  true,
	}

	prevTester, _ := loadUserIsTester(userID)
	forceTenant := testerForcesTenant(body)
	testerInBody := false
	testerVal := false
	if v, ok := body["is_tester"].(bool); ok {
		testerInBody = true
		testerVal = v
	}

	setClauses := []string{}
	args := []interface{}{}
	for key, allowed := range allowedFields {
		if !allowed {
			continue
		}
		if key == "is_tenant" && forceTenant {
			continue
		}
		if key == "is_tester" {
			continue
		}
		val, ok := body[key]
		if !ok {
			continue
		}
		boolVal, isBool := val.(bool)
		if !isBool {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, boolVal)
	}
	if forceTenant {
		setClauses = append(setClauses, "is_tenant = ?")
		args = append(args, true)
	}

	if len(setClauses) == 0 && !testerInBody {
		writeErrorDetail(w, r, http.StatusBadRequest, "no valid fields to update")
		return
	}

	if len(setClauses) > 0 {
		args = append(args, userID)
		query := fmt.Sprintf("UPDATE auth_user SET %s WHERE id = ?", strings.Join(setClauses, ", "))
		_, err = db.Exec(query, args...)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
	}
	if testerInBody {
		if err := setUserTesterFlag(r.Context(), userID, testerVal); err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
	}

	maybePublishTesterFlagChanged(r.Context(), userID, prevTester, body)

	// Return updated user
	payload, err := buildUserDetailJSON(userID)
	if err == sql.ErrNoRows {
		writeErrorDetail(w, r, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// parseIntParam parses an integer query parameter with defaults and max.
func parseIntParam(r *http.Request, name string, defaultVal, maxVal int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val < 0 {
		return defaultVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

// batchLoginMethods fetches login methods for multiple users in one query.
func batchLoginMethods(userIDs []string) map[string][]map[string]interface{} {
	result := make(map[string][]map[string]interface{})
	if len(userIDs) == 0 {
		return result
	}
	for _, uid := range userIDs {
		result[uid] = []map[string]interface{}{}
	}

	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))
	for i, uid := range userIDs {
		placeholders[i] = "?"
		args[i] = uid
	}

	query := fmt.Sprintf(`
		SELECT object_id, method_type, identifier, is_verified
		FROM auth_login_method
		WHERE object_id IN (%s) AND binding_voided_at IS NULL
		ORDER BY method_type`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var objectID, methodType, identifier string
		var isVerified bool
		if err := rows.Scan(&objectID, &methodType, &identifier, &isVerified); err != nil {
			continue
		}
		result[objectID] = append(result[objectID], map[string]interface{}{
			"method_type": methodType,
			"identifier":  identifier,
			"is_verified": isVerified,
		})
	}
	return result
}

// batchProfileUsernames 批量取 user_id → auth_user_profile.username。
// 与 buildUserDetailJSON 一致：username 顶层字段优先取 profile 用户名，
// 而非 username 登录方式 identifier（OPT-20260807-010）。
func batchProfileUsernames(userIDs []string) map[string]string {
	result := make(map[string]string, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}
	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))
	for i, uid := range userIDs {
		placeholders[i] = "?"
		args[i] = uid
	}
	rows, err := db.Query(
		`SELECT user_id, COALESCE(username, '') FROM auth_user_profile WHERE user_id IN (`+
			strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var uid, username string
		if err := rows.Scan(&uid, &username); err != nil {
			continue
		}
		result[uid] = username
	}
	return result
}
