package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"authz"
	"gatewayauth"
)

func handleSystemAdminStepFullCOS(w http.ResponseWriter, r *http.Request) {
	gatewayauth.ApplyGatewayUser(r, cfg.GatewayInternalSecret)
	if !authz.IsPlatformStaff(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "需要系统管理员权限")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, stepFullCOSPublicView())
	case http.MethodPatch, http.MethodPut:
		handleSystemAdminStepFullCOSPatch(w, r)
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleSystemAdminStepFullCOSPatch(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "failed to read body")
		return
	}
	body := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	next := stepFullCOSCfg
	if v, ok := body["backend"].(string); ok {
		next.Backend = strings.TrimSpace(v)
	}
	if v, ok := body["bucket"].(string); ok {
		next.Bucket = strings.TrimSpace(v)
	}
	if v, ok := body["region"].(string); ok {
		next.Region = strings.TrimSpace(v)
	}
	if v, ok := body["pathRule"].(string); ok {
		next.PathRule = strings.TrimSpace(v)
	}
	if v, ok := body["startupLogsPathRule"].(string); ok {
		next.StartupLogsPathRule = strings.TrimSpace(v)
	}
	if v, ok := body["keyPrefix"].(string); ok {
		next.KeyPrefix = strings.TrimSpace(v)
	}
	if v, ok := body["secretId"].(string); ok && strings.TrimSpace(v) != "" {
		next.SecretID = strings.TrimSpace(v)
	}
	if v, ok := body["secretKey"].(string); ok && strings.TrimSpace(v) != "" {
		next.SecretKey = strings.TrimSpace(v)
	}
	if err := applyStepFullCOSConfig(next); err != nil {
		if strings.Contains(err.Error(), "pathRule") || strings.Contains(err.Error(), "PathRule") || strings.Contains(err.Error(), "backend") {
			writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
			return
		}
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	_ = publishDomainEvent(r.Context(), "StepFullCOSConfigUpdated", map[string]interface{}{
		"backend":  stepFullCOSCfg.Backend,
		"bucket":   stepFullCOSCfg.Bucket,
		"region":   stepFullCOSCfg.Region,
		"pathRule": stepFullCOSCfg.PathRule,
	}, "step-full-cos")
	writeJSON(w, http.StatusOK, stepFullCOSPublicView())
}
