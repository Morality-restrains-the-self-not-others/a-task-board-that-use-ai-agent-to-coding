package main

// 个人信息导出内部端点（PIPL「导出权」）：taskAuth 聚合导出时调用。
// 契约：GET /api/internal/taskbill/users/{user_id}/personal-data/ → {"data": {...}}
// 数据边界：仅可直接归因到 user_id 的账单数据；不含支付凭据/退款单号等敏感凭证。
// 安全边界：referrer_user_id 是引荐人标识，同样导出（被引荐人可导出自己的引荐关系）。

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

// collectBillPersonalData 汇总该用户在 taskBill 的全部个人数据 section。
func collectBillPersonalData(ctx context.Context, userID string) (map[string]interface{}, error) {
	uid, _ := strconv.ParseInt(userID, 10, 64)
	data := map[string]interface{}{}

	// orders: 资源订单（user_id 为下单会话用户，BIGINT）
	orders := make([]map[string]interface{}, 0)
	orderRows, err := db.QueryContext(ctx, `
		SELECT id, order_number, tenant_id, status, total_yuan_cents,
		       COALESCE(payment_method,''), COALESCE(paid_at,''), COALESCE(cancelled_at,''), created_at
		FROM billing_resource_order WHERE user_id = ? ORDER BY created_at DESC LIMIT 200`, uid)
	if err != nil {
		return nil, err
	}
	defer orderRows.Close()
	for orderRows.Next() {
		var id int64
		var orderNumber, status, paymentMethod, paidAt, cancelledAt, createdAt string
		var tenantID int64
		var totalCents int64
		if err := orderRows.Scan(&id, &orderNumber, &tenantID, &status, &totalCents,
			&paymentMethod, &paidAt, &cancelledAt, &createdAt); err != nil {
			continue
		}
		item := map[string]interface{}{
			"order_id":          strconv.FormatInt(id, 10),
			"order_number":      orderNumber,
			"tenant_id":         strconv.FormatInt(tenantID, 10),
			"status":            status,
			"total_yuan_cents":  totalCents,
			"payment_method":    paymentMethod,
			"created_at":        createdAt,
		}
		if paidAt != "" {
			item["paid_at"] = paidAt
		}
		if cancelledAt != "" {
			item["cancelled_at"] = cancelledAt
		}
		orders = append(orders, item)
	}
	data["orders"] = orders

	// invoices: 与本人订单关联的发票（开票信息 buyer_snapshot 含联系方式，不导出）
	invoices := make([]map[string]interface{}, 0)
	invRows, err := db.QueryContext(ctx, `
		SELECT i.fapiao_id, i.kind, i.purpose, i.status, i.amount_yuan_cents, i.created_at
		FROM billing_invoice i
		INNER JOIN billing_resource_order o ON o.id = i.order_id
		WHERE o.user_id = ? ORDER BY i.created_at DESC LIMIT 200`, uid)
	if err != nil {
		return nil, err
	}
	defer invRows.Close()
	for invRows.Next() {
		var fapiaoID, kind, purpose, status, createdAt string
		var amountCents int64
		if err := invRows.Scan(&fapiaoID, &kind, &purpose, &status, &amountCents, &createdAt); err != nil {
			continue
		}
		invoices = append(invoices, map[string]interface{}{
			"fapiao_id":        fapiaoID,
			"kind":             kind,
			"purpose":          purpose,
			"status":           status,
			"amount_yuan_cents": amountCents,
			"created_at":       createdAt,
		})
	}
	data["invoices"] = invoices

	// refund_applications: 本人提交的退款申请
	refunds := make([]map[string]interface{}, 0)
	refundRows, err := db.QueryContext(ctx, `
		SELECT tenant_id, frozen_points, status, COALESCE(reason,''), created_at, COALESCE(reviewed_at,'')
		FROM billing_refund_application WHERE applicant_user_id = ?
		ORDER BY created_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer refundRows.Close()
	for refundRows.Next() {
		var tenantID int64
		var frozenPoints int64
		var status, reason, createdAt, reviewedAt string
		if err := refundRows.Scan(&tenantID, &frozenPoints, &status, &reason, &createdAt, &reviewedAt); err != nil {
			continue
		}
		item := map[string]interface{}{
			"tenant_id":     strconv.FormatInt(tenantID, 10),
			"frozen_points": frozenPoints,
			"status":        status,
			"reason":        reason,
			"created_at":    createdAt,
		}
		if reviewedAt != "" {
			item["reviewed_at"] = reviewedAt
		}
		refunds = append(refunds, item)
	}
	data["refund_applications"] = refunds

	// pending_payments: 支付进行中记录
	pending := make([]map[string]interface{}, 0)
	pendingRows, err := db.QueryContext(ctx, `
		SELECT out_trade_no, tenant_id, order_id, amount_fen, status, created_at, COALESCE(paid_at,'')
		FROM billing_payment_pending WHERE user_id = ? ORDER BY created_at DESC LIMIT 200`, uid)
	if err != nil {
		return nil, err
	}
	defer pendingRows.Close()
	for pendingRows.Next() {
		var outTradeNo, status, createdAt, paidAt string
		var tenantID, orderID, amountFen int64
		if err := pendingRows.Scan(&outTradeNo, &tenantID, &orderID, &amountFen, &status, &createdAt, &paidAt); err != nil {
			continue
		}
		item := map[string]interface{}{
			"out_trade_no": outTradeNo,
			"tenant_id":    strconv.FormatInt(tenantID, 10),
			"order_id":     strconv.FormatInt(orderID, 10),
			"amount_fen":   amountFen,
			"status":       status,
			"created_at":   createdAt,
		}
		if paidAt != "" {
			item["paid_at"] = paidAt
		}
		pending = append(pending, item)
	}
	data["pending_payments"] = pending

	// consents: 协议/隐私政策签署记录（含版本与场景）
	licenseConsents := make([]map[string]interface{}, 0)
	licRows, err := db.QueryContext(ctx, `
		SELECT la.title, la.version, c.consented_at, COALESCE(c.context,'')
		FROM billing_user_license_agreement_consents c
		INNER JOIN billing_license_agreements la ON la.id = c.license_agreement_id
		WHERE c.user_id = ? ORDER BY c.consented_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer licRows.Close()
	for licRows.Next() {
		var title, version, consentedAt, contextStr string
		if err := licRows.Scan(&title, &version, &consentedAt, &contextStr); err != nil {
			continue
		}
		licenseConsents = append(licenseConsents, map[string]interface{}{
			"title":        title,
			"version":      version,
			"consented_at": consentedAt,
			"context":      contextStr,
		})
	}
	data["license_agreement_consents"] = licenseConsents

	privacyConsents := make([]map[string]interface{}, 0)
	priRows, err := db.QueryContext(ctx, `
		SELECT pp.title, pp.version, c.consented_at, COALESCE(c.context,'')
		FROM billing_user_privacy_policy_consents c
		INNER JOIN billing_privacy_policies pp ON pp.id = c.privacy_policy_id
		WHERE c.user_id = ? ORDER BY c.consented_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer priRows.Close()
	for priRows.Next() {
		var title, version, consentedAt, contextStr string
		if err := priRows.Scan(&title, &version, &consentedAt, &contextStr); err != nil {
			continue
		}
		privacyConsents = append(privacyConsents, map[string]interface{}{
			"title":        title,
			"version":      version,
			"consented_at": consentedAt,
			"context":      contextStr,
		})
	}
	data["privacy_policy_consents"] = privacyConsents

	// transactions: 旧积分台账中的本人口径流水（user_id 为 varchar）
	transactions := make([]map[string]interface{}, 0)
	txRows, err := db.QueryContext(ctx, `
		SELECT transaction_type, amount, balance_after, COALESCE(description,''), created_at
		FROM billing_transaction WHERE user_id = ? ORDER BY created_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer txRows.Close()
	for txRows.Next() {
		var txType, description, createdAt string
		var amount, balanceAfter int64
		if err := txRows.Scan(&txType, &amount, &balanceAfter, &description, &createdAt); err != nil {
			continue
		}
		transactions = append(transactions, map[string]interface{}{
			"transaction_type": txType,
			"amount":           amount,
			"balance_after":    balanceAfter,
			"description":      description,
			"created_at":       createdAt,
		})
	}
	data["transactions"] = transactions

	// referral_edge: 引荐关系（作为被引荐人或引荐人）
	referral := make([]map[string]interface{}, 0)
	refRows, err := db.QueryContext(ctx, `
		SELECT referred_user_id, referrer_user_id, referrer_tenant_id, COALESCE(bound_at,''), created_at
		FROM billing_referral_edge
		WHERE referred_user_id = ? OR referrer_user_id = ?
		ORDER BY created_at DESC LIMIT 200`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer refRows.Close()
	for refRows.Next() {
		var referredID, referrerID, boundAt, createdAt string
		var referrerTenantID int64
		if err := refRows.Scan(&referredID, &referrerID, &referrerTenantID, &boundAt, &createdAt); err != nil {
			continue
		}
		item := map[string]interface{}{
			"referred_user_id":    referredID,
			"referrer_user_id":    referrerID,
			"referrer_tenant_id":  strconv.FormatInt(referrerTenantID, 10),
			"created_at":          createdAt,
		}
		if boundAt != "" {
			item["bound_at"] = boundAt
		}
		referral = append(referral, item)
	}
	data["referral_edge"] = referral

	return data, nil
}

// handleInternalUserPersonalData 处理内部 personal-data 请求（注册于 handleInternalTaskBillUsersRouter）。
func handleInternalUserPersonalData(w http.ResponseWriter, r *http.Request) {
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
	if len(parts) != 2 || parts[1] != "personal-data" {
		writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	userID := strings.TrimSpace(parts[0])
	if userID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "user_id required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	data, err := collectBillPersonalData(r.Context(), userID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}
