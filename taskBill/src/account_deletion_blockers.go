package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

type billDeletionBlocker struct {
	Code       string `json:"code"`
	Blocking   bool   `json:"blocking"`
	Message    string `json:"message"`
	ActionURL  string `json:"action_url,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	TenantName string `json:"tenant_name,omitempty"`
}

func handleInternalUserAccountDeletionBlockers(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/taskbill/users/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "account-deletion-blockers" {
		writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	userID := strings.TrimSpace(parts[0])
	if userID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "user_id required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	blockers, err := collectBillDeletionBlockers(r, userID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"blockers": blockers})
}

func collectBillDeletionBlockers(r *http.Request, userID string) ([]billDeletionBlocker, error) {
	var blockers []billDeletionBlocker
	tenantIDs, err := listUserTenantIDsForDeletion(userID)
	if err != nil {
		return nil, err
	}
	for _, tid := range tenantIDs {
		tidInt, _ := strconv.ParseInt(tid, 10, 64)
		if tidInt <= 0 {
			continue
		}
		acc, err := getBillingAccount(tidInt)
		if err == nil {
			if acc.Balance > 0 || acc.FrozenBalance > 0 {
				blockers = append(blockers, billDeletionBlocker{
					Code: "BILLING_BALANCE_REMAINING", Blocking: true,
					Message:   fmt.Sprintf("租户账户仍有可用或冻结余额（租户 %s）", tid),
					ActionURL: billingDashboardActionURL(tid),
					TenantID:  tid,
				})
			}
		}
		orders, _, err := listTenantOrders(tidInt, 20, 0, OrderStatusPending)
		if err == nil && len(orders) > 0 {
			blockers = append(blockers, billDeletionBlocker{
				Code: "BILLING_ORDER_PENDING", Blocking: true,
				Message:   fmt.Sprintf("租户 %s 仍有 %d 笔未完成订单", tid, len(orders)),
				ActionURL: billingOrdersActionURL(tid),
				TenantID:  tid,
			})
		}
		pendingRefunds, err := listRefundApplications(r.Context(), refundListFilter{TenantID: tidInt, Status: refundStatusPending})
		if err == nil && len(pendingRefunds) > 0 {
			blockers = append(blockers, billDeletionBlocker{
				Code: "BILLING_REFUND_PENDING", Blocking: true,
				Message:   fmt.Sprintf("租户 %s 有进行中的退款申请", tid),
				ActionURL: billingDashboardActionURL(tid),
				TenantID:  tid,
			})
		}
		if active, err := tenantHasActiveGitlabResource(tidInt); err == nil && active {
			blockers = append(blockers, billDeletionBlocker{
				Code: "BILLING_GITLAB_RESOURCE_ACTIVE", Blocking: true,
				Message:   fmt.Sprintf("租户 %s 仍有有效 GitLab 资源订阅", tid),
				ActionURL: gitlabResourceActionURL(tid),
				TenantID:  tid,
			})
		}
	}
	pendingPayments, err := listPendingPaymentsForUser(r.Context(), userID)
	if err == nil {
		for _, p := range pendingPayments {
			tid := ""
			if p.TenantID > 0 {
				tid = strconv.FormatInt(p.TenantID, 10)
			}
			blockers = append(blockers, billDeletionBlocker{
				Code: "BILLING_PAYMENT_PENDING", Blocking: true,
				Message:   fmt.Sprintf("有待完成的支付（%s）", p.OutTradeNo),
				ActionURL: pendingPaymentActionURL(p.TenantID, p.OrderID),
				TenantID:  tid,
			})
		}
	}
	return blockers, nil
}

func listUserTenantIDsForDeletion(userID string) ([]string, error) {
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	if base == "" {
		return nil, fmt.Errorf("tenant service not configured")
	}
	url := fmt.Sprintf("%s/api/internal/tenant/members?user_id=%s", base, userID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := tenantHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tenant members status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var members []map[string]interface{}
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var ids []string
	for _, m := range members {
		cid, _ := m["company_id"].(string)
		cid = strings.TrimSpace(cid)
		if cid == "" {
			continue
		}
		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}
		ids = append(ids, cid)
	}
	return ids, nil
}

func tenantHasActiveGitlabResource(tenantID int64) (bool, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(1) FROM billing_tenant_gitlab_resource
		WHERE tenant_id = ? AND COALESCE(provisioning_status, 'active') = 'active'`, tenantID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

type pendingPaymentRef struct {
	OutTradeNo string
	TenantID   int64
	OrderID    int64
}

func listPendingPaymentsForUser(ctx context.Context, userID string) ([]pendingPaymentRef, error) {
	// 仅未完成支付可阻断注销。黑名单 NOT IN ('paid','cancelled') 会把
	// status=refunded 以及「支付行仍 pending、关联订单已退款/已付/已取消」误判为待处理。
	rows, err := db.QueryContext(ctx, `
		SELECT p.out_trade_no, COALESCE(p.tenant_id, 0), COALESCE(p.order_id, 0)
		FROM billing_payment_pending p
		LEFT JOIN billing_resource_order o ON p.order_id <> 0 AND o.id = p.order_id
		WHERE p.user_id = ?
		  AND p.status = 'pending'
		  AND (p.order_id = 0 OR o.status = ?)
		LIMIT 20`, userID, OrderStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []pendingPaymentRef
	for rows.Next() {
		var p pendingPaymentRef
		if err := rows.Scan(&p.OutTradeNo, &p.TenantID, &p.OrderID); err == nil && p.OutTradeNo != "" {
			refs = append(refs, p)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "account_deletion_pending_payments_listed",
		"count", len(refs))
	return refs, nil
}
