package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

var billHTTP = &http.Client{Timeout: 15 * time.Second}

type insufficientBalanceError struct {
	BalancePoints  int
	RequiredPoints int
	Message        string
}

func (e *insufficientBalanceError) Error() string { return e.Message }

func taskbillForwardStage(path string) string {
	switch {
	case strings.Contains(path, "charge-server-start"):
		return "taskbill_charge_server_start"
	case strings.Contains(path, "check-server-start-balance"):
		return "taskbill_check_server_start_balance"
	case strings.Contains(path, "charge-gitlab-traffic"):
		return "taskbill_charge_gitlab_traffic"
	default:
		return "taskbill_internal"
	}
}

func checkServerStartBalance(ctx context.Context, tenantID, traceID string) error {
	if cfg.TaskBillURL == "" || tenantID == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	path := "/api/internal/taskbill/check-server-start-balance/"
	body, _ := json.Marshal(map[string]string{"tenant_id": tenantID})
	url := strings.TrimRight(cfg.TaskBillURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if cfg.TaskBillInternalSecret != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", cfg.TaskBillInternalSecret)
	}
	stage := taskbillForwardStage(path)
	start := time.Now()
	resp, err := billHTTP.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		tracelog.LogForwardStage(ctx, stage, map[string]any{
			"upstream_status": 502,
			"duration_ms":     duration,
			"path":            path,
			"detail":          "taskBill unreachable",
		})
		return fmt.Errorf("taskBill unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	tracelog.LogForwardStage(ctx, stage, map[string]any{
		"upstream_status": resp.StatusCode,
		"duration_ms":     duration,
		"path":            path,
	})
	if resp.StatusCode == http.StatusPaymentRequired {
		var payload map[string]interface{}
		_ = json.Unmarshal(raw, &payload)
		return &insufficientBalanceError{
			Message:        billStrField(payload, "error"),
			BalancePoints:  billIntField(payload, "balance_points"),
			RequiredPoints: billIntField(payload, "required_points"),
		}
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("taskBill balance check failed: %d %s", resp.StatusCode, string(raw))
	}
	return nil
}

func chargeServerStart(ctx context.Context, tenantID, cloudEventID, taskID, workspaceID, userID, projectID, traceID string) error {
	if cfg.TaskBillURL == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	path := "/api/internal/taskbill/charge-server-start/"
	body, _ := json.Marshal(map[string]string{
		"tenant_id":      tenantID,
		"cloud_event_id": cloudEventID,
		"task_id":        taskID,
		"workspace_id":   workspaceID,
		"user_id":        userID,
		"project_id":     projectID,
	})
	url := strings.TrimRight(cfg.TaskBillURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if cfg.TaskBillInternalSecret != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", cfg.TaskBillInternalSecret)
	}
	stage := taskbillForwardStage(path)
	start := time.Now()
	resp, err := billHTTP.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		tracelog.LogForwardStage(ctx, stage, map[string]any{
			"upstream_status": 502,
			"duration_ms":     duration,
			"path":            path,
			"cloud_event_id":  cloudEventID,
			"detail":          "taskBill unreachable",
		})
		return fmt.Errorf("taskBill unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	tracelog.LogForwardStage(ctx, stage, map[string]any{
		"upstream_status": resp.StatusCode,
		"duration_ms":     duration,
		"path":            path,
		"cloud_event_id":  cloudEventID,
		"task_id":         taskID,
	})
	if resp.StatusCode == http.StatusPaymentRequired {
		var payload map[string]interface{}
		_ = json.Unmarshal(raw, &payload)
		return &insufficientBalanceError{
			Message:        billStrField(payload, "error"),
			BalancePoints:  billIntField(payload, "balance_points"),
			RequiredPoints: billIntField(payload, "required_points"),
		}
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("taskBill charge failed: %d %s", resp.StatusCode, string(raw))
	}
	return nil
}

func billStrField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func billIntField(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
