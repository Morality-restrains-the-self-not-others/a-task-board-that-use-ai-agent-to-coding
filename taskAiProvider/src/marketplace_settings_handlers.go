package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// isPlatformStaffRequest 判定网关注入的平台角色（super_admin / employee）。
// 与 shareLib/authz.IsPlatformStaff 语义一致，避免为本服务引入 authz 依赖。
func isPlatformStaffRequest(r *http.Request) bool {
	for _, role := range strings.Split(r.Header.Get("X-User-Roles"), ",") {
		role = strings.TrimSpace(role)
		if role == "super_admin" || role == "employee" {
			return true
		}
	}
	return false
}

func (a *App) allowMarketplaceAdmin(w http.ResponseWriter, r *http.Request) bool {
	if isPlatformStaffRequest(r) {
		return true
	}
	_, ok := a.requireStaff(w, r)
	return ok
}

func marketplaceSettingsJSON(reviewEnabled bool) map[string]any {
	return map[string]any{
		"vendor_application_review_enabled": reviewEnabled,
	}
}

// handleMarketplaceSettings GET — 公开读审核开关（失败时默认开启）。
func (a *App) handleMarketplaceSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	enabled, err := a.DB.GetMarketplaceSettings()
	if err != nil {
		logWarn(r.Context(), "marketplace-settings load failed: %v", err)
		enabled = true
	}
	writeJSON(w, http.StatusOK, marketplaceSettingsJSON(enabled))
}

// handleAdminMarketplaceSettings GET|PATCH — 平台运营或 ai-provider staff。
func (a *App) handleAdminMarketplaceSettings(w http.ResponseWriter, r *http.Request) {
	if !a.allowMarketplaceAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		enabled, err := a.DB.GetMarketplaceSettings()
		if err != nil {
			logWarn(r.Context(), "admin-marketplace-settings load failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "读取设置失败"})
			return
		}
		writeJSON(w, http.StatusOK, marketplaceSettingsJSON(enabled))
	case http.MethodPatch, http.MethodPut:
		var body struct {
			VendorApplicationReviewEnabled *bool `json:"vendor_application_review_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.VendorApplicationReviewEnabled == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "需要 vendor_application_review_enabled 布尔字段"})
			return
		}
		enabled := *body.VendorApplicationReviewEnabled
		if err := a.DB.SetVendorApplicationReviewEnabled(enabled); err != nil {
			logWarn(r.Context(), "admin-marketplace-settings save failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "保存设置失败"})
			return
		}
		logInfo("event=MarketplaceSettingsUpdated vendor_application_review_enabled=%v", enabled)
		writeJSON(w, http.StatusOK, marketplaceSettingsJSON(enabled))
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
	}
}
