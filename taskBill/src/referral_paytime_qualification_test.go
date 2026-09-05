package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func stubPaytimeQualification(t *testing.T, fn func(ctx context.Context, referrerUserID string) bool) {
	t.Helper()
	orig := lookupReferrerPaytimeQualification
	lookupReferrerPaytimeQualification = fn
	t.Cleanup(func() { lookupReferrerPaytimeQualification = orig })
}

func TestLookupReferrerPaytimeQualificationLiveActive(t *testing.T) {
	origURL, origSecret := cfg.TaskReferralBaseURL, cfg.TaskReferralInternalSecret
	origFn := lookupReferrerPaytimeQualification
	t.Cleanup(func() {
		cfg.TaskReferralBaseURL, cfg.TaskReferralInternalSecret = origURL, origSecret
		lookupReferrerPaytimeQualification = origFn
	})
	lookupReferrerPaytimeQualification = lookupReferrerPaytimeQualificationLive

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/referral/qualification/active/" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("user_id") != "ref-live-1" {
			t.Errorf("user_id=%s", r.URL.Query().Get("user_id"))
		}
		if r.Header.Get("X-TaskReferral-Internal-Secret") != "sec-paytime" {
			t.Errorf("missing internal secret")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"user_id": "ref-live-1", "active": true})
	}))
	t.Cleanup(srv.Close)
	cfg.TaskReferralBaseURL = srv.URL
	cfg.TaskReferralInternalSecret = "sec-paytime"

	if !lookupReferrerPaytimeQualificationLive(context.Background(), "ref-live-1") {
		t.Fatal("want active true")
	}
}

func TestLookupReferrerPaytimeQualificationLiveFailClosed(t *testing.T) {
	origURL := cfg.TaskReferralBaseURL
	origFn := lookupReferrerPaytimeQualification
	t.Cleanup(func() {
		cfg.TaskReferralBaseURL = origURL
		lookupReferrerPaytimeQualification = origFn
	})
	lookupReferrerPaytimeQualification = lookupReferrerPaytimeQualificationLive

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	cfg.TaskReferralBaseURL = srv.URL

	if lookupReferrerPaytimeQualificationLive(context.Background(), "ref-down") {
		t.Fatal("HTTP 503 must fail-closed")
	}
}
