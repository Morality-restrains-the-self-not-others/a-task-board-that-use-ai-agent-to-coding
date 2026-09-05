package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var billHTTP = &http.Client{Timeout: 30 * time.Second}

type insufficientBalanceError struct {
	BalancePoints  int
	RequiredPoints int
	Message        string
}

func (e *insufficientBalanceError) Error() string { return e.Message }

// consumeTaskPostQuota 消耗 1 个预购任务帖配额并记录消费流水。
// 返回创建帖的到期时间（"2006-01-02 15:04:05.000000"，UTC），供落库 post_expires_at。
// 以 var 暴露以便测试注入配额不足/失败路径（与 publishTaskCreatedFn 同模式）。
var consumeTaskPostQuota = consumeTaskPostQuotaImpl

func consumeTaskPostQuotaImpl(tenantID, taskID, workspaceID, userID, projectID string) (string, error) {
	if cfg.TaskBillURL == "" {
		// 未配置计费服务（开发/测试环境）：默认 12 个月有效期的本地兜底
		return time.Now().UTC().AddDate(0, 12, 0).Format("2006-01-02 15:04:05.000000"), nil
	}
	body, _ := json.Marshal(map[string]string{
		"tenant_id":       tenantID,
		"task_id":         taskID,
		"workspace_id":    workspaceID,
		"user_id":         userID,
		"project_id":      projectID,
		"idempotency_key": "task_post_quota:" + taskID,
	})
	url := strings.TrimRight(cfg.TaskBillURL, "/") + "/api/internal/taskbill/consume-task-post-quota/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.TaskBillInternalSecret != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", cfg.TaskBillInternalSecret)
	}
	resp, err := billHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("taskBill unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 402 {
		var payload map[string]interface{}
		_ = json.Unmarshal(raw, &payload)
		return "", &insufficientBalanceError{
			Message:        strMapField(payload, "error"),
			BalancePoints:  intField(payload, "balance_points"),
			RequiredPoints: intField(payload, "required_points"),
		}
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("taskBill consume quota failed: %d %s", resp.StatusCode, string(raw))
	}
	var payload map[string]interface{}
	_ = json.Unmarshal(raw, &payload)
	return strMapField(payload, "expires_at"), nil
}

// consumeTaskPostRenewal 续存任务帖 — 消耗 1 个创建帖次数，返回新的到期时间。
// 续存 = max(now, current_expires_at) + 12 个月；current_expires_at 可为空（视为新帖）。
func consumeTaskPostRenewal(tenantID, taskID, workspaceID, userID, projectID, currentExpiresAt string) (string, error) {
	if cfg.TaskBillURL == "" {
		base := time.Now().UTC()
		if parsed, err := time.Parse("2006-01-02 15:04:05.000000", currentExpiresAt); err == nil && parsed.After(base) {
			base = parsed
		}
		return base.AddDate(0, 12, 0).Format("2006-01-02 15:04:05.000000"), nil
	}
	body, _ := json.Marshal(map[string]string{
		"tenant_id":          tenantID,
		"task_id":            taskID,
		"workspace_id":       workspaceID,
		"user_id":            userID,
		"project_id":         projectID,
		"current_expires_at": currentExpiresAt,
		"idempotency_key":    "task_post_renewal:" + taskID,
	})
	url := strings.TrimRight(cfg.TaskBillURL, "/") + "/api/internal/taskbill/consume-task-post-renewal/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.TaskBillInternalSecret != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", cfg.TaskBillInternalSecret)
	}
	resp, err := billHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("taskBill unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 402 {
		var payload map[string]interface{}
		_ = json.Unmarshal(raw, &payload)
		return "", &insufficientBalanceError{
			Message:        strMapField(payload, "error"),
			BalancePoints:  intField(payload, "balance_points"),
			RequiredPoints: intField(payload, "required_points"),
		}
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("taskBill renew failed: %d %s", resp.StatusCode, string(raw))
	}
	var payload map[string]interface{}
	_ = json.Unmarshal(raw, &payload)
	return strMapField(payload, "expires_at"), nil
}

func strMapField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func intField(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
