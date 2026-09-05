package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	referralCommissionRateNum    = int64(5)
	referralCommissionRateDen    = int64(100)
	referralValidityDays         = 365
	referralSettleDelayDays      = 15
	referralCommissionSourceType = "referral_commission"
	referralAccrualPending       = "pending"
	referralAccrualSettled       = "settled"
	referralAccrualVoided        = "voided"
)

func getReferralConfigForDisplay() commissionRateInfo {
	percent := getReferralRatePercent()
	return commissionRateInfo{
		Percent: percent,
		Display: formatCommissionRateDisplay(percent),
		Source:  "fixed_5",
	}
}

func referralCommissionPoints(consumptionPoints int64) int64 {
	if consumptionPoints <= 0 {
		return 0
	}
	num := getReferralRatePercent()
	// ROUND_HALF_UP(points * rate / 100)
	return (consumptionPoints*num + referralCommissionRateDen/2) / referralCommissionRateDen
}

func parseBillingTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q", s)
}

func upsertReferralEdge(referrerUserID, referredUserID string, referrerTenantID int64, boundAt, channelCode string, commissionEligible bool) error {
	referrerUserID = strings.TrimSpace(referrerUserID)
	referredUserID = strings.TrimSpace(referredUserID)
	channelCode = strings.TrimSpace(channelCode)
	if referrerUserID == "" || referredUserID == "" || referrerTenantID <= 0 {
		return fmt.Errorf("referrer_user_id, referred_user_id, referrer_tenant_id required")
	}
	if referrerUserID == referredUserID {
		return fmt.Errorf("self-referral not allowed")
	}
	if strings.TrimSpace(boundAt) == "" {
		boundAt = utcNow()
	}
	if _, err := parseBillingTime(boundAt); err != nil {
		return err
	}
	now := utcNow()
	eligible := 0
	if commissionEligible {
		eligible = 1
	}
	_, err := db.Exec(`
			INSERT INTO billing_referral_edge (
				referred_user_id, referrer_user_id, referrer_tenant_id, bound_at,
				channel_code, commission_eligible, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				referrer_user_id=VALUES(referrer_user_id),
				referrer_tenant_id=VALUES(referrer_tenant_id),
				bound_at=VALUES(bound_at),
				channel_code=IF(channel_code='', VALUES(channel_code), channel_code),
				commission_eligible=commission_eligible,
				updated_at=VALUES(updated_at)
		`, referredUserID, referrerUserID, referrerTenantID, boundAt, channelCode, eligible, now, now)
	return err
}

func tryAccrueReferralFromConsumption(
	ctx context.Context,
	referredUserID string,
	sourceTxnID string,
	sourceTxnDBID int64,
	consumptionPoints int64,
	consumedAt string,
) {
	if err := accrueReferralFromConsumption(ctx, referredUserID, sourceTxnID, sourceTxnDBID, consumptionPoints, consumedAt); err != nil {
		slog.Warn("referral accrue failed", "error", err.Error(), "user_id", referredUserID, "txn", sourceTxnID)
	}
}

func accrueReferralFromConsumption(
	ctx context.Context,
	referredUserID string,
	sourceTxnID string,
	sourceTxnDBID int64,
	consumptionPoints int64,
	consumedAt string,
) error {
	referredUserID = strings.TrimSpace(referredUserID)
	sourceTxnID = strings.TrimSpace(sourceTxnID)
	if referredUserID == "" || sourceTxnID == "" || consumptionPoints <= 0 {
		return nil
	}
	var (
		referrerUserID   string
		referrerTenantID int64
		boundAt          string
		channelCode      string
	)
	err := db.QueryRow(`
		SELECT referrer_user_id, referrer_tenant_id, bound_at, channel_code
		FROM billing_referral_edge WHERE referred_user_id = ?`, referredUserID,
	).Scan(&referrerUserID, &referrerTenantID, &boundAt, &channelCode)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	// 与微信分账同口径（ADR-0033）：资格在消费/支付时刻现查，不再依赖绑边
	// commission_eligible 快照（先推荐后获资的边此前永远不计提点数）。
	if !lookupReferrerPaytimeQualification(ctx, referrerUserID) {
		slog.Info("referral accrue skipped ineligible", "user_id", referredUserID, "txn", sourceTxnID)
		return nil
	}
	consumed, err := parseBillingTime(consumedAt)
	if err != nil {
		return err
	}
	bound, err := parseBillingTime(boundAt)
	if err != nil {
		return err
	}
	if consumed.Before(bound) || !consumed.Before(bound.AddDate(0, 0, referralValidityDays)) {
		return nil
	}
	points := referralCommissionPoints(consumptionPoints)
	if points <= 0 {
		return nil
	}
	settleAfter := consumed.AddDate(0, 0, getReferralSettleDelayDays()).Format("2006-01-02 15:04:05.000000")
	now := utcNow()
	id := generateSnowflakeID()
	_, err = db.Exec(`
				INSERT INTO billing_referral_commission_accrual (
					id, referrer_user_id, referred_user_id, referrer_tenant_id,
					source_txn_id, source_txn_db_id, consumption_points, commission_points,
					status, consumed_at, settle_after, channel_code, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE source_txn_id = source_txn_id
			`, id, referrerUserID, referredUserID, referrerTenantID,
		sourceTxnID, nullInt64(sourceTxnDBID), consumptionPoints, points,
		referralAccrualPending, consumedAt, settleAfter, channelCode, now, now)
	return err
}

func nullInt64(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

// backfillReferralAccrualsForReferredUser accrues commission for historical
// consumption rows after a referral edge is first bound.
func orderReferralSourceTxnID(orderNumber string) string {
	return "order:" + strings.TrimSpace(orderNumber)
}

// billingBuyerUserID treats BIGINT 0 / empty as missing (resource orders default user_id=0).
func billingBuyerUserID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || s == "0" {
		return ""
	}
	return s
}

func voidReferralAccrualForOrderID(orderID int64) error {
	if orderID <= 0 {
		return nil
	}
	var num string
	err := db.QueryRow(`SELECT order_number FROM billing_resource_order WHERE id = ?`, orderID).Scan(&num)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	return voidReferralAccrualBySourceTxn(orderReferralSourceTxnID(num), "refund")
}

func backfillReferralAccrualsForReferredUser(ctx context.Context, referredUserID string) (int, error) {
	referredUserID = strings.TrimSpace(referredUserID)
	if referredUserID == "" {
		return 0, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT t.transaction_id, t.id, t.amount, t.created_at
		FROM billing_transaction t
		LEFT JOIN billing_resource_order o
		  ON (t.related_order_id IS NOT NULL AND t.related_order_id = o.id)
		  OR (t.related_order_id IS NULL AND t.transaction_id = CONCAT('order:', o.order_number))
		WHERE t.transaction_type = 'consumption'
		  AND t.amount > 0
		  AND (
		    t.user_id = ?
		    OR (o.user_id > 0 AND CAST(o.user_id AS CHAR) = ?)
		  )
		  AND (o.id IS NULL OR o.status NOT IN ('refunded', 'cancelled', 'expired'))
		GROUP BY t.id, t.transaction_id, t.amount, t.created_at
		ORDER BY t.created_at ASC
		LIMIT 5000
	`, referredUserID, referredUserID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var (
			txnID   string
			dbID    int64
			amount  int64
			created time.Time
		)
		if err := rows.Scan(&txnID, &dbID, &amount, &created); err != nil {
			return n, err
		}
		consumedAt := created.UTC().Format("2006-01-02 15:04:05.000000")
		if err := accrueReferralFromConsumption(ctx, referredUserID, txnID, dbID, amount, consumedAt); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func settleDueReferralCommissions(ctx context.Context, now time.Time) (settled int, err error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	nowStr := now.Format("2006-01-02 15:04:05.000000")
	rows, err := db.Query(`
		SELECT id, referrer_user_id, referrer_tenant_id, commission_points, source_txn_id
		FROM billing_referral_commission_accrual
		WHERE status = ? AND settle_after <= ?
		ORDER BY settle_after ASC
		LIMIT 500
	`, referralAccrualPending, nowStr)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type item struct {
		id, tenantID, points int64
		referrerUserID       string
		sourceTxnID          string
	}
	var due []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.referrerUserID, &it.tenantID, &it.points, &it.sourceTxnID); err != nil {
			return settled, err
		}
		due = append(due, it)
	}
	if err := rows.Err(); err != nil {
		return settled, err
	}

	for _, it := range due {
		settleTxnID := fmt.Sprintf("referral-settle:%d", it.id)
		// 余额充值路径已移除（2026-07-26），推荐佣金改为发放任务帖配额
		if _, err := adminGrantResources(
			ctx, it.tenantID,
			[]ResourceGrantInput{{ResourceType: ResourceTypeTaskPost, Quantity: it.points, ExpiresAt: ""}},
			it.referrerUserID, settleTxnID,
		); err != nil {
			slog.Error("referral settle grant failed", "accrual_id", it.id, "error", err.Error())
			continue
		}
		updated := utcNow()
		res, err := db.Exec(`
			UPDATE billing_referral_commission_accrual
			SET status = ?, settled_at = ?, settle_txn_id = ?, updated_at = ?
			WHERE id = ? AND status = ?`,
			referralAccrualSettled, updated, settleTxnID, updated, it.id, referralAccrualPending,
		)
		if err != nil {
			return settled, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			settled++
		}
	}
	return settled, nil
}

func voidReferralAccrualBySourceTxn(sourceTxnID, reason string) error {
	sourceTxnID = strings.TrimSpace(sourceTxnID)
	if sourceTxnID == "" {
		return nil
	}
	now := utcNow()
	_, err := db.Exec(`
		UPDATE billing_referral_commission_accrual
		SET status = ?, voided_at = ?, void_reason = ?, updated_at = ?
		WHERE source_txn_id = ? AND status = ?`,
		referralAccrualVoided, now, reason, now, sourceTxnID, referralAccrualPending,
	)
	return err
}

func referralCommissionSummary(referrerUserID string) (map[string]interface{}, error) {
	referrerUserID = strings.TrimSpace(referrerUserID)
	if referrerUserID == "" {
		return nil, fmt.Errorf("referrer_user_id required")
	}
	var pending, settled int64
	err := db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN status = ? THEN commission_points ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN commission_points ELSE 0 END), 0)
		FROM billing_referral_commission_accrual
		WHERE referrer_user_id = ?`,
		referralAccrualPending, referralAccrualSettled, referrerUserID,
	).Scan(&pending, &settled)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT a.id, a.referred_user_id, a.consumption_points, a.commission_points, a.status,
		       a.consumed_at, a.settle_after, a.settled_at, a.source_txn_id,
		       COALESCE(t.points_source_type, '') as resource_type,
		       COALESCE(t.amount, 0) as txn_amount
		FROM billing_referral_commission_accrual a
		LEFT JOIN billing_transaction t ON t.transaction_id = a.source_txn_id
		WHERE a.referrer_user_id = ? AND a.status IN (?, ?)
		ORDER BY a.consumed_at DESC
		LIMIT 100
	`, referrerUserID, referralAccrualPending, referralAccrualSettled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id, consPts, commPts, txnAmt                          int64
			referred, status, consumed, settleAfter, resourceType string
			settledAt, sourceTxn                                  sql.NullString
		)
		if err := rows.Scan(
			&id, &referred, &consPts, &commPts, &status,
			&consumed, &settleAfter, &settledAt, &sourceTxn,
			&resourceType, &txnAmt,
		); err != nil {
			return nil, err
		}
		row := map[string]interface{}{
			"id":                 formatID(id),
			"referred_user_id":   referred,
			"consumption_points": fmt.Sprintf("%d", consPts),
			"commission_points":  fmt.Sprintf("%d", commPts),
			"status":             status,
			"consumed_at":        consumed,
			"settle_after":       settleAfter,
			"source_txn_id":      sourceTxn.String,
			"resource_type":      resourceType,
		}
		if settledAt.Valid {
			row["settled_at"] = settledAt.String
		}
		items = append(items, row)
	}

	var firstLevelCount, activeCount int64
	_ = db.QueryRow(`SELECT COUNT(*) FROM billing_referral_edge WHERE referrer_user_id = ?`, referrerUserID).Scan(&firstLevelCount)
	cutoff := time.Now().UTC().AddDate(0, 0, -referralValidityDays).Format("2006-01-02 15:04:05.000000")
	_ = db.QueryRow(`
		SELECT COUNT(*) FROM billing_referral_edge
		WHERE referrer_user_id = ? AND bound_at > ?`, referrerUserID, cutoff,
	).Scan(&activeCount)

	rate := getReferralConfigForDisplay()
	return map[string]interface{}{
		"referrer_user_id":          referrerUserID,
		"pending_points":            fmt.Sprintf("%d", pending),
		"settled_points":            fmt.Sprintf("%d", settled),
		"commission_rate":           formatCommissionRateDecimal(rate.Percent),
		"commission_rate_display":   rate.Display,
		"validity_days":             fmt.Sprintf("%d", referralValidityDays),
		"settle_delay_days":         fmt.Sprintf("%d", getReferralSettleDelayDays()),
		"first_level_count":         fmt.Sprintf("%d", firstLevelCount),
		"first_level_active_count":  fmt.Sprintf("%d", activeCount),
		"items":                     items,
		"second_level_count":        "0",
		"second_level_rate":         "0",
		"second_level_rate_display": "0%",
	}, nil
}

// ---- 推荐消费月度汇总 ----

func consumptionMonthlyTotals(userIDs []string) (map[string]int64, error) {
	out := map[string]int64{}
	if len(userIDs) == 0 {
		return out, nil
	}
	placeholders := make([]byte, 0, len(userIDs)*2)
	args := make([]interface{}, 0, len(userIDs))
	for i, uid := range userIDs {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
		args = append(args, uid)
	}
	q := fmt.Sprintf(`
		SELECT DATE_FORMAT(created_at, '%%Y-%%m') AS month, SUM(amount) AS total
		FROM billing_transaction
		WHERE transaction_type = 'consumption'
		  AND user_id IN (%s)
		GROUP BY month`, string(placeholders))
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var month sql.NullString
		var total sql.NullInt64
		if err := rows.Scan(&month, &total); err != nil {
			return nil, err
		}
		if month.Valid && month.String != "" {
			out[month.String] = total.Int64
		}
	}
	return out, rows.Err()
}
