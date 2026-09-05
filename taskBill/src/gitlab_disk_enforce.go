package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tracelog"
)

func resolveGitlabAdminToken() string {
	if t := strings.TrimSpace(cfg.GitlabAdminPrivateToken); t != "" {
		return t
	}
	if t := strings.TrimSpace(os.Getenv("GITLAB_ADMIN_PRIVATE_TOKEN")); t != "" {
		return t
	}
	root, err := findMonorepoRoot()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(root, "gitService", "gitlab_home", ".taskbill_admin_pat"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func resolveGitlabAPIBase() string {
	if b := strings.TrimSpace(cfg.GitlabAPIBase); b != "" {
		return strings.TrimRight(b, "/")
	}
	if b := strings.TrimSpace(os.Getenv("GITLAB_API_BASE")); b != "" {
		return strings.TrimRight(b, "/")
	}
	return "http://127.0.0.1:8012"
}

func gitlabAdminJSONWithToken(ctx context.Context, method, apiURL, token string, body interface{}) (int, []byte, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, nil, fmt.Errorf("missing GitLab admin token")
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, apiURL, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := tracelog.DirectClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw, nil
}

func gitlabAdminJSON(ctx context.Context, method, apiURL string, body interface{}) (int, []byte, error) {
	return gitlabAdminJSONWithToken(ctx, method, apiURL, resolveGitlabAdminToken(), body)
}

func ensureTenantGitlabGroupForRegion(ctx context.Context, tenantID int64, limitBytes int64, region *GitlabRegion) error {
	if region == nil {
		return fmt.Errorf("region required")
	}
	if gitlabRegionIsPendingNode(region) {
		return errGitlabRegionInfraPending
	}
	token := strings.TrimSpace(region.AdminPrivateToken)
	if token == "" {
		return fmt.Errorf("region %s missing admin token", region.Slug)
	}
	base := strings.TrimRight(strings.TrimSpace(region.GitlabAPIBase), "/")
	if base == "" {
		return fmt.Errorf("region %s missing gitlab_api_base", region.Slug)
	}
	return ensureTenantGitlabGroupWith(ctx, tenantID, limitBytes, base, token)
}

func ensureTenantGitlabGroup(ctx context.Context, tenantID int64, limitBytes int64) error {
	return ensureTenantGitlabGroupWith(ctx, tenantID, limitBytes, resolveGitlabAPIBase(), resolveGitlabAdminToken())
}

func ensureTenantGitlabGroupWith(ctx context.Context, tenantID int64, limitBytes int64, base, token string) error {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return fmt.Errorf("missing GitLab API base")
	}
	path := fmt.Sprintf("tenant-%s", formatID(tenantID))
	name := path
	// Try update existing by path
	getURL := fmt.Sprintf("%s/api/v4/groups/%s", base, url.PathEscape(path))
	code, raw, err := gitlabAdminJSONWithToken(ctx, http.MethodGet, getURL, token, nil)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"repository_size_limit": limitBytes,
	}
	if code == http.StatusOK {
		var g struct {
			ID int64 `json:"id"`
		}
		_ = json.Unmarshal(raw, &g)
		putURL := fmt.Sprintf("%s/api/v4/groups/%d", base, g.ID)
		code, raw, err = gitlabAdminJSONWithToken(ctx, http.MethodPut, putURL, token, payload)
		if err != nil {
			return err
		}
		if code >= 300 {
			return fmt.Errorf("update group status=%d body=%s", code, truncateForErr(string(raw), 200))
		}
		return nil
	}
	create := map[string]interface{}{
		"name":                  name,
		"path":                  path,
		"visibility":            "private",
		"repository_size_limit": limitBytes,
	}
	code, raw, err = gitlabAdminJSONWithToken(ctx, http.MethodPost, base+"/api/v4/groups", token, create)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("create group status=%d body=%s", code, truncateForErr(string(raw), 200))
	}
	return nil
}

func gitlabProjectPathFromRepoURL(repoURL string) (string, error) {
	parts, err := parseGitlabRepoPath(repoURL)
	if err != nil {
		return "", err
	}
	return parts, nil
}

func parseGitlabRepoPath(repoURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(repoURL))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid repo url")
	}
	p := strings.Trim(u.Path, "/")
	p = strings.TrimSuffix(p, ".git")
	if p == "" || !strings.Contains(p, "/") {
		return "", fmt.Errorf("invalid project path")
	}
	return p, nil
}

func applyProjectRepositorySizeLimit(ctx context.Context, projectPath string, limitBytes int64) error {
	base := resolveGitlabAPIBase()
	encoded := url.PathEscape(projectPath)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s", base, encoded)
	code, raw, err := gitlabAdminJSON(ctx, http.MethodPut, apiURL, map[string]interface{}{
		"repository_size_limit": limitBytes,
	})
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("project limit status=%d path=%s body=%s", code, projectPath, truncateForErr(string(raw), 200))
	}
	return nil
}

func enforceGitlabDiskQuota(ctx context.Context, item gitlabDiskSyncItem) error {
	tid, err := parseIDField(item.TenantID)
	if err != nil {
		return err
	}
	if err := ensureTenantGitlabGroup(ctx, tid, item.DiskLimitBytes); err != nil {
		slog.WarnContext(ctx, "gitlab_tenant_group_limit_failed",
			"level", "warn",
			"tenant_id", item.TenantID,
			"error", err.Error(),
		)
		// continue to project limits
	}
	var firstErr error
	for _, repoURL := range item.RepoURLs {
		path, err := gitlabProjectPathFromRepoURL(repoURL)
		if err != nil {
			continue
		}
		// 个人命名空间/非 tenant-{id} 前缀仓校验归属租户（OPT-20260824-080，与流量闸门同根因）：
		// 项目归租户逻辑只认 tenant-{id} 前缀，个人命名空间仓可能被任务侧按成员关系误归到本项目，
		// 若仓所有者经「gitlab 用户名 → 凭据 → 成员 → 区域资源」链解析为其他租户，则跳过，
		// 避免把本项目租户的限额强加到别的租户个人仓。解析失败保持 fail-safe（沿用现状继续下发）。
		if !shouldEnforceRepoForTenant(path, item.Region, tid) {
			slog.WarnContext(ctx, "gitlab_disk_project_owner_tenant_mismatch",
				"level", "warn",
				"tenant_id", item.TenantID,
				"project", path,
				"region", item.Region,
			)
			continue
		}
		if err := applyProjectRepositorySizeLimit(ctx, path, item.DiskLimitBytes); err != nil {
			slog.WarnContext(ctx, "gitlab_project_limit_failed",
				"level", "warn",
				"tenant_id", item.TenantID,
				"project", path,
				"error", err.Error(),
			)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// repoOwnerTenant 解析非 tenant-{id} 前缀仓（个人命名空间/组仓）的归属租户：
// 按仓所有者命名空间（path 顶层段）经 gitlab 用户名 → 凭据绑定 → 成员 → 区域资源
// 链归集（与 gitlab_traffic_gate_username.go resolveTenantByUsername 同链）。
// tenant-{id} 前缀仓由 path 直接可解，返回 0；解析失败/多租户歧义同样返回 0，
// 调用方保持 fail-safe（沿用现状）。
func repoOwnerTenant(projectPath, region string) int64 {
	p := strings.Trim(strings.TrimSpace(projectPath), "/")
	if p == "" || tenantGitlabGroupPathRe.MatchString(p) {
		return 0
	}
	owner := strings.SplitN(p, "/", 2)[0]
	if owner == "" || strings.ContainsAny(owner, " \t") {
		return 0
	}
	tid, err := resolveTenantIDFromGitlabUsername(owner, strings.TrimSpace(region))
	if err != nil {
		slog.Warn("gitlab_disk_repo_owner_unresolved",
			"level", "warn",
			"owner", owner,
			"region", strings.TrimSpace(region),
			"project_path", p,
			"error", err.Error(),
		)
		return 0
	}
	return tid
}

// shouldEnforceRepoForTenant 判断一个仓是否应被本项目租户限额执行：
// - tenant-{id} 前缀仓：直接属于本项目，执行；
// - 非前缀仓归属租户解析成功且与 tid 不一致 → 跳过（跨租户个人仓，避免误伤）；
// - 其余（解析失败/一致/无法解析）→ 执行（fail-safe）。
func shouldEnforceRepoForTenant(projectPath, region string, tid int64) bool {
	if tenantGitlabGroupPathRe.MatchString(strings.Trim(strings.TrimSpace(projectPath), "/")) {
		return true
	}
	ownerTid := repoOwnerTenant(projectPath, region)
	if ownerTid <= 0 {
		return true
	}
	return ownerTid == tid
}

func syncAndEnforceGitlabDiskQuotas(ctx context.Context, tenantIDs []int64) (gitlabDiskSyncResult, error) {
	result, err := syncGitlabDiskQuotas(ctx, tenantIDs)
	if err != nil {
		return result, err
	}
	if resolveGitlabAdminToken() == "" {
		slog.WarnContext(ctx, "gitlab_disk_enforce_skipped_no_admin_token", "level", "warn")
		return result, nil
	}
	for i := range result.Enforce {
		if result.Enforce[i].Error != "" {
			continue
		}
		if err := enforceGitlabDiskQuota(ctx, result.Enforce[i]); err != nil {
			result.Enforce[i].Error = err.Error()
		}
	}
	return result, nil
}
