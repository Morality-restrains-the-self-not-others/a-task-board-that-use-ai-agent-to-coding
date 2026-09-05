package main

import (
	"authz"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"tracelog"
)

func requireTenantAdminForRefund(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "未认证", tracelog.TraceIDFromContext(r.Context()))
		return false
	}
	// v63: 判定迁移至 authz 权限码（billing:manage）
	return authz.RequirePerm(w, r, authz.PermBillingManage, tenantID)
}

func handleTenantRefundApplications(w http.ResponseWriter, r *http.Request) {
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		enabled, err := getRefundPolicyEnabled(ctx)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		items, err := listRefundApplications(ctx, refundListFilter{TenantID: tid})
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled": enabled,
			"results": items,
		})
	case http.MethodPost:
		if !requireTenantAdminForRefund(w, r, formatID(tid)) {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		reason, _ := body["reason"].(string)
		userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
		orderID := int64FromInterface(body["order_id"])
		if orderID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "缺少订单 ID",
				"code":  "ORDER_ID_REQUIRED",
			})
			return
		}
		if reason == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "退款原因不能为空",
				"code":  "REASON_REQUIRED",
			})
			return
		}
		app, err := applyRefundApplication(ctx, tid, userID, reason, orderID, strings.TrimSpace(r.Header.Get("Idempotency-Key")))
		if err != nil {
			if errors.Is(err, ErrRefundDisabled) {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"error": "平台未开启退款申请",
					"code":  "REFUND_DISABLED",
				})
				return
			}
			if errors.Is(err, ErrPendingRefundApplication) {
				writeJSON(w, http.StatusConflict, map[string]string{
					"error": "该租户已有待审批的退款申请",
					"code":  "PENDING_REFUND_EXISTS",
				})
				return
			}
			if errors.Is(err, ErrOrderAlreadyRefunded) {
				writeJSON(w, http.StatusConflict, map[string]string{
					"error": err.Error(),
					"code":  "ORDER_REFUND_EXISTS",
				})
				return
			}
			if errors.Is(err, ErrNoRefundableAmount) || errors.Is(err, ErrOrderNotRefundable) {
				writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
				return
			}
			if strings.Contains(err.Error(), "缺少订单 ID") || strings.Contains(err.Error(), "查询关联订单失败") {
				writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
				return
			}
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusCreated, refundApplicationJSON(app))
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

// isRefundApplicationListPath 判断是否为退款申请列表路径（兼容 internal 和 system-admin 路由前缀）
func isRefundApplicationListPath(path string) bool {
	for _, p := range []string{
		"/api/internal/taskbill/refund-applications",
		"/api/system-admin/refund-applications",
	} {
		if path == p || path == p+"/" {
			return true
		}
	}
	return false
}

func parseRefundApplicationID(path string) (int64, bool) {
	prefixes := []string{
		"/api/internal/taskbill/refund-applications/",
		"/api/system-admin/refund-applications/",
	}
	var rest string
	for _, prefix := range prefixes {
		if after, ok := strings.CutPrefix(path, prefix); ok {
			rest = strings.Trim(after, "/")
			break
		}
	}
	if rest == "" {
		return 0, false
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	action := strings.TrimSuffix(parts[1], "/")
	if action != "approve" && action != "reject" {
		return 0, false
	}
	return id, true
}

func parseRefundApplicationAction(path string) string {
	prefixes := []string{
		"/api/internal/taskbill/refund-applications/",
		"/api/system-admin/refund-applications/",
	}
	var rest string
	for _, prefix := range prefixes {
		if after, ok := strings.CutPrefix(path, prefix); ok {
			rest = strings.Trim(after, "/")
			break
		}
	}
	if rest == "" {
		return ""
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSuffix(parts[1], "/")
}

func handleInternalRefundPolicy(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		enabled, err := getRefundPolicyEnabled(ctx)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, refundPolicyJSON(enabled))
	case http.MethodPut, http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		enabledRaw, ok := body["enabled"]
		if !ok {
			writeErrorJSON(w, http.StatusBadRequest, "enabled is required", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		enabled := false
		switch v := enabledRaw.(type) {
		case bool:
			enabled = v
		case float64:
			enabled = v != 0
		case string:
			enabled = strings.EqualFold(v, "true") || v == "1"
		default:
			writeErrorJSON(w, http.StatusBadRequest, "enabled must be boolean", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		actor := strings.TrimSpace(stringField(body, "actor_user_id"))
		if actor == "" {
			actor = strings.TrimSpace(r.Header.Get("X-User-Id"))
		}
		if err := setRefundPolicyEnabled(ctx, enabled, actor); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, refundPolicyJSON(enabled))
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

// handleSystemAdminRefundPolicy 处理系统管理员对退款策略的读写（网关鉴权）。
func handleSystemAdminRefundPolicy(w http.ResponseWriter, r *http.Request) {
	if !requireInternalOrGatewayAuth(w, r) {
		return
	}
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		enabled, err := getRefundPolicyEnabled(ctx)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, refundPolicyJSON(enabled))
	case http.MethodPut, http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		enabledRaw, ok := body["enabled"]
		if !ok {
			writeErrorJSON(w, http.StatusBadRequest, "enabled is required", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		enabled := false
		switch v := enabledRaw.(type) {
		case bool:
			enabled = v
		case float64:
			enabled = v != 0
		case string:
			enabled = strings.EqualFold(v, "true") || v == "1"
		default:
			writeErrorJSON(w, http.StatusBadRequest, "enabled must be boolean", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		actor := strings.TrimSpace(r.Header.Get("X-User-Id"))
		if err := setRefundPolicyEnabled(ctx, enabled, actor); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, refundPolicyJSON(enabled))
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func handleInternalRefundApplicationsRouter(w http.ResponseWriter, r *http.Request) {
	if !requireInternalOrGatewayAuth(w, r) {
		return
	}
	path := r.URL.Path
	if isRefundApplicationListPath(path) {
		if r.Method != http.MethodGet {
			writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		f := refundListFilter{Status: strings.TrimSpace(r.URL.Query().Get("status"))}
		// OPT-20260823-045：管理端表头列过滤 — tenant_id / order_id 等值查询。
		if v := strings.TrimSpace(r.URL.Query().Get("tenant_id")); v != "" {
			f.TenantID, _ = strconv.ParseInt(v, 10, 64)
		}
		if v := strings.TrimSpace(r.URL.Query().Get("order_id")); v != "" {
			f.OrderID, _ = strconv.ParseInt(v, 10, 64)
		}
		items, err := listRefundApplications(r.Context(), f)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": items})
		return
	}

	appID, ok := parseRefundApplicationID(path)
	if !ok {
		writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	reviewerUserID := strings.TrimSpace(stringField(body, "reviewer_user_id"))
	if reviewerUserID == "" {
		reviewerUserID = strings.TrimSpace(r.Header.Get("X-User-Id"))
	}
	note, _ := body["note"].(string)
	ctx := r.Context()
	action := parseRefundApplicationAction(path)
	switch action {
	case "approve":
		app, err := approveRefundApplication(ctx, appID, reviewerUserID, note)
		if err != nil {
			status, msg := refundActionClientError(err)
			writeErrorJSON(w, status, msg, tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, refundApplicationJSON(app))
	case "reject":
		app, err := rejectRefundApplication(ctx, appID, reviewerUserID, note)
		if err != nil {
			status, msg := refundActionClientError(err)
			writeErrorJSON(w, status, msg, tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, refundApplicationJSON(app))
	default:
		writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
	}
}
