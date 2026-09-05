package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"tracelog"
)

const maxWechatAccountQueryRunes = 128

func normalizeWechatAccountQuery(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if utf8.RuneCountInString(s) > maxWechatAccountQueryRunes {
		return "", fmt.Errorf("微信关联账号过长")
	}
	return s, nil
}

func wechatAccountUserIDsToInt64(ids []string) []int64 {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		n, err := parseIDField(id)
		if err != nil || n <= 0 {
			continue
		}
		out = append(out, n)
	}
	return out
}

var resolveWechatLinkedUserIDs = lookupWechatLinkedUserIDsLive

func lookupWechatLinkedUserIDsLive(ctx context.Context, q string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	path := "/api/internal/users/wechat-linked-account/?q=" + url.QueryEscape(q)
	start := time.Now()
	status, raw, err := taskAuthRequest(ctx, http.MethodGet, path, nil)
	elapsed := time.Since(start).Milliseconds()
	slog.InfoContext(ctx, "wechat_linked_account_auth_lookup",
		"status", status,
		"duration_ms", elapsed,
		"query_len", utf8.RuneCountInString(q),
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("taskAuth wechat-linked-account status %d", status)
	}
	var parsed struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("taskAuth wechat-linked-account decode")
	}
	if parsed.UserIDs == nil {
		return []string{}, nil
	}
	return parsed.UserIDs, nil
}

func listOrdersByWechatAccount(userIDs []int64, payIdentity, statusFilter string, limit, offset int) ([]*ResourceOrder, int64, error) {
	conds := []string{"pay_openid = ?", "pay_unionid = ?"}
	args := []interface{}{payIdentity, payIdentity}
	if len(userIDs) > 0 {
		placeholders := make([]string, len(userIDs))
		for i, id := range userIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		conds = append(conds, "user_id IN ("+strings.Join(placeholders, ",")+")")
	}
	where := "(" + strings.Join(conds, " OR ") + ")"
	if statusFilter != "" {
		where += " AND status = ?"
		args = append(args, statusFilter)
	}

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
	return orders, total, rows.Err()
}

func logOrderWechatAccountQuery(ctx context.Context, total int64, queryLen int) {
	slog.InfoContext(ctx, "admin_order_wechat_account_query",
		"level", "info",
		"query_len", queryLen,
		"total", total,
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
}
