package main

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// handleInternalContainerUnreachable — OPT-20260818-008。
// container-gateway 转发到容器 server_url 收到 connection refused 时回调：
// 清除推测性 server_url / business_api_endpoint（保留实例与其余字段），
// 并把对应 comment binding 从 running 降回 starting，等待真实 reachability
// 注册后再晋升。网关侧单次 refused 不代表永久故障，故不清实例、不释放 binding。
func handleInternalContainerUnreachable(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	commentID := strings.TrimSpace(r.URL.Query().Get("comment_id"))
	if tenantID == "" || taskID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "tenant_id and task_id required")
		return
	}

	cfg, err := loadCloudServerConfigForGatewayForward(tenantID, workspaceID, taskID, commentID, "")
	if err != nil || cfg == nil {
		// 无 CSC / 无地址可清：幂等成功，不视为错误
		writeJSON(w, http.StatusOK, map[string]any{"cleared": false, "demoted": false})
		return
	}

	cleared := false
	if trim(cfg.ServerURL) != "" || trim(cfg.BusinessAPIEndpoint) != "" {
		// 直改而非 upsert：cloudServerRuntimeUpsertPreserve 对评论行非终态会保留旧
		// server_url，upsert 无法清掉推测性地址。
		if err := clearSpeculativeReachabilityOnConfig(cfg.ID); err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, "clear reachability failed")
			return
		}
		cleared = true
	}
	demoted := demoteCommentContainerBindingToStarting(cfg, tenantID, taskID)

	writeJSON(w, http.StatusOK, map[string]any{"cleared": cleared, "demoted": demoted})
}

// clearSpeculativeReachabilityOnConfig 直接清空推测性 server_url / business_api_endpoint，
// 绕过 upsert 的运行时保留子句（保留 instance/public_ip/last_runtime_status 等字段）。
func clearSpeculativeReachabilityOnConfig(cscID string) error {
	_, err := db.Exec(
		`UPDATE cloud_server_configs SET server_url='', business_api_endpoint='', updated_at=? WHERE id=?`,
		time.Now().UTC().Format("2006-01-02 15:04:05"), cscID,
	)
	return err
}

// demoteCommentContainerBindingToStarting 把 running 的评论 binding 降回 starting。
// 仅当 binding 当前为 running 时降级；starting/pending/released 原样保留。
func demoteCommentContainerBindingToStarting(cfg *CloudServerConfig, tenantID, taskID string) bool {
	if cfg == nil {
		return false
	}
	commentID := strings.TrimSpace(cfg.CommentID)
	if commentID == "" {
		return false
	}
	b, err := loadCommentContainerBinding(tenantID, taskID, commentID)
	if err != nil || b == nil {
		return false
	}
	if b.Status != ccbStatusRunning {
		return false
	}
	if err := updateCommentContainerBindingStatus(b.ID, ccbStatusStarting); err != nil {
		log.Printf("[taskCloudService] event=comment_container_binding_demote_failed binding_id=%s comment_id=%s err=%v",
			b.ID, commentID, err)
		return false
	}
	b.Status = ccbStatusStarting
	log.Printf("[taskCloudService] event=comment_container_binding_demoted_starting binding_id=%s comment_id=%s task_id=%s reason=upstream_conn_refused",
		b.ID, commentID, taskID)
	return true
}
