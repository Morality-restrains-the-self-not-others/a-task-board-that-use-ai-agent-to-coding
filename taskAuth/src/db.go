package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// errTokenIPMismatch reports that the presented token was bound to a different
// client IP than the current caller. It is an authentication failure (the
// caller should re-login), NOT a server/DB fault — callers must not surface it
// as HTTP 500 "db error".
var errTokenIPMismatch = errors.New("token ip mismatch")

func openDB(dsn string) error {
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	return db.Ping()
}

func loadUserContentTypeID() error {
	var id int64
	err := db.QueryRow(
		`SELECT id FROM auth_django_content_type WHERE app_label = 'accounts' AND model = 'user'`,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("content_type for accounts.user: %w", err)
	}
	cfg.UserContentTypeID = id
	return nil
}

type LoginMethodRow struct {
	ID           int64
	ObjectID     string
	MethodType   string
	Identifier   string
	PasswordHash string
	IsVerified   bool
}

func realignLoginMethodObjectID(loginMethodID int64, objectID string) error {
	_, err := db.Exec(`
		UPDATE auth_login_method
		SET object_id = ?, updated_at = ?
		WHERE id = ?`,
		objectID, time.Now().UTC().Format("2006-01-02 15:04:05.000000"), loginMethodID)
	return err
}

func userIsActive(userID string) (bool, error) {
	var active, archived bool
	err := db.QueryRow(`SELECT is_active, COALESCE(is_archived, 0) FROM auth_user WHERE id = ?`, userID).Scan(&active, &archived)
	if err == sql.ErrNoRows {
		return false, nil
	}
	// Archived accounts are treated as inactive — blocked from login/auth.
	return active && !archived, err
}

// userHasLiveToken reports whether the user holds at least one row in
// auth_customtoken. Login always creates one (getOrCreateToken) and a DB
// re-initialisation wipes the table — so the row's existence proves the user
// logged in after the last reset. Used to gate the userId-cookie SSO bridge
// fallback: a bare userId cookie alone must never authenticate (deterministic
// user IDs such as 'bootstrap-admin' are re-seeded identically on re-init,
// which previously kept stale browser sessions "logged in" after a DB reset).
func userHasLiveToken(userID string) (bool, error) {
	var cnt int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM auth_customtoken WHERE content_type_id = ? AND object_id = ?`,
		cfg.UserContentTypeID, userID,
	).Scan(&cnt)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// getUserMustChangePassword returns true if the user is required to change
// their password before proceeding. This flag is set by dataMigrate for
// bootstrap accounts and cleared after the first successful password change.
func getUserMustChangePassword(userID string) (bool, error) {
	var mustChange bool
	err := db.QueryRow(`SELECT COALESCE(must_change_password, 0) FROM auth_user WHERE id = ?`, userID).Scan(&mustChange)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return mustChange, err
}

// clearMustChangePassword 清除 must_change_password 标记。
// 管理员走邮箱/验证码重置密码成功后调用 — 用户已自行设置新密码，
// 无需再强制修改（OPT-20260824-001 随机密码 + 邮箱重置流程）。
func clearMustChangePassword(userID string) error {
	_, err := db.Exec(`UPDATE auth_user SET must_change_password = 0 WHERE id = ?`, userID)
	return err
}

func getOrCreateToken(userID string, contentTypeID int64, clientIP string) (string, error) {
	ip := strings.TrimSpace(clientIP)
	var key string
	var existingIP string
	err := db.QueryRow(`
		SELECT `+"`"+`key`+"`"+`, COALESCE(client_ip, '') FROM auth_customtoken
		WHERE content_type_id = ? AND object_id = ?`, contentTypeID, userID).Scan(&key, &existingIP)
	if err == nil {
		// Update last_ip on each getOrCreateToken (login/activate-session)
		if ip != "" {
			_, _ = db.Exec(`UPDATE auth_customtoken SET last_ip = ? WHERE `+"`"+`key`+"`"+` = ?`, ip, key)
		}
		return key, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	key, err = generateTokenKey()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err = db.Exec(`
		INSERT INTO auth_customtoken (`+"`"+`key`+"`"+`, content_type_id, object_id, created, client_ip, last_ip)
		VALUES (?, ?, ?, ?, ?, ?)`, key, contentTypeID, userID, now, ip, ip)
	return key, err
}

// resolveTokenUserIDWithIP resolves a token key to user ID with optional IP binding.
// When token has a stored client_ip, the current clientIP must match.
// Empty stored IP (grandfathered tokens) skips the check.
func resolveTokenUserIDWithIP(tokenKey, clientIP string) (string, error) {
	if tokenKey == "" {
		return "", sql.ErrNoRows
	}
	if isImpersonationToken(tokenKey) {
		sess, err := lookupOpenImpersonationByToken(tokenKey)
		if err != nil {
			return "", err
		}
		return sess.TargetUserID, nil
	}
	ip := strings.TrimSpace(clientIP)
	var objectID string
	var storedClientIP string
	err := db.QueryRow(`
		SELECT object_id, COALESCE(client_ip, '') FROM auth_customtoken
		WHERE `+"`"+`key`+"`"+` = ? AND content_type_id = ?`, tokenKey, cfg.UserContentTypeID,
	).Scan(&objectID, &storedClientIP)
	if err != nil {
		return "", err
	}
	// IP binding: if token was created with an IP, verify match
	if storedClientIP != "" && ip != "" && storedClientIP != ip {
		// Also check last_ip in case IP changed legitimately (e.g. mobile network switch)
		var lastIP string
		if err2 := db.QueryRow(`SELECT COALESCE(last_ip, '') FROM auth_customtoken WHERE `+"`"+`key`+"`"+` = ?`, tokenKey).Scan(&lastIP); err2 == nil && lastIP == ip {
			// last_ip matches — accept and promote to client_ip
			_, _ = db.Exec(`UPDATE auth_customtoken SET client_ip = ? WHERE `+"`"+`key`+"`"+` = ?`, ip, tokenKey)
			return objectID, nil
		}
		// Same-host dev topology tolerance: the local :4000 nginx proxy resolves
		// the caller as loopback (127.0.0.1) while the same request straight to the
		// docker-published gateway resolves as the bridge gateway (172.x.0.1) —
		// two views of the same physical host (OPT-20260904-006). Treat a loopback
		// on either side as same-host and promote, so dev/E2E login flows are not
		// rejected as if the token were stolen. Genuine cross-network (public IP)
		// mismatches below are still refused.
		if isLoopbackAddr(storedClientIP) || isLoopbackAddr(ip) {
			_, _ = db.Exec(`UPDATE auth_customtoken SET client_ip = ? WHERE `+"`"+`key`+"`"+` = ?`, ip, tokenKey)
			return objectID, nil
		}
		return "", errTokenIPMismatch
	}
	return objectID, nil
}

// isLoopbackAddr reports whether s is a loopback address (127.0.0.0/8, ::1).
func isLoopbackAddr(s string) bool {
	ip := net.ParseIP(strings.TrimSpace(s))
	return ip != nil && ip.IsLoopback()
}

// resolveTokenUserID resolves a token key without IP verification (internal use only).
func resolveTokenUserID(tokenKey string) (string, error) {
	return resolveTokenUserIDWithIP(tokenKey, "")
}

func deleteTokenByKey(key string) error {
	_, err := db.Exec(`DELETE FROM auth_customtoken WHERE `+"`"+`key`+"`"+` = ?`, key)
	return err
}

func minimalUserJSON(userID string) (map[string]interface{}, error) {
	var isActive, isArchived, isTenant, isTester bool
	err := db.QueryRow(
		`SELECT is_active, COALESCE(is_archived, 0), COALESCE(is_tenant, 0), COALESCE(is_tester, 0) FROM auth_user WHERE id = ?`, userID,
	).Scan(&isActive, &isArchived, &isTenant, &isTester)
	if err != nil {
		return nil, err
	}
	isSuper, err := isSuperAdminUser(userID)
	if err != nil {
		return nil, err
	}
	email := ""
	row := db.QueryRow(`
		SELECT identifier FROM auth_login_method
		WHERE object_id = ? AND method_type = 'email' LIMIT 1`, userID)
	var identifier string
	if row.Scan(&identifier) == nil && identifier != "" {
		email = identifier
	}
	return map[string]interface{}{
		"id":            userID,
		"is_active":     isActive,
		"is_archived":   isArchived,
		"is_tenant":     isTenant,
		"is_tester":     isTester,
		"is_superuser":  isSuper,
		"email":         email,
		"username":      email,
		"companies":     []interface{}{},
		"login_methods": []interface{}{},
	}, nil
}

func createUserWithEmailLogin(email, password string) (userID string, activationToken string, err error) {
	userIDInt := generateSnowflakeID()
	userID = fmt.Sprintf("%d", userIDInt)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return "", "", fmt.Errorf("bcrypt hash: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO auth_user (id, password, last_login, is_superuser, is_staff, is_active, is_tenant, date_joined)
		VALUES (?, '', NULL, 0, 0, 1, 0, ?)`, userID, now)
	if err != nil {
		return "", "", err
	}

	lmID := generateSnowflakeID()
	activationToken, err = generateActivationToken()
	if err != nil {
		return "", "", err
	}
	expires := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02 15:04:05.000000")
	_, err = tx.Exec(`
		INSERT INTO auth_login_method (
			id, content_type_id, object_id, method_type, identifier,
			password_hash, is_verified, activation_token, activation_token_expires_at,
			created_at, updated_at
		) VALUES (?, ?, ?, 'email', ?, ?, 0, ?, ?, ?, ?)`,
		lmID, cfg.UserContentTypeID, userID, email, hashedPassword,
		activationToken, expires, now, now)
	if err != nil {
		return "", "", err
	}
	if err := tx.Commit(); err != nil {
		return "", "", err
	}
	return userID, activationToken, nil
}

// deleteUser removes a user and their login methods. Used to roll back registration
// when djangoPostRegister fails, ensuring no orphan users exist without company/workspace.
func deleteUser(userID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM auth_login_method WHERE object_id = ?`, userID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM auth_user WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// parseDateTime attempts to parse a datetime string stored by taskAuth or rewritten
// by the SQLite driver (ISO 8601 / RFC 3339) into a time.Time.
func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range formats {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("parseDateTime: unable to parse %q", s)
}

func findLoginMethodByActivationToken(token string) (*LoginMethodRow, error) {
	row := db.QueryRow(`
		SELECT id, object_id, method_type, identifier, COALESCE(password_hash,''), is_verified
		FROM auth_login_method
		WHERE activation_token = ?`, token)
	var lm LoginMethodRow
	err := row.Scan(&lm.ID, &lm.ObjectID, &lm.MethodType, &lm.Identifier, &lm.PasswordHash, &lm.IsVerified)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lm, nil
}

func activateLoginMethod(id int64) error {
	_, err := db.Exec(`
		UPDATE auth_login_method
		SET is_verified = 1, activation_token = NULL, activation_token_expires_at = NULL,
		    updated_at = ?
		WHERE id = ?`, time.Now().UTC().Format("2006-01-02 15:04:05.000000"), id)
	return err
}

func isActivationTokenValid(token string) (bool, *LoginMethodRow) {
	lm, err := findLoginMethodByActivationToken(token)
	if err != nil || lm == nil {
		return false, nil
	}
	var expires sql.NullString
	err = db.QueryRow(`
		SELECT activation_token_expires_at FROM auth_login_method WHERE id = ?`, lm.ID).Scan(&expires)
	if err != nil || !expires.Valid {
		return false, lm
	}
	t, parseErr := parseDateTime(expires.String)
	if parseErr != nil {
		return false, lm
	}
	return time.Now().UTC().Before(t), lm
}

func setPasswordResetToken(loginMethodID int64, token, expiresAt string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		UPDATE auth_login_method
		SET password_reset_token = ?, password_reset_token_expires_at = ?, updated_at = ?
		WHERE id = ?`, token, expiresAt, now, loginMethodID)
	return err
}

func findLoginMethodByPasswordResetToken(token string) (*LoginMethodRow, error) {
	row := db.QueryRow(`
		SELECT id, object_id, method_type, identifier, COALESCE(password_hash,''), is_verified
		FROM auth_login_method
		WHERE password_reset_token = ?`, token)
	var lm LoginMethodRow
	err := row.Scan(&lm.ID, &lm.ObjectID, &lm.MethodType, &lm.Identifier, &lm.PasswordHash, &lm.IsVerified)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lm, nil
}

func isPasswordResetTokenValid(token string) (bool, *LoginMethodRow) {
	lm, err := findLoginMethodByPasswordResetToken(token)
	if err != nil || lm == nil {
		return false, nil
	}
	var expires sql.NullString
	err = db.QueryRow(`
		SELECT password_reset_token_expires_at FROM auth_login_method WHERE id = ?`, lm.ID).Scan(&expires)
	if err != nil || !expires.Valid {
		return false, lm
	}
	t, parseErr := parseDateTime(expires.String)
	if parseErr != nil {
		return false, lm
	}
	return time.Now().UTC().Before(t), lm
}

func updatePasswordHashClearResetToken(loginMethodID int64, password string) error {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("bcrypt hash: %w", err)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err = db.Exec(`
		UPDATE auth_login_method
		SET password_hash = ?, password_reset_token = NULL, password_reset_token_expires_at = NULL, updated_at = ?
		WHERE id = ?`, hashedPassword, now, loginMethodID)
	return err
}

func updatePasswordHashOnly(loginMethodID int64, password string) error {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("bcrypt hash: %w", err)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err = db.Exec(`
		UPDATE auth_login_method
		SET password_hash = ?, updated_at = ?
		WHERE id = ?`, hashedPassword, now, loginMethodID)
	return err
}

func updatePhoneLoginPassword(loginMethodID int64, password string) error {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("bcrypt hash: %w", err)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err = db.Exec(`
		UPDATE auth_login_method
		SET password_hash = ?, is_verified = 1, updated_at = ?
		WHERE id = ?`, hashedPassword, now, loginMethodID)
	return err
}

func createUserWithPhoneLogin(countryCode, national, password, email string) (userID string, err error) {
	userIDInt := generateSnowflakeID()
	userID = fmt.Sprintf("%d", userIDInt)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	var hashedPassword string
	if password != "" {
		hashedPassword, err = hashPassword(password)
		if err != nil {
			return "", fmt.Errorf("bcrypt hash: %w", err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO auth_user (id, password, last_login, is_superuser, is_staff, is_active, is_tenant, date_joined)
		VALUES (?, '', NULL, 0, 0, 1, 0, ?)`, userID, now)
	if err != nil {
		return "", err
	}

	lmID := generateSnowflakeID()
	_, err = tx.Exec(`
		INSERT INTO auth_login_method (
			id, content_type_id, object_id, method_type, identifier,
			phone_country_calling_code, password_hash, is_verified,
			created_at, updated_at
		) VALUES (?, ?, ?, 'phone', ?, ?, ?, 1, ?, ?)`,
		lmID, cfg.UserContentTypeID, userID, national, countryCode, hashedPassword, now, now)
	if err != nil {
		return "", err
	}

	if email != "" {
		emailLmID := generateSnowflakeID()
		_, err = tx.Exec(`
			INSERT INTO auth_login_method (
				id, content_type_id, object_id, method_type, identifier,
				password_hash, is_verified, created_at, updated_at
			) VALUES (?, ?, ?, 'email', ?, ?, 1, ?, ?)`,
			emailLmID, cfg.UserContentTypeID, userID, email, hashedPassword, now, now)
		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}
