package main

import (
	"context"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
)

func TestUnfreezeRemainingWhenNoProfitSharingRow(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 94011
	orderID := int64(3000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-UF-1", "42000055556666")

	origClient := wechatClient
	origCall := profitSharingUnfreezeOrderCall
	origMode := wechatCfg.Mode
	origLive := wechatLiveOK
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingUnfreezeOrderCall = origCall
		wechatCfg.Mode = origMode
		wechatLiveOK = origLive
	})
	wechatCfg.Mode = "live"
	wechatLiveOK = true
	wechatClient = &core.Client{}

	var got profitsharing.UnfreezeOrderRequest
	calls := 0
	profitSharingUnfreezeOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		calls++
		got = req
		return nil, nil, nil
	}

	maybeUnfreezeWechatRemainderIfNoReceiver(context.Background(), orderID, "wechat")
	if calls != 1 {
		t.Fatalf("unfreeze calls=%d want 1", calls)
	}
	wantNo := wechatUnfreezeOutOrderNo(orderID)
	if got.OutOrderNo == nil || *got.OutOrderNo != wantNo {
		t.Fatalf("out_order_no=%v want %s", got.OutOrderNo, wantNo)
	}
	if got.TransactionId == nil || *got.TransactionId != "42000055556666" {
		t.Fatalf("transaction_id=%v", got.TransactionId)
	}
}

func TestUnfreezeRemainingSkippedWhenProfitSharingRowExists(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 94012
	orderID := int64(3000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-UF-2", "42000077778888")
	seedProfitSharingRecord(t, orderID, tenantID)

	origCall := profitSharingUnfreezeOrderCall
	origClient := wechatClient
	origMode := wechatCfg.Mode
	origLive := wechatLiveOK
	t.Cleanup(func() {
		profitSharingUnfreezeOrderCall = origCall
		wechatClient = origClient
		wechatCfg.Mode = origMode
		wechatLiveOK = origLive
	})
	wechatCfg.Mode = "live"
	wechatLiveOK = true
	wechatClient = &core.Client{}
	profitSharingUnfreezeOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		t.Fatal("must not unfreeze when commission row exists")
		return nil, nil, nil
	}

	maybeUnfreezeWechatRemainderIfNoReceiver(context.Background(), orderID, "wechat")
}

func TestUnfreezeRemainingSkippedForMockWechat(t *testing.T) {
	origMode := wechatCfg.Mode
	origCall := profitSharingUnfreezeOrderCall
	t.Cleanup(func() {
		wechatCfg.Mode = origMode
		profitSharingUnfreezeOrderCall = origCall
	})
	wechatCfg.Mode = "mock"
	profitSharingUnfreezeOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		t.Fatal("mock must not call WeChat unfreeze")
		return nil, nil, nil
	}
	maybeUnfreezeWechatRemainderIfNoReceiver(context.Background(), 1, "wechat")
}

func TestUnfreezeRemainingSkippedWhenClientNil(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 94013
	orderID := int64(3000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-UF-3", "42000099990000")

	origClient := wechatClient
	origCall := profitSharingUnfreezeOrderCall
	origMode := wechatCfg.Mode
	origLive := wechatLiveOK
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingUnfreezeOrderCall = origCall
		wechatCfg.Mode = origMode
		wechatLiveOK = origLive
	})
	wechatCfg.Mode = "live"
	wechatLiveOK = true
	wechatClient = nil
	profitSharingUnfreezeOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		t.Fatal("nil wechat client must not call unfreeze")
		return nil, nil, nil
	}
	maybeUnfreezeWechatRemainderIfNoReceiver(context.Background(), orderID, "wechat")
}

func TestLookupWechatTransactionIDOnDBNil(t *testing.T) {
	if _, err := lookupWechatTransactionIDOnDB(nil, 1); err == nil {
		t.Fatal("nil db must error")
	}
}

