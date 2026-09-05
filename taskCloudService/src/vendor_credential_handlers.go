package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

func handleVendorCloudCredentialRoutes(w http.ResponseWriter, r *http.Request) {
	vendorAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/vendor/cloud-platform-credentials")
		path = strings.Trim(path, "/")
		vendorID := getVendorID(r)

		if path == "" {
			switch r.Method {
			case http.MethodGet:
				listVendorCloudCredentials(w, r, vendorID)
			case http.MethodPost:
				createVendorCloudCredential(w, r, vendorID)
			default:
				writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}

		parts := strings.SplitN(path, "/", 2)
		credID := parts[0]
		if credID == "" {
			writeErrorJSON(w, r, http.StatusNotFound, "not found")
			return
		}

		if len(parts) == 2 && parts[1] == "verify-credentials" {
			if r.Method != http.MethodPost {
				writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			verifyVendorCloudCredential(w, r, vendorID, credID)
			return
		}

		switch r.Method {
		case http.MethodGet:
			getVendorCloudCredential(w, r, vendorID, credID)
		case http.MethodPut, http.MethodPatch:
			updateVendorCloudCredential(w, r, vendorID, credID)
		case http.MethodDelete:
			deleteVendorCloudCredential(w, r, vendorID, credID)
		default:
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
	})(w, r)
}

func scanVendorCredentialRow(row interface {
	Scan(dest ...any) error
}) (map[string]interface{}, error) {
	var id, vendorID, pt, sid, sk, remark string
	var lastErr sql.NullString
	var active bool
	var lastVerified sql.NullTime
	var ca, ua time.Time
	if err := row.Scan(&id, &vendorID, &pt, &sid, &sk, &remark, &lastVerified, &lastErr, &active, &ca, &ua); err != nil {
		return nil, err
	}
	lastErrVal := ""
	if lastErr.Valid {
		lastErrVal = lastErr.String
	}
	out := map[string]interface{}{
		"id": id, "vendor_id": vendorID, "platform_type": pt,
		"secret_id": maskCredential(sid), "secret_key": maskCredential(sk),
		"remark": remark, "last_verify_error": lastErrVal,
		"is_active": active, "created_at": ca, "updated_at": ua,
	}
	if lastVerified.Valid {
		out["last_verified_at"] = lastVerified.Time
	} else {
		out["last_verified_at"] = nil
	}
	return out, nil
}


func listVendorCloudCredentials(w http.ResponseWriter, r *http.Request, vendorID string) {
	rows, err := db.Query(`SELECT id,vendor_id,platform_type,secret_id,secret_key,remark,last_verified_at,last_verify_error,is_active,created_at,updated_at
		FROM cloud_vendor_platform_credentials WHERE vendor_id=? ORDER BY platform_type`, vendorID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		item, scanErr := scanVendorCredentialRow(rows)
		if scanErr != nil {
			logError("listVendorCloudCredentials scan: " + scanErr.Error(), "")
			continue
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		logError("listVendorCloudCredentials rows iteration: " + err.Error(), "")
		writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	writeJSON(w, 200, out)
}

func createVendorCloudCredential(w http.ResponseWriter, r *http.Request, vendorID string) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, 400, "invalid json")
		return
	}
	pt := strField(body, "platform_type")
	if pt == "" {
		pt = "aliyun"
	}
	sid := strField(body, "secret_id")
	sk := strField(body, "secret_key")
	if sid == "" || sk == "" {
		writeErrorJSON(w, r, 400, "secret_id 与 secret_key 为必填项")
		return
	}
	id := genID("vcc")
	_, err = db.Exec(`INSERT INTO cloud_vendor_platform_credentials(id,vendor_id,platform_type,secret_id,secret_key,remark,is_active)
		VALUES(?,?,?,?,?,?,1)`,
		id, vendorID, pt, sid, sk, strField(body, "remark"))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeErrorJSON(w, r, 400, "该云平台已存在测试密钥，请编辑现有记录")
			return
		}
		writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	// Return full credential object so frontend can render it immediately
	row := db.QueryRow(`SELECT id,vendor_id,platform_type,secret_id,secret_key,remark,last_verified_at,last_verify_error,is_active,created_at,updated_at
		FROM cloud_vendor_platform_credentials WHERE id=?`, id)
	out, scanErr := scanVendorCredentialRow(row)
	if scanErr != nil {
		writeJSON(w, 201, map[string]string{"id": id, "status": "created"})
		return
	}
	writeJSON(w, 201, out)
}

func getVendorCloudCredential(w http.ResponseWriter, r *http.Request, vendorID, credID string) {
	row := db.QueryRow(`SELECT id,vendor_id,platform_type,secret_id,secret_key,remark,last_verified_at,last_verify_error,is_active,created_at,updated_at
		FROM cloud_vendor_platform_credentials WHERE id=? AND vendor_id=?`, credID, vendorID)
	out, err := scanVendorCredentialRow(row)
	if err != nil {
		writeErrorJSON(w, r, 404, "凭证不存在")
		return
	}
	writeJSON(w, 200, out)
}

func updateVendorCloudCredential(w http.ResponseWriter, r *http.Request, vendorID, credID string) {
	body, _ := readJSONBody(r)
	isActiveRaw := strField(body, "is_active")
	if isActiveRaw == "true" || isActiveRaw == "1" {
		_, _ = db.Exec(`UPDATE cloud_vendor_platform_credentials SET is_active=1, updated_at=CURRENT_TIMESTAMP WHERE id=? AND vendor_id=?`, credID, vendorID)
	} else if isActiveRaw == "false" || isActiveRaw == "0" {
		_, _ = db.Exec(`UPDATE cloud_vendor_platform_credentials SET is_active=0, updated_at=CURRENT_TIMESTAMP WHERE id=? AND vendor_id=?`, credID, vendorID)
	}
	_, err := db.Exec(`UPDATE cloud_vendor_platform_credentials SET
		platform_type=COALESCE(NULLIF(?,''),platform_type),
		secret_id=COALESCE(NULLIF(?,''),secret_id),
		secret_key=COALESCE(NULLIF(?,''),secret_key),
		remark=COALESCE(NULLIF(?,''),remark),
		updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND vendor_id=?`,
		strField(body, "platform_type"), credentialFieldForUpdate(strField(body, "secret_id")), credentialFieldForUpdate(strField(body, "secret_key")),
		strField(body, "remark"), credID, vendorID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	writeJSON(w, 200, map[string]string{"id": credID, "status": "updated"})
}

func deleteVendorCloudCredential(w http.ResponseWriter, r *http.Request, vendorID, credID string) {
	res, err := db.Exec(`DELETE FROM cloud_vendor_platform_credentials WHERE id=? AND vendor_id=?`, credID, vendorID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeErrorJSON(w, r, 404, "凭证不存在")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func verifyVendorCloudCredential(w http.ResponseWriter, r *http.Request, vendorID, credID string) {
	row := db.QueryRow(`SELECT platform_type,secret_id,secret_key FROM cloud_vendor_platform_credentials WHERE id=? AND vendor_id=? AND is_active=1`, credID, vendorID)
	var pt, sid, sk string
	if err := row.Scan(&pt, &sid, &sk); err != nil {
		writeErrorJSON(w, r, 404, "凭证不存在或未启用")
		return
	}
	result := verifyAccessKeyCallerIdentity(pt, sid, sk)
	if ok, _ := result["success"].(bool); ok {
		_, _ = db.Exec(`UPDATE cloud_vendor_platform_credentials SET last_verified_at=CURRENT_TIMESTAMP, last_verify_error='', updated_at=CURRENT_TIMESTAMP WHERE id=?`, credID)
		writeJSON(w, 200, result)
		return
	}
	detail, _ := result["detail"].(string)
	_, _ = db.Exec(`UPDATE cloud_vendor_platform_credentials SET last_verify_error=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, detail, credID)
	writeJSON(w, 400, result)
}

func loadVendorCloudCredentialSecrets(vendorID, platformType string) (secretID, secretKey string, err error) {
	row := db.QueryRow(`SELECT secret_id,secret_key FROM cloud_vendor_platform_credentials
		WHERE vendor_id=? AND platform_type=? AND is_active=1`, vendorID, platformType)
	if err := row.Scan(&secretID, &secretKey); err != nil {
		return "", "", err
	}
	return secretID, secretKey, nil
}
