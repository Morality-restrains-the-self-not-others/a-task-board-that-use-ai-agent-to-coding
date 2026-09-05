package main

import (
	"fmt"
	"log"
	"time"
)

// OPT-20260823-050：支付超过微信约 30 天冻结窗口仍未成功分账的可观测告警。
//
// 25 天兜底只处理 paid_at 落在 (now-30d, now-25d] 的订单；超出窗口后
// CreateOrder 会失败、扫描直接跳过，漏分账只能靠人工对账。本扫描对
// 「paid 且窗口外、分账非 finished」的订单计数并打 warn 日志（只含
// order_id/status/paid_at，不含 openid），Loki 侧可按
// `{job="task-bill"} |= "OVER_WINDOW_UNSHARED"` 建告警；计数也可由未来
// Prometheus gauge 复用。

type overWindowProfitSharingRow struct {
	OrderID int64
	Status  string
	PaidAt  string
}

// queryOverWindowProfitSharings 返回 paid 且 paid_at 早于 30 天窗口、分账状态
// 非 finished（pending/failed/processing）的订单数及最多 100 条明细。
func queryOverWindowProfitSharings(now time.Time) (int64, []overWindowProfitSharingRow, error) {
	cutoff30 := now.UTC().AddDate(0, 0, -profitSharingWechatWindowDays).Format(time.RFC3339)
	cond := `
		FROM billing_profit_sharing ps
		INNER JOIN billing_resource_order o ON o.id = ps.order_id
		WHERE o.status = 'paid'
		  AND o.paid_at IS NOT NULL AND o.paid_at != ''
		  AND o.paid_at <= ?
		  AND ps.status <> ?`
	var count int64
	if err := db.QueryRow(`SELECT COUNT(*) `+cond, cutoff30, psStatusFinished).Scan(&count); err != nil {
		return 0, nil, fmt.Errorf("count over-window profit sharings: %w", err)
	}
	rows, err := db.Query(`
		SELECT ps.order_id, ps.status, o.paid_at `+cond+`
		ORDER BY o.paid_at ASC
		LIMIT 100`, cutoff30, psStatusFinished)
	if err != nil {
		return 0, nil, fmt.Errorf("query over-window profit sharings: %w", err)
	}
	defer rows.Close()
	var items []overWindowProfitSharingRow
	for rows.Next() {
		var it overWindowProfitSharingRow
		if err := rows.Scan(&it.OrderID, &it.Status, &it.PaidAt); err != nil {
			return count, items, fmt.Errorf("scan over-window profit sharing: %w", err)
		}
		items = append(items, it)
	}
	return count, items, rows.Err()
}

// logOverWindowProfitSharings 对超出窗口仍非 finished 的分账记录打 warn 日志。
// 幂等、只读，不触发任何出站调用；任何 tick 失败仅记一行，不影响分账主流程。
func logOverWindowProfitSharings(now time.Time) {
	count, items, err := queryOverWindowProfitSharings(now)
	if err != nil {
		log.Printf("[taskBill] profit sharing over-window scan failed: %v", err)
		return
	}
	if count == 0 {
		return
	}
	log.Printf("[taskBill] profit sharing OVER_WINDOW_UNSHARED count=%d (paid>%dd, ps not finished; 已无法补分，需人工对账)",
		count, profitSharingWechatWindowDays)
	for _, it := range items {
		log.Printf("[taskBill] profit sharing over-window unshared order_id=%s status=%s paid_at=%s",
			formatID(it.OrderID), it.Status, it.PaidAt)
	}
}
