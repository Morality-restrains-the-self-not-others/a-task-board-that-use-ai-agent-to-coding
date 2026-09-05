package main

// 推荐人本人待微信确认分账提示（OPT-20260823-043）。
//
//	GET /api/billing/profit-sharing/referrer-pending/ — 推荐人本人「待确认 N 笔，最晚 YYYY-MM-DD」
//
// 微信向个人分账时下发服务通知，接收方须在有效期内点击确认；逾期该笔分账关闭无法补分。
// 本端点查询推荐人近期已创建分账单（processing/finished）的微信 receivers.result，统计仍
// 为 PENDING（待确认）的笔数并给出最晚确认截止（支付成功 + profitSharingExpireDays 天）。
//
// 约束：
//   - 仅推荐人本人（referrer_user_id == X-User-Id），他人视为空
//   - 微信查询频率受限 / 单笔查询失败时跳过该笔（不阻塞整体），避免页面依赖微信可用性
//   - 仅统计 paid_at 落在近 profitSharingExpireDays+5 天的记录，过期分账单不再提示
//   - 禁止返回 openid / 微信分账单号

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"tracelog"
)

// queryProfitSharingPendingReceiverCount 查询分账单中仍为 PENDING（待接收方确认）的接收方数。
func queryProfitSharingPendingReceiverCount(ctx context.Context, outOrderNo, wechatTransactionID string) (int, error) {
	if wechatClient == nil {
		return 0, fmt.Errorf("wechat pay client not initialized")
	}
	req := profitsharing.QueryOrderRequest{
		TransactionId: core.String(wechatTransactionID),
		OutOrderNo:    core.String(outOrderNo),
	}
	svc := profitsharing.OrdersApiService{Client: wechatClient}
	resp, _, err := profitSharingQueryOrderCall(ctx, &svc, req)
	if err != nil {
		return 0, wechatSDKResultError("query profit sharing", err)
	}
	if resp == nil {
		return 0, fmt.Errorf("query profit sharing failed: empty response")
	}
	pending := 0
	for _, r := range resp.Receivers {
		if r.Result != nil && *r.Result == profitsharing.DETAILSTATUS_PENDING {
			pending++
		}
	}
	return pending, nil
}

// handleReferrerProfitSharingPending 返回推荐人本人待微信确认的分账笔数与最晚截止日。
func handleReferrerProfitSharingPending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "missing user identity", tracelog.TraceIDFromContext(ctx))
		return
	}
	if !wechatLiveOK {
		writeJSON(w, http.StatusOK, map[string]interface{}{"pending_count": 0})
		return
	}
	cutoff := time.Now().UTC().Add(-time.Duration(profitSharingExpireDays+5) * 24 * time.Hour)
	rows, err := db.Query(`
		SELECT ps.id, COALESCE(ps.out_profit_sharing_no, ''), o.paid_at
		FROM billing_profit_sharing ps
		INNER JOIN billing_resource_order o ON o.id = ps.order_id
		WHERE ps.referrer_user_id = ?
		  AND ps.status IN (?, ?)
		  AND o.paid_at >= ?
		ORDER BY ps.created_at DESC
		LIMIT 50`, userID, psStatusProcessing, psStatusFinished, cutoff.Format(time.RFC3339))
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(ctx))
		return
	}
	defer rows.Close()

	pendingCount := 0
	var minDeadline time.Time
	for rows.Next() {
		var psID int64
		var outNo, paidAt string
		if err := rows.Scan(&psID, &outNo, &paidAt); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(ctx))
			return
		}
		outNo = strings.TrimSpace(outNo)
		if outNo == "" {
			continue
		}
		var orderID int64
		if err := db.QueryRow(`SELECT order_id FROM billing_profit_sharing WHERE id=?`, psID).Scan(&orderID); err != nil {
			continue
		}
		txnID, err := lookupWechatTransactionIDForOrder(orderID)
		if err != nil {
			continue
		}
		cnt, qErr := queryProfitSharingPendingReceiverCount(ctx, outNo, txnID)
		if qErr != nil {
			// 微信查询频率受限 / 单笔失败：跳过该笔，不阻塞整体响应
			tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "warn", "referrer_pending_profit_sharing_query_skipped",
				"wechat_profit_sharing", map[string]string{
					"ps_id":  formatID(psID),
					"error":  humanizeProfitSharingQueryError(qErr),
				})
			continue
		}
		if cnt <= 0 {
			continue
		}
		pendingCount += cnt
		paid, ok := parseFlexibleTime(paidAt)
		if !ok {
			continue
		}
		deadline := paid.Add(time.Duration(profitSharingExpireDays) * 24 * time.Hour)
		if minDeadline.IsZero() || deadline.Before(minDeadline) {
			minDeadline = deadline
		}
	}
	out := map[string]interface{}{"pending_count": pendingCount}
	if !minDeadline.IsZero() {
		out["deadline"] = minDeadline.Format("2006-01-02")
	}
	writeJSON(w, http.StatusOK, out)
}
