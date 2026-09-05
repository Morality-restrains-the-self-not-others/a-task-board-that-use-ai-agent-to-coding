package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

func attachTransactionDisplayFields(ctx context.Context, tenantID int64, list []map[string]interface{}) {
	if len(list) == 0 {
		return
	}
	for _, row := range list {
		src, _ := row["points_source_type"].(string)
		row["points_source_type_display"] = pointsSourceTypeDisplay(src)
	}
	enrichAdminGrantResourceChanges(ctx, tenantID, list)
	attachQuotaConsumptionChangeDisplay(list)
	attachMoneyChangeDisplay(list)
}

func isGenericAdminGrantDescription(desc string) bool {
	d := strings.TrimSpace(desc)
	if d == "" || d == "管理员后台赠送资源" {
		return true
	}
	return strings.HasPrefix(d, "管理员后台赠送资源:")
}

func enrichAdminGrantResourceChanges(ctx context.Context, tenantID int64, list []map[string]interface{}) {
	minT, maxT := "", ""
	grantRows := 0
	var fkIDs []int64
	for _, row := range list {
		src, _ := row["points_source_type"].(string)
		if src != "admin_grant" {
			continue
		}
		grantRows++
		sec := createdAtSecond(fmt.Sprint(row["created_at"]))
		if sec == "" {
			continue
		}
		if minT == "" || sec < minT {
			minT = sec
		}
		if maxT == "" || sec > maxT {
			maxT = sec
		}
		if v, err := parseIDField(fmt.Sprint(row["id"])); err == nil && v > 0 {
			fkIDs = append(fkIDs, v)
		}
	}
	if grantRows == 0 || minT == "" {
		return
	}

	// 优先按稳定外键 billing_transaction_id 关联；无 FK 的行保留秒级兜底。
	byTxnID := map[int64][]resourceChange{}
	if len(fkIDs) > 0 {
		rows, err := db.Query(`
			SELECT billing_transaction_id, resource_type, quantity
			FROM billing_resource_grant
			WHERE billing_transaction_id IN (`+placeholders(len(fkIDs))+`)
			ORDER BY id ASC`, anySlice(fkIDs)...)
		if err != nil {
			slog.WarnContext(ctx, "billing_txn_grant_enrich_fk_failed",
				"level", "warn",
				"tenant_id", formatID(tenantID),
				"error", err.Error(),
			)
		} else {
			for rows.Next() {
				var txnID int64
				var resourceType string
				var quantity int64
				if err := rows.Scan(&txnID, &resourceType, &quantity); err != nil {
					continue
				}
				byTxnID[txnID] = append(byTxnID[txnID], resourceChange{
					ResourceType: resourceType,
					Quantity:     quantity,
					Label:        resourceTypeLabelZH(resourceType),
					Display:      formatResourceChangeLine(resourceType, quantity, ""),
				})
			}
			rows.Close()
		}
	}

	// 秒级兜底：只匹配尚未建立 FK 的旧 grant，避免把已精确归属的行再串到其它流水。
	bySecond := map[string][]resourceChange{}
	rows, err := db.Query(`
		SELECT created_at, resource_type, quantity
		FROM billing_resource_grant
		WHERE tenant_id = ?
		  AND billing_transaction_id IS NULL
		  AND created_at >= ?
		  AND created_at < DATE_ADD(?, INTERVAL 1 SECOND)
		ORDER BY created_at ASC, id ASC`,
		tenantID, minT, maxT)
	if err != nil {
		slog.WarnContext(ctx, "billing_txn_grant_enrich_failed",
			"level", "warn",
			"tenant_id", formatID(tenantID),
			"error", err.Error(),
		)
	} else {
		defer rows.Close()
		for rows.Next() {
			var created, resourceType string
			var quantity int64
			if err := rows.Scan(&created, &resourceType, &quantity); err != nil {
				continue
			}
			sec := createdAtSecond(created)
			bySecond[sec] = append(bySecond[sec], resourceChange{
				ResourceType: resourceType,
				Quantity:     quantity,
				Label:        resourceTypeLabelZH(resourceType),
				Display:      formatResourceChangeLine(resourceType, quantity, ""),
			})
		}
		if err := rows.Err(); err != nil {
			slog.WarnContext(ctx, "billing_txn_grant_enrich_scan_failed",
				"level", "warn",
				"tenant_id", formatID(tenantID),
				"error", err.Error(),
			)
		}
	}

	enriched := 0
	for _, row := range list {
		src, _ := row["points_source_type"].(string)
		if src != "admin_grant" {
			continue
		}
		var changes []resourceChange
		if v, err := parseIDField(fmt.Sprint(row["id"])); err == nil && v > 0 {
			changes = byTxnID[v]
		}
		if len(changes) == 0 {
			changes = bySecond[createdAtSecond(fmt.Sprint(row["created_at"]))]
		}
		if len(changes) == 0 {
			continue
		}
		payload := make([]map[string]interface{}, 0, len(changes))
		for _, c := range changes {
			payload = append(payload, resourceChangeJSON(c))
		}
		row["resource_changes"] = payload
		display := joinResourceChangeDisplays(changes)
		row["change_display"] = display
		desc, _ := row["description"].(string)
		if isGenericAdminGrantDescription(desc) {
			row["description"] = "管理员后台赠送：" + display
		}
		enriched++
	}
	if enriched > 0 {
		slog.InfoContext(ctx, "billing_txn_grant_display_enriched",
			"level", "info",
			"tenant_id", formatID(tenantID),
			"enriched_rows", enriched,
		)
	}
}

func placeholders(n int) string {
	if n <= 0 {
		return "NULL"
	}
	out := make([]string, n)
	for i := range out {
		out[i] = "?"
	}
	return strings.Join(out, ",")
}

func anySlice(v []int64) []interface{} {
	out := make([]interface{}, len(v))
	for i, x := range v {
		out[i] = x
	}
	return out
}

func attachQuotaConsumptionChangeDisplay(list []map[string]interface{}) {
	for _, row := range list {
		if _, exists := row["change_display"]; exists {
			continue
		}
		amount := jsonNumberAsInt64(row["amount_points"])
		if amount != 0 {
			continue
		}
		usage := jsonNumberAsFloat64(row["usage_amount"])
		if usage == 0 {
			continue
		}
		unitName := ""
		if bu, ok := row["billing_unit"].(map[string]interface{}); ok {
			unitName, _ = bu["name"].(string)
		}
		sign := "-"
		if txnType, _ := row["transaction_type"].(string); txnType == "recharge" {
			sign = "+"
		}
		qty := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", usage), "0"), ".")
		if qty == "" {
			qty = "0"
		}
		if unitName != "" {
			row["change_display"] = sign + qty + " " + unitName
			continue
		}
		row["change_display"] = sign + qty + " 配额"
	}
}

func attachMoneyChangeDisplay(list []map[string]interface{}) {
	for _, row := range list {
		if _, exists := row["change_display"]; exists {
			continue
		}
		amount := jsonNumberAsInt64(row["amount_points"])
		if amount == 0 {
			continue
		}
		sign := "-"
		if txnType, _ := row["transaction_type"].(string); txnType == "recharge" {
			sign = "+"
		}
		row["change_display"] = sign + centsToYuanStr(amount) + " 元"
	}
}

func jsonNumberAsInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func jsonNumberAsFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}
