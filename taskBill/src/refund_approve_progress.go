package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// refundOutRefundNo builds a deterministic merchant refund id for channel idempotency.
// Retries of the same (application, ledger) allocation reuse this value (WeChat out_refund_no /
// PayPal PayPal-Request-Id).
func refundOutRefundNo(appID, ledgerID int64) string {
	return fmt.Sprintf("rf%d_%d", appID, ledgerID)
}

func parseRefundProviderProgress(raw string) map[string]map[string]interface{} {
	out := map[string]map[string]interface{}{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	var list []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return out
	}
	for _, item := range list {
		lid := strings.TrimSpace(fmt.Sprint(item["ledger_id"]))
		if lid == "" || lid == "<nil>" {
			continue
		}
		out[lid] = item
	}
	return out
}

// persistRefundProviderProgress commits channel refund refs outside the approve txn so a
// later local commit failure does not lose idempotency metadata for retry.
func persistRefundProviderProgress(ctx context.Context, appID int64, refs []map[string]interface{}) error {
	raw, err := json.Marshal(refs)
	if err != nil {
		return err
	}
	now := utcNow()
	_, err = db.ExecContext(ctx, `
		UPDATE billing_refund_application
		SET payment_refund_refs = ?, updated_at = ?
		WHERE id = ? AND status = ?`,
		string(raw), now, appID, refundStatusPending,
	)
	return err
}

func ledgerIDKey(id int64) string {
	return formatID(id)
}
