package main

import (
	"database/sql"
	"fmt"
)

func loadOrderRow(query string, args ...any) (*ResourceOrder, error) {
	var o ResourceOrder
	err := db.QueryRow(query, args...).Scan(
		&o.ID, &o.TenantID, &o.OrderNumber, &o.Status, &o.TotalYuanCents,
		&o.PaymentMethod, &o.PaymentRef, &o.OutTradeNo, &o.WechatTransactionID,
		&o.UserID, &o.CreatedAt, &o.PaidAt, &o.CancelledAt, &o.BuyerNote,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("订单不存在")
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func loadOrderItems(orderID int64) ([]ResourceOrderItem, error) {
	rows, err := db.Query(`
		SELECT id, resource_type, quantity, unit_price_yuan_cents, subtotal_yuan_cents, COALESCE(region,''), disk_months, created_at
		FROM billing_resource_order_item WHERE order_id = ? ORDER BY id`, orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ResourceOrderItem
	for rows.Next() {
		var item ResourceOrderItem
		if err := rows.Scan(&item.ID, &item.ResourceType, &item.Quantity,
			&item.UnitPriceYuanCents, &item.SubtotalYuanCents, &item.Region, &item.DiskMonths, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.OrderID = orderID
		items = append(items, item)
	}
	return items, rows.Err()
}

const orderSelectCols = `SELECT id, tenant_id, order_number, status, total_yuan_cents,
		       COALESCE(payment_method,''), COALESCE(payment_ref,''), COALESCE(out_trade_no,''), COALESCE(wechat_transaction_id,''),
		       COALESCE(user_id,''), created_at, paid_at, cancelled_at, COALESCE(buyer_note,'')
		FROM billing_resource_order`

// loadOrder 按租户分片键 + 主键加载（ADR-0018）。
func loadOrder(tenantID, orderID int64) (*ResourceOrder, []ResourceOrderItem, error) {
	if tenantID <= 0 || orderID <= 0 {
		return nil, nil, fmt.Errorf("订单不存在")
	}
	o, err := loadOrderRow(orderSelectCols+` WHERE tenant_id = ? AND id = ?`, tenantID, orderID)
	if err != nil {
		return nil, nil, err
	}
	items, err := loadOrderItems(orderID)
	if err != nil {
		return nil, nil, err
	}
	return o, items, nil
}

// loadOrderByID 仅主键加载：管理端 / 支付回调在尚无租户谓词时使用。
// 分片后须改为定位表；不得用于租户面 IDOR 边界。
func loadOrderByID(orderID int64) (*ResourceOrder, []ResourceOrderItem, error) {
	if orderID <= 0 {
		return nil, nil, fmt.Errorf("订单不存在")
	}
	o, err := loadOrderRow(orderSelectCols+` WHERE id = ?`, orderID)
	if err != nil {
		return nil, nil, err
	}
	items, err := loadOrderItems(orderID)
	if err != nil {
		return nil, nil, err
	}
	return o, items, nil
}
