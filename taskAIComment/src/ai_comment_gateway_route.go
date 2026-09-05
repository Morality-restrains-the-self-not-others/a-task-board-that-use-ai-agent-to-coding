package main

import (
	"errors"
	"net/http"
	"strings"

	"gatewayauth"
)

// errAICommentGatewayNotFound — 路径为空或无法识别资源前缀。
var errAICommentGatewayNotFound = errors.New("ai-comment gateway route not found")

// errAICommentGatewayMissingScope — 缺少 tenant_id / workspace_id / task_id。
var errAICommentGatewayMissingScope = errors.New("ai-comment gateway scope required")

// aiCommentGatewayRoute 是 /api/ai-comment/* 经约定路径解析后的分发输入。
type aiCommentGatewayRoute struct {
	FuncName    string
	TenantID    string
	WorkspaceID string
	TaskID      string
	Rest        string
}

// parseAICommentGatewayRoute 解析网关前缀后的相对路径。
// 使用 gatewayauth.ParseConventionPath，保证 kv-last 后的单个位置段
// （如 ai-comments、container-agent id）不会被吞掉。
func parseAICommentGatewayRoute(r *http.Request, path string) (aiCommentGatewayRoute, error) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return aiCommentGatewayRoute{}, errAICommentGatewayNotFound
	}
	remaining := gatewayauth.ParseConventionPath(r, trimmed)
	parts := strings.Split(strings.Trim(remaining, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return aiCommentGatewayRoute{}, errAICommentGatewayNotFound
	}
	tenantID := strings.TrimSpace(r.Header.Get(gatewayauth.HeaderAuthTenantID))
	workspaceID := strings.TrimSpace(r.Header.Get("X-Workspace-Id"))
	taskID := strings.TrimSpace(r.Header.Get("X-Task-Id"))
	if tenantID == "" || workspaceID == "" || taskID == "" {
		return aiCommentGatewayRoute{}, errAICommentGatewayMissingScope
	}
	if getAuthTenant(r) == "" {
		r.Header.Set(gatewayauth.HeaderAuthTenantID, tenantID)
	}
	return aiCommentGatewayRoute{
		FuncName:    parts[0],
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		TaskID:      taskID,
		Rest:        strings.Trim(strings.Join(parts[1:], "/"), "/"),
	}, nil
}
