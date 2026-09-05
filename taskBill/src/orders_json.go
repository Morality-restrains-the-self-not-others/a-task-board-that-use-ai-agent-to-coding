package main

import (
	"database/sql"
	"fmt"
)

func orderJSON(o *ResourceOrder, items []ResourceOrderItem) map[string]interface{} {
	itemList := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		row := map[string]interface{}{
			"id":              formatID(it.ID),
			"resource_type":   it.ResourceType,
			"quantity":        it.Quantity,
			"unit_price_yuan": centsToYuanStr(it.UnitPriceYuanCents),
			"subtotal_yuan":   centsToYuanStr(it.SubtotalYuanCents),
		}
		if it.Region != "" {
			row["region"] = it.Region
		}
		if it.ResourceType == ResourceTypeGitlabDisk {
			row["disk_months"] = it.DiskMonths
		}
		itemList = append(itemList, row)
	}

	m := map[string]interface{}{
		"id":                    formatID(o.ID),
		"order_number":          o.OrderNumber,
		"status":                o.Status,
		"total_yuan":            centsToYuanStr(o.TotalYuanCents),
		"total_yuan_cents":      o.TotalYuanCents,
		"payment_method":        o.PaymentMethod,
		"payment_ref":           o.PaymentRef,
		"out_trade_no":          o.OutTradeNo,
		"wechat_transaction_id": o.WechatTransactionID,
		"items":                 itemList,
		"created_at":            o.CreatedAt,
		"buyer_note":            o.BuyerNote,
	}
	if o.PaidAt.Valid {
		m["paid_at"] = o.PaidAt.String
	}
	if o.CancelledAt.Valid {
		m["cancelled_at"] = o.CancelledAt.String
	}
	attachOrderResourceConsumption(m, o.ID, o.TenantID, items)
	attachOrderInvoices(m, o.ID)
	return m
}

// centsToYuanStr 分转元字符串
func centsToYuanStr(cents int64) string {
	yuan := float64(cents) / 100.0
	return fmt.Sprintf("%.2f", yuan)
}

// tenantHasGitlabDisk 检查租户是否已购买 GitLab 磁盘（disk_gb > 0）
func tenantHasGitlabDisk(tenantID int64) (bool, error) {
	var diskGB sql.NullInt64
	err := db.QueryRow(`
		SELECT disk_gb FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND disk_gb > 0 LIMIT 1`,
		tenantID,
	).Scan(&diskGB)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return diskGB.Valid && diskGB.Int64 > 0, nil
}
