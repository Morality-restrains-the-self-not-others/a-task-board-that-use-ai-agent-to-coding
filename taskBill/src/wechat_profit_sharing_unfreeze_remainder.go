package main

import (
	"context"
	"fmt"
	"strings"

	"tracelog"
)

const wechatRemainderUnfreezeReason = "无分账接收方，解冻剩余资金"

func wechatUnfreezeOutOrderNo(orderID int64) string {
	return fmt.Sprintf("UF%d", orderID)
}

// maybeUnfreezeWechatRemainderIfNoReceiver 在已打分账标但无佣金接收方时
// 解冻剩余资金，避免货款冻结至微信 30 天自动解冻。mock / 非微信渠道跳过。
func maybeUnfreezeWechatRemainderIfNoReceiver(ctx context.Context, orderID int64, paymentMethod string) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !strings.EqualFold(strings.TrimSpace(paymentMethod), "wechat") {
		return
	}
	if wechatIsMock() {
		return
	}
	d := db
	if d == nil {
		return
	}
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM billing_profit_sharing WHERE order_id = ?`, orderID).Scan(&n); err != nil {
		tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "warn", "unfreeze remainder count failed", "wechat_profit_sharing", map[string]string{
			"order_id": fmt.Sprintf("%d", orderID),
			"err":      truncateBytes([]byte(err.Error()), 300),
		})
		return
	}
	if n > 0 {
		return
	}
	if wechatClient == nil {
		return
	}
	txnID, err := lookupWechatTransactionIDOnDB(d, orderID)
	if err != nil {
		tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "warn", "unfreeze remainder missing transaction_id", "wechat_profit_sharing", map[string]string{
			"order_id": fmt.Sprintf("%d", orderID),
			"err":      truncateBytes([]byte(err.Error()), 300),
		})
		return
	}
	outNo := wechatUnfreezeOutOrderNo(orderID)
	if err := unfreezeProfitSharing(ctx, outNo, txnID, wechatRemainderUnfreezeReason); err != nil {
		tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "warn", "unfreeze remainder failed", "wechat_profit_sharing", map[string]string{
			"order_id":     fmt.Sprintf("%d", orderID),
			"out_order_no": outNo,
			"err":          truncateBytes([]byte(err.Error()), 300),
		})
		return
	}
	tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "info", "unfreeze remainder ok", "wechat_profit_sharing", map[string]string{
		"order_id":     fmt.Sprintf("%d", orderID),
		"out_order_no": outNo,
	})
}
