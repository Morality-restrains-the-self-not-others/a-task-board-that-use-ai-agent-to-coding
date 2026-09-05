package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260821-031 regression: the batch referrer client must POST the user ids
// to the taskReferral internal endpoint and return the referrer edge map.
func TestFetchReferrersBatchPostsUserIDs(t *testing.T) {
	var gotBody struct {
		UserIDs []string `json:"user_ids"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/referral/referrers/lookup/" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"referrers": map[string]map[string]string{
				"u1": {"referrer_user_id": "r1", "channel_code": "CODE-A"},
			},
		})
	}))
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

	refs := fetchReferrersBatch(t.Context(), []string{"u1", "u2"})
	if len(gotBody.UserIDs) != 2 || gotBody.UserIDs[0] != "u1" || gotBody.UserIDs[1] != "u2" {
		t.Fatalf("posted user_ids=%v", gotBody.UserIDs)
	}
	e, ok := refs["u1"]
	if !ok || e["referrer_user_id"] != "r1" || e["channel_code"] != "CODE-A" {
		t.Fatalf("refs=%v", refs)
	}
}

func TestFetchReferrersBatchEmptyInput(t *testing.T) {
	refs := fetchReferrersBatch(t.Context(), nil)
	if len(refs) != 0 {
		t.Fatalf("expected empty, got %v", refs)
	}
}

func TestFetchReferrersBatchDoesNotFailOnDownService(t *testing.T) {
	cfg.ReferralServiceURL = "http://127.0.0.1:1"
	refs := fetchReferrersBatch(t.Context(), []string{"u1"})
	if len(refs) != 0 {
		t.Fatalf("expected empty on down service, got %v", refs)
	}
}

func TestReferrerDisplayName(t *testing.T) {
	profiles := map[string]string{"r1": "referrer-nick"}
	logins := map[string][]map[string]interface{}{
		"r2": {{"method_type": "phone", "identifier": "13800000000", "is_verified": true}},
		"r3": {{"method_type": "email", "identifier": "r3@test.com", "is_verified": true}},
	}
	if got := referrerDisplayName("r1", profiles, logins); got != "referrer-nick" {
		t.Fatalf("profile username should win, got %q", got)
	}
	if got := referrerDisplayName("r2", profiles, logins); got != "13800000000" {
		t.Fatalf("phone fallback, got %q", got)
	}
	if got := referrerDisplayName("r3", profiles, logins); got != "r3@test.com" {
		t.Fatalf("email fallback, got %q", got)
	}
	if got := referrerDisplayName("r9", profiles, logins); got != "r9" {
		t.Fatalf("raw id fallback, got %q", got)
	}
}
