package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"tracelog"
)

// withTraceID returns req with an injected request trace id so error rows can
// expose wechat_error_trace_id in unit tests (OPT-20260826-008).
func withTraceID(req *http.Request, tid string) *http.Request {
	return req.WithContext(tracelog.ContextWithTraceID(req.Context(), tid))
}

func staffAdminJSONReq(t *testing.T, method, path string, payload interface{}) *http.Request {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "super_admin")
	return req
}

func markProfitSharingWechatSubmitted(t *testing.T, psID int64) {
	t.Helper()
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET wechat_profit_sharing_id = ? WHERE id = ?`,
		"300845"+formatID(psID), psID); err != nil {
		t.Fatalf("mark wechat submitted: %v", err)
	}
}

func TestHandleSystemAdminRefreshProfitSharingWechatSkipsUnsubmitted(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	_, psID := seedReferrerFlaggedOrder(t, referrer, buyer, psStatusPending, "2026-08-20T00:00:00Z")

	calls := 0
	origClient := wechatClient
	origCall := profitSharingQueryOrderCall
	wechatClient = &core.Client{}
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingQueryOrderCall = origCall
	})
	profitSharingQueryOrderCall = func(context.Context, *profitsharing.OrdersApiService, profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		calls++
		return &profitsharing.OrdersEntity{State: profitsharing.OrderStatus("FINISHED").Ptr()}, nil, nil
	}

	req := staffAdminJSONReq(t, http.MethodPost, "/api/system-admin/profit-sharing/refresh-wechat/", map[string]interface{}{
		"referrer_user_id": formatID(referrer),
		"ids":              []string{formatID(psID)},
	})
	req = withTraceID(req, "trace-not-submitted")
	rec := httptest.NewRecorder()
	handleSystemAdminRefreshProfitSharingWechat(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if calls != 0 {
		t.Fatalf("QueryOrder calls=%d want 0 (local pending is not a WeChat order yet)", calls)
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["wechat_state"] != wechatProfitSharingStateNotSubmitted {
		t.Fatalf("wechat_state=%v want %q", row["wechat_state"], wechatProfitSharingStateNotSubmitted)
	}
	if errText, _ := row["wechat_error"].(string); errText != "" {
		t.Fatalf("wechat_error=%q want empty", errText)
	}
	if _, has := row["wechat_error_trace_id"]; has {
		t.Fatalf("not-submitted row must not carry wechat_error_trace_id: %v", row)
	}
}

func TestHandleSystemAdminRefreshProfitSharingWechatSubmittedNotFound(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	_, psID := seedReferrerFlaggedOrder(t, referrer, buyer, psStatusProcessing, "2026-08-20T00:00:00Z")
	markProfitSharingWechatSubmitted(t, psID)

	origClient := wechatClient
	origCall := profitSharingQueryOrderCall
	wechatClient = &core.Client{}
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingQueryOrderCall = origCall
	})
	profitSharingQueryOrderCall = func(context.Context, *profitsharing.OrdersApiService, profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		return nil, nil, &core.APIError{
			StatusCode: http.StatusNotFound,
			Code:       "RESOURCE_NOT_EXISTS",
			Message:    "记录不存在",
			Body:       `{"code":"RESOURCE_NOT_EXISTS","message":"记录不存在"}`,
		}
	}

	req := staffAdminJSONReq(t, http.MethodPost, "/api/system-admin/profit-sharing/refresh-wechat/", map[string]interface{}{
		"referrer_user_id": formatID(referrer),
		"ids":              []string{formatID(psID)},
	})
	req = withTraceID(req, "trace-not-found")
	rec := httptest.NewRecorder()
	handleSystemAdminRefreshProfitSharingWechat(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	row, _ := items[0].(map[string]interface{})
	errText, _ := row["wechat_error"].(string)
	if errText != "微信侧未找到分账单" {
		t.Fatalf("wechat_error=%q", errText)
	}
	if tid, _ := row["wechat_error_trace_id"].(string); tid != "trace-not-found" {
		t.Fatalf("wechat_error_trace_id=%q want trace-not-found", tid)
	}
}

func TestHandleSystemAdminRefreshProfitSharingWechatRejectsForeignIDs(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrerA := generateSnowflakeID()
	buyerA := generateSnowflakeID()
	_, idA := seedReferrerFlaggedOrder(t, referrerA, buyerA, psStatusPending, "2026-08-20T00:00:00Z")
	referrerB := generateSnowflakeID()
	buyerB := generateSnowflakeID()
	_, idB := seedReferrerFlaggedOrder(t, referrerB, buyerB, psStatusPending, "2026-08-21T00:00:00Z")

	calls := 0
	origClient := wechatClient
	origCall := profitSharingQueryOrderCall
	wechatClient = &core.Client{}
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingQueryOrderCall = origCall
	})
	profitSharingQueryOrderCall = func(context.Context, *profitsharing.OrdersApiService, profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		calls++
		return &profitsharing.OrdersEntity{State: profitsharing.OrderStatus("FINISHED").Ptr()}, nil, nil
	}

	req := staffAdminJSONReq(t, http.MethodPost, "/api/system-admin/profit-sharing/refresh-wechat/", map[string]interface{}{
		"referrer_user_id": formatID(referrerA),
		"ids":              []string{formatID(idA), formatID(idB)},
	})
	rec := httptest.NewRecorder()
	handleSystemAdminRefreshProfitSharingWechat(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s want 400", rec.Code, rec.Body.String())
	}
	if calls != 0 {
		t.Fatalf("QueryOrder calls=%d want 0", calls)
	}
}

func TestHandleSystemAdminRefreshProfitSharingWechatFinishedDoesNotWriteStatus(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	_, psID := seedReferrerFlaggedOrder(t, referrer, buyer, psStatusPending, "2026-08-20T00:00:00Z")
	markProfitSharingWechatSubmitted(t, psID)

	origClient := wechatClient
	origCall := profitSharingQueryOrderCall
	wechatClient = &core.Client{}
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingQueryOrderCall = origCall
	})
	profitSharingQueryOrderCall = func(_ context.Context, _ *profitsharing.OrdersApiService, req profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		if req.OutOrderNo == nil || strings.TrimSpace(*req.OutOrderNo) == "" {
			t.Fatal("missing out_order_no")
		}
		return &profitsharing.OrdersEntity{State: profitsharing.OrderStatus("FINISHED").Ptr()}, nil, nil
	}

	req := staffAdminJSONReq(t, http.MethodPost, "/api/system-admin/profit-sharing/refresh-wechat/", map[string]interface{}{
		"referrer_user_id": formatID(referrer),
		"ids":              []string{formatID(psID)},
	})
	rec := httptest.NewRecorder()
	handleSystemAdminRefreshProfitSharingWechat(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["wechat_state"] != "FINISHED" {
		t.Fatalf("wechat_state=%v", row["wechat_state"])
	}
	if strings.Contains(rec.Body.String(), "openid") {
		t.Fatalf("leaked openid: %s", rec.Body.String())
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM billing_profit_sharing WHERE id = ?`, psID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != psStatusPending {
		t.Fatalf("db status=%s want pending (must not write back)", status)
	}
}

func TestHandleSystemAdminRefreshProfitSharingWechatMissingTxnContinues(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrer := generateSnowflakeID()
	buyerOK := generateSnowflakeID()
	_, okID := seedReferrerFlaggedOrder(t, referrer, buyerOK, psStatusPending, "2026-08-20T00:00:00Z")
	markProfitSharingWechatSubmitted(t, okID)
	buyerMissing := generateSnowflakeID()
	tenantID := generateSnowflakeID()
	orderMissing := generateSnowflakeID()
	seedPaidOrderForProfitSharing(t, tenantID, orderMissing, buyerMissing, 800)
	if err := upsertReferralEdge(formatID(referrer), formatID(buyerMissing), tenantID, utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}
	missingID := seedProfitSharingRecordStatus(t, orderMissing, tenantID, psStatusPending, "2026-08-21T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_user_id = ? WHERE id = ?`, formatID(referrer), missingID); err != nil {
		t.Fatal(err)
	}
	markProfitSharingWechatSubmitted(t, missingID)

	origClient := wechatClient
	origCall := profitSharingQueryOrderCall
	wechatClient = &core.Client{}
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingQueryOrderCall = origCall
	})
	queryCalls := 0
	profitSharingQueryOrderCall = func(context.Context, *profitsharing.OrdersApiService, profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		queryCalls++
		return &profitsharing.OrdersEntity{State: profitsharing.OrderStatus("PROCESSING").Ptr()}, nil, nil
	}

	req := staffAdminJSONReq(t, http.MethodPost, "/api/system-admin/profit-sharing/refresh-wechat/", map[string]interface{}{
		"referrer_user_id": formatID(referrer),
		"ids":              []string{formatID(okID), formatID(missingID)},
	})
	req = withTraceID(req, "trace-missing-txn")
	rec := httptest.NewRecorder()
	handleSystemAdminRefreshProfitSharingWechat(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if queryCalls != 1 {
		t.Fatalf("QueryOrder calls=%d want 1", queryCalls)
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	byID := map[string]map[string]interface{}{}
	for _, raw := range items {
		row, _ := raw.(map[string]interface{})
		id, _ := row["id"].(string)
		byID[id] = row
	}
	if byID[formatID(okID)]["wechat_state"] != "PROCESSING" {
		t.Fatalf("ok row=%v", byID[formatID(okID)])
	}
	errText, _ := byID[formatID(missingID)]["wechat_error"].(string)
	if !strings.Contains(errText, "微信支付单号") {
		t.Fatalf("missing txn error=%q", errText)
	}
	if tid, _ := byID[formatID(missingID)]["wechat_error_trace_id"].(string); tid != "trace-missing-txn" {
		t.Fatalf("missing txn wechat_error_trace_id=%q want trace-missing-txn", tid)
	}
}

func TestHandleSystemAdminRefreshProfitSharingWechatFrequencyLimited(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	_, psID := seedReferrerFlaggedOrder(t, referrer, buyer, psStatusPending, "2026-08-20T00:00:00Z")
	markProfitSharingWechatSubmitted(t, psID)

	origClient := wechatClient
	origCall := profitSharingQueryOrderCall
	wechatClient = &core.Client{}
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingQueryOrderCall = origCall
	})
	profitSharingQueryOrderCall = func(context.Context, *profitsharing.OrdersApiService, profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		return nil, nil, &core.APIError{
			StatusCode: http.StatusTooManyRequests,
			Code:       "FREQUENCY_LIMITED",
			Message:    "频率超限",
			Body:       `{"code":"FREQUENCY_LIMITED","secret":"sk_live_REDACTED"}`,
		}
	}

	req := staffAdminJSONReq(t, http.MethodPost, "/api/system-admin/profit-sharing/refresh-wechat/", map[string]interface{}{
		"referrer_user_id": formatID(referrer),
		"ids":              []string{formatID(psID)},
	})
	req = withTraceID(req, "trace-freq-limited")
	rec := httptest.NewRecorder()
	handleSystemAdminRefreshProfitSharingWechat(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	bodyText := rec.Body.String()
	if strings.Contains(bodyText, "sk_live") || strings.Contains(bodyText, "secret") {
		t.Fatalf("leaked secret: %s", bodyText)
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	row, _ := items[0].(map[string]interface{})
	errText, _ := row["wechat_error"].(string)
	if !strings.Contains(errText, "频率") {
		t.Fatalf("wechat_error=%q", errText)
	}
	if tid, _ := row["wechat_error_trace_id"].(string); tid != "trace-freq-limited" {
		t.Fatalf("wechat_error_trace_id=%q want trace-freq-limited", tid)
	}
}

func TestHandleSystemAdminRefreshProfitSharingWechatForbidden(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/profit-sharing/refresh-wechat/", strings.NewReader(`{}`))
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec := httptest.NewRecorder()
	handleSystemAdminRefreshProfitSharingWechat(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestHandleSystemAdminRefreshProfitSharingWechatMuxRoute(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	_, psID := seedReferrerFlaggedOrder(t, referrer, buyer, psStatusPending, "2026-08-20T00:00:00Z")
	markProfitSharingWechatSubmitted(t, psID)
	origClient := wechatClient
	origCall := profitSharingQueryOrderCall
	wechatClient = &core.Client{}
	t.Cleanup(func() {
		wechatClient = origClient
		profitSharingQueryOrderCall = origCall
	})
	profitSharingQueryOrderCall = func(context.Context, *profitsharing.OrdersApiService, profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
		return &profitsharing.OrdersEntity{State: profitsharing.OrderStatus("FINISHED").Ptr()}, nil, nil
	}
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := staffAdminJSONReq(t, http.MethodPost, "/api/system-admin/profit-sharing/refresh-wechat/", map[string]interface{}{
		"referrer_user_id": formatID(referrer),
		"ids":              []string{formatID(psID)},
	})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mux status=%d body=%s", rec.Code, rec.Body.String())
	}
}
