package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// providerRefundRequest carries ledger allocation details for channel refunds.
type providerRefundRequest struct {
	Channel        string
	ProviderRef    string
	CaptureID      string
	Currency       string
	RefundPoints   int64
	OriginalPoints int64
	AmountMinor    int64
	// OutRefundNo is the deterministic merchant refund id (WeChat out_refund_no /
	// PayPal PayPal-Request-Id). Empty → generate a new snowflake (non-retryable path).
	OutRefundNo string
}

// executeProviderRefundFn is replaceable in tests.
var executeProviderRefundFn = executeProviderRefund

func providerRefundForceMock() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("TASKBILL_REFUND_PROVIDER")))
	return v == "mock" || v == "1" || v == "true"
}

func executeProviderRefund(ctx context.Context, req providerRefundRequest) (refundRef string, err error) {
	req.Channel = strings.TrimSpace(strings.ToLower(req.Channel))
	req.ProviderRef = strings.TrimSpace(req.ProviderRef)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Channel == "" || req.ProviderRef == "" || req.RefundPoints <= 0 {
		return "", fmt.Errorf("invalid provider refund params")
	}
	refundMinor := proportionalAmountMinor(req.AmountMinor, req.OriginalPoints, req.RefundPoints)
	if refundMinor <= 0 {
		return "", fmt.Errorf("refund amount minor is zero")
	}

	if providerRefundForceMock() {
		return mockProviderRefund(ctx, req, refundMinor), nil
	}

	switch req.Channel {
	case "paypal":
		if !paypalConfigured() {
			return mockProviderRefund(ctx, req, refundMinor), nil
		}
		captureID, err := paypalResolveCaptureID(ctx, req.ProviderRef, req.CaptureID)
		if err != nil {
			return "", err
		}
		currency := req.Currency
		if currency == "" {
			currency = paypalCfg.Currency
		}
		id, err := paypalRefundCapture(ctx, captureID, refundMinor, currency, req.OutRefundNo)
		if err != nil {
			return "", err
		}
		slog.InfoContext(ctx, "provider_refund_paypal_ok",
			"level", "info",
			"provider_ref", req.ProviderRef,
			"capture_id", captureID,
			"refund_id", id,
			"amount_minor", refundMinor,
			"currency", currency,
		)
		return "paypal:" + id, nil
	case "wechat":
		if wechatIsMock() || !wechatLiveOK {
			return mockProviderRefund(ctx, req, refundMinor), nil
		}
		id, err := wechatCreateRefund(ctx, req.ProviderRef, refundMinor, req.AmountMinor, req.OutRefundNo)
		if err != nil {
			return "", err
		}
		slog.InfoContext(ctx, "provider_refund_wechat_ok",
			"level", "info",
			"provider_ref", req.ProviderRef,
			"refund_id", id,
			"refund_fen", refundMinor,
			"total_fen", req.AmountMinor,
		)
		return "wechat:" + id, nil
	default:
		return "", fmt.Errorf("unsupported refund channel %q", req.Channel)
	}
}

func mockProviderRefund(ctx context.Context, req providerRefundRequest, refundMinor int64) string {
	ref := fmt.Sprintf("mock:%s:%s:%d", req.Channel, req.ProviderRef, req.RefundPoints)
	slog.InfoContext(ctx, "provider_refund_mock_success",
		"level", "info",
		"channel", req.Channel,
		"provider_ref", req.ProviderRef,
		"points", req.RefundPoints,
		"amount_minor", refundMinor,
		"currency", req.Currency,
		"refund_ref", ref,
	)
	return ref
}
