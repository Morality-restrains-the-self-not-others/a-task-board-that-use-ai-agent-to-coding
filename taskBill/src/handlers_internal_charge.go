package main

import (
	"net/http"

	"tracelog"
)

func handleInternalChargeServerStart(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	result, err := chargeServerStart(
		r.Context(),
		tid,
		stringField(body, "cloud_event_id"),
		stringField(body, "task_id"),
		stringField(body, "workspace_id"),
		stringField(body, "user_id"),
		stringField(body, "project_id"),
	)
	if err != nil {
		if ib, ok := err.(*InsufficientBalanceError); ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
				"error": ib.Error(), "balance_points": ib.BalancePoints, "required_points": ib.RequiredPoints,
			})
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func handleInternalChargeGitlabTraffic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	in, err := gitlabTrafficMeterReqFromJSON(body)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	result, err := meterGitlabOutboundTraffic(r.Context(), in)
	if err != nil {
		if ib, ok := err.(*InsufficientBalanceError); ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
				"error": ib.Error(), "balance_points": ib.BalancePoints, "required_points": ib.RequiredPoints,
			})
			return
		}
		if tq, ok := err.(*TrafficQuotaExceededError); ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":              tq.Error(),
				"code":               tq.Code,
				"traffic_prepaid_gb": tq.PrepaidGB,
				"traffic_used_gb":    tq.UsedGB,
			})
			return
		}
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleInternalConsumeTaskPostQuota POST /api/internal/taskbill/consume-task-post-quota/
// 消耗 1 个任务帖预购配额并记录消费流水。
// 由 taskTaskService 在创建任务时调用。
func handleInternalConsumeTaskPostQuota(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	idempotencyKey := stringField(body, "idempotency_key")
	result, err := consumeAndRecordTaskPostQuota(
		r.Context(),
		tid,
		stringField(body, "task_id"),
		stringField(body, "workspace_id"),
		stringField(body, "user_id"),
		stringField(body, "project_id"),
		idempotencyKey,
	)
	if err != nil {
		if ib, ok := err.(*InsufficientBalanceError); ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":           ib.Error(),
				"balance_points":  ib.BalancePoints,
				"required_points": ib.RequiredPoints,
				"code":            "INSUFFICIENT_TASK_POST_QUOTA",
			})
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleInternalConsumeTaskPostRenewal POST /api/internal/taskbill/consume-task-post-renewal/
// 续存任务帖：只消耗 1 个创建帖次数（FEFO），不扣钱包。
// 由 taskTaskService 在 /tasks/{id}/renew/ 时调用。
// 请求体: tenant_id, task_id, current_expires_at（当前到期时间，可为空）, idempotency_key
// 响应: expires_at = max(now, current_expires_at) + 12 个月
func handleInternalConsumeTaskPostRenewal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	result, err := consumeTaskPostRenewal(
		r.Context(),
		tid,
		stringField(body, "task_id"),
		stringField(body, "workspace_id"),
		stringField(body, "user_id"),
		stringField(body, "project_id"),
		stringField(body, "current_expires_at"),
		stringField(body, "idempotency_key"),
	)
	if err != nil {
		if ib, ok := err.(*InsufficientBalanceError); ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":           ib.Error(),
				"balance_points":  ib.BalancePoints,
				"required_points": ib.RequiredPoints,
				"code":            "INSUFFICIENT_TASK_POST_QUOTA",
			})
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}
