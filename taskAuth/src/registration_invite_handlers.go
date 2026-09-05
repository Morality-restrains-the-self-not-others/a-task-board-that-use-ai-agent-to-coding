package main

import (
	"authz"
	"context"
	"log"
	"net/http"
	"strconv"
)

func requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := resolveUserIDFromRequest(r)
	if !ok || userID == "" {
		writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
		return "", false
	}
	return userID, true
}

func requireSuperuser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return "", false
	}
	// v63 RBAC: 平台全权判定迁移至 authz 权限码（网关注入 X-User-Roles）
	if authz.HasPlatformPerm(r, authz.PermPlatformManage) {
		return userID, true
	}
	// 过渡兜底: 直接调用（无网关注入头）时回退 is_superuser 查表；硬切换完成后删除
	_, isSuperuser, _, _, err := loadUserAuthFlags(userID)
	if err != nil {
		log.Printf("[taskAuth] superuser check failed: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return "", false
	}
	if !isSuperuser {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return "", false
	}
	return userID, true
}

func publicRegistrationInvitePolicyPayload() (map[string]interface{}, error) {
	policy, err := getRegistrationInvitePolicy()
	if err != nil {
		return nil, err
	}
	remaining, err := remainingToday(policy)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"enabled":          policy.Enabled,
		"daily_quota":      policy.DailyQuota,
		"remaining_today":  remaining,
	}, nil
}

func handlePublicRegistrationInvitePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	payload, err := publicRegistrationInvitePolicyPayload()
	if err != nil {
		log.Printf("[taskAuth] public registration invite policy: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func handleSystemAdminRegistrationInvitePolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		adminUserID, ok := requireSuperuser(w, r)
		if !ok {
			return
		}
		_ = adminUserID
		payload, err := publicRegistrationInvitePolicyPayload()
		if err != nil {
			log.Printf("[taskAuth] admin registration invite policy get: %v", err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		policy, err := getRegistrationInvitePolicy()
		if err != nil {
			log.Printf("[taskAuth] admin registration invite policy get detail: %v", err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		payload["updated_at"] = policy.UpdatedAt
		payload["updated_by"] = policy.UpdatedBy
		writeJSON(w, http.StatusOK, payload)
	case http.MethodPut:
		adminUserID, ok := requireSuperuser(w, r)
		if !ok {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		enabled := false
		if v, ok := body["enabled"]; ok && v != nil {
			switch t := v.(type) {
			case bool:
				enabled = t
			case float64:
				enabled = t != 0
			default:
				enabled = strField(body, "enabled") == "true" || strField(body, "enabled") == "1"
			}
		}
		dailyQuota := 0
		if v, ok := body["daily_quota"]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				dailyQuota = int(t)
			case int:
				dailyQuota = t
			default:
				if parsed, err := strconv.Atoi(strField(body, "daily_quota")); err == nil {
					dailyQuota = parsed
				}
			}
		}
		if dailyQuota < 0 {
			dailyQuota = 0
		}
		policy, err := updateRegistrationInvitePolicy(enabled, dailyQuota, adminUserID)
		if err != nil {
			log.Printf("[taskAuth] admin registration invite policy update: %v", err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		publishInviteEvent(r.Context(), "REGISTRATION_INVITE_POLICY_UPDATED", map[string]interface{}{
			"enabled":      policy.Enabled,
			"daily_quota":  policy.DailyQuota,
			"updated_by":   adminUserID,
			"updated_at":   policy.UpdatedAt,
		}, registrationInvitePolicyKey)
		remaining, err := remainingToday(policy)
		if err != nil {
			log.Printf("[taskAuth] admin registration invite policy remaining: %v", err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled":          policy.Enabled,
			"daily_quota":      policy.DailyQuota,
			"remaining_today":  remaining,
			"updated_at":       policy.UpdatedAt,
			"updated_by":       policy.UpdatedBy,
		})
	default:
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleSystemAdminRegistrationInviteRelations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := requireSuperuser(w, r); !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	items, total, err := listRegistrationInviteRelations(page, pageSize)
	if err != nil {
		log.Printf("[taskAuth] registration invite relations list: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if items == nil {
		items = []registrationInviteRelation{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results":   items,
		"total":     total,
		"page":      max(page, 1),
		"page_size": pageSizeOrDefault(pageSize),
	})
}

func pageSizeOrDefault(pageSize int) int {
	if pageSize < 1 {
		return 20
	}
	if pageSize > 100 {
		return 100
	}
	return pageSize
}

func handleApplyRegistrationInviteCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	row, err := applyRegistrationInviteCode(r.Context(), userID)
	if err != nil {
		code := inviteErrorCode(err)
		status := http.StatusBadRequest
		if code == "internal_error" {
			status = http.StatusInternalServerError
		}
		writeError(w, r, status, code)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func handleListRegistrationInviteCodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	items, err := listRegistrationInviteCodesForIssuer(userID)
	if err != nil {
		log.Printf("[taskAuth] registration invite codes list: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if items == nil {
		items = []registrationInviteCodeRow{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": items})
}

func writeInviteRegisterError(w http.ResponseWriter, r *http.Request, err error) {
	writeError(w, r, http.StatusBadRequest, inviteErrorCode(err))
}

func registerInviteRollback(ctx context.Context, userID string, body map[string]interface{}, redeemErr error) {
	if delErr := deleteUser(userID); delErr != nil {
		log.Printf("[taskAuth] registration invite rollback delete user %s failed: %v", userID, delErr)
	}
	log.Printf("[taskAuth] registration invite redeem failed for user %s: %v", userID, redeemErr)
}
