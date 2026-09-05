package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

// ErrRefundDisabled is returned when platform refund applications are turned off.
var ErrRefundDisabled = errors.New("refund applications disabled")

func ensureRefundPolicyRow(ctx context.Context) error {
	now := utcNow()
	_, err := db.ExecContext(ctx, `
		INSERT IGNORE INTO billing_refund_policy (id, enabled, updated_at)
		VALUES (1, 1, ?)`, now)
	return err
}

func getRefundPolicyEnabled(ctx context.Context) (bool, error) {
	if err := ensureRefundPolicyRow(ctx); err != nil {
		return false, err
	}
	var enabled int
	err := db.QueryRowContext(ctx, `SELECT enabled FROM billing_refund_policy WHERE id = 1`).Scan(&enabled)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return enabled != 0, nil
}

func setRefundPolicyEnabled(ctx context.Context, enabled bool, actorUserID string) error {
	if err := ensureRefundPolicyRow(ctx); err != nil {
		return err
	}
	val := 0
	if enabled {
		val = 1
	}
	now := utcNow()
	res, err := db.ExecContext(ctx, `
		UPDATE billing_refund_policy SET enabled = ?, updated_at = ? WHERE id = 1`, val, now)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("refund policy row missing")
	}
	slog.InfoContext(ctx, "refund_policy_updated",
		"level", "info",
		"enabled", enabled,
		"actor_user_id", actorUserID,
	)
	return nil
}

func refundPolicyJSON(enabled bool) map[string]interface{} {
	return map[string]interface{}{
		"enabled": enabled,
	}
}
