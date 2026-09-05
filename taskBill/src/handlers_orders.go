package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

// ---- 订单 CRUD ----

// handleCreateOrder POST /api/tenant/{id}/billing/orders/
func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireTenantAdmin(w, r, formatID(tid)) {
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	rawItems, ok := body["items"].([]interface{})
	if !ok || len(rawItems) == 0 {
		writeErrorJSON(w, http.StatusBadRequest, "须提供 items（至少一项）", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	var items []orderItemInput
	for _, raw := range rawItems {
		item, ok := raw.(map[string]interface{})
		if !ok {
			writeErrorJSON(w, http.StatusBadRequest, "items 每项须为对象", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		rt := stringField(item, "resource_type")
		qty, err := parseNonNegInt64(item["quantity"], "quantity")
		if err != nil || qty <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, fmt.Sprintf("resource_type=%s 的 quantity 须 >= 1", rt), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		oi := orderItemInput{ResourceType: rt, Quantity: qty, Region: strings.TrimSpace(stringField(item, "region"))}
		if rt == ResourceTypeGitlabDisk || rt == ResourceTypeGitlabTraffic {
			if oi.Region == "" {
				writeErrorJSON(w, http.StatusBadRequest, "region required", tracelog.TraceIDFromContext(r.Context()))
				return
			}
		}
		if rt == ResourceTypeGitlabDisk {
			if dm, e := parseNonNegInt64(item["disk_months"], "disk_months"); e == nil && dm > 0 {
				oi.DiskMonths = dm
			} else {
				oi.DiskMonths = 1
			}
		}
		items = append(items, oi)
	}

	buyerNote := strings.TrimSpace(stringField(body, "buyer_note"))
	order, orderItems, err := createOrderWithNote(contextWithTester(r.Context(), requestIsTester(r)), tid, items, buyerNote, strings.TrimSpace(r.Header.Get("X-User-Id")))
	if err != nil {
		slog.WarnContext(r.Context(), "resource_order_create_failed",
			"level", "warn",
			"tenant_id", formatID(tid),
			"error", err.Error(),
		)
		status := http.StatusBadRequest
		if errors.Is(err, errGitlabRegionDevModeForbidden) {
			status = http.StatusForbidden
		}
		writeErrorJSON(w, status, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	writeJSON(w, http.StatusOK, orderJSON(order, orderItems))
}

// handleListOrders GET /api/tenant/{id}/billing/orders/
// 可选 query：order_id — 将 offset 对齐到包含该订单的页，并在响应中回传 focus_order_id。
// 可选 query：order_number — 等值匹配展示号 / 主键 / payment_ref / out_trade_no / wechat_transaction_id（仅本租户）。
func handleListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 50 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	statusFilter := r.URL.Query().Get("status")

	tradeNo, nerr := normalizeTradeNoQuery(r.URL.Query().Get("order_number"))
	if nerr != nil {
		writeErrorJSON(w, http.StatusBadRequest, nerr.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	var focusOrderID string
	if tradeNo == "" {
		if raw := strings.TrimSpace(r.URL.Query().Get("order_id")); raw != "" {
			oid, err := parseIDField(raw)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "invalid order_id", tracelog.TraceIDFromContext(r.Context()))
				return
			}
			aligned, found, ferr := tenantOrderFocusOffset(tid, oid, statusFilter, limit)
			if ferr != nil {
				writeErrorJSON(w, http.StatusInternalServerError, ferr.Error(), tracelog.TraceIDFromContext(r.Context()))
				return
			}
			if found {
				offset = aligned
				focusOrderID = formatID(oid)
			}
		}
	}

	var orders []*ResourceOrder
	var total int64
	var err error
	if tradeNo != "" {
		orders, total, err = listOrdersByTradeNo(tid, tradeNo, statusFilter, limit, offset)
	} else {
		orders, total, err = listTenantOrders(tid, limit, offset, statusFilter)
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if tradeNo != "" {
		logOrderTradeNoQuery(r.Context(), "tenant_list", tid, total, tradeNo)
	}

	orderList := make([]map[string]interface{}, 0, len(orders))
	for _, o := range orders {
		paidAt := ""
		if o.PaidAt.Valid {
			paidAt = o.PaidAt.String
		}
		orderList = append(orderList, map[string]interface{}{
			"id":             formatID(o.ID),
			"order_number":   o.OrderNumber,
			"status":         o.Status,
			"total_yuan":     centsToYuanStr(o.TotalYuanCents),
			"payment_method": o.PaymentMethod,
			"created_at":     o.CreatedAt,
			"paid_at":        paidAt,
		})
	}

	resp := map[string]interface{}{
		"orders": orderList,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	if focusOrderID != "" {
		resp["focus_order_id"] = focusOrderID
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleGetOrder GET /api/tenant/{id}/billing/orders/{orderId}/
func handleGetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	// OPT-20260815-001: IDOR 防护 — URL 租户与订单归属不一致按 404 处理，避免探测订单存在性。
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusNotFound, "订单不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	order, items, err := loadOrder(tid, orderID)
	if err != nil {
		writeOrderLookupMiss(w, r, tid, orderID)
		return
	}
	if err := ensureTaskPostPurchaseLots(r.Context(), tid); err != nil {
		slog.WarnContext(r.Context(), "task_post_purchase_lots_backfill_failed",
			"level", "warn",
			"tenant_id", formatID(tid),
			"order_id", formatID(orderID),
			"error", err.Error(),
		)
	}

	writeJSON(w, http.StatusOK, orderJSON(order, items))
}

func writeOrderLookupMiss(w http.ResponseWriter, r *http.Request, tenantID, orderID int64) {
	slog.WarnContext(r.Context(), "resource_order_lookup_miss",
		"tenant_id", formatID(tenantID),
		"order_id", formatID(orderID),
	)
	writeErrorJSON(w, http.StatusNotFound, "订单不存在", tracelog.TraceIDFromContext(r.Context()))
}
