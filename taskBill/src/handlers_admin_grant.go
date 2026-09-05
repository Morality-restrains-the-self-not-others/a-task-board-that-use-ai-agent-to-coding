package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"tracelog"
)

// adminGrantIdempotencyKey 优先取 body idempotency_key，其次取 Idempotency-Key header。
// 前端 createClickGuard 双通道发送同值，兼容老客户端只发 body。
func adminGrantIdempotencyKey(r *http.Request, body map[string]interface{}) string {
	if ik := strings.TrimSpace(stringField(body, "idempotency_key")); ik != "" {
		return ik
	}
	return strings.TrimSpace(r.Header.Get("Idempotency-Key"))
}

// handleAdminGrantResources 管理员前台赠送资源（经 BillingProxyView 转发）。
// 路径: /api/tenant/{tid}/billing/accounts/admin_grant_points/
func handleAdminGrantResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// 兼容旧格式：{points: N, description: "..."} 自动转为单条 task_post grant
	if grantsRaw, ok := body["resources"]; ok {
		grantsList, ok := grantsRaw.([]interface{})
		if !ok {
			writeErrorJSON(w, http.StatusBadRequest, "resources 须为数组", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		tid, ok := parseTenantID(r.URL.Path)
		if !ok {
			writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		grants := make([]ResourceGrantInput, 0, len(grantsList))
		for i, raw := range grantsList {
			m, ok := raw.(map[string]interface{})
			if !ok {
				writeErrorJSON(w, http.StatusBadRequest, "resources 每项须为对象", tracelog.TraceIDFromContext(r.Context()))
				return
			}
			rt := stringField(m, "resource_type")
			if rt == "" {
				rt = ResourceTypeTaskPost
			}
			qty, err := parseInt64Field(m["quantity"])
			if err != nil || qty <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]interface{}{
					"error": "quantity 须为正整数", "index": i,
				})
				return
			}
			grants = append(grants, ResourceGrantInput{
				ResourceType: rt,
				Quantity:     qty,
				ExpiresAt:    stringField(m, "expires_at"),
				Reason:       stringField(m, "reason"),
				Region:       stringField(m, "region"),
			})
		}
		userID := r.Header.Get("X-User-Id")
		result, err := adminGrantResourcesWithMembership(
			r.Context(), tid, grants, userID, adminGrantIdempotencyKey(r, body),
			strings.TrimSpace(stringField(body, "membership_tier")),
			stringField(body, "membership_reason"),
		)
		if err != nil {
			slog.WarnContext(r.Context(), "admin_grant_resources_failed",
				"level", "warn",
				"tenant_id", formatID(tid),
				"error", err.Error(),
			)
			writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}

	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// 仅改 VIP：无 resources 键
	if tier := strings.TrimSpace(stringField(body, "membership_tier")); tier != "" {
		userID := r.Header.Get("X-User-Id")
		result, err := adminGrantResourcesWithMembership(
			r.Context(), tid, nil, userID, adminGrantIdempotencyKey(r, body),
			tier, stringField(body, "membership_reason"),
		)
		if err != nil {
			slog.WarnContext(r.Context(), "admin_grant_resources_failed",
				"level", "warn",
				"tenant_id", formatID(tid),
				"error", err.Error(),
			)
			writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}

	// 旧格式兼容：{points: N, description: "..."} → 单 task_post，无过期
	points, err := parseInt64Field(body["points"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid points", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	desc := stringField(body, "description")
	if desc == "" {
		desc = "后台赠送资源"
	}
	grants := []ResourceGrantInput{{
		ResourceType: ResourceTypeTaskPost,
		Quantity:     points,
		ExpiresAt:    "",
		Reason:       desc,
	}}
	userID := r.Header.Get("X-User-Id")
	result, err := adminGrantResources(r.Context(), tid, grants, userID, adminGrantIdempotencyKey(r, body))
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	// 兼容旧前端响应字段
	result["granted_points"] = points
	result["points"] = result["task_post_quota_after"]
	writeJSON(w, http.StatusOK, result)
}

// handleInternalBackfillGrantOrders 补建已有 grants 的订单记录（幂等）。
// POST /api/internal/taskbill/backfill-grant-orders/
func handleInternalBackfillGrantOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	created, err := backfillGrantOrders()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"orders_created": created,
		"message":        fmt.Sprintf("补建了 %d 个赠送订单", created),
	})
}

// handleInternalAdminGrantResources 内部 API（Django 事件处理器 / 内部调用）。
// POST /api/internal/taskbill/admin-grant-resources/
func handleInternalAdminGrantResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	grantsRaw, ok := body["grants"]
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "missing grants", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	grantsList, ok := grantsRaw.([]interface{})
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "grants 须为数组", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	grants := make([]ResourceGrantInput, 0, len(grantsList))
	for _, raw := range grantsList {
		m, ok := raw.(map[string]interface{})
		if !ok {
			writeErrorJSON(w, http.StatusBadRequest, "grants 每项须为对象", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		rt := stringField(m, "resource_type")
		if rt == "" {
			rt = ResourceTypeTaskPost
		}
		qty, err := parseInt64Field(m["quantity"])
		if err != nil || qty <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, "quantity 须为正整数", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		grants = append(grants, ResourceGrantInput{
			ResourceType: rt,
			Quantity:     qty,
			ExpiresAt:    stringField(m, "expires_at"),
			Reason:       stringField(m, "reason"),
			Region:       stringField(m, "region"),
		})
	}

	ik := adminGrantIdempotencyKey(r, body)
	result, err := adminGrantResources(r.Context(), tid, grants, stringField(body, "user_id"), ik)
	if err != nil {
		slog.WarnContext(r.Context(), "admin_grant_resources_failed",
			"level", "warn",
			"tenant_id", formatID(tid),
			"error", err.Error(),
		)
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}
