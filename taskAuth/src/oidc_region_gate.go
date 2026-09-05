package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OIDC 区域模式闸门（OPT-20260823-051）：
// GitLab 区域实例（client_id 以 gitlab-git-service 开头）直连 taskAuth OIDC 登录时，
// 按 taskBill billing_gitlab_region.access_mode 判定目标区域：
//
//	development → 仅 is_tester=1 账号可签发授权码；其余以 access_denied 拒绝。
//	区域模式查询失败 → 按 release 放行并告警（与区域模式缺省 release 一致）；
//	已知 development 但 is_tester 校验失败 → 拒绝（fail-closed，不能证明测试角色不放行）。
//
// SSOT：区域模式在 taskBill；禁止 auth 直连 bill 库，走内部只读 API
// /api/internal/taskbill/gitlab-regions-admin/（X-TaskBill-Internal-Secret）。
const (
	// oidcGitlabClientPrefix 同时覆盖默认实例 gitlab-git-service 与
	// 区域实例 gitlab-git-service-<slug>（conf/infra/git-service*/config.yaml oidcClientId）。
	oidcGitlabClientPrefix      = "gitlab-git-service"
	oidcRegionAccessDevelopment = "development"
	oidcRegionAccessRelease     = "release"
	oidcRegionGateDenyMsg       = "该区域处于开发模式，仅测试角色账号可使用"
)

type oidcRegionModeCacheEntry struct {
	hostModes map[string]string // gitlab 主机名（小写）→ 归一化 access_mode
	fetchedAt time.Time
}

var (
	oidcRegionModeCacheMu sync.Mutex
	oidcRegionModeCache   *oidcRegionModeCacheEntry
)

func oidcRegionModeCacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TASKAUTH_OIDC_REGION_MODE_CACHE_TTL_MS"))
	if raw == "" {
		return 30 * time.Second
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return 30 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

// resetOidcRegionModeCacheForTest 清空区域模式缓存（测试隔离用）。
func resetOidcRegionModeCacheForTest() {
	oidcRegionModeCacheMu.Lock()
	defer oidcRegionModeCacheMu.Unlock()
	oidcRegionModeCache = nil
}

// oidcRegionGateEnabled 仅对 GitLab 区域实例 OIDC client 启用闸门；
// ai-provider / chrome-extension 等其他 client 不参与区域模式判定。
func oidcRegionGateEnabled(clientID string) bool {
	return strings.HasPrefix(strings.TrimSpace(clientID), oidcGitlabClientPrefix)
}

func normalizeOidcRegionAccessMode(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), oidcRegionAccessDevelopment) {
		return oidcRegionAccessDevelopment
	}
	return oidcRegionAccessRelease
}

func hostOfURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return strings.ToLower(u.Hostname()), nil
}

// gitlabRegionAccessModeForRedirectURI 返回 redirect_uri 对应区域的 access_mode。
// known=false 表示该 redirect_uri 不属于任何已登记区域（不拦截，fail-open）。
// err 仅表示区域模式查询本身失败（调用方按 release 放行并告警）。
func gitlabRegionAccessModeForRedirectURI(ctx context.Context, redirectURI string) (mode string, known bool, err error) {
	host, err := hostOfURL(redirectURI)
	if err != nil || host == "" {
		return "", false, fmt.Errorf("invalid redirect_uri host: %w", err)
	}
	hostModes, err := gitlabRegionHostModes(ctx)
	if err != nil {
		return "", false, err
	}
	mode, ok := hostModes[host]
	if !ok {
		return "", false, nil
	}
	return normalizeOidcRegionAccessMode(mode), true, nil
}

// gitlabRegionHostModes 拉取区域模式映射（gitlab 主机 → access_mode），带短 TTL 缓存。
// 使用 gitlab-regions-admin 全量端点（非 /gitlab-regions/，后者会按调用方
// X-User-Is-Tester 过滤掉 development 区域，导致闸门看不到开发区域）。
func gitlabRegionHostModes(ctx context.Context) (map[string]string, error) {
	oidcRegionModeCacheMu.Lock()
	if oidcRegionModeCache != nil && time.Since(oidcRegionModeCache.fetchedAt) < oidcRegionModeCacheTTL() {
		cached := oidcRegionModeCache.hostModes
		oidcRegionModeCacheMu.Unlock()
		return cached, nil
	}
	oidcRegionModeCacheMu.Unlock()

	raw, status, err := internalServiceGet(ctx,
		billServiceBaseURL()+"/api/internal/taskbill/gitlab-regions-admin/",
		"X-TaskBill-Internal-Secret", cfg.BillInternalSecret)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("taskbill gitlab-regions-admin status=%d", status)
	}
	var resp struct {
		Regions []struct {
			GitlabWebURL string `json:"gitlab_web_url"`
			AccessMode   string `json:"access_mode"`
		} `json:"regions"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	hostModes := make(map[string]string, len(resp.Regions))
	for _, r := range resp.Regions {
		host, err := hostOfURL(r.GitlabWebURL)
		if err != nil || host == "" {
			continue
		}
		hostModes[host] = normalizeOidcRegionAccessMode(r.AccessMode)
	}

	oidcRegionModeCacheMu.Lock()
	oidcRegionModeCache = &oidcRegionModeCacheEntry{hostModes: hostModes, fetchedAt: time.Now()}
	oidcRegionModeCacheMu.Unlock()
	slog.DebugContext(ctx, "oidc_region_modes_refreshed", "level", "debug", "regions", len(hostModes))
	return hostModes, nil
}
