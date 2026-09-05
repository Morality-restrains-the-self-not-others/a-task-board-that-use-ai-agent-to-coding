package main

import (
	"log"
	"net/http"
	"strings"
)

// ---- public KYC endpoints (authenticated via session/cookie/token, not internal secret) ----

// requireStaffOrSuperuser checks the authenticated user is staff or superuser.
// Returns userID if authorized, empty string + writes error response if not.
func requireStaffOrSuperuser(w http.ResponseWriter, r *http.Request) string {
	userID, err := resolveTokenUserIDFromRequestStrict(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusUnauthorized, "未登录")
		return ""
	}
	isActive, isSuperuser, isStaff, isArchived, err := loadUserAuthFlags(userID)
	if err != nil || !isActive || isArchived {
		writeErrorDetail(w, r, http.StatusUnauthorized, "未登录")
		return ""
	}
	if !isSuperuser && !isStaff {
		writeErrorDetail(w, r, http.StatusForbidden, "权限不足")
		return ""
	}
	return userID
}

// requireAuthenticated checks the user is authenticated and returns userID.
func requireAuthenticated(w http.ResponseWriter, r *http.Request) string {
	userID, err := resolveTokenUserIDFromRequestStrict(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusUnauthorized, "未登录")
		return ""
	}
	isActive, _, _, isArchived, err := loadUserAuthFlags(userID)
	if err != nil || !isActive || isArchived {
		writeErrorDetail(w, r, http.StatusUnauthorized, "未登录")
		return ""
	}
	return userID
}

// GET /api/kyc/admin/users/{user_id}/
// Admin aggregate: profile + audit + latest AML hint.
func handlePublicKycAdminGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if requireStaffOrSuperuser(w, r) == "" {
		return
	}
	targetUserID := strings.Trim(r.PathValue("user_id"), "/")
	if targetUserID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "缺少 user_id")
		return
	}

	profile, err := getOrCreateKycProfile(targetUserID)
	if err != nil {
		log.Printf("[taskAuth] public kyc admin get: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	auditItems, err := listKycAudit(targetUserID, 50)
	if err != nil {
		log.Printf("[taskAuth] public kyc admin get audit: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if auditItems == nil {
		auditItems = []kycAuditEntry{}
	}

	latestAML := extractLatestAMLFromAudit(auditItems)

	// OPT-20260819-029：附带实时限额策略，超管 drawer 等级说明与 auth_kyc_limit_policy 同源。
	policies, err := listKycLimitPolicies()
	if err != nil {
		log.Printf("[taskAuth] public kyc admin get policies: %v", err)
		policies = []kycLimitPolicy{}
	}
	if policies == nil {
		policies = []kycLimitPolicy{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":        targetUserID,
		"profile":        profile,
		"audit":          auditItems,
		"latest_aml":     latestAML,
		"limit_policies": policies,
	})
}

// POST /api/kyc/admin/users/{user_id}/override/
// Admin override KYC tier/status.
func handlePublicKycAdminOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	actorID := requireStaffOrSuperuser(w, r)
	if actorID == "" {
		return
	}
	targetUserID := strings.Trim(r.PathValue("user_id"), "/")
	if targetUserID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "缺少 user_id")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	tier := strField(body, "tier")
	status := strField(body, "status")
	if tier == "" || status == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "tier 与 status 必填")
		return
	}
	if !isValidKycTier(tier) {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid tier")
		return
	}
	if !isValidKycStatus(status) {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid status")
		return
	}
	reasonCode := strField(body, "reason_code")
	if reasonCode == "" {
		reasonCode = "ADMIN_OVERRIDE"
	}
	reasonDetail := strField(body, "reason_detail")
	profile, err := adminOverrideKyc(r.Context(), targetUserID, tier, status, reasonCode, reasonDetail, actorID)
	if err != nil {
		log.Printf("[taskAuth] public kyc admin override: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// POST /api/kyc/admin/users/{user_id}/aml/
// Record AML screening result.
func handlePublicKycAdminAml(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	actorID := requireStaffOrSuperuser(w, r)
	if actorID == "" {
		return
	}
	targetUserID := strings.Trim(r.PathValue("user_id"), "/")
	if targetUserID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "缺少 user_id")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	result := strField(body, "result")
	if result == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "result 必填")
		return
	}
	provider := strField(body, "provider")
	if provider == "" {
		provider = "manual"
	}
	r = withRequestTrace(r)
	rec, err := recordAmlScreening(
		r.Context(),
		targetUserID,
		result,
		provider,
		strField(body, "notes"),
		actorID,
		strField(body, "screening_ref"),
	)
	if err != nil {
		log.Printf("[taskAuth] public kyc admin aml: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// POST /api/kyc/admin/users/{user_id}/evaluate/
// Trigger KYC auto-evaluation.
func handlePublicKycAdminEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if requireStaffOrSuperuser(w, r) == "" {
		return
	}
	targetUserID := strings.Trim(r.PathValue("user_id"), "/")
	if targetUserID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "缺少 user_id")
		return
	}
	r = withRequestTrace(r)
	profile, err := evaluateKyc(r.Context(), targetUserID)
	if err != nil {
		log.Printf("[taskAuth] public kyc admin evaluate: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// GET /api/kyc/me/
// Authenticated user's own KYC summary (profile + limit policies).
func handlePublicKycMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := requireAuthenticated(w, r)
	if userID == "" {
		return
	}
	profile, err := getOrCreateKycProfile(userID)
	if err != nil {
		log.Printf("[taskAuth] public kyc me profile: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	policies, err := listKycLimitPolicies()
	if err != nil {
		log.Printf("[taskAuth] public kyc me policies: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, profileSummaryWithLimits(profile, policies))
}

// ---- helpers (mirrored from Django kyc_client.py aggregation logic) ----

// extractLatestAMLFromAudit scans audit items for the most recent AML record.
func extractLatestAMLFromAudit(auditItems []kycAuditEntry) map[string]interface{} {
	for _, item := range auditItems {
		if item.TriggerSource != "aml" {
			continue
		}
		detail := item.ReasonDetail
		result := ""
		const prefix = "aml result="
		if strings.HasPrefix(detail, prefix) {
			result = strings.TrimSpace(detail[len(prefix):])
		}
		return map[string]interface{}{
			"result":        result,
			"actor_id":      item.ActorID,
			"reason_code":   item.ReasonCode,
			"reason_detail": detail,
			"checked_at":    item.CreatedAt,
			"source":        "audit",
		}
	}
	return nil
}

// profileSummaryWithLimits builds the user-facing KYC summary.
func profileSummaryWithLimits(profile kycProfile, policies []kycLimitPolicy) map[string]interface{} {
	tier := profile.Tier
	if tier == "" {
		tier = "T0_unverified"
	}
	status := profile.Status
	if status == "" {
		status = "none"
	}

	maxSingle := 0
	maxDaily := 0
	rechargeAllowed := false
	for _, p := range policies {
		if p.Tier == tier {
			maxSingle = p.MaxSingleYuan
			maxDaily = p.MaxDailyYuan
			rechargeAllowed = p.RechargeAllowed
			break
		}
	}

	return map[string]interface{}{
		"user_id":          profile.UserID,
		"tier":             tier,
		"status":           status,
		"max_single_yuan":  maxSingle,
		"max_daily_yuan":   maxDaily,
		"recharge_allowed": rechargeAllowed,
		"effective_at":     profile.EffectiveAt,
		"expires_at":       profile.ExpiresAt,
	}
}
