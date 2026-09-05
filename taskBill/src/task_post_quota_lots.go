package main

import (
	"context"
	"database/sql"
	"log/slog"
)

const (
	grantSourceGift     = "gift"
	grantSourcePurchase = "purchase"
)

type quotaLotHit struct {
	GrantID    int64
	OrderID    int64
	SourceKind string
}

type taskPostQuotaSplit struct {
	Total     int64
	Gifted    int64
	Purchased int64
}

func getTaskPostQuotaSplit(tenantID int64) (taskPostQuotaSplit, error) {
	total, err := getTaskPostQuotaRemaining(tenantID)
	if err != nil {
		return taskPostQuotaSplit{}, err
	}
	var gifted sql.NullInt64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(remaining), 0) FROM billing_resource_grant
		WHERE tenant_id = ? AND resource_type = ?
		  AND COALESCE(source_kind, ?) = ?
		  AND remaining > 0
		  AND (expires_at IS NULL OR expires_at > ?)`,
		tenantID, ResourceTypeTaskPost, grantSourceGift, grantSourceGift, utcNow(),
	).Scan(&gifted)
	if err != nil {
		return taskPostQuotaSplit{}, err
	}
	g := gifted.Int64
	if g > total {
		g = total
	}
	if g < 0 {
		g = 0
	}
	return taskPostQuotaSplit{Total: total, Gifted: g, Purchased: total - g}, nil
}

func insertTaskPostPurchaseLotTx(tx *sql.Tx, tenantID, orderID, quantity int64, now string) error {
	if quantity <= 0 {
		return nil
	}
	id := generateSnowflakeID()
	_, err := tx.Exec(`
		INSERT INTO billing_resource_grant (
			id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at, source_kind, order_id
		) VALUES (?, ?, ?, ?, ?, 'resource_purchase', NULL, ?, ?, ?)`,
		id, tenantID, ResourceTypeTaskPost, quantity, quantity, now, grantSourcePurchase, orderID,
	)
	return err
}

func ensureTaskPostPurchaseLots(ctx context.Context, tenantID int64) error {
	split, err := getTaskPostQuotaSplit(tenantID)
	if err != nil {
		return err
	}
	var existingPurchaseRem sql.NullInt64
	if err := db.QueryRow(`
		SELECT COALESCE(SUM(remaining), 0) FROM billing_resource_grant
		WHERE tenant_id = ? AND resource_type = ? AND source_kind = ?`,
		tenantID, ResourceTypeTaskPost, grantSourcePurchase,
	).Scan(&existingPurchaseRem); err != nil {
		return err
	}
	unallocated := split.Purchased - existingPurchaseRem.Int64
	if unallocated < 0 {
		unallocated = 0
	}

	rows, err := db.Query(`
		SELECT o.id, i.quantity
		FROM billing_resource_order o
		JOIN billing_resource_order_item i ON i.order_id = o.id
		WHERE o.tenant_id = ? AND o.status = 'paid'
		  AND COALESCE(o.payment_method, '') != 'admin_grant'
		  AND i.resource_type = ?
		ORDER BY COALESCE(o.paid_at, o.created_at) DESC, o.id DESC`,
		tenantID, ResourceTypeTaskPost,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		orderID  int64
		quantity int64
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.orderID, &it.quantity); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	created := 0
	for _, it := range items {
		var exists int64
		if err := db.QueryRow(`
			SELECT COUNT(*) FROM billing_resource_grant
			WHERE tenant_id = ? AND order_id = ? AND resource_type = ? AND source_kind = ?`,
			tenantID, it.orderID, ResourceTypeTaskPost, grantSourcePurchase,
		).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		rem := int64(0)
		if unallocated > 0 {
			rem = it.quantity
			if rem > unallocated {
				rem = unallocated
			}
			unallocated -= rem
		}
		id := generateSnowflakeID()
		if _, err := db.Exec(`
			INSERT INTO billing_resource_grant (
				id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at, source_kind, order_id
			) VALUES (?, ?, ?, ?, ?, 'resource_purchase_backfill', NULL, ?, ?, ?)`,
			id, tenantID, ResourceTypeTaskPost, it.quantity, rem, utcNow(), grantSourcePurchase, it.orderID,
		); err != nil {
			return err
		}
		created++
	}
	if created > 0 {
		slog.InfoContext(ctx, "task_post_purchase_lots_backfilled",
			"level", "info",
			"tenant_id", formatID(tenantID),
			"lots_created", created,
		)
	}
	return nil
}

func consumeTaskPostQuotaLotTx(ctx context.Context, tx *sql.Tx, tenantID int64) (*quotaLotHit, error) {
	now := utcNow()
	hit := &quotaLotHit{SourceKind: grantSourcePurchase}
	var grantID sql.NullInt64
	var orderID sql.NullInt64
	var sourceKind sql.NullString
	err := tx.QueryRow(`
		SELECT id, order_id, COALESCE(source_kind, ?)
		FROM billing_resource_grant
		WHERE tenant_id = ? AND resource_type = ?
		  AND remaining > 0
		  AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY CASE WHEN COALESCE(source_kind, ?) = ? THEN 0 ELSE 1 END,
		         CASE WHEN expires_at IS NULL THEN 1 ELSE 0 END,
		         expires_at ASC, created_at ASC
		LIMIT 1
		FOR UPDATE`,
		grantSourceGift, tenantID, ResourceTypeTaskPost, now, grantSourceGift, grantSourceGift,
	).Scan(&grantID, &orderID, &sourceKind)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && grantID.Valid {
		res, err := tx.Exec(`UPDATE billing_resource_grant SET remaining = remaining - 1 WHERE id = ? AND remaining > 0`, grantID.Int64)
		if err != nil {
			return nil, err
		}
		n, _ := res.RowsAffected()
		if n == 1 {
			hit.GrantID = grantID.Int64
			if orderID.Valid {
				hit.OrderID = orderID.Int64
			}
			if sourceKind.Valid && sourceKind.String != "" {
				hit.SourceKind = sourceKind.String
			} else {
				hit.SourceKind = grantSourceGift
			}
		}
	}

	result, err := tx.Exec(`
		UPDATE billing_account SET task_post_quota = task_post_quota - 1, updated_at = ?
		WHERE tenant_id = ? AND task_post_quota > 0`, now, tenantID)
	if err != nil {
		return nil, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, &InsufficientBalanceError{BalancePoints: 0, RequiredPoints: 1}
	}
	if hit.GrantID != 0 {
		slog.InfoContext(ctx, "task_post_lot_consumed",
			"level", "info",
			"tenant_id", formatID(tenantID),
			"grant_id", formatID(hit.GrantID),
			"order_id", formatID(hit.OrderID),
			"source_kind", hit.SourceKind,
		)
	}
	return hit, nil
}
