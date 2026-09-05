package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// --- Internal: email delivery callback from taskEvents consumer ---

// handleEmailDeliveryCallback receives delivery confirmation from the taskEvents
// Kafka consumer after it processes an EMAIL_SENT event. This closes the loop on
// async Kafka delivery — when the consumer successfully sends via SMTP, it calls
// back here to update delivery_status from "queued" to "delivered".
// POST /api/internal/email-delivery-callback/
func handleEmailDeliveryCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	email := strings.TrimSpace(strField(body, "email"))
	invitationIDStr := strings.TrimSpace(strField(body, "invitation_id"))
	status := strings.TrimSpace(strField(body, "status"))        // "delivered" or "failed"
	errMsg := strings.TrimSpace(strField(body, "error_message")) // only for failed
	attemptNumberStr := strings.TrimSpace(strField(body, "attempt_number"))

	if email == "" || status == "" {
		writeError(w, r, http.StatusBadRequest, "email and status are required")
		return
	}
	if status != "delivered" && status != "failed" && status != "skipped_unsubscribed" {
		writeError(w, r, http.StatusBadRequest, "status must be delivered, failed, or skipped_unsubscribed")
		return
	}

	// Find the pending invitation by email (or by invitation_id if provided)
	var inviteID int64
	if invitationIDStr != "" {
		fmt.Sscanf(invitationIDStr, "%d", &inviteID)
	} else {
		err = db.QueryRow(`
			SELECT id FROM auth_email_registration_invite
			WHERE email = ? AND status = 'pending'
			ORDER BY created_at DESC LIMIT 1`, email,
		).Scan(&inviteID)
		if err != nil {
			log.Printf("[taskAuth] delivery callback: no pending invite for %s: %v", email, err)
			writeError(w, r, http.StatusNotFound, "no pending invitation found for this email")
			return
		}
	}

	// Update the invitation's delivery status
	deliveryStatus := status
	_, err = db.Exec(`
		UPDATE auth_email_registration_invite
		SET delivery_status = ?, delivery_error = ?
		WHERE id = ?`, deliveryStatus, errMsg, inviteID)
	if err != nil {
		log.Printf("[taskAuth] delivery callback: db update failed for invite %d: %v", inviteID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	// Record the delivery attempt update from consumer
	attemptNum := 0
	if attemptNumberStr != "" {
		fmt.Sscanf(attemptNumberStr, "%d", &attemptNum)
	}
	method := "smtp" // consumer always sends via SMTP
	if status == "failed" {
		method = "smtp"
	}
	recordDeliveryAttempt(inviteID, attemptNum, method, status, errMsg)

	log.Printf("[taskAuth] delivery callback: invite=%d email=%s status=%s", inviteID, email, status)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "delivery status updated",
		"invitation_id": fmt.Sprintf("%d", inviteID),
		"status":        status,
	})
}
