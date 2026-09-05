package main

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

// wechatProfitSharingStateNotSubmitted 本地台账尚未向微信 POST /v3/profitsharing/orders。
// 查询接口（GET /v3/profitsharing/orders/{out_order_no}）在记录不存在时返回 404
// RESOURCE_NOT_EXISTS（https://pay.weixin.qq.com/doc/v3/merchant/4012525210 ），
// 待分账冻结期内这是预期状态，禁止当成故障展示「微信侧未找到分账单」。
// OPT-20260826-009：哨兵用机器码 not_submitted，展示文案只在 UI 层映射；
// 旧中文「尚未提交微信」仍可能出现在旧二进制/存量数据，前端兼容识别。
const wechatProfitSharingStateNotSubmitted = "not_submitted"

func refreshProfitSharingWechatStates(r *http.Request, ids []int64) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(ids))
	ctx := r.Context()
	for _, id := range ids {
		row := map[string]interface{}{
			"id":           formatID(id),
			"wechat_state": "",
			"wechat_error": "",
		}
		var orderID int64
		var outNo, wechatPSID string
		err := db.QueryRow(`
			SELECT order_id, COALESCE(out_profit_sharing_no, ''), COALESCE(wechat_profit_sharing_id, '')
			FROM billing_profit_sharing WHERE id = ?`, id).Scan(&orderID, &outNo, &wechatPSID)
		if err != nil {
			slog.WarnContext(ctx, "admin_profit_sharing_refresh_row_load_failed",
				"level", "warn",
				"ps_id", formatID(id),
				"error", err.Error(),
			)
			row["wechat_error"] = "加载分账记录失败"
			row["wechat_error_trace_id"] = tracelog.TraceIDFromContext(ctx)
			out = append(out, row)
			continue
		}
		if strings.TrimSpace(wechatPSID) == "" {
			slog.InfoContext(ctx, "admin_profit_sharing_refresh_not_submitted",
				"level", "info",
				"ps_id", formatID(id),
			)
			row["wechat_state"] = wechatProfitSharingStateNotSubmitted
			out = append(out, row)
			continue
		}
		txnID, err := lookupWechatTransactionIDForOrder(orderID)
		if err != nil {
			slog.InfoContext(ctx, "admin_profit_sharing_refresh_missing_txn",
				"level", "info",
				"ps_id", formatID(id),
				"order_id", formatID(orderID),
			)
			row["wechat_error"] = "缺少微信支付单号，无法同步"
			row["wechat_error_trace_id"] = tracelog.TraceIDFromContext(ctx)
			out = append(out, row)
			continue
		}
		started := time.Now()
		slog.DebugContext(ctx, "admin_profit_sharing_refresh_query_begin",
			"level", "debug",
			"ps_id", formatID(id),
		)
		state, err := queryProfitSharingOrder(ctx, outNo, txnID)
		slog.InfoContext(ctx, "admin_profit_sharing_refresh_query_order",
			"level", "info",
			"ps_id", formatID(id),
			"duration_ms", time.Since(started).Milliseconds(),
			"ok", err == nil,
		)
		if err != nil {
			slog.WarnContext(ctx, "admin_profit_sharing_refresh_query_failed",
				"level", "warn",
				"ps_id", formatID(id),
				"error", humanizeProfitSharingQueryError(err),
				"trace_id", tracelog.TraceIDFromContext(ctx),
			)
			row["wechat_error"] = humanizeProfitSharingQueryError(err)
			row["wechat_error_trace_id"] = tracelog.TraceIDFromContext(ctx)
			out = append(out, row)
			continue
		}
		row["wechat_state"] = state
		out = append(out, row)
	}
	return out
}

func humanizeProfitSharingQueryError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if strings.Contains(s, "FREQUENCY_LIMITED") || strings.Contains(s, "频率") {
		return "微信查询频率受限，请稍后重试"
	}
	if strings.Contains(s, "transaction_id") && strings.Contains(s, "not found") {
		return "缺少微信支付单号，无法同步"
	}
	if strings.Contains(s, "RESOURCE_NOT_EXISTS") || strings.Contains(s, "HTTP 404") {
		return "微信侧未找到分账单"
	}
	return "查询微信分账状态失败，请稍后重试"
}
