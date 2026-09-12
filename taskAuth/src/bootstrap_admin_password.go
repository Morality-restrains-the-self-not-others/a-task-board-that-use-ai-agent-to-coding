package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
)

// 022 种子 SQL 写入的待填哨兵；bootstrap-admin 首次执行时替换为真随机 bcrypt。
const bootstrapAdminPasswordPending = "__BOOTSTRAP_ADMIN_PASSWORD_PENDING__"

// 历史共享「随机」哈希（曾硬编码进 022，全环境相同）。仍命中则轮换为每环境独立随机。
const legacySharedBootstrapPasswordHash = "$2a$12$reqIRh/zFv.aez0dNtwWJ.kED6CpqFsppSm8pBOk6Vd9n8hKOK2A2"

var knownWeakBootstrapPasswords = []string{"admin123", "rgNodkdq8677!ci"}

func generateRandomAdminPassword() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func needsBootstrapAdminPasswordRotation(passwordHash string) bool {
	h := strings.TrimSpace(passwordHash)
	if h == "" || h == bootstrapAdminPasswordPending {
		return true
	}
	if h == legacySharedBootstrapPasswordHash {
		return true
	}
	for _, weak := range knownWeakBootstrapPasswords {
		if checkPasswordHash(weak, h) {
			return true
		}
	}
	return false
}

// ensureBootstrapAdminRandomPassword 在初始化时生成每环境独立的随机密码哈希。
// 明文不落盘、不打印；管理员须通过「忘记密码」邮箱重置设置可用密码。
// 幂等：已是非弱、非哨兵、非历史共享哈希时跳过，避免覆盖运维已设密码。
func ensureBootstrapAdminRandomPassword() error {
	existing, err := findEmailLoginMethodByUserID(bootstrapAdminUserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf(
			"bootstrap admin %q 没有 email 登录方式 — dataMigrate 可能未执行",
			bootstrapAdminUserID,
		)
	}
	if !needsBootstrapAdminPasswordRotation(existing.PasswordHash) {
		return nil
	}
	plain, err := generateRandomAdminPassword()
	if err != nil {
		return err
	}
	hash, err := hashPassword(plain)
	plain = "" // 尽快丢弃明文引用
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}
	if _, err := db.Exec(
		`UPDATE auth_login_method SET password_hash = ?, updated_at = NOW() WHERE id = ?`,
		hash, existing.ID,
	); err != nil {
		return err
	}
	if _, err := db.Exec(
		`UPDATE auth_user SET must_change_password = 1 WHERE id = ?`,
		bootstrapAdminUserID,
	); err != nil {
		return err
	}
	log.Printf("[taskAuth] bootstrap-admin: wrote random password hash (user_id=%s; plaintext discarded; use email reset)", bootstrapAdminUserID)
	return nil
}
