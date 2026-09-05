package main

import (
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"taskAuth/domain"
)

var errUsernameTaken = errors.New("username taken")
var errPhoneTaken = errors.New("phone taken")
var errPhoneBindLimit = errors.New("phone bind limit")
var errPhoneAmbiguous = errors.New("phone ambiguous")
var errEmailTaken = errors.New("email taken")

func handleLookupUserByEmail(w http.ResponseWriter, r *http.Request) {
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
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	email := strField(body, "email")
	if email == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "email required")
		return
	}
	lm, err := findLoginMethodByEmail(email)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if lm == nil {
		writeErrorDetail(w, r, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"user_id": lm.ObjectID})
}

func handleUsernameTaken(w http.ResponseWriter, r *http.Request) {
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
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	identifier := strField(body, "identifier")
	excludeUserID := strField(body, "exclude_user_id")
	if identifier == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "identifier required")
		return
	}
	taken, err := usernameLoginMethodTaken(identifier, excludeUserID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"taken": taken})
}

func handleUpsertUsernameLoginMethod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	userID := strings.Trim(r.PathValue("user_id"), "/")
	if userID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	identifier := strField(body, "identifier")
	if err := upsertUsernameLoginMethod(userID, identifier); err != nil {
		if errors.Is(err, errUsernameTaken) {
			writeErrorDetail(w, r, http.StatusConflict, "username taken")
			return
		}
		if err == sql.ErrNoRows {
			writeErrorDetail(w, r, http.StatusNotFound, "user not found")
			return
		}
		log.Printf("[taskAuth] upsert username failed for user %s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func usernameLoginMethodTaken(identifier, excludeUserID string) (bool, error) {
	var objectID string
	err := db.QueryRow(`
		SELECT object_id FROM auth_login_method
		WHERE method_type = 'username' AND LOWER(identifier) = LOWER(?)
		  AND binding_voided_at IS NULL
		LIMIT 1`, identifier).Scan(&objectID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if excludeUserID != "" && objectID == excludeUserID {
		return false, nil
	}
	return true, nil
}

func upsertUsernameLoginMethod(userID, identifier string) error {
	var exists int
	err := db.QueryRow(`SELECT 1 FROM auth_user WHERE id = ?`, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	identifier = strings.TrimSpace(identifier)
	if identifier != "" {
		taken, err := usernameLoginMethodTaken(identifier, userID)
		if err != nil {
			return err
		}
		if taken {
			return errUsernameTaken
		}
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var rowID int64
	err = db.QueryRow(`
		SELECT id FROM auth_login_method
		WHERE object_id = ? AND method_type = 'username' AND binding_voided_at IS NULL
		LIMIT 1`, userID).Scan(&rowID)
	if err == sql.ErrNoRows {
		if identifier == "" {
			return nil
		}
		lmID := generateSnowflakeID()
		_, err = db.Exec(`
			INSERT INTO auth_login_method (
				id, content_type_id, object_id, method_type, identifier, password_hash, is_verified, created_at, updated_at
			) VALUES (?, ?, ?, 'username', ?, '', 1, ?, ?)`,
			lmID, cfg.UserContentTypeID, userID, identifier, now, now)
		return err
	}
	if err != nil {
		return err
	}
	if identifier == "" {
		_, err = db.Exec(`DELETE FROM auth_login_method WHERE id = ?`, rowID)
		return err
	}
	_, err = db.Exec(`
		UPDATE auth_login_method
		SET identifier = ?, is_verified = 1, updated_at = ?
		WHERE id = ?`, identifier, now, rowID)
	return err
}

func handlePhoneTaken(w http.ResponseWriter, r *http.Request) {
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
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	cc := strField(body, "country_calling_code")
	national := strField(body, "national_number")
	excludeUserID := strField(body, "exclude_user_id")
	if cc == "" || national == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "country_calling_code and national_number required")
		return
	}
	taken, err := phoneLoginMethodTaken(cc, national, excludeUserID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"taken": taken})
}

func handleUpsertPhoneLoginMethod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	userID := strings.Trim(r.PathValue("user_id"), "/")
	if userID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	cc := strField(body, "country_calling_code")
	national := strField(body, "national_number")
	if err := upsertPhoneLoginMethod(userID, cc, national); err != nil {
		if errors.Is(err, errPhoneBindLimit) {
			writeErrorDetail(w, r, http.StatusConflict, "phone bind limit")
			return
		}
		if errors.Is(err, errPhoneTaken) {
			writeErrorDetail(w, r, http.StatusConflict, "phone taken")
			return
		}
		if err == sql.ErrNoRows {
			writeErrorDetail(w, r, http.StatusNotFound, "user not found")
			return
		}
		log.Printf("[taskAuth] upsert phone failed for user %s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	maybeEvaluateKycAfterPhoneVerified(withRequestTrace(r).Context(), userID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func phoneLoginMethodTaken(countryCode, national, excludeUserID string) (bool, error) {
	d, _, err := phoneBindDecision(excludeUserID, countryCode, national)
	if err != nil {
		return false, err
	}
	return d != domain.PhoneShareAllow, nil
}

func upsertPhoneLoginMethod(userID, countryCode, national string) error {
	var exists int
	err := db.QueryRow(`SELECT 1 FROM auth_user WHERE id = ?`, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	countryCode = strings.TrimSpace(countryCode)
	national = strings.TrimSpace(national)
	if countryCode == "" || national == "" {
		return errors.New("invalid phone")
	}
	d, _, err := phoneBindDecision(userID, countryCode, national)
	if err != nil {
		return err
	}
	if d == domain.PhoneShareLimit {
		return errPhoneBindLimit
	}
	if d == domain.PhoneShareTaken {
		return errPhoneTaken
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var rowID int64
	err = db.QueryRow(`
		SELECT id FROM auth_login_method
		WHERE object_id = ? AND method_type = 'phone' AND binding_voided_at IS NULL
		LIMIT 1`, userID).Scan(&rowID)
	if err == sql.ErrNoRows {
		lmID := generateSnowflakeID()
		_, err = db.Exec(`
			INSERT INTO auth_login_method (
				id, content_type_id, object_id, method_type, identifier,
				phone_country_calling_code, password_hash, is_verified, created_at, updated_at
			) VALUES (?, ?, ?, 'phone', ?, ?, '', 1, ?, ?)`,
			lmID, cfg.UserContentTypeID, userID, national, countryCode, now, now)
		return err
	}
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE auth_login_method
		SET identifier = ?, phone_country_calling_code = ?, is_verified = 1,
		    binding_voided_at = NULL, updated_at = ?
		WHERE id = ?`, national, countryCode, now, rowID)
	return err
}

// voidPhoneLoginMethodsForUser 作废该用户全部未作废的 phone login method。
// 系统管理编辑表单清空手机号时调用；0 行也视为成功（幂等）。
func voidPhoneLoginMethodsForUser(userID string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		UPDATE auth_login_method
		SET binding_voided_at = ?, updated_at = ?
		WHERE object_id = ? AND method_type = 'phone' AND binding_voided_at IS NULL`,
		now, now, userID)
	return err
}

// reclaimPhoneLoginMethod 在短信已证明持有该号后，作废其他活跃账号的绑定并绑定到当前用户。
func reclaimPhoneLoginMethod(userID, countryCode, national string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	res, err := db.Exec(`
		UPDATE auth_login_method
		SET binding_voided_at = ?, updated_at = ?
		WHERE method_type = 'phone'
		  AND phone_country_calling_code = ?
		  AND identifier = ?
		  AND binding_voided_at IS NULL
		  AND object_id != ?`, now, now, countryCode, national, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		slog.Info("phone_binding_reclaimed", "to_user_id", userID, "voided_others", n)
	}
	return upsertPhoneLoginMethod(userID, countryCode, national)
}

// upsertEmailLoginMethod 绑定/换绑邮箱登录方式（OPT-20260806-065/066 厂商门户前置）。
// 与 upsertPhoneLoginMethod 同构：同人已绑定 → 更新 identifier；被其他账号占用
// → errEmailTaken。邮箱小写归一；拒绝合成邮箱（sso-<id>@sso.invalid）。
func upsertEmailLoginMethod(userID, email string) error {
	var exists int
	err := db.QueryRow(`SELECT 1 FROM auth_user WHERE id = ?`, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") || strings.HasSuffix(email, "@sso.invalid") {
		return errors.New("invalid email")
	}
	lm, err := findLoginMethodByEmail(email)
	if err != nil {
		return err
	}
	if lm != nil && lm.ObjectID != userID {
		return errEmailTaken
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var rowID int64
	err = db.QueryRow(`
		SELECT id FROM auth_login_method
		WHERE object_id = ? AND method_type = 'email' AND binding_voided_at IS NULL
		LIMIT 1`, userID).Scan(&rowID)
	if err == sql.ErrNoRows {
		lmID := generateSnowflakeID()
		_, err = db.Exec(`
			INSERT INTO auth_login_method (
				id, content_type_id, object_id, method_type, identifier,
				password_hash, is_verified, created_at, updated_at
			) VALUES (?, ?, ?, 'email', ?, '', 1, ?, ?)`,
			lmID, cfg.UserContentTypeID, userID, email, now, now)
		return err
	}
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE auth_login_method
		SET identifier = ?, is_verified = 1, binding_voided_at = NULL, updated_at = ?
		WHERE id = ?`, email, now, rowID)
	return err
}
