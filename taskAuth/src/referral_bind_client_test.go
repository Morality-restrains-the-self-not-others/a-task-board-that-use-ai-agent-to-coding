package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBindReferralAfterRegister_PostsAccessCode(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/referral/bind-from-code/" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

	bindReferralAfterRegister(t.Context(), "newbie-1", "DR2AKvP9J9")
	if got["referred_user_id"] != "newbie-1" || got["access_code"] != "DR2AKvP9J9" {
		t.Fatalf("body=%v", got)
	}
}

func TestBindReferralAfterRegister_SkipsEmpty(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	cfg.ReferralServiceURL = srv.URL
	bindReferralAfterRegister(t.Context(), "u1", "")
	bindReferralAfterRegister(t.Context(), "", "CODE")
	time.Sleep(20 * time.Millisecond)
	if called {
		t.Fatal("empty access code must not call referral service")
	}
}

func TestBindReferralAfterRegister_DoesNotFailOnDownService(t *testing.T) {
	cfg.ReferralServiceURL = "http://127.0.0.1:1"
	bindReferralAfterRegister(t.Context(), "u1", "CODE12")
}

func TestExtractAccessCodeFromRegisterBody(t *testing.T) {
	if got := extractAccessCodeFromRegisterBody(map[string]interface{}{"access_code": " AbC "}); got != "AbC" {
		t.Fatalf("got %q", got)
	}
	if got := extractAccessCodeFromRegisterBody(map[string]interface{}{"accessCode": "xyz"}); got != "xyz" {
		t.Fatalf("camelCase got %q", got)
	}
	if got := extractAccessCodeFromRegisterBody(nil); got != "" {
		t.Fatalf("nil got %q", got)
	}
}
