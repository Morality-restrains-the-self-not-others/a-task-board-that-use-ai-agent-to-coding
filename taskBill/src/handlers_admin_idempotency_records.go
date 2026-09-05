package main

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"authz"
	"tracelog"
)

// handleSystemAdminListIdempotencyRecords GET /api/system-admin/idempotency-records/
// 超管查询幂等键审计记录（OPT-20260823-058）。
// admin_grant 等管理操作落库 operator_user_id / tenant_id / op_type，
// 可追溯「哪次操作产生了哪张赠送订单」。仅平台员工（super_admin/employee）可查。
func handleSystemAdminListIdempotencyRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 50 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	conds := []string{"1=1"}
	var args []interface{}

	if op := strings.TrimSpace(r.URL.Query().Get("op_type")); op != "" {
		conds = append(conds, "op_type = ?")
		args = append(args, op)
	}
	if tidRaw := strings.TrimSpace(r.URL.Query().Get("tenant_id")); tidRaw != "" {
		tid, err := strconv.ParseInt(tidRaw, 10, 64)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "tenant_id 须为数字", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		conds = append(conds, "tenant_id = ?")
		args = append(args, tid)
	}
	if uid := strings.TrimSpace(r.URL.Query().Get("operator_user_id")); uid != "" {
		conds = append(conds, "operator_user_id = ?")
		args = append(args, uid)
	}
	if keyPrefix := strings.TrimSpace(r.URL.Query().Get("key")); keyPrefix != "" {
		conds = append(conds, "`key` LIKE ?")
		args = append(args, keyPrefix+"%")
	}

	countQuery := `SELECT COUNT(*) FROM billing_idempotency_key WHERE ` + strings.Join(conds, " AND ")
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		slog.WarnContext(r.Context(), "admin_idempotency_count_failed",
			"level", "warn",
			"error", err.Error(),
		)
		writeErrorJSON(w, http.StatusInternalServerError, "count failed", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	query := `SELECT id, ` + "`key`" + `, created_at, transaction_id,
		COALESCE(operator_user_id, ''), COALESCE(tenant_id, 0), COALESCE(op_type, '')
		FROM billing_idempotency_key WHERE ` + strings.Join(conds, " AND ") +
		` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		slog.WarnContext(r.Context(), "admin_idempotency_list_failed",
			"level", "warn",
			"error", err.Error(),
		)
		writeErrorJSON(w, http.StatusInternalServerError, "query failed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()

	records := make([]map[string]interface{}, 0, limit)
	for rows.Next() {
		var (
			id            int64
			key           string
			createdAt     string
			transactionID int64
			operator      string
			tenantID      int64
			opType        string
		)
		if err := rows.Scan(&id, &key, &createdAt, &transactionID, &operator, &tenantID, &opType); err != nil {
			slog.WarnContext(r.Context(), "admin_idempotency_scan_failed",
				"level", "warn",
				"error", err.Error(),
			)
			writeErrorJSON(w, http.StatusInternalServerError, "scan failed", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		record := map[string]interface{}{
			"id":             formatID(id),
			"key":            key,
			"created_at":     createdAt,
			"transaction_id": formatID(transactionID),
			"op_type":        opType,
		}
		if operator != "" {
			record["operator_user_id"] = operator
		}
		if tenantID != 0 {
			record["tenant_id"] = formatID(tenantID)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(r.Context(), "admin_idempotency_rows_failed",
			"level", "warn",
			"error", err.Error(),
		)
		writeErrorJSON(w, http.StatusInternalServerError, "rows failed", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"records": records,
	})
}
