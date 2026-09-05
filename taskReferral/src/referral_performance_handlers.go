package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"tracelog"
)

// ── Types ──

type referredUserRow struct {
	UserID        string `json:"user_id"`
	RechargeCount int    `json:"recharge_count"`
	RechargeTotal string `json:"recharge_total"`
}

type commissionItemRow struct {
	ID                string `json:"id"`
	ReferredUserID    string `json:"referred_user_id"`
	ResourceType      string `json:"resource_type"`
	ConsumptionPoints string `json:"consumption_points"`
	CommissionPoints  string `json:"commission_points"`
	Status            string `json:"status"`
	ConsumedAt        string `json:"consumed_at"`
}

type commissionBlock struct {
	PendingPoints         string              `json:"pending_points"`
	SettledPoints         string              `json:"settled_points"`
	CommissionRateDisplay string              `json:"commission_rate_display"`
	SettleDelayDays       int                 `json:"settle_delay_days"`
	Items                 []commissionItemRow `json:"items"`
}

type asReferredBlock struct {
	ReferrerUserID string `json:"referrer_user_id"`
	RechargeCount  int    `json:"recharge_count"`
	RechargeTotal  string `json:"recharge_total"`
}

type ownPaymentRow struct {
	PaidAt           string `json:"paid_at"`
	Amount           string `json:"amount"`
	PointsSourceType string `json:"points_source_type"`
}

type referralPerformanceResponse struct {
	ReferralCount         int               `json:"referral_count"`
	OwnRechargeTotal      string            `json:"own_recharge_total"`
	OwnRechargeCount      int               `json:"own_recharge_count"`
	ReferredRechargeTotal string            `json:"referred_recharge_total"`
	OwnPayments           []ownPaymentRow   `json:"own_payments"`
	ReferredUsers         []referredUserRow `json:"referred_users"`
	AsReferred            *asReferredBlock  `json:"as_referred"`
	Commission            *commissionBlock  `json:"commission"`
}

// ── Handler ──

func handleAdminReferralPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	if _, ok := requireSuperuser(w, r); !ok {
		return
	}

	uid := r.PathValue("uid")
	if uid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing user ID"})
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	result, err := getReferralPerformance(uid, startDate, endDate)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_referral_performance_failed",
			"level", "error",
			"uid", uid,
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		return
	}
	asReferred := false
	if result.AsReferred != nil {
		asReferred = true
	}
	slog.InfoContext(r.Context(), "admin_referral_performance",
		"level", "info",
		"uid", uid,
		"referral_count", result.ReferralCount,
		"referred_users", len(result.ReferredUsers),
		"own_recharge_count", result.OwnRechargeCount,
		"as_referred", asReferred,
		"trace_id", tracelog.TraceIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, result)
}

// ── Query logic ──

func getReferralPerformance(uid, startDate, endDate string) (*referralPerformanceResponse, error) {
	if billDB == nil {
		return nil, fmt.Errorf("billDB unavailable")
	}

	// referral_count
	var referralCount int
	err := billDB.QueryRow(
		`SELECT COUNT(*) FROM billing_referral_edge WHERE referrer_user_id = ?`, uid,
	).Scan(&referralCount)
	if err != nil {
		return nil, err
	}

	// date filter clause for billing_transaction queries
	dateFilter := ""
	dateArgs := []interface{}{}
	if startDate != "" {
		dateFilter += " AND created_at >= ?"
		dateArgs = append(dateArgs, startDate+" 00:00:00")
	}
	if endDate != "" {
		dateFilter += " AND created_at <= ?"
		dateArgs = append(dateArgs, endDate+" 23:59:59")
	}

	ownTotal, ownCount, err := queryUserPaymentTotals(uid, dateFilter, dateArgs)
	if err != nil {
		return nil, err
	}
	ownPayments, err := queryUserOwnPayments(uid, dateFilter, dateArgs)
	if err != nil {
		return nil, err
	}

	// referred users - get all referred user IDs from referral edge
	rows, err := billDB.Query(
		`SELECT referred_user_id FROM billing_referral_edge WHERE referrer_user_id = ?`, uid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referredIDs []string
	for rows.Next() {
		var rid string
		if err := rows.Scan(&rid); err != nil {
			return nil, err
		}
		referredIDs = append(referredIDs, rid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// per-referred-user recharge totals and total referred recharge
	referredUsers := make([]referredUserRow, 0, len(referredIDs))
	var referredRechargeTotal int64
	for _, rid := range referredIDs {
		total, count, err := queryUserPaymentTotals(rid, dateFilter, dateArgs)
		if err != nil {
			return nil, err
		}
		referredUsers = append(referredUsers, referredUserRow{
			UserID:        rid,
			RechargeCount: count,
			RechargeTotal: centsToYuan(total),
		})
		referredRechargeTotal += total
	}

	asReferred, err := lookupAsReferred(uid, ownCount, ownTotal)
	if err != nil {
		return nil, err
	}

	// commission: pending/settled sums
	var pendingSum, settledSum int64
	err = billDB.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN status = 'pending' THEN commission_points ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'settled' THEN commission_points ELSE 0 END), 0)
		FROM billing_referral_commission_accrual
		WHERE referrer_user_id = ?`, uid,
	).Scan(&pendingSum, &settledSum)
	if err != nil {
		return nil, err
	}

	// settle_delay_days from billing_referral_config
	settleDelayDays := 15
	var cfgDays int
	err = billDB.QueryRow(
		`SELECT settle_delay_days FROM billing_referral_config WHERE singleton_key = 'global'`,
	).Scan(&cfgDays)
	if err == nil && cfgDays >= 8 {
		settleDelayDays = cfgDays
	}

	// commission items (pending + settled)
	commissionDateFilter := ""
	commissionDateArgs := []interface{}{uid}
	if startDate != "" {
		commissionDateFilter += " AND a.consumed_at >= ?"
		commissionDateArgs = append(commissionDateArgs, startDate+" 00:00:00")
	}
	if endDate != "" {
		commissionDateFilter += " AND a.consumed_at <= ?"
		commissionDateArgs = append(commissionDateArgs, endDate+" 23:59:59")
	}

	itemRows, err := billDB.Query(`
		SELECT a.id, a.referred_user_id, a.consumption_points, a.commission_points,
		       a.status, a.consumed_at,
		       COALESCE(t.points_source_type, '') as resource_type
		FROM billing_referral_commission_accrual a
		LEFT JOIN billing_transaction t ON t.transaction_id = a.source_txn_id
		WHERE a.referrer_user_id = ? AND a.status IN ('pending', 'settled')`+
		commissionDateFilter+`
		ORDER BY a.consumed_at DESC
		LIMIT 200`, commissionDateArgs...)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	items := make([]commissionItemRow, 0)
	for itemRows.Next() {
		var (
			id                   int64
			referredUserID       string
			consumptionPointsRaw int64
			commissionPointsRaw  int64
			status               string
			consumedAt           string
			resourceType         string
		)
		if err := itemRows.Scan(&id, &referredUserID, &consumptionPointsRaw, &commissionPointsRaw,
			&status, &consumedAt, &resourceType); err != nil {
			return nil, err
		}
		if resourceType == "" {
			resourceType = "unknown"
		}
		items = append(items, commissionItemRow{
			ID:                fmt.Sprintf("%d", id),
			ReferredUserID:    referredUserID,
			ResourceType:      resourceType,
			ConsumptionPoints: centsToYuan(consumptionPointsRaw),
			CommissionPoints:  centsToYuan(commissionPointsRaw),
			Status:            status,
			ConsumedAt:        consumedAt,
		})
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}
	if items == nil {
		items = []commissionItemRow{}
	}

	commission := &commissionBlock{
		PendingPoints:         centsToYuan(pendingSum),
		SettledPoints:         centsToYuan(settledSum),
		CommissionRateDisplay: configuredReferralRateDisplay(),
		SettleDelayDays:       settleDelayDays,
		Items:                 items,
	}

	return &referralPerformanceResponse{
		ReferralCount:         referralCount,
		OwnRechargeTotal:      centsToYuan(ownTotal),
		OwnRechargeCount:      ownCount,
		ReferredRechargeTotal: centsToYuan(referredRechargeTotal),
		OwnPayments:           ownPayments,
		ReferredUsers:         referredUsers,
		AsReferred:            asReferred,
		Commission:            commission,
	}, nil
}

// userPaymentPredicate 与 taskBill 累计支付口径对齐：钱包实付充值 + 资源订单实付 − 退款；不含 admin_grant。
// user_id 绑两次：直连流水 + 同账户反查（历史 resource_purchase 常缺 user_id）。
func userPaymentPredicate() string {
	return `(user_id = ? OR account_id IN (
			SELECT account_id FROM (
				SELECT DISTINCT account_id FROM billing_transaction
				WHERE user_id = ? AND account_id IS NOT NULL AND account_id <> 0
			) AS user_accounts
		))
		AND (
			(transaction_type = 'recharge' AND points_source_type IN ('user_recharge','user_recharge_paypal','user_recharge_admin','user_recharge_wechat'))
			OR (transaction_type = 'consumption' AND points_source_type = 'resource_purchase')
			OR transaction_type = 'refund'
		)`
}

func queryUserPaymentTotals(uid, dateFilter string, dateArgs []interface{}) (total int64, count int, err error) {
	args := []interface{}{uid, uid}
	args = append(args, dateArgs...)
	err = billDB.QueryRow(
		`SELECT COALESCE(SUM(amount), 0), COUNT(*)
		 FROM billing_transaction
		 WHERE `+userPaymentPredicate()+dateFilter, args...,
	).Scan(&total, &count)
	return
}

// queryUserOwnPayments 返回用户自身支付流水（时间、金额、points_source_type），口径与
// queryUserPaymentTotals 一致（userPaymentPredicate + 日期过滤），按时间倒序取最近 50 条。
func queryUserOwnPayments(uid, dateFilter string, dateArgs []interface{}) ([]ownPaymentRow, error) {
	args := []interface{}{uid, uid}
	args = append(args, dateArgs...)
	rows, err := billDB.Query(
		`SELECT created_at, amount, COALESCE(points_source_type, '')
		 FROM billing_transaction
		 WHERE `+userPaymentPredicate()+dateFilter+`
		 ORDER BY created_at DESC
		 LIMIT 50`, args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ownPaymentRow, 0, 16)
	for rows.Next() {
		var (
			createdAt        string
			amount           int64
			pointsSourceType string
		)
		if err := rows.Scan(&createdAt, &amount, &pointsSourceType); err != nil {
			return nil, err
		}
		if pointsSourceType == "" {
			pointsSourceType = "unknown"
		}
		out = append(out, ownPaymentRow{
			PaidAt:           createdAt,
			Amount:           centsToYuan(amount),
			PointsSourceType: pointsSourceType,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func lookupAsReferred(uid string, ownCount int, ownTotal int64) (*asReferredBlock, error) {
	var referrerID string
	err := billDB.QueryRow(
		`SELECT referrer_user_id FROM billing_referral_edge WHERE referred_user_id = ?`, uid,
	).Scan(&referrerID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if referrerID == "" {
		return nil, nil
	}
	return &asReferredBlock{
		ReferrerUserID: referrerID,
		RechargeCount:  ownCount,
		RechargeTotal:  centsToYuan(ownTotal),
	}, nil
}

func centsToYuan(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100.0)
}
