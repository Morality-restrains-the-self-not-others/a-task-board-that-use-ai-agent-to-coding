package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

// OPT-20260823-034: 微信 Native 预下单失败若带 WeChat HTTP dump（Wechatpay-Signature
// 签名头），响应 JSON 不得把 dump 泄漏给浏览器，且应含可读中文提示。
func TestHandlePayOrderWechatPrepayFailureDoesNotLeakDump(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origKyc, origSms := cfg.KycGateMode, cfg.SmsGateMode
	origMode, origLive := wechatCfg.Mode, wechatLiveOK
	origAppid, origMchid, origNotify := wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL
	origCall := wechatPrepayCall
	t.Cleanup(func() {
		cfg.KycGateMode, cfg.SmsGateMode = origKyc, origSms
		wechatCfg.Mode, wechatLiveOK = origMode, origLive
		wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL = origAppid, origMchid, origNotify
		wechatPrepayCall = origCall
	})
	cfg.KycGateMode, cfg.SmsGateMode = "off", "off"
	wechatCfg.Mode, wechatLiveOK = "live", true
	wechatCfg.Appid = "wx-test-appid"
	wechatCfg.Mchid = "1900000109"
	wechatCfg.NotifyURL = "https://example.com/wechat/notify"

	wechatPrepayCall = func(ctx context.Context, svc *native.NativeApiService, req native.PrepayRequest) (*native.PrepayResponse, *core.APIResult, error) {
		return nil, nil, &core.APIError{
			StatusCode: 500,
			Code:       "SYSTEM_ERROR",
			Message:    "系统繁忙",
			Header:     http.Header{"Wechatpay-Signature": []string{"sig-leak-034"}},
		}
	}

	tenantID := int64(1001)
	orderID := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at)
		VALUES (?, ?, ?, 'pending', ?, ?)`,
		orderID, tenantID, fmt.Sprintf("ORD-LEAK-%d", orderID), 8800, utcNow()); err != nil {
		t.Fatalf("seed order: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/tenant/%d/billing/orders/%d/pay/", tenantID, orderID),
		bytes.NewBufferString(`{"payment_method":"wechat"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "10001")
	rec := httptest.NewRecorder()
	handlePayOrder(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("pay status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "Wechatpay-Signature") || strings.Contains(body, "sig-leak-034") {
		t.Fatalf("response leaked WeChat dump: %s", body)
	}
	if !strings.Contains(body, "微信支付下单失败") {
		t.Fatalf("want friendly prefix 微信支付下单失败, got %s", body)
	}
}
