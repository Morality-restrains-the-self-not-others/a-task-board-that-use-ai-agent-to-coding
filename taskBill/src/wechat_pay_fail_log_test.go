package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return string(out)
}

// TestWechatPrepayFailLogsAppidMchid 注入失败 Prepay（APPID_MCHID_NOT_MATCH），
// 断言失败日志含 appid / mchid / 微信错误码且不含密钥（OPT-20260817-031）。
func TestWechatPrepayFailLogsAppidMchid(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

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
	wechatCfg.Appid = "wx625802b55b33608f"
	wechatCfg.Mchid = "1900000109"
	wechatCfg.NotifyURL = "https://example.com/wechat/notify"

	wechatPrepayCall = func(ctx context.Context, svc *native.NativeApiService, req native.PrepayRequest) (*native.PrepayResponse, *core.APIResult, error) {
		return nil, nil, &core.APIError{
			StatusCode: 400,
			Code:       "APPID_MCHID_NOT_MATCH",
			Message:    "appid和mch_id不匹配",
			Body:       `{"code":"APPID_MCHID_NOT_MATCH","message":"appid和mch_id不匹配"}`,
		}
	}

	out := captureStdout(t, func() {
		// amountFen=100 → 1.00 元；失败路径只校验日志字段，不关心金额换算。
		_, _, err := wechatPrepay(context.Background(), 1, "877890668700139520", 100, "资源购买-测试", "https://example.com/wechat/notify")
		if err == nil {
			t.Fatal("注入失败 Prepay 后 wechatPrepay 应返回错误")
		}
	})

	for _, want := range []string{
		`"appid":"wx625802b55b33608f"`,
		`"mchid":"1900000109"`,
		`"msg":"wechat prepay failed"`,
		"APPID_MCHID_NOT_MATCH",
		`"out_trade_no":"WX`,
		`"amount_fen":"100"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("失败日志缺少 %s；实际输出: %s", want, out)
		}
	}
	if strings.Contains(out, "api_v3_key") || strings.Contains(out, "merchant_private_key") {
		t.Fatalf("失败日志禁止写入密钥：%s", out)
	}
}
