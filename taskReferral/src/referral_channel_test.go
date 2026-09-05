package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateReferralChannelDistinctCodes(t *testing.T) {
	setupTestReferralDB(t)
	const uid = "channel-owner-1"
	def, err := ensureUserShareCode(uid)
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	ch, err := createReferralChannel(uid, "微信")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ch.Code == "" || ch.Code == def || ch.Name != "微信" || ch.IsDefault {
		t.Fatalf("channel=%+v default=%s", ch, def)
	}
	again, err := createReferralChannel(uid, "微信")
	if err != nil {
		t.Fatalf("idempotent: %v", err)
	}
	if again.Code != ch.Code {
		t.Fatalf("same name must reuse code first=%s second=%s", ch.Code, again.Code)
	}
	owner, err := lookupShareCodeOwner(ch.Code)
	if err != nil || owner != uid {
		t.Fatalf("owner=%q err=%v", owner, err)
	}
}

func TestDisableReferralChannelStopsLookup(t *testing.T) {
	setupTestReferralDB(t)
	const uid = "channel-owner-2"
	if _, err := ensureUserShareCode(uid); err != nil {
		t.Fatalf("default: %v", err)
	}
	ch, err := createReferralChannel(uid, "抖音")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := disableReferralChannel(uid, ch.Code); err != nil {
		t.Fatalf("disable: %v", err)
	}
	owner, err := lookupShareCodeOwner(ch.Code)
	if err != nil || owner != "" {
		t.Fatalf("disabled code must not resolve, owner=%q err=%v", owner, err)
	}
	def, _ := ensureUserShareCode(uid)
	if err := disableReferralChannel(uid, def); err == nil || err.Error() != "cannot_disable_default" {
		t.Fatalf("default disable err=%v", err)
	}
}

func TestLookupShareCodeOwnerRejectsDisabledAndDerived(t *testing.T) {
	setupTestReferralDB(t)
	got, err := lookupShareCodeOwner("u873093522473906176")
	if err != nil || got != "" {
		t.Fatalf("must not parse u{{userId}}, got %q err=%v", got, err)
	}
}

func TestHandleCreateReferralChannelHTTP(t *testing.T) {
	setupTestReferralDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := withReferralUser(httptest.NewRequest(http.MethodPost, "/api/referral/channels/", strings.NewReader(`{"name":"小红书"}`)), "http-ch-user")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var row referralChannelRow
	if err := json.Unmarshal(rec.Body.Bytes(), &row); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if row.Name != "小红书" || row.Code == "" || row.IsDefault {
		t.Fatalf("row=%+v", row)
	}
}

func TestDeleteReferralChannelStopsLookup(t *testing.T) {
	setupTestReferralDB(t)
	const uid = "channel-del-owner-1"
	if _, err := ensureUserShareCode(uid); err != nil {
		t.Fatalf("default: %v", err)
	}
	ch, err := createReferralChannel(uid, "小红书")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := deleteReferralChannel(uid, ch.Code); err != nil {
		t.Fatalf("delete: %v", err)
	}
	owner, err := lookupShareCodeOwner(ch.Code)
	if err != nil || owner != "" {
		t.Fatalf("deleted code must not resolve, owner=%q err=%v", owner, err)
	}
	list, err := listReferralChannels(uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, row := range list {
		if row.Code == ch.Code {
			t.Fatalf("deleted channel still listed: %+v", row)
		}
	}
}

func TestDeleteReferralChannelDefaultForbidden(t *testing.T) {
	setupTestReferralDB(t)
	const uid = "channel-del-owner-2"
	def, err := ensureUserShareCode(uid)
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	if err := deleteReferralChannel(uid, def); err == nil || err.Error() != "cannot_delete_default" {
		t.Fatalf("default delete err=%v", err)
	}
}

func TestDeleteReferralChannelNotOwned(t *testing.T) {
	setupTestReferralDB(t)
	const owner = "channel-del-owner-3"
	const other = "channel-del-owner-4"
	if _, err := ensureUserShareCode(owner); err != nil {
		t.Fatalf("owner default: %v", err)
	}
	ch, err := createReferralChannel(owner, "抖音")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := deleteReferralChannel(other, ch.Code); err == nil || err.Error() != "not_found" {
		t.Fatalf("foreign delete err=%v", err)
	}
	// Channel still owned and resolvable.
	got, err := lookupShareCodeOwner(ch.Code)
	if err != nil || got != owner {
		t.Fatalf("owner=%q err=%v", got, err)
	}
}

func TestDeleteReferralChannelDisabled(t *testing.T) {
	setupTestReferralDB(t)
	const uid = "channel-del-owner-5"
	if _, err := ensureUserShareCode(uid); err != nil {
		t.Fatalf("default: %v", err)
	}
	ch, err := createReferralChannel(uid, "知乎")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := disableReferralChannel(uid, ch.Code); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if err := deleteReferralChannel(uid, ch.Code); err != nil {
		t.Fatalf("delete disabled: %v", err)
	}
	if got, _ := lookupShareCodeOwner(ch.Code); got != "" {
		t.Fatalf("deleted disabled channel still resolves, owner=%q", got)
	}
}

func TestHandleDeleteReferralChannelHTTP(t *testing.T) {
	setupTestReferralDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)
	const uid = "http-del-owner"
	if _, err := ensureUserShareCode(uid); err != nil {
		t.Fatalf("default: %v", err)
	}
	ch, err := createReferralChannel(uid, "B站")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	delReq := withReferralUser(httptest.NewRequest(http.MethodDelete, "/api/referral/channels/code/"+ch.Code+"/", nil), uid)
	delRec := httptest.NewRecorder()
	mux.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", delRec.Code, delRec.Body.String())
	}

	// Second delete must 404 (already gone).
	again := withReferralUser(httptest.NewRequest(http.MethodDelete, "/api/referral/channels/code/"+ch.Code+"/", nil), uid)
	againRec := httptest.NewRecorder()
	mux.ServeHTTP(againRec, again)
	if againRec.Code != http.StatusNotFound {
		t.Fatalf("re-delete status=%d body=%s", againRec.Code, againRec.Body.String())
	}

	// Non-DELETE method must 405 (PUT has no conflicting prefix route).
	bad := withReferralUser(httptest.NewRequest(http.MethodPut, "/api/referral/channels/code/"+ch.Code+"/", nil), uid)
	badRec := httptest.NewRecorder()
	mux.ServeHTTP(badRec, bad)
	if badRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("get status=%d body=%s", badRec.Code, badRec.Body.String())
	}
}
