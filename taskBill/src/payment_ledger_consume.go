package main

import (
	"context"
	"database/sql"
	"fmt"
)

// consumePaymentLedgerFEFO reduces remaining_points on credit lots in First-Expiry-First-Out
// order (expires_at ASC, created_at ASC). Expired lots should already be cleared by
// expireCreditLots; the WHERE clause skips them as a safety net.
// If cost exceeds ledger remaining, the rest is treated as non-ledger points.
func consumePaymentLedgerFEFO(ctx context.Context, tx *sql.Tx, tenantID int64, cost int64) error {
	if cost <= 0 {
		return nil
	}
	now := utcNow()
	rows, err := tx.QueryContext(ctx, `
		SELECT id, remaining_points
		FROM billing_payment_ledger
		WHERE tenant_id = ?
		  AND remaining_points > 0
		  AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY CASE WHEN expires_at IS NULL THEN 1 ELSE 0 END,
		         expires_at ASC, created_at ASC`, tenantID, now)
	if err != nil {
		return fmt.Errorf("ledger fefo query: %w", err)
	}
	defer rows.Close()

	type row struct {
		id        int64
		remaining int64
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.remaining); err != nil {
			return err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	left := cost
	for _, r := range list {
		if left <= 0 {
			break
		}
		take := r.remaining
		if take > left {
			take = left
		}
		res, err := tx.ExecContext(ctx, `
			UPDATE billing_payment_ledger
			SET remaining_points = remaining_points - ?
			WHERE id = ? AND remaining_points >= ?`,
			take, r.id, take,
		)
		if err != nil {
			return fmt.Errorf("ledger fefo update: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected != 1 {
			return fmt.Errorf("ledger fefo concurrent update on id=%d", r.id)
		}
		left -= take
	}
	return nil
}

// consumePaymentLedgerFIFO is retained as an alias for callers/tests that still use the old name.
func consumePaymentLedgerFIFO(ctx context.Context, tx *sql.Tx, tenantID int64, cost int64) error {
	return consumePaymentLedgerFEFO(ctx, tx, tenantID, cost)
}
