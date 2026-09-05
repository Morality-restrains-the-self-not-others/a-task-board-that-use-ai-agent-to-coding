package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// handleModelBudgetUsage：Cloud 校验 items 后直写 task_budget.db 账本（不再经 Django ORM）。
func handleModelBudgetUsage(
	w http.ResponseWriter,
	ctx context.Context,
	body map[string]any,
	tenantID, workspaceID, taskID string,
) {
	_ = ctx
	items, ok := body["items"].([]any)
	if !ok {
		writeErrorJSON(w, nil, http.StatusBadRequest, "items 必须是数组")
		return
	}
	for _, raw := range items {
		item, isObj := raw.(map[string]any)
		if !isObj {
			writeErrorJSON(w, nil, http.StatusBadRequest, "items 元素必须是对象")
			return
		}
		provider := strings.TrimSpace(fmt.Sprintf("%v", item["provider"]))
		modelName := strings.TrimSpace(fmt.Sprintf("%v", item["model_name"]))
		idem := strings.TrimSpace(fmt.Sprintf("%v", item["idempotency_key"]))
		if provider == "" || provider == "<nil>" || modelName == "" || modelName == "<nil>" || idem == "" || idem == "<nil>" {
			writeErrorJSON(w, nil, http.StatusBadRequest, "provider、model_name、idempotency_key 必填")
			return
		}
	}
	out, status, err := recordBudgetUsageBatchLocal(tenantID, workspaceID, taskID, items)
	if err != nil && status >= 500 {
		writeErrorJSON(w, nil, status, err.Error())
		return
	}
	if out == nil {
		writeErrorJSON(w, nil, status, "budget write failed")
		return
	}
	writeJSON(w, status, out)
}

// handleFeatureParamsEnv：Cloud 鉴权后 Go 完整层级 resolve + snapshot 写；不再 HTTP fallback Django。
func handleFeatureParamsEnv(
	w http.ResponseWriter,
	ctx context.Context,
	cfgRow *CloudServerConfig,
	body map[string]any,
	tenantID, workspaceID, taskID, accessToken string,
) {
	_ = body
	out, err := resolveFeatureParamsEnvLocal(ctx, getBudgetDB(), tenantID, workspaceID, taskID, cfgRow.CompanyID, accessToken)
	if err != nil {
		writeErrorJSON(w, nil, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleRelayStatusPush：立即返回 {status,task_id,ack}；同进程 Redis converge + SSE publish。
func handleRelayStatusPush(
	w http.ResponseWriter,
	ctx context.Context,
	cfgRow *CloudServerConfig,
	body map[string]any,
	tenantID, workspaceID, taskID, accessToken, traceID string,
) {
	_ = cfgRow
	_ = accessToken
	seq := parseNonNegInt(body["seq"])
	relayStatus, _ := body["status"].(map[string]any)
	if relayStatus == nil {
		relayStatus = map[string]any{}
	}

	ackVal := any(nil)
	if seq != nil {
		ackVal = *seq
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"task_id": taskID,
		"ack":     ackVal,
	})

	bgCtx := ctx
	if bgCtx == nil {
		bgCtx = context.Background()
	} else {
		bgCtx = context.WithoutCancel(ctx)
	}
	tid := strings.TrimSpace(traceID)
	go func() {
		_ = applyRelayStatusPushLocal(bgCtx, tenantID, workspaceID, taskID, seq, relayStatus, tid)
	}()
}
