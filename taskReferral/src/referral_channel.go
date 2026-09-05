package main

import (
	"net/http"
	"strings"
)

func handleListReferralChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	list, err := listReferralChannels(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"channels": list})
}

func handleCreateReferralChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	row, err := createReferralChannel(userID, strField(body, "name"))
	if err != nil {
		switch err.Error() {
		case "invalid_name":
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid_name"})
		case "channel_limit":
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "channel_limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		}
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func handleDisableReferralChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	code := strings.TrimSpace(r.PathValue("code"))
	if err := disableReferralChannel(userID, code); err != nil {
		switch err.Error() {
		case "not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "not_found"})
		case "cannot_disable_default":
			writeJSON(w, http.StatusForbidden, map[string]string{"detail": "cannot_disable_default"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleDeleteReferralChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	code := strings.TrimSpace(r.PathValue("code"))
	if err := deleteReferralChannel(userID, code); err != nil {
		switch err.Error() {
		case "not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "not_found"})
		case "cannot_delete_default":
			writeJSON(w, http.StatusForbidden, map[string]string{"detail": "cannot_delete_default"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
