package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

// containerHeartbeatProbeTimeout 须明显小于 taskAgentSupport timeouts.heartbeatSec，
// 否则 TAS 先断开 → 请求 ctx 取消 → 旧逻辑下 Kafka SSE 写失败 → 前端永久 idle。
const containerHeartbeatProbeTimeout = 2500 * time.Millisecond

// 30s 内复用上次探测结果，避免每次心跳同步堵 2.5s（OPT-20260827-028）。
const containerHeartbeatProbeReuseTTL = 30 * time.Second

const probeFailBodyMax = 500

func truncateProbeBody(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if len(s) <= probeFailBodyMax {
		return s
	}
	return s[:probeFailBodyMax] + "…"
}

func logContainerHeartbeatProbeFailure(parent context.Context, cfgRow *CloudServerConfig, saasSeq int, reason string, attrs map[string]any) {
	if attrs == nil {
		attrs = map[string]any{}
	}
	attrs["reason"] = reason
	attrs["saas_seq"] = saasSeq
	if cfgRow != nil {
		if tid := strings.TrimSpace(cfgRow.TaskID); tid != "" {
			attrs["task_id"] = tid
		}
		if base := strings.TrimSpace(cfgRow.ServerURL); base != "" {
			attrs["server_url"] = strings.TrimRight(base, "/")
		}
	}
	tracelog.LogForwardStage(parent, "container_heartbeat_probe_fail", attrs)
}

func probeContainerHeartbeat(parent context.Context, cfgRow *CloudServerConfig, saasSeq int, accessToken string) (bool, *int) {
	base := strings.TrimSpace(strings.TrimRight(cfgRow.ServerURL, "/"))
	base = preferLoopbackContainerBaseURL(base)
	if base == "" {
		logContainerHeartbeatProbeFailure(parent, cfgRow, saasSeq, "missing_server_url", nil)
		return false, nil
	}
	// 优先使用本轮已校验的 access_token，避免再调 credential by-scope（可额外吃掉数秒，挤爆 TAS 5s 预算）
	token := strings.TrimSpace(accessToken)
	if token == "" {
		token = fetchTokenByScope(cfgRow.CompanyID, cfgRow.WorkspaceID, cfgRow.TaskID, cfgRow.CommentID)
	}
	if strings.TrimSpace(token) == "" {
		logContainerHeartbeatProbeFailure(parent, cfgRow, saasSeq, "missing_access_token", nil)
		return false, nil
	}
	u := fmt.Sprintf("%s/api/saas-heartbeat-probe?seq=%d&access_token=%s", base, saasSeq, url.QueryEscape(token))
	if parent == nil {
		parent = context.Background()
	}
	// 探测预算独立于入站 ctx：即使 TAS 已断开，仍尽快结束探测并让调用方发布 SSE
	probeParent := context.WithoutCancel(parent)
	ctx, cancel := context.WithTimeout(probeParent, containerHeartbeatProbeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		logContainerHeartbeatProbeFailure(parent, cfgRow, saasSeq, "build_request_failed", map[string]any{
			"detail": err.Error(),
		})
		return false, nil
	}
	// Must propagate full span context: onlineServiceJS rejects X-Trace-Id-only with HTTP 400.
	tracelog.ApplyOutboundHeaders(req, parent)
	resp, err := djangoHTTP.Do(req)
	if err != nil {
		logContainerHeartbeatProbeFailure(parent, cfgRow, saasSeq, "http_error", map[string]any{
			"detail": err.Error(),
		})
		return false, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		logContainerHeartbeatProbeFailure(parent, cfgRow, saasSeq, "bad_status", map[string]any{
			"status": resp.StatusCode,
			"body":   truncateProbeBody(raw),
		})
		return false, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		logContainerHeartbeatProbeFailure(parent, cfgRow, saasSeq, "invalid_json", map[string]any{
			"status": resp.StatusCode,
			"body":   truncateProbeBody(raw),
			"detail": err.Error(),
		})
		return false, nil
	}
	ack := parseNonNegInt(out["ack"])
	return true, ack
}

func heartbeatCachedProbe(sess *heartbeatSession, ackConfirmsDownlink bool) (probeOK, downlinkOK, fresh bool, saasAck *int) {
	if sess == nil || sess.lastProbeAt.IsZero() {
		return false, ackConfirmsDownlink, false, nil
	}
	if time.Since(sess.lastProbeAt) >= containerHeartbeatProbeReuseTTL {
		return sess.lastProbeOK, sess.lastDownlinkOK || ackConfirmsDownlink, false, nil
	}
	if sess.lastProbeAckSet {
		v := sess.lastProbeAckVal
		saasAck = &v
	}
	return sess.lastProbeOK, sess.lastDownlinkOK || ackConfirmsDownlink, true, saasAck
}

func scheduleHeartbeatDownlinkProbe(parent context.Context, cfgRow *CloudServerConfig, saasSeq int, accessToken, msg string, uplinkOK bool) {
	if cfgRow == nil {
		return
	}
	rowCopy := *cfgRow
	tok := accessToken
	go func() {
		defer func() {
			hbMu.Lock()
			if s := hbSessions[hbKey(&rowCopy)]; s != nil {
				s.probeInFlight = false
			}
			hbMu.Unlock()
		}()
		probeOK, saasAck := probeContainerHeartbeat(parent, &rowCopy, saasSeq, tok)
		downlinkOK := probeOK && saasAck != nil && *saasAck == saasSeq
		hbMu.Lock()
		if s := hbSessions[hbKey(&rowCopy)]; s != nil {
			s.lastProbeOK = probeOK
			s.lastDownlinkOK = downlinkOK
			s.lastProbeAt = time.Now()
			if saasAck != nil {
				s.lastProbeAckVal = *saasAck
				s.lastProbeAckSet = true
			}
		}
		hbMu.Unlock()
		publishContainerHeartbeatSSE(parent, &rowCopy, msg, uplinkOK, downlinkOK, probeOK, saasSeq, saasAck, nil, nil)
	}()
}

func publishContainerHeartbeatSSE(ctx context.Context, cfgRow *CloudServerConfig, msg string, uplinkOK, downlinkOK, probeOK bool, saasSeq int, saasAck, containerSeq, containerAck *int) {
	if cfgRow == nil {
		return
	}
	if msg == "" {
		msg = "容器心跳上报"
	}
	status := "partial"
	if uplinkOK && downlinkOK {
		status = "ok"
	}
	commentID := strings.TrimSpace(cfgRow.CommentID)
	containerName := ""
	if commentID != "" {
		containerName = buildCommentMockContainerName(cfgRow.TaskID, commentID)
	}
	_ = publishSSEMessage(ctx, cfgRow.TaskID, map[string]interface{}{
		"event_name":       "container_heartbeat",
		"status":           status,
		"message":          msg,
		"bidirectional_ok": uplinkOK && downlinkOK,
		"uplink_ok":        uplinkOK,
		"downlink_ok":      downlinkOK,
		"probe_ok":         probeOK,
		"container_seq":    nilOrInt(containerSeq),
		"container_ack":    nilOrInt(containerAck),
		"saas_seq":         saasSeq,
		"saas_ack":         nilOrInt(saasAck),
		"comment_id":       commentID,
		"container_name":   containerName,
	})
}
