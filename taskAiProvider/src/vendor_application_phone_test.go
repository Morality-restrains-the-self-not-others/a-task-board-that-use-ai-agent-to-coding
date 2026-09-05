package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVendorApplicationPhoneStatusUnauthorized(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/vendor/application/phone-status/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestVendorApplicationVerifyPhoneUnauthorized(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/ai-provider/vendor-application/verify-phone/", strings.NewReader(`{"phone":"+8613800138000","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestVendorApplicationSendSMSUnauthorized(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/application/send-sms/", strings.NewReader(`{"phone":"+8613800138000"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMaskAndParseApplicantPhone(t *testing.T) {
	if got := maskApplicantPhone("+8613900001234"); got != "139****1234" {
		t.Fatalf("mask = %q", got)
	}
	cc, nat := parseApplicantE164("+8613800138000")
	if cc != "86" || nat != "13800138000" {
		t.Fatalf("parseE164 cc=%q nat=%q", cc, nat)
	}
	cc, nat = parseApplicantE164("13800138000")
	if cc != "86" || nat != "13800138000" {
		t.Fatalf("parse local 11-digit cc=%q nat=%q", cc, nat)
	}
	cc, nat = parseApplicantE164("+86 13800138000")
	if cc != "86" || nat != "13800138000" {
		t.Fatalf("parse spaced +86 cc=%q nat=%q", cc, nat)
	}
}
