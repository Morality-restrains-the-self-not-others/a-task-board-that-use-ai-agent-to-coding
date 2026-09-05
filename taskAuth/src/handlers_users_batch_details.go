package main

import (
	"fmt"
	"net/http"
	"strings"
)

// handleBatchUserDetails returns email/phone/username for a batch of user IDs.
// Internal endpoint — requires X-TaskAuth-Internal-Secret.
//
// Body: {"user_ids": ["...", ...]} (max 200).
// Response: {"results": {"<user_id>": {"user_id": "...", "email": "...", "phone": "...", "username": "..."}}}.
func handleBatchUserDetails(w http.ResponseWriter, r *http.Request) {
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
	rawIDs, ok := body["user_ids"]
	if !ok {
		writeError(w, r, http.StatusBadRequest, "user_ids required")
		return
	}
	idList, ok := rawIDs.([]interface{})
	if !ok {
		writeError(w, r, http.StatusBadRequest, "user_ids must be array")
		return
	}
	if len(idList) > 200 {
		writeError(w, r, http.StatusBadRequest, "too many user_ids (max 200)")
		return
	}
	userIDs := make([]string, 0, len(idList))
	seen := make(map[string]bool)
	for _, item := range idList {
		var userID string
		switch v := item.(type) {
		case string:
			userID = strings.TrimSpace(v)
		default:
			userID = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if userID == "" || seen[userID] {
			continue
		}
		seen[userID] = true
		userIDs = append(userIDs, userID)
	}

	results := make(map[string]interface{}, len(userIDs))
	if len(userIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
		return
	}

	emailByUser := loginIdentifiersByType(userIDs, "email")
	phoneByUser := loginIdentifiersByType(userIDs, "phone")
	usernameByUser := batchProfileUsernames(userIDs)

	for _, uid := range userIDs {
		results[uid] = map[string]interface{}{
			"user_id":  uid,
			"email":    emailByUser[uid],
			"phone":    phoneByUser[uid],
			"username": usernameByUser[uid],
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

func loginIdentifiersByType(userIDs []string, methodType string) map[string]string {
	out := make(map[string]string, len(userIDs))
	if len(userIDs) == 0 {
		return out
	}
	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, 0, len(userIDs)+1)
	args = append(args, methodType)
	for i, uid := range userIDs {
		placeholders[i] = "?"
		args = append(args, uid)
	}
	rows, err := db.Query(`
		SELECT object_id, identifier FROM auth_login_method
		WHERE method_type = ? AND object_id IN (`+strings.Join(placeholders, ",")+`)
		  AND binding_voided_at IS NULL
		ORDER BY is_verified DESC`, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var objectID, identifier string
		if rows.Scan(&objectID, &identifier) != nil {
			continue
		}
		if _, exists := out[objectID]; !exists {
			out[objectID] = identifier
		}
	}
	return out
}
