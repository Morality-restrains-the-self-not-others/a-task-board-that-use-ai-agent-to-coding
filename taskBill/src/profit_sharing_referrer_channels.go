package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"tracelog"
)

// referrerChannelView 推荐人可见的渠道聚合行：不含订单号 / 受推荐人 ID。
type referrerChannelView struct {
	ChannelCode              string `json:"channel_code"`
	PeriodFrom               string `json:"period_from,omitempty"`
	PeriodTo                 string `json:"period_to,omitempty"`
	OrderAmountYuanCents     int64  `json:"order_amount_yuan_cents"`
	FrozenAmountYuanCents    int64  `json:"frozen_amount_yuan_cents"`
	FailedAmountYuanCents    int64  `json:"failed_amount_yuan_cents"`
	ShareableAmountYuanCents int64  `json:"shareable_amount_yuan_cents"`
	Shareable                bool   `json:"shareable"`
}

type referrerPSListRow struct {
	ChannelCode string
	Commission  int64
	Total       int64
	Status      string
	PaidAt      string
}

type referrerShareChannelBody struct {
	ChannelCode string `json:"channel_code"`
}

type channelAgg struct {
	from, to                     time.Time
	hasTime                      bool
	order, frozen, failed, share int64
}

func aggregateReferrerChannels(rows []referrerPSListRow, now time.Time) []referrerChannelView {
	accs := map[string]*channelAgg{}
	keys := make([]string, 0, 4)
	for _, row := range rows {
		a, ok := accs[row.ChannelCode]
		if !ok {
			a = &channelAgg{}
			accs[row.ChannelCode] = a
			keys = append(keys, row.ChannelCode)
		}
		a.order += row.Total
		display, canShare := referrerProfitSharingDisplay(row.Status, row.PaidAt, now)
		if display == referrerPSFrozen {
			a.frozen += row.Commission
		}
		if display == referrerPSFailed {
			a.failed += row.Commission
		}
		if canShare {
			a.share += row.Commission
		}
		if t, ok := parseFlexibleTime(row.PaidAt); ok {
			if !a.hasTime || t.Before(a.from) {
				a.from = t
			}
			if !a.hasTime || t.After(a.to) {
				a.to = t
			}
			a.hasTime = true
		}
	}
	sort.Strings(keys)
	out := make([]referrerChannelView, 0, len(keys))
	for _, k := range keys {
		a := accs[k]
		v := referrerChannelView{
			ChannelCode:              k,
			OrderAmountYuanCents:     a.order,
			FrozenAmountYuanCents:    a.frozen,
			FailedAmountYuanCents:    a.failed,
			ShareableAmountYuanCents: a.share,
			Shareable:                a.share > 0,
		}
		if a.hasTime {
			v.PeriodFrom = a.from.UTC().Format(time.RFC3339)
			v.PeriodTo = a.to.UTC().Format(time.RFC3339)
		}
		out = append(out, v)
	}
	return out
}

func handleReferrerProfitSharingList(w http.ResponseWriter, r *http.Request, userID string) {
	rows, err := db.Query(`
		SELECT COALESCE(e.channel_code, ''),
		       ps.commission_yuan_cents, ps.total_yuan_cents,
		       ps.status, COALESCE(o.paid_at, '')
		FROM billing_profit_sharing ps
		INNER JOIN billing_resource_order o ON o.id = ps.order_id
		LEFT JOIN billing_referral_edge e
		  ON e.referred_user_id = (CAST(o.user_id AS CHAR) COLLATE utf8mb4_unicode_ci)
		 AND e.referrer_user_id = ps.referrer_user_id
		WHERE ps.referrer_user_id = ?
		ORDER BY ps.created_at DESC
		LIMIT 500`, userID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()

	items := make([]referrerPSListRow, 0, 8)
	for rows.Next() {
		var it referrerPSListRow
		if err := rows.Scan(&it.ChannelCode, &it.Commission, &it.Total, &it.Status, &it.PaidAt); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		it.ChannelCode = strings.TrimSpace(it.ChannelCode)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	channels := aggregateReferrerChannels(items, time.Now().UTC())
	slog.InfoContext(r.Context(), "referrer_channel_profit_sharing_list",
		"level", "info",
		"referrer_user_id", userID,
		"channel_count", len(channels),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{"channels": channels})
}

func handleReferrerShareChannel(w http.ResponseWriter, r *http.Request, userID string) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid body", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	var body referrerShareChannelBody
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
			return
		}
	}
	channelCode := strings.TrimSpace(body.ChannelCode)

	qrows, err := db.Query(`
		SELECT ps.id
		FROM billing_profit_sharing ps
		INNER JOIN billing_resource_order o ON o.id = ps.order_id
		LEFT JOIN billing_referral_edge e
		  ON e.referred_user_id = (CAST(o.user_id AS CHAR) COLLATE utf8mb4_unicode_ci)
		 AND e.referrer_user_id = ps.referrer_user_id
		WHERE ps.referrer_user_id = ?
		  AND COALESCE(e.channel_code, '') = ?`, userID, channelCode)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer qrows.Close()

	ids := make([]int64, 0, 8)
	for qrows.Next() {
		var id int64
		if err := qrows.Scan(&id); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		ids = append(ids, id)
	}
	if err := qrows.Err(); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if len(ids) == 0 {
		writeErrorJSON(w, http.StatusNotFound, "profit sharing record not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	shared := 0
	conflictOnly := true
	now := time.Now().UTC()
	for _, id := range ids {
		code, _, idempotent, shareErr := shareReferrerProfitSharingRecord(r, userID, id, now)
		if code == http.StatusInternalServerError {
			writeErrorJSON(w, http.StatusInternalServerError, shareErr, tracelog.TraceIDFromContext(r.Context()))
			return
		}
		if code == http.StatusOK {
			conflictOnly = false
			if !idempotent {
				shared++
			}
			continue
		}
		if code == http.StatusBadGateway || code == http.StatusBadRequest {
			writeErrorJSON(w, code, shareErr, tracelog.TraceIDFromContext(r.Context()))
			return
		}
	}
	if conflictOnly && shared == 0 {
		writeErrorJSON(w, http.StatusConflict, "当前状态不可分账", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	slog.InfoContext(r.Context(), "referrer_share_channel_ok",
		"level", "info",
		"referrer_user_id", userID,
		"channel_code", channelCode,
		"shared_count", shared,
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "state": "shared", "shared_count": shared})
}
