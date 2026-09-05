package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"tracelog"
)

func handleRegisterReachability(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	// 终态 410 已由 handleContainerInboundToken 的 refuseInboundIfTaskTerminal 处理。
	// 禁止再仅凭 leftover terminal_released 拒绝新一代实例的可达性注册。

	prevServerURL := cfgRow.ServerURL
	updated := false
	if v, present, errMsg := optionalHTTPURL(body["server_url"]); present {
		if errMsg != "" {
			writeErrorJSON(w, nil, http.StatusBadRequest, "server_url 必须是 http/https URL")
			return
		}
		cfgRow.ServerURL = v
		updated = true
	}
	if v, present, errMsg := optionalHTTPURL(body["business_api_endpoint"]); present {
		if errMsg != "" {
			writeErrorJSON(w, nil, http.StatusBadRequest, "business_api_endpoint 必须是 http/https URL")
			return
		}
		cfgRow.BusinessAPIEndpoint = v
		updated = true
	}
	if v, present, errMsg := optionalHTTPURL(body["container_vscode_url"]); present {
		if errMsg != "" {
			writeErrorJSON(w, nil, http.StatusBadRequest, "container_vscode_url 必须是 http/https URL")
			return
		}
		cfgRow.ContainerVscodeURL = v
		updated = true
	}
	if raw := strings.TrimSpace(fmt.Sprintf("%v", body["public_ip"])); raw != "" && raw != "<nil>" {
		cfgRow.PublicIP = raw
		updated = true
	}
	if !updated {
		writeErrorJSON(w, nil, http.StatusBadRequest, "至少提供 public_ip、server_url、business_api_endpoint、container_vscode_url 之一")
		return
	}
	if err := upsertCloudServerConfig(*cfgRow); err != nil {
		writeErrorJSON(w, nil, http.StatusInternalServerError, err.Error())
		return
	}
	_ = maybeClearIdleOnServerURLSet(cfgRow, prevServerURL)
	resetHeartbeatSession(cfgRow)
	_ = publishTaskSSE(ctx, cfgRow.TaskID, cfgRow.CommentID, map[string]interface{}{
		"status":     "container_task_ui_ready",
		"event_name": "server_status_update",
		"message":    "容器可达地址已注册",
		"progress":   100,
	})
	_ = publishContainerTaskUIContextSSE(tenantID, workspaceID, taskID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"task_id": taskID,
	})
}

func handleContainerHeartbeat(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID, accessToken string) {
	containerSeq := parseNonNegInt(body["seq"])
	containerAck := parseNonNegInt(body["ack"])
	msg := strings.TrimSpace(fmt.Sprintf("%v", body["message"]))
	if msg == "<nil>" {
		msg = ""
	}
	if len(msg) > 500 {
		msg = msg[:500]
	}

	hbMu.Lock()
	sess := hbSessions[hbKey(cfgRow)]
	if sess == nil {
		sess = &heartbeatSession{}
		hbSessions[hbKey(cfgRow)] = sess
	}
	uplinkOK := true
	if containerSeq != nil {
		// 与 Django process_container_heartbeat_round 对齐：容器重启后 seq 从 1 重计时重置会话
		if *containerSeq > 0 && (sess.lastContainerSeq == 0 || *containerSeq < sess.lastContainerSeq) {
			sess.lastContainerSeq = 0
			sess.lastSaasSeq = 0
			sess.lastContainerAck = 0
			uplinkOK = true
			sess.lastContainerSeq = *containerSeq
		} else if *containerSeq > sess.lastContainerSeq {
			uplinkOK = true
			sess.lastContainerSeq = *containerSeq
		} else if *containerSeq == sess.lastContainerSeq && *containerSeq > 0 {
			uplinkOK = true // 重传同一 seq
		} else if *containerSeq < sess.lastContainerSeq {
			uplinkOK = false
		}
	}
	ackConfirmsDownlink := false
	if containerAck != nil {
		if sess.lastSaasSeq > 0 && *containerAck >= sess.lastSaasSeq {
			sess.lastContainerAck = *containerAck
			ackConfirmsDownlink = true
		} else {
			sess.lastContainerAck = *containerAck
		}
	}
	saasSeq := sess.lastSaasSeq + 1
	sess.lastSaasSeq = saasSeq
	probeOK, downlinkOK, probeFresh, saasAck := heartbeatCachedProbe(sess, ackConfirmsDownlink)
	needProbe := !probeFresh
	if needProbe && sess.probeInFlight {
		needProbe = false
	}
	if needProbe {
		sess.probeInFlight = true
	}
	hbMu.Unlock()

	if ackConfirmsDownlink {
		downlinkOK = true
	}
	bidirectionalOK := uplinkOK && downlinkOK

	round := map[string]interface{}{
		"container_seq":    nilOrInt(containerSeq),
		"container_ack":    nilOrInt(containerAck),
		"saas_seq":         saasSeq,
		"saas_ack":         nilOrInt(saasAck),
		"uplink_ok":        uplinkOK,
		"downlink_ok":      downlinkOK,
		"probe_ok":         probeOK,
		"bidirectional_ok": bidirectionalOK,
	}
	if msg == "" {
		msg = "容器心跳上报"
	}
	// Persist last heartbeat timestamp so cold-open UI can show "双向已连接" immediately,
	// without waiting for the next heartbeat round (OPT-20260719-034).
	_, _ = db.Exec(`UPDATE cloud_server_configs SET last_heartbeat_at=CURRENT_TIMESTAMP WHERE id=?`, cfgRow.ID)
	applyInstructionIdleFromHeartbeat(ctx, cfgRow, body)
	publishContainerHeartbeatSSE(ctx, cfgRow, msg, uplinkOK, downlinkOK, probeOK, saasSeq, saasAck, containerSeq, containerAck)
	if needProbe {
		scheduleHeartbeatDownlinkProbe(ctx, cfgRow, saasSeq, accessToken, msg, uplinkOK)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":           "ok",
		"task_id":          taskID,
		"ack":              nilOrInt(&saasSeq),
		"seq":              saasSeq,
		"uplink_ok":        uplinkOK,
		"downlink_ok":      downlinkOK,
		"probe_ok":         probeOK,
		"bidirectional_ok": bidirectionalOK,
		"round":            round,
	})
}

// cloneProgressForwardedMu 保护 cloneProgressForwarded 高水位：评论级/任务级克隆进度
// 只升不降，挡住旧容器镜像迟到上报的低百分比（FE 已单调，Cloud 兜底不等换镜像）。
var (
	cloneProgressForwardedMu sync.Mutex
	cloneProgressForwarded   = map[string]int{}
)

// publishCloneProgressSSE 便于单测拦截 SSE 发布（默认走真实 Kafka 发布）。
var publishCloneProgressSSE = publishTaskSSE

var (
	cloneProgressFailureRe = regexp.MustCompile(`失败|未完成|fatal|fatal:`)
	cloneProgressResetRe   = regexp.MustCompile(`准备第\s*\d+\s*/\s*\d+\s*次重试|【重新克隆】\s*开始`)
)

func cloneProgressStateKey(commentID, repoURL string) string {
	return strings.TrimSpace(commentID) + "\x00" + strings.TrimSpace(repoURL)
}

// shouldDropCloneProgress 判断迟到低进度是否应被忽略：低于该 comment_id+repo_url 已转发的
// 高水位、且非失败/重试/重新克隆语义时返回 true（避免 100% 后被迟到 9% 覆盖）。
func shouldDropCloneProgress(commentID, repoURL string, next int, message string) bool {
	message = strings.TrimSpace(message)
	if cloneProgressFailureRe.MatchString(message) {
		return false
	}
	if cloneProgressResetRe.MatchString(message) {
		return false
	}
	if next >= 100 {
		return false
	}
	key := cloneProgressStateKey(commentID, repoURL)
	cloneProgressForwardedMu.Lock()
	defer cloneProgressForwardedMu.Unlock()
	prev, ok := cloneProgressForwarded[key]
	return ok && next < prev
}

func recordCloneProgressForwarded(commentID, repoURL string, progress int) {
	key := cloneProgressStateKey(commentID, repoURL)
	cloneProgressForwardedMu.Lock()
	defer cloneProgressForwardedMu.Unlock()
	if len(cloneProgressForwarded) >= 4096 {
		// 有界缓存，防极端多评论/多仓长期运行下内存无界增长；进程重启丢失可接受。
		cloneProgressForwarded = map[string]int{key: progress}
		return
	}
	cloneProgressForwarded[key] = progress
}

func handleGitCloneProgress(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	p := 0
	if raw := body["progress"]; raw != nil {
		if n := parseNonNegInt(raw); n != nil {
			p = *n
		}
	}
	if p > 100 {
		p = 100
	}
	message := strings.TrimSpace(fmt.Sprintf("%v", body["message"]))
	if message == "" || message == "<nil>" {
		message = fmt.Sprintf("容器内 Git 克隆进度 %d%%", p)
	}
	repoURL := strings.TrimSpace(fmt.Sprintf("%v", body["repo_url"]))
	if repoURL == "<nil>" {
		repoURL = ""
	}
	if shouldDropCloneProgress(cfgRow.CommentID, repoURL, p, message) {
		// 旧容器镜像仍可能按 overall=recv 上报迟到低进度；忽略但不报错（容器无需重试）。
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "task_id": taskID, "progress": p, "dropped": true})
		return
	}
	recordCloneProgressForwarded(cfgRow.CommentID, repoURL, p)
	statusData := map[string]interface{}{
		"status":     "container_git_clone_progress",
		"event_name": "server_status_update",
		"progress":   p,
		"message":    message,
	}
	if repoURL != "" {
		statusData["repo_url"] = repoURL
	}
	if seg, ok := body["segment"]; ok {
		statusData["segment"] = seg
	}
	if t, ok := body["bootstrap_log_text"]; ok {
		statusData["bootstrap_log_text"] = t
	}
	if s, ok := body["bootstrap_log_segments"]; ok {
		statusData["bootstrap_log_segments"] = s
	}
	_ = publishCloneProgressSSE(ctx, cfgRow.TaskID, cfgRow.CommentID, statusData)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "task_id": taskID, "progress": p})
}

// handleBootProgress 接收云主机 UserData / init_from_task2app.sh 初始化逐步进度，
// 经 SSE 转发到任务详情前端（评论级启动日志依赖 comment_id）。
// 不使用 status=success（避免误标服务器已就绪）；进度上限 99。
func handleBootProgress(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	cfgRow = resolveBootProgressCSC(cfgRow, tenantID, workspaceID, taskID)
	statusData := buildBootProgressStatusData(cfgRow, body, taskID)
	// 实例脚本 TRACE_ID（启机 trace）优先写入 SSE，便于评论「启动 TraceId」与 Loki 对齐
	if tid := trim(fmt.Sprintf("%v", body["trace_id"])); tid != "" && tid != "<nil>" && tid != "-" {
		statusData["trace_id"] = tid
	}
	commentID := ""
	if cfgRow != nil {
		commentID = trim(cfgRow.CommentID)
	}
	_ = publishTaskSSE(ctx, taskID, commentID, statusData)
	promoteCommentCSCRunningOnBootEvidence(cfgRow)
	p := 0
	if raw, ok := statusData["progress"].(int); ok {
		p = raw
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "task_id": taskID, "progress": p})
}

// resolveBootProgressCSC：token 落到任务级 CSC（CommentID 空）时，切到带 comment_id 的
// 最新评论级 CSC（含「任务级已挂同一 instance」的 attach/reuse 场景）。
// 旧 UserData COMMENT_ID='-' 不报 comment_id 时，仍须把 userdata_boot SSE 写入评论启动日志。
func resolveBootProgressCSC(cfgRow *CloudServerConfig, tenantID, workspaceID, taskID string) *CloudServerConfig {
	if cfgRow == nil {
		return nil
	}
	if trim(cfgRow.CommentID) != "" {
		return cfgRow
	}
	if db == nil {
		return cfgRow
	}
	withInst, err := loadNewestCloudServerConfigWithInstance(tenantID, workspaceID, taskID)
	if err != nil || withInst == nil || trim(withInst.CommentID) == "" {
		return cfgRow
	}
	return withInst
}

// buildBootProgressStatusData 构造 userdata_boot SSE 载荷（含评论级 log_label）。
func buildBootProgressStatusData(cfgRow *CloudServerConfig, body map[string]any, taskID string) map[string]interface{} {
	p := 0
	if raw := body["progress"]; raw != nil {
		if n := parseNonNegInt(raw); n != nil {
			p = *n
		}
	}
	if p > 99 {
		p = 99
	}
	message := strings.TrimSpace(fmt.Sprintf("%v", body["message"]))
	if message == "" || message == "<nil>" {
		message = fmt.Sprintf("UserData 初始化进度 %d%%", p)
	}
	if len(message) > 500 {
		message = message[:500]
	}
	statusName := strings.TrimSpace(fmt.Sprintf("%v", body["status"]))
	if statusName == "" || statusName == "<nil>" {
		statusName = "processing"
	}
	// 禁止 UserData 侧冒充最终成功/失败关闭态
	switch statusName {
	case "success", "error", "stopped":
		statusName = "processing"
	}
	statusData := map[string]interface{}{
		"status":     statusName,
		"event_name": "server_status_update",
		"progress":   p,
		"message":    message,
		"phase":      "userdata_boot",
	}
	scopeBody := map[string]interface{}{"task_id": taskID}
	if cfgRow != nil {
		if cid := trim(cfgRow.CommentID); cid != "" {
			scopeBody["comment_id"] = cid
			statusData["comment_id"] = cid
		}
		if inst := trim(cfgRow.InstanceID); inst != "" {
			scopeBody["instance_id"] = inst
		}
	}
	if cname := trim(fmt.Sprintf("%v", body["container_name"])); cname != "" && cname != "<nil>" {
		scopeBody["container_name"] = cname
	} else if cfgRow != nil {
		if cid := trim(cfgRow.CommentID); cid != "" {
			scopeBody["container_name"] = deriveStartupContainerName(taskID, cid)
		}
	}
	return withStartupImageLogScope(scopeBody, statusData)
}

func handleLayerGraphPush(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	layers, okL := body["layers"].([]any)
	jobs, okJ := body["jobs"].([]any)
	if !okL || !okJ {
		writeErrorJSON(w, nil, http.StatusBadRequest, "layers 与 jobs 须为 JSON 数组")
		return
	}
	statusData := map[string]interface{}{
		"status":     "container_layer_graph",
		"event_name": "server_status_update",
		"layers":     layers,
		"jobs":       jobs,
	}
	extra := map[string]any{}
	if lr, ok := body["layers_root"].(string); ok && strings.TrimSpace(lr) != "" {
		statusData["layers_root"] = strings.TrimSpace(lr)
		extra["layers_root"] = strings.TrimSpace(lr)
	}
	if bs, ok := body["bootstrap_layer_id"].(string); ok && strings.TrimSpace(bs) != "" {
		statusData["bootstrap_layer_id"] = strings.TrimSpace(bs)
		extra["bootstrap_layer_id"] = strings.TrimSpace(bs)
	}
	if err := persistLayerGraphPush(ctx, cfgRow, layers, jobs, extra, tenantID, workspaceID, taskID); err != nil {
		tracelog.LogForwardStage(ctx, "layer_graph_snapshot_persist_err", map[string]any{
			"error": err.Error(), "task_id": taskID, "workspace_id": workspaceID,
		})
	}
	parentID := ""
	if cfgRow != nil {
		parentID = strings.TrimSpace(cfgRow.CommentID)
	}
	ensureGitPrRepliesFromLayers(ctx, tenantID, workspaceID, taskID, parentID, layers)
	_ = publishTaskSSE(ctx, cfgRow.TaskID, cfgRow.CommentID, statusData)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "task_id": taskID})
}

func handleLayerChangesPush(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	statusData := map[string]interface{}{
		"status":     "container_layer_changes",
		"event_name": "server_status_update",
	}
	for _, k := range []string{"layer_id", "changes", "parent_layer_id", "same", "truncated", "detail"} {
		if v, ok := body[k]; ok {
			statusData[k] = v
		}
	}
	_ = publishTaskSSE(ctx, cfgRow.TaskID, cfgRow.CommentID, statusData)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "task_id": taskID})
}
