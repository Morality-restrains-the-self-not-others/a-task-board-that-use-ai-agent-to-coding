package main

import (
	"errors"
	"testing"
)

func TestApplyReferralRequiresServiceAccountBound(t *testing.T) {
	setupTestReferralDB(t)
	orig := userHasServiceAccountBound
	userHasServiceAccountBound = func(string) bool { return false }
	t.Cleanup(func() { userHasServiceAccountBound = orig })

	_, err := applyReferralWithConsent("user-no-mp", testValidPersonalIntro)
	if !errors.Is(err, errReferralServiceAccountNotFollowed) {
		t.Fatalf("want service_account_not_followed, got %v", err)
	}
	if referralErrorCode(err) != "service_account_not_followed" {
		t.Fatalf("error code %s", referralErrorCode(err))
	}
}

func TestApplyReferralWhenServiceAccountBound(t *testing.T) {
	setupTestReferralDB(t)
	orig := userHasServiceAccountBound
	userHasServiceAccountBound = func(string) bool { return true }
	t.Cleanup(func() { userHasServiceAccountBound = orig })

	result, err := applyReferralWithConsent("user-has-mp", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.Status != "pending" && result.Status != "approved" {
		t.Fatalf("status %s", result.Status)
	}
}

func TestReferralStatusIncludesServiceAccountBound(t *testing.T) {
	setupTestReferralDB(t)
	orig := userHasServiceAccountBound
	userHasServiceAccountBound = func(uid string) bool { return uid == "user-bound" }
	t.Cleanup(func() { userHasServiceAccountBound = orig })

	st, err := getReferralCodeStatus("user-bound")
	if err != nil {
		t.Fatal(err)
	}
	if !st.ServiceAccountBound {
		t.Fatal("expected bound")
	}
	st2, err := getReferralCodeStatus("user-unbound")
	if err != nil {
		t.Fatal(err)
	}
	if st2.ServiceAccountBound {
		t.Fatal("expected unbound")
	}
}

func TestUserHasServiceAccountBoundLiveQueriesIdentity(t *testing.T) {
	setupTestReferralDB(t)
	authDB = db
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS wechat_identity (
		  id VARCHAR(64) NOT NULL PRIMARY KEY,
		  user_id VARCHAR(64) NOT NULL,
		  app_key VARCHAR(32) NOT NULL,
		  app_id VARCHAR(64) NOT NULL DEFAULT '',
		  openid VARCHAR(128) NOT NULL,
		  unionid VARCHAR(128) NOT NULL DEFAULT '',
		  nickname VARCHAR(255) NOT NULL DEFAULT '',
		  avatar_url VARCHAR(512) NOT NULL DEFAULT '',
		  created_at VARCHAR(64) NOT NULL,
		  updated_at VARCHAR(64) NOT NULL,
		  UNIQUE KEY uk_app_openid (app_key, openid)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		t.Fatalf("create wechat_identity: %v", err)
	}
	if userHasServiceAccountBoundLive("u-live") {
		t.Fatal("empty table should be unbound")
	}
	_, err = db.Exec(`INSERT INTO wechat_identity
		(id, user_id, app_key, app_id, openid, unionid, nickname, avatar_url, created_at, updated_at)
		VALUES ('1', 'u-live', 'mp', 'wxpay', 'oMp1', 'un1', '', '', NOW(), NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	if !userHasServiceAccountBoundLive("u-live") {
		t.Fatal("mp alias should bind")
	}
}
