package main

// 推荐人本人待微信确认分账提示（OPT-20260823-043）：
//   - 仅统计本人 processing/finished 分账单中微信 receivers.result=PENDING 的笔数
//   - 无 PENDING → pending_count=0 且无 deadline
//   - 微信查询单笔失败跳过该笔，不阻塞整体

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
)

func TestReferrerProfitSharingPendingCountsPENDING(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origClient := wechatClient
	wechatLiveOK = true
	wechatClient = &core.Client{}
	t.Cleanup(func() { wechatLiveOK = origLive; wechatClient = origClient })

	const tenantID int64 = 95102
	paidAt10 := daysAgo(10)
	paidAt20 := daysAgo(20)
	seedReferrerOrder(t, tenantID, 4001, "referrer-1", psStatusFinished, paidAt10)
	seedReferrerOrder(t, tenantID, 4002, "referrer-1", psStatusFinished, paidAt20)
	seedReferrerOrder(t, tenantID, 4003, "referrer-2", psStatusFinished, paidAt10) // 他人记录不计

	var outNo4001, outNo4002 string
	if err := db.QueryRow(`SELECT out_profit_sharing_no FROM billing_profit_sharing WHERE order_id=?`, int64(4001)).Scan(&outNo4001); err != nil {
		t.Fatalf("load out_no 4001: %v", err)
	}
	if err := db.QueryRow(`SELECT out_profit_sharing_no FROM billing_profit_sharing WHERE order_id=?`, int64(4002)).Scan(&outNo4002); err != nil {
		t.Fatalf("load out_no 4002: %v", err)
	}

	prev := profitSharingQueryOrderCall
	pending := profitsharing.DETAILSTATUS_PENDING
	success := profitsharing.DETAILSTATUS_SUCCESS
	st := profitsharing.ORDERSTATUS_FINISHED
	profitSharingQueryOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		recs := []profitsharing.OrderReceiverDetail{{Account: core.String("openid-x"), Result: &success}}
		if req.OutOrderNo != nil && *req.OutOrderNo == outNo4001 {
			recs = []profitsharing.OrderReceiverDetail{{Account: core.String("openid-x"), Result: &pending}}
		}
		return &profitsharing.OrdersEntity{State: &st, Receivers: recs}, nil, nil
	}
	t.Cleanup(func() { profitSharingQueryOrderCall = prev })

	// 无身份 → 401
	rec := httptest.NewRecorder()
	handleReferrerProfitSharingPending(rec, referrerReq(http.MethodGet, "/api/billing/profit-sharing/referrer-pending/", ""))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no identity: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// referrer-1：仅 4001 的接收方 PENDING → pending_count=1，deadline=paidAt10+30d
	rec = httptest.NewRecorder()
	handleReferrerProfitSharingPending(rec, referrerReq(http.MethodGet, "/api/billing/profit-sharing/referrer-pending/", "referrer-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("pending: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "openid") {
		t.Fatalf("response leaked openid: %s", rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	if got := int(body["pending_count"].(float64)); got != 1 {
		t.Fatalf("pending_count=%d want 1", got)
	}
	paid, parseErr := time.Parse(time.RFC3339, paidAt10)
	if parseErr != nil {
		t.Fatalf("parse paidAt10: %v", parseErr)
	}
	wantDeadline := paid.Add(30 * 24 * time.Hour).Format("2006-01-02")
	if body["deadline"] != wantDeadline {
		t.Fatalf("deadline=%v want %s", body["deadline"], wantDeadline)
	}

	// 他人 referrer-2：record 4003 的 out_no != 4001 → SUCCESS → pending_count=0
	rec = httptest.NewRecorder()
	handleReferrerProfitSharingPending(rec, referrerReq(http.MethodGet, "/api/billing/profit-sharing/referrer-pending/", "referrer-2"))
	if rec.Code != http.StatusOK {
		t.Fatalf("pending other: status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = decodeJSONMap(t, rec)
	if got := int(body["pending_count"].(float64)); got != 0 {
		t.Fatalf("other pending_count=%d want 0", got)
	}
	if _, has := body["deadline"]; has {
		t.Fatalf("other deadline should be absent: %v", body["deadline"])
	}
	_ = outNo4002
}

func TestReferrerProfitSharingPendingSkipsQueryFailure(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origClient := wechatClient
	wechatLiveOK = true
	wechatClient = &core.Client{}
	t.Cleanup(func() { wechatLiveOK = origLive; wechatClient = origClient })

	const tenantID int64 = 95103
	seedReferrerOrder(t, tenantID, 4101, "referrer-1", psStatusFinished, daysAgo(10))

	prev := profitSharingQueryOrderCall
	profitSharingQueryOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, _ profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		return nil, nil, fmt.Errorf("FREQUENCY_LIMITED")
	}
	t.Cleanup(func() { profitSharingQueryOrderCall = prev })

	rec := httptest.NewRecorder()
	handleReferrerProfitSharingPending(rec, referrerReq(http.MethodGet, "/api/billing/profit-sharing/referrer-pending/", "referrer-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	if got := int(body["pending_count"].(float64)); got != 0 {
		t.Fatalf("pending_count=%d want 0 on query failure", got)
	}
}
