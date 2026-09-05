package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// serverContentHTTPClient 可注入（单测替换 Transport 模拟连接拒绝）。
var serverContentHTTPClient = &http.Client{Timeout: 8 * time.Second}

func handleServerContent(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少任务ID"})
		return
	}
	if tenantID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "无法获取租户ID"})
		return
	}
	commentID := commentIDFromComputeRequest(r, nil)
	if rejectMissingComputeCommentID(w, r, commentID) {
		return
	}
	cfg, err := resolveScopedCloudServerConfig(tenantID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{"status": "error", "message": "未找到任务对应的服务器配置"})
		return
	}
	publicIP := ensureCloudServerConfigPublicIP(cfg)
	if publicIP == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "服务器公网 IP 不存在，无法拉取服务器内容",
		})
		return
	}
	targetURL := "http://" + publicIP + ":8080"
	resp, err := serverContentHTTPClient.Get(targetURL)
	if err != nil {
		// OPT-20260812-042：公网 IP 已回填但容器业务端口尚未监听（UserData/容器仍在启动中），
		// connection refused 是「启动中」信号——返回可区分的 status=starting，前端降级展示
		// 而非硬错误；其余网络错误仍 502。
		if errors.Is(err, syscall.ECONNREFUSED) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status":     "starting",
				"target_url": targetURL,
				"message":    "容器业务端口尚未就绪，正在启动中",
			})
			return
		}
		writeErrorMapJSON(w, r, http.StatusBadGateway, map[string]interface{}{
			"status": "error", "message": "拉取服务器内容失败: " + err.Error(), "target_url": targetURL,
		})
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	text := string(raw)
	truncated := len(text) > 20000
	if truncated {
		text = text[:20000]
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "success",
		"target_url":   targetURL,
		"http_status":  resp.StatusCode,
		"content_type": resp.Header.Get("Content-Type"),
		"content":      text,
		"truncated":    truncated,
	})
}

// ensureCloudServerConfigPublicIP 优先用 CSC.public_ip；为空时用 instance_id 直查云厂商并回填。
// 修复：start-vm 仅把 instance_id 落库、poll 到的公网 IP 只进 SSE 时，server-content 永久 400。
func ensureCloudServerConfigPublicIP(cfg *CloudServerConfig) string {
	if cfg == nil {
		return ""
	}
	if ip := strings.TrimSpace(cfg.PublicIP); ip != "" {
		return ip
	}
	instanceID := strings.TrimSpace(cfg.InstanceID)
	authID := strings.TrimSpace(cfg.AuthorizationID)
	region := strings.TrimSpace(cfg.Region)
	companyID := strings.TrimSpace(cfg.CompanyID)
	if instanceID == "" || authID == "" || region == "" || companyID == "" {
		return ""
	}
	auth, err := loadCloudAuth(companyID, authID)
	if err != nil || auth == nil {
		return ""
	}
	ip, _ := queryInstancePublicIPFn(auth.SecretID, auth.SecretKey, region, instanceID)
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	// 只回填 public_ip；勿把推测性 http://ip:8080 写入 CSC.server_url / 内存字段，
	// 否则 binding 会在容器未监听前被 promote 为「服务可用」。
	persistCloudServerPublicIPByInstanceID(instanceID, ip, "")
	cfg.PublicIP = ip
	return ip
}
