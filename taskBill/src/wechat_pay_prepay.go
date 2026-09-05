package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"

	"tracelog"
)

// wechatPrepayCall 是 Native 预下单的可注入接缝：默认走 SDK svc.Prepay，
// 单测通过替换该变量注入失败 Prepay（OPT-20260817-031）。
var wechatPrepayCall = func(ctx context.Context, svc *native.NativeApiService, req native.PrepayRequest) (*native.PrepayResponse, *core.APIResult, error) {
	return svc.Prepay(ctx, req)
}

// logWechatPrepayFailure 在 live 预下单失败时输出结构化 JSON 日志，字段含 appid / mchid /
// out_trade_no / amount_fen / 微信错误码，便于按 Loki `{job="task-bill"} | json | msg=~"wechat.*prepay"`
// 定位 APPID_MCHID_NOT_MATCH 等绑定类错误。禁止写入 api_v3_key / 私钥。
func logWechatPrepayFailure(ctx context.Context, outTradeNo string, amountFen int64, prepayErr error) {
	wechatCode, wechatMsg := "", ""
	if apiErr, ok := prepayErr.(*core.APIError); ok {
		wechatCode = apiErr.Code
		wechatMsg = apiErr.Message
	}
	fields := map[string]string{
		"appid":        wechatCfg.Appid,
		"mchid":        wechatCfg.Mchid,
		"out_trade_no": outTradeNo,
		"amount_fen":   fmt.Sprintf("%d", amountFen),
		"err_code":     wechatCode,
		"err_msg":      wechatMsg,
	}
	if prepayErr != nil {
		fields["err"] = truncateBytes([]byte(prepayErr.Error()), 500)
	}
	tracelog.EmitWithTrace(
		tracelog.TraceIDFromContext(ctx),
		"error",
		"wechat prepay failed",
		"wechat_pay",
		fields,
	)
}

// wechatPrepay Native 预下单。amountFen 为订单金额（分），原样写入 amount.total。
func wechatPrepay(ctx context.Context, tenantID int64, userID string, amountFen int64, description string, notifyURL string) (outTradeNo, codeURL string, err error) {
	if _, err := wechatNativeAmountFen(amountFen); err != nil {
		return "", "", err
	}
	if !wechatConfigured() {
		return "", "", fmt.Errorf("wechat pay not configured")
	}
	outTradeNo = fmt.Sprintf("WX%d", generateSnowflakeID())
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = fmt.Sprintf("%s %s元", wechatCfg.DescriptionPrefix, centsToYuanStr(amountFen))
	}
	if len(desc) > 120 {
		desc = desc[:120]
	}
	pending := wechatPendingOrder{
		OutTradeNo:  outTradeNo,
		TenantID:    tenantID,
		UserID:      userID,
		AmountYuan:  amountFen / 100,
		AmountFen:   amountFen,
		Description: desc,
		CreatedAt:   time.Now().UTC(),
	}
	storeWechatPending(pending)

	if wechatIsMock() {
		codeURL = "weixin://wxpay/bizpayurl?pr=" + outTradeNo
		log.Printf("[taskBill] wechat mock prepay out_trade_no=%s tenant=%d amount_fen=%d", outTradeNo, tenantID, amountFen)
		return outTradeNo, codeURL, nil
	}

	nu := strings.TrimSpace(notifyURL)
	if nu == "" {
		nu = strings.TrimSpace(wechatCfg.NotifyURL)
	}
	if nu == "" {
		return "", "", fmt.Errorf("notify_url required for live mode")
	}
	attach := fmt.Sprintf("tenant:%d:user:%s", tenantID, userID)
	svc := native.NativeApiService{Client: wechatClient}
	prepayReq := native.PrepayRequest{
		Appid:       core.String(wechatCfg.Appid),
		Mchid:       core.String(wechatCfg.Mchid),
		Description: core.String(desc),
		OutTradeNo:  core.String(outTradeNo),
		Attach:      core.String(attach),
		NotifyUrl:   core.String(nu),
		Amount: &native.Amount{
			Total:    core.Int64(amountFen),
			Currency: core.String("CNY"),
		},
		// ADR-0040: 租户微信订单一律打分账标识，无论是否有推荐人。
		SettleInfo: &native.SettleInfo{ProfitSharing: core.Bool(true)},
	}
	resp, _, err := wechatPrepayCall(ctx, &svc, prepayReq)
	if err != nil {
		logWechatPrepayFailure(ctx, outTradeNo, amountFen, err)
		return "", "", err
	}
	if resp == nil || resp.CodeUrl == nil || *resp.CodeUrl == "" {
		logWechatPrepayFailure(ctx, outTradeNo, amountFen, fmt.Errorf("empty code_url from wechat"))
		return "", "", fmt.Errorf("empty code_url from wechat")
	}
	flagged := prepayReq.SettleInfo != nil && prepayReq.SettleInfo.ProfitSharing != nil && *prepayReq.SettleInfo.ProfitSharing
	tracelog.EmitWithTrace(
		tracelog.TraceIDFromContext(ctx),
		"info",
		"wechat prepay ok",
		"wechat_pay",
		map[string]string{
			"out_trade_no":   outTradeNo,
			"amount_fen":     fmt.Sprintf("%d", amountFen),
			"tenant_id":      fmt.Sprintf("%d", tenantID),
			"profit_sharing": fmt.Sprintf("%t", flagged),
		},
	)
	log.Printf("[taskBill] wechat live prepay out_trade_no=%s tenant=%d amount_fen=%d", outTradeNo, tenantID, amountFen)
	return outTradeNo, *resp.CodeUrl, nil
}
