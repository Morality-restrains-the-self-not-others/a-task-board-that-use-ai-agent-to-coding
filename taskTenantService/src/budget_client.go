package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// fetchLLMBudgetMeta loads llm_budget_enabled and per-member can_raise_task_budget
// map, both from taskCloudService (OPT-026: migrated from Django).
func fetchLLMBudgetMeta(companyID string) (enabled bool, raiseByMember map[string]bool) {
	raiseByMember = map[string]bool{}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return false, raiseByMember
	}
	enabled = fetchLLMBudgetEnabled(companyID)
	if !enabled {
		return false, raiseByMember
	}
	raiseByMember = fetchBudgetRaiseByMember(companyID)
	return enabled, raiseByMember
}

func fetchLLMBudgetEnabled(companyID string) bool {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskCloudServiceURL), "/")
	if base == "" {
		return false
	}
	q := url.Values{}
	q.Set("company_id", companyID)
	req, err := http.NewRequest(http.MethodGet, base+"/api/internal/cloud/tenant-budget-enabled/?"+q.Encode(), nil)
	if err != nil {
		return false
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return false
	}
	var out map[string]interface{}
	if json.Unmarshal(raw, &out) != nil {
		return false
	}
	if b, ok := out["llm_budget_enabled"].(bool); ok {
		return b
	}
	return false
}

func fetchBudgetRaiseByMember(companyID string) map[string]bool {
	out := map[string]bool{}
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskCloudServiceURL), "/")
	if base == "" {
		return out
	}
	q := url.Values{}
	q.Set("company_id", companyID)
	req, err := http.NewRequest(http.MethodGet, base+"/api/internal/budget/tenant-permissions/?"+q.Encode(), nil)
	if err != nil {
		return out
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return out
	}
	var body map[string]interface{}
	if json.Unmarshal(raw, &body) != nil {
		return out
	}
	items, _ := body["items"].([]interface{})
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		if strField(m, "subject_type") != "member" {
			continue
		}
		sid := strField(m, "subject_id")
		if sid == "" {
			continue
		}
		can := boolField(m, "can_raise_task_budget")
		if can {
			out[sid] = true
		}
	}
	return out
}
