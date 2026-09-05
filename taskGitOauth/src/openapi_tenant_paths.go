package main

func openAPITenantConnectionPaths(xUserID, bridgeHeader any) map[string]any {
	tidParam := map[string]any{
		"name": "tid", "in": "path", "required": true, "schema": map[string]any{"type": "string"},
	}
	putBody := map[string]any{
		"required": true,
		"content": map[string]any{"application/json": map[string]any{
			"schema": map[string]any{
				"type":     "object",
				"required": []string{"base_url", "client_id", "client_secret"},
				"properties": map[string]any{
					"base_url":      map[string]any{"type": "string", "format": "uri"},
					"client_id":     map[string]any{"type": "string"},
					"client_secret": map[string]any{"type": "string"},
					"remark":        map[string]any{"type": "string"},
					"intranet":      map[string]any{"type": "boolean", "description": "内网 GitLab：平台探测不可达属预期"},
				},
			},
		}},
	}
	return map[string]any{
		"/api/git-oauth/tenant-connection/tenant_id/{tid}/": map[string]any{
			"get": map[string]any{
				"summary":     "获取租户 GitLab OAuth 连接（脱敏）",
				"operationId": "getTenantGitlabOAuthConnection",
				"parameters":  []any{tidParam, xUserID},
				"responses": map[string]any{
					"200": map[string]any{"description": "连接信息（无 client_secret）"},
					"401": map[string]any{"description": "未认证"},
					"403": map[string]any{"description": "非租户成员"},
				},
			},
			"put": map[string]any{
				"summary":     "创建或更新租户 GitLab OAuth 连接",
				"operationId": "putTenantGitlabOAuthConnection",
				"parameters": []any{
					tidParam, xUserID,
					map[string]any{"name": "X-Is-Admin", "in": "header", "schema": map[string]any{"type": "string"}, "description": "测试旁路时管理员标记"},
				},
				"requestBody": putBody,
				"responses": map[string]any{
					"200": map[string]any{"description": "已保存"},
					"403": map[string]any{"description": "需要租户管理员"},
				},
			},
			"delete": map[string]any{
				"summary":     "删除租户 GitLab OAuth 连接并级联清除凭据",
				"operationId": "deleteTenantGitlabOAuthConnection",
				"parameters":  []any{tidParam, xUserID},
				"responses": map[string]any{
					"204": map[string]any{"description": "已删除"},
					"403": map[string]any{"description": "需要租户管理员"},
				},
			},
		},
		"/api/git-oauth/tenant-connection/{tid}/gitlab-oauth-connection/": map[string]any{
			"get": map[string]any{
				"summary":     "获取租户 GitLab OAuth 连接（脱敏，legacy）",
				"deprecated":  true,
				"operationId": "getTenantGitlabOAuthConnectionLegacy",
				"parameters":  []any{tidParam, xUserID},
				"responses": map[string]any{
					"200": map[string]any{"description": "连接信息（无 client_secret）"},
					"401": map[string]any{"description": "未认证"},
					"403": map[string]any{"description": "非租户成员"},
				},
			},
			"put": map[string]any{
				"summary":     "创建或更新租户 GitLab OAuth 连接（legacy）",
				"deprecated":  true,
				"operationId": "putTenantGitlabOAuthConnectionLegacy",
				"parameters": []any{
					tidParam, xUserID,
					map[string]any{"name": "X-Is-Admin", "in": "header", "schema": map[string]any{"type": "string"}},
				},
				"requestBody": putBody,
				"responses": map[string]any{
					"200": map[string]any{"description": "已保存"},
					"403": map[string]any{"description": "需要租户管理员"},
				},
			},
			"delete": map[string]any{
				"summary":     "删除租户 GitLab OAuth 连接（legacy）",
				"deprecated":  true,
				"operationId": "deleteTenantGitlabOAuthConnectionLegacy",
				"parameters":  []any{tidParam, xUserID},
				"responses": map[string]any{
					"204": map[string]any{"description": "已删除"},
					"403": map[string]any{"description": "需要租户管理员"},
				},
			},
		},
		"/api/git-oauth/tenant-connection/tenant_id/{tid}/reachability/": map[string]any{
			"get": map[string]any{
				"summary":     "探测已保存的自建 GitLab 是否从平台可达（内网则跳过）",
				"operationId": "getTenantGitlabOAuthReachability",
				"parameters":  []any{tidParam, xUserID},
				"responses": map[string]any{
					"200": map[string]any{"description": "reachable / skipped_intranet / unconfigured"},
					"401": map[string]any{"description": "未认证"},
					"403": map[string]any{"description": "非租户成员"},
				},
			},
		},
		"/api/internal/git-oauth/gitlab-tenant-connection/": map[string]any{"get": map[string]any{
			"summary":     "内部：按 company_id 返回租户 GitLab 连接（catalog 合并，无 secret）",
			"operationId": "internalTenantGitlabOAuthConnection",
			"parameters": []any{
				bridgeHeader,
				map[string]any{"name": "company_id", "in": "query", "required": true, "schema": map[string]any{"type": "string"}},
			},
			"responses": map[string]any{
				"200": map[string]any{"description": "公开字段"},
				"401": map[string]any{"description": "bridge secret 无效"},
			},
		}},
	}
}
