package main

import (
	"log"
)

// The auth_email_invite_delivery_attempt table is now managed via dataMigrate SQL:
//   dataMigrate/taskAuth/015_email_invite_delivery_attempts.sql
// Migrated from Go inline DDL to dataMigrate directory (2026-08-03).

// recordDeliveryAttempt inserts a row into the delivery_attempt history table.
func recordDeliveryAttempt(invitationID int64, attemptNumber int, method, status, errMsg string) {
	now := timeNowUTC()
	id := generateSnowflakeID()
	deliveryMethod := string(method)
	if deliveryMethod == "" {
		deliveryMethod = "unknown"
	}
	_, dbErr := db.Exec(`
		INSERT INTO auth_email_invite_delivery_attempt
		(id, invitation_id, attempt_number, delivery_method, delivery_status, delivery_error, attempted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, invitationID, attemptNumber, deliveryMethod, status, errMsg, now,
	)
	if dbErr != nil {
		log.Printf("[taskAuth] recordDeliveryAttempt failed for invite %d: %v", invitationID, dbErr)
	}
}

// --- Delivery attempt history helper ---

// getDeliveryAttempts returns delivery attempt history for an invitation.
func getDeliveryAttempts(invitationID int64) []map[string]interface{} {
	rows, err := db.Query(`
		SELECT COALESCE(attempt_number, 0), delivery_method, delivery_status,
		       COALESCE(delivery_error, ''),
		       attempted_at
		FROM auth_email_invite_delivery_attempt
		WHERE invitation_id = ?
		ORDER BY attempt_number ASC`, invitationID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var attempts []map[string]interface{}
	for rows.Next() {
		var attemptNum int
		var method, status, errMsg, attemptedAt string
		if err := rows.Scan(&attemptNum, &method, &status, &errMsg, &attemptedAt); err != nil {
			continue
		}
		entry := map[string]interface{}{
			"attemptNumber":  attemptNum,
			"deliveryMethod": method,
			"deliveryStatus": status,
			"attemptedAt":    attemptedAt,
		}
		if errMsg != "" {
			entry["deliveryError"] = errMsg
		}
		attempts = append(attempts, entry)
	}
	if attempts == nil {
		return []map[string]interface{}{}
	}
	return attempts
}
