package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func seedProfitSharingRecordStatus(t *testing.T, orderID, tenantID int64, status, settleAfter string) int64 {
	t.Helper()
	id := seedProfitSharingRecord(t, orderID, tenantID)
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET status = ?, settle_after = ? WHERE id = ?`,
		status, settleAfter, id); err != nil {
		t.Fatalf("update profit sharing status: %v", err)
	}
	return id
}

func seedAdminProfitSharingRow(t *testing.T, status, settleAfter string) (orderID int64, psID int64) {
	t.Helper()
	tenantID := generateSnowflakeID()
	orderID = generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WXPSA"+formatID(orderID), "420000"+formatID(orderID))
	psID = seedProfitSharingRecordStatus(t, orderID, tenantID, status, settleAfter)
	return orderID, psID
}

func TestHandleSystemAdminListProfitSharingDefaultOpenOmitsFinished(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	seedAdminProfitSharingRow(t, psStatusPending, "2026-08-01T00:00:00Z")
	seedAdminProfitSharingRow(t, psStatusProcessing, "2026-08-02T00:00:00Z")
	seedAdminProfitSharingRow(t, psStatusFailed, "2026-08-03T00:00:00Z")
	finishedOrder, _ := seedAdminProfitSharingRow(t, psStatusFinished, "2026-07-01T00:00:00Z")

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	total, _ := body["total"].(float64)
	if int(total) != 3 {
		t.Fatalf("total=%v want 3 body=%v", body["total"], body)
	}
	items, _ := body["items"].([]interface{})
	if len(items) != 3 {
		t.Fatalf("items len=%d want 3", len(items))
	}
	finishedNum := "ORD" + formatID(finishedOrder)
	for _, raw := range items {
		row, _ := raw.(map[string]interface{})
		if row["order_number"] == finishedNum {
			t.Fatalf("finished row leaked into open filter: %v", row)
		}
		if _, has := row["referrer_openid"]; has {
			t.Fatal("referrer_openid key must not appear")
		}
		if _, has := row["app_id"]; !has {
			t.Fatal("app_id must appear for platform staff")
		}
		if got, _ := row["openid"].(string); got != "openid-1" {
			t.Fatalf("openid=%v want snapshot openid-1", row["openid"])
		}
		if _, has := row["wechat_transaction_id"]; !has {
			t.Fatal("wechat_transaction_id must appear for platform staff")
		}
		if _, has := row["wechat_profit_sharing_id"]; !has {
			t.Fatal("wechat_profit_sharing_id must appear for platform staff")
		}
		if _, has := row["out_trade_no"]; !has {
			t.Fatal("out_trade_no must appear for platform staff")
		}
		if _, has := row["out_profit_sharing_no"]; !has {
			t.Fatal("out_profit_sharing_no must appear for platform staff")
		}
	}
}

func TestHandleSystemAdminListProfitSharingStatusPending(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	pendingOrder, _ := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-10T00:00:00Z")
	seedAdminProfitSharingRow(t, psStatusFailed, "2026-08-11T00:00:00Z")

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?status=pending")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["status"] != psStatusPending {
		t.Fatalf("status=%v", row["status"])
	}
	if row["order_number"] != "ORD"+formatID(pendingOrder) {
		t.Fatalf("order_number=%v", row["order_number"])
	}
}

func TestHandleSystemAdminListProfitSharingAuth(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/profit-sharing/", nil)
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no auth status=%d want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/system-admin/profit-sharing/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec = httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member status=%d want 403", rec.Code)
	}
}

func TestHandleSystemAdminListProfitSharingInvalidStatus(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?status=bogus")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSystemAdminListProfitSharingSettleAfterOrderAndPaging(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	late, _ := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-30T00:00:00Z")
	early, _ := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-01T00:00:00Z")
	mid, _ := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-15T00:00:00Z")

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?limit=1&offset=0")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	if int(body["total"].(float64)) != 3 {
		t.Fatalf("total=%v", body["total"])
	}
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("page0 items=%v", items)
	}
	row0, _ := items[0].(map[string]interface{})
	if row0["order_number"] != "ORD"+formatID(early) {
		t.Fatalf("first=%v want earliest %s", row0["order_number"], formatID(early))
	}

	req = staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?limit=1&offset=1")
	rec = httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	body = decodeJSONMap(t, rec)
	items, _ = body["items"].([]interface{})
	row1, _ := items[0].(map[string]interface{})
	if row1["order_number"] != "ORD"+formatID(mid) {
		t.Fatalf("second=%v want mid %s", row1["order_number"], formatID(mid))
	}

	req = staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?limit=1&offset=2")
	rec = httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	body = decodeJSONMap(t, rec)
	items, _ = body["items"].([]interface{})
	row2, _ := items[0].(map[string]interface{})
	if row2["order_number"] != "ORD"+formatID(late) {
		t.Fatalf("third=%v want late %s", row2["order_number"], formatID(late))
	}
}

func TestParseProfitSharingStatusFilter(t *testing.T) {
	got, err := parseProfitSharingStatusFilter("")
	if err != nil || fmt.Sprint(got) != fmt.Sprint([]string{psStatusPending, psStatusProcessing, psStatusFailed}) {
		t.Fatalf("default open: %v %v", got, err)
	}
	all, err := parseProfitSharingStatusFilter("all")
	if err != nil || all != nil {
		t.Fatalf("all: %v %v", all, err)
	}
	if _, err := parseProfitSharingStatusFilter("nope"); err == nil {
		t.Fatal("want invalid status error")
	}
}

func TestHandleSystemAdminListProfitSharingReturnsWechatIDs(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	orderID, psID := seedAdminProfitSharingRow(t, psStatusFailed, "2026-08-30T00:00:00Z")
	wantTxn := "420000" + formatID(orderID)
	wantPS := "300845" + formatID(psID)
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET wechat_profit_sharing_id = ? WHERE id = ?`,
		wantPS, psID); err != nil {
		t.Fatalf("set wechat ps id: %v", err)
	}

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?status=failed")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["wechat_transaction_id"] != wantTxn {
		t.Fatalf("wechat_transaction_id=%v want %s", row["wechat_transaction_id"], wantTxn)
	}
	if row["wechat_profit_sharing_id"] != wantPS {
		t.Fatalf("wechat_profit_sharing_id=%v want %s", row["wechat_profit_sharing_id"], wantPS)
	}
}

func TestHandleSystemAdminListProfitSharingReturnsMerchantNos(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	orderID, psID := seedAdminProfitSharingRow(t, psStatusFailed, "2026-08-30T00:00:00Z")
	wantOutTrade := "WX" + formatID(orderID)
	if _, err := db.Exec(`UPDATE billing_resource_order SET out_trade_no = ? WHERE id = ?`,
		wantOutTrade, orderID); err != nil {
		t.Fatalf("set out_trade_no: %v", err)
	}
	var wantOutPS string
	if err := db.QueryRow(`SELECT out_profit_sharing_no FROM billing_profit_sharing WHERE id = ?`, psID).
		Scan(&wantOutPS); err != nil {
		t.Fatalf("load out_profit_sharing_no: %v", err)
	}
	if wantOutPS == "" {
		t.Fatal("seed out_profit_sharing_no empty")
	}

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?status=failed")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["out_trade_no"] != wantOutTrade {
		t.Fatalf("out_trade_no=%v want %s", row["out_trade_no"], wantOutTrade)
	}
	if row["out_profit_sharing_no"] != wantOutPS {
		t.Fatalf("out_profit_sharing_no=%v want %s", row["out_profit_sharing_no"], wantOutPS)
	}
}

func TestHandleSystemAdminListProfitSharingOutTradeNoFallsBackToPaymentRef(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	orderID, _ := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-30T00:00:00Z")
	want := "WXPSA" + formatID(orderID)

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?status=pending")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["out_trade_no"] != want {
		t.Fatalf("out_trade_no=%v want payment_ref fallback %s", row["out_trade_no"], want)
	}
}

func TestHandleSystemAdminListProfitSharingWechatStateNotSubmitted(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, unsubmittedPS := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-10T00:00:00Z")
	_, submittedPS := seedAdminProfitSharingRow(t, psStatusFailed, "2026-08-11T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET wechat_profit_sharing_id = ? WHERE id = ?`,
		"300845"+formatID(submittedPS), submittedPS); err != nil {
		t.Fatalf("set wechat ps id: %v", err)
	}

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?status=all")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("items=%v", items)
	}
	notSubmitted := 0
	for _, raw := range items {
		row, _ := raw.(map[string]interface{})
		psID, _ := row["wechat_profit_sharing_id"].(string)
		state, _ := row["wechat_state"].(string)
		if psID == "" {
			notSubmitted++
			if state != wechatProfitSharingStateNotSubmitted {
				t.Fatalf("row %v: wechat_state=%q want %q", row["id"], state, wechatProfitSharingStateNotSubmitted)
			}
		} else if state != "" {
			t.Fatalf("row %v: submitted wechat_state=%q want empty", row["id"], state)
		}
	}
	if notSubmitted != 1 {
		t.Fatalf("not-submitted rows=%d want 1 (id=%s)", notSubmitted, formatID(unsubmittedPS))
	}
}
