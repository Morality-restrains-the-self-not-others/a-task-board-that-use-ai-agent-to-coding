package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

func adminOrderJSON(ctx context.Context, o *ResourceOrder, items []ResourceOrderItem) (map[string]interface{}, error) {
	m := orderJSON(o, items)
	m["tenant_id"] = formatID(o.TenantID)
	shares, err := listProfitSharingForOrder(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	m["profit_sharing"] = shares
	return m, nil
}

func listProfitSharingForOrder(ctx context.Context, orderID int64) ([]map[string]interface{}, error) {
	out := make([]map[string]interface{}, 0)
	// OPT-20260826-012：与管理端待分账队列一致，下发 app_id/openid（接收方成对身份，
	// 未登记时回退台账 referrer_openid 快照）；不暴露原始 referrer_openid 键。
	rows, err := db.Query(`
		SELECT ps.referrer_user_id, ps.commission_yuan_cents, ps.status,
		       COALESCE(ps.settle_after, ''), COALESCE(ps.settled_at, ''), COALESCE(ps.fail_reason, ''),
		       COALESCE(ps.fail_trace_id, ''),
		       COALESCE(NULLIF(TRIM(rcv.appid), ''), ''),
		       COALESCE(NULLIF(TRIM(rcv.openid), ''), NULLIF(TRIM(ps.referrer_openid), ''), '')
		FROM billing_profit_sharing ps
		LEFT JOIN billing_profit_sharing_receiver rcv ON rcv.referrer_user_id = ps.referrer_user_id
		WHERE ps.order_id = ?
		ORDER BY ps.id ASC`, orderID)
	if err != nil {
		return nil, fmt.Errorf("list profit sharing: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var receiver string
		var cents int64
		var status, settleAfter, settledAt, failReason, failTraceID, appID, openID string
		if err := rows.Scan(&receiver, &cents, &status, &settleAfter, &settledAt, &failReason, &failTraceID, &appID, &openID); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"receiver_user_id":  receiver,
			"amount_yuan":       centsToYuanStr(cents),
			"amount_yuan_cents": cents,
			"status":            status,
			"settle_after":      settleAfter,
			"settled_at":        settledAt,
			"fail_reason":       failReason,
			"fail_trace_id":     failTraceID,
			"app_id":            appID,
			"openid":            openID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	enrichProfitSharingReceiverDisplays(ctx, out)
	return out, nil
}

// profitSharingQueueFromSQL 返回待分账队列 FROM/JOIN 子句；referrerUserID 非空时
// 走推荐图内过滤（推荐绩效抽屉），其占位符须在 WHERE 条件之前入参（SQL 文本序）。
func profitSharingQueueFromSQL(referrerUserID string) (fromSQL string, fromArg interface{}, hasFromArg bool) {
	if strings.TrimSpace(referrerUserID) == "" {
		return `
FROM billing_profit_sharing ps
LEFT JOIN billing_resource_order o ON o.id = ps.order_id
LEFT JOIN billing_profit_sharing_receiver rcv ON rcv.referrer_user_id = ps.referrer_user_id`, nil, false
	}
	return `
FROM billing_profit_sharing ps
INNER JOIN billing_resource_order o ON o.id = ps.order_id AND o.user_id > 0
INNER JOIN billing_referral_edge e
  ON e.referrer_user_id = ?
 AND e.referred_user_id = (CAST(o.user_id AS CHAR) COLLATE utf8mb4_unicode_ci)
LEFT JOIN billing_profit_sharing_receiver rcv ON rcv.referrer_user_id = ps.referrer_user_id`,
		strings.TrimSpace(referrerUserID), true
}

// profitSharingQueueFilter 管理端待分账队列过滤条件（OPT-20260823-045：表头列过滤）。
type profitSharingQueueFilter struct {
	Statuses       []string // status IN (...)
	ReferrerUserID string   // 推荐图内过滤（抽屉），入参在 FROM 之后 WHERE 之前
	OrderNumber    string   // ps.order_number LIKE
	TenantID       int64    // ps.tenant_id 等值
	ReceiverUserID string   // 分账接收方 user_id 等值（ps.referrer_user_id）
	OutTradeNo     string   // OPT-20260825-023: 商户单号 LIKE（命中 o.out_trade_no 或 payment_ref 去 wechat: 前缀）
}

func listProfitSharingQueue(ctx context.Context, f profitSharingQueueFilter, limit, offset int) ([]map[string]interface{}, int64, error) {
	out := make([]map[string]interface{}, 0)
	// 入参顺序必须与 SQL 占位符文本序一致：FROM/JOIN 的 ? 在 WHERE 的 ? 之前。
	fromSQL, fromArg, hasFromArg := profitSharingQueueFromSQL(f.ReferrerUserID)
	args := make([]interface{}, 0, 8)
	if hasFromArg {
		args = append(args, fromArg)
	}
	conds := make([]string, 0, 8)
	if len(f.Statuses) > 0 {
		ph := make([]string, len(f.Statuses))
		for i, st := range f.Statuses {
			ph[i] = "?"
			args = append(args, st)
		}
		conds = append(conds, "ps.status IN ("+strings.Join(ph, ",")+")")
	}
	if on := strings.TrimSpace(f.OrderNumber); on != "" {
		conds = append(conds, "ps.order_number LIKE ?")
		args = append(args, "%"+on+"%")
	}
	if f.TenantID > 0 {
		conds = append(conds, "ps.tenant_id = ?")
		args = append(args, f.TenantID)
	}
	if rid := strings.TrimSpace(f.ReceiverUserID); rid != "" {
		conds = append(conds, "ps.referrer_user_id = ?")
		args = append(args, rid)
	}
	// OPT-20260825-023: 商户单号 LIKE — 命中订单表 out_trade_no 或 payment_ref 去前缀。
	// 运营对照微信支付商户后台排查时复制的是 WX 商户单号，而非内部 ORD- 订单号。
	if ot := strings.TrimSpace(f.OutTradeNo); ot != "" {
		conds = append(conds,
			"(o.out_trade_no LIKE ? OR REPLACE(COALESCE(o.payment_ref, ''), 'wechat:', '') LIKE ?)")
		args = append(args, "%"+ot+"%", "%"+ot+"%")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	countSQL := "SELECT COUNT(*) " + fromSQL + " " + where
	var total int64
	if err := db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count profit sharing queue: %w", err)
	}
	listSQL := `
		SELECT ps.id, ps.order_id, ps.order_number, ps.tenant_id, ps.referrer_user_id,
		       ps.total_yuan_cents, ps.commission_yuan_cents, ps.status,
		       COALESCE(ps.settle_after, ''), COALESCE(ps.settled_at, ''), COALESCE(ps.fail_reason, ''),
		       COALESCE(ps.fail_trace_id, ''),
		       COALESCE(CAST(o.user_id AS CHAR), ''),
		       COALESCE((
		         SELECT NULLIF(TRIM(l.provider_capture_id), '')
		         FROM billing_payment_ledger l
		         WHERE l.channel = 'wechat'
		           AND l.provider_ref = REPLACE(COALESCE(o.payment_ref, ''), 'wechat:', '')
		           AND COALESCE(l.provider_capture_id, '') <> ''
		         ORDER BY l.created_at DESC
		         LIMIT 1
		       ), NULLIF(TRIM(o.wechat_transaction_id), ''), ''),
		       COALESCE(NULLIF(TRIM(ps.wechat_profit_sharing_id), ''), ''),
		       COALESCE(NULLIF(TRIM(o.out_trade_no), ''), NULLIF(TRIM(REPLACE(COALESCE(o.payment_ref, ''), 'wechat:', '')), ''), ''),
		       COALESCE(NULLIF(TRIM(ps.out_profit_sharing_no), ''), ''),
		       COALESCE(NULLIF(TRIM(rcv.appid), ''), ''),
		       COALESCE(NULLIF(TRIM(rcv.openid), ''), NULLIF(TRIM(ps.referrer_openid), ''), '')
		` + fromSQL + ` ` + where + `
		ORDER BY ps.settle_after ASC, ps.id ASC
		LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	rows, err := db.Query(listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list profit sharing queue: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, orderID, tenantID, totalCents, commissionCents int64
		var orderNumber, receiver, status, settleAfter, settledAt, failReason, failTraceID, referredUserID string
		var wechatTxnID, wechatPSID, outTradeNo, outProfitSharingNo, appID, openID string
		if err := rows.Scan(&id, &orderID, &orderNumber, &tenantID, &receiver,
			&totalCents, &commissionCents, &status, &settleAfter, &settledAt, &failReason, &failTraceID, &referredUserID,
			&wechatTxnID, &wechatPSID, &outTradeNo, &outProfitSharingNo, &appID, &openID); err != nil {
			return nil, 0, err
		}
		if referredUserID == "0" {
			referredUserID = ""
		}
		// OPT-20260825-016：本地台账未向微信 POST 的记录（wechat_profit_sharing_id 为空）
		// 列表首屏直接给 not_submitted（前端映射「未向微信发起分账」），
		// 避免管理员先点同步才看到预期状态。
		wechatState := ""
		if strings.TrimSpace(wechatPSID) == "" {
			wechatState = wechatProfitSharingStateNotSubmitted
		}
		out = append(out, map[string]interface{}{
			"id":                       formatID(id),
			"order_id":                 formatID(orderID),
			"order_number":             orderNumber,
			"tenant_id":                formatID(tenantID),
			"receiver_user_id":         receiver,
			"referred_user_id":         referredUserID,
			"total_yuan":               centsToYuanStr(totalCents),
			"amount_yuan":              centsToYuanStr(commissionCents),
			"amount_yuan_cents":        commissionCents,
			"status":                   status,
			"settle_after":             settleAfter,
			"settled_at":               settledAt,
			"fail_reason":              failReason,
			"fail_trace_id":            failTraceID,
			"wechat_transaction_id":    wechatTxnID,
			"wechat_profit_sharing_id": wechatPSID,
			"wechat_state":             wechatState,
			"out_trade_no":             outTradeNo,
			"out_profit_sharing_no":    outProfitSharingNo,
			"app_id":                   appID,
			"openid":                   openID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, total, err
	}
	enrichProfitSharingReceiverDisplays(ctx, out)
	return out, total, nil
}

func receiverRegistrationStatusForReferrer(referrerUserID string) string {
	row, ok := loadProfitSharingReceiver(strings.TrimSpace(referrerUserID))
	if !ok {
		return ""
	}
	return row.Status
}

func profitSharingIDsInReferrerGraph(referrerUserID string, ids []int64) (map[int64]struct{}, error) {
	out := make(map[int64]struct{}, len(ids))
	if len(ids) == 0 || strings.TrimSpace(referrerUserID) == "" {
		return out, nil
	}
	ph := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, strings.TrimSpace(referrerUserID))
	for i, id := range ids {
		ph[i] = "?"
		args = append(args, id)
	}
	rows, err := db.Query(`
		SELECT ps.id
		FROM billing_profit_sharing ps
		INNER JOIN billing_resource_order o ON o.id = ps.order_id AND o.user_id > 0
		INNER JOIN billing_referral_edge e
		  ON e.referrer_user_id = ?
		 AND e.referred_user_id = (CAST(o.user_id AS CHAR) COLLATE utf8mb4_unicode_ci)
		WHERE ps.id IN (`+strings.Join(ph, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}

// userDisplayInfo 是 taskAuth /api/internal/users/batch/details/ 单用户命中信息。
type userDisplayInfo struct {
	Email    string
	Username string
}

// fetchUserBatchDetailsFn 批量解析 user_id → 显示信息（可注入测试）。
var fetchUserBatchDetailsFn = fetchUserBatchDetails

func fetchUserBatchDetails(ctx context.Context, userIDs []interface{}) (map[string]userDisplayInfo, error) {
	statusCode, raw, err := taskAuthRequest(ctx, http.MethodPost,
		"/api/internal/users/batch/details/", map[string]interface{}{"user_ids": userIDs})
	if err != nil {
		slog.ErrorContext(ctx, "enrichProfitSharingReceiverDisplays: taskAuth request failed",
			"error", err.Error(), "user_ids", len(userIDs))
		return nil, err
	}
	if statusCode != http.StatusOK {
		slog.ErrorContext(ctx, "enrichProfitSharingReceiverDisplays: taskAuth returned non-200",
			"status", statusCode)
		return nil, fmt.Errorf("taskAuth batch/details status %d", statusCode)
	}
	var resp struct {
		Results map[string]struct {
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		slog.ErrorContext(ctx, "enrichProfitSharingReceiverDisplays: decode failed",
			"error", err.Error())
		return nil, err
	}
	out := make(map[string]userDisplayInfo, len(resp.Results))
	for uid, info := range resp.Results {
		out[uid] = userDisplayInfo{Email: info.Email, Username: info.Username}
	}
	return out, nil
}

// enrichProfitSharingReceiverDisplays 批量解析 receiver_user_id → 显示名（username →
// email → user_id 兜底），供管理端分账接收方列表/订单详情展示。失败优雅降级
// （receiver_display 保持为空，前端回退 user_id），绝不因上游抖动阻塞账单数据。
// 显示名解析不附带 openid；openid/app_id 由列表 SQL 单独下发给平台员工。
func enrichProfitSharingReceiverDisplays(ctx context.Context, rows []map[string]interface{}) {
	if len(rows) == 0 {
		return
	}
	rawIDs := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		if id, ok := row["receiver_user_id"].(string); ok && id != "" {
			rawIDs = append(rawIDs, id)
		}
	}
	if len(rawIDs) == 0 {
		return
	}
	byID, err := fetchUserBatchDetailsFn(ctx, rawIDs)
	if err != nil {
		return
	}
	for _, row := range rows {
		uid, _ := row["receiver_user_id"].(string)
		info, ok := byID[uid]
		if !ok {
			continue
		}
		display := strings.TrimSpace(info.Username)
		if display == "" {
			display = strings.TrimSpace(info.Email)
		}
		if display == "" {
			display = uid
		}
		row["receiver_display"] = display
	}
}
