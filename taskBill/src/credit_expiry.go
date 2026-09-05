package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	creditExpiryPointsSourceType = "credit_expiry"
	creditExpiryTxnType          = "expiry"
	creditLotTTLYears            = 1
)

func parseBillingUTC(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp %q", s)
}

func creditLotExpiresAt(now string) (string, error) {
	t, err := parseBillingUTC(now)
	if err != nil {
		return "", err
	}
	return t.AddDate(creditLotTTLYears, 0, 0).Format("2006-01-02 15:04:05.000000"), nil
}

type expiredLotRow struct {
	id        int64
	accountID int64
	remaining int64
}

// expireCreditLots zeros expired credit lots for a tenant, reduces balance,
// writes billing_transaction (credit_expiry), and inserts BILLING_CREDIT_EXPIRED outbox.
func expireCreditLots(ctx context.Context, tenantID int64) error {
	if tenantID <= 0 {
		return nil
	}
	now := utcNow()
	conn, err := beginImmediateConn(ctx)
	if err != nil {
		return err
	}
	defer rollbackImmediateConn(ctx, conn)

	rows, err := conn.QueryContext(ctx, `
		SELECT id, account_id, remaining_points
		FROM billing_payment_ledger
		WHERE tenant_id = ?
		  AND remaining_points > 0
		  AND expires_at IS NOT NULL
		  AND expires_at <= ?
		ORDER BY account_id, id`, tenantID, now)
	if err != nil {
		return fmt.Errorf("expire credit lots query: %w", err)
	}
	var lots []expiredLotRow
	for rows.Next() {
		var r expiredLotRow
		if err := rows.Scan(&r.id, &r.accountID, &r.remaining); err != nil {
			rows.Close()
			return err
		}
		lots = append(lots, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(lots) == 0 {
		if err := commitImmediateConn(ctx, conn); err != nil {
			return err
		}
		conn = nil
		return nil
	}

	type accountExpiry struct {
		sum    int64
		lotIDs []int64
	}
	byAccount := map[int64]*accountExpiry{}
	for _, lot := range lots {
		a := byAccount[lot.accountID]
		if a == nil {
			a = &accountExpiry{}
			byAccount[lot.accountID] = a
		}
		a.sum += lot.remaining
		a.lotIDs = append(a.lotIDs, lot.id)
	}

	var emits []map[string]interface{}

	for accountID, agg := range byAccount {
		if agg.sum <= 0 {
			continue
		}
		for _, lotID := range agg.lotIDs {
			res, err := conn.ExecContext(ctx, `
				UPDATE billing_payment_ledger
				SET remaining_points = 0
				WHERE id = ? AND remaining_points > 0 AND expires_at IS NOT NULL AND expires_at <= ?`,
				lotID, now,
			)
			if err != nil {
				return fmt.Errorf("expire lot %d: %w", lotID, err)
			}
			affected, _ := res.RowsAffected()
			if affected != 1 {
				return fmt.Errorf("expire lot concurrent update on id=%d", lotID)
			}
		}

		var balance int64
		if err := conn.QueryRowContext(ctx, `
			SELECT balance FROM billing_account WHERE id = ?`, accountID).Scan(&balance); err != nil {
			return err
		}
		deduct := agg.sum
		if deduct > balance {
			deduct = balance
		}
		after := balance - deduct
		if _, err := conn.ExecContext(ctx, `
			UPDATE billing_account SET balance = ?, updated_at = ? WHERE id = ?`,
			after, now, accountID,
		); err != nil {
			return err
		}

		tid := generateSnowflakeID()
		txnID := fmt.Sprintf("credit_expiry:%d:%d", accountID, tid)
		desc := fmt.Sprintf("积分批次到期失效 %d 分", agg.sum)
		if _, err := conn.ExecContext(ctx, `
			INSERT INTO billing_transaction (
				id, account_id, transaction_type, amount, balance_before, balance_after,
				points_source_type, user_id, description, transaction_id, created_at, usage_amount
			) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, 0)`,
			tid, accountID, creditExpiryTxnType, -agg.sum, balance, after,
			creditExpiryPointsSourceType, desc, txnID, now,
		); err != nil {
			return err
		}

		payload := map[string]interface{}{
			"tenant_id":          formatID(tenantID),
			"account_id":         formatID(accountID),
			"expired_points":     agg.sum,
			"balance_deducted":   deduct,
			"transaction_id":     txnID,
			"transaction_db_id":  formatID(tid),
			"points_source_type": creditExpiryPointsSourceType,
			"transaction_type":   creditExpiryTxnType,
			"created_at":         now,
			"lot_ids":            lotIDsToStrings(agg.lotIDs),
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, `
			INSERT INTO billing_outbox_message (id, billing_transaction_id, event_type, payload, status, created_at)
			VALUES (?, ?, 'BILLING_CREDIT_EXPIRED', ?, 'pending', ?)`,
			generateSnowflakeID(), tid, string(raw), now,
		); err != nil {
			return err
		}
		emits = append(emits, payload)

		slog.InfoContext(ctx, "credit_lots_expired",
			"level", "info",
			"tenant_id", formatID(tenantID),
			"account_id", formatID(accountID),
			"expired_points", agg.sum,
			"balance_deducted", deduct,
			"lot_count", len(agg.lotIDs),
		)
	}

	if err := commitImmediateConn(ctx, conn); err != nil {
		return err
	}
	conn = nil

	for _, payload := range emits {
		go djangoEmitEvent(ctx, "BILLING_CREDIT_EXPIRED", payload)
	}
	return nil
}

func lotIDsToStrings(ids []int64) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, formatID(id))
	}
	return out
}

func listActiveCreditLots(ctx context.Context, tenantID int64) ([]map[string]interface{}, error) {
	now := utcNow()
	rows, err := db.QueryContext(ctx, `
		SELECT id, points, remaining_points, expires_at, channel
		FROM billing_payment_ledger
		WHERE tenant_id = ?
		  AND remaining_points > 0
		  AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY CASE WHEN expires_at IS NULL THEN 1 ELSE 0 END,
		         expires_at ASC, created_at ASC`, tenantID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, points, remaining int64
		var expiresAt, channel string
		if err := rows.Scan(&id, &points, &remaining, &expiresAt, &channel); err != nil {
			return nil, err
		}
		list = append(list, map[string]interface{}{
			"id":               formatID(id),
			"points":           points,
			"remaining_points": remaining,
			"expires_at":       expiresAt,
			"channel":          channel,
		})
	}
	return list, rows.Err()
}

func listCreditLotsFiltered(ctx context.Context, tenantID int64, userID string) ([]map[string]interface{}, error) {
	now := utcNow()
	q := `
		SELECT pl.id, pl.tenant_id, pl.account_id, pl.points, pl.remaining_points,
		       pl.expires_at, pl.channel, pl.created_at, pl.billing_transaction_id,
		       COALESCE(bt.user_id, '') AS user_id
		FROM billing_payment_ledger pl
		LEFT JOIN billing_transaction bt ON bt.id = pl.billing_transaction_id
		WHERE pl.remaining_points > 0
		  AND (pl.expires_at IS NULL OR pl.expires_at > ?)`
	args := []interface{}{now}
	if tenantID > 0 {
		q += ` AND pl.tenant_id = ?`
		args = append(args, tenantID)
	}
	if uid := strings.TrimSpace(userID); uid != "" {
		q += ` AND bt.user_id = ?`
		args = append(args, uid)
	}
	q += ` ORDER BY pl.expires_at ASC, pl.created_at ASC LIMIT 5000`

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, tid, accountID, points, remaining int64
		var expiresAt, channel, createdAt, uid string
		var txnDBID sql.NullInt64
		if err := rows.Scan(&id, &tid, &accountID, &points, &remaining, &expiresAt, &channel, &createdAt, &txnDBID, &uid); err != nil {
			return nil, err
		}
		row := map[string]interface{}{
			"id":               formatID(id),
			"tenant_id":        formatID(tid),
			"account_id":       formatID(accountID),
			"points":           points,
			"remaining_points": remaining,
			"expires_at":       expiresAt,
			"channel":          channel,
			"created_at":       createdAt,
			"user_id":          uid,
		}
		if txnDBID.Valid {
			row["billing_transaction_id"] = formatID(txnDBID.Int64)
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func listUserRecharges(ctx context.Context, userID string, tenantID int64) ([]map[string]interface{}, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	q := `
		SELECT bt.id, ba.tenant_id, bt.account_id, bt.amount, bt.points_source_type,
		       bt.transaction_id, bt.created_at, COALESCE(pl.expires_at, '') AS expires_at,
		       COALESCE(pl.channel, '') AS channel, COALESCE(pl.remaining_points, 0) AS remaining_points,
		       COALESCE(pl.provider_ref, '') AS provider_ref,
		       COALESCE(bt.description, '') AS description,
		       COALESCE(o.consent_id, '') AS order_consent_id
		FROM billing_transaction bt
		JOIN billing_account ba ON ba.id = bt.account_id
		LEFT JOIN billing_payment_ledger pl ON pl.billing_transaction_id = bt.id
		LEFT JOIN billing_resource_order o
		  ON o.user_id = bt.user_id
		 AND REPLACE(COALESCE(o.payment_ref, ''), 'wechat:', '') = COALESCE(pl.provider_ref, '')
		WHERE bt.transaction_type = 'recharge'
		  AND bt.user_id = ?`
	args := []interface{}{uid}
	if tenantID > 0 {
		q += ` AND ba.tenant_id = ?`
		args = append(args, tenantID)
	}
	q += ` ORDER BY bt.created_at DESC LIMIT 5000`

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, tid, accountID, amount, remaining int64
		var ptsSrc, txnID, created, expiresAt, channel, providerRef, description, orderConsentID string
		if err := rows.Scan(&id, &tid, &accountID, &amount, &ptsSrc, &txnID, &created, &expiresAt, &channel, &remaining, &providerRef, &description, &orderConsentID); err != nil {
			return nil, err
		}
		row := map[string]interface{}{
			"id":                 formatID(id),
			"tenant_id":          formatID(tid),
			"account_id":         formatID(accountID),
			"amount_points":      amount,
			"points_source_type": ptsSrc,
			"transaction_id":     txnID,
			"created_at":         created,
			"expires_at":         expiresAt,
			"channel":            channel,
			"remaining_points":   remaining,
			"description":        description,
			// OPT-20260825-017: 优先读订单当时签署的 consent_id（048 列），审计可对单笔支付。
			"order_consent_id": orderConsentID,
		}
		if providerRef != "" {
			row["provider_ref"] = providerRef
		}
		list = append(list, row)
	}
	return list, rows.Err()
}
