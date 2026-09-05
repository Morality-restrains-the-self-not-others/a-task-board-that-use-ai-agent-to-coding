# 实施计划：租户自建 GitLab 平台 OIDC SSO

- **日期**: 2026-08-25
- **设计 / 权限 / VS / NFR / DDD**: 同前缀 `2026-08-25-tenant-selfhosted-gitlab-oidc-sso-*`

每个任务可独立验证。TDD：先红后绿。

## Task 1 — 领域纯函数

- [x] `taskAuth/domain/tenant_gitlab_oidc_sso.go` + `_test.go`
- 覆盖：ClientID、RedirectURI、HTTPS、AuthorizeMembership fail-closed、snippet 含 discovery
- 验证：`cd taskAuth && go test ./domain -count=1`

## Task 2 — DDL

- [x] `dataMigrate/taskAuth/043_oidc_client_tenant_sso.sql`
- 列：`owner_company_id`、`purpose`；`managed_by` 允许 `tenant`；幂等表 `auth_oidc_sso_idempotency`
- `ensureOidcClient` 对 `tenant` 行不 UPDATE
- 验证：迁移测试 + `python3 -m py_compile` 不适用；SQL 走 migrate 测试夹具

## Task 3 — Authorize 成员闸门

- [x] 扩展 `loadOidcClient`；`handleOidcAuthorize` 在区域闸门之后对 tenant client 调 membership
- [x] `oidc_tenant_sso_gate_test.go`：成员发码 / 非成员 denied / 超管非成员 denied / 解析失败 denied / `gitlab-git-service` 不走本闸门
- 验证：`go test ./src -count=1 -run TenantGitLabOidc`

## Task 4 — CRUD + 事件 + OpenAPI + 网关

- [x] handlers、membership client、Path A client
- [x] `eventTopicMap` + `conf/events/domain-events/tenant_gitlab_oidc_sso_{enabled,secret_rotated,disabled}/event.yaml`
- [x] `taskAuth/src/openapi.yaml` 路径
- [x] `taskGateway/routes/routes.yaml` priority 868 → taskAuth
- [x] `db/api_route_ownership.yaml`
- [x] `043` 资源组成员
- 验证：handler 测；yaml.safe_load routes

## Task 5 — 前端

- [x] 拆分 `WorkspaceSettingsGitlabOidcSso.vue`（避免 gitlab-connection 超 500 行）
- [x] Path A 帮助折叠；SSO enable/rotate/disable + clickGuard
- 验证：vitest 相关文件

## Task 6 — 意图文档 + WSD

- [x] `docs/intents/backend/tenant_gitlab_oidc_sso.intent.md` + test-intent
- [x] `docs/intents/frontend/tenant_gitlab_oidc_sso.intent.md` + test-intent
- [x] 更新 `docs/flows/value-stream-test-integration.wsd`

## Task 7 — 精准重启登记

- [x] `scripts/register-precise-restart.sh task-auth taskFE task-gateway`（或写入 `.runall/precise_restart_services.txt`）
