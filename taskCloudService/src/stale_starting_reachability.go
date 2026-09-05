package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"tracelog"
)

const (
	defaultStaleStartingReachability = 20 * time.Minute
	staleReachabilityErrorCode       = "CONTAINER_REACHABILITY_TIMEOUT"
	staleReachabilityMessage         = "云主机已运行，但容器服务未在时限内登记可达地址。不是阿里云 API 连不上：实例已 Running，请检查镜像 UserData / 安全组出站 / 容器 HTTP 进程后重试。"
	staleReachabilityProbeTimeout    = 1500 * time.Millisecond
	commentContainerHTTPPort         = "8080"
)

// probeStaleStartingReachability 超时后探活 public_ip:8080。默认走直连 HTTP；测试可替换以免打到文档网段。
var probeStaleStartingReachability = defaultProbeStaleStartingReachability

var staleStartingProbeHTTPClient = tracelog.DirectClient(staleReachabilityProbeTimeout)

// reconcileStaleStartingReachability 收口「VM 已 Running、server_url 仍空」过久的 starting binding。
// 公网 IP 就绪 ≠ 容器 HTTP 已监听（见 persistCloudServerPublicIPByInstanceID）；
// 若 UserData 从未 register-reachability，UI 会永久卡在「启动中」。
// 超时后先探活端口：通则只 warn 并延长宽限，拒绝才 failBinding。
func reconcileStaleStartingReachability(now time.Time, timeout time.Duration) (failed int, err error) {
	if db == nil {
		return 0, nil
	}
	if timeout <= 0 {
		timeout = defaultStaleStartingReachability
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := db.Query(
		`SELECT c.id, b.id
		 FROM cloud_server_configs c
		 INNER JOIN cloud_comment_container_bindings b ON b.csc_id = c.id
		 WHERE TRIM(COALESCE(c.comment_id,'')) != ''
		   AND TRIM(COALESCE(c.instance_id,'')) != ''
		   AND TRIM(COALESCE(c.server_url,'')) = ''
		   AND LOWER(TRIM(COALESCE(c.last_runtime_status,''))) = 'running'
		   AND b.status = ?`,
		ccbStatusStarting,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type pair struct{ cscID, bindingID string }
	var candidates []pair
	for rows.Next() {
		var cscID, bindingID string
		if err := rows.Scan(&cscID, &bindingID); err != nil {
			return 0, err
		}
		cscID = trim(cscID)
		bindingID = trim(bindingID)
		if cscID == "" || bindingID == "" {
			continue
		}
		candidates = append(candidates, pair{cscID: cscID, bindingID: bindingID})
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	cscIDs := make([]string, 0, len(candidates))
	for _, p := range candidates {
		cscIDs = append(cscIDs, p.cscID)
	}
	byCSC, loadErr := loadCloudServerConfigsByIDs(cscIDs)
	if loadErr != nil {
		return 0, loadErr
	}

	for _, p := range candidates {
		cfg := byCSC[p.cscID]
		if cfg == nil {
			continue
		}
		if trim(cfg.ServerURL) != "" {
			continue
		}
		b, loadErr := loadCommentContainerBinding(cfg.CompanyID, cfg.TaskID, cfg.CommentID)
		if loadErr != nil || b == nil || trim(b.ID) != p.bindingID {
			continue
		}
		if b.Status != ccbStatusStarting {
			continue
		}
		anchor := staleStartingAnchor(cfg, b)
		if anchor.IsZero() || now.Sub(anchor) < timeout {
			continue
		}
		if probeStaleStartingReachability(cfg) {
			extendStaleStartingGrace(cfg, b, now)
			continue
		}
		if failBindingForStaleReachability(cfg, b) {
			failed++
		}
	}
	if failed > 0 {
		logInfo("event=stale_starting_reachability_reconciled failed="+strconv.Itoa(failed), "")
	}
	return failed, nil
}

func commentContainerProbeHostPort(publicIP string) string {
	s := trim(publicIP)
	if s == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
	}
	return net.JoinHostPort(s, commentContainerHTTPPort)
}

func defaultProbeStaleStartingReachability(cfg *CloudServerConfig) bool {
	if cfg == nil {
		return false
	}
	addr := commentContainerProbeHostPort(cfg.PublicIP)
	if addr == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), staleReachabilityProbeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/", nil)
	if err != nil {
		return false
	}
	client := staleStartingProbeHTTPClient
	if client == nil {
		client = tracelog.DirectClient(staleReachabilityProbeTimeout)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return true
}

func extendStaleStartingGrace(cfg *CloudServerConfig, b *CommentContainerBinding, now time.Time) {
	traceID := ""
	commentID := ""
	cscID := ""
	instanceID := ""
	if b != nil {
		traceID = b.StartTraceID
		commentID = b.CommentID
	}
	if cfg != nil {
		cscID = cfg.ID
		instanceID = cfg.InstanceID
	}
	logWarn("event=stale_starting_reachability_http_ready comment_id="+commentID+" csc_id="+cscID+" instance_id="+instanceID+" action=extend_grace", traceID)
	if db == nil || cfg == nil || trim(cfg.ID) == "" {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := db.Exec(
		`UPDATE cloud_server_configs SET updated_at=? WHERE id=?`,
		formatMySQLUTCDateTime(now), cfg.ID,
	)
	if err != nil {
		logWarn("event=stale_starting_reachability_extend_grace_failed csc_id="+cfg.ID+" err="+err.Error(), traceID)
	}
}

func staleStartingAnchor(cfg *CloudServerConfig, b *CommentContainerBinding) time.Time {
	if cfg != nil && !cfg.CreatedAt.IsZero() {
		return cfg.CreatedAt.UTC()
	}
	if b != nil && !b.CreatedAt.IsZero() {
		return b.CreatedAt.UTC()
	}
	return time.Time{}
}

func failBindingForStaleReachability(cfg *CloudServerConfig, b *CommentContainerBinding) bool {
	if cfg == nil || b == nil {
		return false
	}
	msg := staleReachabilityMessage
	if err := appendCommentContainerBindingLogMessage(b, ccbStageServerFailed, msg); err != nil {
		logWarn("event=stale_starting_reachability_log_failed comment_id="+b.CommentID+" err="+err.Error(), b.StartTraceID)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if _, err := db.Exec(
		`UPDATE cloud_server_configs SET error_reason=?, updated_at=? WHERE id=?`,
		msg, now, cfg.ID,
	); err != nil {
		logWarn("event=stale_starting_reachability_error_reason_failed csc_id="+cfg.ID+" err="+err.Error(), b.StartTraceID)
	}
	markCommentBindingFailedAfterStartError(b)
	statusData := map[string]interface{}{
		"status":         "error",
		"message":        msg,
		"error_code":     staleReachabilityErrorCode,
		"progress":       0,
		"event_name":     "server_status_update",
		"runtime_status": machineRuntimeRunning,
		"tenant_id":      cfg.CompanyID,
		"workspace_id":   cfg.WorkspaceID,
		"comment_id":     cfg.CommentID,
	}
	if tid := trim(b.StartTraceID); tid != "" {
		statusData["trace_id"] = tid
	}
	// 走 publishSSEMessage 而非 publishTaskSSE：后者会 persistCommentCSCStartFailure，
	// 把 last_runtime_status 改成 Failed。此处 VM 仍 Running，只是容器未登记。
	_ = publishSSEMessage(context.Background(), cfg.TaskID, statusData)
	logWarn("event=stale_starting_reachability_failed comment_id="+b.CommentID+" csc_id="+cfg.ID+" instance_id="+cfg.InstanceID, b.StartTraceID)
	return true
}

func handleInternalReconcileStaleStartingBindings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	n, err := reconcileStaleStartingReachability(time.Now().UTC(), defaultStaleStartingReachability)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"failed": n,
	})
}
