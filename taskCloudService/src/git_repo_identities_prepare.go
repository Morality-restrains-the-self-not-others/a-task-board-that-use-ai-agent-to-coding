package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

type userCompanyGitIdentityRow struct {
	GitUserName  string
	GitUserEmail string
}

// fetchUserCompanyGitIdentityFn is the active lookup implementation. Tests may
// replace it to avoid HTTP calls to taskTaskService.
var fetchUserCompanyGitIdentityFn = realFetchUserCompanyGitIdentity

// fetchUserCompanyGitIdentity loads git author fields from taskTaskService's
// task_git_identities via internal API (OPT-20260820-016, single-table single-service).
// Migrated from saas.accounts_user_company_git_identity (OPT-052: Django retirement).
func fetchUserCompanyGitIdentity(ctx context.Context, identityID, userID, companyID string) (*userCompanyGitIdentityRow, error) {
	return fetchUserCompanyGitIdentityFn(ctx, identityID, userID, companyID)
}

// realFetchUserCompanyGitIdentity is the production implementation that calls
// taskTaskService POST /api/internal/git-identities/lookup/. It replaces the
// previous cross-database SQL into the taskTaskService-owned git identity
// table, which broke when the table was renamed (MySQL 1146). Ownership
// (user_id/company_id) is validated on the owner service side.
func realFetchUserCompanyGitIdentity(ctx context.Context, identityID, userID, companyID string) (*userCompanyGitIdentityRow, error) {
	identityID = strings.TrimSpace(identityID)
	userID = strings.TrimSpace(userID)
	companyID = strings.TrimSpace(companyID)
	if identityID == "" || userID == "" || companyID == "" {
		return nil, nil
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if base == "" {
		return nil, fmt.Errorf("task service url not configured")
	}
	payload, err := json.Marshal(map[string]string{
		"identity_id": identityID,
		"user_id":     userID,
		"company_id":  companyID,
	})
	if err != nil {
		return nil, err
	}
	// OPT-20260821-015: 透传入站 X-Trace-Id，使 Cloud→Task lookup 被同一把钥匙跨服务重建。
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/internal/git-identities/lookup/", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	req.Header.Set("Content-Type", "application/json")
	// taskTaskService lookup requires isInternalCall (X-Auth-User-Id=internal)
	// or X-Internal-Secret. Production shared.internalSecret is often empty, so
	// the internal user header is the service-to-service contract (same as
	// git_push_auth_context / feature_params_resolve). Missing it yields
	// 403 {"error":"internal only"} and layer-git-push prepare 502.
	req.Header.Set("X-Auth-User-Id", "internal")
	if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("git identity lookup: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("git identity lookup status=%d body=%s", resp.StatusCode, truncateForLog(string(raw), 200))
	}
	var out struct {
		Found        bool   `json:"found"`
		GitUserName  string `json:"git_user_name"`
		GitUserEmail string `json:"git_user_email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.Found {
		return nil, nil
	}
	return &userCompanyGitIdentityRow{
		GitUserName:  strings.TrimSpace(out.GitUserName),
		GitUserEmail: strings.TrimSpace(out.GitUserEmail),
	}, nil
}

// lookupUserCompanyGitIdentity checks taskTaskService task_git_identities ownership.
func lookupUserCompanyGitIdentity(ctx context.Context, identityID, userID, companyID string) (bool, error) {
	row, err := fetchUserCompanyGitIdentity(ctx, identityID, userID, companyID)
	if err != nil {
		return false, err
	}
	return row != nil, nil
}

// handleInternalLayerGitRepoIdentitiesPrepare implements
// POST /api/internal/layer-git-repo-identities/prepare
//
// Browser sends repos:[{repo_url, identity_id}]; this resolves to the container
// body repos:[{repo_match_key, user_name, user_email}] after ownership checks.
func handleInternalLayerGitRepoIdentitiesPrepare(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "detail": "forbidden"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "detail": "method not allowed"})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "detail": "invalid json"})
		return
	}

	tenantID := strField(body, "tenant_id")
	userID := strField(body, "user_id")
	if tenantID == "" || userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":     false,
			"detail": "tenant_id 与 user_id 必填",
		})
		return
	}

	rawRepos, ok := body["repos"].([]any)
	if !ok || len(rawRepos) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":     false,
			"detail": "repos 须为非空数组，每项含 repo_url 与 identity_id",
		})
		return
	}

	upstream := make([]map[string]string, 0, len(rawRepos))
	for _, item := range rawRepos {
		row, ok := item.(map[string]any)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":     false,
				"detail": "repos 每项须为对象",
			})
			return
		}
		repoURL := strField(row, "repo_url")
		identityID := strField(row, "identity_id")
		if repoURL == "" || identityID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":     false,
				"detail": "repos 每项须含非空的 repo_url 与 identity_id",
			})
			return
		}
		identity, err := fetchUserCompanyGitIdentity(r.Context(), identityID, userID, tenantID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok":     false,
				"detail": fmt.Sprintf("saas identity lookup failed: %v", err),
			})
			return
		}
		if identity == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"ok":     false,
				"detail": fmt.Sprintf("Git 身份不存在或不属于当前公司: %s", identityID),
			})
			return
		}
		matchKey := repoMatchKeyFromURL(repoURL)
		if matchKey == "" || identity.GitUserName == "" || identity.GitUserEmail == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok": false,
				"detail": fmt.Sprintf(
					"身份 %s 解析后无效（repo_match_key/user_name/user_email 须非空）",
					identityID,
				),
			})
			return
		}
		upstream = append(upstream, map[string]string{
			"repo_match_key": matchKey,
			"user_name":      identity.GitUserName,
			"user_email":     identity.GitUserEmail,
		})
	}

	if len(upstream) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":     false,
			"detail": "无有效 repos 条目",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"repos": upstream,
	})
}
