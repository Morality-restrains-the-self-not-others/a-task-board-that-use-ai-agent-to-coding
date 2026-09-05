package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLookupWechatPayOpenID_PrefersMatchingAppID(t *testing.T) {
	setupAuthTestDB(t)
	u1 := mustCreateWechatTestUser(t, "u-pay-openid-001")
	now := timeNowUTC()
	upsertWechatIdentity(u1, "web", "wxWEB", "o-web-001", "union-1", "n", "", now)
	upsertWechatIdentity(u1, "inapp", "wxPAY", "o-pay-001", "union-1", "n", "", now)

	row, ok, err := lookupWechatPayOpenID(u1, "wxPAY")
	if err != nil || !ok {
		t.Fatalf("lookup: ok=%v err=%v", ok, err)
	}
	if row.AppID != "wxPAY" || row.OpenID != "o-pay-001" {
		t.Fatalf("got %+v", row)
	}
}

func TestLookupWechatPayOpenID_FallsBackToAnyBoundApp(t *testing.T) {
	setupAuthTestDB(t)
	u1 := mustCreateWechatTestUser(t, "u-pay-openid-002")
	upsertWechatIdentity(u1, "web", "wxWEB", "o-web-002", "", "n", "", timeNowUTC())

	row, ok, err := lookupWechatPayOpenID(u1, "wxPAY")
	if err != nil || !ok {
		t.Fatalf("lookup: ok=%v err=%v", ok, err)
	}
	if row.AppID != "wxWEB" || row.OpenID != "o-web-002" {
		t.Fatalf("got %+v", row)
	}
}

func TestLookupWechatPayOpenID_Missing(t *testing.T) {
	setupAuthTestDB(t)
	u1 := mustCreateWechatTestUser(t, "u-pay-openid-003")
	_, ok, err := lookupWechatPayOpenID(u1, "wxPAY")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if ok {
		t.Fatal("expected missing identity")
	}
}

func TestHandleInternalWechatPayOpenID(t *testing.T) {
	setupAuthTestDB(t)
	origSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() { cfg.InternalSecret = origSecret })
	u1 := mustCreateWechatTestUser(t, "u-pay-openid-004")
	upsertWechatIdentity(u1, "web", "wxWEB", "o-web-004", "", "n", "", timeNowUTC())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/internal/users/id/{user_id}/wechat-pay-openid/", handleInternalWechatPayOpenID)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/users/id/"+u1+"/wechat-pay-openid/?preferred_app_id=wxPAY", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["openid"] != "o-web-004" || out["app_id"] != "wxWEB" || out["user_id"] != u1 {
		t.Fatalf("out=%v", out)
	}
}

func TestHandleInternalWechatPayOpenID_NotFound(t *testing.T) {
	setupAuthTestDB(t)
	origSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() { cfg.InternalSecret = origSecret })
	u1 := mustCreateWechatTestUser(t, "u-pay-openid-005")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/internal/users/id/{user_id}/wechat-pay-openid/", handleInternalWechatPayOpenID)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/users/id/"+u1+"/wechat-pay-openid/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}
