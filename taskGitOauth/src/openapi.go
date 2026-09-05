package main

import "net/http"

func openAPIDocument() map[string]any {
	bridgeHeader := map[string]any{
		"name":        "X-GitOauth-Bridge-Secret",
		"in":          "header",
		"required":    true,
		"schema":      map[string]any{"type": "string"},
		"description": "Shared bridge secret (same as task2appSsoJwtSecret / GITOAUTH_BRIDGE_JWT_SECRET).",
	}
	xUserID := map[string]any{
		"name":     "X-User-Id",
		"in":       "header",
		"required": true,
		"schema":   map[string]any{"type": "string"},
	}
	errSchema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"detail": map[string]any{"type": "string"}},
	}
	accessReq := map[string]any{
		"type":     "object",
		"required": []string{"user_id"},
		"properties": map[string]any{
			"user_id":        map[string]any{"type": "integer", "format": "int64"},
			"provider_key":   map[string]any{"type": "string", "example": "github:github-official"},
			"github_user_id": map[string]any{"type": "string"},
			"audit": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action":       map[string]any{"type": "string"},
					"company_id":   map[string]any{"type": "integer", "format": "int64"},
					"workspace_id": map[string]any{"type": "integer", "format": "int64"},
					"detail":       map[string]any{"type": "object"},
				},
			},
		},
	}
	accessResp := map[string]any{
		"type":     "object",
		"required": []string{"access_token"},
		"properties": map[string]any{
			"access_token":   map[string]any{"type": "string"},
			"cached":         map[string]any{"type": "boolean"},
			"expires_in":     map[string]any{"type": "integer"},
			"github_user_id": map[string]any{"type": "string"},
		},
	}
	summaryResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"connected":      map[string]any{"type": "boolean"},
			"github_user_id": map[string]any{"type": "string", "nullable": true},
			"github_login":   map[string]any{"type": "string", "nullable": true},
			"scope":          map[string]any{"type": "string", "nullable": true},
			"bind_status":    map[string]any{"type": "string", "nullable": true},
			"bind_error":     map[string]any{"type": "string", "nullable": true},
			"connections":    map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
		},
	}
	jsonBody := func(schemaRef string) map[string]any {
		return map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/" + schemaRef},
				},
			},
		}
	}
	jsonResp := func(code, schemaRef, desc string) map[string]any {
		return map[string]any{
			code: map[string]any{
				"description": desc,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{"$ref": "#/components/schemas/" + schemaRef},
					},
				},
			},
		}
	}
	mergeResp := func(maps ...map[string]any) map[string]any {
		out := map[string]any{}
		for _, m := range maps {
			for k, v := range m {
				out[k] = v
			}
		}
		return out
	}

	internalAccess := map[string]any{
		"post": map[string]any{
			"summary":     "按用户换发 ephemeral access_token",
			"operationId": "accessForUser",
			"parameters":  []any{bridgeHeader},
			"requestBody": jsonBody("AccessForUserRequest"),
			"responses": mergeResp(
				jsonResp("200", "AccessForUserResponse", "换票成功"),
				jsonResp("401", "Error", "缺少或错误的 Bridge Secret"),
				jsonResp("404", "Error", "无 active 凭据"),
			),
		},
	}
	internalSummary := map[string]any{
		"post": map[string]any{
			"summary":     "用户绑定摘要",
			"operationId": "summaryForUser",
			"parameters":  []any{bridgeHeader},
			"requestBody": jsonBody("SummaryForUserRequest"),
			"responses": mergeResp(
				jsonResp("200", "SummaryForUserResponse", "摘要"),
				jsonResp("401", "Error", "未授权"),
			),
		},
	}
	internalRefresh := map[string]any{
		"post": map[string]any{
			"summary":     "用 refresh_token 换 access",
			"operationId": "refreshToken",
			"parameters":  []any{bridgeHeader},
			"requestBody": map[string]any{
				"required": true,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"type":     "object",
							"required": []string{"refresh_token"},
							"properties": map[string]any{
								"refresh_token": map[string]any{"type": "string"},
								"provider_key":  map[string]any{"type": "string"},
							},
						},
					},
				},
			},
			"responses": mergeResp(
				map[string]any{"200": map[string]any{
					"description": "OK",
					"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"access_token":  map[string]any{"type": "string"},
								"refresh_token": map[string]any{"type": "string"},
							},
						},
					}},
				}},
				jsonResp("401", "Error", "未授权"),
			),
		},
	}
	startGW := map[string]any{
		"get": map[string]any{
			"summary":     "Gateway 启动 OAuth（返回 authorize_url）",
			"operationId": "startFromGateway",
			"parameters": []any{
				xUserID,
				map[string]any{"name": "next", "in": "query", "schema": map[string]any{"type": "string"}},
				map[string]any{"name": "return_key", "in": "query", "schema": map[string]any{"type": "string"}},
				map[string]any{"name": "service_provider", "in": "query", "schema": map[string]any{"type": "string"}},
				map[string]any{"name": "allowed_host", "in": "query", "schema": map[string]any{"type": "string"}},
				map[string]any{"name": "repo_url", "in": "query", "schema": map[string]any{"type": "string"}},
				map[string]any{"name": "grant_kind", "in": "query", "schema": map[string]any{"type": "string"}, "description": "project | comment | pending"},
				map[string]any{"name": "grant_id", "in": "query", "schema": map[string]any{"type": "string"}},
			},
			"responses": map[string]any{
				"200": map[string]any{
					"description": "authorize_url",
					"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{
							"type":     "object",
							"required": []string{"authorize_url"},
							"properties": map[string]any{
								"authorize_url": map[string]any{"type": "string", "format": "uri"},
							},
						},
					}},
				},
				"503": map[string]any{"description": "missing_x_user_id / bad_state", "content": map[string]any{
					"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Error"}},
				}},
			},
		},
	}
	health := map[string]any{
		"get": map[string]any{
			"summary":     "健康检查",
			"operationId": "health",
			"responses": map[string]any{
				"200": map[string]any{
					"description": "OK",
					"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{"$ref": "#/components/schemas/Health"},
					}},
				},
				"503": map[string]any{"description": "DB unavailable"},
			},
		},
	}

	providersResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"providers": map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/components/schemas/ProviderCatalogEntry"},
			},
		},
	}

	paths := map[string]any{
		"/api/health/": health,
		"/api/git-oauth/providers/": map[string]any{"get": map[string]any{
			"summary":     "Git OAuth provider catalog（替换 Django GitOauthProvidersCatalogView）",
			"operationId": "listGitOauthProviders",
			"parameters": []any{map[string]any{
				"name": "company_id", "in": "query", "required": false,
				"schema":      map[string]any{"type": "string"},
				"description": "可选：按公司/租户过滤",
			}},
			"responses": map[string]any{
				"200": map[string]any{
					"description": "提供者列表",
					"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{"$ref": "#/components/schemas/ProviderCatalogResponse"},
					}},
				},
			},
		}},
		"/api/git-oauth/github-start/": map[string]any{"get": map[string]any{
			"summary": "GitHub OAuth 启动（Bridge JWT query token）",
			"parameters": []any{map[string]any{
				"name": "token", "in": "query", "required": true,
				"schema": map[string]any{"type": "string"},
			}},
			"responses": map[string]any{"302": map[string]any{"description": "Redirect to GitHub authorize"}},
		}},
		"/api/git-oauth/github-start-from-gateway/": startGW,
		"/api/git-oauth/github-app-start/":          startGW,
		"/api/git-oauth/github-callback/": map[string]any{"get": map[string]any{
			"summary":   "GitHub OAuth 回调（扁平化路由）",
			"responses": map[string]any{"302": map[string]any{"description": "Redirect to frontend"}},
		}},
		"/api/git-oauth/gitlab-start/": map[string]any{"get": map[string]any{
			"summary":   "GitLab OAuth 启动（Bridge JWT）",
			"responses": map[string]any{"302": map[string]any{"description": "Redirect to GitLab authorize"}},
		}},
		"/api/git-oauth/gitlab-start-from-gateway/": startGW,
		"/api/git-oauth/gitlab-app-start/":          startGW,
		"/api/git-oauth/gitlab-callback/": map[string]any{"get": map[string]any{
			"summary":   "GitLab OAuth 回调（扁平化路由）",
			"responses": map[string]any{"302": map[string]any{"description": "Redirect to frontend"}},
		}},
		// 浏览器回调契约路径（RegisterRoutes 注册 /api/accounts/ 动态分发，勿再扁平化——
		// 8c47b8a 曾扁平化致回调 404，见 handleAccountsDynamic）
		"/api/accounts/{service_provider}/oauth/callback/": map[string]any{"get": map[string]any{
			"summary": "按 service_provider 动态回调（浏览器回调契约，保留）",
			"parameters": []any{map[string]any{
				"name": "service_provider", "in": "path", "required": true,
				"schema": map[string]any{"type": "string"},
			}},
			"responses": map[string]any{
				"302": map[string]any{"description": "Redirect"},
				"404": map[string]any{"description": "unknown provider"},
				"409": map[string]any{"description": "ambiguous provider"},
			},
		}},
		// v2 浏览器回调契约（SSOT redirect_uri，ResolveProviderByGitsite 分发）
		"/redirect/gitsite/{gitsite}/oauth/callback/": map[string]any{"get": map[string]any{
			"summary": "v2 浏览器回调契约（按 gitsite 解析 provider 分发）",
			"parameters": []any{map[string]any{
				"name": "gitsite", "in": "path", "required": true,
				"schema": map[string]any{"type": "string"},
			}},
			"responses": map[string]any{
				"302": map[string]any{"description": "Redirect to frontend"},
				"404": map[string]any{"description": "unknown gitsite"},
			},
		}},
		"/api/internal/gitsite/{gitsite}/oauth/access-for-user/": map[string]any{"post": map[string]any{
			"summary": "内部换票按 Git site 寻址（v97 target，OPT-20260822-036）",
			"parameters": []any{bridgeHeader, map[string]any{
				"name": "gitsite", "in": "path", "required": true,
				"schema": map[string]any{"type": "string"},
			}},
			"requestBody": jsonBody("AccessForUserRequest"),
			"responses": mergeResp(
				jsonResp("200", "Ok", "已换取 access_token（仅服务间，禁回浏览器）"),
				jsonResp("404", "NotFound", "not_found / 未识别的 Git 站点"),
				jsonResp("401", "Unauthorized", "bridge secret 拒绝"),
			),
		}},
		"/api/internal/git-oauth/grant-ticket/consume/": map[string]any{"post": map[string]any{
			"summary":     "一次性消费 pending grant_ticket（创建评论/自动运行前）",
			"parameters":  []any{bridgeHeader},
			"requestBody": jsonBody("GrantTicketConsumeRequest"),
			"responses": mergeResp(
				jsonResp("200", "Ok", "consumed"),
				jsonResp("404", "NotFound", "ticket 不存在或已消费"),
				jsonResp("401", "Unauthorized", "bridge secret 拒绝"),
			),
		}},
		"/api/internal/git-oauth/github-access-for-user/":    internalAccess,
		"/api/internal/git-oauth/gitlab-access-for-user/":    internalAccess,
		"/api/internal/git-oauth/github-refresh/":            internalRefresh,
		"/api/internal/git-oauth/gitlab-refresh/":            internalRefresh,
		"/api/internal/git-oauth/github-credential-summary/": internalSummary,
		"/api/internal/git-oauth/gitlab-credential-summary/": internalSummary,
		"/api/internal/git-oauth/github-credential-delete/": map[string]any{"post": map[string]any{
			"summary": "删除用户凭据", "parameters": []any{bridgeHeader},
			"requestBody": jsonBody("DeleteCredentialRequest"),
			"responses":   mergeResp(jsonResp("200", "OkDeleted", "deleted"), jsonResp("401", "Error", "未授权")),
		}},
		"/api/internal/git-oauth/gitlab-credential-delete/": map[string]any{"post": map[string]any{
			"summary": "删除用户凭据", "parameters": []any{bridgeHeader},
			"requestBody": jsonBody("DeleteCredentialRequest"),
			"responses":   mergeResp(jsonResp("200", "OkDeleted", "deleted"), jsonResp("401", "Error", "未授权")),
		}},
		"/api/internal/git-oauth/github-credential-user-ids/": map[string]any{"get": map[string]any{
			"summary": "已绑定用户 ID 列表", "parameters": []any{bridgeHeader,
				map[string]any{"name": "provider_key", "in": "query", "schema": map[string]any{"type": "string"}},
			},
			"responses": map[string]any{"200": map[string]any{
				"description": "OK",
				"content": map[string]any{"application/json": map[string]any{
					"schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"user_ids": map[string]any{"type": "array", "items": map[string]any{"type": "integer", "format": "int64"}},
						},
					},
				}},
			}},
		}},
		"/api/internal/git-oauth/gitlab-credential-user-ids/": map[string]any{"get": map[string]any{
			"summary": "已绑定用户 ID 列表", "parameters": []any{bridgeHeader},
			"responses": map[string]any{"200": map[string]any{"description": "OK"}},
		}},
		"/api/internal/git-oauth/github-token-use-report/": map[string]any{"post": map[string]any{
			"summary": "token 使用审计", "parameters": []any{bridgeHeader},
			"responses": map[string]any{"200": map[string]any{"description": "OK"}},
		}},
		"/api/internal/git-oauth/gitlab-token-use-report/": map[string]any{"post": map[string]any{
			"summary": "token 使用审计", "parameters": []any{bridgeHeader},
			"responses": map[string]any{"200": map[string]any{"description": "OK"}},
		}},
		"/api/internal/git-oauth/github-audit-report/": map[string]any{"post": map[string]any{
			"summary": "任务级凭据审计", "parameters": []any{bridgeHeader},
			"responses": map[string]any{"200": map[string]any{"description": "OK"}},
		}},
		"/api/internal/git-oauth/gitlab-audit-report/": map[string]any{"post": map[string]any{
			"summary": "任务级凭据审计", "parameters": []any{bridgeHeader},
			"responses": map[string]any{"200": map[string]any{"description": "OK"}},
		}},
		"/api/git-oauth/user-app-connection/": map[string]any{
			"get": map[string]any{
				"summary":     "获取用户应用连接状态（含绑定/未绑定）",
				"operationId": "getUserAppConnection",
				"parameters": []any{
					xUserID,
					map[string]any{"name": "service_provider", "in": "query", "schema": map[string]any{"type": "string"}, "description": "服务提供方标识，缺省 \"default\"；凭据按配置 service_provider 存储，状态检查会自动扩展配置存储键"},
					map[string]any{"name": "provider", "in": "query", "schema": map[string]any{"type": "string"}},
					map[string]any{"name": "repo_url", "in": "query", "schema": map[string]any{"type": "string"}},
					map[string]any{
						"name":        "probe_access_token",
						"in":          "query",
						"required":    false,
						"schema":      map[string]any{"type": "boolean"},
						"description": "为 true 时对已绑定凭据实时校验 AccessToken（cache/refresh），响应含 access_token_valid 且永不返回 access_token；未绑定不调用上游。GitLab 另含 network_status（ok / unreachable / skipped_intranet）：非内网不可达时不得仅凭 cache 当作已绑定",
					},
				},
				"responses": map[string]any{
					"200": map[string]any{"description": "连接状态"},
					"401": map[string]any{"description": "未认证"},
				},
			},
			"delete": map[string]any{
				"summary":     "删除用户应用连接（解绑）",
				"operationId": "deleteUserAppConnection",
				"parameters": []any{
					xUserID,
					map[string]any{"name": "service_provider", "in": "query", "schema": map[string]any{"type": "string"}, "description": "服务提供方标识，缺省 \"default\""},
				},
				"responses": map[string]any{
					"200": map[string]any{"description": "OK"},
					"401": map[string]any{"description": "未认证"},
				},
			},
		},
	}
	for _, extra := range []map[string]any{
		openAPITenantConnectionPaths(xUserID, bridgeHeader),
		openAPIMergeRequestPaths(),
	} {
		for k, v := range extra {
			paths[k] = v
		}
	}

	_ = accessReq
	_ = accessResp
	_ = summaryResp
	_ = errSchema
	_ = providersResp

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "gitOauth API (taskGitOauth)",
			"description": "Browser OAuth + internal token APIs. Internal routes require X-GitOauth-Bridge-Secret.",
			"version":     "1.1.0",
		},
		"servers": []any{
			map[string]any{"url": "http://127.0.0.1:8002", "description": "local runAll"},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"Error":                 errSchema,
				"AccessForUserRequest":  accessReq,
				"AccessForUserResponse": accessResp,
				"GrantTicketConsumeRequest": map[string]any{
					"type":     "object",
					"required": []string{"id", "user_id", "gitsite"},
					"properties": map[string]any{
						"id":      map[string]any{"type": "string"},
						"user_id": map[string]any{"type": "string"},
						"gitsite": map[string]any{"type": "string"},
					},
				},
				"SummaryForUserRequest": map[string]any{
					"type":     "object",
					"required": []string{"user_id"},
					"properties": map[string]any{
						"user_id":      map[string]any{"type": "integer", "format": "int64"},
						"provider_key": map[string]any{"type": "string"},
					},
				},
				"SummaryForUserResponse": summaryResp,
				"DeleteCredentialRequest": map[string]any{
					"type":     "object",
					"required": []string{"user_id"},
					"properties": map[string]any{
						"user_id":        map[string]any{"type": "integer", "format": "int64"},
						"provider_key":   map[string]any{"type": "string"},
						"github_user_id": map[string]any{"type": "string"},
					},
				},
				"OkDeleted": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":            map[string]any{"type": "boolean"},
						"deleted_count": map[string]any{"type": "integer"},
					},
				},
				"Health": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"service":                      map[string]any{"type": "string"},
						"ok":                           map[string]any{"type": "boolean"},
						"checks":                       map[string]any{"type": "object"},
						"public_base_url":              map[string]any{"type": "string"},
						"gitlab_provider_config_count": map[string]any{"type": "integer"},
					},
				},
				"ProviderCatalogEntry": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"provider":         map[string]any{"type": "string"},
						"service_provider": map[string]any{"type": "string"},
						"provider_key":     map[string]any{"type": "string"},
						"website":          map[string]any{"type": "string"},
						"label":            map[string]any{"type": "string"},
					},
				},
				"ProviderCatalogResponse": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"providers": map[string]any{
							"type":  "array",
							"items": map[string]any{"$ref": "#/components/schemas/ProviderCatalogEntry"},
						},
					},
				},
			},
			"securitySchemes": map[string]any{
				"BridgeSecret": map[string]any{
					"type": "apiKey",
					"in":   "header",
					"name": "X-GitOauth-Bridge-Secret",
				},
			},
		},
		"paths": paths,
	}
}

func (a *App) handleSwagger(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, openAPIDocument())
}
