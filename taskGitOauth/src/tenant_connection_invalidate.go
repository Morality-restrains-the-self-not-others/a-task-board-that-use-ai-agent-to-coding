package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// invalidateTaskProjectGitLabConnCache asks taskProjectService to drop its
// in-process Path A GitLab connection cache after a tenant-connection PUT/DELETE,
// so branch previews immediately use the new base_url/provider instead of a stale
// 30s(→5s) cached entry. Best-effort: failures only log; the shortened TTL is the
// backstop. OPT-20260827-006.
func (a *App) invalidateTaskProjectGitLabConnCache(tenantID string) {
	base := strings.TrimRight(strings.TrimSpace(a.Cfg.TaskProjectServiceURL), "/")
	if base == "" {
		return
	}
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return
	}
	endpoint := fmt.Sprintf("%s/api/internal/taskproject/tenant-gitlab-cache/invalidate?company_id=%s", base, url.QueryEscape(tid))
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		logWarn("taskProject gitlab-conn cache invalidate request: %v", err)
		return
	}
	req.Header.Set("Accept", "application/json")
	if sec := strings.TrimSpace(a.Cfg.DjangoInternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logWarn("taskProject gitlab-conn cache invalidate: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		logWarn("taskProject gitlab-conn cache invalidate http %d", resp.StatusCode)
	}
}
