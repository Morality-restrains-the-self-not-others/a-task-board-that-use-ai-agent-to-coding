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

func TestProfitSharingReceiverAddPayload_OmitsNameAndCustomRelation(t *testing.T) {
	t.Parallel()
	ident := wechatPayIdentity{
		AppID:  "wx_login_app",
		AppKey: "web",
		OpenID: "o-referrer-openid",
	}
	payload := profitSharingReceiverAddPayload(ident)
	if payload.Appid == nil || *payload.Appid != "wx_login_app" {
		t.Fatalf("appid=%v want identity appid (not pay mch appid)", payload.Appid)
	}
	if payload.Type == nil || string(*payload.Type) != "PERSONAL_OPENID" {
		t.Fatalf("type=%v", payload.Type)
	}
	if payload.Account == nil || *payload.Account != "o-referrer-openid" {
		t.Fatalf("account=%v", payload.Account)
	}
	if payload.RelationType == nil || string(*payload.RelationType) != "DISTRIBUTOR" {
		t.Fatalf("relation_type=%v", payload.RelationType)
	}
	if payload.Name != nil {
		t.Fatal("PERSONAL_OPENID name must be omitted unless encrypted KYC name is available")
	}
	if payload.CustomRelation != nil {
		t.Fatal("custom_relation is only valid when relation_type=CUSTOM")
	}
}

func TestProfitSharingReceiverAddPayload_SetsNameWhenLegalNamePresent(t *testing.T) {
	t.Parallel()
	payload := profitSharingReceiverAddPayload(wechatPayIdentity{
		AppID: "wx_login_app", AppKey: "web", OpenID: "o-referrer-openid", LegalName: "张三",
	})
	if payload.Name == nil || *payload.Name != "张三" {
		t.Fatalf("name=%v", payload.Name)
	}
}

func TestEnsureProfitSharingReceiver_RegistersOnce(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	wechatLiveOK = true
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", AppKey: "web", OpenID: "oid-1"}, nil
	}
	posts := 0
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error {
		posts++
		return nil
	}
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
	})

	first, err := ensureProfitSharingReceiver(context.Background(), "user-ps-1")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if first.Status != psReceiverRegistered {
		t.Fatalf("first status=%s reason=%s", first.Status, first.Reason)
	}
	second, err := ensureProfitSharingReceiver(context.Background(), "user-ps-1")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.Status != psReceiverRegistered {
		t.Fatalf("second status=%s", second.Status)
	}
	if posts != 1 {
		t.Fatalf("WeChat add called %d times, want 1 (idempotent skip)", posts)
	}
}

func TestEnsureProfitSharingReceiver_BackfillsReferralEdgeOpenid(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	wechatLiveOK = true
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", AppKey: "web", OpenID: "oid-1"}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error {
		return nil
	}
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
	})

	// 空 referrer_openid 的边：登记后应回填为接收方 openid
	if _, err := db.Exec(`
		INSERT INTO billing_referral_edge (referred_user_id, referrer_user_id, referrer_tenant_id, bound_at, commission_eligible, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, NOW(), NOW())`,
		"ref-buyer-1", "user-ps-edge-1", int64(123), "2026-08-01T00:00:00Z"); err != nil {
		t.Fatalf("seed empty edge: %v", err)
	}
	// 已有过期网站应用 openid 的边：登记后覆盖为支付/mp 身份
	if _, err := db.Exec(`
		INSERT INTO billing_referral_edge (referred_user_id, referrer_user_id, referrer_tenant_id, bound_at, referrer_openid, commission_eligible, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, NOW(), NOW())`,
		"ref-buyer-2", "user-ps-edge-1", int64(123), "2026-08-01T00:00:00Z", "existing-oid"); err != nil {
		t.Fatalf("seed existing edge: %v", err)
	}

	first, err := ensureProfitSharingReceiver(context.Background(), "user-ps-edge-1")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if first.Status != psReceiverRegistered {
		t.Fatalf("first status=%s reason=%s", first.Status, first.Reason)
	}
	// 幂等分支（already registered）也应回填
	second, err := ensureProfitSharingReceiver(context.Background(), "user-ps-edge-1")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.Status != psReceiverRegistered {
		t.Fatalf("second status=%s", second.Status)
	}

	var filled, existing string
	if err := db.QueryRow(`SELECT referrer_openid FROM billing_referral_edge WHERE referred_user_id = 'ref-buyer-1'`).Scan(&filled); err != nil {
		t.Fatalf("load filled edge: %v", err)
	}
	if filled != "oid-1" {
		t.Fatalf("empty edge referrer_openid=%q want oid-1", filled)
	}
	if err := db.QueryRow(`SELECT referrer_openid FROM billing_referral_edge WHERE referred_user_id = 'ref-buyer-2'`).Scan(&existing); err != nil {
		t.Fatalf("load existing edge: %v", err)
	}
	if existing != "oid-1" {
		t.Fatalf("existing edge referrer_openid=%q want oid-1 (overwrite stale web snapshot)", existing)
	}
}

func TestEnsureProfitSharingReceiver_PendingWhenNoOpenid(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	wechatLiveOK = true
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error {
		t.Fatal("must not call WeChat without openid")
		return nil
	}
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
	})

	got, err := ensureProfitSharingReceiver(context.Background(), "user-no-wx")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if got.Status != psReceiverPendingOpenid {
		t.Fatalf("status=%s reason=%s", got.Status, got.Reason)
	}
}

func TestEnsureProfitSharingReceiver_SkipsWhenWechatNotLive(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	wechatLiveOK = false
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", OpenID: "oid-1"}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error {
		t.Fatal("must not call WeChat when not live")
		return nil
	}
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
	})

	got, err := ensureProfitSharingReceiver(context.Background(), "user-dev")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if got.Status != psReceiverSkippedNotLive {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestEnsureProfitSharingReceiver_RecordsWechatFailure(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	wechatLiveOK = true
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", OpenID: "oid-fail"}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error {
		return errors.New("NO_AUTH appid not bound to mchid")
	}
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
	})

	got, err := ensureProfitSharingReceiver(context.Background(), "user-fail")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if got.Status != psReceiverFailed {
		t.Fatalf("status=%s", got.Status)
	}
	if !strings.Contains(got.Reason, "NO_AUTH") {
		t.Fatalf("reason=%s", got.Reason)
	}
}

func TestHandleInternalEnsureProfitSharingReceiver(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	origSecret, origGW := cfg.InternalSecret, cfg.GatewayInternalSecret
	cfg.InternalSecret, cfg.GatewayInternalSecret = "", ""
	t.Cleanup(func() {
		cfg.InternalSecret, cfg.GatewayInternalSecret = origSecret, origGW
	})

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	wechatLiveOK = true
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", OpenID: "oid-http"}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error { return nil }
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
	})

	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/profit-sharing/receivers/ensure/", strings.NewReader(`{"user_id":"user-http-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalEnsureProfitSharingReceiver(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["status"] != psReceiverRegistered {
		t.Fatalf("out=%v", out)
	}
	if out["user_id"] != "user-http-1" {
		t.Fatalf("user_id=%v", out["user_id"])
	}
}

func TestHandleInternalEnsureProfitSharingReceiver_RequiresUserID(t *testing.T) {
	origSecret, origGW := cfg.InternalSecret, cfg.GatewayInternalSecret
	cfg.InternalSecret, cfg.GatewayInternalSecret = "", ""
	t.Cleanup(func() {
		cfg.InternalSecret, cfg.GatewayInternalSecret = origSecret, origGW
	})
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/profit-sharing/receivers/ensure/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalEnsureProfitSharingReceiver(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestDeleteWechatProfitSharingReceiver_NoRegisteredRowSkipsWechat(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origDelete := deleteWechatReceiver
	wechatLiveOK = true
	deletes := 0
	deleteWechatReceiver = func(context.Context, string, string) error {
		deletes++
		return nil
	}
	t.Cleanup(func() {
		wechatLiveOK = origLive
		deleteWechatReceiver = origDelete
	})

	got, err := deleteWechatProfitSharingReceiver(context.Background(), "user-no-reg")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got.Status != psReceiverDeleted {
		t.Fatalf("status=%s reason=%s", got.Status, got.Reason)
	}
	if deletes != 0 {
		t.Fatalf("WeChat delete called %d times, want 0 for unregistered user", deletes)
	}
}

func TestDeleteWechatProfitSharingReceiver_DeletesRegistered(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	origDelete := deleteWechatReceiver
	wechatLiveOK = true
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", OpenID: "oid-1"}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error { return nil }
	deleteWechatReceiver = func(context.Context, string, string) error { return nil }
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
		deleteWechatReceiver = origDelete
	})

	// 先 ensure 落一行 registered
	ensure, err := ensureProfitSharingReceiver(context.Background(), "user-del-1")
	if err != nil || ensure.Status != psReceiverRegistered {
		t.Fatalf("ensure: %v status=%s", err, ensure.Status)
	}
	deletes := 0
	deleteWechatReceiver = func(_ context.Context, appid, openid string) error {
		deletes++
		if appid != "wxAAA" || openid != "oid-1" {
			t.Fatalf("delete called appid=%s openid=%s", appid, openid)
		}
		return nil
	}

	got, err := deleteWechatProfitSharingReceiver(context.Background(), "user-del-1")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got.Status != psReceiverDeleted {
		t.Fatalf("status=%s reason=%s", got.Status, got.Reason)
	}
	if deletes != 1 {
		t.Fatalf("WeChat delete called %d times, want 1", deletes)
	}
	row, ok := loadProfitSharingReceiver("user-del-1")
	if !ok || row.Status != psReceiverDeleted {
		t.Fatalf("receiver row after delete: %+v ok=%v", row, ok)
	}
}

func TestDeleteWechatProfitSharingReceiver_WechatFailureMarkedFailed(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	origDelete := deleteWechatReceiver
	wechatLiveOK = true
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", OpenID: "oid-fail"}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error { return nil }
	deleteWechatReceiver = func(context.Context, string, string) error {
		return errors.New("NO_AUTH appid not bound")
	}
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
		deleteWechatReceiver = origDelete
	})

	if _, err := ensureProfitSharingReceiver(context.Background(), "user-del-fail"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got, err := deleteWechatProfitSharingReceiver(context.Background(), "user-del-fail")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got.Status != psReceiverFailed {
		t.Fatalf("status=%s want %s", got.Status, psReceiverFailed)
	}
	if !strings.Contains(got.Reason, "NO_AUTH") {
		t.Fatalf("reason=%s", got.Reason)
	}
}

func TestDeleteWechatProfitSharingReceiver_NotLiveDoesNotCallWechat(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	origDelete := deleteWechatReceiver
	wechatLiveOK = false
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", OpenID: "oid-dev"}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error { return nil }
	deleteWechatReceiver = func(context.Context, string, string) error {
		t.Fatal("must not call WeChat when not live")
		return nil
	}
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
		deleteWechatReceiver = origDelete
	})

	got, err := deleteWechatProfitSharingReceiver(context.Background(), "user-del-dev")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got.Status != psReceiverDeleted {
		t.Fatalf("status=%s reason=%s", got.Status, got.Reason)
	}
}

func TestHandleInternalDeleteProfitSharingReceiver(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	origSecret, origGW := cfg.InternalSecret, cfg.GatewayInternalSecret
	cfg.InternalSecret, cfg.GatewayInternalSecret = "", ""
	t.Cleanup(func() {
		cfg.InternalSecret, cfg.GatewayInternalSecret = origSecret, origGW
	})

	origLive := wechatLiveOK
	origLookup := lookupWechatPayIdentity
	origPost := postProfitSharingReceiver
	origDelete := deleteWechatReceiver
	wechatLiveOK = true
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return wechatPayIdentity{AppID: "wxAAA", OpenID: "oid-http-del"}, nil
	}
	postProfitSharingReceiver = func(context.Context, wechatPayIdentity) error { return nil }
	deleteWechatReceiver = func(context.Context, string, string) error { return nil }
	t.Cleanup(func() {
		wechatLiveOK = origLive
		lookupWechatPayIdentity = origLookup
		postProfitSharingReceiver = origPost
		deleteWechatReceiver = origDelete
	})
	if _, err := ensureProfitSharingReceiver(context.Background(), "user-del-http"); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/profit-sharing/receivers/delete/", strings.NewReader(`{"user_id":"user-del-http"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalDeleteProfitSharingReceiver(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["status"] != psReceiverDeleted {
		t.Fatalf("out=%v", out)
	}
	if out["user_id"] != "user-del-http" {
		t.Fatalf("user_id=%v", out["user_id"])
	}
}

func TestHandleInternalDeleteProfitSharingReceiver_RequiresUserID(t *testing.T) {
	origSecret, origGW := cfg.InternalSecret, cfg.GatewayInternalSecret
	cfg.InternalSecret, cfg.GatewayInternalSecret = "", ""
	t.Cleanup(func() {
		cfg.InternalSecret, cfg.GatewayInternalSecret = origSecret, origGW
	})
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/profit-sharing/receivers/delete/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalDeleteProfitSharingReceiver(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestLookupWechatPayIdentityLive_ParsesAuthResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/users/id/u-1/wechat-pay-openid/" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("preferred_app_id") != "wxPAY" {
			t.Fatalf("preferred=%s", r.URL.Query().Get("preferred_app_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user_id":"u-1","app_id":"wxWEB","app_key":"web","openid":"oid-web"}`))
	}))
	t.Cleanup(srv.Close)

	origURL := cfg.TaskAuthBaseURL
	cfg.TaskAuthBaseURL = srv.URL
	origApp := wechatCfg.Appid
	wechatCfg.Appid = "wxPAY"
	t.Cleanup(func() {
		cfg.TaskAuthBaseURL = origURL
		wechatCfg.Appid = origApp
	})

	got, err := lookupWechatPayIdentityLive(context.Background(), "u-1", "wxPAY")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got.OpenID != "oid-web" || got.AppID != "wxWEB" {
		t.Fatalf("got=%+v", got)
	}
}
