package main

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"authz"
	"tracelog"
)

// handleSystemAdminListOrders GET /api/system_admin/orders/
// System admin (superuser) lists all orders across tenants.
// Auth: gateway forward-auth → X-User-Id + X-Gateway-Auth-Verified + X-User-Roles.
// v63: 平台角色（super_admin/employee）判定取代遗留 X-Auth-Superuser/X-Auth-Staff 头。
// OPT-049: Migrated from Django, reactivated 2026-07-30.
func handleSystemAdminListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	doAdminListOrders(w, r)
}

// handleInternalAdminListAllOrders GET /api/internal/taskbill/admin/orders/
// 系统管理员跨租户查看所有订单（需 internal secret，由 Django 侧校验 is_staff）
func handleInternalAdminListAllOrders(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	doAdminListOrders(w, r)
}

func doAdminListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
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
	wechatAccount, werr := normalizeWechatAccountQuery(r.URL.Query().Get("wechat_account"))
	if werr != nil {
		writeErrorJSON(w, http.StatusBadRequest, werr.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if tradeNo != "" && wechatAccount != "" {
		writeErrorJSON(w, http.StatusBadRequest, "order_number and wechat_account are mutually exclusive", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	var filterTenant int64
	if raw := strings.TrimSpace(r.URL.Query().Get("tenant_id")); raw != "" {
		tid, err := parseIDField(raw)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		filterTenant = tid
	}

	var focusOrderID string
	if tradeNo == "" && wechatAccount == "" {
		if raw := strings.TrimSpace(r.URL.Query().Get("order_id")); raw != "" {
			oid, err := parseIDField(raw)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "invalid order_id", tracelog.TraceIDFromContext(r.Context()))
				return
			}
			aligned, found, ferr := adminOrderFocusOffset(oid, statusFilter, limit)
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
		orders, total, err = listOrdersByTradeNo(filterTenant, tradeNo, statusFilter, limit, offset)
	} else if wechatAccount != "" {
		ids, lerr := resolveWechatLinkedUserIDs(r.Context(), wechatAccount)
		if lerr != nil {
			writeErrorJSON(w, http.StatusBadGateway, "wechat account lookup failed", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		orders, total, err = listOrdersByWechatAccount(wechatAccountUserIDsToInt64(ids), wechatAccount, statusFilter, limit, offset)
	} else if filterTenant != 0 {
		orders, total, err = listTenantOrders(filterTenant, limit, offset, statusFilter)
	} else {
		orders, total, err = listAllOrders(limit, offset, statusFilter)
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if tradeNo != "" {
		logOrderTradeNoQuery(r.Context(), "admin_list", 0, total, tradeNo)
	}
	if wechatAccount != "" {
		logOrderWechatAccountQuery(r.Context(), total, utf8.RuneCountInString(wechatAccount))
	}

	orderList := make([]map[string]interface{}, 0, len(orders))
	for _, o := range orders {
		paidAt := ""
		if o.PaidAt.Valid {
			paidAt = o.PaidAt.String
		}
		orderList = append(orderList, map[string]interface{}{
			"id":             formatID(o.ID),
			"tenant_id":      formatID(o.TenantID),
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
