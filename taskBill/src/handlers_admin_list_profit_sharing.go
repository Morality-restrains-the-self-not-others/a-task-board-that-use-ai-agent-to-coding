package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"authz"
	"tracelog"
)

// handleSystemAdminProfitSharingRoutes 分发列表 GET 与 /{id}/share/ POST。
// 不可注册 {id}/share 通配：与 refresh-wechat/ 在 net/http ServeMux 冲突。
func handleSystemAdminProfitSharingRoutes(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSuffix(r.URL.Path, "/")
	if strings.HasSuffix(p, "/share") {
		handleSystemAdminShareProfitSharing(w, r)
		return
	}
	handleSystemAdminListProfitSharing(w, r)
}

// handleSystemAdminListProfitSharing GET /api/system-admin/profit-sharing/
// 平台员工读取待分账（及按 status 筛选）队列。含成对 app_id/openid（接收方登记，对齐 wechat_identity）；
// 不含 referrer_openid 键。含商户单号与微信订单号/分账单号（可空）。
// out_profit_sharing_no 仍回传：同一分账记录上的微信支付 out_order_no 幂等键（非独立实体），前端折叠进微信分账单号单元格。
func handleSystemAdminListProfitSharing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		slog.WarnContext(r.Context(), "admin_profit_sharing_list_unauthorized",
			"level", "warn",
			"reason", "missing_gateway_verify",
		)
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		slog.WarnContext(r.Context(), "admin_profit_sharing_list_forbidden",
			"level", "warn",
			"reason", "not_platform_staff",
		)
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
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
	var filter profitSharingQueueFilter
	filter.ReferrerUserID = strings.TrimSpace(r.URL.Query().Get("referrer_user_id"))
	statusRaw := strings.TrimSpace(r.URL.Query().Get("status"))
	if filter.ReferrerUserID != "" && statusRaw == "" {
		statusRaw = "all"
	}
	statuses, err := parseProfitSharingStatusFilter(statusRaw)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	filter.Statuses = statuses
	// OPT-20260823-045：管理端表头列过滤（订单号 LIKE / 租户 ID 等值 / 接收方 user_id 等值）。
	filter.OrderNumber = strings.TrimSpace(r.URL.Query().Get("order_number"))
	if raw := strings.TrimSpace(r.URL.Query().Get("tenant_id")); raw != "" {
		tid, terr := strconv.ParseInt(raw, 10, 64)
		if terr != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		filter.TenantID = tid
	}
	filter.ReceiverUserID = strings.TrimSpace(r.URL.Query().Get("receiver_user_id"))
	// OPT-20260825-023: 商户单号（微信 out_trade_no / payment_ref 去前缀）表头列过滤。
	filter.OutTradeNo = strings.TrimSpace(r.URL.Query().Get("out_trade_no"))

	items, total, err := listProfitSharingQueue(r.Context(), filter, limit, offset)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_profit_sharing_list_query_failed",
			"level", "error",
			"error", err.Error(),
		)
		writeErrorJSON(w, http.StatusInternalServerError, "加载待分账列表失败", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	slog.InfoContext(r.Context(), "admin_profit_sharing_list_ok",
		"level", "info",
		"status_filter", statusRaw,
		"referrer_user_id_len", len(filter.ReferrerUserID),
		"order_number_len", len(filter.OrderNumber),
		"tenant_id", filter.TenantID,
		"receiver_user_id_len", len(filter.ReceiverUserID),
		"out_trade_no_len", len(filter.OutTradeNo),
		"total", total,
		"limit", limit,
		"offset", offset,
		"returned", len(items),
	)
	resp := map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	if filter.ReferrerUserID != "" {
		resp["receiver_registration_status"] = receiverRegistrationStatusForReferrer(filter.ReferrerUserID)
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseProfitSharingStatusFilter(raw string) ([]string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" || s == "open" {
		return []string{psStatusPending, psStatusProcessing, psStatusFailed}, nil
	}
	if s == "all" {
		return nil, nil
	}
	switch s {
	case psStatusPending, psStatusProcessing, psStatusFailed, psStatusFinished:
		return []string{s}, nil
	default:
		return nil, fmt.Errorf("invalid status")
	}
}
