package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

func handleCloudWorkspaceRoutes(w http.ResponseWriter, r *http.Request, subPath string) {
	tenantID := getAuthTenant(r)
	workspaceID := r.Header.Get("X-Workspace-Id")

	sub := strings.Trim(strings.TrimPrefix(subPath, "cloud/"), "/")

	// workspace-level cloud platform listing (OPT-049 gap: Django endpoint no longer exists)
	if sub == "platforms" {
		handleWorkspaceCloudPlatforms(w, r, tenantID, workspaceID, "")
		return
	}
	if strings.HasPrefix(sub, "platforms/") {
		handleWorkspaceCloudPlatforms(w, r, tenantID, workspaceID, strings.TrimPrefix(sub, "platforms/"))
		return
	}

	// v64: /api/cloud/ 约定下恢复工作区级 budget/feature-params API
	//（前端已迁移 kv-last 路径；此前 ead5583 移除 /api/tenant/ 时未重新挂载）
	if sub == "feature-params" || strings.HasPrefix(sub, "feature-params/") {
		handleWorkspaceFeatureParams(w, r, tenantID, workspaceID)
		return
	}
	if sub == "model-budget-defaults" || strings.HasPrefix(sub, "model-budget-defaults/") {
		handleUserWorkspaceModelBudgetDefaults(w, r, tenantID, workspaceID)
		return
	}

	// 工作区级 cloud 路由与任务级共用 compute handler；task_id 来自 query
	if taskID := strings.TrimSpace(r.URL.Query().Get("task_id")); taskID != "" {
		r.Header.Set("X-Task-Id", taskID)
	}
	handleCloudTaskRoutes(w, r, subPath)
}

// handleWorkspaceCloudPlatforms returns cloud platform bindings available to a workspace.
// OPT-049: replaces the decommissioned Django workspaces/{id}/cloud/platforms/ endpoint.
func handleWorkspaceCloudPlatforms(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, sub string) {
	_ = workspaceID // reserved for future workspace-level filtering (e.g. machine policy)

	sub = strings.TrimSuffix(strings.TrimSpace(sub), "/")

	// GET platforms/ — list all active cloud platform authorizations for the company
	if sub == "" || strings.HasPrefix(sub, "?") {
		if r.Method != http.MethodGet {
			writeErrorJSON(w, r, 405, "method not allowed")
			return
		}
		rows, err := db.Query(
			`SELECT id, platform_type, authorization_type, secret_id, secret_key, remark, company_id, active, oauth_token_id, created_at, updated_at
			 FROM cloud_platform_authorizations WHERE company_id=? ORDER BY created_at ASC`, tenantID)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
			return
		}
		defer rows.Close()
		platforms := []map[string]interface{}{}
		for rows.Next() {
			a, scanErr := scanAuth(rows)
			if scanErr != nil {
				logError("handleWorkspaceCloudPlatforms scan: "+scanErr.Error(), "")
				continue
			}
			authID, _ := a["id"].(string)
			a["is_active"] = authIsActive(tenantID, authID, activeAuthMethod(strField(a, "authorization_type")))
			// Ensure authorization_id field for frontend compatibility
			if _, ok := a["authorization_id"]; !ok {
				a["authorization_id"] = authID
			}
			// Add human-readable platform_name derived from platform_type
			if pt, ok := a["platform_type"].(string); ok {
				a["platform_name"] = platformTypeToName(pt)
			}
			// Attach IAM associations
			iamRows, iamErr := db.Query(
				`SELECT id, access_key, iam_id FROM cloud_access_key_iam_associations WHERE cloud_platform_auth_id=?`, authID)
			if iamErr == nil {
				type iamEntry struct{ ID, AccessKey, IAMID string }
				iams := []iamEntry{}
				for iamRows.Next() {
					var e iamEntry
					if se := iamRows.Scan(&e.ID, &e.AccessKey, &e.IAMID); se == nil {
						iams = append(iams, e)
					}
				}
				iamRows.Close()
				if len(iams) > 0 {
					a["iam_id"] = iams[0].IAMID
					a["iam_associations"] = iams
				}
			}
			platforms = append(platforms, a)
		}
		if err := rows.Err(); err != nil {
			logError("handleWorkspaceCloudPlatforms rows iteration: "+err.Error(), "")
		}
		writeJSON(w, 200, map[string]interface{}{"platforms": platforms})
		return
	}

	// GET platforms/default-config — return default server configs applicable to this workspace
	if sub == "default-config" {
		if r.Method != http.MethodGet {
			writeErrorJSON(w, r, 405, "method not allowed")
			return
		}
		rows, err := db.Query(
			`SELECT `+serverConfigDefaultSelectCols+`
			 FROM cloud_server_config_defaults WHERE company_id=? ORDER BY created_at DESC`, tenantID)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error(), "trace_id": traceIDFromRequest(r)})
			return
		}
		defer rows.Close()
		configs := []map[string]interface{}{}
		for rows.Next() {
			var id, cid, aid, pt, region, zoneID, vpc, vswitch, sg, payType, bwMode string
			var bw int
			var cpuCores, memoryGB, instanceType, systemDisk, dataDisk string
			var ioOptimized, spotStrategy string
			var ca, ua time.Time
			if scanErr := rows.Scan(
				&id, &cid, &aid, &pt, &region, &zoneID, &vpc, &vswitch, &sg, &payType, &bwMode, &bw,
				&cpuCores, &memoryGB, &instanceType, &systemDisk, &dataDisk,
				&ioOptimized, &spotStrategy, &ca, &ua,
			); scanErr != nil {
				logError("handleWorkspaceCloudPlatforms default-config scan: "+scanErr.Error(), "")
				continue
			}
			configs = append(configs, mapServerConfigDefaultRow(
				id, cid, aid, pt, region, zoneID, vpc, vswitch, sg, payType, bwMode, bw,
				cpuCores, memoryGB, instanceType, systemDisk, dataDisk,
				ioOptimized, spotStrategy, ca, ua,
			))
		}
		if err := rows.Err(); err != nil {
			logError("handleWorkspaceCloudPlatforms default-config rows iteration: "+err.Error(), "")
		}
		writeJSON(w, 200, map[string]interface{}{"status": "success", "data": configs})
		return
	}

	writeErrorMapJSON(w, r, 404, map[string]interface{}{"error": "platforms sub-route not found", "path": sub})
}

// --- OAuth ---

func handleCloudOAuthRoutes(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) >= 2 && parts[0] == "aliyun" && parts[1] == "login" {
		handleAliyunOAuthLogin(w, r)
		return
	}
	writeErrorJSON(w, r, 404, "oauth route not found")
}

// --- Cloud Platform Detail ---

func handleCloudPlatformDetail(w http.ResponseWriter, r *http.Request) {
	authID := r.Header.Get("X-Authorization-Id")
	row := db.QueryRow("SELECT id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at FROM cloud_platform_authorizations WHERE id=?", authID)
	var id, pt, at, sid, sk, cid string
	var remark, oid sql.NullString
	var active bool
	var ca, ua sql.NullTime
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
		"company_id": cid, "active": active, "created_at": ca, "updated_at": ua,
	})
}
