package main

import (
	"authz"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"tracelog"
)

func emitOrderCommentCreated(r *http.Request, c *OrderComment) {
	publishBillingEvent(r.Context(), "BILLING_ORDER_COMMENT_CREATED", map[string]interface{}{
		"comment_id":     formatID(c.ID),
		"order_id":       formatID(c.OrderID),
		"tenant_id":      formatID(c.TenantID),
		"author_side":    c.AuthorSide,
		"author_user_id": c.AuthorUserID,
		"created_at":     c.CreatedAt,
	})
}

// handleTenantOrderComments GET|POST /api/tenant/{tid}/billing/orders/{orderId}/comments/
func handleTenantOrderComments(w http.ResponseWriter, r *http.Request) {
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tenantKey := formatID(tid)

	switch r.Method {
	case http.MethodGet:
		if !authz.RequirePerm(w, r, authz.PermBillingView, tenantKey) {
			return
		}
		comments, err := listOrderComments(r.Context(), tid, orderID, defaultCommentLimit)
		if err != nil {
			writeOrderCommentErr(w, r, err)
			return
		}
		list := make([]map[string]interface{}, 0, len(comments))
		for i := range comments {
			list = append(list, orderCommentJSON(&comments[i]))
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"comments": list,
			"total":    len(list),
		})
	case http.MethodPost:
		userID := strings.TrimSpace(r.Header.Get(authz.HeaderUserID))
		if userID == "" {
			writeErrorJSON(w, http.StatusUnauthorized, "未认证", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		if !authz.RequirePerm(w, r, authz.PermBillingManage, tenantKey) {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		content, _ := body["content"].(string)
		c, err := createOrderComment(r.Context(), tid, orderID, userID, AuthorSideTenant, content)
		if err != nil {
			writeOrderCommentErr(w, r, err)
			return
		}
		emitOrderCommentCreated(r, c)
		tracelog.LogForwardStage(r.Context(), "billing_order_comment_created", map[string]any{
			"comment_id":  formatID(c.ID),
			"order_id":    formatID(c.OrderID),
			"tenant_id":   formatID(c.TenantID),
			"author_side": c.AuthorSide,
		})
		writeJSON(w, http.StatusCreated, orderCommentJSON(c))
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

// handleSystemAdminOrderComments GET|POST /api/system_admin/orders/{orderId}/comments/
func handleSystemAdminOrderComments(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get(authz.HeaderGatewayVerify)) != "1" {
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	switch r.Method {
	case http.MethodGet:
		tid, err := resolveAdminCommentTenantID(r, orderID)
		if err != nil {
			writeOrderCommentErr(w, r, err)
			return
		}
		comments, err := listOrderComments(r.Context(), tid, orderID, defaultCommentLimit)
		if err != nil {
			writeOrderCommentErr(w, r, err)
			return
		}
		list := make([]map[string]interface{}, 0, len(comments))
		for i := range comments {
			list = append(list, orderCommentJSON(&comments[i]))
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"comments": list,
			"total":    len(list),
		})
	case http.MethodPost:
		userID := strings.TrimSpace(r.Header.Get(authz.HeaderUserID))
		if userID == "" {
			writeErrorJSON(w, http.StatusUnauthorized, "未认证", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		content, _ := body["content"].(string)
		tid := int64FromInterface(body["tenant_id"])
		if tid <= 0 {
			if q := strings.TrimSpace(r.URL.Query().Get("tenant_id")); q != "" {
				tid, _ = strconv.ParseInt(q, 10, 64)
			}
		}
		if tid <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, "缺少 tenant_id", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		c, err := createOrderComment(r.Context(), tid, orderID, userID, AuthorSideSystemAdmin, content)
		if err != nil {
			writeOrderCommentErr(w, r, err)
			return
		}
		emitOrderCommentCreated(r, c)
		tracelog.LogForwardStage(r.Context(), "billing_order_comment_created", map[string]any{
			"comment_id":  formatID(c.ID),
			"order_id":    formatID(c.OrderID),
			"tenant_id":   formatID(c.TenantID),
			"author_side": c.AuthorSide,
		})
		writeJSON(w, http.StatusCreated, orderCommentJSON(c))
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func resolveAdminCommentTenantID(r *http.Request, orderID int64) (int64, error) {
	if q := strings.TrimSpace(r.URL.Query().Get("tenant_id")); q != "" {
		tid, err := strconv.ParseInt(q, 10, 64)
		if err != nil || tid <= 0 {
			return 0, ErrOrderCommentOrder
		}
		return tid, nil
	}
	var tid int64
	err := db.QueryRowContext(r.Context(), `SELECT tenant_id FROM billing_resource_order WHERE id = ?`, orderID).Scan(&tid)
	if err != nil {
		return 0, ErrOrderCommentOrder
	}
	return tid, nil
}

func writeOrderCommentErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrOrderCommentEmpty), errors.Is(err, ErrOrderCommentTooLong), errors.Is(err, ErrOrderCommentSide):
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
	case errors.Is(err, ErrOrderCommentOrder):
		writeErrorJSON(w, http.StatusNotFound, err.Error(), tracelog.TraceIDFromContext(r.Context()))
	default:
		if err != nil && strings.Contains(err.Error(), "缺少作者") {
			writeErrorJSON(w, http.StatusUnauthorized, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
	}
}
