package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

const (
	profitSharingFallbackAfterDays = 25
	profitSharingWechatWindowDays  = 30
)

func profitSharingFallbackWindow(now time.Time) (cutoff25, cutoff30 string) {
	now = now.UTC()
	cutoff25 = now.AddDate(0, 0, -profitSharingFallbackAfterDays).Format(time.RFC3339)
	cutoff30 = now.AddDate(0, 0, -profitSharingWechatWindowDays).Format(time.RFC3339)
	return
}

func scanProfitSharingWorkItems(rows *sql.Rows) ([]profitSharingRecord, error) {
	var records []profitSharingRecord
	for rows.Next() {
		var r profitSharingRecord
		if err := rows.Scan(&r.ID, &r.OutProfitSharingNo, &r.OrderID, &r.OrderNumber,
			&r.TenantID, &r.ReferrerUserID, &r.ReferrerOpenid,
			&r.CommissionYuanCents, &r.TotalYuanCents); err != nil {
			log.Printf("[taskBill] scan profit sharing row: %v", err)
			continue
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func queryFallbackProfitSharings(now time.Time) ([]profitSharingRecord, error) {
	cutoff25, cutoff30 := profitSharingFallbackWindow(now)
	rows, err := db.Query(`
		SELECT ps.id, ps.out_profit_sharing_no, ps.order_id, ps.order_number, ps.tenant_id,
		       ps.referrer_user_id, ps.referrer_openid, ps.commission_yuan_cents, ps.total_yuan_cents
		FROM billing_profit_sharing ps
		INNER JOIN billing_resource_order o ON o.id = ps.order_id
		WHERE ps.status IN (?, ?)
		  AND o.status = 'paid'
		  AND o.paid_at IS NOT NULL AND o.paid_at != ''
		  AND o.paid_at <= ?
		  AND o.paid_at > ?
		  AND (ps.wechat_profit_sharing_id IS NULL OR ps.wechat_profit_sharing_id = '')
		ORDER BY o.paid_at ASC
		LIMIT 50`, psStatusPending, psStatusFailed, cutoff25, cutoff30)
	if err != nil {
		return nil, fmt.Errorf("query fallback profit sharings: %w", err)
	}
	defer rows.Close()
	return scanProfitSharingWorkItems(rows)
}

func markUnmarkedFallbackProfitSharings(now time.Time) (int, error) {
	cutoff25, cutoff30 := profitSharingFallbackWindow(now)
	rows, err := db.Query(`
		SELECT o.id, o.tenant_id
		FROM billing_resource_order o
		LEFT JOIN billing_profit_sharing ps ON ps.order_id = o.id
		WHERE ps.id IS NULL
		  AND o.status = 'paid'
		  AND o.user_id > 0
		  AND o.paid_at IS NOT NULL AND o.paid_at != ''
		  AND o.paid_at <= ?
		  AND o.paid_at > ?
		LIMIT 20`, cutoff25, cutoff30)
	if err != nil {
		return 0, fmt.Errorf("query unmarked fallback orders: %w", err)
	}
	defer rows.Close()

	type orderRef struct {
		id       int64
		tenantID int64
	}
	var orders []orderRef
	for rows.Next() {
		var o orderRef
		if err := rows.Scan(&o.id, &o.tenantID); err != nil {
			log.Printf("[taskBill] scan unmarked fallback order: %v", err)
			continue
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	marked := 0
	ctx := context.Background()
	for _, o := range orders {
		if err := markOrderForProfitSharing(ctx, o.id, o.tenantID); err != nil {
			log.Printf("[taskBill] profit sharing fallback mark order=%d err=%v", o.id, err)
			continue
		}
		log.Printf("[taskBill] profit sharing fallback marked missing row order=%d", o.id)
		marked++
	}
	return marked, nil
}

func mergeProfitSharingRecords(parts ...[]profitSharingRecord) []profitSharingRecord {
	seen := make(map[int64]struct{})
	var out []profitSharingRecord
	for _, list := range parts {
		for _, r := range list {
			if _, ok := seen[r.ID]; ok {
				continue
			}
			seen[r.ID] = struct{}{}
			out = append(out, r)
		}
	}
	return out
}
