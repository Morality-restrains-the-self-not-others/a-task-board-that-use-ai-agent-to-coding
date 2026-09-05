package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 推荐人本人分账订单列表与手动分账（2026-08-23-referrer-manual-profit-sharing-design.md）：
//   - 列表仅返回本人记录（referrer_user_id == X-User-Id），禁止泄漏 openid / 微信分账单号
//   - 手动分账：15–30 天窗口内可分享，他人记录 404，processing/finished 幂等重放

// seedReferrerOrder 造一笔已支付订单（带 paid_at）+ 分账记录，返回 ps id。
func seedReferrerOrder(t *testing.T, tenantID, orderID int64, referrerUserID string, status string, paidAt string) int64 {
	t.Helper()
	return seedReferrerOrderOnChannel(t, tenantID, orderID, referrerUserID, status, paidAt, "CH-"+referrerUserID)
}

func seedReferrerOrderOnChannel(t *testing.T, tenantID, orderID int64, referrerUserID, status, paidAt, channelCode string) int64 {
	t.Helper()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, payment_method, payment_ref, paid_at, created_at, user_id)
		VALUES (?, ?, ?, 'paid', 1000, 'wechat', ?, ?, ?, ?)`,
		orderID, tenantID, "ORD"+formatID(orderID), "wechat:WXR"+formatID(orderID), paidAt, now, orderID); err != nil {
		t.Fatalf("seed order: %v", err)
	}
	if err := upsertReferralEdge(referrerUserID, formatID(orderID), tenantID, paidAt, channelCode, true); err != nil {
		t.Fatalf("seed edge: %v", err)
	}
	if _, err := db.Exec(`INSERT IGNORE INTO billing_account (id, tenant_id, balance, created_at, updated_at) VALUES (?, ?, 0, ?, ?)`,
		generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_payment_ledger (id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
			points, remaining_points, amount_minor, currency, billing_transaction_id, created_at, expires_at)
		VALUES (?, ?, (SELECT id FROM billing_account WHERE tenant_id = ?), 'wechat', ?, ?, 1000, 1000, 1000, 'CNY', NULL, ?, ?)`,
		generateSnowflakeID(), tenantID, tenantID, "WXR"+formatID(orderID), "420000"+formatID(orderID), now, now); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}
	outNo := "PSR" + formatID(generateSnowflakeID())
	if _, err := db.Exec(`
		INSERT INTO billing_profit_sharing (out_profit_sharing_no, order_id, order_number, tenant_id,
			referrer_user_id, referrer_openid, total_yuan_cents, commission_yuan_cents, status, settle_after, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'openid-secret-1', 1000, 50, ?, ?, ?, ?)`,
		outNo, orderID, "ORD"+formatID(orderID), tenantID, referrerUserID, status, paidAt, now, now); err != nil {
		t.Fatalf("seed profit sharing: %v", err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM billing_profit_sharing WHERE out_profit_sharing_no = ?`, outNo).Scan(&id); err != nil {
		t.Fatalf("load ps id: %v", err)
	}
	return id
}

func daysAgo(d int) string {
	return time.Now().UTC().Add(-time.Duration(d) * 24 * time.Hour).Format(time.RFC3339)
}

func referrerReq(method, path, userID string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
	}
	return req
}

func referrerJSONReq(method, path, userID, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
	}
	return req
}

func TestReferrerProfitSharingListOwnershipAndDisplay(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	const tenantID int64 = 95101
	seedReferrerOrderOnChannel(t, tenantID, 3001, "referrer-1", psStatusPending, daysAgo(5), "CH-A")   // 冻结
	seedReferrerOrderOnChannel(t, tenantID, 3002, "referrer-1", psStatusPending, daysAgo(20), "CH-A")  // 可分账
	seedReferrerOrderOnChannel(t, tenantID, 3003, "referrer-1", psStatusFinished, daysAgo(20), "CH-B") // 已分账
	seedReferrerOrderOnChannel(t, tenantID, 3004, "referrer-2", psStatusPending, daysAgo(20), "CH-A")  // 他人同渠道码

	rec := httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodGet, "/api/billing/profit-sharing/referrer-orders/", ""))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no identity: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodGet, "/api/billing/profit-sharing/referrer-orders/", "referrer-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("list: status=%d body=%s", rec.Code, rec.Body.String())
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "openid") {
		t.Fatalf("response leaked openid: %s", raw)
	}
	if strings.Contains(raw, "order_number") || strings.Contains(raw, "order_id") {
		t.Fatalf("response leaked order identity: %s", raw)
	}
	if strings.Contains(raw, "ORD") {
		t.Fatalf("response leaked order number: %s", raw)
	}
	body := decodeJSONMap(t, rec)
	channels, _ := body["channels"].([]interface{})
	if len(channels) != 2 {
		t.Fatalf("referrer-1 channels=%d want 2 body=%s", len(channels), raw)
	}
	byCode := map[string]map[string]interface{}{}
	for _, c := range channels {
		m := c.(map[string]interface{})
		byCode[fmt.Sprintf("%v", m["channel_code"])] = m
	}
	a := byCode["CH-A"]
	if a == nil {
		t.Fatalf("CH-A missing: %s", raw)
	}
	if int64(a["order_amount_yuan_cents"].(float64)) != 2000 {
		t.Fatalf("CH-A order_amount=%v", a["order_amount_yuan_cents"])
	}
	if int64(a["frozen_amount_yuan_cents"].(float64)) != 50 {
		t.Fatalf("CH-A frozen=%v", a["frozen_amount_yuan_cents"])
	}
	if int64(a["shareable_amount_yuan_cents"].(float64)) != 50 {
		t.Fatalf("CH-A shareable=%v", a["shareable_amount_yuan_cents"])
	}
	if a["shareable"] != true {
		t.Fatalf("CH-A shareable flag=%v", a["shareable"])
	}
	b := byCode["CH-B"]
	if b == nil || b["shareable"] != false {
		t.Fatalf("CH-B=%v", b)
	}

	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodGet, "/api/billing/profit-sharing/referrer-orders/", "referrer-2"))
	body = decodeJSONMap(t, rec)
	if channels, _ := body["channels"].([]interface{}); len(channels) != 1 {
		t.Fatalf("referrer-2 channels=%d want 1 body=%s", len(channels), rec.Body.String())
	}
}

func TestReferrerOrderShareWindowAndOwnership(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	const tenantID int64 = 95102
	frozenID := seedReferrerOrder(t, tenantID, 3101, "referrer-1", psStatusPending, daysAgo(5))
	shareableID := seedReferrerOrder(t, tenantID, 3102, "referrer-1", psStatusPending, daysAgo(20))
	otherID := seedReferrerOrder(t, tenantID, 3103, "referrer-2", psStatusPending, daysAgo(20))

	// 无身份 → 401
	rec := httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodPost,
		fmt.Sprintf("/api/billing/profit-sharing/referrer-orders/%d/share/", shareableID), ""))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no identity share: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 他人记录 → 404（不暴露存在性）
	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodPost,
		fmt.Sprintf("/api/billing/profit-sharing/referrer-orders/%d/share/", otherID), "referrer-1"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other user share: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 冻结窗口内 → 409
	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodPost,
		fmt.Sprintf("/api/billing/profit-sharing/referrer-orders/%d/share/", frozenID), "referrer-1"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("frozen share: status=%d body=%s", rec.Code, rec.Body.String())
	}

	failedFrozenID := seedReferrerOrder(t, tenantID, 3104, "referrer-1", psStatusFailed, daysAgo(5))
	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodPost,
		fmt.Sprintf("/api/billing/profit-sharing/referrer-orders/%d/share/", failedFrozenID), "referrer-1"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("failed-in-freeze share: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), referrerPSFailed) {
		t.Fatalf("failed-in-freeze share body=%s want %s", rec.Body.String(), referrerPSFailed)
	}

	// 非法 id → 400
	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodPost,
		"/api/billing/profit-sharing/referrer-orders/not-a-number/share/", "referrer-1"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 可分账窗口内：mock 微信分账出站，校验 200 + finished
	orig := createProfitSharingOrder
	createProfitSharingOrder = func(_ context.Context, _, _ string, _ string, _ []profitSharingReceiver) (string, error) {
		return "wx-ps-referrer-1", nil
	}
	t.Cleanup(func() { createProfitSharingOrder = orig })

	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodPost,
		fmt.Sprintf("/api/billing/profit-sharing/referrer-orders/%d/share/", shareableID), "referrer-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("shareable: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var psStatus string
	if err := db.QueryRow(`SELECT status FROM billing_profit_sharing WHERE id = ?`, shareableID).Scan(&psStatus); err != nil {
		t.Fatal(err)
	}
	if psStatus != psStatusFinished {
		t.Fatalf("status=%q want finished", psStatus)
	}
}

func TestReferrerOrderShareIdempotent(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	const tenantID int64 = 95103
	processingID := seedReferrerOrder(t, tenantID, 3201, "referrer-1", psStatusProcessing, daysAgo(20))
	finishedID := seedReferrerOrder(t, tenantID, 3202, "referrer-1", psStatusFinished, daysAgo(20))

	// processing 重放 → 200 idempotent，不再出站
	rec := httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodPost,
		fmt.Sprintf("/api/billing/profit-sharing/referrer-orders/%d/share/", processingID), "referrer-1"))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"idempotent":true`) {
		t.Fatalf("processing replay: status=%d body=%s", rec.Code, rec.Body.String())
	}
	// finished 重放 → 200 idempotent
	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerReq(http.MethodPost,
		fmt.Sprintf("/api/billing/profit-sharing/referrer-orders/%d/share/", finishedID), "referrer-1"))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"idempotent":true`) {
		t.Fatalf("finished replay: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReferrerShareChannelAggregatesWithoutLeakingOthers(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	const tenantID int64 = 95104
	frozenID := seedReferrerOrderOnChannel(t, tenantID, 3301, "referrer-1", psStatusPending, daysAgo(5), "CH-A")
	shareableID := seedReferrerOrderOnChannel(t, tenantID, 3302, "referrer-1", psStatusPending, daysAgo(20), "CH-A")
	_ = seedReferrerOrderOnChannel(t, tenantID, 3303, "referrer-1", psStatusPending, daysAgo(5), "CH-FROZEN")
	otherID := seedReferrerOrderOnChannel(t, tenantID, 3304, "referrer-2", psStatusPending, daysAgo(20), "CH-A")

	orig := createProfitSharingOrder
	createProfitSharingOrder = func(_ context.Context, _, _ string, _ string, _ []profitSharingReceiver) (string, error) {
		return "wx-ps-channel-1", nil
	}
	t.Cleanup(func() { createProfitSharingOrder = orig })

	rec := httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerJSONReq(http.MethodPost,
		"/api/billing/profit-sharing/referrer-orders/share-channel/", "referrer-1",
		`{"channel_code":"CH-MISSING"}`))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing channel: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerJSONReq(http.MethodPost,
		"/api/billing/profit-sharing/referrer-orders/share-channel/", "referrer-1",
		`{"channel_code":"CH-FROZEN"}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("frozen channel: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	handleReferrerProfitSharingOrders(rec, referrerJSONReq(http.MethodPost,
		"/api/billing/profit-sharing/referrer-orders/share-channel/", "referrer-1",
		`{"channel_code":"CH-A"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("share CH-A: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "ORD") || strings.Contains(rec.Body.String(), "order_number") {
		t.Fatalf("share leaked order identity: %s", rec.Body.String())
	}

	var frozenStatus, shareStatus, otherStatus string
	if err := db.QueryRow(`SELECT status FROM billing_profit_sharing WHERE id = ?`, frozenID).Scan(&frozenStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM billing_profit_sharing WHERE id = ?`, shareableID).Scan(&shareStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM billing_profit_sharing WHERE id = ?`, otherID).Scan(&otherStatus); err != nil {
		t.Fatal(err)
	}
	if frozenStatus != psStatusPending {
		t.Fatalf("frozen row status=%q want pending", frozenStatus)
	}
	if shareStatus != psStatusFinished {
		t.Fatalf("shareable row status=%q want finished", shareStatus)
	}
	if otherStatus != psStatusPending {
		t.Fatalf("other referrer row touched: %q", otherStatus)
	}
}
