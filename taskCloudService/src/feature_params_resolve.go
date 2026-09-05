package main

import (
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

// Hierarchy resolve aligned with Django FeatureParamsResolver:
// personal (if workspace allows) → workspace override → company.

type featureParamsConfig struct {
	ID              string
	ProvidersJSON   string
	AgentModel      string
	AgentProvider   string
	AgentMaxSteps   int
	SummaryModel    string
	SummaryProvider string
	ExtraEnvJSON    string
	DisplayName     string
	Scope           string // company | workspace | personal
}

type taskFeatureParamsBinding struct {
	Source           string
	PersonalConfigID string
	OwnerID          string
	WorkspaceID      string
	TenantID         string
}

type featureParamsSnapshotMeta struct {
	Source            string
	SourceConfigID    string
	SourceDisplayName string
}

var (
	fetchTaskFeatureParamsBindingFn = fetchTaskFeatureParamsBindingHTTP
	fetchWorkspaceAllowPersonalFn   = fetchWorkspaceAllowPersonalHTTP
	featureParamsHTTPClient         = &http.Client{Timeout: 10 * time.Second}
)

func companyConfigFromTenantRow(params *tenantFeatureParamsRow, displayName string) *featureParamsConfig {
	if params == nil {
		return &featureParamsConfig{
			ID:            "",
			ProvidersJSON: "[]",
			AgentMaxSteps: 200,
			ExtraEnvJSON:  "[]",
			DisplayName:   displayName,
			Scope:         "company",
		}
	}
	steps := params.AgentMaxSteps
	if steps <= 0 {
		steps = 200
	}
	extra := params.ExtraEnvJSON
	if strings.TrimSpace(extra) == "" {
		extra = "[]"
	}
	providers := params.ProvidersJSON
	if strings.TrimSpace(providers) == "" {
		providers = "[]"
	}
	return &featureParamsConfig{
		ID:              params.ID,
		ProvidersJSON:   providers,
		AgentModel:      params.AgentModel,
		AgentProvider:   params.AgentProvider,
		AgentMaxSteps:   steps,
		SummaryModel:    params.SummaryModel,
		SummaryProvider: params.SummaryProvider,
		ExtraEnvJSON:    extra,
		DisplayName:     displayName,
		Scope:           "company",
	}
}

func resolveFeatureParamsHierarchy(
	ctx context.Context, companyID, workspaceID, source, personalConfigID, userID string,
) (*featureParamsConfig, featureParamsSnapshotMeta, error) {
	params, err := loadTenantFeatureParams(companyID)
	if err != nil {
		return nil, featureParamsSnapshotMeta{}, err
	}
	companyCfg := companyConfigFromTenantRow(params, "公司默认")
	src := strings.ToLower(strings.TrimSpace(source))
	if src == "" {
		src = "company"
	}

	switch src {
	case "personal":
		allowed, err := fetchWorkspaceAllowPersonalFn(ctx, workspaceID)
		if err != nil {
			logWarn("feature_params: workspace allow_personal lookup failed: "+err.Error(), "")
			allowed = false
		}
		if !allowed {
			meta := featureParamsSnapshotMeta{
				Source:            "company",
				SourceConfigID:    companyCfg.ID,
				SourceDisplayName: "公司默认（个人配置被工作空间策略禁用，已回退）",
			}
			return companyCfg, meta, nil
		}
		personal, err := loadPersonalFeatureParams(personalConfigID, userID)
		if err != nil {
			return nil, featureParamsSnapshotMeta{}, err
		}
		if personal != nil {
			return personal, featureParamsSnapshotMeta{
				Source:            "personal",
				SourceConfigID:    personal.ID,
				SourceDisplayName: personal.DisplayName,
			}, nil
		}
		return companyCfg, featureParamsSnapshotMeta{
			Source:            "company",
			SourceConfigID:    companyCfg.ID,
			SourceDisplayName: "公司默认（个人配置无效，已回退，已回退）",
		}, nil

	case "workspace":
		ws, hasCustom, err := loadWorkspaceFeatureParams(workspaceID)
		if err != nil {
			return nil, featureParamsSnapshotMeta{}, err
		}
		if hasCustom && ws != nil {
			return ws, featureParamsSnapshotMeta{
				Source:            "workspace",
				SourceConfigID:    ws.ID,
				SourceDisplayName: ws.DisplayName,
			}, nil
		}
		return companyCfg, featureParamsSnapshotMeta{
			Source:            "company",
			SourceConfigID:    companyCfg.ID,
			SourceDisplayName: "公司默认（工作空间继承）",
		}, nil

	default:
		return companyCfg, featureParamsSnapshotMeta{
			Source:            "company",
			SourceConfigID:    companyCfg.ID,
			SourceDisplayName: "公司默认",
		}, nil
	}
}

func serializeFeatureParamsEnv(cfgRow *featureParamsConfig) map[string]string {
	if cfgRow == nil {
		cfgRow = companyConfigFromTenantRow(nil, "公司默认")
	}
	providers := cfgRow.ProvidersJSON
	if strings.TrimSpace(providers) == "" {
		providers = "[]"
	}
	maxSteps := cfgRow.AgentMaxSteps
	if maxSteps <= 0 {
		maxSteps = 200
	}
	env := map[string]string{
		envProvidersJSON:   providers,
		envAgentModel:      cfgRow.AgentModel,
		envAgentProvider:   cfgRow.AgentProvider,
		envAgentMaxSteps:   fmt.Sprintf("%d", maxSteps),
		envSummaryModel:    cfgRow.SummaryModel,
		envSummaryProvider: cfgRow.SummaryProvider,
	}
	var extras []map[string]any
	if err := json.Unmarshal([]byte(cfgRow.ExtraEnvJSON), &extras); err == nil {
		for _, item := range extras {
			key := strings.TrimSpace(fmt.Sprintf("%v", item["key"]))
			if key == "" || key == "<nil>" {
				continue
			}
			env[key] = fmt.Sprintf("%v", item["value"])
		}
	}
	scope := strings.TrimSpace(cfgRow.Scope)
	if scope == "" {
		scope = "company"
	}
	env[envScope] = scope
	env[envConfigID] = cfgRow.ID
	env[envConfigName] = cfgRow.DisplayName
	coerceProvidersBaseURLsInEnv(env)
	return env
}

func fetchTaskFeatureParamsBindingHTTP(tenantID, taskID string) (*taskFeatureParamsBinding, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if base == "" {
		return nil, fmt.Errorf("task service url not configured")
	}
	u := base + "/api/tasks/" + url.PathEscape(taskID) + "/"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := featureParamsHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task service status=%d body=%s", resp.StatusCode, truncateForLog(string(raw), 200))
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	owner := strings.TrimSpace(fmt.Sprintf("%v", body["owner"]))
	if owner == "" || owner == "<nil>" {
		owner = strings.TrimSpace(fmt.Sprintf("%v", body["owner_id"]))
	}
	if owner == "<nil>" {
		owner = ""
	}
	source := strings.TrimSpace(fmt.Sprintf("%v", body["feature_params_source"]))
	if source == "" || source == "<nil>" {
		source = "company"
	}
	pfpc := strings.TrimSpace(fmt.Sprintf("%v", body["personal_feature_params_config_id"]))
	if pfpc == "<nil>" {
		pfpc = ""
	}
	ws := strings.TrimSpace(fmt.Sprintf("%v", body["workspace_id"]))
	if ws == "<nil>" {
		ws = ""
	}
	tid := strings.TrimSpace(fmt.Sprintf("%v", body["tenant_id"]))
	if tid == "<nil>" || tid == "" {
		tid = tenantID
	}
	return &taskFeatureParamsBinding{
		Source:           source,
		PersonalConfigID: pfpc,
		OwnerID:          owner,
		WorkspaceID:      ws,
		TenantID:         tid,
	}, nil
}

func fetchWorkspaceAllowPersonalHTTP(ctx context.Context, workspaceID string) (bool, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.ProjectServiceURL), "/")
	if base == "" {
		return false, fmt.Errorf("project service url not configured")
	}
	u := base + "/api/internal/workspaces/" + url.PathEscape(workspaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	// OPT-20260821-012: 把入站 X-Trace-Id / span 透传到 taskProjectService。
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := featureParamsHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("project service status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return false, err
	}
	switch v := body["allow_personal_feature_params"].(type) {
	case bool:
		return v, nil
	case float64:
		return v != 0, nil
	case string:
		return strings.EqualFold(v, "true") || v == "1", nil
	default:
		return false, nil
	}
}

func truncateForLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
