package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

func authValidateSession(ctx context.Context, r *http.Request, sc scope) (validateSessionResult, int, []byte) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskAuthURL), "/")
	if base == "" {
		return validateSessionResult{}, http.StatusBadGateway, []byte(`{"detail":"task auth url not configured"}`)
	}
	payload := map[string]any{
		"cookie":        r.Header.Get("Cookie"),
		"authorization": r.Header.Get("Authorization"),
		"tenant_id":     sc.TenantID,
		"workspace_id":  sc.WorkspaceID,
		"task_id":       sc.TaskID,
		"path":          r.URL.Path,
		"method":        r.Method,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return validateSessionResult{}, http.StatusInternalServerError, []byte(`{"detail":"failed to encode auth payload"}`)
	}
	url := base + "/api/internal/container-gateway/validate-session/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return validateSessionResult{}, http.StatusBadGateway, []byte(`{"detail":"auth unreachable"}`)
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(cfg.TaskAuthInternalSecret); secret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", secret)
	}
	start := time.Now()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		tracelog.LogForwardStage(ctx, "auth_validate", map[string]any{
			"auth_status": 502,
			"duration_ms": duration,
			"detail":      "auth unreachable",
		})
		return validateSessionResult{}, http.StatusBadGateway, []byte(`{"detail":"auth unreachable"}`)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tracelog.LogForwardStage(ctx, "auth_validate", map[string]any{
			"auth_status": 502,
			"duration_ms": duration,
			"detail":      "failed to read auth response",
		})
		return validateSessionResult{}, http.StatusBadGateway, []byte(`{"detail":"failed to read auth response"}`)
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	tracelog.LogForwardStage(ctx, "auth_validate", map[string]any{
		"auth_status": resp.StatusCode,
		"duration_ms": duration,
	})
	if resp.StatusCode != http.StatusOK {
		return validateSessionResult{}, resp.StatusCode, body
	}
	var parsed struct {
		UserID     string `json:"user_id"`
		AuthMethod string `json:"auth_method"`
		ScopeOK    bool   `json:"scope_ok"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return validateSessionResult{}, http.StatusBadGateway, []byte(`{"detail":"invalid auth validate-session response"}`)
	}
	return validateSessionResult{
		UserID:     parsed.UserID,
		AuthMethod: parsed.AuthMethod,
		ScopeOK:    parsed.ScopeOK,
	}, http.StatusOK, body
}

func cloudGetJSON(ctx context.Context, pathWithQuery string, stage string) (int, []byte) {
	base := strings.TrimRight(strings.TrimSpace(cfg.CloudServiceURL), "/")
	if base == "" {
		return http.StatusBadGateway, []byte(`{"detail":"task cloud service url not configured"}`)
	}
	url := base + pathWithQuery
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return http.StatusBadGateway, []byte(`{"detail":"cloud unreachable"}`)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(cfg.CloudInternalSecret); secret != "" {
		req.Header.Set("X-Internal-Secret", secret)
	}
	start := time.Now()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		tracelog.LogForwardStage(ctx, stage, map[string]any{
			"cloud_status": 502,
			"duration_ms":  duration,
			"detail":       "cloud unreachable",
		})
		return http.StatusBadGateway, []byte(fmt.Sprintf(`{"detail":"%s unreachable"}`, stage))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tracelog.LogForwardStage(ctx, stage, map[string]any{
			"cloud_status": 502,
			"duration_ms":  duration,
			"detail":       "failed to read cloud response",
		})
		return http.StatusBadGateway, []byte(`{"detail":"failed to read cloud response"}`)
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	tracelog.LogForwardStage(ctx, stage, map[string]any{
		"cloud_status": resp.StatusCode,
		"duration_ms":  duration,
	})
	return resp.StatusCode, body
}

// cloudPostJSON posts JSON to taskCloudService and returns status + body (Phase C git-push).
func cloudPostJSON(ctx context.Context, path string, payload map[string]any, stage string) (int, []byte) {
	base := strings.TrimRight(strings.TrimSpace(cfg.CloudServiceURL), "/")
	if base == "" {
		return http.StatusBadGateway, []byte(`{"detail":"task cloud service url not configured"}`)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return http.StatusInternalServerError, []byte(`{"detail":"failed to encode cloud payload"}`)
	}
	url := base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return http.StatusBadGateway, []byte(`{"detail":"cloud unreachable"}`)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(cfg.CloudInternalSecret); secret != "" {
		req.Header.Set("X-Internal-Secret", secret)
	}
	start := time.Now()
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		tracelog.LogForwardStage(ctx, stage, map[string]any{
			"cloud_status": 502,
			"duration_ms":  duration,
			"detail":       "cloud unreachable",
		})
		return http.StatusBadGateway, []byte(`{"detail":"cloud unreachable"}`)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tracelog.LogForwardStage(ctx, stage, map[string]any{
			"cloud_status": 502,
			"duration_ms":  duration,
			"detail":       "failed to read cloud response",
		})
		return http.StatusBadGateway, []byte(`{"detail":"failed to read cloud response"}`)
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	tracelog.LogForwardStage(ctx, stage, map[string]any{
		"cloud_status": resp.StatusCode,
		"duration_ms":  duration,
	})
	return resp.StatusCode, body
}

func cloudAssertTenantMember(ctx context.Context, tenantID, userID string) (int, []byte) {
	q := "/api/internal/tenant-member/?tenant_id=" + url.QueryEscape(tenantID) + "&user_id=" + url.QueryEscape(userID)
	status, body := cloudGetJSON(ctx, q, "cloud_member")
	if status == http.StatusNotFound {
		return http.StatusForbidden, []byte(`{"detail":"forbidden scope"}`)
	}
	return status, body
}

func cloudLookupConfig(ctx context.Context, sc scope, commentID string) (int, []byte) {
	q := "/api/internal/cloud-server-config/lookup?tenant_id=" + url.QueryEscape(sc.TenantID) +
		"&workspace_id=" + url.QueryEscape(sc.WorkspaceID) +
		"&task_id=" + url.QueryEscape(sc.TaskID)
	if cid := strings.TrimSpace(commentID); cid != "" {
		q += "&comment_id=" + url.QueryEscape(cid)
	}
	status, body := cloudGetJSON(ctx, q, "cloud_lookup")
	if status == http.StatusNotFound {
		return http.StatusForbidden, []byte(`{"detail":"容器配置不存在或尚未就绪"}`)
	}
	return status, body
}

func cloudResolveTarget(ctx context.Context, sc scope, containerPageURL, commentID string) (containerTarget, int, []byte) {
	q := "/api/internal/cloud-server-config/container-target/?tenant_id=" + url.QueryEscape(sc.TenantID) +
		"&workspace_id=" + url.QueryEscape(sc.WorkspaceID) +
		"&task_id=" + url.QueryEscape(sc.TaskID)
	if strings.TrimSpace(containerPageURL) != "" {
		q += "&container_page_url=" + url.QueryEscape(containerPageURL)
	}
	if cid := strings.TrimSpace(commentID); cid != "" {
		q += "&comment_id=" + url.QueryEscape(cid)
	}
	status, body := cloudGetJSON(ctx, q, "cloud_resolve")
	if status != http.StatusOK {
		return containerTarget{}, status, body
	}
	var parsed struct {
		BaseURL     string `json:"base_url"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return containerTarget{}, http.StatusBadGateway, []byte(`{"detail":"invalid container-target response"}`)
	}
	return containerTarget{BaseURL: parsed.BaseURL, AccessToken: parsed.AccessToken}, http.StatusOK, body
}

func relayToTraePath(path string) bool {
	return strings.Contains(path, "/relay-to-trae/")
}

// forwardedTrustedUserID reads the browser user Cloud already resolved (APISIX
// X-Auth-User-Id / X-User-Id). Internal-secret bypass must not invent
// "internal_gateway" when a real user is present — git identity / OAuth lookup
// is keyed by that user (404 "Git 身份不存在或不属于当前租户" on ztree push).
func forwardedTrustedUserID(r *http.Request) string {
	for _, key := range []string{"X-Auth-User-Id", "X-User-Id"} {
		v := strings.TrimSpace(r.Header.Get(key))
		if v == "" || v == "internal" || v == "internal_gateway" {
			continue
		}
		return v
	}
	return ""
}

// authorizeContainerRequest runs zero-Django hot-path auth: identity → membership → (optional) config exists.
func authorizeContainerRequest(ctx context.Context, r *http.Request, sc scope) (validateSessionResult, int, []byte) {
	// Internal-secret bypass: allow trusted callers (e.g. taskCloudService proxy/relay)
	// to skip Cookie/Token IP-bound validate-session. Cloud does not forward Cookie,
	// but DOES forward X-Auth-User-Id; use that as session.UserID so git-push prepare
	// can look up task_git_identities / gitOauth for the browser user.
	if gwSecret := strings.TrimSpace(cfg.InternalSecret); gwSecret != "" {
		if strings.TrimSpace(r.Header.Get("X-TaskContainerGateway-Internal-Secret")) == gwSecret {
			userID := forwardedTrustedUserID(r)
			if userID == "" {
				userID = "internal_gateway"
			}
			tracelog.LogForwardStage(ctx, "auth_internal_bypass", map[string]any{
				"auth_status": 200,
				"auth_method": "internal_secret",
				"user_id":     userID,
			})
			return validateSessionResult{
				UserID:     userID,
				AuthMethod: "internal_secret",
				ScopeOK:    true,
			}, http.StatusOK, nil
		}
	}

	session, status, body := authValidateSession(ctx, r, sc)
	if status != http.StatusOK {
		return validateSessionResult{}, status, body
	}
	if strings.TrimSpace(sc.TenantID) == "" {
		return validateSessionResult{}, http.StatusForbidden, []byte(`{"detail":"forbidden scope"}`)
	}
	mStatus, mBody := cloudAssertTenantMember(ctx, sc.TenantID, session.UserID)
	if mStatus != http.StatusOK {
		return validateSessionResult{}, mStatus, mBody
	}
	// relay may create CloudServerConfig; skip lookup like Phase A relay exemption.
	if !relayToTraePath(r.URL.Path) && strings.TrimSpace(sc.TaskID) != "" {
		commentID := lookupCommentID(r, sc)
		lStatus, lBody := cloudLookupConfig(ctx, sc, commentID)
		if lStatus != http.StatusOK {
			return validateSessionResult{}, lStatus, lBody
		}
	}
	session.ScopeOK = true
	return session, http.StatusOK, body
}
