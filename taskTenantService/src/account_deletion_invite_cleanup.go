package main

import (
	"fmt"
	"net/http"
)

// cleanupStaleInvitesForUser invalidates pending invitations in every tenant where the
// given user is the only remaining active member. Called by taskAuth after executing
// account deletion (the user's membership is effectively archived, but the
// tenant_company_member row is left in place). OPT-20260820-019: after the last member
// of a tenant is archived, `tenant_invitation` rows that were never accepted must no
// longer be usable — otherwise an unexpired share link can still admit someone into a
// tenant whose members are all gone.
//
// Invalidation is implemented by back-dating invitation_token_expires_at (no schema
// change): handleJoin rejects expired tokens ("邀请链接无效或已过期"), and
// handlePendingInvitations lists only unexpired rows, so the invite disappears from
// both join and pending-list paths.
func cleanupStaleInvitesForUser(userID string) (int, error) {
	if userID == "" {
		return 0, nil
	}
	// Discover tenants from any membership row of the deleting user — the member row
	// is left in place (only auth_user is archived), so is_active is not filtered here.
	rows, err := db.Query(`SELECT DISTINCT company_id FROM tenant_company_member WHERE user_id=?`, userID)
	if err != nil {
		return 0, err
	}
	var tenants []string
	for rows.Next() {
		var cid string
		if err := rows.Scan(&cid); err != nil || cid == "" {
			continue
		}
		tenants = append(tenants, cid)
	}
	rows.Close()

	invalidated := 0
	for _, cid := range tenants {
		var otherActive int
		if err := db.QueryRow(
			`SELECT COUNT(1) FROM tenant_company_member WHERE company_id=? AND is_active=1 AND user_id<>?`,
			cid, userID,
		).Scan(&otherActive); err != nil {
			return invalidated, err
		}
		if otherActive > 0 {
			continue // tenant still has active members; keep invites usable
		}
		res, err := db.Exec(`
			UPDATE tenant_invitation
			SET invitation_token_expires_at = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 1 SECOND),
			    updated_at = CURRENT_TIMESTAMP
			WHERE company_id=? AND is_accepted=0
			  AND invitation_token_expires_at > CURRENT_TIMESTAMP`, cid)
		if err != nil {
			return invalidated, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			invalidated += int(n)
		}
	}
	return invalidated, nil
}

// handleInternalCleanupStaleInvites — POST /api/internal/tenant/account-deletion/cleanup-invites/
// Body: { "user_id": "..." }
func handleInternalCleanupStaleInvites(w http.ResponseWriter, r *http.Request) {
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	userID := strField(body, "user_id")
	if userID == "" {
		writeError(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	n, err := cleanupStaleInvitesForUser(userID)
	if err != nil {
		logError("cleanup_stale_invites failed user_id="+userID+" err="+err.Error(), traceIDForError(r))
		writeError(w, r, http.StatusInternalServerError, "cleanup failed")
		return
	}
	logInfo("cleanup_stale_invites user_id="+userID+" invalidated="+fmt.Sprintf("%d", n), traceIDForError(r))
	writeJSON(w, http.StatusOK, map[string]interface{}{"invalidated": n})
}
