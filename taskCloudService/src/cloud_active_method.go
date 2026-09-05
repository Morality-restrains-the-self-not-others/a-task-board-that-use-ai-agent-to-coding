package main

import (
	"database/sql"
	"net/http"
	"time"
)

type activeMethodRecord struct {
	ID             string
	CompanyID      string
	PlatformType   string
	AuthMethod     string
	AuthInstanceID string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// failSetExclusiveActiveMethodTx is a test-only hook: when non-nil it is
// invoked at the start of setExclusiveActiveMethodTx to force the active-method
// write to fail. Used to verify cloud-auth create rolls back atomically
// (OPT-20260828-022).
var failSetExclusiveActiveMethodTx func() error

// activeAuthMethod returns the active-method key to use for a
// cloud_platform_authorizations row: its authorization_type, defaulting to
// access_key for legacy rows (OPT-20260828-023). List/detail/toggle must all
// key on this so an oauth-type CPA is not misread as "disabled".
func activeAuthMethod(authorizationType string) string {
	if authorizationType == "" {
		return "access_key"
	}
	return authorizationType
}

func authIsActive(tenantID, authInstanceID, authMethod string) bool {
	var active int
	err := db.QueryRow(`SELECT is_active FROM cloud_platform_authorization_active_methods
		WHERE company_id=? AND auth_instance_id=? AND auth_method=?`,
		tenantID, authInstanceID, authMethod).Scan(&active)
	return err == nil && active != 0
}

func listActiveMethodRows(tenantID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT id,company_id,platform_type,auth_method,auth_instance_id,is_active,created_at,updated_at
		FROM cloud_platform_authorization_active_methods WHERE company_id=? ORDER BY updated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		rec, err := scanActiveMethodRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, activeMethodToJSON(rec))
	}
	return out, nil
}

func scanActiveMethodRow(rows *sql.Rows) (activeMethodRecord, error) {
	var rec activeMethodRecord
	var active int
	err := rows.Scan(&rec.ID, &rec.CompanyID, &rec.PlatformType, &rec.AuthMethod,
		&rec.AuthInstanceID, &active, &rec.CreatedAt, &rec.UpdatedAt)
	rec.IsActive = active != 0
	return rec, err
}

func activeMethodToJSON(rec activeMethodRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":               rec.ID,
		"company_id":       rec.CompanyID,
		"platform_type":    rec.PlatformType,
		"auth_method":      rec.AuthMethod,
		"auth_instance_id": rec.AuthInstanceID,
		"is_active":        rec.IsActive,
		"created_at":       rec.CreatedAt,
		"updated_at":       rec.UpdatedAt,
	}
}

func clearActiveMethod(tenantID, authMethod, authInstanceID string) error {
	_, err := db.Exec(`UPDATE cloud_platform_authorization_active_methods SET is_active=0, updated_at=CURRENT_TIMESTAMP
		WHERE company_id=? AND auth_instance_id=? AND auth_method=?`,
		tenantID, authInstanceID, authMethod)
	return err
}

func clearActiveMethodTx(tx *sql.Tx, tenantID, authMethod, authInstanceID string) error {
	_, err := tx.Exec(`UPDATE cloud_platform_authorization_active_methods SET is_active=0, updated_at=CURRENT_TIMESTAMP
		WHERE company_id=? AND auth_instance_id=? AND auth_method=?`,
		tenantID, authInstanceID, authMethod)
	return err
}

func applyAuthorizationActive(tenantID, platformType, authMethod, authInstanceID string, wantActive bool) error {
	if wantActive {
		_, err := setExclusiveActiveMethod(tenantID, platformType, authMethod, authInstanceID)
		return err
	}
	return clearActiveMethod(tenantID, authMethod, authInstanceID)
}

func setExclusiveActiveMethod(tenantID, platformType, authMethod, authInstanceID string) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err := setExclusiveActiveMethodTx(tx, tenantID, platformType, authMethod, authInstanceID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// setExclusiveActiveMethodTx runs the exclusive active-method write on the
// given transaction so callers can co-locate it with other writes (e.g. the
// cloud_platform_authorizations INSERT on create) in a single tx.
func setExclusiveActiveMethodTx(tx *sql.Tx, tenantID, platformType, authMethod, authInstanceID string) (bool, error) {
	if failSetExclusiveActiveMethodTx != nil {
		if err := failSetExclusiveActiveMethodTx(); err != nil {
			return false, err
		}
	}

	if _, err := tx.Exec(`UPDATE cloud_platform_authorization_active_methods SET is_active=0, updated_at=CURRENT_TIMESTAMP WHERE company_id=?`, tenantID); err != nil {
		return false, err
	}

	var existingID string
	err := tx.QueryRow(`SELECT id FROM cloud_platform_authorization_active_methods
		WHERE company_id=? AND platform_type=? AND auth_method=? AND auth_instance_id=?`,
		tenantID, platformType, authMethod, authInstanceID).Scan(&existingID)

	if err == sql.ErrNoRows {
		existingID = genID("cpam")
		_, err = tx.Exec(`INSERT INTO cloud_platform_authorization_active_methods
			(id,company_id,platform_type,auth_method,auth_instance_id,is_active) VALUES(?,?,?,?,?,1)`,
			existingID, tenantID, platformType, authMethod, authInstanceID)
	} else if err == nil {
		_, err = tx.Exec(`UPDATE cloud_platform_authorization_active_methods SET is_active=1, updated_at=CURRENT_TIMESTAMP WHERE id=?`, existingID)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func importActiveMethods(rows []map[string]interface{}) (int, error) {
	count := 0
	for _, row := range rows {
		id := strField(row, "id")
		if id == "" {
			id = genID("cpam")
		}
		active := 0
		if v, ok := row["is_active"].(bool); ok && v {
			active = 1
		} else if v, ok := row["is_active"].(float64); ok && v != 0 {
			active = 1
		}
		_, err := db.Exec(`REPLACE INTO cloud_platform_authorization_active_methods
			(id,company_id,platform_type,auth_method,auth_instance_id,is_active,created_at,updated_at)
			VALUES(?,?,?,?,?,?,COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP),COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP))`,
			id, strField(row, "company_id"), strField(row, "platform_type"),
			strField(row, "auth_method"), strField(row, "auth_instance_id"), active,
			strField(row, "created_at"), strField(row, "updated_at"))
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func importCloudAuthorizations(rows []map[string]interface{}) (int, error) {
	count := 0
	for _, row := range rows {
		id := strField(row, "id")
		if id == "" {
			continue
		}
		active := 1
		if v, ok := row["active"].(bool); ok && !v {
			active = 0
		}
		_, err := db.Exec(`REPLACE INTO cloud_platform_authorizations
			(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP),COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP))`,
			id, strField(row, "platform_type"), strField(row, "authorization_type"),
			strField(row, "secret_id"), strField(row, "secret_key"), strField(row, "remark"),
			strField(row, "company_id"), active, strField(row, "oauth_token_id"),
			strField(row, "created_at"), strField(row, "updated_at"))
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func handleToggleActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	tenantID := getAuthTenant(r)
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	body, _ := readJSONBody(r)
	authID := strField(body, "id")
	if authID == "" {
		writeJSON(w, 400, map[string]string{"message": "缺少必要参数"})
		return
	}
	platformType, authMethod, err := resolveToggleAuthTarget(tenantID, authID)
	if err != nil {
		writeJSON(w, 404, map[string]string{"message": "指定的授权不存在"})
		return
	}
	wantActive := true
	if _, ok := body["is_active"]; ok {
		wantActive = boolField(body, "is_active")
	}
	if !wantActive {
		if err := clearActiveMethod(tenantID, authMethod, authID); err != nil {
			writeJSON(w, 500, map[string]string{"message": "更新云平台授权状态失败: " + err.Error(), "trace_id": traceIDFromRequest(r)})
			return
		}
		logInfo("cloud auth deactivated: "+authID, r.Header.Get("X-Trace-Id"))
		writeJSON(w, 200, map[string]interface{}{
			"message": "云平台授权状态更新成功", "is_active": false,
		})
		return
	}
	isActive, err := setExclusiveActiveMethod(tenantID, platformType, authMethod, authID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": "更新云平台授权状态失败: " + err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	logInfo("cloud auth activated: "+authID, r.Header.Get("X-Trace-Id"))
	writeJSON(w, 200, map[string]interface{}{
		"message": "云平台授权状态更新成功", "is_active": isActive,
	})
}

func resolveToggleAuthTarget(tenantID, authID string) (platformType, authMethod string, err error) {
	row := db.QueryRow(`SELECT platform_type, authorization_type FROM cloud_platform_authorizations WHERE id=? AND company_id=?`, authID, tenantID)
	var at string
	if err = row.Scan(&platformType, &at); err == nil {
		return platformType, activeAuthMethod(at), nil
	}
	var oauthPlatform string
	err = db.QueryRow(`SELECT platform_type FROM cloud_oauth_tokens WHERE id=? AND company_id=?`, authID, tenantID).Scan(&oauthPlatform)
	if err == nil {
		return oauthPlatform, "oauth", nil
	}
	return "", "", err
}

func handleActiveList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	tenantID := getAuthTenant(r)
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	rows, err := listActiveMethodRows(tenantID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	writeJSON(w, 200, rows)
}
