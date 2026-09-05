package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
)

func TestWechatPayClientErrorNOT_ENOUGH(t *testing.T) {
	src := &core.APIError{
		StatusCode: 403,
		Code:       "NOT_ENOUGH",
		Message:    "基本账户余额不足，请充值后重新发起",
		Header:     http.Header{"Wechatpay-Signature": []string{"sig-secret"}},
	}
	err := wechatPayClientError(src)
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if strings.Contains(err.Error(), "Wechatpay-Signature") || strings.Contains(err.Error(), "sig-secret") {
		t.Fatalf("public error leaked dump: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "余额不足") {
		t.Fatalf("want 余额不足, got %s", err.Error())
	}
	status, msg := refundActionClientError(err)
	if status != http.StatusConflict {
		t.Fatalf("status=%d want 409", status)
	}
	if strings.Contains(msg, "Wechatpay-Signature") {
		t.Fatalf("handler message leaked dump: %s", msg)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("unwrap should still yield *core.APIError for logs")
	}
}

func TestHumanizeProviderErrorTextDump(t *testing.T) {
	dump := "error http response:[StatusCode: 403 Code: \"NOT_ENOUGH\"\nMessage: 基本账户余额不足，请充值后重新发起\nHeader:\n - Wechatpay-Signature=[abc]]"
	got := humanizeProviderErrorText(dump)
	if !strings.Contains(got, "余额不足") {
		t.Fatalf("got %s", got)
	}
	if strings.Contains(got, "Wechatpay-Signature") || strings.Contains(got, "abc") {
		t.Fatalf("leaked dump: %s", got)
	}
}

func TestRefundActionClientErrorNotPending(t *testing.T) {
	status, msg := refundActionClientError(errors.New("refund application is not pending"))
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d", status)
	}
	if msg != "refund application is not pending" {
		t.Fatalf("msg=%s", msg)
	}
}

// OPT-20260823-034: 非退款支付动作（下单/分账）同样不得把 WeChat dump 写进浏览器，
// 且措辞用「微信支付失败」而非「微信退款失败」。
func TestPaymentActionClientErrorStripsWechatDump(t *testing.T) {
	src := &core.APIError{
		StatusCode: 500,
		Code:       "SYSTEM_ERROR",
		Message:    "系统繁忙",
		Header:     http.Header{"Wechatpay-Signature": []string{"sig-secret-034"}},
	}
	_, msg := paymentActionClientError(src)
	if strings.Contains(msg, "Wechatpay-Signature") || strings.Contains(msg, "sig-secret-034") {
		t.Fatalf("payment action message leaked dump: %s", msg)
	}
	if strings.Contains(msg, "微信退款失败") {
		t.Fatalf("payment action should not say 退款: %s", msg)
	}
	if !strings.Contains(msg, "系统繁忙") {
		t.Fatalf("want detail 系统繁忙, got %s", msg)
	}
}

func TestPaymentActionClientErrorNOT_ENOUGH(t *testing.T) {
	src := &core.APIError{
		StatusCode: 403,
		Code:       "NOT_ENOUGH",
		Message:    "基本账户余额不足，请充值后重新发起",
		Header:     http.Header{"Wechatpay-Signature": []string{"sig-secret-034"}},
	}
	status, msg := paymentActionClientError(src)
	if status != http.StatusConflict {
		t.Fatalf("status=%d want 409", status)
	}
	if !strings.Contains(msg, "余额不足") {
		t.Fatalf("want 余额不足, got %s", msg)
	}
	if strings.Contains(msg, "Wechatpay-Signature") || strings.Contains(msg, "微信退款失败") {
		t.Fatalf("message not sanitized: %s", msg)
	}
}

// TestPaymentActionClientErrorBusinessErrorPassthrough 业务错误（非渠道 dump）原样返回，
// 不丢上下文。
func TestPaymentActionClientErrorBusinessErrorPassthrough(t *testing.T) {
	_, msg := paymentActionClientError(errors.New("notify_url required for live mode"))
	if msg != "notify_url required for live mode" {
		t.Fatalf("msg=%s", msg)
	}
}
