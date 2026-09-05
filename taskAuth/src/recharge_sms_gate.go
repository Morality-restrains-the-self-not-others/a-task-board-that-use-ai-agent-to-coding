package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	rechargeSMSOkTTL      = 15 * time.Minute
	rechargePendingTTL    = 10 * time.Minute
)

func setRechargeSMSVerified(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return sql.ErrNoRows
	}
	until := time.Now().UTC().Add(rechargeSMSOkTTL).Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		INSERT INTO auth_recharge_sms_gate (user_id, sms_ok_until, pending_phone, pending_until)
		VALUES (?, ?, NULL, NULL)
		ON DUPLICATE KEY UPDATE
			sms_ok_until = VALUES(sms_ok_until)`,
		userID, until,
	)
	return err
}

func isRechargeSMSVerified(userID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, nil
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var until sql.NullString
	err := db.QueryRow(`
		SELECT sms_ok_until FROM auth_recharge_sms_gate
		WHERE user_id = ?`, userID).Scan(&until)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return until.Valid && until.String > now, nil
}

func clearRechargeSMSGate(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	_, err := db.Exec(`DELETE FROM auth_recharge_sms_gate WHERE user_id = ?`, userID)
	return err
}

func setRechargePendingPhone(userID, phone string) error {
	userID = strings.TrimSpace(userID)
	phone = strings.TrimSpace(phone)
	if userID == "" || phone == "" {
		return sql.ErrNoRows
	}
	until := time.Now().UTC().Add(rechargePendingTTL).Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		INSERT INTO auth_recharge_sms_gate (user_id, sms_ok_until, pending_phone, pending_until)
		VALUES (?, NULL, ?, ?)
		ON DUPLICATE KEY UPDATE
			pending_phone = VALUES(pending_phone),
			pending_until = VALUES(pending_until)`,
		userID, phone, until,
	)
	return err
}

func getRechargePendingPhone(userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", nil
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var phone, until sql.NullString
	err := db.QueryRow(`
		SELECT pending_phone, pending_until FROM auth_recharge_sms_gate
		WHERE user_id = ?`, userID).Scan(&phone, &until)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !phone.Valid || !until.Valid || until.String <= now {
		return "", nil
	}
	return phone.String, nil
}

func clearRechargePendingPhone(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	_, err := db.Exec(`
		UPDATE auth_recharge_sms_gate
		SET pending_phone = NULL, pending_until = NULL
		WHERE user_id = ?`, userID)
	return err
}

func handleInternalRechargeSMSGate(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	switch r.Method {
	case http.MethodGet:
		userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
		ok, err := isRechargeSMSVerified(userID)
		if err != nil {
			log.Printf("[taskAuth] recharge gate get: %v", err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		pending, err := getRechargePendingPhone(userID)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"sms_verified":   ok,
			"pending_phone":  pending,
		})
	case http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		userID := strField(body, "user_id")
		action := strField(body, "action")
		switch action {
		case "set_verified":
			if err := setRechargeSMSVerified(userID); err != nil {
				writeErrorDetail(w, r, http.StatusBadRequest, "invalid user_id")
				return
			}
		case "set_pending":
			if err := setRechargePendingPhone(userID, strField(body, "phone")); err != nil {
				writeErrorDetail(w, r, http.StatusBadRequest, "invalid user_id or phone")
				return
			}
		case "clear_pending":
			_ = clearRechargePendingPhone(userID)
		case "clear_all":
			_ = clearRechargeSMSGate(userID)
		default:
			writeErrorDetail(w, r, http.StatusBadRequest, "unknown action")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	case http.MethodDelete:
		userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
		_ = clearRechargeSMSGate(userID)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}
