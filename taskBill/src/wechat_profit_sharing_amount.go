package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"tracelog"
)

// WeChat 分账金额必须是整数分，且不得超过
// floor((订单金额 − 退款 − 补差回退) × 最大分账比例)。
// 台账用 ROUND_HALF_UP 存 5%（55 分 → 3 分）会超过该上限（向下取整为 2 分），
// 微信返回 INVALID_REQUEST「分账金额超出最大分账比例，最大可分账金额需等比例扣除退款…」。
// 见 https://pay.weixin.qq.com/doc/v3/merchant/4014547102.md
// 与 https://pay.weixin.qq.com/doc/v3/merchant/4012524936.md receivers.amount。

var errProfitSharingAmountZero = errors.New("profit_sharing_amount_zero")

const profitSharingAmountZeroPublic = "可分账金额不足 1 分（已按订单净额与分账比例向下取整）。订单可能已退款或金额过小。"

// wechatShareAmountFen 出站分账金额：min(台账佣金, floor(净额×比例/100))。
// totalYuanCents<=0 表示调用方未带订单金额，不按净额封顶（避免把合法台账打成 0）。
func wechatShareAmountFen(totalYuanCents, refundedYuanCents, storedCommission, ratePercent int64) int64 {
	if storedCommission < 0 {
		storedCommission = 0
	}
	if totalYuanCents <= 0 {
		return storedCommission
	}
	net := totalYuanCents - refundedYuanCents
	if net < 0 {
		net = 0
	}
	if ratePercent < 1 {
		ratePercent = referralCommissionRateNum
	}
	maxShare := net * ratePercent / 100
	if storedCommission < maxShare {
		return storedCommission
	}
	return maxShare
}

// minShareAmount 再与微信「剩余待分金额」取小；unsplit<0 表示未查询，跳过。
func minShareAmount(amount, unsplit int64) int64 {
	if unsplit < 0 {
		return amount
	}
	if unsplit < amount {
		return unsplit
	}
	return amount
}

func shareRatePercent(ctx context.Context) int64 {
	rate := getReferralRatePercent()
	if rate < 1 {
		rate = referralCommissionRateNum
	}
	info := resolveCommissionRate(ctx)
	if info.ok() && info.Percent > 0 && info.Percent < rate {
		return info.Percent
	}
	return rate
}

func orderRefundedYuanCents(orderID int64) (int64, error) {
	if db == nil || orderID <= 0 {
		return 0, nil
	}
	var approved int64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(frozen_points), 0) FROM billing_refund_application
		WHERE order_id = ? AND status = ?`, orderID, refundStatusApproved).Scan(&approved)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("sum approved refunds: %w", err)
	}
	var ledger int64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(points - remaining_points), 0) FROM billing_payment_ledger
		WHERE channel = 'wechat'
		  AND provider_ref = (
			SELECT REPLACE(COALESCE(payment_ref, ''), 'wechat:', '') FROM billing_resource_order WHERE id = ?
		  )`, orderID).Scan(&ledger)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("sum ledger refunds: %w", err)
	}
	if ledger > approved {
		return ledger, nil
	}
	return approved, nil
}

func fillProfitSharingAmountsFromDB(r *profitSharingRecord) {
	if r == nil || db == nil || r.ID <= 0 {
		return
	}
	if r.TotalYuanCents > 0 && r.CommissionYuanCents > 0 {
		return
	}
	var total, comm int64
	if err := db.QueryRow(`
		SELECT total_yuan_cents, commission_yuan_cents FROM billing_profit_sharing WHERE id = ?`, r.ID).
		Scan(&total, &comm); err != nil {
		return
	}
	if r.TotalYuanCents <= 0 {
		r.TotalYuanCents = total
	}
	if r.CommissionYuanCents <= 0 {
		r.CommissionYuanCents = comm
	}
}

var profitSharingQueryOrderAmountCall = func(ctx context.Context, svc *profitsharing.TransactionsApiService, req profitsharing.QueryOrderAmountRequest) (*profitsharing.QueryOrderAmountResponse, *core.APIResult, error) {
	return svc.QueryOrderAmount(ctx, req)
}

// queryProfitSharingUnsplitAmount 查询剩余待分金额。ok=false 表示未查询（非 live / 无 client）。
func queryProfitSharingUnsplitAmount(ctx context.Context, wechatTransactionID string) (amount int64, ok bool, err error) {
	if !wechatLiveOK || wechatClient == nil {
		return -1, false, nil
	}
	txn := strings.TrimSpace(wechatTransactionID)
	if txn == "" {
		return -1, false, nil
	}
	req := profitsharing.QueryOrderAmountRequest{TransactionId: core.String(txn)}
	svc := profitsharing.TransactionsApiService{Client: wechatClient}
	resp, _, err := profitSharingQueryOrderAmountCall(ctx, &svc, req)
	if err != nil {
		return 0, false, wechatSDKResultError("query profit sharing unsplit amount", err)
	}
	if resp == nil || resp.UnsplitAmount == nil {
		return 0, false, fmt.Errorf("query profit sharing unsplit amount: empty unsplit_amount")
	}
	return *resp.UnsplitAmount, true, nil
}

func resolveOutboundShareAmount(ctx context.Context, r profitSharingRecord, wechatTransactionID string) (int64, error) {
	fillProfitSharingAmountsFromDB(&r)
	refunded, err := orderRefundedYuanCents(r.OrderID)
	if err != nil {
		return 0, err
	}
	rate := shareRatePercent(ctx)
	amount := wechatShareAmountFen(r.TotalYuanCents, refunded, r.CommissionYuanCents, rate)
	unsplit, queried, qerr := queryProfitSharingUnsplitAmount(ctx, wechatTransactionID)
	if qerr != nil {
		slog.WarnContext(ctx, "profit_sharing_unsplit_query_failed",
			"level", "warn",
			"profit_sharing_id", r.ID,
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"error", qerr.Error(),
		)
	} else if queried {
		amount = minShareAmount(amount, unsplit)
	}
	if amount != r.CommissionYuanCents {
		slog.InfoContext(ctx, "profit_sharing_amount_capped",
			"level", "info",
			"profit_sharing_id", r.ID,
			"stored_commission_fen", r.CommissionYuanCents,
			"outbound_fen", amount,
			"refunded_fen", refunded,
			"rate_percent", rate,
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
	}
	if amount < 1 {
		return 0, errProfitSharingAmountZero
	}
	return amount, nil
}

func extractWechatProfitSharingReject(err error) (code, message string, httpStatus int, ok bool) {
	if err == nil {
		return "", "", 0, false
	}
	var apiErr *core.APIError
	if errors.As(err, &apiErr) && apiErr != nil {
		return strings.TrimSpace(apiErr.Code), strings.TrimSpace(apiErr.Message), apiErr.StatusCode, true
	}
	raw := err.Error()
	idx := strings.Index(raw, "{")
	if idx < 0 {
		return "", "", 0, false
	}
	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(raw[idx:]), &parsed) != nil {
		return "", "", 0, false
	}
	code = strings.TrimSpace(parsed.Code)
	message = strings.TrimSpace(parsed.Message)
	if code == "" && message == "" {
		return "", "", 0, false
	}
	httpStatus = http.StatusBadRequest
	if strings.Contains(raw, "HTTP 403") {
		httpStatus = http.StatusForbidden
	}
	return code, message, httpStatus, true
}

func isWechatProfitSharingBusinessReject(code string) bool {
	switch strings.TrimSpace(code) {
	case "INVALID_REQUEST", "RULE_LIMIT", "NOT_ENOUGH":
		return true
	default:
		return false
	}
}
