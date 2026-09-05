package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	native "github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

// setupQueryGuardTest 把查询兜底接到可注入接缝，并保存/恢复限流与熔断配置。
func setupQueryGuardTest(t *testing.T) {
	t.Helper()
	origClient := wechatClient
	origMchid := wechatCfg.Mchid
	origMode := wechatCfg.Mode
	origCall := wechatQueryOrderByIdCall
	origInterval := wechatQueryMinInterval
	origThreshold := wechatQueryCircuitThreshold
	wechatClient = &core.Client{}
	wechatCfg.Mchid = "1900000109"
	wechatCfg.Mode = "live"
	wechatQueryGuardInst.reset()
	t.Cleanup(func() {
		wechatClient = origClient
		wechatCfg.Mchid = origMchid
		wechatCfg.Mode = origMode
		wechatQueryOrderByIdCall = origCall
		wechatQueryMinInterval = origInterval
		wechatQueryCircuitThreshold = origThreshold
		wechatQueryGuardInst.reset()
	})
}

// OPT-20260823-035：限流窗口内第二次查询不发真实客户端。
func TestQueryWechatOutTradeNoByTxnIDRateLimited(t *testing.T) {
	setupQueryGuardTest(t)
	wechatQueryMinInterval = time.Hour
	calls := 0
	wechatQueryOrderByIdCall = func(ctx context.Context, svc *native.NativeApiService, req native.QueryOrderByIdRequest) (*payments.Transaction, *core.APIResult, error) {
		calls++
		return &payments.Transaction{OutTradeNo: core.String("WX-RATE-1")}, nil, nil
	}
	if _, err := queryWechatOutTradeNoByTxnID("4200000123456789012345678901"); err != nil {
		t.Fatalf("first query: %v", err)
	}
	if _, err := queryWechatOutTradeNoByTxnID("4200000123456789012345678902"); err == nil {
		t.Fatal("second query within min interval should be rate limited")
	}
	if calls != 1 {
		t.Fatalf("client calls=%d want 1（超限不得打真实客户端）", calls)
	}
}

// OPT-20260823-035：连续失败达到阈值后熔断打开，短路期不发真实客户端。
func TestQueryWechatOutTradeNoByTxnIDCircuitBreaker(t *testing.T) {
	setupQueryGuardTest(t)
	wechatQueryMinInterval = time.Nanosecond // 让连续失败调用不被限流挡住
	wechatQueryCircuitThreshold = 2
	calls := 0
	wechatQueryOrderByIdCall = func(ctx context.Context, svc *native.NativeApiService, req native.QueryOrderByIdRequest) (*payments.Transaction, *core.APIResult, error) {
		calls++
		return nil, nil, errors.New("wechat query API error")
	}
	// 前两次失败（真实调用）→ 阈值 2 → 熔断打开
	if _, err := queryWechatOutTradeNoByTxnID("4200000123456789012345678903"); err == nil {
		t.Fatal("call 1 should fail")
	}
	if _, err := queryWechatOutTradeNoByTxnID("4200000123456789012345678904"); err == nil {
		t.Fatal("call 2 should fail")
	}
	// 第三次进入短路期，不应再打真实客户端
	if _, err := queryWechatOutTradeNoByTxnID("4200000123456789012345678905"); err == nil || err != errWechatQueryCircuitOpen {
		t.Fatalf("call 3 want circuit-open error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("client calls=%d want 2（熔断期不得打真实客户端）", calls)
	}
}
