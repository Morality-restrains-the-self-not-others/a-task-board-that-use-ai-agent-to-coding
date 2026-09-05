package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type memberJoinedRecord struct {
	eventType string
	data      map[string]interface{}
}

func memberJoinedOnly(records []memberJoinedRecord) []memberJoinedRecord {
	var out []memberJoinedRecord
	for _, r := range records {
		if r.eventType == "MEMBER_JOINED" {
			out = append(out, r)
		}
	}
	return out
}

func createInviteAndGetToken(t *testing.T, mux *http.ServeMux, body string) string {
	t.Helper()
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create invite: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	token, _ := out["invite_token"].(string)
	if token == "" {
		t.Fatal("missing invite_token")
	}
	return token
}

func joinInvite(t *testing.T, mux *http.ServeMux, token, uid string) {
	t.Helper()
	body := `{"token":"` + token + `"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(body)), uid)
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("join %s: %d %s", uid, rec.Code, rec.Body.String())
	}
}

// OPT-20260829-012: MEMBER_JOINED must carry invitation_id / link_kind /
// use_count / invitation_exhausted — downstream git-identity & notification
// depend on these extra fields. HTTP 201 alone doesn't prove the event contract.
func TestOpenInviteTwoDistinctUsersJoinPublishesMemberJoined(t *testing.T) {
	mux := setupTestService(t)
	var records []memberJoinedRecord
	publishEventHook = func(eventType string, data map[string]interface{}) {
		records = append(records, memberJoinedRecord{eventType: eventType, data: data})
	}
	t.Cleanup(func() { publishEventHook = nil })

	token := createInviteAndGetToken(t, mux, `{"invite_method":"link","link_kind":"open","company_member_name":"Open","workspace_id":"ws1"}`)
	joinInvite(t, mux, token, "openuser1")
	joinInvite(t, mux, token, "openuser2")

	joined := memberJoinedOnly(records)
	if len(joined) != 2 {
		t.Fatalf("expected 2 MEMBER_JOINED, got %d: %v", len(joined), records)
	}
	first := joined[0].data
	second := joined[1].data
	invID := first["invitation_id"]
	if invID == "" || invID != second["invitation_id"] {
		t.Fatalf("invitation_id differs: %v vs %v", invID, second["invitation_id"])
	}
	if first["link_kind"] != inviteLinkKindOpen || second["link_kind"] != inviteLinkKindOpen {
		t.Fatalf("link_kind=%v/%v", first["link_kind"], second["link_kind"])
	}
	if first["use_count"] != 1 || second["use_count"] != 2 {
		t.Fatalf("use_count=%v/%v", first["use_count"], second["use_count"])
	}
	if first["invitation_exhausted"] != false || second["invitation_exhausted"] != false {
		t.Fatalf("invitation_exhausted=%v/%v", first["invitation_exhausted"], second["invitation_exhausted"])
	}
}

func TestSingleInviteJoinPublishesMemberJoinedLinkKindSingle(t *testing.T) {
	mux := setupTestService(t)
	var records []memberJoinedRecord
	publishEventHook = func(eventType string, data map[string]interface{}) {
		records = append(records, memberJoinedRecord{eventType: eventType, data: data})
	}
	t.Cleanup(func() { publishEventHook = nil })

	token := createInviteAndGetToken(t, mux, `{"invite_method":"link","company_member_name":"One","workspace_id":"ws1"}`)
	joinInvite(t, mux, token, "singleuser1")

	joined := memberJoinedOnly(records)
	if len(joined) != 1 {
		t.Fatalf("expected 1 MEMBER_JOINED, got %d: %v", len(joined), records)
	}
	data := joined[0].data
	if data["link_kind"] != inviteLinkKindSingle {
		t.Fatalf("link_kind=%v", data["link_kind"])
	}
	if data["use_count"] != 1 {
		t.Fatalf("use_count=%v", data["use_count"])
	}
	if data["invitation_exhausted"] != true {
		t.Fatalf("invitation_exhausted=%v", data["invitation_exhausted"])
	}
}
