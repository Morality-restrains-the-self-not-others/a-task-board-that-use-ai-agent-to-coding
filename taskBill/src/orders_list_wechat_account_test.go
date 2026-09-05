package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func seedWechatAccountOrder(t *testing.T, id, tenantID, userID int64, orderNumber, payOpenID, payUnionID string) {
	t.Helper()
	created := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, user_id, pay_openid, pay_unionid, created_at
		) VALUES (?, ?, ?, 'paid', 1000, 'wechat', '', ?, ?, ?, ?)`,
		id, tenantID, orderNumber, userID, payOpenID, payUnionID, created,
	)
	if err != nil {
		t.Fatalf("seedWechatAccountOrder: %v", err)
	}
}

func TestListOrdersByWechatAccountMatchesUserIDAndPayIdentity(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	uid := int64(880801)
	idUser := generateSnowflakeID()
	idPay := generateSnowflakeID()
	idNoise := generateSnowflakeID()
	seedWechatAccountOrder(t, idUser, 880201, uid, "ORD-WA-USER", "", "")
	seedWechatAccountOrder(t, idPay, 880202, 0, "ORD-WA-PAY", "oid-pay-1", "uid-pay-1")
	seedWechatAccountOrder(t, idNoise, 880203, 880999, "ORD-WA-NOISE", "other-oid", "other-uid")

	orders, total, err := listOrdersByWechatAccount([]int64{uid}, "oid-pay-1", "", 15, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(orders) != 2 {
		t.Fatalf("total=%d n=%d", total, len(orders))
	}
	got := map[int64]bool{}
	for _, o := range orders {
		got[o.ID] = true
	}
	if !got[idUser] || !got[idPay] || got[idNoise] {
		t.Fatalf("ids=%v", got)
	}

	orders, total, err = listOrdersByWechatAccount(nil, "uid-pay-1", "", 15, 0)
	if err != nil {
		t.Fatalf("unionid list: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != idPay {
		t.Fatalf("unionid total=%d orders=%v", total, orders)
	}
}

func TestDoAdminListOrdersWechatAccount(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	uid := int64(880811)
	id := generateSnowflakeID()
	seedWechatAccountOrder(t, id, 880211, uid, "ORD-WA-HTTP", "", "")

	prev := resolveWechatLinkedUserIDs
	resolveWechatLinkedUserIDs = func(ctx context.Context, q string) ([]string, error) {
		if q != "微信昵称甲" {
			t.Fatalf("q=%s", q)
		}
		return []string{fmt.Sprintf("%d", uid)}, nil
	}
	t.Cleanup(func() { resolveWechatLinkedUserIDs = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/?wechat_account="+url.QueryEscape("微信昵称甲")+"&limit=15", nil)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders := decodeOrderList(t, rec)
	if total != 1 || len(orders) != 1 || orders[0]["id"] != formatID(id) {
		t.Fatalf("total=%d orders=%v", total, orders)
	}
}

func TestDoAdminListOrdersWechatAccountEmpty(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	prev := resolveWechatLinkedUserIDs
	resolveWechatLinkedUserIDs = func(ctx context.Context, q string) ([]string, error) {
		return []string{}, nil
	}
	t.Cleanup(func() { resolveWechatLinkedUserIDs = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/?wechat_account=nobody", nil)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	total, orders := decodeOrderList(t, rec)
	if total != 0 || len(orders) != 0 {
		t.Fatalf("total=%d orders=%v", total, orders)
	}
}

func TestDoAdminListOrdersWechatAccountMutexAndTooLong(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/?order_number=ORD-1&wechat_account=wx", nil)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("mutex status=%d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/?wechat_account="+strings.Repeat("你", 129), nil)
	rec = httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("too long status=%d", rec.Code)
	}
}

func TestDoAdminListOrdersWechatAccountAuthLookupFails502(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	prev := resolveWechatLinkedUserIDs
	resolveWechatLinkedUserIDs = func(ctx context.Context, q string) ([]string, error) {
		return nil, fmt.Errorf("taskAuth wechat-linked-account status 500")
	}
	t.Cleanup(func() { resolveWechatLinkedUserIDs = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/?wechat_account=wxuser", nil)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d want 502 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSystemAdminListOrdersWechatAccountAuth(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/?wechat_account=wx", nil)
	rec := httptest.NewRecorder()
	handleSystemAdminListOrders(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no auth status=%d want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/?wechat_account=wx", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec = httptest.NewRecorder()
	handleSystemAdminListOrders(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member status=%d want 403", rec.Code)
	}
}

func TestLookupWechatLinkedUserIDsLiveParsesAuthResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/users/wechat-linked-account/" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "微信昵称甲" {
			t.Fatalf("q=%s", r.URL.Query().Get("q"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user_ids":["880801"]}`))
	}))
	t.Cleanup(srv.Close)
	prev := cfg.TaskAuthBaseURL
	cfg.TaskAuthBaseURL = srv.URL
	t.Cleanup(func() { cfg.TaskAuthBaseURL = prev })

	ids, err := lookupWechatLinkedUserIDsLive(context.Background(), "微信昵称甲")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(ids) != 1 || ids[0] != "880801" {
		t.Fatalf("ids=%v", ids)
	}
}

func TestBuildOpenAPIJSONContainsWechatAccount(t *testing.T) {
	body, err := buildOpenAPIJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "wechat_account") {
		t.Fatal("public openapi missing wechat_account")
	}
}
