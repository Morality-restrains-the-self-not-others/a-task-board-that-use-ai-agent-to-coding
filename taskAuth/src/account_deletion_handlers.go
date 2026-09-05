package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func handleAccountDeletionRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/accounts/users/me/account-deletion/")
	path = strings.Trim(path, "/")
	switch path {
	case "precheck":
		if r.Method == http.MethodGet {
			handleAccountDeletionPrecheck(w, r)
			return
		}
	case "status":
		if r.Method == http.MethodGet {
			handleAccountDeletionStatus(w, r)
			return
		}
	case "request":
		if r.Method == http.MethodPost {
			handleAccountDeletionRequest(w, r)
			return
		}
	case "cancel":
		if r.Method == http.MethodPost {
			handleAccountDeletionCancel(w, r)
			return
		}
	}
	writeErrorDetail(w, r, http.StatusNotFound, "not found")
}

func handleAccountDeletionPrecheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	payload, err := deletionPrecheckPayload(r.Context(), userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "precheck failed")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func handleAccountDeletionStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	payload, err := deletionStatusPayload(userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "status failed")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func handleAccountDeletionRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	if strField(body, "confirmation_text") != deletionConfirmText {
		writeErrorDetail(w, r, http.StatusBadRequest, "请在确认框输入「注销」")
		return
	}
	password := strField(body, "password")
	if !verifyDeletionReauth(userID, password) {
		writeErrorDetail(w, r, http.StatusUnauthorized, "身份验证失败，请重新输入密码")
		return
	}
	precheck, err := deletionPrecheckPayload(r.Context(), userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "precheck failed")
		return
	}
	if can, _ := precheck["can_request"].(bool); !can {
		writeErrorMap(w, r, http.StatusConflict, map[string]interface{}{
			"error":    "account_deletion_blocked",
			"detail":   "注销条件未满足，请先处理阻断项",
			"blockers": precheck["blockers"],
		})
		return
	}
	req, err := createDeletionRequest(r.Context(), userID, precheck)
	if err != nil {
		if strings.Contains(err.Error(), "already pending") || strings.Contains(err.Error(), "archived") {
			writeErrorDetail(w, r, http.StatusConflict, err.Error())
			return
		}
		writeErrorDetail(w, r, http.StatusInternalServerError, "request failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"status":        req.Status,
		"request_id":    fmt.Sprintf("%d", req.ID),
		"requested_at":  req.RequestedAt.UTC().Format(time.RFC3339),
		"effective_at":  req.EffectiveAt.UTC().Format(time.RFC3339),
		"cooldown_days": accountDeletionCooldownDays(),
	})
}

func handleAccountDeletionCancel(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if err := cancelDeletionRequest(r.Context(), userID); err != nil {
		if strings.Contains(err.Error(), "no pending") {
			writeErrorDetail(w, r, http.StatusConflict, "当前没有可撤回的注销申请")
			return
		}
		writeErrorDetail(w, r, http.StatusInternalServerError, "cancel failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func handleInternalAccountDeletionExecuteDue(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	n, err := executeDueDeletionRequests(r.Context(), 50)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "execute failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"processed": n})
}
