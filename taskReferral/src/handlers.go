package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"gatewaycors"
)

// ── HTTP utilities ──

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// requireInternalSecret 校验内部服务调用密钥（X-TaskReferral-Internal-Secret）。
// 未配置密钥（开发/测试）时放行，与 taskBill 等服务的 requireInternalSecret 语义一致。
func requireInternalSecret(r *http.Request) bool {
	sec := strings.TrimSpace(os.Getenv("TASK_REFERRAL_INTERNAL_SECRET"))
	if sec == "" {
		sec = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	if sec == "" {
		return true
	}
	return r.Header.Get("X-TaskReferral-Internal-Secret") == sec
}

// handleInternalExpireReferralCodes 一次性触发推荐码过期回写（OPT-20260816-031）。
// 由 taskEvents referral_code_expiry_scan timer worker 调用，替代进程内 startReferralExpiryLoop。
func handleInternalExpireReferralCodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"status": "method not allowed"})
		return
	}
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "forbidden"})
		return
	}
	expireReferralCodes()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readJSONBody(r *http.Request) (map[string]interface{}, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var body map[string]interface{}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func boolField(body map[string]interface{}, key string) bool {
	v, ok := body[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}

func strField(body map[string]interface{}, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(jsonString(v))
	}
}

func jsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	s := string(b)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeaders)
			w.Header().Set("Access-Control-Expose-Headers", gatewaycors.ExposeHeaders)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Route registration ──

func mountRoutes(mux *http.ServeMux) {
	// User endpoints
	mux.HandleFunc("GET /api/accounts/users/referral-codes/status/", handleReferralCodeStatus)
	mux.HandleFunc("POST /api/accounts/users/referral-codes/apply/", handleReferralCodeApply)
	// 存量已获资格用户补填个人名称（OPT-20260823-048）
	mux.HandleFunc("POST /api/referral/legal-name/", handleReferralLegalNameUpdate)

	// Referral stats — 前端约定路径 /api/{serviceName}/{funcName}/key/value/（taskFE e105bad 迁移）
	mux.HandleFunc("GET /api/referral/stats/user_id/{userId}/", handleReferralStats)
	// 旧路径兼容（OPT-049 Django 迁移遗留；仓库内已无调用方，保留以防外部消费者）
	mux.HandleFunc("GET /api/user/{userId}/profile/referral-stats/", handleReferralStats)
	mux.HandleFunc("GET /api/referral/channels/", handleListReferralChannels)
	mux.HandleFunc("POST /api/referral/channels/", handleCreateReferralChannel)
	mux.HandleFunc("POST /api/referral/channels/code/{code}/disable/", handleDisableReferralChannel)
	mux.HandleFunc("DELETE /api/referral/channels/code/{code}/", handleDeleteReferralChannel)

	// Admin endpoints
	mux.HandleFunc("GET /api/system-admin/referral/applications/", handleAdminReferralApplications)
	mux.HandleFunc("GET /api/system-admin/referral/share-code/lookup/", handleAdminShareCodeLookup)
	mux.HandleFunc("POST /api/system-admin/referral/applications/{id}/approve/", handleAdminReferralApprove)
	mux.HandleFunc("POST /api/system-admin/referral/applications/{id}/reject/", handleAdminReferralReject)
	mux.HandleFunc("POST /api/system-admin/referral/applications/{id}/revoke/", handleAdminReferralRevoke)
	mux.HandleFunc("GET /api/system-admin/referral/applications/{id}/audit/", handleAdminReferralAudit)
	mux.HandleFunc("GET /api/system-admin/referral/policy/", handleAdminReferralPolicy)
	mux.HandleFunc("PUT /api/system-admin/referral/policy/", handleAdminReferralPolicy)
	mux.HandleFunc("POST /api/system-admin/referral/policy/", handleAdminReferralPolicy)

	// Settlement config
	mux.HandleFunc("GET /api/system-admin/referral/config/", handleAdminReferralConfig)
	mux.HandleFunc("POST /api/system-admin/referral/config/", handleAdminReferralConfig)
	mux.HandleFunc("POST /api/system-admin/referral/config/update/", handleAdminReferralConfig)

	// Referral performance (admin view)
	mux.HandleFunc("GET /api/system-admin/users/{uid}/referral-performance/", handleAdminReferralPerformance)

	// Internal: 一次性推荐码过期回写（OPT-20260816-031，taskEvents timer 调用）
	mux.HandleFunc("POST /api/internal/referral/expire-codes/", handleInternalExpireReferralCodes)
	// Internal: 注册后按 accessCode 绑 billing_referral_edge（经 taskBill sync-edge）
	mux.HandleFunc("POST /api/internal/referral/bind-from-code/", handleInternalBindFromCode)
	// Internal: 批量查被推荐人的推荐边（OPT-20260821-031，taskAuth 用户列表推荐人列）
	mux.HandleFunc("POST /api/internal/referral/referrers/lookup/", handleInternalReferrersLookup)
	// Internal: 支付时刻现查推荐人活跃资格（ADR-0033，taskBill Native 分账标识）
	mux.HandleFunc("GET /api/internal/referral/qualification/active/", handleInternalQualificationActive)
	mux.HandleFunc("POST /api/internal/referral/qualification/active/batch/", handleInternalQualificationActiveBatch)
	// Internal: 注册门禁校验分享码（invite_code 策略开启时 access_code 即邀请凭证）
	mux.HandleFunc("POST /api/internal/referral/share-code/validate/", handleInternalShareCodeValidate)

	// Health
	mux.HandleFunc("GET /api/health/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "taskReferral"})
	})
}
