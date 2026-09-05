package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
)

// —— OPT-20260822-053：分账 live 路径改走 sdk/wechatpay-go profitsharing service ——
// 这些用例替换 SDK 可注入接缝，验证请求映射 / 响应解析 / 错误格式化，
// 不真正打到微信（wechatClient 用 dummy 值，接缝替换后不会触碰 Client 字段）。

func TestCreateProfitSharingOrderSDK_RequestMappingAndOrderID(t *testing.T) {
	origLive := wechatLiveOK
	origCfg := wechatCfg
	origClient := wechatClient
	origCall := profitSharingCreateOrderCall
	wechatLiveOK = true
	wechatCfg.Appid = "wx_pay_app"
	wechatClient = &core.Client{}
	defer func() {
		wechatLiveOK = origLive
		wechatCfg = origCfg
		wechatClient = origClient
		profitSharingCreateOrderCall = origCall
	}()

	var gotReq profitsharing.CreateOrderRequest
	profitSharingCreateOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.CreateOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		gotReq = req
		return &profitsharing.OrdersEntity{OrderId: core.String("ps_order_123")}, nil, nil
	}

	orderID, err := createProfitSharingOrderImpl(context.Background(),
		"out-order-ignored", "42000033334444", "PS20260822", []profitSharingReceiver{
			{Account: "o-referrer", Amount: 500, Description: "推荐佣金"},
		})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if orderID != "ps_order_123" {
		t.Fatalf("order_id=%q want ps_order_123", orderID)
	}
	if gotReq.Appid == nil || *gotReq.Appid != "wx_pay_app" {
		t.Fatalf("appid=%v", gotReq.Appid)
	}
	if gotReq.TransactionId == nil || *gotReq.TransactionId != "42000033334444" {
		t.Fatalf("transaction_id=%v", gotReq.TransactionId)
	}
	if gotReq.OutOrderNo == nil || *gotReq.OutOrderNo != "PS20260822" {
		t.Fatalf("out_order_no=%v", gotReq.OutOrderNo)
	}
	if gotReq.UnfreezeUnsplit == nil || !*gotReq.UnfreezeUnsplit {
		t.Fatal("unfreeze_unsplit must be true")
	}
	if len(gotReq.Receivers) != 1 {
		t.Fatalf("receivers len=%d", len(gotReq.Receivers))
	}
	r := gotReq.Receivers[0]
	if r.Type == nil || *r.Type != "PERSONAL_OPENID" {
		t.Fatalf("receiver type=%v", r.Type)
	}
	if r.Account == nil || *r.Account != "o-referrer" {
		t.Fatalf("receiver account=%v", r.Account)
	}
	if r.Amount == nil || *r.Amount != 500 {
		t.Fatalf("receiver amount=%v", r.Amount)
	}
	if r.Description == nil || *r.Description != "推荐佣金" {
		t.Fatalf("receiver description=%v", r.Description)
	}
}

func TestCreateProfitSharingOrderSDK_FailsClosedWhenClientNil(t *testing.T) {
	origLive := wechatLiveOK
	origClient := wechatClient
	wechatLiveOK = true
	wechatClient = nil
	defer func() {
		wechatLiveOK = origLive
		wechatClient = origClient
	}()

	_, err := createProfitSharingOrderImpl(context.Background(), "o", "t", "ps", nil)
	if err == nil || !strings.Contains(err.Error(), "client not initialized") {
		t.Fatalf("err=%v want fail-closed client not initialized", err)
	}
}

func TestCreateProfitSharingOrderSDK_FormatsAPIError(t *testing.T) {
	origLive := wechatLiveOK
	origCfg := wechatCfg
	origClient := wechatClient
	origCall := profitSharingCreateOrderCall
	wechatLiveOK = true
	wechatCfg.Appid = "wx_pay_app"
	wechatClient = &core.Client{}
	defer func() {
		wechatLiveOK = origLive
		wechatCfg = origCfg
		wechatClient = origClient
		profitSharingCreateOrderCall = origCall
	}()

	profitSharingCreateOrderCall = func(context.Context, *profitsharing.OrdersApiService, profitsharing.CreateOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		return nil, nil, &core.APIError{StatusCode: http.StatusBadRequest, Code: "INVALID_REQUEST", Message: "金额超限", Body: `{"code":"INVALID_REQUEST"}`}
	}

	_, err := createProfitSharingOrderImpl(context.Background(), "o", "t", "ps", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"HTTP 400", "INVALID_REQUEST"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err=%q want contains %q", err.Error(), want)
		}
	}
}

func TestQueryProfitSharingOrderSDK_ExtractsState(t *testing.T) {
	origClient := wechatClient
	origCall := profitSharingQueryOrderCall
	wechatClient = &core.Client{}
	defer func() {
		wechatClient = origClient
		profitSharingQueryOrderCall = origCall
	}()

	profitSharingQueryOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		if req.OutOrderNo == nil || *req.OutOrderNo != "PS20260822" {
			t.Fatalf("out_order_no=%v", req.OutOrderNo)
		}
		return &profitsharing.OrdersEntity{State: profitsharing.OrderStatus("FINISHED").Ptr()}, nil, nil
	}

	state, err := queryProfitSharingOrder(context.Background(), "PS20260822", "42000033334444")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if state != "FINISHED" {
		t.Fatalf("state=%q want FINISHED", state)
	}
}

func TestUnfreezeProfitSharingSDK_RequestMapping(t *testing.T) {
	origClient := wechatClient
	origCall := profitSharingUnfreezeOrderCall
	wechatClient = &core.Client{}
	defer func() {
		wechatClient = origClient
		profitSharingUnfreezeOrderCall = origCall
	}()

	var gotReq profitsharing.UnfreezeOrderRequest
	profitSharingUnfreezeOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		gotReq = req
		return nil, nil, nil
	}

	if err := unfreezeProfitSharing(context.Background(), "PS20260822", "42000033334444", "解冻"); err != nil {
		t.Fatalf("unfreeze: %v", err)
	}
	if gotReq.Description == nil || *gotReq.Description != "解冻" {
		t.Fatalf("description=%v", gotReq.Description)
	}
	if gotReq.TransactionId == nil || *gotReq.TransactionId != "42000033334444" {
		t.Fatalf("transaction_id=%v", gotReq.TransactionId)
	}
}

func TestDeleteProfitSharingReceiverSDK_RequestMapping(t *testing.T) {
	origLive := wechatLiveOK
	origClient := wechatClient
	origCall := profitSharingDeleteReceiverCall
	wechatLiveOK = true
	wechatClient = &core.Client{}
	defer func() {
		wechatLiveOK = origLive
		wechatClient = origClient
		profitSharingDeleteReceiverCall = origCall
	}()

	var gotReq profitsharing.DeleteReceiverRequest
	profitSharingDeleteReceiverCall = func(_ context.Context, _ *profitsharing.ReceiversApiService, req profitsharing.DeleteReceiverRequest) (*profitsharing.DeleteReceiverResponse, *core.APIResult, error) {
		gotReq = req
		return nil, nil, nil
	}

	if err := deleteProfitSharingReceiver(context.Background(), "wx_app", "o-referrer"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if gotReq.Appid == nil || *gotReq.Appid != "wx_app" {
		t.Fatalf("appid=%v", gotReq.Appid)
	}
	if gotReq.Type == nil || string(*gotReq.Type) != "PERSONAL_OPENID" {
		t.Fatalf("type=%v", gotReq.Type)
	}
	if gotReq.Account == nil || *gotReq.Account != "o-referrer" {
		t.Fatalf("account=%v", gotReq.Account)
	}
}

func TestPostProfitSharingReceiverLiveSDK_UsesIdentityAppID(t *testing.T) {
	origClient := wechatClient
	origCall := profitSharingAddReceiverCall
	wechatClient = &core.Client{}
	defer func() {
		wechatClient = origClient
		profitSharingAddReceiverCall = origCall
	}()

	var gotReq profitsharing.AddReceiverRequest
	profitSharingAddReceiverCall = func(_ context.Context, _ *profitsharing.ReceiversApiService, req profitsharing.AddReceiverRequest) (*profitsharing.AddReceiverResponse, *core.APIResult, error) {
		gotReq = req
		return nil, nil, nil
	}

	err := postProfitSharingReceiverLive(context.Background(), wechatPayIdentity{
		AppID: "wx_login_app", OpenID: "o-referrer",
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if gotReq.Appid == nil || *gotReq.Appid != "wx_login_app" {
		t.Fatalf("appid=%v want login appid (not pay mch appid)", gotReq.Appid)
	}
	if gotReq.Type == nil || string(*gotReq.Type) != "PERSONAL_OPENID" {
		t.Fatalf("type=%v", gotReq.Type)
	}
	if gotReq.RelationType == nil || string(*gotReq.RelationType) != "DISTRIBUTOR" {
		t.Fatalf("relation_type=%v", gotReq.RelationType)
	}
}

func TestWechatV3GetAllowErrorSDK_MerchantRatioRoutesThroughSDK(t *testing.T) {
	origClient := wechatClient
	origCall := profitSharingMerchantRatioCall
	wechatClient = &core.Client{}
	defer func() {
		wechatClient = origClient
		profitSharingMerchantRatioCall = origCall
	}()

	var gotReq profitsharing.QueryMerchantRatioRequest
	profitSharingMerchantRatioCall = func(_ context.Context, _ *profitsharing.MerchantsApiService, req profitsharing.QueryMerchantRatioRequest) (*profitsharing.QueryMerchantRatioResponse, *core.APIResult, error) {
		gotReq = req
		return &profitsharing.QueryMerchantRatioResponse{SubMchid: core.String("1114728879"), MaxRatio: core.Int64(1500)}, nil, nil
	}

	status, body, err := wechatV3GetAllowError(context.Background(), "/v3/profitsharing/merchant-configs/1114728879")
	if err != nil {
		t.Fatalf("ratio: %v", err)
	}
	if gotReq.SubMchid == nil || *gotReq.SubMchid != "1114728879" {
		t.Fatalf("sub_mchid=%v", gotReq.SubMchid)
	}
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if !strings.Contains(string(body), `"max_ratio":1500`) {
		t.Fatalf("body=%s want max_ratio:1500", string(body))
	}
}

func TestWechatV3GetAllowErrorSDK_APIErrorBecomesBody(t *testing.T) {
	origClient := wechatClient
	origCall := profitSharingMerchantRatioCall
	wechatClient = &core.Client{}
	defer func() {
		wechatClient = origClient
		profitSharingMerchantRatioCall = origCall
	}()

	profitSharingMerchantRatioCall = func(context.Context, *profitsharing.MerchantsApiService, profitsharing.QueryMerchantRatioRequest) (*profitsharing.QueryMerchantRatioResponse, *core.APIResult, error) {
		return nil, nil, &core.APIError{StatusCode: http.StatusBadRequest, Code: "INVALID_REQUEST", Message: "直连商户不可用", Body: `{"code":"INVALID_REQUEST"}`}
	}

	status, body, err := wechatV3GetAllowError(context.Background(), "/v3/profitsharing/merchant-configs/1114728879")
	if err != nil {
		t.Fatalf("ratio err: %v", err)
	}
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d", status)
	}
	if !strings.Contains(string(body), "INVALID_REQUEST") {
		t.Fatalf("body=%s want code INVALID_REQUEST", string(body))
	}
}

func TestWechatV3GetAllowErrorSDK_RejectsInvalidPath(t *testing.T) {
	_, _, err := wechatV3GetAllowError(context.Background(), "/v3/profitsharing/merchant-configs")
	if err == nil {
		t.Fatal("expected error for path without mchid")
	}
}

func TestWechatSDKResultError_WrapsNonAPIError(t *testing.T) {
	inner := errors.New("boom")
	err := wechatSDKResultError("create profit sharing", inner)
	if !strings.Contains(err.Error(), "create profit sharing failed") || !errors.Is(err, inner) {
		t.Fatalf("err=%v want wrap boom", err)
	}
}

func TestWechatSDKResultError_FormatsAPIError(t *testing.T) {
	err := wechatSDKResultError("create profit sharing", &core.APIError{
		StatusCode: http.StatusBadRequest,
		Code:       "INVALID_REQUEST",
		Message:    "金额超限",
		Body:       `{"code":"INVALID_REQUEST"}`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"HTTP 400", "INVALID_REQUEST"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err=%q want contains %q", err.Error(), want)
		}
	}
}
