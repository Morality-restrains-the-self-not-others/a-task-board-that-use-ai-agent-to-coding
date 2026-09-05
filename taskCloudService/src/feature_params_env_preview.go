package main

import (
	"net/http"
	"strings"
)

// handleFeatureParamsEnvPreview serves GET compute/feature-params-env-preview/
// (personal config only; IDOR: config.user_id must equal auth user).
func handleFeatureParamsEnvPreview(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	_ = tenantID
	_ = workspaceID
	_ = taskID
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := strings.TrimSpace(getAuthUser(r))
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "未认证"})
		return
	}
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source != "personal" {
		logWarn("feature_params_env_preview: blocked non-personal source="+source+" user="+userID, r.Header.Get("X-Trace-Id"))
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "仅支持预览个人配置"})
		return
	}
	personalConfigID := strings.TrimSpace(r.URL.Query().Get("personal_config_id"))
	if personalConfigID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "缺少 personal_config_id 参数"})
		return
	}
	cfg, err := loadPersonalFeatureParams(personalConfigID, userID)
	if err != nil {
		logWarn("feature_params_env_preview: load error: "+err.Error(), r.Header.Get("X-Trace-Id"))
		writeJSON(w, http.StatusBadGateway, map[string]string{"message": "加载配置失败"})
		return
	}
	if cfg == nil {
		// Distinguishing not-found vs IDOR: loadPersonal filters by user_id.
		// Probe without user filter for 403 vs 404 when local store exposes it.
		if ownerID, found, probeErr := peekPersonalFeatureParamsOwner(personalConfigID); probeErr == nil && found {
			if ownerID != userID {
				logWarn("feature_params_env_preview: IDOR attempt user="+userID+" target="+personalConfigID, r.Header.Get("X-Trace-Id"))
				writeJSON(w, http.StatusForbidden, map[string]string{"message": "无权访问该配置"})
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "配置不存在"})
		return
	}
	configName := personalConfigDisplayName(cfg.DisplayName)
	cfg.DisplayName = configName
	env := serializeFeatureParamsEnv(cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"source":      "personal",
		"config_name": configName,
		"env":         env,
	})
}

func personalConfigDisplayName(displayName string) string {
	const prefix = "个人配置:"
	if strings.HasPrefix(displayName, prefix) {
		return strings.TrimPrefix(displayName, prefix)
	}
	return displayName
}

// peekPersonalFeatureParamsOwner returns owner user_id if the config row exists.
// Implemented in local store; HTTP-era stub returns not found.
func peekPersonalFeatureParamsOwner(configID string) (ownerID string, found bool, err error) {
	return peekPersonalFeatureParamsOwnerImpl(configID)
}
