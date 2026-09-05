package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"tracelog"
)

const userRechargeConsumptionAllUsersSQL = `
SELECT DISTINCT user_id FROM (
	SELECT user_id FROM billing_transaction
	WHERE user_id IS NOT NULL AND user_id != ''
	  AND (
	    (transaction_type = 'recharge' AND COALESCE(points_source_type, '') != 'admin_grant')
	    OR transaction_type = 'consumption'
	  )
	UNION
	SELECT bt.user_id FROM billing_payment_ledger pl
	INNER JOIN billing_transaction bt ON bt.id = pl.billing_transaction_id
	WHERE bt.user_id IS NOT NULL AND bt.user_id != ''
) AS t`

func parseLimitOffset(qp map[string][]string) (limit, offset int) {
	limit = 50
	offset = 0
	if v := strings.TrimSpace(firstQueryValue(qp, "limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}
	if v := strings.TrimSpace(firstQueryValue(qp, "offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func firstQueryValue(qp map[string][]string, key string) string {
	if vals, ok := qp[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func parseOptionalUserIDsFilter(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func buildUserRechargeConsumptionRow(userID string, recharge, consumed, ledgerRemaining, ledgerCount int64) map[string]interface{} {
	var unconsumed int64
	if ledgerCount > 0 {
		unconsumed = ledgerRemaining
	} else {
		unconsumed = recharge - consumed
		if unconsumed < 0 {
			unconsumed = 0
		}
	}
	return map[string]interface{}{
		"user_id":               userID,
		"recharge_points":       recharge,
		"recharge_amount_yuan":  pointsToYuanEquivalentStr(recharge),
		"unconsumed_points":     unconsumed,
		"consumed_points":       consumed,
		"commissionable_points": consumed,
	}
}

func listUserRechargeConsumption(_ context.Context, filterUserIDs []string, limit, offset int) ([]map[string]interface{}, int, error) {
	allUsersCTE := userRechargeConsumptionAllUsersSQL
	countArgs := make([]interface{}, 0)
	countQ := `WITH all_users AS (` + allUsersCTE + `) SELECT COUNT(*) FROM all_users`
	if len(filterUserIDs) > 0 {
		placeholders := strings.Repeat("?,", len(filterUserIDs))
		placeholders = strings.TrimSuffix(placeholders, ",")
		countQ += ` WHERE user_id IN (` + placeholders + `)`
		for _, uid := range filterUserIDs {
			countArgs = append(countArgs, uid)
		}
	}

	var total int
	if err := db.QueryRow(countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataArgs := make([]interface{}, 0, len(filterUserIDs)+2)
	dataQ := `
WITH all_users AS (` + allUsersCTE + `),
filtered_users AS (
	SELECT user_id FROM all_users`
	if len(filterUserIDs) > 0 {
		placeholders := strings.Repeat("?,", len(filterUserIDs))
		placeholders = strings.TrimSuffix(placeholders, ",")
		dataQ += ` WHERE user_id IN (` + placeholders + `)`
		for _, uid := range filterUserIDs {
			dataArgs = append(dataArgs, uid)
		}
	}
	dataQ += `
),
page_users AS (
	SELECT user_id FROM filtered_users ORDER BY user_id LIMIT ? OFFSET ?
)
SELECT
	pu.user_id,
	COALESCE(r.recharge_points, 0),
	COALESCE(c.consumed_points, 0),
	COALESCE(l.ledger_remaining, 0),
	COALESCE(l.ledger_count, 0)
FROM page_users pu
LEFT JOIN (
	SELECT user_id, SUM(amount) AS recharge_points
	FROM billing_transaction
	WHERE transaction_type = 'recharge'
	  AND user_id IS NOT NULL AND user_id != ''
	  AND COALESCE(points_source_type, '') != 'admin_grant'
	GROUP BY user_id
) r ON r.user_id = pu.user_id
LEFT JOIN (
	SELECT user_id, SUM(amount) AS consumed_points
	FROM billing_transaction
	WHERE transaction_type = 'consumption'
	  AND user_id IS NOT NULL AND user_id != ''
	GROUP BY user_id
) c ON c.user_id = pu.user_id
LEFT JOIN (
	SELECT bt.user_id, SUM(pl.remaining_points) AS ledger_remaining, COUNT(pl.id) AS ledger_count
	FROM billing_payment_ledger pl
	INNER JOIN billing_transaction bt ON bt.id = pl.billing_transaction_id
	WHERE bt.user_id IS NOT NULL AND bt.user_id != ''
	GROUP BY bt.user_id
) l ON l.user_id = pu.user_id
ORDER BY pu.user_id`
	dataArgs = append(dataArgs, limit, offset)

	rows, err := db.Query(dataQ, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var userID string
		var recharge, consumed, ledgerRemaining, ledgerCount int64
		if err := rows.Scan(&userID, &recharge, &consumed, &ledgerRemaining, &ledgerCount); err != nil {
			return nil, 0, err
		}
		results = append(results, buildUserRechargeConsumptionRow(userID, recharge, consumed, ledgerRemaining, ledgerCount))
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if results == nil {
		results = []map[string]interface{}{}
	}
	return results, total, nil
}

// isNumericIDList reports whether s is a comma-separated list of numeric user
// IDs (snowflake IDs) — i.e. an exact-ID query rather than an attribute search.
func isNumericIDList(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && r != ',' && r != ' ' {
			return false
		}
	}
	return strings.TrimSpace(s) != ""
}

// appendUniqueUserIDs appends src entries that are not already in dst.
func appendUniqueUserIDs(dst, src []string) []string {
	if len(src) == 0 {
		return dst
	}
	seen := make(map[string]struct{}, len(dst)+len(src))
	for _, id := range dst {
		seen[id] = struct{}{}
	}
	out := append([]string{}, dst...)
	for _, id := range src {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// resolveUserSearchQueryIDs resolves a fuzzy q (email/username/phone) into exact
// user IDs by delegating to taskAuth's internal user search. ok=false means the
// search service was unreachable — callers must not fall back to unfiltered data.
func resolveUserSearchQueryIDs(ctx context.Context, q string) ([]string, bool) {
	statusCode, raw, err := taskAuthRequest(ctx, http.MethodGet,
		"/api/internal/users/?q="+url.QueryEscape(strings.TrimSpace(q))+"&limit=200", nil)
	if err != nil {
		slog.ErrorContext(ctx, "resolveUserSearchQueryIDs: taskAuth request failed",
			"error", err.Error(), "q", q)
		return nil, false
	}
	if statusCode != http.StatusOK {
		slog.ErrorContext(ctx, "resolveUserSearchQueryIDs: taskAuth returned non-200",
			"status", statusCode, "q", q)
		return nil, false
	}
	var resp struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		slog.ErrorContext(ctx, "resolveUserSearchQueryIDs: decode failed",
			"error", err.Error(), "q", q)
		return nil, false
	}
	ids := make([]string, 0, len(resp.Users))
	for _, u := range resp.Users {
		if u.ID != "" {
			ids = append(ids, u.ID)
		}
	}
	return ids, true
}

// enrichUserRechargeConsumptionRows fills email/username on each row by batch
// resolving user IDs against taskAuth. Failures degrade gracefully (fields stay
// empty) so billing data is never blocked by an upstream hiccup.
func enrichUserRechargeConsumptionRows(ctx context.Context, results []map[string]interface{}) {
	if len(results) == 0 {
		return
	}
	rawIDs := make([]interface{}, 0, len(results))
	for _, row := range results {
		if id, ok := row["user_id"].(string); ok && id != "" {
			rawIDs = append(rawIDs, id)
		}
	}
	if len(rawIDs) == 0 {
		return
	}
	statusCode, raw, err := taskAuthRequest(ctx, http.MethodPost,
		"/api/internal/users/batch/details/", map[string]interface{}{"user_ids": rawIDs})
	if err != nil {
		slog.ErrorContext(ctx, "enrichUserRechargeConsumptionRows: taskAuth request failed",
			"error", err.Error(), "user_ids", len(rawIDs))
		return
	}
	if statusCode != http.StatusOK {
		slog.ErrorContext(ctx, "enrichUserRechargeConsumptionRows: taskAuth returned non-200",
			"status", statusCode)
		return
	}
	var resp struct {
		Results map[string]struct {
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		slog.ErrorContext(ctx, "enrichUserRechargeConsumptionRows: decode failed",
			"error", err.Error())
		return
	}
	for _, row := range results {
		uid, _ := row["user_id"].(string)
		info, ok := resp.Results[uid]
		if !ok {
			continue
		}
		row["email"] = info.Email
		row["username"] = info.Username
	}
}

func handleInternalUserRechargeConsumption(w http.ResponseWriter, r *http.Request) {
	if !requireInternalOrGatewayAuth(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	qp := r.URL.Query()
	limit, offset := parseLimitOffset(qp)
	filterUserIDs := parseOptionalUserIDsFilter(qp.Get("user_ids"))

	// q 支持按 邮箱/用户名/手机号/用户 ID 模糊搜索：纯数字视为精确用户 ID
	// 列表（兼容旧前端「多个用逗号分隔」行为），其余委托 taskAuth 解析。
	if searchQ := strings.TrimSpace(qp.Get("q")); searchQ != "" {
		if isNumericIDList(searchQ) {
			filterUserIDs = appendUniqueUserIDs(filterUserIDs, parseOptionalUserIDsFilter(searchQ))
		} else {
			matchedIDs, ok := resolveUserSearchQueryIDs(r.Context(), searchQ)
			if !ok {
				writeErrorJSON(w, http.StatusBadGateway, "用户搜索服务暂不可用，请稍后重试",
					tracelog.TraceIDFromContext(r.Context()))
				return
			}
			if len(matchedIDs) == 0 && len(filterUserIDs) == 0 {
				// 搜索无匹配 → 返回空结果，避免退化为全量
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"results": []map[string]interface{}{},
					"total":   0,
					"limit":   limit,
					"offset":  offset,
				})
				return
			}
			filterUserIDs = appendUniqueUserIDs(filterUserIDs, matchedIDs)
		}
	}

	results, total, err := listUserRechargeConsumption(r.Context(), filterUserIDs, limit, offset)
	if err != nil {
		slog.ErrorContext(r.Context(), "listUserRechargeConsumption failed",
			"error", err.Error(),
			"path", r.URL.Path,
		)
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	enrichUserRechargeConsumptionRows(r.Context(), results)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// scanUserRechargeConsumptionRow supports tests that query aggregates directly.
func scanUserRechargeConsumptionRow(userID string) (map[string]interface{}, error) {
	var recharge, consumed sql.NullInt64
	var ledgerRemaining, ledgerCount sql.NullInt64

	err := db.QueryRow(`
		SELECT
			(SELECT COALESCE(SUM(amount), 0) FROM billing_transaction
			 WHERE user_id = ? AND transaction_type = 'recharge'
			   AND COALESCE(points_source_type, '') != 'admin_grant'),
			(SELECT COALESCE(SUM(amount), 0) FROM billing_transaction
			 WHERE user_id = ? AND transaction_type = 'consumption'),
			(SELECT COALESCE(SUM(pl.remaining_points), 0) FROM billing_payment_ledger pl
			 INNER JOIN billing_transaction bt ON bt.id = pl.billing_transaction_id
			 WHERE bt.user_id = ?),
			(SELECT COUNT(pl.id) FROM billing_payment_ledger pl
			 INNER JOIN billing_transaction bt ON bt.id = pl.billing_transaction_id
			 WHERE bt.user_id = ?)`,
		userID, userID, userID, userID,
	).Scan(&recharge, &consumed, &ledgerRemaining, &ledgerCount)
	if err != nil {
		return nil, fmt.Errorf("scan user %s: %w", userID, err)
	}
	return buildUserRechargeConsumptionRow(
		userID,
		nullInt64OrZero(recharge),
		nullInt64OrZero(consumed),
		nullInt64OrZero(ledgerRemaining),
		nullInt64OrZero(ledgerCount),
	), nil
}
