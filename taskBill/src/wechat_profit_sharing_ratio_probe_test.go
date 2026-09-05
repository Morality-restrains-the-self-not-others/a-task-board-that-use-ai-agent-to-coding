package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
)

// 现网探测：直连商户「分账管理比例」有没有可签的 GET。
// 官方文档只给合作伙伴 GET /v3/profitsharing/merchant-configs/{sub_mchid}。
// 运行：WECHAT_RATIO_PROBE=1 go test ./src -count=1 -timeout 60s -run TestProbeWechatProfitSharingRatioAPIs
func TestProbeWechatProfitSharingRatioAPIs(t *testing.T) {
	if os.Getenv("WECHAT_RATIO_PROBE") != "1" {
		t.Skip("set WECHAT_RATIO_PROBE=1 to hit live WeChat APIs")
	}
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	loadWechatPayConfig(root)
	if !wechatLiveOK || wechatClient == nil {
		t.Fatal("wechat live client not ready")
	}
	mchid := strings.TrimSpace(wechatCfg.Mchid)
	if mchid == "" {
		t.Fatal("mchid empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	paths := []string{
		"/v3/profitsharing/merchant-configs",
		"/v3/profitsharing/merchant-configs/" + mchid,
		"/v3/profitsharing/merchant-config",
		"/v3/profitsharing/configs",
		"/v3/profitsharing/merchant-configs/" + mchid + "/max-ratio",
		"/v3/profitsharing/orders/max-ratio",
		"/v3/profitsharing/merchant-ratio",
		"/v3/merchant/fund/profitsharing",
	}
	for _, path := range paths {
		status, snippet := probeWechatGET(ctx, path)
		t.Logf("GET %s status=%d body=%s", path, status, snippet)
	}
}

func probeWechatGET(ctx context.Context, path string) (int, string) {
	status, body, err := wechatV3Get(ctx, path)
	if err == nil {
		return status, truncateBytes(body, 400)
	}
	var apiErr *core.APIError
	if errors.As(err, &apiErr) && apiErr != nil {
		msg := apiErr.Code + " " + apiErr.Message
		if apiErr.Body != "" {
			msg = truncateBytes([]byte(apiErr.Body), 400)
		}
		return apiErr.StatusCode, msg
	}
	return 0, truncateBytes([]byte(err.Error()), 400)
}
