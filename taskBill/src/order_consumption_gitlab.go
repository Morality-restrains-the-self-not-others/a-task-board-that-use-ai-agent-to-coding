package main

import (
	"log/slog"
	"strings"
)

func attachGitlabOrderConsumption(rc map[string]interface{}, orderID, tenantID int64, items []ResourceOrderItem, status string) {
	if status != OrderStatusPaid && status != OrderStatusRefunded {
		return
	}
	attachGitlabKind(rc, orderID, tenantID, items, status, ResourceTypeGitlabDisk)
	attachGitlabKind(rc, orderID, tenantID, items, status, ResourceTypeGitlabTraffic)
}

func attachGitlabKind(rc map[string]interface{}, orderID, tenantID int64, items []ResourceOrderItem, status, resourceType string) {
	granted, region := sumOrderItemQuantity(items, resourceType)
	if granted <= 0 {
		return
	}
	remaining := float64(0)
	consumed := float64(granted)
	if status == OrderStatusPaid {
		rem, err := gitlabOrderRemaining(tenantID, orderID, region, resourceType, granted)
		if err != nil {
			slog.Warn("order_gitlab_consumption_attach_failed",
				"level", "warn",
				"order_id", formatID(orderID),
				"tenant_id", formatID(tenantID),
				"resource_type", resourceType,
				"error", err.Error(),
			)
			remaining = float64(granted)
			consumed = 0
		} else {
			remaining = rem
			consumed = float64(granted) - remaining
			if consumed < 0 {
				consumed = 0
			}
		}
	}
	row := map[string]interface{}{
		"source_kind": grantSourcePurchase,
		"granted":     granted,
		"consumed":    consumed,
		"remaining":   remaining,
		"unit":        "GB",
		"events":      []interface{}{},
	}
	if region != "" {
		row["region"] = region
	}
	rc[resourceType] = row
}

func sumOrderItemQuantity(items []ResourceOrderItem, resourceType string) (qty int64, region string) {
	for _, it := range items {
		if it.ResourceType != resourceType {
			continue
		}
		qty += it.Quantity
		if region == "" {
			region = strings.TrimSpace(it.Region)
		}
	}
	return qty, region
}

func gitlabOrderRemaining(tenantID, orderID int64, region, resourceType string, thisGranted int64) (float64, error) {
	if region == "" {
		return float64(thisGranted), nil
	}
	snap, err := getTenantGitlabResourceByRegion(tenantID, region)
	if err != nil {
		return 0, err
	}
	var quota, used float64
	switch resourceType {
	case ResourceTypeGitlabDisk:
		quota = float64(snap.DiskGB)
		used = diskUsedGBFromBytes(snap.DiskUsedBytes)
	case ResourceTypeGitlabTraffic:
		quota = float64(snap.TrafficPrepaidGB)
		used = snap.TrafficUsedGB
	default:
		return float64(thisGranted), nil
	}
	pool := quota - used
	if pool < 0 {
		pool = 0
	}
	rows, err := db.Query(`
		SELECT o.id, SUM(i.quantity)
		FROM billing_resource_order o
		JOIN billing_resource_order_item i ON i.order_id = o.id
		WHERE o.tenant_id = ? AND o.status = ?
		  AND i.resource_type = ? AND i.region = ?
		GROUP BY o.id
		ORDER BY MAX(COALESCE(o.paid_at, o.created_at)) DESC, o.id DESC`,
		tenantID, OrderStatusPaid, resourceType, region,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	found := false
	remaining := float64(0)
	for rows.Next() {
		var oid, quantity int64
		if err := rows.Scan(&oid, &quantity); err != nil {
			return 0, err
		}
		take := float64(quantity)
		if take > pool {
			take = pool
		}
		pool -= take
		if oid == orderID {
			found = true
			remaining = take
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}
	if remaining > float64(thisGranted) {
		remaining = float64(thisGranted)
	}
	return remaining, nil
}
