package main

import (
	"context"
	"testing"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
)

func setOrderPaidAtHoursAgo(t *testing.T, orderID int64, hours int) {
	t.Helper()
	setOrderPaidAt(t, orderID, time.Now().UTC().Add(-time.Duration(hours)*time.Hour).Format(time.RFC3339))
}

func setupUnfreezeCompensateTest(t *testing.T, unfreezeCall func(context.Context, *profitsharing.OrdersApiService, profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error)) *int {
	t.Helper()
	origClient := wechatClient
	origCall := profitSharingUnfreezeOrderCall
	origMode := wechatCfg.Mode
	origLive := wechatLiveOK
	calls := 0
	wechatCfg.Mode = "live"
	wechatLiveOK = true
	wechatClient = &core.Client{}
	profitSharingUnfreezeOrderCall = func(ctx context.Context, svc *profitsharing.OrdersApiService, req profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		calls++
		if unfreezeCall != nil {
			return unfreezeCall(ctx, svc, req)
		}
		return nil, nil, nil
	}
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingUnfreezeOrderCall = origCall
		wechatCfg.Mode = origMode
		wechatLiveOK = origLive
	})
	return &calls
}

// OPT-20260823-042：无佣金行、paid 超过 10 分钟且 ledger 有 transaction_id → 补偿 Unfreeze。
func TestCompensateUnfreezeRetriesOrderWithoutCommissionRow(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 94021
	orderID := int64(3000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-UFC-1", "420000ufc0001")
	setOrderPaidAtHoursAgo(t, orderID, 1)

	calls := setupUnfreezeCompensateTest(t, nil)
	if err := compensateUnfreezeRemainderForPaidWechatOrders(time.Now()); err != nil {
		t.Fatalf("compensate: %v", err)
	}
	if *calls != 1 {
		t.Fatalf("unfreeze calls=%d want 1", *calls)
	}
}

// OPT-20260823-042：有佣金行不重试。
func TestCompensateUnfreezeSkipsOrderWithCommissionRow(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 94022
	orderID := int64(3000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-UFC-2", "420000ufc0002")
	setOrderPaidAtHoursAgo(t, orderID, 1)
	seedProfitSharingRecord(t, orderID, tenantID)

	calls := setupUnfreezeCompensateTest(t, func(context.Context, *profitsharing.OrdersApiService, profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		t.Fatal("有佣金行不得补偿解冻")
		return nil, nil, nil
	})
	if err := compensateUnfreezeRemainderForPaidWechatOrders(time.Now()); err != nil {
		t.Fatalf("compensate: %v", err)
	}
	if *calls != 0 {
		t.Fatalf("unfreeze calls=%d want 0", *calls)
	}
}

// OPT-20260823-042：paid 未超过 10 分钟（首次解冻窗口内）不重试，避免竞态。
func TestCompensateUnfreezeSkipsTooRecentPayment(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 94023
	orderID := int64(3000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-UFC-3", "420000ufc0003")
	setOrderPaidAtHoursAgo(t, orderID, 0) // 刚支付（少于 10 分钟）

	calls := setupUnfreezeCompensateTest(t, func(context.Context, *profitsharing.OrdersApiService, profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		t.Fatal("支付未满 10 分钟不得补偿解冻")
		return nil, nil, nil
	})
	if err := compensateUnfreezeRemainderForPaidWechatOrders(time.Now()); err != nil {
		t.Fatalf("compensate: %v", err)
	}
	if *calls != 0 {
		t.Fatalf("unfreeze calls=%d want 0", *calls)
	}
}
