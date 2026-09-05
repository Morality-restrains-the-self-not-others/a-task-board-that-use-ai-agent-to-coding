package main

import (
	"context"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

func captureWechatPrepaySettle(t *testing.T, payerUserID string) native.PrepayRequest {
	t.Helper()
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origMode, origLiveOK := wechatCfg.Mode, wechatLiveOK
	origAppid, origMchid, origNotify := wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL
	origCall := wechatPrepayCall
	t.Cleanup(func() {
		wechatCfg.Mode, wechatLiveOK = origMode, origLiveOK
		wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL = origAppid, origMchid, origNotify
		wechatPrepayCall = origCall
	})
	wechatCfg.Mode = "live"
	wechatLiveOK = true
	wechatCfg.Appid = "wx-test-appid"
	wechatCfg.Mchid = "1900000109"
	wechatCfg.NotifyURL = "https://example.com/wechat/notify"

	var got native.PrepayRequest
	codeURL := "weixin://wxpay/bizpayurl?pr=ps-flag"
	wechatPrepayCall = func(ctx context.Context, svc *native.NativeApiService, req native.PrepayRequest) (*native.PrepayResponse, *core.APIResult, error) {
		got = req
		return &native.PrepayResponse{CodeUrl: &codeURL}, nil, nil
	}

	if _, _, err := wechatPrepay(context.Background(), 1, payerUserID, 100, "资源购买-分账标识", "https://example.com/wechat/notify"); err != nil {
		t.Fatalf("wechatPrepay: %v", err)
	}
	return got
}

func prepayProfitSharingTrue(req native.PrepayRequest) bool {
	return req.SettleInfo != nil && req.SettleInfo.ProfitSharing != nil && *req.SettleInfo.ProfitSharing
}

func TestWechatPrepaySetsProfitSharingWhenPaytimeQualified(t *testing.T) {
	payer := formatID(generateSnowflakeID())
	referrer := formatID(generateSnowflakeID())
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	if err := upsertReferralEdge(referrer, payer, generateSnowflakeID(), utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool {
		return referrerUserID == referrer
	})

	origMode, origLiveOK := wechatCfg.Mode, wechatLiveOK
	origAppid, origMchid, origNotify := wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL
	origCall := wechatPrepayCall
	t.Cleanup(func() {
		wechatCfg.Mode, wechatLiveOK = origMode, origLiveOK
		wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL = origAppid, origMchid, origNotify
		wechatPrepayCall = origCall
	})
	wechatCfg.Mode, wechatLiveOK = "live", true
	wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL = "wx-test-appid", "1900000109", "https://example.com/wechat/notify"
	var got native.PrepayRequest
	codeURL := "weixin://wxpay/bizpayurl?pr=ps-yes"
	wechatPrepayCall = func(ctx context.Context, svc *native.NativeApiService, req native.PrepayRequest) (*native.PrepayResponse, *core.APIResult, error) {
		got = req
		return &native.PrepayResponse{CodeUrl: &codeURL}, nil, nil
	}
	if _, _, err := wechatPrepay(context.Background(), 1, payer, 100, "资源购买-分账标识", "https://example.com/wechat/notify"); err != nil {
		t.Fatalf("prepay: %v", err)
	}
	if !prepayProfitSharingTrue(got) {
		t.Fatalf("want SettleInfo.ProfitSharing=true, got %+v", got.SettleInfo)
	}
}

func TestWechatPrepaySetsProfitSharingWithoutReferrer(t *testing.T) {
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool {
		t.Fatal("prepay must not query qualification to set the WeChat flag")
		return true
	})
	got := captureWechatPrepaySettle(t, formatID(generateSnowflakeID()))
	if !prepayProfitSharingTrue(got) {
		t.Fatalf("no referrer must still set profit_sharing, got %+v", got.SettleInfo)
	}
}

func TestWechatPrepaySetsProfitSharingDespiteIneligibleBindSnapshot(t *testing.T) {
	payer := formatID(generateSnowflakeID())
	referrer := formatID(generateSnowflakeID())
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	if err := upsertReferralEdge(referrer, payer, generateSnowflakeID(), utcNow(), "CH", false); err != nil {
		t.Fatal(err)
	}
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool {
		return referrerUserID == referrer
	})

	origMode, origLiveOK := wechatCfg.Mode, wechatLiveOK
	origAppid, origMchid, origNotify := wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL
	origCall := wechatPrepayCall
	t.Cleanup(func() {
		wechatCfg.Mode, wechatLiveOK = origMode, origLiveOK
		wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL = origAppid, origMchid, origNotify
		wechatPrepayCall = origCall
	})
	wechatCfg.Mode, wechatLiveOK = "live", true
	wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL = "wx-test-appid", "1900000109", "https://example.com/wechat/notify"
	var got native.PrepayRequest
	codeURL := "weixin://wxpay/bizpayurl?pr=ps-late-qual"
	wechatPrepayCall = func(ctx context.Context, svc *native.NativeApiService, req native.PrepayRequest) (*native.PrepayResponse, *core.APIResult, error) {
		got = req
		return &native.PrepayResponse{CodeUrl: &codeURL}, nil, nil
	}
	if _, _, err := wechatPrepay(context.Background(), 1, payer, 55, "资源购买-后获资格", "https://example.com/wechat/notify"); err != nil {
		t.Fatalf("prepay: %v", err)
	}
	if !prepayProfitSharingTrue(got) {
		t.Fatal("bind-time ineligible + paytime qualified must still set profit_sharing")
	}
}

func TestWechatPrepaySetsProfitSharingWhenPaytimeUnqualified(t *testing.T) {
	payer := formatID(generateSnowflakeID())
	referrer := formatID(generateSnowflakeID())
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	if err := upsertReferralEdge(referrer, payer, generateSnowflakeID(), utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool {
		t.Fatal("prepay must not query qualification to set the WeChat flag")
		return false
	})

	origMode, origLiveOK := wechatCfg.Mode, wechatLiveOK
	origAppid, origMchid, origNotify := wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL
	origCall := wechatPrepayCall
	t.Cleanup(func() {
		wechatCfg.Mode, wechatLiveOK = origMode, origLiveOK
		wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL = origAppid, origMchid, origNotify
		wechatPrepayCall = origCall
	})
	wechatCfg.Mode, wechatLiveOK = "live", true
	wechatCfg.Appid, wechatCfg.Mchid, wechatCfg.NotifyURL = "wx-test-appid", "1900000109", "https://example.com/wechat/notify"
	var got native.PrepayRequest
	codeURL := "weixin://wxpay/bizpayurl?pr=ps-no"
	wechatPrepayCall = func(ctx context.Context, svc *native.NativeApiService, req native.PrepayRequest) (*native.PrepayResponse, *core.APIResult, error) {
		got = req
		return &native.PrepayResponse{CodeUrl: &codeURL}, nil, nil
	}
	if _, _, err := wechatPrepay(context.Background(), 1, payer, 100, "资源购买-无资格", "https://example.com/wechat/notify"); err != nil {
		t.Fatalf("prepay: %v", err)
	}
	if !prepayProfitSharingTrue(got) {
		t.Fatal("paytime unqualified must still set profit_sharing")
	}
}
