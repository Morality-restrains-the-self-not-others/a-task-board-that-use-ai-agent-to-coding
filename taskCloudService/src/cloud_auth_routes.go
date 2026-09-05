package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

// --- Cloud Authorization ---

func handleCloudAuthRoutes(w http.ResponseWriter, r *http.Request, subParts []string) {
	tenantID := getAuthTenant(r)
	sub := strings.Trim(strings.Join(subParts, "/"), "/")

	if sub == "" {
		switch r.Method {
		case http.MethodGet:
			rows, err := db.Query("SELECT id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at FROM cloud_platform_authorizations WHERE company_id=?", tenantID)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
				return
			}
			defer rows.Close()
			auths := []map[string]interface{}{}
			for rows.Next() {
				a, scanErr := scanAuth(rows)
				if scanErr != nil {
					logError("handleCloudAuthRoutes scan: "+scanErr.Error(), "")
					continue
				}
				authID, _ := a["id"].(string)
				a["is_active"] = authIsActive(tenantID, authID, activeAuthMethod(strField(a, "authorization_type")))
				// Add human-readable platform_name derived from platform_type
				if pt, ok := a["platform_type"].(string); ok {
					a["platform_name"] = platformTypeToName(pt)
				}
				auths = append(auths, a)
			}
			if err := rows.Err(); err != nil {
				logError("handleCloudAuthRoutes rows iteration: "+err.Error(), "")
			}
			writeJSON(w, 200, auths)
		case http.MethodPost:
			body, _ := readJSONBody(r)
			id, err := createCloudAuth(tenantID, body)
			if err != nil {
				logError("cloud auth create: "+err.Error(), r.Header.Get("X-Trace-Id"))
				writeJSON(w, 500, map[string]string{"error": "create failed", "trace_id": traceIDFromRequest(r)})
				return
			}
			pt := strField(body, "platform_type")
			if pt == "" {
				pt = "aliyun"
			}
			at := strField(body, "authorization_type")
			if at == "" {
				at = "access_key"
			}
			publishCloudPlatformAuthorizationCreated(id, pt, at, strField(body, "secret_id"), strField(body, "secret_key"), tenantID)
			logInfo("cloud auth created: "+id, r.Header.Get("X-Trace-Id"))
			writeJSON(w, 201, map[string]string{"id": id, "status": "created"})
		default:
			writeErrorJSON(w, r, 405, "method not allowed")
		}
		return
	}

	parts := strings.SplitN(sub, "/", 2)
	authID := parts[0]
	if authID == "" {
		writeErrorJSON(w, r, 404, "authorization not found")
		return
	}

	if len(parts) == 2 && parts[1] == "verify-credentials" {
		if r.Method != http.MethodPost {
			writeErrorJSON(w, r, 405, "method not allowed")
			return
		}
		row := db.QueryRow("SELECT platform_type,authorization_type,secret_id,secret_key FROM cloud_platform_authorizations WHERE id=? AND company_id=?", authID, tenantID)
		var pt, at, sid, sk string
		if err := row.Scan(&pt, &at, &sid, &sk); err != nil {
			writeErrorJSON(w, r, 404, "authorization not found")
			return
		}
		if at != "access_key" {
			writeErrorMapJSON(w, r, 400, map[string]interface{}{
				"success": false,
				"detail":  "仅支持 Access Key 授权方式的凭据校验",
			})
			return
		}
		result := verifyAccessKeyCallerIdentity(pt, sid, sk)
		if ok, _ := result["success"].(bool); ok {
			writeJSON(w, 200, result)
			return
		}
		writeJSON(w, 400, result)
		return
	}

	switch r.Method {
	case http.MethodGet:
		row := db.QueryRow("SELECT id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at FROM cloud_platform_authorizations WHERE id=? AND company_id=?", authID, tenantID)
		var id, pt, at, sid, sk, remark, cid, oid string
		var active bool
		var ca, ua time.Time
		if err := row.Scan(&id, &pt, &at, &sid, &sk, &remark, &cid, &active, &oid, &ca, &ua); err != nil {
			writeErrorJSON(w, r, 404, "authorization not found")
			return
		}
		if len(sk) > 4 {
			sk = maskCredential(sk)
		}
		if len(sid) > 4 {
			sid = maskCredential(sid)
		}
		writeJSON(w, 200, map[string]interface{}{
			"id": id, "platform_type": pt, "authorization_type": at,
			"secret_id": sid, "secret_key": sk, "remark": remark,
			"company_id": cid, "active": active, "is_active": authIsActive(tenantID, id, activeAuthMethod(at)),
			"oauth_token_id": oid, "created_at": ca, "updated_at": ua,
		})
	case http.MethodPut, http.MethodPatch:
		body, _ := readJSONBody(r)
		_, err := db.Exec(`UPDATE cloud_platform_authorizations SET platform_type=COALESCE(NULLIF(?,''),platform_type), secret_id=COALESCE(NULLIF(?,''),secret_id), secret_key=COALESCE(NULLIF(?,''),secret_key), remark=COALESCE(NULLIF(?,''),remark), updated_at=CURRENT_TIMESTAMP WHERE id=? AND company_id=?`,
			strField(body, "platform_type"), credentialFieldForUpdate(strField(body, "secret_id")), credentialFieldForUpdate(strField(body, "secret_key")), strField(body, "remark"), authID, tenantID)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "update failed", "trace_id": traceIDFromRequest(r)})
			return
		}
		if _, ok := body["is_active"]; ok {
			var pt string
			if scanErr := db.QueryRow(`SELECT platform_type FROM cloud_platform_authorizations WHERE id=? AND company_id=?`, authID, tenantID).Scan(&pt); scanErr != nil {
				writeJSON(w, 500, map[string]string{"error": "update active flag failed", "trace_id": traceIDFromRequest(r)})
				return
			}
			wantActive := boolField(body, "is_active")
			if actErr := applyAuthorizationActive(tenantID, pt, "access_key", authID, wantActive); actErr != nil {
				logError("cloud auth update is_active: "+actErr.Error(), r.Header.Get("X-Trace-Id"))
				writeJSON(w, 500, map[string]string{"error": "update active flag failed", "trace_id": traceIDFromRequest(r)})
				return
			}
			logInfo("cloud auth is_active updated: "+authID, r.Header.Get("X-Trace-Id"))
		}
		writeJSON(w, 200, map[string]string{"id": authID, "status": "updated"})
	case http.MethodDelete:
		res, err := db.Exec("DELETE FROM cloud_platform_authorizations WHERE id=? AND company_id=?", authID, tenantID)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "delete failed", "trace_id": traceIDFromRequest(r)})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeErrorJSON(w, r, 404, "authorization not found")
			return
		}
		logInfo("cloud auth deleted: "+authID, r.Header.Get("X-Trace-Id"))
		w.WriteHeader(http.StatusNoContent)
	default:
		writeErrorJSON(w, r, 405, "method not allowed")
	}
}

// createCloudAuth inserts the cloud_platform_authorizations row and its active
// method in ONE transaction so a failed active-method write rolls back the
// whole create and leaves no residual "disabled" authorization behind
// (OPT-20260828-022).
func createCloudAuth(tenantID string, body map[string]interface{}) (string, error) {
	id := genID("cpa")
	pt := strField(body, "platform_type")
	if pt == "" {
		pt = "aliyun"
	}
	at := strField(body, "authorization_type")
	if at == "" {
		at = "access_key"
	}
	wantActive := true
	if _, ok := body["is_active"]; ok {
		wantActive = boolField(body, "is_active")
	}
	activeInt := 0
	if wantActive {
		activeInt = 1
	}
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO cloud_platform_authorizations(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active) VALUES(?,?,?,?,?,?,?,?)`,
		id, pt, at, strField(body, "secret_id"), strField(body, "secret_key"), strField(body, "remark"), tenantID, activeInt); err != nil {
		return "", err
	}
	if wantActive {
		if _, err := setExclusiveActiveMethodTx(tx, tenantID, pt, at, id); err != nil {
			return "", err
		}
	} else {
		if err := clearActiveMethodTx(tx, tenantID, at, id); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

func scanAuth(rows *sql.Rows) (map[string]interface{}, error) {
	var id, pt, at, sid, sk, cid string
	var remark, oid sql.NullString
	var active bool
	var ca, ua sql.NullTime
	if err := rows.Scan(&id, &pt, &at, &sid, &sk, &remark, &cid, &active, &oid, &ca, &ua); err != nil {
		return nil, err
	}
	if len(sk) > 4 {
		sk = maskCredential(sk)
	}
	if len(sid) > 4 {
		sid = maskCredential(sid)
	}
	remarkVal := ""
	if remark.Valid {
		remarkVal = remark.String
	}
	oidVal := ""
	if oid.Valid {
		oidVal = oid.String
	}
	out := map[string]interface{}{
		"id": id, "platform_type": pt, "authorization_type": at,
		"secret_id": sid, "secret_key": sk, "remark": remarkVal,
		"company_id": cid, "active": active, "oauth_token_id": oidVal,
	}
	if ca.Valid {
		out["created_at"] = ca.Time
	} else {
		out["created_at"] = nil
	}
	if ua.Valid {
		out["updated_at"] = ua.Time
	} else {
		out["updated_at"] = nil
	}
	return out, nil
}

// platformTypeToName maps platform_type codes to human-readable Chinese names.
func platformTypeToName(pt string) string {
	switch pt {
	case "aliyun":
		return "阿里云"
	case "tencentcloud":
		return "腾讯云"
	case "huaweicloud":
		return "华为云"
	case "ctyun":
		return "天翼云"
	case "cmcc":
		return "移动云"
	case "cucloud":
		return "联通云"
	case "baiducloud":
		return "百度智能云"
	case "aws":
		return "AWS"
	case "jdcloud":
		return "京东云"
	default:
		return pt
	}
}
