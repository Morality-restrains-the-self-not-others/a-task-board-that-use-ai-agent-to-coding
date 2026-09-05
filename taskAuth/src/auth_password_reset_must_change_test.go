package main

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestBootstrapAdminRandomPassword 护栏（OPT-20260824-001）：
// 种子管理员密码必须是随机哈希，不得再命中已知弱明文（admin123 / rgNodkdq8677!ci）。
func TestBootstrapAdminRandomPassword(t *testing.T) {
	setupAuthTestDB(t)

	lm, err := findLoginMethodByEmail(mustConfAdminEmail(t))
	if err != nil {
		t.Fatalf("find admin login method: %v", err)
	}
	if lm == nil {
		t.Fatal("bootstrap admin login method not found")
	}

	for _, weak := range []string{"admin123", "rgNodkdq8677!ci"} {
		if bcrypt.CompareHashAndPassword([]byte(lm.PasswordHash), []byte(weak)) == nil {
			t.Fatalf("bootstrap admin 密码仍是已知弱明文 %q — 种子必须为随机密码", weak)
		}
	}
}

// TestResetPasswordWithLinkClearsMustChangePassword（OPT-20260824-001）：
// 管理员通过邮箱重置链接设置新密码后，must_change_password 必须清除，
// 否则重置后登录还会被强制改密（随机密码引导流程闭环）。
func TestResetPasswordWithLinkClearsMustChangePassword(t *testing.T) {
	setupAuthTestDB(t)

	lm, err := findLoginMethodByEmail(mustConfAdminEmail(t))
	if err != nil {
		t.Fatalf("find admin login method: %v", err)
	}
	if lm == nil {
		t.Fatal("bootstrap admin login method not found")
	}

	must, err := getUserMustChangePassword(lm.ObjectID)
	if err != nil {
		t.Fatalf("get must_change_password: %v", err)
	}
	if !must {
		t.Fatal("种子管理员 must_change_password 应为 1")
	}

	// 模拟邮箱链接重置：更新密码哈希 + 清除强制改密标记
	if err := updatePasswordHashClearResetToken(lm.ID, "Admin@Reset2026!new"); err != nil {
		t.Fatalf("updatePasswordHashClearResetToken: %v", err)
	}
	if err := clearMustChangePassword(lm.ObjectID); err != nil {
		t.Fatalf("clearMustChangePassword: %v", err)
	}

	must, err = getUserMustChangePassword(lm.ObjectID)
	if err != nil {
		t.Fatalf("get must_change_password after reset: %v", err)
	}
	if must {
		t.Fatal("邮箱重置后 must_change_password 应被清除")
	}

	// 新密码可正常通过校验
	lm2, err := findLoginMethodByEmail(mustConfAdminEmail(t))
	if err != nil || lm2 == nil {
		t.Fatalf("re-find admin login method: %v", err)
	}
	if !checkPasswordHash("Admin@Reset2026!new", lm2.PasswordHash) {
		t.Fatal("重置后的新密码应通过 bcrypt 校验")
	}
	if checkPasswordHash("wrong-password", lm2.PasswordHash) {
		t.Fatal("错误密码不应通过校验")
	}
}
