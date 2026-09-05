package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

// TestHandlePayOrderWechatDoesNotRoundSubYuanToOneYuan 回归 ORD-20260818-003：
// 任务帖 0.55 元（55 分）不得因「整数元」误假设向微信 Native 下单 1.00 元（100 分）。
// 官方 APIv3 Native 下单 amount.total 单位为分，必须大于 0；1 元应填 100。
func TestHandlePayOrderWechatDoesNotRoundSubYuanToOneYuan(t *testing.T) {
	gotFen := payOrderCaptureWechatFen(t, 55)
	if gotFen != 55 {
		t.Fatalf("WeChat amount.total=%d want 55 fen (0.55 元); 向上取整到整数元会下单 100 分", gotFen)
	}
}

func TestHandlePayOrderWechatIntegerYuanStillFen(t *testing.T) {
	gotFen := payOrderCaptureWechatFen(t, 800)
	if gotFen != 800 {
		t.Fatalf("WeChat amount.total=%d want 800 fen (8.00 元)", gotFen)
	}
}

func payOrderCaptureWechatFen(t *testing.T, orderCents int64) int64 {
	t.Helper()
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

	var gotFen int64 = -1
	codeURL := "weixin://wxpay/bizpayurl?pr=amount-fen"
	wechatPrepayCall = func(ctx context.Context, svc *native.NativeApiService, req native.PrepayRequest) (*native.PrepayResponse, *core.APIResult, error) {
		if req.Amount != nil && req.Amount.Total != nil {
			gotFen = *req.Amount.Total
		}
		return &native.PrepayResponse{CodeUrl: &codeURL}, nil, nil
	}

	tenantID := int64(1001)
	orderID := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at)
		VALUES (?, ?, ?, 'pending', ?, ?)`,
		orderID, tenantID, fmt.Sprintf("ORD-AMT-%d", orderID), orderCents, utcNow()); err != nil {
		t.Fatalf("seed order: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/tenant/%d/billing/orders/%d/pay/", tenantID, orderID),
		bytes.NewBufferString(`{"payment_method":"wechat"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "10001")
	rec := httptest.NewRecorder()
	handlePayOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pay status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gotFen < 0 {
		t.Fatal("wechat Prepay 未被调用")
	}
	return gotFen
}

func TestWechatNativeAmountFenKeepsOrderCents(t *testing.T) {
	fen, err := wechatNativeAmountFen(55)
	if err != nil {
		t.Fatalf("wechatNativeAmountFen(55): %v", err)
	}
	if fen != 55 {
		t.Fatalf("wechatNativeAmountFen(55)=%d want 55", fen)
	}
}

func TestWechatNativeAmountFenRejectsZero(t *testing.T) {
	if _, err := wechatNativeAmountFen(0); err == nil {
		t.Fatal("0 分应拒绝")
	}
}

func TestWechatNativeAmountFenRejectsOverMax(t *testing.T) {
	if _, err := wechatNativeAmountFen(wechatNativeMaxFen + 1); err == nil {
		t.Fatal("超过 4999 元应拒绝")
	}
}

func TestWechatKycAmountYuanCeilIndependentOfFen(t *testing.T) {
	if got := kycAmountYuanCeil(55); got != 1 {
		t.Fatalf("kycAmountYuanCeil(55)=%d want 1（限额接口仍按整数元上取整）", got)
	}
	if got := kycAmountYuanCeil(800); got != 8 {
		t.Fatalf("kycAmountYuanCeil(800)=%d want 8", got)
	}
}
