package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type monthlyEarningRow struct {
	Month       string `json:"month"`
	Consumption string `json:"consumption"`
	Commission  string `json:"commission"`
}

type channelStatRow struct {
	ChannelCode   string `json:"channel_code"`
	Name          string `json:"name"`
	IsDefault     bool   `json:"is_default"`
	Status        string `json:"status"`
	ReferralCount int    `json:"referral_count"`
	Consumption   string `json:"consumption"`
	Commission    string `json:"commission"`
}

type referralStatsResponse struct {
	ReferralCount       int                 `json:"referral_count"`
	ReferralRateDisplay string              `json:"referral_rate_display"`
	MonthlyEarnings     []monthlyEarningRow `json:"monthly_earnings"`
	Channels            []channelStatRow    `json:"channels"`
}

type referralStatsFilter struct {
	ChannelCode string
	FromDate    string
	ToDate      string
}

func handleReferralStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}

	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	filter := referralStatsFilter{
		ChannelCode: strings.TrimSpace(q.Get("channel_code")),
		FromDate:    strings.TrimSpace(q.Get("from")),
		ToDate:      strings.TrimSpace(q.Get("to")),
	}
	stats, err := getReferralStatsFiltered(userID, filter)
	if err != nil {
		log.Printf("[taskReferral] referral stats: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func getReferralStats(referrerUserID string) (*referralStatsResponse, error) {
	return getReferralStatsFiltered(referrerUserID, referralStatsFilter{})
}

func appendBoundAtFilter(where *[]string, args *[]interface{}, filter referralStatsFilter, col string) {
	if filter.FromDate != "" {
		*where = append(*where, col+" >= ?")
		*args = append(*args, filter.FromDate+" 00:00:00")
	}
	if filter.ToDate != "" {
		*where = append(*where, col+" <= ?")
		*args = append(*args, filter.ToDate+" 23:59:59.999999")
	}
}

func getReferralStatsFiltered(referrerUserID string, filter referralStatsFilter) (*referralStatsResponse, error) {
	if billDB == nil {
		return nil, fmt.Errorf("billDB unavailable")
	}

	where := []string{"referrer_user_id = ?"}
	args := []interface{}{referrerUserID}
	if filter.ChannelCode != "" {
		where = append(where, "channel_code = ?")
		args = append(args, filter.ChannelCode)
	}
	appendBoundAtFilter(&where, &args, filter, "bound_at")

	var referralCount int
	err := billDB.QueryRow(
		`SELECT COUNT(*) FROM billing_referral_edge WHERE `+strings.Join(where, " AND "),
		args...,
	).Scan(&referralCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	monthlyEarnings, err := getMonthlyEarnings(referrerUserID, filter)
	if err != nil {
		return nil, err
	}
	channels, err := getChannelStats(referrerUserID, filter)
	if err != nil {
		return nil, err
	}

	return &referralStatsResponse{
		ReferralCount:       referralCount,
		ReferralRateDisplay: configuredReferralRateDisplay(),
		MonthlyEarnings:     monthlyEarnings,
		Channels:            channels,
	}, nil
}

func getMonthlyEarnings(referrerUserID string, filter referralStatsFilter) ([]monthlyEarningRow, error) {
	where := []string{"referrer_user_id = ?"}
	args := []interface{}{referrerUserID}
	if filter.ChannelCode != "" {
		where = append(where, "channel_code = ?")
		args = append(args, filter.ChannelCode)
	}
	appendBoundAtFilter(&where, &args, filter, "consumed_at")

	rows, err := billDB.Query(`
		SELECT
			DATE_FORMAT(consumed_at, '%Y-%m') AS month,
			COALESCE(SUM(consumption_points), 0) AS total_consumption,
			COALESCE(SUM(commission_points), 0) AS total_commission
		FROM billing_referral_commission_accrual
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY month
		ORDER BY month DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	earnings := make([]monthlyEarningRow, 0)
	for rows.Next() {
		var month string
		var consumption, commission int64
		if err := rows.Scan(&month, &consumption, &commission); err != nil {
			return nil, err
		}
		earnings = append(earnings, monthlyEarningRow{
			Month:       month,
			Consumption: fmt.Sprintf("%.2f", float64(consumption)/100.0),
			Commission:  fmt.Sprintf("%.2f", float64(commission)/100.0),
		})
	}
	return earnings, rows.Err()
}

func getChannelStats(referrerUserID string, filter referralStatsFilter) ([]channelStatRow, error) {
	where := []string{"referrer_user_id = ?"}
	args := []interface{}{referrerUserID}
	if filter.ChannelCode != "" {
		where = append(where, "channel_code = ?")
		args = append(args, filter.ChannelCode)
	}
	appendBoundAtFilter(&where, &args, filter, "bound_at")

	countRows, err := billDB.Query(`
		SELECT channel_code, COUNT(*) FROM billing_referral_edge
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY channel_code
	`, args...)
	if err != nil {
		return nil, err
	}
	defer countRows.Close()
	counts := map[string]int{}
	for countRows.Next() {
		var code string
		var n int
		if err := countRows.Scan(&code, &n); err != nil {
			return nil, err
		}
		counts[code] = n
	}
	if err := countRows.Err(); err != nil {
		return nil, err
	}

	earnWhere := []string{"referrer_user_id = ?"}
	earnArgs := []interface{}{referrerUserID}
	if filter.ChannelCode != "" {
		earnWhere = append(earnWhere, "channel_code = ?")
		earnArgs = append(earnArgs, filter.ChannelCode)
	}
	appendBoundAtFilter(&earnWhere, &earnArgs, filter, "consumed_at")
	earnRows, err := billDB.Query(`
		SELECT channel_code,
		       COALESCE(SUM(consumption_points), 0),
		       COALESCE(SUM(commission_points), 0)
		FROM billing_referral_commission_accrual
		WHERE `+strings.Join(earnWhere, " AND ")+`
		GROUP BY channel_code
	`, earnArgs...)
	if err != nil {
		return nil, err
	}
	defer earnRows.Close()
	type earn struct{ cons, comm int64 }
	earns := map[string]earn{}
	for earnRows.Next() {
		var code string
		var e earn
		if err := earnRows.Scan(&code, &e.cons, &e.comm); err != nil {
			return nil, err
		}
		earns[code] = e
	}
	if err := earnRows.Err(); err != nil {
		return nil, err
	}

	meta := map[string]referralChannelRow{}
	if db != nil {
		chRows, err := db.Query(
			`SELECT code, channel_name, is_default, status FROM referral_share_code WHERE user_id = ?`,
			referrerUserID,
		)
		if err == nil {
			defer chRows.Close()
			for chRows.Next() {
				var row referralChannelRow
				var isDefault int
				if err := chRows.Scan(&row.Code, &row.Name, &isDefault, &row.Status); err != nil {
					break
				}
				row.IsDefault = isDefault == 1
				meta[row.Code] = row
			}
		}
	}

	seen := map[string]bool{}
	out := make([]channelStatRow, 0)
	appendRow := func(code string) {
		if seen[code] {
			return
		}
		seen[code] = true
		name := "历史（未分渠道）"
		isDefault := false
		status := ""
		if m, ok := meta[code]; ok {
			name = m.Name
			isDefault = m.IsDefault
			status = m.Status
		} else if code != "" {
			name = code
		}
		e := earns[code]
		out = append(out, channelStatRow{
			ChannelCode:   code,
			Name:          name,
			IsDefault:     isDefault,
			Status:        status,
			ReferralCount: counts[code],
			Consumption:   fmt.Sprintf("%.2f", float64(e.cons)/100.0),
			Commission:    fmt.Sprintf("%.2f", float64(e.comm)/100.0),
		})
	}
	for code := range counts {
		appendRow(code)
	}
	for code := range earns {
		appendRow(code)
	}
	if filter.ChannelCode == "" {
		for code := range meta {
			appendRow(code)
		}
	}
	return out, nil
}
