package main

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"tracelog"
)

// ── User endpoints ──

func handleReferralCodeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	result, err := getReferralCodeStatus(userID)
	if err != nil {
		log.Printf("[taskAuth] referral status: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		return
	}
	log.Printf("[taskReferral] referral status user_id=%s has_active=%v access_code_set=%v",
		userID, result.HasActiveCode, result.AccessCode != "")
	writeJSON(w, http.StatusOK, result)
}

func handleReferralCodeApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	body, _ := readJSONBody(r)
	intro := ""
	legalName := ""
	consent := false
	if body != nil {
		intro = strField(body, "personal_intro")
		legalName = strField(body, "legal_name")
		consent = boolField(body, "identity_bind_consent")
	}
	result, err := applyReferralCode(userID, intro, legalName, consent)
	if err != nil {
		code := referralErrorCode(err)
		// OPT-20260819-007: 完整错误只打结构化日志并带 trace_id（元规则 24），
		// 不把 MySQL/SQL 字段原文回传前端；用户凭 data-traceId 排障。
		slog.ErrorContext(r.Context(), "referral_apply_failed",
			"level", "error",
			"user_id", userID,
			"error_code", code,
			"error", err.Error(),
			"personal_intro_len", utf8.RuneCountInString(strings.TrimSpace(intro)),
			"legal_name_len", utf8.RuneCountInString(strings.TrimSpace(legalName)),
			"identity_bind_consent", consent,
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		if code == "internal_error" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": code, "detail": "申请失败，请稍后重试"})
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": code, "detail": err.Error()})
		}
		return
	}
	slog.InfoContext(r.Context(), "referral_application_submitted",
		"level", "info",
		"user_id", userID,
		"status", result.Status,
		"personal_intro_len", utf8.RuneCountInString(strings.TrimSpace(intro)),
		"legal_name_len", utf8.RuneCountInString(strings.TrimSpace(legalName)),
		"identity_bind_consent", true,
		"trace_id", tracelog.TraceIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, result)
}

// handleReferralLegalNameUpdate 存量已获资格用户补填个人名称（OPT-20260823-048）：
// 校验名称 → 更新 referral_code.legal_name → 重新 ensure 微信分账接收方。
// 只允许已有 active（approved 且未过期）资格的用户调用；新申请仍走 apply 路径。
func handleReferralLegalNameUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	body, _ := readJSONBody(r)
	legalName := ""
	if body != nil {
		legalName = strField(body, "legal_name")
	}
	name, err := normalizeLegalName(legalName)
	if err != nil {
		slog.WarnContext(r.Context(), "referral_legal_name_update_invalid",
			"level", "warn",
			"user_id", userID,
			"error_code", "invalid_legal_name",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_legal_name", "detail": err.Error()})
		return
	}
	active := getActiveReferralCode(userID)
	if active == nil {
		slog.WarnContext(r.Context(), "referral_legal_name_update_no_active",
			"level", "warn",
			"user_id", userID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_active_qualification", "detail": "无有效推荐资格"})
		return
	}
	if _, err := db.Exec(`UPDATE referral_code SET legal_name = ? WHERE id = ?`, name, active.ID); err != nil {
		slog.ErrorContext(r.Context(), "referral_legal_name_update_failed",
			"level", "error",
			"user_id", userID,
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error", "detail": "保存失败，请稍后重试"})
		return
	}
	recv := ensureWechatProfitSharingReceiver(userID)
	slog.InfoContext(r.Context(), "referral_legal_name_updated",
		"level", "info",
		"user_id", userID,
		"legal_name_len", utf8.RuneCountInString(name),
		"wechat_receiver_status", recv.Status,
		"trace_id", tracelog.TraceIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":                 "ok",
		"legal_name":             name,
		"wechat_receiver_status": recv.Status,
		"wechat_receiver_reason": recv.Reason,
	})
}

// ── Admin endpoints ──

func handleAdminReferralApplications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	if _, ok := requireSuperuser(w, r); !ok {
		return
	}

	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	offset := 0
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	items, total, err := listReferralApplications(status, limit, offset)
	if err != nil {
		log.Printf("[taskAuth] referral list: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		return
	}
	if items == nil {
		items = []referralCodeRow{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func handleAdminReferralApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	reviewerID, ok := requireSuperuser(w, r)
	if !ok {
		return
	}

	// Extract application ID from URL path: /api/system-admin/referral/applications/{id}/approve/
	idStr := r.PathValue("id")
	appID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid application ID"})
		return
	}

	body, _ := readJSONBody(r)
	reason := ""
	if body != nil {
		reason = strField(body, "reason")
	}
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	result, err := approveReferralApplication(r.Context(), appID, reviewerID, reason, idemKey)
	if err != nil {
		log.Printf("[taskReferral] referral approve: %v", err)
		status := http.StatusBadRequest
		if err.Error() == "internal_error" {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func handleAdminReferralReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	reviewerID, ok := requireSuperuser(w, r)
	if !ok {
		return
	}

	idStr := r.PathValue("id")
	appID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid application ID"})
		return
	}

	body, _ := readJSONBody(r)
	reason := ""
	if body != nil {
		reason = strField(body, "reason")
	}
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	result, err := rejectReferralApplication(r.Context(), appID, reviewerID, reason, idemKey)
	if err != nil {
		log.Printf("[taskReferral] referral reject: %v", err)
		status := http.StatusBadRequest
		if err.Error() == "internal_error" {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func handleAdminReferralRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	reviewerID, ok := requireSuperuser(w, r)
	if !ok {
		return
	}
	idStr := r.PathValue("id")
	appID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid application ID"})
		return
	}
	body, _ := readJSONBody(r)
	reason := ""
	if body != nil {
		reason = strField(body, "reason")
	}
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	result, err := revokeReferralQualification(r.Context(), appID, reviewerID, reason, idemKey)
	if err != nil {
		slog.WarnContext(r.Context(), "referral_revoke_failed",
			"level", "warn",
			"app_id", appID,
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func handleAdminReferralAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	if _, ok := requireSuperuser(w, r); !ok {
		return
	}
	idStr := r.PathValue("id")
	appID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid application ID"})
		return
	}
	limit := 50
	offset := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v >= 0 {
		offset = v
	}
	items, total, err := listQualificationAudit(appID, limit, offset)
	if err != nil {
		slog.ErrorContext(r.Context(), "referral_audit_list_failed",
			"level", "error",
			"app_id", appID,
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func handleAdminReferralPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if _, ok := requireSuperuser(w, r); !ok {
			return
		}
		policy, err := getReferralPolicy()
		if err != nil {
			log.Printf("[taskAuth] referral policy get: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": policy})

	case http.MethodPut, http.MethodPost:
		if _, ok := requireSuperuser(w, r); !ok {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		mode := strField(body, "mode")
		message := strField(body, "message")

		policy, err := updateReferralPolicy(mode, message)
		if err != nil {
			log.Printf("[taskAuth] referral policy update: %v", err)
			if strings.Contains(err.Error(), "invalid mode") {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": policy})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
	}
}

// ── Helpers ──

func referralErrorCode(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, errReferralIntroRequired), errors.Is(err, errReferralIntroTooShort), errors.Is(err, errReferralIntroTooLong):
		return "invalid_intro"
	case errors.Is(err, errReferralLegalNameRequired), errors.Is(err, errReferralLegalNameTooShort), errors.Is(err, errReferralLegalNameTooLong), errors.Is(err, errReferralLegalNameInvalid):
		return "invalid_legal_name"
	case errors.Is(err, errReferralIdentityBindConsentRequired):
		return "identity_bind_consent_required"
	case errors.Is(err, errReferralServiceAccountNotFollowed):
		return "service_account_not_followed"
	case err.Error() == "referral_already_pending" || strings.Contains(err.Error(), "已有待审批"):
		return "already_pending"
	case err.Error() == "referral_already_active" || strings.Contains(err.Error(), "仍在有效期内"):
		return "already_active"
	case err.Error() == "referral_cooldown" || strings.Contains(err.Error(), "上次申请被拒绝"):
		return "cooldown"
	default:
		return "internal_error"
	}
}
