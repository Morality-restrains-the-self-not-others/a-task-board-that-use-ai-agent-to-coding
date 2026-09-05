package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

func handleOAuthTokens(w http.ResponseWriter, r *http.Request, subParts []string) {
	tenantID := getAuthTenant(r)
	if tenantID == "" {
		writeErrorJSON(w, r, 400, "无法获取租户ID")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}

	sub := strings.Trim(strings.Join(subParts, "/"), "/")
	if sub == "" {
		switch r.Method {
		case http.MethodGet:
			tokens, err := listOAuthTokens(tenantID)
			if err != nil {
				writeErrorJSON(w, r, 500, err.Error())
				return
			}
			writeJSON(w, 200, tokens)
		case http.MethodPost:
			body, _ := readJSONBody(r)
			token, err := createOAuthToken(tenantID, body)
			if err != nil {
				if strings.Contains(err.Error(), "unique") {
					writeErrorJSON(w, r, 400, "该租户下已存在相同平台与授权方式的 OAuth Token")
					return
				}
				writeErrorJSON(w, r, 500, err.Error())
				return
			}
			writeJSON(w, 201, token)
		default:
			writeErrorJSON(w, r, 405, "method not allowed")
		}
		return
	}

	tokenID := strings.TrimSuffix(sub, "/")
	switch r.Method {
	case http.MethodGet:
		token, err := getOAuthToken(tenantID, tokenID)
		if err == sql.ErrNoRows {
			writeErrorJSON(w, r, 404, "未找到")
			return
		}
		if err != nil {
			writeErrorJSON(w, r, 500, err.Error())
			return
		}
		writeJSON(w, 200, token)
	case http.MethodPut, http.MethodPatch:
		body, _ := readJSONBody(r)
		token, err := updateOAuthToken(tenantID, tokenID, body)
		if err == sql.ErrNoRows {
			writeErrorJSON(w, r, 404, "未找到")
			return
		}
		if err != nil {
			writeErrorJSON(w, r, 500, err.Error())
			return
		}
		writeJSON(w, 200, token)
	case http.MethodDelete:
		res, err := db.Exec(`DELETE FROM cloud_oauth_tokens WHERE id=? AND company_id=?`, tokenID, tenantID)
		if err != nil {
			writeErrorJSON(w, r, 500, err.Error())
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeErrorJSON(w, r, 404, "未找到")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeErrorJSON(w, r, 405, "method not allowed")
	}
}

func handleInternalOAuthTokenLookup(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tokenID := strings.TrimSpace(r.URL.Query().Get("id"))
	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if tokenID == "" || companyID == "" {
		writeErrorJSON(w, r, 400, "id and company_id required")
		return
	}
	var platformType, authType string
	err := db.QueryRow(`SELECT platform_type, authorization_type FROM cloud_oauth_tokens WHERE id=? AND company_id=?`,
		tokenID, companyID).Scan(&platformType, &authType)
	if err == sql.ErrNoRows {
		writeErrorJSON(w, r, 404, "not found")
		return
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	if platformType == "" {
		platformType = "aliyun"
	}
	if authType == "" {
		authType = "oauth"
	}
	writeJSON(w, 200, map[string]string{
		"id":                 tokenID,
		"company_id":         companyID,
		"platform_type":      platformType,
		"authorization_type": authType,
	})
}

func listOAuthTokens(tenantID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT id,platform_type,authorization_type,company_id,COALESCE(scope,''),expires_at,created_at,updated_at
		FROM cloud_oauth_tokens WHERE company_id=? ORDER BY updated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		item, err := scanOAuthTokenRow(rows)
		if err != nil {
			return nil, err
		}
		id, _ := item["id"].(string)
		item["is_active"] = authIsActive(tenantID, id, "oauth")
		out = append(out, item)
	}
	return out, nil
}

func getOAuthToken(tenantID, tokenID string) (map[string]interface{}, error) {
	row := db.QueryRow(`SELECT id,platform_type,authorization_type,company_id,COALESCE(scope,''),expires_at,created_at,updated_at
		FROM cloud_oauth_tokens WHERE id=? AND company_id=?`, tokenID, tenantID)
	item, err := scanOAuthTokenRow(row)
	if err != nil {
		return nil, err
	}
	id, _ := item["id"].(string)
	item["is_active"] = authIsActive(tenantID, id, "oauth")
	return item, nil
}

func createOAuthToken(tenantID string, body map[string]interface{}) (map[string]interface{}, error) {
	id := genID("oat")
	pt := strField(body, "platform_type")
	if pt == "" {
		pt = "aliyun"
	}
	at := strField(body, "authorization_type")
	if at == "" {
		at = "oauth"
	}
	scope := strField(body, "scope")
	expiresAt := strField(body, "expires_at")
	accessToken := strField(body, "access_token")
	if accessToken == "" {
		accessToken = "pending"
	}
	refreshToken := strField(body, "refresh_token")
	_, err := db.Exec(`INSERT INTO cloud_oauth_tokens(id,authorization_id,access_token,refresh_token,company_id,platform_type,authorization_type,scope,expires_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		id, id, accessToken, refreshToken, tenantID, pt, at, scope, nullIfEmpty(expiresAt))
	if err != nil {
		return nil, err
	}
	if boolField(body, "is_active") {
		if actErr := applyAuthorizationActive(tenantID, pt, "oauth", id, true); actErr != nil {
			return nil, actErr
		}
	}
	return getOAuthToken(tenantID, id)
}

func updateOAuthToken(tenantID, tokenID string, body map[string]interface{}) (map[string]interface{}, error) {
	res, err := db.Exec(`UPDATE cloud_oauth_tokens SET
		platform_type=COALESCE(NULLIF(?,''),platform_type),
		authorization_type=COALESCE(NULLIF(?,''),authorization_type),
		access_token=COALESCE(NULLIF(?,''),access_token),
		refresh_token=COALESCE(NULLIF(?,''),refresh_token),
		scope=COALESCE(NULLIF(?,''),scope),
		expires_at=CASE WHEN ?='' THEN expires_at ELSE ? END,
		updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND company_id=?`,
		strField(body, "platform_type"), strField(body, "authorization_type"),
		strField(body, "access_token"), strField(body, "refresh_token"),
		strField(body, "scope"),
		strField(body, "expires_at"), strField(body, "expires_at"),
		tokenID, tenantID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, sql.ErrNoRows
	}
	return getOAuthToken(tenantID, tokenID)
}

func scanOAuthTokenRow(scanner interface {
	Scan(dest ...interface{}) error
}) (map[string]interface{}, error) {
	var id, pt, at, cid, scope string
	var expiresAt sql.NullTime
	var createdAt, updatedAt time.Time
	if err := scanner.Scan(&id, &pt, &at, &cid, &scope, &expiresAt, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	isExpired := false
	if expiresAt.Valid {
		isExpired = expiresAt.Time.Before(time.Now())
	}
	return map[string]interface{}{
		"id":                 id,
		"platform_type":      pt,
		"authorization_type": at,
		"company_id":         cid,
		"scope":              scope,
		"is_expired":         isExpired,
		"created_at":         createdAt,
		"updated_at":         updatedAt,
	}, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
