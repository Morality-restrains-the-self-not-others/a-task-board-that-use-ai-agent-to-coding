package main

// 个人信息导出内部端点（PIPL「导出权」）：taskAuth 聚合导出时调用。
// 契约：GET /api/internal/cloud/users/{user_id}/personal-data/ → {"data": {...}}
// 数据边界：该用户（image_invoker_user_id）调起的云资源配置。
// 安全边界：不含 verification_secret / client_token / business_api_endpoint 等内部凭证。

import (
	"net/http"
)

// collectCloudPersonalData 汇总该用户在 taskCloudService 的云资源数据。
func collectCloudPersonalData(userID string) (map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT id, company_id, workspace_id, task_id, COALESCE(comment_id,''),
		       COALESCE(platform,''), COALESCE(instance_id,''), COALESCE(region,''),
		       COALESCE(public_ip,''), COALESCE(server_url,''),
		       COALESCE(last_runtime_status,''), COALESCE(started_via,''),
		       COALESCE(created_at,''), COALESCE(updated_at,''), COALESCE(idle_since,'')
		FROM cloud_server_configs
		WHERE image_invoker_user_id = ?
		ORDER BY created_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	servers := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, companyID, workspaceID, taskID, commentID, platform, instanceID, region,
			publicIP, serverURL, runtimeStatus, startedVia, createdAt, updatedAt, idleSince string
		if err := rows.Scan(&id, &companyID, &workspaceID, &taskID, &commentID, &platform,
			&instanceID, &region, &publicIP, &serverURL, &runtimeStatus, &startedVia,
			&createdAt, &updatedAt, &idleSince); err != nil {
			continue
		}
		item := map[string]interface{}{
			"id":                  id,
			"company_id":          companyID,
			"workspace_id":        workspaceID,
			"task_id":             taskID,
			"comment_id":          commentID,
			"platform":            platform,
			"instance_id":         instanceID,
			"region":              region,
			"public_ip":           publicIP,
			"server_url":          serverURL,
			"last_runtime_status": runtimeStatus,
			"started_via":         startedVia,
			"created_at":          createdAt,
			"updated_at":          updatedAt,
		}
		if idleSince != "" {
			item["idle_since"] = idleSince
		}
		servers = append(servers, item)
	}
	return map[string]interface{}{"cloud_servers": servers}, nil
}

// handleInternalCloudPersonalData 处理内部 personal-data 请求（注册于 handleInternalCloudUsersRouter）。
func handleInternalCloudPersonalData(w http.ResponseWriter, r *http.Request, userID string) {
	data, err := collectCloudPersonalData(userID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}
