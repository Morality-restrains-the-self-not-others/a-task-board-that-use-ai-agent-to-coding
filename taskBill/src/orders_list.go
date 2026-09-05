package main

// listTenantOrders 列出租户的订单，statusFilter 为空时不过滤状态
func listTenantOrders(tenantID int64, limit, offset int, statusFilter string) ([]*ResourceOrder, int64, error) {
	var total int64
	var countQuery string
	var countArgs []interface{}
	if statusFilter != "" {
		countQuery = `SELECT COUNT(*) FROM billing_resource_order WHERE tenant_id = ? AND status = ?`
		countArgs = []interface{}{tenantID, statusFilter}
	} else {
		countQuery = `SELECT COUNT(*) FROM billing_resource_order WHERE tenant_id = ?`
		countArgs = []interface{}{tenantID}
	}
	if err := db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	var query string
	var args []interface{}
	if statusFilter != "" {
		query = `SELECT id, tenant_id, order_number, status, total_yuan_cents,
		       COALESCE(payment_method,''), COALESCE(payment_ref,''), created_at, paid_at, cancelled_at
		       FROM billing_resource_order WHERE tenant_id = ? AND status = ?
		       ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
		args = []interface{}{tenantID, statusFilter, limit, offset}
	} else {
		query = `SELECT id, tenant_id, order_number, status, total_yuan_cents,
		       COALESCE(payment_method,''), COALESCE(payment_ref,''), created_at, paid_at, cancelled_at
		       FROM billing_resource_order WHERE tenant_id = ?
		       ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
		args = []interface{}{tenantID, limit, offset}
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*ResourceOrder
	for rows.Next() {
		var o ResourceOrder
		if err := rows.Scan(&o.ID, &o.TenantID, &o.OrderNumber, &o.Status, &o.TotalYuanCents,
			&o.PaymentMethod, &o.PaymentRef, &o.CreatedAt, &o.PaidAt, &o.CancelledAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, &o)
	}
	return orders, total, nil
}

// listAllOrders 列出所有租户的订单（系统管理员用），statusFilter 为空时不过滤状态
func listAllOrders(limit, offset int, statusFilter string) ([]*ResourceOrder, int64, error) {
	var total int64
	var countQuery string
	var countArgs []interface{}
	if statusFilter != "" {
		countQuery = `SELECT COUNT(*) FROM billing_resource_order WHERE status = ?`
		countArgs = []interface{}{statusFilter}
	} else {
		countQuery = `SELECT COUNT(*) FROM billing_resource_order`
	}
	if err := db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	var query string
	var args []interface{}
	if statusFilter != "" {
		query = `SELECT id, tenant_id, order_number, status, total_yuan_cents,
		       COALESCE(payment_method,''), COALESCE(payment_ref,''), created_at, paid_at, cancelled_at
		       FROM billing_resource_order WHERE status = ?
		       ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
		args = []interface{}{statusFilter, limit, offset}
	} else {
		query = `SELECT id, tenant_id, order_number, status, total_yuan_cents,
		       COALESCE(payment_method,''), COALESCE(payment_ref,''), created_at, paid_at, cancelled_at
		       FROM billing_resource_order
		       ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
		args = []interface{}{limit, offset}
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*ResourceOrder
	for rows.Next() {
		var o ResourceOrder
		if err := rows.Scan(&o.ID, &o.TenantID, &o.OrderNumber, &o.Status, &o.TotalYuanCents,
			&o.PaymentMethod, &o.PaymentRef, &o.CreatedAt, &o.PaidAt, &o.CancelledAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, &o)
	}
	return orders, total, nil
}
