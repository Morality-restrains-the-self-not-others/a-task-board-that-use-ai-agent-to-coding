package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchQualificationsBatchPostsUserIDs(t *testing.T) {
	var gotBody struct {
		UserIDs []string `json:"user_ids"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/referral/qualification/active/batch/" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"qualifications": map[string]bool{"u1": true, "u2": false},
		})
	}))
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

	got, ok := fetchQualificationsBatch(t.Context(), []string{"u1", "u2"})
	if !ok {
		t.Fatal("expected ok")
	}
	if len(gotBody.UserIDs) != 2 || gotBody.UserIDs[0] != "u1" || gotBody.UserIDs[1] != "u2" {
		t.Fatalf("posted user_ids=%v", gotBody.UserIDs)
	}
	if !got["u1"] || got["u2"] {
		t.Fatalf("got=%v", got)
	}
}

func TestFetchQualificationsBatchEmptyInput(t *testing.T) {
	got, ok := fetchQualificationsBatch(t.Context(), nil)
	if !ok || len(got) != 0 {
		t.Fatalf("ok=%v got=%v", ok, got)
	}
}

func TestFetchQualificationsBatchDoesNotFailOnDownService(t *testing.T) {
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })
	got, ok := fetchQualificationsBatch(t.Context(), []string{"u1"})
	if ok || got != nil {
		t.Fatalf("down service must be unknown, ok=%v got=%v", ok, got)
	}
}

func TestAttachProfitSharingQualificationsNullWhenUnknown(t *testing.T) {
	users := []map[string]interface{}{{"id": "u1"}}
	attachProfitSharingQualifications(users, nil, false)
	if users[0]["has_profit_sharing_qualification"] != nil {
		t.Fatalf("unknown must be nil, got %v", users[0]["has_profit_sharing_qualification"])
	}
	attachProfitSharingQualifications(users, map[string]bool{"u1": true}, true)
	if users[0]["has_profit_sharing_qualification"] != true {
		t.Fatalf("active must be true, got %v", users[0]["has_profit_sharing_qualification"])
	}
}
