package main

import (
	"snowflake"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

func insertFeatureParamsSnapshotLocal(
	taskID, workspaceID, tenantID string,
	meta featureParamsSnapshotMeta,
	env map[string]string,
	cfgRow *featureParamsConfig,
) error {
	conn, err := featureParamsConn()
	if err != nil {
		return err
	}
	providersJSON := "[]"
	if cfgRow != nil {
		providersJSON = cfgRow.ProvidersJSON
	}
	summaryRaw := providersSummaryJSON(providersJSON)
	agentModel := env[envAgentModel]
	agentProvider := env[envAgentProvider]
	summaryModel := env[envSummaryModel]
	summaryProvider := env[envSummaryProvider]
	maxSteps := 200
	if v := strings.TrimSpace(env[envAgentMaxSteps]); v != "" {
		if n, scanErr := fmt.Sscanf(v, "%d", &maxSteps); scanErr != nil || n != 1 {
			maxSteps = 200
		}
	}
	envJSON, _ := json.Marshal(env)
	now := fpNowUTC()
	_, err = conn.Exec(`
		INSERT INTO cloud_task_feature_params_snapshot (
			id, task_id, workspace_id, tenant_id, source, source_config_id, source_display_name,
			resolved_env, providers_summary, agent_model, agent_model_provider, agent_max_steps,
			summary_model, summary_model_provider, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snowflake.GenerateIDString(), taskID, workspaceID, tenantID,
		meta.Source, meta.SourceConfigID, meta.SourceDisplayName,
		string(envJSON), summaryRaw, agentModel, agentProvider, maxSteps,
		summaryModel, summaryProvider, now,
	)
	return err
}

func insertTaskApiKeyUsageLocal(
	tenantID, workspaceID, taskID string,
	providers []map[string]any,
) error {
	conn, err := featureParamsConn()
	if err != nil {
		return err
	}
	now := fpNowUTC()
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		providerName := strings.TrimSpace(fmt.Sprintf("%v", provider["provider"]))
		if providerName == "" || providerName == "<nil>" {
			continue
		}
		apiKey := strings.TrimSpace(fmt.Sprintf("%v", provider["api_key"]))
		if apiKey == "<nil>" {
			apiKey = ""
		}
		apiKeyHash := ""
		if apiKey != "" {
			sum := sha256.Sum256([]byte(apiKey))
			apiKeyHash = fmt.Sprintf("%x", sum)
		}
		keyType := "master"
		if providerBool(provider["use_sub_token"]) {
			keyType = "sub"
		}
		_, err = conn.Exec(`
			INSERT INTO cloud_task_api_key_usage (
				id, task_id, workspace_id, tenant_id, provider_name, api_key_hash, key_type, used_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			snowflake.GenerateIDString(), taskID, workspaceID, tenantID,
			providerName, apiKeyHash, keyType, now,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertFeatureParamsAccessAuditLocal(payload map[string]interface{}) error {
	conn, err := featureParamsConn()
	if err != nil {
		return err
	}
	statusCode := intFromAny(payload["status_code"], 0)
	_, err = conn.Exec(`
		INSERT INTO cloud_feature_params_access_audit (
			id, user_id, company_id, workspace_id, resource, access_context, view_mode,
			auth_method, http_method, path, status_code, client_ip, user_agent, referer,
			trace_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snowflake.GenerateIDString(),
		truncateStr(strField(payload, "user_id"), 64),
		truncateStr(strField(payload, "company_id"), 64),
		truncateStr(strField(payload, "workspace_id"), 64),
		truncateStr(strField(payload, "resource"), 32),
		truncateStr(strField(payload, "access_context"), 64),
		truncateStr(strField(payload, "view_mode"), 16),
		truncateStr(strField(payload, "auth_method"), 16),
		truncateStr(strField(payload, "http_method"), 16),
		truncateStr(strField(payload, "path"), 512),
		statusCode,
		truncateStr(strField(payload, "client_ip"), 64),
		truncateStr(strField(payload, "user_agent"), 512),
		truncateStr(strField(payload, "referer"), 1024),
		truncateStr(strField(payload, "trace_id"), 64),
		fpNowUTC(),
	)
	return err
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func countFeatureParamsSnapshots(taskID string) (int, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return 0, err
	}
	var n int
	err = conn.QueryRow(
		`SELECT COUNT(*) FROM cloud_task_feature_params_snapshot WHERE task_id = ?`, taskID,
	).Scan(&n)
	return n, err
}

func countTaskApiKeyUsage(taskID string) (int, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return 0, err
	}
	var n int
	err = conn.QueryRow(
		`SELECT COUNT(*) FROM cloud_task_api_key_usage WHERE task_id = ?`, taskID,
	).Scan(&n)
	return n, err
}
