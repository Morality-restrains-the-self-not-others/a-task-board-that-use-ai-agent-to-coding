package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"authz"
	"tracelog"
)

// handleSystemAdminParseOrderNumber POST /api/system-admin/order-number/parse/
// 管理端「粘贴订单号」：解析 ADR-0017/0018 展示订单号，返回租户基因供前端跳转。
// 四段号 ORD-日期-tenant-id 含租户；三段号无租户基因（存量），需手工选租户。
// 鉴权由网关完成（X-Gateway-Auth-Verified + 平台角色），禁止新增匿名公网查单接口。
func handleSystemAdminParseOrderNumber(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	var body struct {
		OrderNumber string `json:"order_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	parsed, err := ParseResourceOrderNumber(body.OrderNumber)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"order_number": strings.TrimSpace(body.OrderNumber),
		"has_tenant":   parsed.HasTenant,
		"tenant_id":    formatID(parsed.TenantID),
		"order_id":     formatID(parsed.OrderID),
		"date_utc":     parsed.DateUTC,
	})
}
