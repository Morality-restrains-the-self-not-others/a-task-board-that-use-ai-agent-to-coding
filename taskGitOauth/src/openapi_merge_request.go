package main

func openAPIMergeRequestPaths() map[string]any {
	tidParam := map[string]any{
		"name": "tid", "in": "path", "required": true, "schema": map[string]any{"type": "string"},
	}
	xUserID := map[string]any{
		"name": "X-User-Id", "in": "header", "required": true, "schema": map[string]any{"type": "string"},
	}
	errResp := map[string]any{
		"401": map[string]any{"description": "未认证"},
		"403": map[string]any{"description": "非租户成员"},
		"400": map[string]any{"description": "非法 html_url / host"},
	}
	return map[string]any{
		"/api/git-oauth/merge-request-status/tenant_id/{tid}/": map[string]any{
			"post": map[string]any{
				"summary":     "批量查询 GitHub PR / GitLab MR 合并状态",
				"operationId": "mergeRequestStatus",
				"parameters":  []any{tidParam, xUserID},
				"requestBody": map[string]any{
					"required": true,
					"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{
							"type":     "object",
							"required": []string{"html_urls"},
							"properties": map[string]any{
								"html_urls": map[string]any{
									"type": "array", "maxItems": 20,
									"items": map[string]any{"type": "string", "format": "uri"},
								},
							},
						},
					}},
				},
				"responses": map[string]any{
					"200": map[string]any{"description": "各 URL 的 state（open/merged/closed/unknown）"},
					"401": errResp["401"],
					"403": errResp["403"],
					"400": errResp["400"],
				},
			},
		},
		"/api/git-oauth/merge-request-merge/tenant_id/{tid}/": map[string]any{
			"post": map[string]any{
				"summary":     "一键合并 GitHub PR / GitLab MR（审计 merge_request_merge）",
				"operationId": "mergeRequestMerge",
				"parameters":  []any{tidParam, xUserID},
				"requestBody": map[string]any{
					"required": true,
					"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{
							"type":     "object",
							"required": []string{"html_url"},
							"properties": map[string]any{
								"html_url":   map[string]any{"type": "string", "format": "uri"},
								"task_id":    map[string]any{"type": "string"},
								"comment_id": map[string]any{"type": "string"},
							},
						},
					}},
				},
				"responses": map[string]any{
					"200": map[string]any{"description": "已合并或已是 merged 的幂等成功"},
					"401": errResp["401"],
					"403": errResp["403"],
					"400": errResp["400"],
					"409": map[string]any{"description": "尚未绑定 Git 网站 OAuth"},
					"502": map[string]any{"description": "换取 Git 访问令牌或调用 Git API 失败"},
				},
			},
		},
	}
}
