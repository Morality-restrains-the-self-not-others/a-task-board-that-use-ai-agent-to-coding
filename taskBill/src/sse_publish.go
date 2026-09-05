package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

// publishOrderPaidSSEFn 可注入以便测试统计/短路；生产默认走 HTTP 发布。
var publishOrderPaidSSEFn = publishOrderPaidSSE

// ssePublishHTTP 独立连接池，避免与支付主链路共享 DefaultClient 无超时风险。
var ssePublishHTTP = &http.Client{Timeout: 5 * time.Second}

// publishOrderPaidSSE 在订单支付落地后经 taskSSE /internal/publish 推送
// order_paid 事件。taskSSE 的 normalizeInboundMessage 无 task_id 时按 user_id
// 推导 billing:user:{userId} hub key，与前端 /api/sse/recharge-events/ 订阅对齐。
// 失败仅记录日志，不阻塞支付主链路。
func publishOrderPaidSSE(ctx context.Context, order *ResourceOrder) {
	url := strings.TrimSpace(cfg.TaskSseURL)
	if url == "" {
		return
	}
	uid := strings.TrimSpace(order.UserID)
	if uid == "" {
		log.Printf("[taskBill] event=order_sse_skip order=%d user_id_empty", order.ID)
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	statusData := map[string]interface{}{
		"event_name":   "order_paid",
		"status":       "completed",
		"order_id":     order.ID,
		"order_number": order.OrderNumber,
		"tenant_id":    order.TenantID,
		"user_id":      uid,
		"total_yuan":   centsToYuanStr(order.TotalYuanCents),
	}
	body, _ := json.Marshal(map[string]interface{}{
		"user_id":     uid,
		"status_data": statusData,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(url, "/")+"/internal/publish", bytes.NewReader(body))
	if err != nil {
		log.Printf("[taskBill] event=order_sse_build_err order=%d: %v", order.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.TaskSseSecret != "" {
		req.Header.Set("X-Task-Sse-Secret", cfg.TaskSseSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := ssePublishHTTP.Do(req)
	if err != nil {
		log.Printf("[taskBill] event=order_sse_err order=%d: %v", order.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		log.Printf("[taskBill] event=order_sse_status order=%d status=%d body=%s",
			order.ID, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
}
