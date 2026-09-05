package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"tracelog"
)

// tenantMemberHit is one membership row from taskTenantService batch-get.
type tenantMemberHit struct {
	UserID      string
	CompanyID   string
	CompanyName string
	IsActive    bool
}

// groupActiveTenantCompanies keeps active memberships, drops duplicate
// company_id per user, falls back empty names to company_id, and sorts by name.
func groupActiveTenantCompanies(hits []tenantMemberHit) map[string][]map[string]string {
	result := make(map[string][]map[string]string)
	seen := map[string]map[string]bool{}
	for _, h := range hits {
		uid := strings.TrimSpace(h.UserID)
		cid := strings.TrimSpace(h.CompanyID)
		if uid == "" || cid == "" || !h.IsActive {
			continue
		}
		if seen[uid] == nil {
			seen[uid] = map[string]bool{}
		}
		if seen[uid][cid] {
			continue
		}
		seen[uid][cid] = true
		name := strings.TrimSpace(h.CompanyName)
		if name == "" {
			name = cid
		}
		result[uid] = append(result[uid], map[string]string{"id": cid, "name": name})
	}
	for uid, list := range result {
		sort.Slice(list, func(i, j int) bool {
			if list[i]["name"] == list[j]["name"] {
				return list[i]["id"] < list[j]["id"]
			}
			return list[i]["name"] < list[j]["name"]
		})
		result[uid] = list
	}
	return result
}

// fetchTenantCompaniesBatch 批量查 taskTenantService 内部 members/batch-get，
// 返回 user_id → [{id, name}]。best-effort：服务不可达/异常时返回空 map，
// 用户列表仍正常返回。
func fetchTenantCompaniesBatch(ctx context.Context, userIDs []string) map[string][]map[string]string {
	result := make(map[string][]map[string]string)
	if len(userIDs) == 0 {
		return result
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.TenantServiceURL), "/")
	if base == "" {
		slog.WarnContext(ctx, "tenant_companies_lookup_skipped", "reason", "empty_tenant_service_url", "user_count", len(userIDs))
		return result
	}
	payload, err := json.Marshal(map[string]interface{}{"user_ids": userIDs})
	if err != nil {
		slog.WarnContext(ctx, "tenant_companies_lookup_marshal_failed", "error", err.Error())
		return result
	}
	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/internal/tenant/members/batch-get/", bytes.NewReader(payload))
	if err != nil {
		slog.WarnContext(ctx, "tenant_companies_lookup_request_failed", "error", err.Error())
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(5 * time.Second).Do(req)
	if err != nil {
		slog.WarnContext(ctx, "tenant_companies_lookup_http_failed", "error", err.Error(), "user_count", len(userIDs))
		return result
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		slog.WarnContext(ctx, "tenant_companies_lookup_rejected", "status", resp.StatusCode, "user_count", len(userIDs), "duration_ms", time.Since(started).Milliseconds())
		return result
	}
	var parsed struct {
		Members []struct {
			UserID      string `json:"user_id"`
			CompanyID   string `json:"company_id"`
			CompanyName string `json:"company_name"`
			IsActive    bool   `json:"is_active"`
		} `json:"members"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		slog.WarnContext(ctx, "tenant_companies_lookup_unmarshal_failed", "error", err.Error())
		return result
	}
	hits := make([]tenantMemberHit, 0, len(parsed.Members))
	for _, m := range parsed.Members {
		hits = append(hits, tenantMemberHit{
			UserID:      m.UserID,
			CompanyID:   m.CompanyID,
			CompanyName: m.CompanyName,
			IsActive:    m.IsActive,
		})
	}
	result = groupActiveTenantCompanies(hits)
	slog.InfoContext(ctx, "tenant_companies_lookup_ok", "user_count", len(userIDs), "hit_users", len(result), "duration_ms", time.Since(started).Milliseconds())
	return result
}

func attachTenantCompanies(users []map[string]interface{}, byUser map[string][]map[string]string) {
	for _, u := range users {
		uid, _ := u["id"].(string)
		if list := byUser[uid]; len(list) > 0 {
			u["tenant_companies"] = list
			continue
		}
		u["tenant_companies"] = []map[string]string{}
	}
}
