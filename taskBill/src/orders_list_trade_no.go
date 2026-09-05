package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"tracelog"
)

const maxTradeNoQueryRunes = 128

func normalizeTradeNoQuery(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "`'\"")
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if utf8.RuneCountInString(s) > maxTradeNoQueryRunes {
		return "", fmt.Errorf("交易单号过长")
	}
	return s, nil
}

func listOrdersByTradeNo(tenantID int64, q, statusFilter string, limit, offset int) ([]*ResourceOrder, int64, error) {
	orders, total, err := listOrdersByTradeNoSQL(tenantID, q, statusFilter, limit, offset)
	if err != nil || total > 0 || tenantID != 0 {
		return orders, total, err
	}
	if !looksLikeWechatTxnID(q) {
		return orders, total, err
	}
	outNo, qerr := resolveWechatTxnToOutTradeNo(q)
	if qerr != nil {
		slog.Warn("wechat_txn_resolve_failed",
			"level", "warn",
			"query_len", utf8.RuneCountInString(q),
			"err", qerr.Error(),
		)
		return orders, total, nil
	}
	outNo = strings.TrimSpace(outNo)
	if outNo == "" || outNo == q {
		return orders, total, nil
	}
	orders2, total2, err2 := listOrdersByTradeNoSQL(tenantID, outNo, statusFilter, limit, offset)
	if err2 != nil || total2 == 0 {
		return orders2, total2, err2
	}
	persistWechatPayVouchers(orders2[0].ID, outNo, q)
	return orders2, total2, nil
}

// listOrdersByTradeNoSQL 按交易单号等值查询。tenantID==0 表示跨租户（仅管理端调用）。
func listOrdersByTradeNoSQL(tenantID int64, q, statusFilter string, limit, offset int) ([]*ResourceOrder, int64, error) {
	where, args := tradeNoWhere(tenantID, q, statusFilter)

	var total int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_resource_order WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, tenant_id, order_number, status, total_yuan_cents,
		COALESCE(payment_method,''), COALESCE(payment_ref,''), created_at, paid_at, cancelled_at
		FROM billing_resource_order WHERE ` + where + `
		ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]interface{}{}, args...), limit, offset)
	rows, err := db.Query(query, queryArgs...)
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

func tradeNoWhere(tenantID int64, q, statusFilter string) (string, []interface{}) {
	plain := strings.TrimPrefix(q, "wechat:")
	prefixed := q
	if !strings.HasPrefix(q, "wechat:") {
		prefixed = "wechat:" + q
	}
	conds := []string{
		"order_number = ?",
		"payment_ref = ?",
		"payment_ref = ?",
		"out_trade_no = ?",
		"wechat_transaction_id = ?",
	}
	args := []interface{}{q, q, prefixed, plain, q}
	if id, err := parseIDField(q); err == nil && id > 0 {
		conds = append(conds, "id = ?")
		args = append(args, id)
	}
	where := "(" + strings.Join(conds, " OR ") + ")"
	if tenantID > 0 {
		where += " AND tenant_id = ?"
		args = append(args, tenantID)
	}
	if statusFilter != "" {
		where += " AND status = ?"
		args = append(args, statusFilter)
	}
	return where, args
}

func logOrderTradeNoQuery(ctx context.Context, source string, tenantID, total int64, q string) {
	slog.InfoContext(ctx, "order_trade_no_query",
		"level", "info",
		"source", source,
		"query_len", utf8.RuneCountInString(q),
		"tenant_id", formatID(tenantID),
		"total", total,
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
}
