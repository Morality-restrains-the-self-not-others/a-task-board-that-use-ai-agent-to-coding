package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// voidOrphanLoginMethods 批量作废无对应用户的孤儿 login_method（OPT-20260824-090）。
//
// 注册占用逻辑已忽略「用户行缺失」的 phone/email 绑定，并在新注册时作废同号旧行；
// 但历史数据中仍可能残留 binding_voided_at IS NULL 而 auth_user 已不存在的孤儿行
// （用户被删除/账号迁移后 login_method 未随行清理），干扰人工查库与审计。
//
// 批量 UPDATE 通过 LEFT JOIN auth_user 判定 `u.id IS NULL`，只作废用户行缺失的绑定，
// 不误伤活跃用户；LIMIT 分批防止一次锁太多行。返回作废行数。
func voidOrphanLoginMethods(limit int64) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	// MySQL 不允许多表 UPDATE 直接 LIMIT；用双层子查询圈定孤儿 id 集后再更新
	//（内层派生表规避 "can't specify target table for update in FROM clause"）。
	res, err := db.Exec(`
		UPDATE auth_login_method
		SET binding_voided_at = ?, updated_at = ?
		WHERE id IN (
			SELECT id FROM (
				SELECT lm2.id
				FROM auth_login_method lm2
				LEFT JOIN auth_user u ON u.id = lm2.object_id
				WHERE lm2.binding_voided_at IS NULL
				  AND u.id IS NULL
				LIMIT ?
			) AS orphan_ids
		)`, now, now, limit)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// handleInternalVoidOrphanLoginMethods 内部维护端点：POST /api/internal/taskauth/
// login-methods/void-orphans/?limit=N — 供 cron/运维定时清理孤儿 login_method。
func handleInternalVoidOrphanLoginMethods(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := int64(500)
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			limit = v
		}
	}
	n, err := voidOrphanLoginMethods(limit)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"voided": n})
}
