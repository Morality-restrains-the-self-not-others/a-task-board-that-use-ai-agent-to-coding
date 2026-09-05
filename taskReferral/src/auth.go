package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
)

// resolveUserIDFromRequest reads X-User-Id header (set by gateway after token verification).
func resolveUserIDFromRequest(r *http.Request) (string, bool) {
	uid := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if uid != "" {
		return uid, true
	}
	// Fallback: Authorization header
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, "Token ") {
		return "", false // token validation requires taskAuth DB access
	}
	return "", false
}

// requireAuthenticatedUser checks that the request has a valid user identity.
func requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := resolveUserIDFromRequest(r)
	if !ok || userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "authentication required"})
		return "", false
	}
	return userID, true
}

// requireSuperuser checks that the user is a superuser.
func requireSuperuser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return "", false
	}
	isSuper, err := isSuperuser(userID)
	if err != nil {
		log.Printf("[taskReferral] superuser check: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		return "", false
	}
	if !isSuper {
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": "forbidden"})
		return "", false
	}
	return userID, true
}

// isSuperuser checks both auth_user.is_superuser and auth_super_admin.
func isSuperuser(userID string) (bool, error) {
	if authDB == nil {
		return false, nil
	}
	var isSuper bool
	err := authDB.QueryRow(
		`SELECT is_superuser FROM auth_user WHERE id = ?`, userID,
	).Scan(&isSuper)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if isSuper {
		return true, nil
	}
	// Also check auth_super_admin
	var count int
	err = authDB.QueryRow(
		`SELECT COUNT(*) FROM auth_super_admin WHERE user_id = ?`, userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
