package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleProfitSharingChangeNotifyMockAppliesAndDedups(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	origMode := wechatCfg.Mode
	wechatCfg.Mode = "mock"
	t.Cleanup(func() { wechatCfg.Mode = origMode })

	_, psID := seedAdminProfitSharingRow(t, psStatusProcessing, "2026-08-20T00:00:00Z")
	var outNo string
	if err := db.QueryRow(`SELECT out_profit_sharing_no FROM billing_profit_sharing WHERE id = ?`, psID).Scan(&outNo); err != nil {
		t.Fatalf("out no: %v", err)
	}

	payload := map[string]interface{}{
		"id":             "EV-ps-change-1",
		"event_type":     "TRANSACTION.SUCCESS",
		"out_order_no":   outNo,
		"transaction_id": "42000099990001",
		"order_id":       "1217752501201407033233368018",
		"success_time":   "2026-08-25T10:00:00+08:00",
	}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/billing/profitsharing/change-notify/", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleProfitSharingNotify(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp["code"] != "SUCCESS" {
		t.Fatalf("resp=%s", rec.Body.String())
	}

	var status, wechatID string
	if err := db.QueryRow(`SELECT status, COALESCE(wechat_profit_sharing_id,'') FROM billing_profit_sharing WHERE id = ?`, psID).
		Scan(&status, &wechatID); err != nil {
		t.Fatalf("row: %v", err)
	}
	if status != psStatusFinished || wechatID != "1217752501201407033233368018" {
		t.Fatalf("status=%s wechatID=%s", status, wechatID)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/billing/profitsharing/change-notify/", strings.NewReader(string(raw)))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handleProfitSharingNotify(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("dup status=%d", rec2.Code)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_profit_sharing_change_notify WHERE notify_id = ?`, "EV-ps-change-1").Scan(&n); err != nil {
		t.Fatalf("inbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("inbox rows=%d want 1", n)
	}
}

func TestHandleProfitSharingChangeNotifyUnknownOutOrderStillSuccess(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	origMode := wechatCfg.Mode
	wechatCfg.Mode = "mock"
	t.Cleanup(func() { wechatCfg.Mode = origMode })

	payload := `{"id":"EV-unknown-1","event_type":"TRANSACTION.SUCCESS","out_order_no":"PS-not-exist","transaction_id":"4200","order_id":"1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/billing/profitsharing/change-notify/", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	handleProfitSharingNotify(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleProfitSharingChangeNotifyLiveWithoutHandlerFails(t *testing.T) {
	origMode, origH := wechatCfg.Mode, wechatNotifyH
	wechatCfg.Mode = "live"
	wechatNotifyH = nil
	t.Cleanup(func() {
		wechatCfg.Mode = origMode
		wechatNotifyH = origH
	})
	req := httptest.NewRequest(http.MethodPost, "/api/billing/profitsharing/change-notify/", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handleProfitSharingNotify(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("want non-success, body=%s", rec.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["code"] == "SUCCESS" {
		t.Fatalf("code=%v", resp)
	}
}
