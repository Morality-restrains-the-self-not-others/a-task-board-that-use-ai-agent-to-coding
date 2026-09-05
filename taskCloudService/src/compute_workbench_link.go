package main

import (
	"net/http"
	"net/url"
	"strings"
)

func buildAliyunWorkbenchURL(region, instanceID, username string) string {
	q := url.Values{}
	q.Set("from", "ecs.console")
	q.Set("type", "ecs")
	q.Set("regionId", region)
	q.Set("instanceId", instanceID)
	if strings.TrimSpace(username) != "" {
		q.Set("username", strings.TrimSpace(username))
	}
	return "https://ecs-workbench.aliyun.com/?" + q.Encode()
}

func handleWorkbenchLink(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	traceID := r.Header.Get("X-Trace-Id")
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "缺少任务ID",
		})
		return
	}
	if tenantID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "无法获取租户ID",
		})
		return
	}

	commentID := commentIDFromComputeRequest(r, nil)
	if rejectMissingComputeCommentID(w, r, commentID) {
		return
	}
	cfg, err := resolveScopedCloudServerConfig(tenantID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		logWarn("workbench-link: server config not found task_id="+taskID, traceID)
		writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{
			"status": "error", "message": "未找到任务对应的服务器配置",
		})
		return
	}

	instanceID := strings.TrimSpace(cfg.InstanceID)
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = lookupRegionForWorkbench(tenantID, workspaceID, taskID, instanceID)
	}
	if instanceID == "" || region == "" {
		logWarn("workbench-link: missing instance_id or region task_id="+taskID, traceID)
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "服务器配置缺少实例ID或地域",
		})
		return
	}

	if strings.HasPrefix(instanceID, "mock-") {
		logInfo("workbench-link: mock instance not supported task_id="+taskID+" instance_id="+instanceID, traceID)
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":      "error",
			"message":     "Mock 实例不支持 Workbench 链接（本地 Docker 容器）",
			"mock":        true,
			"instance_id": instanceID,
		})
		return
	}

	platform := strings.ToLower(strings.TrimSpace(cfg.Platform))
	if platform != "aliyun" && commentCSCPlatformIsMockOrEmpty(platform) && !isMockMachineInstanceID(instanceID) {
		// 真实 instance 卡在遗留 mock 标签：按阿里云控制台拼 URL（heal 未落到任务级模板时的兜底）。
		platform = "aliyun"
	}
	if platform != "aliyun" {
		logWarn("workbench-link: unsupported platform="+cfg.Platform+" task_id="+taskID, traceID)
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "暂不支持平台 " + cfg.Platform + " 的 Workbench 链接",
		})
		return
	}

	workbenchURL := buildAliyunWorkbenchURL(region, instanceID, r.URL.Query().Get("username"))
	logInfo("workbench-link: generated url task_id="+taskID+" instance_id="+instanceID+" region="+region, traceID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"workbench_url": workbenchURL,
		"platform":      cfg.Platform,
		"instance_id":   instanceID,
		"region":        region,
	})
}

// lookupRegionForWorkbench 在当前 CSC 缺 region 时，从同任务其它 CSC 回填（优先同 instance_id）。
func lookupRegionForWorkbench(companyID, workspaceID, taskID, instanceID string) string {
	companyID = strings.TrimSpace(companyID)
	workspaceID = strings.TrimSpace(workspaceID)
	taskID = strings.TrimSpace(taskID)
	instanceID = strings.TrimSpace(instanceID)
	if taskID == "" {
		return ""
	}
	q := `SELECT region FROM cloud_server_configs
		WHERE task_id=? AND TRIM(COALESCE(region,'')) != ''`
	args := []interface{}{taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	if companyID != "" {
		q += ` AND company_id=?`
		args = append(args, companyID)
	}
	if instanceID != "" {
		q += ` ORDER BY CASE WHEN TRIM(COALESCE(instance_id,''))=? THEN 0 ELSE 1 END,
			CASE WHEN COALESCE(comment_id,'')='' THEN 0 ELSE 1 END, updated_at DESC LIMIT 1`
		args = append(args, instanceID)
	} else {
		q += ` ORDER BY CASE WHEN COALESCE(comment_id,'')='' THEN 0 ELSE 1 END, updated_at DESC LIMIT 1`
	}
	var region string
	if err := db.QueryRow(q, args...).Scan(&region); err != nil {
		return ""
	}
	return strings.TrimSpace(region)
}
