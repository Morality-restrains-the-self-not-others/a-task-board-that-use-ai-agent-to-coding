package main

import (
	"log/slog"
	"net/http"
	"strings"

	"authz"
	"tracelog"
)

// handleSystemAdminGetOrder GET /api/system-admin/orders/{order_id}/
// 平台员工读取订单详情（含分账快照）。租户 GET 不得调用本函数。
func handleSystemAdminGetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		slog.WarnContext(r.Context(), "admin_order_detail_unauthorized",
			"level", "warn",
			"reason", "missing_gateway_verify",
		)
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		slog.WarnContext(r.Context(), "admin_order_detail_forbidden",
			"level", "warn",
			"reason", "not_platform_staff",
		)
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	order, items, err := loadOrderByID(orderID)
	if err != nil {
		slog.WarnContext(r.Context(), "admin_order_detail_miss",
			"level", "warn",
			"order_id", formatID(orderID),
		)
		writeErrorJSON(w, http.StatusNotFound, "订单不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err := ensureTaskPostPurchaseLots(r.Context(), order.TenantID); err != nil {
		slog.WarnContext(r.Context(), "task_post_purchase_lots_backfill_failed",
			"level", "warn",
			"tenant_id", formatID(order.TenantID),
			"order_id", formatID(orderID),
			"error", err.Error(),
		)
	}
	body, err := adminOrderJSON(r.Context(), order, items)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_order_detail_profit_sharing_query_failed",
			"level", "error",
			"order_id", formatID(orderID),
			"error", err.Error(),
		)
		writeErrorJSON(w, http.StatusInternalServerError, "加载分账记录失败", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	shareCount := 0
	if raw, ok := body["profit_sharing"].([]map[string]interface{}); ok {
		shareCount = len(raw)
	}
	slog.InfoContext(r.Context(), "admin_order_detail_ok",
		"level", "info",
		"order_id", formatID(orderID),
		"tenant_id", formatID(order.TenantID),
		"share_count", shareCount,
	)
	writeJSON(w, http.StatusOK, body)
}
