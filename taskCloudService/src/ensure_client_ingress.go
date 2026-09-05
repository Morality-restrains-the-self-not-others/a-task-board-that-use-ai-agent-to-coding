package main

import (
	"fmt"
	"net/http"
	"strings"
)

// handleEnsureClientIngress POST …/cloud/compute/ensure-client-ingress/
// Ensures the caller's public IP(s) are authorized on the task VM security group
// so the browser can DIRECT-connect to container_page_url (IP:8765) without
// proxying console traffic through SaaS.
func handleEnsureClientIngress(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	if tenantID == "" || workspaceID == "" || taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "缺少租户/工作区/任务ID",
		})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "无效的请求体",
		})
		return
	}
	ips := collectEnsureClientIngressIPs(r, body)
	if len(ips) == 0 {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "无法解析客户端公网 IP",
		})
		return
	}

	commentID := commentIDFromComputeRequest(r, body)
	cfg, err := loadCloudServerConfigForGatewayForward(tenantID, workspaceID, taskID, commentID, "")
	if err != nil || cfg == nil {
		writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{
			"status": "error", "message": "未找到容器服务器配置",
		})
		return
	}
	platform := strings.ToLower(strings.TrimSpace(cfg.Platform))
	sgID := strings.TrimSpace(cfg.SecurityGroupID)
	region := strings.TrimSpace(cfg.Region)
	authID := strings.TrimSpace(cfg.AuthorizationID)

	if platform == "mock" || platform == "relay-local" || strings.HasPrefix(platform, "relay") {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":     "success",
			"skipped":    true,
			"reason":     "local_or_mock_platform",
			"client_ips": ips,
		})
		return
	}
	if sgID == "" || region == "" || authID == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":     "success",
			"skipped":    true,
			"reason":     "missing_security_group",
			"client_ips": ips,
		})
		return
	}
	auth, err := loadCloudAuth(tenantID, authID)
	if err != nil || auth == nil {
		writeErrorMapJSON(w, r, http.StatusBadGateway, map[string]interface{}{
			"status": "error", "message": "云授权凭证不可用",
		})
		return
	}
	if strings.ToLower(strings.TrimSpace(auth.PlatformType)) != "aliyun" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":     "success",
			"skipped":    true,
			"reason":     "unsupported_platform",
			"client_ips": ips,
		})
		return
	}

	results := make([]map[string]interface{}, 0, len(ips))
	anyAuthorized := false
	var firstErr string
	for _, ip := range ips {
		rid, already, authErr := aliyunAuthorizeClientIngress(auth.SecretID, auth.SecretKey, region, sgID, ip)
		item := map[string]interface{}{
			"client_ip": ip,
			"cidr":      normalizePublicHostCIDR(ip),
		}
		if authErr != nil {
			item["ok"] = false
			item["error"] = authErr.Error()
			if firstErr == "" {
				firstErr = authErr.Error()
			}
			logWarn("ensure-client-ingress authorize failed ip="+ip+" err="+authErr.Error(), r.Header.Get("X-Trace-Id"))
		} else {
			item["ok"] = true
			item["already"] = already
			item["request_id"] = rid
			anyAuthorized = true
		}
		results = append(results, item)
	}
	if !anyAuthorized && firstErr != "" {
		writeErrorMapJSON(w, r, http.StatusBadGateway, map[string]interface{}{
			"status": "error", "message": "安全组授权失败: " + firstErr, "results": results,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":            "success",
		"skipped":           false,
		"security_group_id": sgID,
		"region":            region,
		"client_ips":        ips,
		"results":           results,
	})
}

func collectEnsureClientIngressIPs(r *http.Request, body map[string]interface{}) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		cidr := normalizePublicHostCIDR(raw)
		if cidr == "" {
			return
		}
		// store host form without mask for response clarity
		host := strings.TrimSuffix(strings.TrimSuffix(cidr, "/32"), "/128")
		if host == "" {
			host = raw
		}
		key := cidr
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, host)
	}
	if body != nil {
		add(strField(body, "client_public_ip"))
		if arr, ok := body["client_public_ips"].([]interface{}); ok {
			for _, v := range arr {
				add(strings.TrimSpace(toStringAny(v)))
			}
		}
	}
	add(resolveClientIP(r))
	return out
}

func toStringAny(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
