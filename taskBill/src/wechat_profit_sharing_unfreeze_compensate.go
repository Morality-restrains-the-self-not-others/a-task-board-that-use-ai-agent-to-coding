package main

import (
	"context"
	"log"
	"time"
)

// OPT-20260823-042：无佣金行的已打标微信单解冻失败补偿扫描。
//
// maybeUnfreezeWechatRemainderIfNoReceiver 在支付成功 goroutine 内对无
// billing_profit_sharing 行的微信单调用 Unfreeze；为避免与落佣金行竞态，
// already-paid 回调重放不再解冻，Unfreeze 瞬时失败时货款会冻到微信约 30 天
// 自动解冻。本扫描在 billing_profit_sharing_scan timer 内重试：已支付微信单、
// 无佣金行、paid_at 落在 [now-30d, now-10min]（给首次解冻留窗口且不超过微信
// 冻结窗口）即调用 Unfreeze。依赖微信对相同 out_order_no（UF{order_id}）的
// 幂等，重复调用安全；成功/失败均经 tracelog 事件可观测。

const wechatUnfreezeCompensateAfter = 10 * time.Minute
const wechatUnfreezeCompensateBatch = 20

// compensateUnfreezeRemainderForPaidWechatOrders 补偿解冻无佣金行的已支付微信单。
func compensateUnfreezeRemainderForPaidWechatOrders(now time.Time) error {
	if wechatIsMock() || wechatClient == nil {
		return nil
	}
	now = now.UTC()
	cutoff := now.Add(-wechatUnfreezeCompensateAfter).Format(time.RFC3339)
	windowStart := now.AddDate(0, 0, -profitSharingWechatWindowDays).Format(time.RFC3339)
	rows, err := db.Query(`
		SELECT o.id, o.tenant_id
		FROM billing_resource_order o
		LEFT JOIN billing_profit_sharing ps ON ps.order_id = o.id
		WHERE ps.id IS NULL
		  AND o.status = 'paid'
		  AND o.payment_method = 'wechat'
		  AND o.paid_at IS NOT NULL AND o.paid_at != ''
		  AND o.paid_at <= ?
		  AND o.paid_at > ?
		LIMIT ?`, cutoff, windowStart, wechatUnfreezeCompensateBatch)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, tenantID int64
		if err := rows.Scan(&id, &tenantID); err != nil {
			log.Printf("[taskBill] profit sharing unfreeze compensate scan row: %v", err)
			continue
		}
		log.Printf("[taskBill] profit sharing unfreeze compensate order=%s", formatID(id))
		maybeUnfreezeWechatRemainderIfNoReceiver(context.Background(), id, "wechat")
	}
	return rows.Err()
}
