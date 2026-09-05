package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"taskGitOauth/infrastructure"
)

func (a *App) handleAccessForUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider := providerFromPath(r.URL.Path)
	if resolved := strings.TrimSpace(r.Header.Get(headerResolvedProviderKey)); resolved != "" {
		// OPT-20260822-036: gitsite 路径已由 ResolveProviderByGitsite 解析出 provider_key，
		// 以 site 解析结果为准（SSOT），body provider_key 仅作一致性校验。
		provider = resolved
	}
	body, err := readJSONNumbered(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	providerKey := normalizeProviderKey(fmt.Sprint(body["provider_key"]), provider)
	uid := strings.TrimSpace(fmt.Sprint(body["user_id"]))
	if uid == "" || uid == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "bad user_id"})
		return
	}
	ghUID := ""
	if v := body["github_user_id"]; v != nil {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			ghUID = s
		}
	}

	row, err := a.DB.FindActiveCredential(providerKey, uid, ghUID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}
	if row == nil && strings.Contains(providerKey, ":") {
		prefix := strings.SplitN(providerKey, ":", 2)[0] + ":"
		candidate, err := a.DB.FindActiveCredentialPrefix(prefix, uid, ghUID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"detail": truncate(err.Error(), 500)})
			return
		}
		if candidate != nil && infrastructure.AllowGitCredentialPrefixFallback(a.Cfg, providerKey, candidate.Provider) {
			row = candidate
		} else if candidate != nil {
			logWarn("access-for-user skip prefix fallback requested=%s stored=%s uid=%s trace_id=%s",
				providerKey, candidate.Provider, uid, requestTraceID(r))
		}
	}
	if row == nil || strings.TrimSpace(row.RefreshTokenCipher) == "" {
		// OPT-20260821-037: 缺凭据 404 必须带可读 errDetail，否则调用方只能拼出「未能换取 Git OAuth 凭据」。
		writeJSON(w, http.StatusNotFound, map[string]any{
			"detail":    "not_found",
			"errDetail": "未找到该用户的 Git OAuth 授权，请重新完成授权",
		})
		return
	}
	refreshProviderKey := strings.TrimSpace(strings.ToLower(row.Provider))
	if refreshProviderKey == "" {
		refreshProviderKey = providerKey
	}
	cacheID := cacheKeyID(refreshProviderKey, row.RemoteUserID)
	wasCached := a.Cache.Get(cacheID, uid) != ""

	access, err := a.issueAccessTokenFromCredential(uid, row)
	if err != nil {
		if errors.Is(err, errGitOAuthNotConnected) {
			// OPT-20260821-037: 区分「无凭据」与「已失效/换票失败」，调用方据此展示可读文案。
			writeJSON(w, http.StatusNotFound, map[string]any{
				"detail":    "not_connected",
				"errDetail": "该用户的 Git OAuth 授权已失效，请重新完成授权",
			})
			return
		}
		if strings.Contains(err.Error(), "decrypt_failed") {
			logWarn("gitOauth 解密 refresh 失败 task2app_user_id=%s trace_id=%s", uid, requestTraceID(r))
			writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "decrypt_failed"})
			return
		}
		logWarn("access-for-user refresh 失败 uid=%s trace_id=%s: %v", uid, requestTraceID(r), err)
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}

	if audit, ok := body["audit"].(map[string]any); ok {
		if strings.TrimSpace(fmt.Sprint(audit["action"])) != "" {
			if err := a.writeTokenUseAudit(uid, refreshProviderKey, audit, access, requestTraceID(r)); err != nil {
				logWarn("access-for-user audit write failed uid=%s provider=%s trace_id=%s: %v", uid, refreshProviderKey, requestTraceID(r), err)
			}
		}
	}
	resp := map[string]any{
		"access_token":   access,
		"github_user_id": row.RemoteUserID,
	}
	if wasCached {
		resp["cached"] = true
	}
	if ttl := a.Cache.TTLSeconds(cacheID, uid); ttl > 0 {
		resp["expires_in"] = ttl
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) writeTokenUseAudit(userID string, provider string, audit map[string]any, accessToken string, traceID string) error {
	act := truncate(strings.TrimSpace(fmt.Sprint(audit["action"])), 64)
	if act == "" || act == "<nil>" {
		return nil
	}
	fp := accessTokenFingerprint(accessToken)
	if fp == "" {
		fp = extractFingerprint(audit)
	}
	detail, _ := audit["detail"].(map[string]any)
	site := a.accessAuditSite(provider)
	if site == "" {
		logWarn("access audit skip: site unresolved provider_key=%s trace_id=%s", provider, traceID)
		return fmt.Errorf("access audit site unresolved")
	}
	if err := a.DB.InsertAccessAudit(
		site, userID,
		optionalInt64(audit["company_id"]),
		optionalInt64(audit["workspace_id"]),
		act, fp, sanitizeAuditDetail(detail),
	); err != nil {
		logWarn("access audit insert failed provider_key=%s trace_id=%s: %v", provider, traceID, err)
		return err
	}
	return nil
}

func extractFingerprint(audit map[string]any) string {
	if direct := strings.TrimSpace(strings.ToLower(fmt.Sprint(audit["access_token_fingerprint"]))); direct != "" && direct != "<nil>" {
		if len(direct) > 64 {
			direct = direct[:64]
		}
		return direct
	}
	if detail, ok := audit["detail"].(map[string]any); ok {
		if nested := strings.TrimSpace(strings.ToLower(fmt.Sprint(detail["access_token_fingerprint"]))); nested != "" && nested != "<nil>" {
			if len(nested) > 64 {
				nested = nested[:64]
			}
			return nested
		}
		if at := strings.TrimSpace(fmt.Sprint(detail["access_token"])); at != "" && at != "<nil>" {
			return accessTokenFingerprint(at)
		}
	}
	if at := strings.TrimSpace(fmt.Sprint(audit["access_token"])); at != "" && at != "<nil>" {
		return accessTokenFingerprint(at)
	}
	return ""
}
