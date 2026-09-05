package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func insertReferralEdgeForLookup(t *testing.T, referred, referrer, channel string) {
	t.Helper()
	_, err := billDB.Exec(`
		INSERT INTO billing_referral_edge (
			referred_user_id, referrer_user_id, referrer_tenant_id, bound_at, channel_code, created_at, updated_at
		) VALUES (?, ?, 1, 't', ?, 't', 't')`, referred, referrer, channel)
	if err != nil {
		t.Fatalf("insert edge %s→%s: %v", referred, referrer, err)
	}
}

// OPT-20260821-031 regression: batch referrer lookup must return the referrer
// edge for each requested referred user, and skip unknown user ids.
func TestInternalReferrersLookupBatch(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	t.Cleanup(cleanup)

	insertReferralEdgeForLookup(t, "newbie-1", "ref-1", "DR2AKvP9J9")
	insertReferralEdgeForLookup(t, "newbie-2", "ref-2", "CH-2")

	body := `{"user_ids":["newbie-1","newbie-2","no-such-user"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/referral/referrers/lookup/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalReferrersLookup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Referrers map[string]referrerEdge `json:"referrers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Referrers == nil {
		t.Fatal("referrers is nil")
	}
	if e := resp.Referrers["newbie-1"]; e.ReferrerUserID != "ref-1" || e.ChannelCode != "DR2AKvP9J9" {
		t.Fatalf("newbie-1 edge=%+v", e)
	}
	if e := resp.Referrers["newbie-2"]; e.ReferrerUserID != "ref-2" || e.ChannelCode != "CH-2" {
		t.Fatalf("newbie-2 edge=%+v", e)
	}
	if _, ok := resp.Referrers["no-such-user"]; ok {
		t.Fatalf("unknown user must be absent, got %+v", resp.Referrers["no-such-user"])
	}
}

func TestInternalReferrersLookupEmptyUserIDs(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/referral/referrers/lookup/", strings.NewReader(`{"user_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalReferrersLookup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if referrers, _ := resp["referrers"].(map[string]interface{}); len(referrers) != 0 {
		t.Fatalf("expected empty referrers, got %v", referrers)
	}
}
