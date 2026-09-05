package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"taskAuth/domain"
	"tracelog"
)

// ——— DB operations ———

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func generateRawToken() string {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("rand: %v", err))
	}
	return "at_" + hex.EncodeToString(b)
}

// expiresAtFromBody computes an ISO datetime from expires_in_days (default: 0 = no expiry).
func expiresAtFromBody(body map[string]interface{}) string {
	daysRaw := strField(body, "expires_in_days")
	if daysRaw == "" || daysRaw == "0" {
		return ""
	}
	days, err := strconv.Atoi(daysRaw)
	if err != nil || days <= 0 {
		return ""
	}
	return time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Format("2006-01-02 15:04:05.000000")
}

func createAccessToken(userID, name string, expiresAt string) (map[string]interface{}, string, error) {
	raw := generateRawToken()
	hash := hashToken(raw)
	last4 := raw[len(raw)-4:]
	now := timeNowUTC()
	id := generateSnowflakeID()

	var err error
	if expiresAt != "" {
		_, err = db.Exec(`
			INSERT INTO auth_user_access_token (id, user_id, name, token_hash, last_4, created_at, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, userID, name, hash, last4, now, expiresAt)
	} else {
		_, err = db.Exec(`
			INSERT INTO auth_user_access_token (id, user_id, name, token_hash, last_4, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, userID, name, hash, last4, now)
	}
	if err != nil {
		return nil, "", err
	}

	record := map[string]interface{}{
		"id":         fmt.Sprintf("%d", id),
		"name":       name,
		"last_4":     last4,
		"created_at": now,
		"is_revoked": false,
	}
	if expiresAt != "" {
		record["expires_at"] = expiresAt
	}

	// Audit: token created
	insertAccessTokenAudit(userID, fmt.Sprintf("%d", id), "created", name)

	return record, raw, nil
}

func validateAccessToken(raw string) (userID string, err error) {
	h := hashToken(raw)
	var uid string
	var isRevoked bool
	var expires sql.NullString
	var tokenID int64
	err = db.QueryRow(`
		SELECT id, user_id, is_revoked, expires_at
		FROM auth_user_access_token
		WHERE token_hash = ?`, h).Scan(&tokenID, &uid, &isRevoked, &expires)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if isRevoked {
		return "", nil
	}
	if expires.Valid {
		t, parseErr := parseDateTime(expires.String)
		if parseErr == nil && time.Now().UTC().After(t) {
			return "", nil
		}
	}
	// Update last_used_at (best-effort)
	_, _ = db.Exec(`UPDATE auth_user_access_token SET last_used_at = ? WHERE token_hash = ?`,
		timeNowUTC(), h)
	// Audit: token used
	insertAccessTokenAudit(uid, fmt.Sprintf("%d", tokenID), "used", "")
	return uid, nil
}

func listAccessTokensForUser(userID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT id, name, last_4, created_at, last_used_at, expires_at, is_revoked
		FROM auth_user_access_token
		WHERE user_id = ?
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, last4, createdAt string
		var lastUsed, expires sql.NullString
		var isRevoked bool
		if err := rows.Scan(&id, &name, &last4, &createdAt, &lastUsed, &expires, &isRevoked); err != nil {
			return nil, err
		}
		entry := map[string]interface{}{
			"id":         fmt.Sprintf("%d", id),
			"name":       name,
			"last_4":     last4,
			"created_at": createdAt,
			"is_revoked": isRevoked,
		}
		if lastUsed.Valid {
			entry["last_used_at"] = lastUsed.String
		}
		if expires.Valid {
			entry["expires_at"] = expires.String
		}
		tokens = append(tokens, entry)
	}
	if tokens == nil {
		tokens = []map[string]interface{}{}
	}
	return tokens, nil
}

func revokeAccessToken(tokenID, userID string) (bool, error) {
	result, err := db.Exec(`
		UPDATE auth_user_access_token SET is_revoked = 1
		WHERE id = ? AND user_id = ? AND is_revoked = 0`, tokenID, userID)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		// Audit: token revoked
		insertAccessTokenAudit(userID, tokenID, "revoked", "")
	}
	return n > 0, nil
}

// ——— Audit log ———

func insertAccessTokenAudit(userID, tokenID, action, detail string) {
	now := timeNowUTC()
	id := generateSnowflakeID()
	_, err := db.Exec(`
		INSERT INTO auth_user_access_token_audit (id, user_id, token_id, action, detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, tokenID, action, detail, now)
	if err != nil {
		log.Printf("[taskAuth] access token audit insert error: %v", err)
	}
}

// ——— tokenFromRequest (extract Bearer or Token from Authorization header) ———

func tokenFromAuthHeader(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if strings.HasPrefix(auth, "Token ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Token "))
	}
	return auth
}

// resolveUserIDFromRequest tries: X-User-Id (gateway) → Bearer/Token → access token → custom token.
// Returns (userID, true) on success.
func resolveUserIDFromRequest(r *http.Request) (string, bool) {
	if uid := strings.TrimSpace(r.Header.Get("X-User-Id")); uid != "" {
		return uid, true
	}

	authHeader := tokenFromAuthHeader(r)
	if authHeader == "" {
		return "", false
	}

	// 1. Try bearer as access token first
	if uid, err := validateAccessToken(authHeader); err == nil && uid != "" {
		return uid, true
	}

	// 2. Try as custom token (existing auth flow)
	if uid, err := resolveTokenUserID(authHeader); err == nil && uid != "" {
		return uid, true
	}

	return "", false
}

// ——— HTTP handlers ———

func handleCreateAccessToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := resolveUserIDFromRequest(r)
	if !ok {
		if !requireInternalSecret(r) {
			writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
			return
		}
		body, _ := readJSONBody(r)
		userID = strField(body, "user_id")
		if userID == "" {
			writeError(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		record, raw, err := createAccessToken(userID, strField(body, "name"), expiresAtFromBody(body))
		if err != nil {
			log.Printf("[taskAuth] create access token error: %v", err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"token":  raw,
			"record": record,
		})
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	name := strField(body, "name")
	if name == "" {
		name = "default"
	}
	record, raw, err := createAccessToken(userID, name, expiresAtFromBody(body))
	if err != nil {
		log.Printf("[taskAuth] create access token error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"token":  raw,
		"record": record,
	})
}

func handleListAccessTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := resolveUserIDFromRequest(r)
	if !ok {
		if !requireInternalSecret(r) {
			writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
			return
		}
		userID = r.URL.Query().Get("user_id")
		if userID == "" {
			writeError(w, r, http.StatusBadRequest, "user_id required")
			return
		}
	}
	tokens, err := listAccessTokensForUser(userID)
	if err != nil {
		log.Printf("[taskAuth] list access tokens error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tokens": tokens})
}

func handleRevokeAccessToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := resolveUserIDFromRequest(r)
	if !ok {
		if !requireInternalSecret(r) {
			writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
			return
		}
		body, _ := readJSONBody(r)
		userID = strField(body, "user_id")
	}
	tokenID := strings.Trim(r.PathValue("token_id"), "/")
	if tokenID == "" {
		writeError(w, r, http.StatusBadRequest, "token_id required")
		return
	}
	revoked, err := revokeAccessToken(tokenID, userID)
	if err != nil {
		log.Printf("[taskAuth] revoke access token error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !revoked {
		writeErrorDetail(w, r, http.StatusNotFound, "token not found or already revoked")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// accessTokenMatchesIdentifier returns true when identifier resolves to userID.
func accessTokenMatchesIdentifier(userID, identifier string) (bool, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || userID == "" {
		return false, nil
	}
	lm, err := findLoginMethodByIdentifier(identifier)
	if err != nil {
		return false, err
	}
	if lm != nil && lm.ObjectID == userID {
		return true, nil
	}
	_, username, _, err := lookupUserByID(userID)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(username), identifier), nil
}

func handleLoginWithAccessToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	accessToken := strField(body, "access_token")
	if accessToken == "" {
		writeError(w, r, http.StatusBadRequest, "access_token required")
		return
	}
	username := strField(body, "username")
	if username == "" {
		username = strField(body, "email")
	}
	if username == "" {
		writeError(w, r, http.StatusBadRequest, "username required")
		return
	}

	userID, err := validateAccessToken(accessToken)
	if err != nil {
		log.Printf("[taskAuth] access token login db error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "令牌无效或已过期")
		return
	}

	matched, matchErr := accessTokenMatchesIdentifier(userID, username)
	if matchErr != nil {
		log.Printf("[taskAuth] access token login identifier match error: %v", matchErr)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !matched {
		writeError(w, r, http.StatusUnauthorized, "账号与令牌不匹配")
		return
	}

	// 入口分离（OPT-20260824-001）：访问令牌登录属客户入口，管理员账号拒绝。
	if !loginEntryAllowedForRole(w, r, userID, "customer") {
		return
	}

	active, err := userIsActive(userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !active {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"non_field_errors": []string{"账号已被禁用"},
		})
		return
	}

	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, resolveClientIP(r))
	if err != nil {
		log.Printf("[taskAuth] access token login — token error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"detail":   "token error",
			"trace_id": tracelog.TraceIDFromContext(r.Context()),
		})
		return
	}

	recordSuccessfulLoginFromRequest(r, userID, username, "token", domain.LoginEntryAccessToken, "")
	touchLastLogin(userID)
	resp := buildLoginResponse(userID, token)
	writeJSON(w, http.StatusOK, resp)
}

// constantTimeCompare is a thin wrapper for tests.
func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// handleInternalResolveAccessToken resolves an access token for internal callers (Django).
// Requires X-TaskAuth-Internal-Secret.
func handleInternalResolveAccessToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	accessToken := strField(body, "access_token")
	if accessToken == "" {
		writeError(w, r, http.StatusBadRequest, "access_token required")
		return
	}
	userID, err := validateAccessToken(accessToken)
	if err != nil {
		log.Printf("[taskAuth] internal resolve access token db error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if userID == "" {
		writeErrorDetail(w, r, http.StatusNotFound, "invalid access token")
		return
	}
	active, err := userIsActive(userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !active {
		writeErrorDetail(w, r, http.StatusForbidden, "user inactive")
		return
	}
	isSuper, _ := isSuperAdminUser(userID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":      userID,
		"is_active":    active,
		"is_superuser": isSuper,
		"is_staff":     isSuper,
	})
}
