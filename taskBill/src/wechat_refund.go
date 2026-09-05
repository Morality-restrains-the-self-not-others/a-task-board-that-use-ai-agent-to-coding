package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
)

func wechatCreateRefund(ctx context.Context, outTradeNo string, refundFen, totalFen int64, outRefundNo string) (refundID string, err error) {
	outTradeNo = strings.TrimSpace(outTradeNo)
	if outTradeNo == "" || refundFen <= 0 || totalFen <= 0 {
		return "", fmt.Errorf("invalid wechat refund params")
	}
	if refundFen > totalFen {
		return "", fmt.Errorf("wechat refund fen %d exceeds total %d", refundFen, totalFen)
	}
	if wechatClient == nil {
		return "", fmt.Errorf("wechat pay client not initialized")
	}
	outRefundNo = strings.TrimSpace(outRefundNo)
	if outRefundNo == "" {
		outRefundNo = fmt.Sprintf("rf%d", generateSnowflakeID())
	}
	svc := refunddomestic.RefundsApiService{Client: wechatClient}
	resp, result, err := svc.Create(ctx, refunddomestic.CreateRequest{
		OutTradeNo:  core.String(outTradeNo),
		OutRefundNo: core.String(outRefundNo),
		Reason:      core.String("租户积分退款"),
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(refundFen),
			Total:    core.Int64(totalFen),
			Currency: core.String("CNY"),
		},
	})
	if err != nil {
		status := 0
		if result != nil && result.Response != nil {
			status = result.Response.StatusCode
		}
		slog.WarnContext(ctx, "wechat_refund_failed",
			"level", "warn",
			"out_trade_no", outTradeNo,
			"out_refund_no", outRefundNo,
			"http_status", status,
			"error", err.Error(),
		)
		return "", wechatPayClientError(err)
	}
	if resp != nil && resp.RefundId != nil && strings.TrimSpace(*resp.RefundId) != "" {
		return *resp.RefundId, nil
	}
	if resp != nil && resp.OutRefundNo != nil {
		return *resp.OutRefundNo, nil
	}
	return outRefundNo, nil
}
