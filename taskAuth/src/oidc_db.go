package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type oidcClientRow struct {
	ID               string
	ClientID         string
	ClientSecretHash string
	Name             string
	RedirectURIs     string
	ManagedBy        string // 'bootstrap' | 'admin' | 'tenant'（ADR-0043）
	OwnerCompanyID   string
	Purpose          string
}

type oidcAuthRow struct {
	ID                  string
	Code                string
	ClientID            string
	UserID              string
	RedirectURI         string
	Scope               string
	Nonce               sql.NullString
	CodeChallenge       sql.NullString
	CodeChallengeMethod sql.NullString
	ExpiresAt           string
	Used                bool
}

func loadOidcClient(clientID string) (*oidcClientRow, error) {
	row := db.QueryRow(`
		SELECT id, client_id, client_secret_hash, name, redirect_uris,
		       COALESCE(managed_by, 'bootstrap'), COALESCE(owner_company_id, ''), COALESCE(purpose, '')
		FROM auth_oidc_client WHERE client_id = ?`, clientID)
	var r oidcClientRow
	err := row.Scan(&r.ID, &r.ClientID, &r.ClientSecretHash, &r.Name, &r.RedirectURIs, &r.ManagedBy, &r.OwnerCompanyID, &r.Purpose)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func countOidcClients() (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM auth_oidc_client`).Scan(&n)
	return n, err
}

func ensureOidcClient(clientID, clientSecretPlain, name, redirectURIsJSON string) error {
	row, err := loadOidcClient(clientID)
	if err != nil {
		return err
	}
	if row != nil {
		// OPT-20260808-025: managed_by='admin' 行是管理员托管（以 DB 为准）——
		// seed 仅 INSERT 缺失，不 UPDATE（否则覆盖管理员对 redirect_uris 的改库）。
		// managed_by='bootstrap'（默认）维持原 conf 自愈语义，行为不变。
		if row.ManagedBy == "admin" || row.ManagedBy == "tenant" {
			return nil
		}
		// Update redirect_uris if changed (self-healing on config change)
		if row.RedirectURIs != redirectURIsJSON {
			now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
			_, err = db.Exec(`UPDATE auth_oidc_client SET redirect_uris = ?, updated_at = ? WHERE client_id = ?`,
				redirectURIsJSON, now, clientID)
			if err != nil {
				return fmt.Errorf("update auth_oidc_client redirect_uris: %w", err)
			}
			log.Printf("[taskAuth] oidc bootstrap client redirect_uris updated: %s -> %s", row.RedirectURIs, redirectURIsJSON)
		}
		return nil
	}
	id := fmt.Sprintf("%d", generateSnowflakeID())
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	secretHash := hashClientSecret(clientSecretPlain)
	_, err = db.Exec(`
		INSERT INTO auth_oidc_client (id, client_id, client_secret_hash, name, redirect_uris, managed_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'bootstrap', ?, ?)`,
		id, clientID, secretHash, name, redirectURIsJSON, now, now)
	return err
}

func storeAuthorizationCode(code, clientID, userID, redirectURI, scope, nonce, codeChallenge, codeChallengeMethod string) error {
	id := fmt.Sprintf("%d", generateSnowflakeID())
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	expires := time.Now().UTC().Add(10 * time.Minute).Format("2006-01-02 15:04:05.000000")
	var nonceVal interface{} = nil
	if nonce != "" {
		nonceVal = nonce
	}
	var challengeVal interface{} = nil
	if codeChallenge != "" {
		challengeVal = codeChallenge
	}
	var challengeMethodVal interface{} = nil
	if codeChallengeMethod != "" {
		challengeMethodVal = codeChallengeMethod
	}
	_, err := db.Exec(`
		INSERT INTO auth_oidc_authorization (id, code, client_id, user_id, redirect_uri, scope, nonce, code_challenge, code_challenge_method, expires_at, used, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`,
		id, code, clientID, userID, redirectURI, scope, nonceVal, challengeVal, challengeMethodVal, expires, now)
	return err
}

func consumeAuthorizationCode(code, clientID, redirectURI string) (*oidcAuthRow, error) {
	// Atomic: mark used in a single UPDATE and check rowsAffected to prevent TOCTOU reuse.
	result, err := db.Exec(`
		UPDATE auth_oidc_authorization SET used = 1
		WHERE code = ? AND used = 0 AND client_id = ? AND redirect_uri = ?`,
		code, clientID, redirectURI)
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil // code not found, already used, or client/redirect mismatch
	}

	// Now fetch the row we just claimed (used=1)
	row := db.QueryRow(`
		SELECT id, code, client_id, user_id, redirect_uri, scope,
		       COALESCE(nonce,''), COALESCE(code_challenge,''), COALESCE(code_challenge_method,''), expires_at
		FROM auth_oidc_authorization
		WHERE code = ?`, code)
	var r oidcAuthRow
	var nonceStr, challengeStr, challengeMethodStr string
	err = row.Scan(&r.ID, &r.Code, &r.ClientID, &r.UserID, &r.RedirectURI, &r.Scope,
		&nonceStr, &challengeStr, &challengeMethodStr, &r.ExpiresAt)
	if err != nil {
		return nil, err
	}
	r.Nonce = sql.NullString{String: nonceStr, Valid: nonceStr != ""}
	r.CodeChallenge = sql.NullString{String: challengeStr, Valid: challengeStr != ""}
	r.CodeChallengeMethod = sql.NullString{String: challengeMethodStr, Valid: challengeMethodStr != ""}
	r.Used = true

	expiresTime, err := parseDateTime(r.ExpiresAt)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(expiresTime) {
		return nil, fmt.Errorf("authorization code expired")
	}

	return &r, nil
}

// ----- OIDC refresh token（offline_access grant）-----
// 轮换制：存储仅保留 SHA-256 哈希（sha256Hash），消费即撤销（revoked=1），
// 同时签发新 token；并发/重放旧 token 一律 invalid_grant（见 consumeRefreshToken）。

type oidcRefreshTokenRow struct {
	ID        string
	TokenHash string
	ClientID  string
	UserID    string
	Scope     string
	ExpiresAt string
	Revoked   bool
}

// storeRefreshToken 写入新 refresh token（ttlSeconds 起算自当前时间）。
// 顺带惰性清理同 client+user 的过期/撤销行，避免表无限膨胀。
func storeRefreshToken(clientID, userID, scope, tokenHash string, ttlSeconds int64) error {
	id := fmt.Sprintf("%d", generateSnowflakeID())
	now := time.Now().UTC()
	expires := now.Add(time.Duration(ttlSeconds) * time.Second).Format("2006-01-02 15:04:05.000000")
	nowStr := now.Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		INSERT INTO auth_oidc_refresh_token (id, token_hash, client_id, user_id, scope, expires_at, revoked, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		id, tokenHash, clientID, userID, scope, expires, nowStr)
	if err != nil {
		return err
	}
	// 惰性清理：仅扫当前用户行，量小
	_, _ = db.Exec(`
		DELETE FROM auth_oidc_refresh_token
		WHERE client_id = ? AND user_id = ?
		  AND (revoked = 1 OR expires_at < ?)`,
		clientID, userID, nowStr)
	return nil
}

// consumeRefreshToken 原子消费 refresh token：先置 revoked=1 再读取，
// 防止 TOCTOU 并发重放。未找到 / 已撤销 / 客户端不匹配 / 已过期 → nil（调用方回 invalid_grant）。
func consumeRefreshToken(tokenPlain, clientID string) (*oidcRefreshTokenRow, error) {
	hash := sha256Hash(tokenPlain)
	result, err := db.Exec(`
		UPDATE auth_oidc_refresh_token SET revoked = 1, last_used_at = ?
		WHERE token_hash = ? AND revoked = 0 AND client_id = ?`,
		time.Now().UTC().Format("2006-01-02 15:04:05.000000"), hash, clientID)
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil // 不存在、已撤销或 client 不匹配
	}
	row := db.QueryRow(`
		SELECT id, token_hash, client_id, user_id, scope, expires_at, revoked
		FROM auth_oidc_refresh_token WHERE token_hash = ?`, hash)
	var r oidcRefreshTokenRow
	if err := row.Scan(&r.ID, &r.TokenHash, &r.ClientID, &r.UserID, &r.Scope, &r.ExpiresAt, &r.Revoked); err != nil {
		return nil, err
	}
	expiresTime, err := parseDateTime(r.ExpiresAt)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(expiresTime) {
		return nil, fmt.Errorf("refresh token expired")
	}
	return &r, nil
}

// scopeHasOfflineAccess 判断请求 scope 是否含 offline_access（决定是否签发 refresh token）。
func scopeHasOfflineAccess(scopes []string) bool {
	for _, s := range scopes {
		if s == "offline_access" {
			return true
		}
	}
	return false
}

// scopeHasOpenID 判断 scope 是否含 openid（refresh grant 据此决定是否补发 id_token）。
func scopeHasOpenID(scopes []string) bool {
	for _, s := range scopes {
		if s == "openid" {
			return true
		}
	}
	return false
}

func lookupUserByID(userID string) (email string, username string, isActive bool, err error) {
	err = db.QueryRow(`SELECT is_active FROM auth_user WHERE id = ?`, userID).Scan(&isActive)
	if err != nil {
		return "", "", false, err
	}
	row := db.QueryRow(`
		SELECT identifier FROM auth_login_method
		WHERE object_id = ? AND method_type = 'email' AND binding_voided_at IS NULL
		ORDER BY created_at LIMIT 1`, userID)
	if err := row.Scan(&email); err != nil {
		email = ""
	}
	username = email
	if username == "" {
		row2 := db.QueryRow(`
			SELECT identifier FROM auth_login_method
			WHERE object_id = ? AND binding_voided_at IS NULL
			ORDER BY created_at LIMIT 1`, userID)
		var ident string
		if err := row2.Scan(&ident); err == nil && ident != "" {
			username = ident
		}
	}
	return email, username, isActive, nil
}
