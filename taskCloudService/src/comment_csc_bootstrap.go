package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

// errCommentCSCPlatformUnsupported 评论 CSC 平台缺失或为 mock 的永久失败信号。
// 调用方（ccbStartBinding / attach / promote）据此区分「未配置云平台（应收口 failed）」
// 与「可等待 reachability（mock 人工回填路径，保持 starting）」，见 OPT-20260812-010。
var errCommentCSCPlatformUnsupported = errors.New("评论级容器启动需要配置云平台（platform 缺失或为 mock）")

// commentCSCReachabilityPossible 判断评论 CSC 是否仍有可能获得 reachability（可等待），而非永久失败。
// 评论 CSC 或任务级 base 任一已登记 server_url/business_api_endpoint/运行闭环即视为可等待
// （mock 人工回填路径：测试/运维可手工写入 server_url 后经 promote 升 running）。
// 两者均无可达性 → 未配置云平台 → bootstrap 永久失败应收口 failed。
func commentCSCReachabilityPossible(csc *CloudServerConfig) bool {
	if csc == nil {
		return false
	}
	if commentCSCHasRuntime(csc) || cscHasReachableEndpoint(csc) {
		return true
	}
	base, err := loadCloudServerConfig(csc.CompanyID, csc.WorkspaceID, csc.TaskID)
	if err != nil || base == nil {
		return false
	}
	return commentCSCHasRuntime(base) || cscHasReachableEndpoint(base)
}

// bootstrapCommentCSCRuntime 在评论级 CSC 行创建后，填满 instance_id/server_url（闭环）。
//   - platform 为空或 mock：返回哨兵错误 errCommentCSCPlatformUnsupported（永久失败信号）。
//     调用方据此区分「未配置云平台（应收口 failed）」与「mock 人工回填 reachability 路径（保持 starting）」，见 OPT-20260812-010。
//   - 云平台：若已有 instance/server_url 则复用；否则异步触发 start-vm-auto（comment_id）
func bootstrapCommentCSCRuntime(csc *CloudServerConfig) (*CloudServerConfig, error) {
	if csc == nil {
		return nil, fmt.Errorf("csc required")
	}
	commentID := trim(csc.CommentID)
	if commentID == "" {
		// 任务级 CSC 不走本路径
		return csc, nil
	}

	platform := strings.ToLower(trim(csc.Platform))
	if platform == "" || platform == "mock" {
		log.Printf("[taskCloudService] event=comment_csc_bootstrap_platform_unsupported company_id=%s task_id=%s comment_id=%s csc_id=%s platform=%q",
			csc.CompanyID, csc.TaskID, commentID, csc.ID, platform)
		// 返回原 csc 不落库 + 哨兵错误：调用方据此区分「未配置云平台（应收口 failed）」与
		// 「mock 人工回填 reachability 路径（保持 starting）」，见 OPT-20260812-010。
		return csc, errCommentCSCPlatformUnsupported
	}

	// 云平台：已有可达性则视为完成；否则异步触发 start-vm-auto
	if trim(csc.InstanceID) != "" && trim(csc.ServerURL) != "" {
		log.Printf("[taskCloudService] event=comment_csc_bootstrap_cloud_ready company_id=%s task_id=%s comment_id=%s csc_id=%s",
			csc.CompanyID, csc.TaskID, commentID, csc.ID)
		return csc, nil
	}
	// 已有 instance_id 但未登记可达性：VM 正在启动/待登记，勿重复触发 start-vm（否则每轮调度重复建机）
	if trim(csc.InstanceID) != "" {
		log.Printf("[taskCloudService] event=comment_csc_bootstrap_awaiting_runtime company_id=%s task_id=%s comment_id=%s csc_id=%s instance_id=%s",
			csc.CompanyID, csc.TaskID, commentID, csc.ID, csc.InstanceID)
		return csc, nil
	}
	if reason, inst, skip := commentCSCBootstrapShouldSkipStart(csc); skip {
		logInfo(fmt.Sprintf("event=comment_csc_bootstrap_start_inflight_skipped company_id=%s task_id=%s comment_id=%s csc_id=%s reason=%s instance_id=%s",
			csc.CompanyID, csc.TaskID, commentID, csc.ID, reason, inst), "")
		return csc, nil
	}
	log.Printf("[taskCloudService] event=comment_csc_bootstrap_cloud_pending company_id=%s task_id=%s comment_id=%s csc_id=%s platform=%s",
		csc.CompanyID, csc.TaskID, commentID, csc.ID, platform)
	startCommentCSCBootstrap(csc)
	return csc, nil
}

// startCommentCSCBootstrap 可在测试中替换，断言 bootstrap 是否真正发起 start-vm。
var startCommentCSCBootstrap = triggerCommentCSCStartBootstrapAsync

// triggerCommentCSCStartBootstrapAsync 异步触发 start-vm（本服务约定路由，按 comment_id）。
// 瞬时/可重试失败仅打日志（binding 保持 starting，由后续 promote 再触发）。
// start-vm 返回 4xx 等客户端永久失败时收口 binding=failed，避免 UI 永久卡在 Starting。
func triggerCommentCSCStartBootstrapAsync(csc *CloudServerConfig) {
	if csc == nil || trim(csc.CommentID) == "" {
		return
	}
	if commentCSCHasRuntime(csc) {
		return
	}
	snapshot := *csc
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[taskCloudService] event=comment_csc_start_bootstrap_panic comment_id=%s recover=%v",
					snapshot.CommentID, rec)
			}
		}()
		if err := postCommentCSCStartBootstrap(&snapshot); err != nil {
			log.Printf("[taskCloudService] event=comment_csc_start_bootstrap_failed company_id=%s task_id=%s comment_id=%s csc_id=%s mode=start-vm-auto err=%v",
				snapshot.CompanyID, snapshot.TaskID, snapshot.CommentID, snapshot.ID, err)
			if isCommentCSCStartBootstrapPermanent(err) {
				failCommentBindingAfterBootstrapError(&snapshot, err)
			}
			return
		}
		log.Printf("[taskCloudService] event=comment_csc_start_bootstrap_ok company_id=%s task_id=%s comment_id=%s csc_id=%s mode=start-vm-auto",
			snapshot.CompanyID, snapshot.TaskID, snapshot.CommentID, snapshot.ID)
	}()
}

// isCommentCSCStartBootstrapPermanent 判定 bootstrap 出站失败是否为不可自愈的客户端错误。
// start-vm status=4xx（缺 header / 参数非法等）不会因重试变好，应收口 failed。
func isCommentCSCStartBootstrapPermanent(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "start-vm status=4") {
		return true
	}
	// 本地校验失败（缺网络资源 / 缺 invoker / 缺 csc 字段）同样不会自愈
	// 缺 start 事件载荷不算永久失败：任务级云平台 / start-vm 可能尚未落库，
	// 后续 advance 会再次触发 bootstrap（mock 首轮保持 starting，见 OPT-20260812-010）。
	permanentSnippets := []string{
		"image_invoker_user_id required",
		"missing network resources",
		"company/workspace/task/comment required",
	}
	for _, s := range permanentSnippets {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

func failCommentBindingAfterBootstrapError(csc *CloudServerConfig, bootErr error) {
	if csc == nil {
		return
	}
	b, err := loadCommentContainerBinding(trim(csc.CompanyID), trim(csc.TaskID), trim(csc.CommentID))
	if err != nil || b == nil {
		log.Printf("[taskCloudService] event=comment_csc_bootstrap_fail_binding_load_miss comment_id=%s err=%v boot_err=%v",
			trim(csc.CommentID), err, bootErr)
		return
	}
	if id := trim(csc.ID); id != "" {
		b.CSCID = id
	}
	markCommentBindingFailedAfterStartError(b)
}

func postCommentCSCStartBootstrap(csc *CloudServerConfig) error {
	if csc == nil {
		return fmt.Errorf("csc required")
	}
	companyID := trim(csc.CompanyID)
	workspaceID := trim(csc.WorkspaceID)
	taskID := trim(csc.TaskID)
	commentID := trim(csc.CommentID)
	if companyID == "" || workspaceID == "" || taskID == "" || commentID == "" {
		return fmt.Errorf("company/workspace/task/comment required")
	}
	return postCommentCSCStartVM(csc)
}

// postCommentCSCStartVM 直连本服务 /api/cloud/compute/start-vm/ 约定路由启动评论级云服务器。
// 镜像/网络/硬件参数复用任务级最近一次 start 事件的 event_data（ead5583 后网关不再转发 start-vm
// 系列旧 /api/tenant/ 路径，此前 bootstrap 走网关 404 导致评论容器永远卡 starting，见 OPT-20260809-023）。
// 调用方为用户（镜像调用人员 csc.ImageInvokerUserID），X-Auth-User-Id/X-Auth-Tenant-Id 对齐任务级启动。
func postCommentCSCStartVM(csc *CloudServerConfig) error {
	companyID := trim(csc.CompanyID)
	workspaceID := trim(csc.WorkspaceID)
	taskID := trim(csc.TaskID)
	commentID := trim(csc.CommentID)

	ev := loadStartEventPayload(companyID, taskID, commentID)
	if ev == nil {
		return fmt.Errorf("no start event payload (task-level or comment-scoped, event_type=start) for comment csc bootstrap")
	}
	invoker := trim(csc.ImageInvokerUserID)
	if invoker == "" {
		invoker = strField(ev, "image_invoker_user_id")
	}
	if invoker == "" {
		return fmt.Errorf("image_invoker_user_id required for comment csc start-vm")
	}

	payload := map[string]any{
		"tenant_id":    companyID,
		"workspace_id": workspaceID,
		"task_id":      taskID,
		"comment_id":   commentID,
		"csc_id":       trim(csc.ID),
	}
	// UserData __TASK2APP_COMMENT_ID__ / 启动日志路由依赖 comment_id；
	// container_name 供 eventData 与 boot-progress 作用域（task_<task>_<comment>）。
	if taskID != "" && commentID != "" {
		cname := buildCommentMockContainerName(taskID, commentID)
		payload["container_name"] = cname
		payload["mock_container_name"] = cname
	}
	// 镜像 / 网络 / 硬件配置：复用任务级 start 事件参数（评论 CSC 只克隆了 region/auth/platform）
	for _, key := range []string{
		"container_image_id", "cloud_server_image_id", "container_image_url",
		"vpc_id", "security_group_id", "vswitch_id", "hardware_config",
		"region_id", "zone_id",
	} {
		if v, ok := ev[key]; ok {
			payload[key] = v
		}
	}
	if auth := trim(csc.AuthorizationID); auth != "" {
		payload["authorization_id"] = auth
	}
	if plat := trim(csc.Platform); plat != "" {
		payload["platform_type"] = plat
	}
	if region := trim(csc.Region); region != "" {
		payload["region_id"] = region
	}
	payload["image_invoker_user_id"] = invoker

	// 前置校验：任务级 start-vm（既有网络）必须带齐 vpc/sg/vswitch/region
	if strField(payload, "region_id") == "" || strField(payload, "vpc_id") == "" ||
		strField(payload, "security_group_id") == "" || strField(payload, "vswitch_id") == "" {
		return fmt.Errorf("task start event missing network resources (region/vpc/security_group/vswitch) for comment csc start-vm")
	}

	port := cfg.Port
	if port <= 0 {
		port = 8018
	}
	path := fmt.Sprintf("/api/cloud/compute/start-vm/tenant_id/%s/workspace_id/%s/task_id/%s/",
		companyID, workspaceID, taskID)
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-User-Id", invoker)
	req.Header.Set("X-Auth-User-Id", invoker)
	req.Header.Set("X-Auth-Tenant-Id", companyID)
	// 严格 Middleware 拒绝「仅有 X-Trace-Id、无 parent span」的入站请求
	// （RejectTraceIdOnlyHTTP → 400 trace propagation incomplete）。
	// 必须 ApplyOutboundHeaders：当前 span 作为下游 X-Parent-Span-Id。
	// TraceId 优先复用 binding.start_trace_id，便于与页面「启动 TraceId」在 Loki 对齐。
	tracelog.ApplyOutboundHeaders(req, commentCSCBootstrapTraceContext(csc))
	if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	// start-vm 为同步创建（RunInstances + IP 轮询），超时放宽至 90s
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("start-vm status=%d body=%s", resp.StatusCode, truncateForLog(string(body), 240))
	}
	return nil
}

// commentCSCBootstrapTraceContext 为评论 CSC 自调用 start-vm 构造出站关联上下文。
// 优先复用 binding.start_trace_id（页面「启动 TraceId」），否则新建；始终带 SpanID
// 以便 ApplyOutboundHeaders 写出 X-Parent-Span-Id / traceparent。
func commentCSCBootstrapTraceContext(csc *CloudServerConfig) context.Context {
	tid := ""
	if csc != nil {
		if b, err := loadCommentContainerBinding(trim(csc.CompanyID), trim(csc.TaskID), trim(csc.CommentID)); err == nil && b != nil {
			tid = strings.TrimSpace(b.StartTraceID)
		}
		if tid == "" {
			tid = loadPendingBindingStartTraceID(trim(csc.TaskID), trim(csc.CommentID))
		}
	}
	if tid == "" {
		tid = tracelog.NewTraceID()
	}
	return tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{
		TraceID: tid,
		SpanID:  tracelog.NewSpanID(),
	})
}

// loadStartEventPayload 加载最近一次 start 事件的 event_data（含镜像/网络/硬件配置），
// 供评论级 CSC 启动复用启动参数。优先本评论，其次任务级 comment_id=”，
// 再回退同任务任意评论的最近 start（第二条 @镜像 常见只有第一条的 comment-scoped 行）。
func loadStartEventPayload(companyID, taskID, commentID string) map[string]interface{} {
	companyID = trim(companyID)
	taskID = trim(taskID)
	commentID = trim(commentID)
	if companyID == "" || taskID == "" || db == nil {
		return nil
	}
	if commentID != "" {
		var raw []byte
		q := `SELECT event_data FROM cloud_server_events
			WHERE company_id=? AND task_id=? AND event_type='start' AND comment_id=?
			ORDER BY created_at DESC, id DESC LIMIT 1`
		if err := db.QueryRow(q, companyID, taskID, commentID).Scan(&raw); err == nil && len(raw) > 0 {
			var out map[string]interface{}
			if json.Unmarshal(raw, &out) == nil {
				return out
			}
		}
	}
	if tl := loadTaskStartEventPayload(companyID, taskID); tl != nil {
		return tl
	}
	sib := loadLatestTaskStartEventPayload(companyID, taskID)
	if sib != nil {
		// 结构化日志携带独立 trace_id（≠ task_id），便于按页面启动 TraceId 在 Loki 检索。
		logInfo("event=start_event_payload_sibling_fallback company_id="+companyID+" task_id="+taskID+" comment_id="+commentID,
			tracelog.NewTraceID())
	}
	return sib
}

// loadLatestTaskStartEventPayload 取同任务最近一次 start 事件（不限 comment_id）。
// 第二条及后续 @镜像 评论常只有第一条的 comment-scoped 行、没有 comment_id=” 任务级行。
func loadLatestTaskStartEventPayload(companyID, taskID string) map[string]interface{} {
	companyID = trim(companyID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" || db == nil {
		return nil
	}
	var raw []byte
	q := `SELECT event_data FROM cloud_server_events
		WHERE company_id=? AND task_id=? AND event_type='start'
		ORDER BY created_at DESC, id DESC LIMIT 1`
	if err := db.QueryRow(q, companyID, taskID).Scan(&raw); err != nil || len(raw) == 0 {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// loadTaskStartEventPayload 加载任务级最近一次 start 事件的 event_data（含镜像/网络/硬件配置），
// 供评论级 CSC 启动复用任务级启动参数（仅任务级 comment_id=”）。
func loadTaskStartEventPayload(companyID, taskID string) map[string]interface{} {
	companyID = trim(companyID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" || db == nil {
		return nil
	}
	var raw []byte
	q := `SELECT event_data FROM cloud_server_events
		WHERE company_id=? AND task_id=? AND event_type='start' AND COALESCE(comment_id,'')=''
		ORDER BY created_at DESC, id DESC LIMIT 1`
	if err := db.QueryRow(q, companyID, taskID).Scan(&raw); err != nil || len(raw) == 0 {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// commentCSCHasRuntime 判断评论 CSC 是否已具备运行闭环（instance + server_url）。
func commentCSCHasRuntime(csc *CloudServerConfig) bool {
	if csc == nil {
		return false
	}
	return trim(csc.InstanceID) != "" && trim(csc.ServerURL) != ""
}
