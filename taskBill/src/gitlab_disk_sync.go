package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"tracelog"
)

type tenantDiskUsageRemote struct {
	TenantID      string `json:"tenant_id"`
	DiskUsedBytes int64  `json:"disk_used_bytes"`
	RepoCount     int    `json:"repo_count"`
	MeasuredCount int    `json:"measured_count"`
	Repos         []struct {
		RepoURL   string `json:"repo_url"`
		SizeBytes int64  `json:"size_bytes"`
		OK        bool   `json:"ok"`
	} `json:"repos"`
}

type gitlabDiskSyncItem struct {
	TenantID       string   `json:"tenant_id"`
	Region         string   `json:"region,omitempty"`
	DiskGB         int64    `json:"disk_gb"`
	DiskUsedBytes  int64    `json:"disk_used_bytes"`
	DiskUsedGB     float64  `json:"disk_used_gb"`
	DiskLimitBytes int64    `json:"disk_limit_bytes"`
	RepoURLs       []string `json:"repo_urls"`
	Error          string   `json:"error,omitempty"`
}

type gitlabDiskSyncResult struct {
	Synced   []gitlabDiskSyncItem `json:"synced"`
	Enforce  []gitlabDiskSyncItem `json:"enforce"`
	SyncedAt string               `json:"synced_at"`
}

var (
	diskUsageRefreshMu    sync.Mutex
	diskUsageRefreshCache = map[int64]time.Time{}
)

const diskUsageRefreshTTL = 60 * time.Second

func listTenantGitlabResourceIDs() ([]int64, error) {
	rows, err := db.Query(`SELECT tenant_id FROM billing_tenant_gitlab_resource ORDER BY tenant_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]int64, 0)
	for rows.Next() {
		var tid int64
		if err := rows.Scan(&tid); err != nil {
			return nil, err
		}
		out = append(out, tid)
	}
	return out, rows.Err()
}

func fetchTenantDiskUsageFromProjectService(ctx context.Context, tenantID int64) (*tenantDiskUsageRemote, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskProjectServiceBase), "/")
	if base == "" {
		base = "http://127.0.0.1:8016"
	}
	url := fmt.Sprintf("%s/api/internal/tenants/%s/gitlab-local-disk-usage/", base, formatID(tenantID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	if t := strings.TrimSpace(cfg.GitlabAdminPrivateToken); t != "" {
		req.Header.Set("X-Gitlab-Admin-Token", t)
	}
	client := tracelog.DirectClient(20 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("taskProjectService status=%d body=%s", resp.StatusCode, truncateForErr(string(body), 200))
	}
	var out tenantDiskUsageRemote
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func truncateForErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func diskLimitBytesFromGB(diskGB int64) int64 {
	if diskGB <= 0 {
		return 1 // GitLab treats 0 as unlimited
	}
	return diskGB * 1024 * 1024 * 1024
}

func refreshTenantDiskUsage(ctx context.Context, tenantID int64, regionSlug string, force bool) error {
	diskUsageRefreshMu.Lock()
	if !force {
		if at, ok := diskUsageRefreshCache[tenantID]; ok && time.Since(at) < diskUsageRefreshTTL {
			diskUsageRefreshMu.Unlock()
			return nil
		}
	}
	diskUsageRefreshMu.Unlock()

	remote, err := fetchTenantDiskUsageFromProjectService(ctx, tenantID)
	if err != nil {
		return err
	}
	if err := reportGitlabDiskUsage(tenantID, remote.DiskUsedBytes, regionSlug); err != nil {
		return err
	}
	diskUsageRefreshMu.Lock()
	diskUsageRefreshCache[tenantID] = time.Now()
	diskUsageRefreshMu.Unlock()
	return nil
}

func fetchTenantIDsWithGitlabLocalRepos(ctx context.Context) ([]int64, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskProjectServiceBase), "/")
	if base == "" {
		base = "http://127.0.0.1:8016"
	}
	url := base + "/api/internal/tenants/with-gitlab-local-repos/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := tracelog.DirectClient(20 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("candidates status=%d body=%s", resp.StatusCode, truncateForErr(string(body), 200))
	}
	var payload struct {
		TenantIDs []interface{} `json:"tenant_ids"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(payload.TenantIDs))
	for _, v := range payload.TenantIDs {
		id, err := parseIDField(v)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

func syncGitlabDiskQuotas(ctx context.Context, tenantIDs []int64) (gitlabDiskSyncResult, error) {
	result := gitlabDiskSyncResult{
		Synced:   []gitlabDiskSyncItem{},
		Enforce:  []gitlabDiskSyncItem{},
		SyncedAt: utcNow(),
	}
	ids := tenantIDs
	if len(ids) == 0 {
		var err error
		ids, err = listTenantGitlabResourceIDs()
		if err != nil {
			return result, err
		}
		if candidates, cErr := fetchTenantIDsWithGitlabLocalRepos(ctx); cErr == nil {
			seen := map[int64]struct{}{}
			for _, id := range ids {
				seen[id] = struct{}{}
			}
			for _, id := range candidates {
				if _, ok := seen[id]; ok {
					continue
				}
				ids = append(ids, id)
				seen[id] = struct{}{}
			}
		} else {
			slog.WarnContext(ctx, "gitlab_disk_candidates_fetch_failed", "level", "warn", "error", cErr.Error())
		}
	}
	for _, tid := range ids {
		item := gitlabDiskSyncItem{TenantID: formatID(tid), RepoURLs: []string{}}
		res, err := getTenantGitlabResource(tid)
		if err != nil {
			item.Error = err.Error()
			result.Synced = append(result.Synced, item)
			continue
		}
		item.DiskGB = res.DiskGB
		item.Region = res.Region
		item.DiskLimitBytes = diskLimitBytesFromGB(res.DiskGB)

		remote, err := fetchTenantDiskUsageFromProjectService(ctx, tid)
		if err != nil {
			item.Error = err.Error()
			result.Synced = append(result.Synced, item)
			continue
		}
		if err := reportGitlabDiskUsage(tid, remote.DiskUsedBytes, "tencent-shanghai-5"); err != nil {
			item.Error = err.Error()
			result.Synced = append(result.Synced, item)
			continue
		}
		item.DiskUsedBytes = remote.DiskUsedBytes
		item.DiskUsedGB = diskUsedGBFromBytes(remote.DiskUsedBytes)
		for _, r := range remote.Repos {
			if strings.TrimSpace(r.RepoURL) != "" {
				item.RepoURLs = append(item.RepoURLs, r.RepoURL)
			}
		}
		result.Synced = append(result.Synced, item)
		result.Enforce = append(result.Enforce, item)
		diskUsageRefreshMu.Lock()
		diskUsageRefreshCache[tid] = time.Now()
		diskUsageRefreshMu.Unlock()
	}
	return result, nil
}

func handleInternalListGitlabResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	ids, err := listTenantGitlabResourceIDs()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	items := make([]map[string]interface{}, 0, len(ids))
	for _, tid := range ids {
		view, err := gitlabResourceView(tid)
		if err != nil {
			continue
		}
		items = append(items, view)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": items, "total": len(items)})
}

func handleInternalSyncGitlabDiskQuotas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	var ids []int64
	if raw, ok := body["tenant_ids"].([]interface{}); ok {
		for _, v := range raw {
			id, err := parseIDField(v)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_ids", tracelog.TraceIDFromContext(r.Context()))
				return
			}
			ids = append(ids, id)
		}
	} else if body["tenant_id"] != nil {
		id, err := parseIDField(body["tenant_id"])
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		ids = []int64{id}
	}
	// 用量同步在 taskBill；硬限额由 gitService/scripts/sync_tenant_gitlab_disk_quota.sh
	// 经 gitlab-rails 下发（CE REST 不接受 repository_size_limit）。
	result, err := syncAndEnforceGitlabDiskQuotas(r.Context(), ids)
	if err != nil {
		slog.ErrorContext(r.Context(), "gitlab_disk_quota_sync_failed", "level", "error", "error", err.Error())
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}
