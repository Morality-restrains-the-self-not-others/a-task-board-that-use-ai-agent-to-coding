package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLookupShareCodeOwner_UnknownAndFound(t *testing.T) {
	setupTestReferralDB(t)
	code, err := ensureUserShareCode("referrer-1")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got, err := lookupShareCodeOwner(code)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got != "referrer-1" {
		t.Fatalf("owner=%q want referrer-1", got)
	}
	missing, err := lookupShareCodeOwner("no-such-code")
	if err != nil {
		t.Fatalf("lookup missing: %v", err)
	}
	if missing != "" {
		t.Fatalf("expected empty owner for unknown code, got %q", missing)
	}
}

func TestLookupShareCodeOwner_RejectsUserIDDerived(t *testing.T) {
	setupTestReferralDB(t)
	const uid = "873093522473906176"
	got, err := lookupShareCodeOwner("u" + uid)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got != "" {
		t.Fatalf("must not resolve u{{userId}} codes, got %q", got)
	}
}

func TestBindReferralFromAccessCode_Table(t *testing.T) {
	var synced []string
	deps := referralBindDeps{
		lookupOwner: func(code string) (string, error) {
			if code == "GOODCODE12" {
				return "referrer-9", nil
			}
			return "", nil
		},
		lookupTenant: func(referrerID string) (int64, error) {
			if referrerID == "referrer-9" {
				return 850256677331562501, nil
			}
			return 0, nil
		},
		syncEdge: func(referrer, referred string, tenantID int64, boundAt string, commissionEligible bool, channelCode string) error {
			synced = append(synced, referrer+"→"+referred)
			if tenantID <= 0 {
				t.Fatalf("tenant must be > 0")
			}
			if boundAt == "" {
				t.Fatalf("boundAt required")
			}
			if channelCode != "GOODCODE12" {
				t.Fatalf("channelCode=%q", channelCode)
			}
			if commissionEligible {
				t.Fatalf("default deps have no qualification")
			}
			return nil
		},
	}

	t.Run("ineligible_still_binds", func(t *testing.T) {
		synced = nil
		st, reason, err := deps.Bind("new-user", "GOODCODE12")
		if err != nil || st != "ok" {
			t.Fatalf("got status=%s reason=%s err=%v", st, reason, err)
		}
		if len(synced) != 1 || synced[0] != "referrer-9→new-user" {
			t.Fatalf("must still sync edge without qualification, synced=%v", synced)
		}
	})
	t.Run("eligible_when_qualified", func(t *testing.T) {
		var gotEligible bool
		d := deps
		d.hasQualification = func(string) bool { return true }
		d.syncEdge = func(referrer, referred string, tenantID int64, boundAt string, commissionEligible bool, channelCode string) error {
			gotEligible = commissionEligible
			return nil
		}
		st, _, err := d.Bind("new-user", "GOODCODE12")
		if err != nil || st != "ok" || !gotEligible {
			t.Fatalf("status=%s eligible=%v err=%v", st, gotEligible, err)
		}
	})
	t.Run("unknown_code", func(t *testing.T) {
		st, reason, err := deps.Bind("new-user", "NOPE")
		if err != nil || st != "skipped" || reason != "unknown_code" {
			t.Fatalf("got status=%s reason=%s err=%v", st, reason, err)
		}
	})
	t.Run("empty_code", func(t *testing.T) {
		st, reason, err := deps.Bind("new-user", "  ")
		if err != nil || st != "skipped" || reason != "empty_code" {
			t.Fatalf("got status=%s reason=%s err=%v", st, reason, err)
		}
	})
	t.Run("self_referral", func(t *testing.T) {
		st, reason, err := deps.Bind("referrer-9", "GOODCODE12")
		if err != nil || st != "skipped" || reason != "self_referral" {
			t.Fatalf("got status=%s reason=%s err=%v", st, reason, err)
		}
	})
	t.Run("no_tenant", func(t *testing.T) {
		d := deps
		d.lookupTenant = func(string) (int64, error) { return 0, nil }
		st, reason, err := d.Bind("new-user", "GOODCODE12")
		if err != nil || st != "skipped" || reason != "no_tenant" {
			t.Fatalf("got status=%s reason=%s err=%v", st, reason, err)
		}
	})
}

func TestHandleInternalBindFromCode_HTTP(t *testing.T) {
	setupTestReferralDB(t)
	code, err := ensureUserShareCode("http-referrer")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}

	var gotBody map[string]interface{}
	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/by-creator") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]string{{
			"id": "850256677331562501", "name": "我的公司", "creator_id": "http-referrer",
		}})
	}))
	defer tenantSrv.Close()
	billSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/taskbill/referral/sync-edge/" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer billSrv.Close()

	prevTenant, prevBill := cfg.TenantServiceURL, cfg.BillServiceURL
	cfg.TenantServiceURL = tenantSrv.URL
	cfg.BillServiceURL = billSrv.URL
	t.Cleanup(func() {
		cfg.TenantServiceURL = prevTenant
		cfg.BillServiceURL = prevBill
	})

	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/referral/bind-from-code/", strings.NewReader(
		`{"referred_user_id":"newbie","access_code":"`+code+`"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if out["status"] != "ok" {
		t.Fatalf("body=%v", out)
	}
	if gotBody["referrer_user_id"] != "http-referrer" || gotBody["referred_user_id"] != "newbie" {
		t.Fatalf("sync-edge body=%v", gotBody)
	}
	if gotBody["channel_code"] != code {
		t.Fatalf("channel_code=%v want %s", gotBody["channel_code"], code)
	}
	if gotBody["commission_eligible"] != false {
		t.Fatalf("unqualified referrer must snapshot eligible=false, body=%v", gotBody)
	}
}

func TestSyncReferralEdgeHTTP_SendsInternalSecret(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/taskbill/referral/sync-edge/" {
			http.NotFound(w, r)
			return
		}
		gotHeader = r.Header.Get("X-TaskBill-Internal-Secret")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()
	prevURL, prevSec := cfg.BillServiceURL, cfg.BillInternalSecret
	cfg.BillServiceURL = srv.URL
	cfg.BillInternalSecret = "bill-secret-for-test"
	t.Cleanup(func() {
		cfg.BillServiceURL = prevURL
		cfg.BillInternalSecret = prevSec
	})
	if err := syncReferralEdgeHTTP("r1", "d1", 1, "2026-01-01 00:00:00.000000", false, "CODEA"); err != nil {
		t.Fatalf("sync-edge: %v", err)
	}
	if gotHeader != "bill-secret-for-test" {
		t.Fatalf("X-TaskBill-Internal-Secret=%q, want bill-secret-for-test", gotHeader)
	}
}
