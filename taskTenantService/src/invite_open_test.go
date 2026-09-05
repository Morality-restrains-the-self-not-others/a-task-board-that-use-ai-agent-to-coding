package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseInviteLinkKindDefaultsSingle(t *testing.T) {
	kind, maxUses, err := parseInviteLinkKind(map[string]interface{}{}, "link")
	if err != nil || kind != inviteLinkKindSingle || maxUses != 1 {
		t.Fatalf("got kind=%s max=%d err=%v", kind, maxUses, err)
	}
}

func TestParseInviteLinkKindOpenUnlimited(t *testing.T) {
	kind, maxUses, err := parseInviteLinkKind(map[string]interface{}{"link_kind": "open"}, "link")
	if err != nil || kind != inviteLinkKindOpen || maxUses != 0 {
		t.Fatalf("got kind=%s max=%d err=%v", kind, maxUses, err)
	}
}

func TestParseInviteLinkKindOpenCapped(t *testing.T) {
	kind, maxUses, err := parseInviteLinkKind(map[string]interface{}{"link_kind": "open", "max_uses": float64(3)}, "link")
	if err != nil || kind != inviteLinkKindOpen || maxUses != 3 {
		t.Fatalf("got kind=%s max=%d err=%v", kind, maxUses, err)
	}
}

func TestParseInviteLinkKindOpenRejectedForEmail(t *testing.T) {
	_, _, err := parseInviteLinkKind(map[string]interface{}{"link_kind": "open"}, "email")
	if err == nil || err.Error() != "open_invite_link_only" {
		t.Fatalf("expected open_invite_link_only, got %v", err)
	}
}

func TestInviteLinkResponseDefaultSingle(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","company_member_name":"One","role":"member","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["link_kind"] != inviteLinkKindSingle {
		t.Fatalf("link_kind=%v", out["link_kind"])
	}
	if out["max_uses"] != float64(1) {
		t.Fatalf("max_uses=%v", out["max_uses"])
	}
}

func TestOpenInviteTwoDistinctUsersJoin(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","link_kind":"open","company_member_name":"Open","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create open invite: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	token, _ := out["invite_token"].(string)
	if token == "" {
		t.Fatal("missing token")
	}
	if out["link_kind"] != inviteLinkKindOpen {
		t.Fatalf("link_kind=%v", out["link_kind"])
	}

	joinBody := `{"token":"` + token + `"}`
	for _, uid := range []string{"openuser1", "openuser2"} {
		reqJ := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), uid)
		recJ := httptest.NewRecorder()
		gatewayUserMiddleware(mux).ServeHTTP(recJ, reqJ)
		if recJ.Code != 201 {
			t.Fatalf("join %s: %d %s", uid, recJ.Code, recJ.Body.String())
		}
	}

	reqV := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/validate-invite/?token="+token, nil)
	recV := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(recV, reqV)
	if recV.Code != 200 {
		t.Fatalf("validate after two joins: %d %s", recV.Code, recV.Body.String())
	}

	reqDup := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), "openuser1")
	recDup := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(recDup, reqDup)
	if recDup.Code != 400 {
		t.Fatalf("same user second join expected 400, got %d %s", recDup.Code, recDup.Body.String())
	}
}

func TestOpenInviteMaxUsesExhausted(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","link_kind":"open","max_uses":2,"workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	token, _ := out["invite_token"].(string)

	joinBody := `{"token":"` + token + `"}`
	for _, uid := range []string{"cap1", "cap2"} {
		reqJ := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), uid)
		recJ := httptest.NewRecorder()
		gatewayUserMiddleware(mux).ServeHTTP(recJ, reqJ)
		if recJ.Code != 201 {
			t.Fatalf("join %s: %d %s", uid, recJ.Code, recJ.Body.String())
		}
	}
	req3 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), "cap3")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 400 {
		t.Fatalf("third join expected 400, got %d %s", rec3.Code, rec3.Body.String())
	}

	reqV := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/validate-invite/?token="+token, nil)
	recV := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(recV, reqV)
	if recV.Code != 400 {
		t.Fatalf("validate exhausted expected 400, got %d %s", recV.Code, recV.Body.String())
	}
}

func TestOpenInviteEmailMethodRejected(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"email","email":"a@example.com","link_kind":"open","workspace_id":"ws1","company_member_name":"X"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestPendingOpenInviteIncludesUseCount(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","link_kind":"open","max_uses":5,"workspace_id":"ws1","company_member_name":"Batch"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}

	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("pending: %d %s", rec2.Code, rec2.Body.String())
	}
	var list []map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Fatal("expected pending row")
	}
	found := false
	for _, item := range list {
		if item["link_kind"] == inviteLinkKindOpen {
			found = true
			if item["max_uses"] != float64(5) {
				t.Fatalf("max_uses=%v", item["max_uses"])
			}
			if item["use_count"] != float64(0) {
				t.Fatalf("use_count=%v", item["use_count"])
			}
		}
	}
	if !found {
		t.Fatalf("no open invite in pending: %v", list)
	}
}

func TestPendingOpenInviteUseCountAfterJoin(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","link_kind":"open","max_uses":5,"workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	token, _ := out["invite_token"].(string)
	joinBody := `{"token":"` + token + `"}`
	reqJ := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), "openuser1")
	recJ := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(recJ, reqJ)
	if recJ.Code != 201 {
		t.Fatalf("join: %d %s", recJ.Code, recJ.Body.String())
	}

	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("pending: %d %s", rec2.Code, rec2.Body.String())
	}
	var list []map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &list)
	found := false
	for _, item := range list {
		if item["link_kind"] == inviteLinkKindOpen {
			found = true
			if item["use_count"] != float64(1) {
				t.Fatalf("use_count after join=%v", item["use_count"])
			}
		}
	}
	if !found {
		t.Fatalf("no open invite in pending after join: %v", list)
	}
}
