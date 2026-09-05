package main

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNormalizePersonalIntro(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{name: "empty", raw: "   ", wantErr: errReferralIntroRequired},
		{name: "too short", raw: "我是用户请批准资格", wantErr: errReferralIntroTooShort},
		{name: "too long", raw: strings.Repeat("字", personalIntroMaxRunes+1), wantErr: errReferralIntroTooLong},
		{name: "ok", raw: testValidPersonalIntro, wantErr: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizePersonalIntro(tc.raw)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err=%v want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if utf8.RuneCountInString(got) < personalIntroMinRunes {
				t.Fatalf("normalized too short: %q", got)
			}
		})
	}
}

func TestApplyReferralCodeRequiresPersonalIntro(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	_, err := applyReferralWithConsent("user-intro-empty", "")
	if !errors.Is(err, errReferralIntroRequired) {
		t.Fatalf("empty intro err=%v", err)
	}

	_, err = applyReferralWithConsent("user-intro-short", "太短了不够二十个字")
	if !errors.Is(err, errReferralIntroTooShort) {
		t.Fatalf("short intro err=%v", err)
	}

	result, err := applyReferralWithConsent("user-intro-ok", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("valid intro apply: %v", err)
	}
	if result.Status != "pending" {
		t.Fatalf("status=%s", result.Status)
	}

	var stored string
	if err := db.QueryRow(
		`SELECT personal_intro FROM referral_code WHERE user_id = ?`,
		"user-intro-ok",
	).Scan(&stored); err != nil {
		t.Fatalf("query intro: %v", err)
	}
	if stored != testValidPersonalIntro {
		t.Fatalf("stored=%q", stored)
	}

	status, err := getReferralCodeStatus("user-intro-ok")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.PersonalIntro != testValidPersonalIntro {
		t.Fatalf("status intro=%q", status.PersonalIntro)
	}

	items, _, err := listReferralApplications("pending", 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, it := range items {
		if it.UserID == "user-intro-ok" {
			found = true
			if it.PersonalIntro != testValidPersonalIntro {
				t.Fatalf("list intro=%q", it.PersonalIntro)
			}
		}
	}
	if !found {
		t.Fatal("pending list missing user-intro-ok")
	}
}
