package main

import (
	"net/http"
)

func handleInternalCloudPlatformAuthorizations(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	auths, _ := body["authorizations"].([]interface{})
	activeRows, _ := body["active_methods"].([]interface{})
	authCount := 0
	activeCount := 0
	var err error
	if len(auths) > 0 {
		rows := make([]map[string]interface{}, 0, len(auths))
		for _, item := range auths {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, m)
			}
		}
		authCount, err = importCloudAuthorizations(rows)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
			return
		}
	}
	if len(activeRows) > 0 {
		rows := make([]map[string]interface{}, 0, len(activeRows))
		for _, item := range activeRows {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, m)
			}
		}
		activeCount, err = importActiveMethods(rows)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "imported_authorizations": authCount, "imported_active_methods": activeCount,
	})
}

func handleInternalCloudPlatformAuthorizationLookup(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantID := r.URL.Query().Get("company_id")
	authID := r.URL.Query().Get("id")
	if authID == "" && tenantID != "" {
		rows, err := db.Query(`SELECT id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at
			FROM cloud_platform_authorizations WHERE company_id=?`, tenantID)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
			return
		}
		defer rows.Close()
		out := []map[string]interface{}{}
		for rows.Next() {
			a, scanErr := scanAuth(rows)
			if scanErr != nil {
				logError("handleInternalCloudPlatformAuthorizationLookup scan: "+scanErr.Error(), "")
				continue
			}
			id, _ := a["id"].(string)
			a["is_active"] = authIsActive(tenantID, id, activeAuthMethod(strField(a, "authorization_type")))
			out = append(out, a)
		}
		if err := rows.Err(); err != nil {
			logError("handleInternalCloudPlatformAuthorizationLookup rows iteration: "+err.Error(), "")
		}
		writeJSON(w, 200, out)
		return
	}
	platformType := r.URL.Query().Get("platform_type")
	auth, err := loadCloudAuth(tenantID, authID)
	if auth == nil && authID != "" && tenantID == "" {
		auth, err = loadCloudAuthByID(authID)
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	if auth == nil {
		writeErrorJSON(w, r, 404, "authorization not found")
		return
	}
	if platformType != "" && auth.PlatformType != platformType {
		writeErrorJSON(w, r, 404, "authorization not found")
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"id": auth.ID, "platform_type": auth.PlatformType,
		"authorization_type": auth.AuthorizationType,
		"secret_id": auth.SecretID, "secret_key": auth.SecretKey,
		"remark": auth.Remark, "company_id": auth.CompanyID,
		"is_active": authIsActive(tenantID, auth.ID, activeAuthMethod(auth.AuthorizationType)),
	})
}

func handleInternalActiveMethodsList(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantID := r.URL.Query().Get("company_id")
	if tenantID == "" {
		writeErrorJSON(w, r, 400, "company_id required")
		return
	}
	rows, err := listActiveMethodRows(tenantID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}
	writeJSON(w, 200, rows)
}
