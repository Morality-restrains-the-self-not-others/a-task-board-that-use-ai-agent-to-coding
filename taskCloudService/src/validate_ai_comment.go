package main

import (
	"fmt"
)

func parseContainerJobContext(body map[string]interface{}) map[string]interface{} {
	optStr := func(key string) interface{} {
		v, ok := body[key]
		if !ok || v == nil {
			return nil
		}
		s := stringsTrim(fmt.Sprintf("%v", v))
		if s == "" {
			return nil
		}
		return s
	}
	commandKind := stringsTrim(fmt.Sprintf("%v", body["command_kind"]))
	if commandKind == "" {
		commandKind = "trae"
	}
	if commandKind != "trae" && commandKind != "shell" {
		commandKind = "trae"
	}
	out := map[string]interface{}{
		"parent_job_id": optStr("parent_job_id"),
		"repo_layer_id": optStr("repo_layer_id"),
		"command_kind":  commandKind,
	}
	if v, ok := body["agent_auto_iteration_count"]; ok {
		out["agent_auto_iteration_count"] = v
	}
	if v, ok := body["agent_models"]; ok {
		out["agent_models"] = v
	}
	return out
}

func stringsTrim(s string) string {
	return trim(s)
}

func validateAICommentPost(tenantID, workspaceID, taskID string, body map[string]interface{}, traceID string) (int, map[string]interface{}) {
	content := stringsTrim(strField(body, "content"))
	if content == "" {
		return 400, map[string]interface{}{"detail": "content required"}
	}

	commentID := stringsTrim(strField(body, "comment_id"))
	cscID := stringsTrim(strField(body, "csc_id"))
	cfg, err := loadCloudServerConfigForGatewayForward(tenantID, workspaceID, taskID, commentID, cscID)
	if err != nil || cfg == nil {
		return 400, map[string]interface{}{
			"detail": "任务云配置中缺少可访问的 server_url，暂无法使用「发送给 AI」",
		}
	}
	baseURL, svcToken := resolveContainerTarget(cfg, "")
	if baseURL == "" {
		return 400, map[string]interface{}{
			"detail": "任务云配置中缺少可访问的 server_url，暂无法使用「发送给 AI」",
		}
	}

	streamBackend := detectStreamBackend(baseURL, svcToken, traceID)
	if streamBackend == "trae" {
		jobCtx := parseContainerJobContext(body)
		pj := jobCtx["parent_job_id"]
		rl := jobCtx["repo_layer_id"]
		hasPJ := pj != nil
		hasRL := rl != nil
		if hasPJ == hasRL {
			return 400, map[string]interface{}{
				"detail": "Trae 在线服务要求创建任务时须且仅能指定 parent_job_id 或 repo_layer_id 之一（与 onlineServiceJS/skill.md 一致）。请先在层级图中选中可写层或任务节点后再发送。",
			}
		}
	}
	return 200, map[string]interface{}{"ok": true}
}
