package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"tracelog"
)

const (
	resourceGrantExpiryPointsSourceType = "resource_grant_expiry"
	resourceGrantExpiryTxnType          = "expiry"
)

// expireResourceGrants zeros expired resource grants (task_post) for a tenant,
// reduces task_post_quota, writes billing_transaction, and inserts outbox.
// Follows the same pattern as expireCreditLots but operates on billing_resource_grant.
func expireResourceGrants(ctx context.Context, tenantID int64) error {
	if tenantID <= 0 {
		return nil
	}
	now := utcNow()
	conn, err := beginImmediateConn(ctx)
	if err != nil {
		return err
	}
	defer rollbackImmediateConn(ctx, conn)

	// Find expired task_post grants with remaining > 0
	rows, err := conn.QueryContext(ctx, `
		SELECT id, remaining
		FROM billing_resource_grant
		WHERE tenant_id = ?
		  AND resource_type = 'task_post'
		  AND remaining > 0
		  AND expires_at IS NOT NULL
		  AND expires_at <= ?
		ORDER BY id`, tenantID, now)
	if err != nil {
		return fmt.Errorf("expire resource grants query: %w", err)
	}

	type expiredGrant struct {
		id        int64
		remaining int64
	}
	var grants []expiredGrant
	for rows.Next() {
		var g expiredGrant
		if err := rows.Scan(&g.id, &g.remaining); err != nil {
			rows.Close()
			return err
		}
		grants = append(grants, g)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(grants) == 0 {
		if err := commitImmediateConn(ctx, conn); err != nil {
			return err
		}
		conn = nil
		return nil
	}

	// Calculate total expired and zero out each grant
	var totalExpired int64
	for _, g := range grants {
		res, err := conn.ExecContext(ctx, `
			UPDATE billing_resource_grant
			SET remaining = 0
			WHERE id = ? AND remaining > 0 AND expires_at IS NOT NULL AND expires_at <= ?`,
			g.id, now,
		)
		if err != nil {
			return fmt.Errorf("expire resource grant %d: %w", g.id, err)
		}
		affected, _ := res.RowsAffected()
		if affected != 1 {
			return fmt.Errorf("expire resource grant concurrent update on id=%d", g.id)
		}
		totalExpired += g.remaining
	}

	if totalExpired <= 0 {
		if err := commitImmediateConn(ctx, conn); err != nil {
			return err
		}
		conn = nil
		return nil
	}

	// Read current task_post_quota and account id
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		return err
	}

	var quota int64
	if err := conn.QueryRowContext(ctx, `
		SELECT task_post_quota FROM billing_account WHERE id = ?`, acc.ID,
	).Scan(&quota); err != nil {
		return err
	}

	deduct := totalExpired
	if deduct > quota {
		deduct = quota
	}
	after := quota - deduct
	if _, err := conn.ExecContext(ctx, `
		UPDATE billing_account SET task_post_quota = ?, updated_at = ? WHERE id = ?`,
		after, now, acc.ID,
	); err != nil {
		return err
	}

	// Write billing_transaction
	tid := generateSnowflakeID()
	txnID := fmt.Sprintf("resource_grant_expiry:%d:%d", tenantID, tid)
	desc := fmt.Sprintf("赠送任务帖到期失效 %d 个", totalExpired)
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, user_id, description, transaction_id, created_at, usage_amount
		) VALUES (?, ?, ?, 0, 0, 0, ?, NULL, ?, ?, ?, ?)`,
		tid, acc.ID, resourceGrantExpiryTxnType,
		resourceGrantExpiryPointsSourceType, desc, txnID, now, totalExpired,
	); err != nil {
		return err
	}

	// Write outbox
	payload := map[string]interface{}{
		"tenant_id":           formatID(tenantID),
		"account_id":          formatID(acc.ID),
		"expired_task_posts":  totalExpired,
		"quota_before":        quota,
		"quota_after":         after,
		"transaction_id":      txnID,
		"transaction_db_id":   formatID(tid),
		"points_source_type":  resourceGrantExpiryPointsSourceType,
		"transaction_type":    resourceGrantExpiryTxnType,
		"created_at":          now,
		"expired_grant_count": len(grants),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO billing_outbox_message (id, billing_transaction_id, event_type, payload, status, created_at)
		VALUES (?, ?, 'BILLING_RESOURCE_GRANT_EXPIRED', ?, 'pending', ?)`,
		generateSnowflakeID(), tid, string(raw), now,
	); err != nil {
		return err
	}

	if err := commitImmediateConn(ctx, conn); err != nil {
		return err
	}
	conn = nil

	slog.InfoContext(ctx, "resource_grants_expired",
		"level", "info",
		"tenant_id", formatID(tenantID),
		"account_id", formatID(acc.ID),
		"expired_task_posts", totalExpired,
		"quota_before", quota,
		"quota_after", after,
		"grant_count", len(grants),
	)

	go djangoEmitEvent(ctx, "BILLING_RESOURCE_GRANT_EXPIRED", payload)
	return nil
}

// handleInternalExpireResourceGrants POST /api/internal/taskbill/expire-resource-grants/
// 过期任务帖赠品配额回收（供 cron 定时调用）
func handleInternalExpireResourceGrants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err := expireResourceGrants(r.Context(), tid); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	acc, _, err := getOrCreateBillingAccount(tid, false)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "ok",
		"task_post_quota": acc.TaskPostQuota,
	})
}
