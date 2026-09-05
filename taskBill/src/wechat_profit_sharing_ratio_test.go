package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseWechatMaxRatioPercent(t *testing.T) {
	t.Parallel()
	percent, err := parseWechatMaxRatioPercent([]byte(`{"sub_mchid":"1114728879","max_ratio":800}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if percent != 8 {
		t.Fatalf("percent=%d want 8 (万分比 800)", percent)
	}
}

func TestParseWechatMaxRatioPercent_TreatsPercentScaleAsPercent(t *testing.T) {
	t.Parallel()
	percent, err := parseWechatMaxRatioPercent([]byte(`{"max_ratio":12}`))
	if err != nil {
		t.Fatalf("parse percent-scale max_ratio: %v", err)
	}
	if percent != 12 {
		t.Fatalf("percent=%d want 12 (百分比分子)", percent)
	}
}

func TestParseWechatMaxRatioPercent_AllowsAboveDefault30(t *testing.T) {
	t.Parallel()
	percent, err := parseWechatMaxRatioPercent([]byte(`{"max_ratio":3100}`))
	if err != nil {
		t.Fatalf("parse 31%% 万分比: %v", err)
	}
	if percent != 31 {
		t.Fatalf("percent=%d want 31 (申请提高后可超过文档默认 30%%)", percent)
	}
}

func TestParseWechatMaxRatioPercent_RejectsOutOfRange(t *testing.T) {
	t.Parallel()
	if _, err := parseWechatMaxRatioPercent([]byte(`{"max_ratio":0}`)); err == nil {
		t.Fatal("expected error for 0")
	}
	if _, err := parseWechatMaxRatioPercent([]byte(`{"max_ratio":101}`)); err == nil {
		t.Fatal("expected error for 101 (既非百分比分子亦非整百万分比)")
	}
}

func TestQueryWechatMerchantMaxRatioLive_ParsesAPINotConf(t *testing.T) {
	origGet := wechatRatioAPIGet
	origLive := wechatLiveOK
	origMch := wechatCfg.Mchid
	wechatLiveOK = true
	wechatCfg.Mchid = "1114728879"
	wechatRatioAPIGet = func(_ context.Context, path string) (int, []byte, error) {
		if !strings.Contains(path, "1114728879") {
			t.Fatalf("path=%s want mchid in merchant-configs", path)
		}
		return http.StatusOK, []byte(`{"sub_mchid":"1114728879","max_ratio":1500}`), nil
	}
	t.Cleanup(func() {
		wechatRatioAPIGet = origGet
		wechatLiveOK = origLive
		wechatCfg.Mchid = origMch
	})

	got, err := queryWechatMerchantMaxRatioLive(context.Background())
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if got != 15 {
		t.Fatalf("got %d want 15 from API max_ratio=1500, not conf 30", got)
	}
}

func TestQueryWechatMerchantMaxRatioLive_Partner400IsUnavailableNotConf(t *testing.T) {
	origGet := wechatRatioAPIGet
	origLive := wechatLiveOK
	origMch := wechatCfg.Mchid
	wechatLiveOK = true
	wechatCfg.Mchid = "1114728879"
	wechatRatioAPIGet = func(context.Context, string) (int, []byte, error) {
		return http.StatusBadRequest, []byte(`{"code":"INVALID_REQUEST","message":"子商户或品牌商户不能为当前请求的服务商商户"}`), nil
	}
	t.Cleanup(func() {
		wechatRatioAPIGet = origGet
		wechatLiveOK = origLive
		wechatCfg.Mchid = origMch
	})

	got, err := queryWechatMerchantMaxRatioLive(context.Background())
	if !errors.Is(err, errWechatRatioUnavailable) {
		t.Fatalf("err=%v want unavailable (直连 400 不得读 conf 30%%)", err)
	}
	if got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}

func TestQueryWechatMerchantMaxRatioLive_UnavailableWithoutLiveClient(t *testing.T) {
	origLive := wechatLiveOK
	wechatLiveOK = false
	t.Cleanup(func() {
		wechatLiveOK = origLive
	})

	if _, err := queryWechatMerchantMaxRatioLive(context.Background()); !errors.Is(err, errWechatRatioUnavailable) {
		t.Fatalf("err=%v want unavailable, must not read conf", err)
	}
}

func TestResolveCommissionRate_UsesWechatQuery(t *testing.T) {
	resetCommissionRateCache()
	orig := wechatMaxRatioQuery
	wechatMaxRatioQuery = func(context.Context) (int64, error) { return 8, nil }
	t.Cleanup(func() {
		wechatMaxRatioQuery = orig
		resetCommissionRateCache()
	})

	info := resolveCommissionRate(context.Background())
	if info.Percent != 8 || info.Display != "8%" || info.Source != commissionRateSourceWechat {
		t.Fatalf("got %+v", info)
	}
	if getCommissionRate() != 5 {
		t.Fatalf("getCommissionRate=%d want 5 (fixed policy capped by wechat 8%%)", getCommissionRate())
	}
}

func TestGetCommissionRateUsesWechatPercentWhenWechatBelowFixedFive(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	if _, err := updateReferralConfig(15, 20); err != nil {
		t.Fatalf("config: %v", err)
	}
	resetCommissionRateCache()
	orig := wechatMaxRatioQuery
	wechatMaxRatioQuery = func(context.Context) (int64, error) { return 3, nil }
	t.Cleanup(func() {
		wechatMaxRatioQuery = orig
		resetCommissionRateCache()
	})

	if got := getCommissionRate(); got != 3 {
		t.Fatalf("payout getCommissionRate=%d want 3 (微信上限低于固定 5%%)", got)
	}
}

func TestGetCommissionRateCapsAtFixedFive(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	if _, err := updateReferralConfig(15, 8); err != nil {
		t.Fatalf("config: %v", err)
	}
	resetCommissionRateCache()
	orig := wechatMaxRatioQuery
	wechatMaxRatioQuery = func(context.Context) (int64, error) { return 30, nil }
	t.Cleanup(func() {
		wechatMaxRatioQuery = orig
		resetCommissionRateCache()
	})
	if got := getCommissionRate(); got != 5 {
		t.Fatalf("getCommissionRate=%d want 5 (min(固定5%%, 微信上限))", got)
	}
}

func TestResolveCommissionRate_DoesNotFallbackWhenWechatFails(t *testing.T) {
	resetCommissionRateCache()
	orig := wechatMaxRatioQuery
	wechatMaxRatioQuery = func(context.Context) (int64, error) {
		return 0, errors.New("INVALID_REQUEST: 子商户或品牌商户不能为当前请求的服务商商户")
	}
	t.Cleanup(func() {
		wechatMaxRatioQuery = orig
		resetCommissionRateCache()
	})

	info := resolveCommissionRate(context.Background())
	if info.Percent != 0 || info.Display == "5%" || info.Display == "30%" || info.Source == "local_fallback" || info.Source == "payout_policy" {
		t.Fatalf("must not fallback to 5%% or conf 30%%, got %+v", info)
	}
	if info.Source != commissionRateSourceUnavailable {
		t.Fatalf("source=%s want %s", info.Source, commissionRateSourceUnavailable)
	}
	if got := getCommissionRate(); got != 0 {
		t.Fatalf("payout getCommissionRate=%d want 0", got)
	}
}

func TestHandleInternalReferralCommissionRate_Unavailable(t *testing.T) {
	resetCommissionRateCache()
	orig := wechatMaxRatioQuery
	wechatMaxRatioQuery = func(context.Context) (int64, error) {
		return 0, errors.New("NO_AUTH")
	}
	t.Cleanup(func() {
		wechatMaxRatioQuery = orig
		resetCommissionRateCache()
	})

	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/referral/commission-rate/", nil)
	rec := httptest.NewRecorder()
	handleInternalReferralCommissionRate(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s want 503", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `"5%"`) || strings.Contains(body, `"0.05"`) || strings.Contains(body, `"30%"`) {
		t.Fatalf("body must not contain 5%% or 30%% fallback: %s", body)
	}
}

func TestHandleInternalReferralCommissionRate(t *testing.T) {
	resetCommissionRateCache()
	orig := wechatMaxRatioQuery
	wechatMaxRatioQuery = func(context.Context) (int64, error) { return 12, nil }
	t.Cleanup(func() {
		wechatMaxRatioQuery = orig
		resetCommissionRateCache()
	})

	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/referral/commission-rate/", nil)
	rec := httptest.NewRecorder()
	handleInternalReferralCommissionRate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["commission_rate_display"] != "12%" {
		t.Fatalf("display=%v want 12%% from API, not min(12,5)", out["commission_rate_display"])
	}
	if out["commission_rate"] != "0.12" {
		t.Fatalf("rate=%v", out["commission_rate"])
	}
	if out["source"] != commissionRateSourceWechat {
		t.Fatalf("source=%v want %s", out["source"], commissionRateSourceWechat)
	}
}

func TestHandleInternalReferralCommissionRate_DisplaysApiValueEvenIf30(t *testing.T) {
	resetCommissionRateCache()
	orig := wechatMaxRatioQuery
	wechatMaxRatioQuery = func(context.Context) (int64, error) { return 30, nil }
	t.Cleanup(func() {
		wechatMaxRatioQuery = orig
		resetCommissionRateCache()
	})

	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/referral/commission-rate/", nil)
	rec := httptest.NewRecorder()
	handleInternalReferralCommissionRate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["commission_rate_display"] != "30%" {
		t.Fatalf("display=%v want 30%% only when API returns 30, not conf/min", out["commission_rate_display"])
	}
	if out["source"] != commissionRateSourceWechat {
		t.Fatalf("source=%v", out["source"])
	}
}

func TestHandleInternalReferralCommissionRate_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/referral/commission-rate/", nil)
	rec := httptest.NewRecorder()
	handleInternalReferralCommissionRate(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rec.Code)
	}
}
