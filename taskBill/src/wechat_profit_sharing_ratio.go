package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"

	"tracelog"
)

// 推荐分成比例只允许来自微信支付查询接口，禁止 conf 默认 30%、本地政策 5%、
// 或 min(上限, 5%) 伪装成商户后台「分账管理比例」。
// 官方唯一查询接口是合作伙伴 GET /v3/profitsharing/merchant-configs/{sub_mchid}
// （https://pay.weixin.qq.com/doc/v3/partner/4012466864），max_ratio 为万分比。
// 直连商户对该 path 会 400 INVALID_REQUEST；无 path 的 merchant-configs 为 404。
// 普通商户「请求分账」POST /v3/profitsharing/orders 没有 ratio 字段。
// 查询失败 → source=unavailable、内部 API 503、前端「—」、打款比例 0。

const (
	commissionRateSourceWechat      = "wechat_merchant_config"
	commissionRateSourceUnavailable = "unavailable"
	commissionRateCacheTTL          = 5 * time.Minute
	wechatMaxRatioPercentMin        = 1
	wechatMaxRatioPercentMax        = 100
)

var errWechatRatioUnavailable = errors.New("wechat profit sharing ratio unavailable")

type commissionRateInfo struct {
	Percent int64
	Display string
	Source  string
}

var wechatMaxRatioQuery = queryWechatMerchantMaxRatioLive

var wechatRatioAPIGet = wechatV3GetAllowError

var commissionRateCache struct {
	mu      sync.Mutex
	info    commissionRateInfo
	expires time.Time
}

func resetCommissionRateCache() {
	commissionRateCache.mu.Lock()
	commissionRateCache.info = commissionRateInfo{}
	commissionRateCache.expires = time.Time{}
	commissionRateCache.mu.Unlock()
}

func formatCommissionRateDisplay(percent int64) string {
	return fmt.Sprintf("%d%%", percent)
}

func formatCommissionRateDecimal(percent int64) string {
	return fmt.Sprintf("0.%02d", percent)
}

func parseWechatMaxRatioPercent(body []byte) (int64, error) {
	var parsed struct {
		MaxRatio int64 `json:"max_ratio"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	return parseWechatMaxRatioPercentMust(parsed.MaxRatio)
}

func parseWechatMaxRatioPercentMust(maxRatio int64) (int64, error) {
	// 合作伙伴文档 max_ratio 为万分比（2000=20%）。申请提高后可超过产品默认 30%。
	// 1–100 且非整百：按百分比分子（商户后台展示口径）。
	var percent int64
	switch {
	case maxRatio >= wechatMaxRatioPercentMin*100 && maxRatio <= wechatMaxRatioPercentMax*100 && maxRatio%100 == 0:
		percent = maxRatio / 100
	case maxRatio >= wechatMaxRatioPercentMin && maxRatio <= wechatMaxRatioPercentMax:
		percent = maxRatio
	default:
		return 0, fmt.Errorf("wechat max_ratio out of range: %d", maxRatio)
	}
	return percent, nil
}

func queryWechatMerchantMaxRatioLive(ctx context.Context) (int64, error) {
	if !wechatLiveOK {
		return 0, errWechatRatioUnavailable
	}
	mchid := strings.TrimSpace(wechatCfg.Mchid)
	if mchid == "" {
		return 0, errWechatRatioUnavailable
	}
	path := "/v3/profitsharing/merchant-configs/" + url.PathEscape(mchid)
	status, body, err := wechatRatioAPIGet(ctx, path)
	if err != nil {
		return 0, err
	}
	if status != http.StatusOK {
		tracelog.EmitWithTrace(
			tracelog.TraceIDFromContext(ctx),
			"warn",
			"wechat profit sharing ratio query not 200, not using conf or local percent",
			"wechat_profit_sharing",
			map[string]string{
				"status": strconv.Itoa(status),
				"body":   truncateBytes(body, 300),
			},
		)
		return 0, errWechatRatioUnavailable
	}
	percent, err := parseWechatMaxRatioPercent(body)
	if err != nil {
		return 0, err
	}
	return percent, nil
}

func unavailableCommissionRate() commissionRateInfo {
	return commissionRateInfo{
		Percent: 0,
		Display: "",
		Source:  commissionRateSourceUnavailable,
	}
}

func (info commissionRateInfo) ok() bool {
	return info.Percent > 0 && info.Source != commissionRateSourceUnavailable && strings.TrimSpace(info.Display) != ""
}

func wechatCommissionRate(percent int64) commissionRateInfo {
	return commissionRateInfo{
		Percent: percent,
		Display: formatCommissionRateDisplay(percent),
		Source:  commissionRateSourceWechat,
	}
}

func resolveCommissionRate(ctx context.Context) commissionRateInfo {
	now := time.Now()
	commissionRateCache.mu.Lock()
	if !commissionRateCache.expires.IsZero() && now.Before(commissionRateCache.expires) {
		info := commissionRateCache.info
		commissionRateCache.mu.Unlock()
		return info
	}
	commissionRateCache.mu.Unlock()

	percent, err := wechatMaxRatioQuery(ctx)
	info := unavailableCommissionRate()
	if err != nil || percent <= 0 {
		tracelog.EmitWithTrace(
			tracelog.TraceIDFromContext(ctx),
			"error",
			"wechat profit sharing ratio unavailable, not falling back to conf or local percent",
			"wechat_profit_sharing",
			map[string]string{
				"source": commissionRateSourceUnavailable,
			},
		)
	} else {
		info = wechatCommissionRate(percent)
		tracelog.EmitWithTrace(
			tracelog.TraceIDFromContext(ctx),
			"info",
			"wechat profit sharing ratio loaded from api",
			"wechat_profit_sharing",
			map[string]string{
				"percent": formatCommissionRateDisplay(percent),
				"source":  commissionRateSourceWechat,
			},
		)
	}

	commissionRateCache.mu.Lock()
	commissionRateCache.info = info
	commissionRateCache.expires = now.Add(commissionRateCacheTTL)
	commissionRateCache.mu.Unlock()
	return info
}

func handleInternalReferralCommissionRate(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	info := resolveCommissionRate(r.Context())
	if !info.ok() {
		writeErrorJSON(w, http.StatusServiceUnavailable, "commission rate unavailable", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tracelog.EmitWithTrace(
		tracelog.TraceIDFromContext(r.Context()),
		"info",
		"referral commission rate from wechat api",
		"wechat_profit_sharing",
		map[string]string{
			"display": info.Display,
			"source":  info.Source,
		},
	)
	writeJSON(w, http.StatusOK, map[string]string{
		"commission_rate":         formatCommissionRateDecimal(info.Percent),
		"commission_rate_display": info.Display,
		"source":                  info.Source,
	})
}

// profitSharingMerchantRatioCall 是分账比例查询的 SDK 可注入接缝。
var profitSharingMerchantRatioCall = func(ctx context.Context, svc *profitsharing.MerchantsApiService, req profitsharing.QueryMerchantRatioRequest) (*profitsharing.QueryMerchantRatioResponse, *core.APIResult, error) {
	return svc.QueryMerchantRatio(ctx, req)
}

// wechatV3GetAllowError 查询分账比例（默认实现走 SDK MerchantsApiService，ADR-0032）。
// 保留 path 入参接缝供单测替换；返回体保持 `(status, body, nil)`，API 错误转成应答体。
func wechatV3GetAllowError(ctx context.Context, path string) (int, []byte, error) {
	mchid := strings.TrimPrefix(path, "/v3/profitsharing/merchant-configs/")
	mchid = strings.Trim(mchid, "/")
	if mchid == "" || strings.Contains(mchid, "/") {
		return 0, nil, fmt.Errorf("invalid merchant-configs path: %s", path)
	}
	if wechatClient == nil {
		return 0, nil, fmt.Errorf("wechat pay client not initialized")
	}
	req := profitsharing.QueryMerchantRatioRequest{SubMchid: core.String(mchid)}
	svc := profitsharing.MerchantsApiService{Client: wechatClient}
	resp, _, err := profitSharingMerchantRatioCall(ctx, &svc, req)
	if err != nil {
		if apiErr, ok := err.(*core.APIError); ok && apiErr != nil {
			b := []byte(apiErr.Body)
			if len(b) == 0 {
				b, _ = json.Marshal(map[string]string{"code": apiErr.Code, "message": apiErr.Message})
			}
			return apiErr.StatusCode, b, nil
		}
		return 0, nil, err
	}
	if resp == nil {
		return 0, nil, fmt.Errorf("empty merchant ratio response")
	}
	body, err := json.Marshal(resp)
	if err != nil {
		return 0, nil, err
	}
	return http.StatusOK, body, nil
}
