package main

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tracelog"
)

func loadBillingUnitMap() map[int64]map[string]interface{} {
	out := map[int64]map[string]interface{}{}
	rows, err := db.Query(`SELECT id, unit_type, name, price, unit, is_active FROM billing_unit`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, price int64
		var unitType, name, unit string
		var active bool
		if err := rows.Scan(&id, &unitType, &name, &price, &unit, &active); err != nil {
			continue
		}
		out[id] = map[string]interface{}{
			"id": formatID(id), "unit_type": unitType, "name": name,
			"price_points": price, "unit": unit, "is_active": active,
		}
	}
	return out
}

func attachBillingUnit(row map[string]interface{}, unitID sql.NullInt64, units map[int64]map[string]interface{}) {
	if !unitID.Valid {
		return
	}
	if u, ok := units[unitID.Int64]; ok {
		row["billing_unit"] = u
		return
	}
	// Unit row missing: still expose id so callers can detect orphan FKs.
	row["billing_unit"] = map[string]interface{}{"id": formatID(unitID.Int64)}
}

// Compatibility aliases for tests / callers expecting main-branch names.
func loadBillingUnitIndex() (map[int64]map[string]interface{}, error) {
	return loadBillingUnitMap(), nil
}

func nestedBillingUnit(unitID int64, index map[int64]map[string]interface{}) map[string]interface{} {
	if unit, ok := index[unitID]; ok {
		return unit
	}
	return map[string]interface{}{"id": formatID(unitID)}
}

func parsePageParams(q url.Values) (page, pageSize int, paged bool) {
	if strings.TrimSpace(q.Get("page")) == "" {
		return 0, 0, false
	}
	page, _ = strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize, true
}

func writeListOrPaged(w http.ResponseWriter, list []map[string]interface{}, page, pageSize int, paged bool, total int) {
	if !paged {
		writeJSON(w, http.StatusOK, list)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results":   list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// voidedResourcePurchasePred 匹配已退款/已取消订单对应的资源购买流水。
// 占位符：o.tenant_id = ?；流水 transaction_id 为 order:<order_number>。
const voidedResourcePurchasePred = `
		bt.points_source_type = 'resource_purchase'
		AND EXISTS (
			SELECT 1 FROM billing_resource_order o
			WHERE o.tenant_id = ?
			  AND (bt.transaction_id = CONCAT('order:', o.order_number) OR bt.transaction_id = o.order_number)
			  AND o.status IN ('refunded', 'cancelled')
		)`

func handleBillingStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	acc, _, err := getOrCreateBillingAccount(tid, false)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	monthStart := strings.TrimSpace(r.URL.Query().Get("month_start"))
	if monthStart == "" {
		now := time.Now()
		monthStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	}

	var totalConsumption, monthlyConsumption, userRecharge sql.NullInt64
	// 消耗不含已退款/已取消订单的 resource_purchase（取消待支付订单本身无流水；退款后订单 status=refunded）。
	err = db.QueryRow(`
		SELECT COALESCE(SUM(bt.amount), 0)
		FROM billing_transaction bt
		WHERE bt.account_id = ? AND bt.transaction_type = 'consumption'
		  AND NOT (`+voidedResourcePurchasePred+`)`, acc.ID, tid).Scan(&totalConsumption)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	err = db.QueryRow(`
		SELECT COALESCE(SUM(bt.amount), 0)
		FROM billing_transaction bt
		WHERE bt.account_id = ? AND bt.transaction_type = 'consumption'
		  AND date(bt.created_at) >= date(?)
		  AND NOT (`+voidedResourcePurchasePred+`)`, acc.ID, monthStart, tid).Scan(&monthlyConsumption)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	// 累计支付：钱包实付充值 + 资源订单实付 − 已完成退款；不含后台赠送。
	// OPT-20260819-041：refund 行 amount 为负，直接并入求和即冲减合计。
	err = db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM billing_transaction
		WHERE account_id = ?
		  AND (
		    (transaction_type = 'recharge' AND points_source_type IN (
		      'user_recharge', 'user_recharge_paypal', 'user_recharge_admin', 'user_recharge_wechat'
		    ))
		    OR (transaction_type = 'consumption' AND points_source_type = 'resource_purchase')
		    OR transaction_type = 'refund'
		  )`,
		acc.ID).Scan(&userRecharge)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	// 退款扣减额（正数），供前端展示「已退款 X」的拆分文案。
	var refundDeduction sql.NullInt64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(-amount), 0) FROM billing_transaction
		WHERE account_id = ? AND transaction_type = 'refund'`, acc.ID).Scan(&refundDeduction)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	totalPts := nullInt64OrZero(totalConsumption)
	monthlyPts := nullInt64OrZero(monthlyConsumption)
	rechargePts := nullInt64OrZero(userRecharge)
	refundPts := nullInt64OrZero(refundDeduction)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"month_start":                monthStart,
		"total_consumption_points":   totalPts,
		"monthly_consumption_points": monthlyPts,
		"user_recharge_points":       rechargePts,
		"refund_deduction_points":    refundPts,
		// cents 别名与 points 同为分，兼容曾误读 *_cents 的前端
		"total_consumption_cents":   totalPts,
		"monthly_consumption_cents": monthlyPts,
		"user_recharge_cents":       rechargePts,
		"refund_deduction_cents":    refundPts,
	})
}

func nullInt64OrZero(v sql.NullInt64) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

func appendTxnListFilters(q string, args []interface{}, qp url.Values, filtered bool) (string, []interface{}) {
	return appendTxnListFiltersWithAlias(q, args, qp, filtered, "")
}

func appendTxnListFiltersPrefixed(q string, args []interface{}, qp url.Values, filtered bool) (string, []interface{}) {
	return appendTxnListFiltersWithAlias(q, args, qp, filtered, "bt.")
}

func appendTxnListFiltersWithAlias(q string, args []interface{}, qp url.Values, filtered bool, alias string) (string, []interface{}) {
	if !filtered {
		return q, args
	}
	col := func(name string) string { return alias + name }
	if v := qp.Get("transaction_type"); v != "" {
		q += ` AND ` + col("transaction_type") + ` = ?`
		args = append(args, v)
	}
	if v := qp.Get("points_source_type"); v != "" {
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			placeholders := strings.Repeat("?,", len(parts))
			placeholders = strings.TrimSuffix(placeholders, ",")
			q += ` AND ` + col("points_source_type") + ` IN (` + placeholders + `)`
			for _, p := range parts {
				args = append(args, strings.TrimSpace(p))
			}
		} else {
			q += ` AND ` + col("points_source_type") + ` = ?`
			args = append(args, v)
		}
	}
	if v := qp.Get("start_date"); v != "" {
		q += ` AND date(` + col("created_at") + `) >= date(?)`
		args = append(args, v)
	}
	if v := qp.Get("end_date"); v != "" {
		q += ` AND date(` + col("created_at") + `) <= date(?)`
		args = append(args, v)
	}
	if v := qp.Get("project_id"); v != "" {
		q += ` AND ` + col("project_id") + ` = ?`
		args = append(args, v)
	}
	if v := qp.Get("user_id"); v != "" {
		q += ` AND ` + col("user_id") + ` = ?`
		args = append(args, v)
	}
	if v := qp.Get("workspace_id"); v != "" {
		q += ` AND ` + col("workspace_id") + ` = ?`
		args = append(args, v)
	}
	if v := qp.Get("task_id"); v != "" {
		q += ` AND ` + col("task_id") + ` = ?`
		args = append(args, v)
	}
	if v := qp.Get("billing_unit_type"); v != "" {
		q += ` AND ` + col("billing_unit_id") + ` IN (SELECT id FROM billing_unit WHERE unit_type = ?)`
		args = append(args, v)
	}
	return q, args
}

func handleTransactionsList(w http.ResponseWriter, r *http.Request, filtered bool) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	acc, _, err := getOrCreateBillingAccount(tid, false)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	qp := r.URL.Query()
	page, pageSize, paged := parsePageParams(qp)

	base := `FROM billing_transaction bt
		LEFT JOIN billing_payment_ledger pl ON pl.billing_transaction_id = bt.id
		WHERE bt.account_id = ?`
	args := []interface{}{acc.ID}
	base, args = appendTxnListFiltersPrefixed(base, args, qp, filtered)

	total := 0
	if paged {
		if err := db.QueryRow(`SELECT COUNT(*) `+base, args...).Scan(&total); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
	}

	q := `SELECT bt.id, bt.transaction_type, bt.amount, bt.balance_before, bt.balance_after, bt.points_source_type,
		bt.project_id, bt.user_id, bt.workspace_id, bt.task_id, bt.billing_unit_id, bt.usage_amount, bt.description, bt.transaction_id, bt.created_at,
		COALESCE(pl.expires_at, '') AS expires_at
		` + base + ` ORDER BY bt.created_at DESC`
	queryArgs := append([]interface{}{}, args...)
	if paged {
		q += ` LIMIT ? OFFSET ?`
		queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)
	} else {
		q += ` LIMIT 5000`
	}

	rows, err := db.Query(q, queryArgs...)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()
	units := loadBillingUnitMap()
	var list []map[string]interface{}
	for rows.Next() {
		var id, amount, bb, ba int64
		var txnType, txnID, created, expiresAt string
		var ptsSrc, desc, proj, uid, wid, tidStr sql.NullString
		var unitID sql.NullInt64
		var usage float64
		if err := rows.Scan(&id, &txnType, &amount, &bb, &ba, &ptsSrc, &proj, &uid, &wid, &tidStr, &unitID, &usage, &desc, &txnID, &created, &expiresAt); err != nil {
			continue
		}
		row := map[string]interface{}{
			"id": formatID(id), "account": formatID(acc.ID), "transaction_type": txnType,
			"amount_points": amount, "balance_before_points": bb, "balance_after_points": ba,
			"transaction_id": txnID, "created_at": created, "usage_amount": usage,
		}
		if expiresAt != "" {
			row["expires_at"] = expiresAt
		}
		if ptsSrc.Valid {
			row["points_source_type"] = ptsSrc.String
		}
		if desc.Valid {
			row["description"] = desc.String
		}
		if proj.Valid && proj.String != "" {
			row["project_id"] = proj.String
		}
		if uid.Valid && uid.String != "" {
			row["user_id"] = uid.String
		}
		if wid.Valid && wid.String != "" {
			row["workspace_id"] = wid.String
		}
		if tidStr.Valid && tidStr.String != "" {
			row["task_id"] = tidStr.String
		}
		attachBillingUnit(row, unitID, units)
		list = append(list, row)
	}
	if list == nil {
		list = []map[string]interface{}{}
	}
	if filtered {
		list = djangoEnrichTransactions(r.Context(), tid, list)
	}
	attachTransactionDisplayFields(r.Context(), tid, list)
	attachLedgerSnapshots(r.Context(), tid, acc.ID, list)
	if !paged {
		total = len(list)
	}
	writeListOrPaged(w, list, page, pageSize, paged, total)
}

func appendUsageListFilters(q string, args []interface{}, qp url.Values) (string, []interface{}) {
	if v := qp.Get("start_date"); v != "" {
		q += ` AND date(usage_time) >= date(?)`
		args = append(args, v)
	}
	if v := qp.Get("end_date"); v != "" {
		q += ` AND date(usage_time) <= date(?)`
		args = append(args, v)
	}
	if v := qp.Get("project_id"); v != "" {
		q += ` AND project_id = ?`
		args = append(args, v)
	}
	if v := qp.Get("workspace_id"); v != "" {
		q += ` AND workspace_id = ?`
		args = append(args, v)
	}
	if v := qp.Get("user_id"); v != "" {
		q += ` AND user_id = ?`
		args = append(args, v)
	}
	if v := qp.Get("task_id"); v != "" {
		q += ` AND task_id = ?`
		args = append(args, v)
	}
	if v := qp.Get("billing_unit_type"); v != "" {
		q += ` AND billing_unit_id IN (SELECT id FROM billing_unit WHERE unit_type = ?)`
		args = append(args, v)
	}
	return q, args
}

func handleUsagesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	acc, _, err := getOrCreateBillingAccount(tid, false)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	qp := r.URL.Query()
	page, pageSize, paged := parsePageParams(qp)

	base := `FROM billing_usage WHERE account_id = ?`
	args := []interface{}{acc.ID}
	base, args = appendUsageListFilters(base, args, qp)

	total := 0
	if paged {
		if err := db.QueryRow(`SELECT COUNT(*) `+base, args...).Scan(&total); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
	}

	q := `SELECT id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id, description, usage_time
		` + base + ` ORDER BY usage_time DESC`
	queryArgs := append([]interface{}{}, args...)
	if paged {
		q += ` LIMIT ? OFFSET ?`
		queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)
	} else {
		q += ` LIMIT 5000`
	}

	rows, err := db.Query(q, queryArgs...)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()
	units := loadBillingUnitMap()
	var list []map[string]interface{}
	for rows.Next() {
		var id, unitID int64
		var amount float64
		var proj, uid, wid, taskID, desc, usageTime sql.NullString
		if err := rows.Scan(&id, &unitID, &amount, &proj, &uid, &wid, &taskID, &desc, &usageTime); err != nil {
			continue
		}
		u := map[string]interface{}{
			"id": formatID(id), "account": formatID(acc.ID),
			"amount": amount, "usage_time": usageTime.String,
		}
		attachBillingUnit(u, sql.NullInt64{Int64: unitID, Valid: true}, units)
		if proj.Valid && proj.String != "" {
			u["project_id"] = proj.String
		}
		if uid.Valid && uid.String != "" {
			u["user_id"] = uid.String
		}
		if wid.Valid && wid.String != "" {
			u["workspace_id"] = wid.String
		}
		if taskID.Valid && taskID.String != "" {
			u["task_id"] = taskID.String
		}
		if desc.Valid {
			u["description"] = desc.String
		}
		list = append(list, u)
	}
	if list == nil {
		list = []map[string]interface{}{}
	}
	list = djangoEnrichTransactions(r.Context(), tid, list)
	if !paged {
		total = len(list)
	}
	writeListOrPaged(w, list, page, pageSize, paged, total)
}
