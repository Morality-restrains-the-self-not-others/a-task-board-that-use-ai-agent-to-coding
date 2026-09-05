package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func seedTradeNoOrder(t *testing.T, id, tenantID int64, orderNumber, paymentRef string) {
	t.Helper()
	created := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at
		) VALUES (?, ?, ?, 'paid', 1000, 'wechat', ?, ?)`,
		id, tenantID, orderNumber, paymentRef, created,
	)
	if err != nil {
		t.Fatalf("seedTradeNoOrder: %v", err)
	}
}

func adminTradeNoURL(orderNumber string) string {
	q := url.Values{}
	q.Set("order_number", orderNumber)
	q.Set("limit", "15")
	return "/api/system-admin/orders/?" + q.Encode()
}

func decodeOrderList(t *testing.T, rec *httptest.ResponseRecorder) (int, []map[string]interface{}) {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	total := 0
	switch v := body["total"].(type) {
	case float64:
		total = int(v)
	}
	raw, _ := body["orders"].([]interface{})
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if ok {
			out = append(out, m)
		}
	}
	return total, out
}

func TestListOrdersByTradeNoMatchesOrderNumber(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	num := fmt.Sprintf("ORD-20260821-%d-%d", int64(880201), id)
	seedTradeNoOrder(t, id, 880201, num, "wx-other")
	seedTradeNoOrder(t, generateSnowflakeID(), 880202, "ORD-NOISE-1", "wx-noise")

	orders, total, err := listOrdersByTradeNo(0, num, "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != id {
		t.Fatalf("order_number match total=%d n=%d id=%v want 1/1/%d", total, len(orders), orders, id)
	}
}

func TestListOrdersByTradeNoMatchesID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	seedTradeNoOrder(t, id, 880203, "ORD-ID-TARGET", "")

	orders, total, err := listOrdersByTradeNo(0, formatID(id), "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != id {
		t.Fatalf("id match total=%d n=%d want 1/1/%d", total, len(orders), id)
	}
}

func TestListOrdersByTradeNoMatchesPaymentRef(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	seedTradeNoOrder(t, id, 880204, "ORD-PAY-TARGET", "wx_txn_trade_no_001")

	orders, total, err := listOrdersByTradeNo(0, "wx_txn_trade_no_001", "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != id {
		t.Fatalf("payment_ref match total=%d n=%d want %d", total, len(orders), id)
	}
}

func TestListOrdersByTradeNoMissReturnsEmpty(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	seedTradeNoOrder(t, generateSnowflakeID(), 880205, "ORD-EXISTS", "wx-exists")

	orders, total, err := listOrdersByTradeNo(0, "ORD-NOT-FOUND-XYZ", "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 0 || len(orders) != 0 {
		t.Fatalf("miss total=%d n=%d want 0", total, len(orders))
	}
}

func TestListOrdersByTradeNoThreeSegmentDoesNotMatchForeignID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	foreignID := int64(555)
	seedTradeNoOrder(t, foreignID, 880206, "ORD-FOREIGN-PK", "")
	wantID := generateSnowflakeID()
	seedTradeNoOrder(t, wantID, 880207, "ORD-20260820-555", "")

	orders, total, err := listOrdersByTradeNo(0, "ORD-20260820-555", "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != wantID {
		t.Fatalf("three-segment must not use NNN as id: total=%d n=%d id=%v want %d", total, len(orders), orders, wantID)
	}
}

func TestListOrdersByTradeNoTenantScoped(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	num := "ORD-SHARED-DISPLAY-1"
	seedTradeNoOrder(t, generateSnowflakeID(), 880208, num+"-A", "")
	otherID := generateSnowflakeID()
	seedTradeNoOrder(t, otherID, 880209, num, "wx-b")

	orders, total, err := listOrdersByTradeNo(880208, num, "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 0 || len(orders) != 0 {
		t.Fatalf("tenant 880208 should not see tenant 880209 order: total=%d", total)
	}

	orders, total, err = listOrdersByTradeNo(880209, num, "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo tenant: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != otherID {
		t.Fatalf("own tenant miss: total=%d id=%v want %d", total, orders, otherID)
	}
}

func TestNormalizeTradeNoQueryTooLong(t *testing.T) {
	_, err := normalizeTradeNoQuery(strings.Repeat("x", 129))
	if err == nil {
		t.Fatal("expected error for 129 chars")
	}
}

func TestDoAdminListOrdersFiltersByOrderNumber(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	num := fmt.Sprintf("ORD-20260821-%d-%d", int64(880210), id)
	seedTradeNoOrder(t, id, 880210, num, "")
	seedTradeNoOrder(t, generateSnowflakeID(), 880211, "ORD-NOISE-ADMIN", "")

	req := httptest.NewRequest(http.MethodGet, adminTradeNoURL(num), nil)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders := decodeOrderList(t, rec)
	if total != 1 || len(orders) != 1 || orders[0]["id"] != formatID(id) {
		t.Fatalf("admin list trade no: total=%d orders=%v want id=%s", total, orders, formatID(id))
	}
}

func TestDoAdminListOrdersTradeNoTooLong400(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	req := httptest.NewRequest(http.MethodGet, adminTradeNoURL(strings.Repeat("n", 129)), nil)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSystemAdminListOrdersTradeNoAuth(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, adminTradeNoURL("ORD-X"), nil)
	rec := httptest.NewRecorder()
	handleSystemAdminListOrders(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no auth status=%d want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, adminTradeNoURL("ORD-X"), nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec = httptest.NewRecorder()
	handleSystemAdminListOrders(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member status=%d want 403 body=%s", rec.Code, rec.Body.String())
	}
}

func tenantTradeNoURL(tid int64, orderNumber string) string {
	q := url.Values{}
	q.Set("order_number", orderNumber)
	q.Set("limit", "15")
	return fmt.Sprintf("/api/tenant/%d/billing/orders/?%s", tid, q.Encode())
}

func seedWechatVoucherOrder(t *testing.T, id, tenantID int64, orderNumber, txn, outNo string) {
	t.Helper()
	seedTradeNoOrder(t, id, tenantID, orderNumber, "wechat:"+outNo)
	if _, err := db.Exec(
		`UPDATE billing_resource_order SET wechat_transaction_id = ?, out_trade_no = ? WHERE id = ?`,
		txn, outNo, id,
	); err != nil {
		t.Fatalf("seed wechat vouchers: %v", err)
	}
}

func TestHandleListOrdersTradeNoDoesNotLeakOtherTenant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	num := "ORD-TENANT-LEAK-CHECK"
	seedTradeNoOrder(t, id, 880212, num, "")

	req := httptest.NewRequest(http.MethodGet, tenantTradeNoURL(880299, num), nil)
	rec := httptest.NewRecorder()
	handleListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders := decodeOrderList(t, rec)
	if total != 0 || len(orders) != 0 {
		t.Fatalf("cross-tenant leak total=%d orders=%v", total, orders)
	}
}

func TestHandleListOrdersMatchesWechatTransactionIDInTenant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	tid := int64(880410)
	txn := "4500000359202608291274536815"
	outNo := "WX878981209491801064"
	seedWechatVoucherOrder(t, id, tid, "ORD-TENANT-TXN", txn, outNo)

	req := httptest.NewRequest(http.MethodGet, tenantTradeNoURL(tid, txn), nil)
	rec := httptest.NewRecorder()
	handleListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders := decodeOrderList(t, rec)
	if total != 1 || len(orders) != 1 || orders[0]["id"] != formatID(id) {
		t.Fatalf("tenant wechat txn: total=%d orders=%v want id=%s", total, orders, formatID(id))
	}
}

func TestHandleListOrdersMatchesMerchantOutTradeNoInTenant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	tid := int64(880411)
	txn := "4500000359202608291274536816"
	outNo := "WX878981209491801065"
	seedWechatVoucherOrder(t, id, tid, "ORD-TENANT-OUT", txn, outNo)

	req := httptest.NewRequest(http.MethodGet, tenantTradeNoURL(tid, outNo), nil)
	rec := httptest.NewRecorder()
	handleListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders := decodeOrderList(t, rec)
	if total != 1 || len(orders) != 1 || orders[0]["id"] != formatID(id) {
		t.Fatalf("tenant out_trade_no: total=%d orders=%v want id=%s", total, orders, formatID(id))
	}
}

func TestHandleListOrdersWechatTxnDoesNotLeakOtherTenant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	txn := "4500000359202608291274536999"
	outNo := "WX878981209491809999"
	seedWechatVoucherOrder(t, id, 880412, "ORD-TXN-LEAK", txn, outNo)

	req := httptest.NewRequest(http.MethodGet, tenantTradeNoURL(880499, txn), nil)
	rec := httptest.NewRecorder()
	handleListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders := decodeOrderList(t, rec)
	if total != 0 || len(orders) != 0 {
		t.Fatalf("wechat txn cross-tenant leak total=%d orders=%v", total, orders)
	}

	req = httptest.NewRequest(http.MethodGet, tenantTradeNoURL(880499, outNo), nil)
	rec = httptest.NewRecorder()
	handleListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders = decodeOrderList(t, rec)
	if total != 0 || len(orders) != 0 {
		t.Fatalf("out_trade_no cross-tenant leak total=%d orders=%v", total, orders)
	}
}

func TestNormalizeTradeNoQueryStripsWechatExportBacktick(t *testing.T) {
	got, err := normalizeTradeNoQuery(" `4500000359202608221274536815 ")
	if err != nil {
		t.Fatalf("normalizeTradeNoQuery: %v", err)
	}
	if got != "4500000359202608221274536815" {
		t.Fatalf("got %q", got)
	}
}

func TestListOrdersByTradeNoMatchesMerchantOutTradeNo(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	outNo := "WX878981209491800064"
	seedTradeNoOrder(t, id, 880301, "ORD-WX-MERCHANT", "wechat:"+outNo)

	orders, total, err := listOrdersByTradeNo(0, outNo, "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != id {
		t.Fatalf("out_trade_no match total=%d n=%d id=%v want %d", total, len(orders), orders, id)
	}
}

func TestListOrdersByTradeNoMatchesWechatTransactionID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	txn := "4500000359202608221274536815"
	outNo := "WX878981209491800064"
	seedTradeNoOrder(t, id, 880302, "ORD-WX-TXN", "wechat:"+outNo)
	if _, err := db.Exec(
		`UPDATE billing_resource_order SET wechat_transaction_id = ?, out_trade_no = ? WHERE id = ?`,
		txn, outNo, id,
	); err != nil {
		t.Fatalf("seed txn: %v", err)
	}

	orders, total, err := listOrdersByTradeNo(0, txn, "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != id {
		t.Fatalf("wechat_transaction_id match total=%d n=%d want %d", total, len(orders), id)
	}
}

func TestDoAdminListOrdersByWechatExportNumbers(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	txn := "4500000359202608221274536815"
	outNo := "WX878981209491800064"
	seedTradeNoOrder(t, id, 880303, "ORD-WX-EXPORT", "wechat:"+outNo)
	if _, err := db.Exec(
		`UPDATE billing_resource_order SET wechat_transaction_id = ?, out_trade_no = ? WHERE id = ?`,
		txn, outNo, id,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, adminTradeNoURL("`"+txn), nil)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders := decodeOrderList(t, rec)
	if total != 1 || len(orders) != 1 || orders[0]["id"] != formatID(id) {
		t.Fatalf("admin wechat txn: total=%d orders=%v want id=%s", total, orders, formatID(id))
	}

	req = httptest.NewRequest(http.MethodGet, adminTradeNoURL(outNo), nil)
	rec = httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	total, orders = decodeOrderList(t, rec)
	if total != 1 || len(orders) != 1 || orders[0]["id"] != formatID(id) {
		t.Fatalf("admin out_trade_no: total=%d orders=%v want id=%s", total, orders, formatID(id))
	}
}

func TestListOrdersByTradeNoResolvesWechatTxnViaAPI(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	id := generateSnowflakeID()
	outNo := "WX878234502453821440"
	txn := "4500000316202608201339559194"
	seedTradeNoOrder(t, id, 880304, "ORD-WX-API", "wechat:"+outNo)

	orig := resolveWechatTxnToOutTradeNo
	resolveWechatTxnToOutTradeNo = func(got string) (string, error) {
		if got != txn {
			t.Fatalf("resolve txn=%q want %s", got, txn)
		}
		return outNo, nil
	}
	t.Cleanup(func() { resolveWechatTxnToOutTradeNo = orig })

	orders, total, err := listOrdersByTradeNo(0, txn, "", 15, 0)
	if err != nil {
		t.Fatalf("listOrdersByTradeNo: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != id {
		t.Fatalf("api resolve total=%d n=%d want %d", total, len(orders), id)
	}
	var stored string
	if err := db.QueryRow(`SELECT wechat_transaction_id FROM billing_resource_order WHERE id = ?`, id).Scan(&stored); err != nil {
		t.Fatalf("load txn: %v", err)
	}
	if stored != txn {
		t.Fatalf("persisted txn=%q want %s", stored, txn)
	}
}
