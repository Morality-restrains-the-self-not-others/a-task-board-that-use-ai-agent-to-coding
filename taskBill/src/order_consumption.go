package main

import (
	"log/slog"
	"strings"
)

func attachOrderResourceConsumption(m map[string]interface{}, orderID, tenantID int64, items []ResourceOrderItem) {
	rc := map[string]interface{}{}
	attachTaskPostOrderConsumption(rc, orderID, tenantID)
	status, _ := m["status"].(string)
	attachGitlabOrderConsumption(rc, orderID, tenantID, items, status)
	if len(rc) == 0 {
		return
	}
	m["resource_consumption"] = rc
}

func attachTaskPostOrderConsumption(rc map[string]interface{}, orderID, tenantID int64) {
	var granted, remaining int64
	var sourceKind string
	err := db.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0), COALESCE(SUM(remaining), 0),
		       COALESCE(MAX(source_kind), '')
		FROM billing_resource_grant
		WHERE tenant_id = ? AND order_id = ? AND resource_type = ?`,
		tenantID, orderID, ResourceTypeTaskPost,
	).Scan(&granted, &remaining, &sourceKind)
	if err != nil {
		slog.Warn("order_task_post_consumption_attach_failed",
			"level", "warn",
			"order_id", formatID(orderID),
			"tenant_id", formatID(tenantID),
			"error", err.Error(),
		)
		return
	}
	consumed := granted - remaining
	if consumed < 0 {
		consumed = 0
	}
	events := make([]interface{}, 0)
	rows, err := db.Query(`
		SELECT created_at, COALESCE(task_id, ''), COALESCE(transaction_id, ''), COALESCE(usage_amount, 0)
		FROM billing_transaction
		WHERE related_order_id = ? AND points_source_type = 'quota_consumption'
		ORDER BY created_at ASC, id ASC
		LIMIT 100`, orderID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var createdAt, taskID, txnID string
			var qty float64
			if err := rows.Scan(&createdAt, &taskID, &txnID, &qty); err != nil {
				continue
			}
			action := "create"
			if strings.Contains(txnID, "task_post_renewal") {
				action = "renewal"
			}
			q := int64(qty)
			if q < 1 {
				q = 1
			}
			events = append(events, map[string]interface{}{
				"created_at":  createdAt,
				"task_id":     taskID,
				"action":      action,
				"quantity":    q,
				"source_kind": sourceKind,
			})
		}
	}
	if granted == 0 && len(events) == 0 {
		return
	}
	rc[ResourceTypeTaskPost] = map[string]interface{}{
		"source_kind": sourceKind,
		"granted":     granted,
		"consumed":    consumed,
		"remaining":   remaining,
		"events":      events,
	}
}
