# 实施计划：租户级自建 GitLab OAuth 连接

**日期：** 2026-07-15  
**设计：** `docs/superpowers/specs/2026-07-15-tenant-gitlab-oauth-connection-design.md`

## Tasks

### T1 — 表与 ownership
- [ ] `taskGitOauth` migration：`tenant_gitlab_oauth_connections`
- [ ] 更新 `db/table_ownership.yaml`
- [ ] 验证：`go test` 打开 DB 后表存在

### T2 — Domain + CRUD handlers（TDD）
- [ ] 红：`TestTenantGitlabConnectionPutGetDelete`
- [ ] 绿：handlers + Fernet secret + UNIQUE
- [ ] `ensureTenantAdmin`（internal resolve-user-member，对齐 cloud）
- [ ] OpenAPI 条目
- [ ] 事件：Upserted / Deleted（Kafka 可选）
- [ ] 注册路由 `/api/tenant/{tid}/gitlab-oauth-connection/`

### T3 — Resolve 接入 OAuth
- [ ] `ResolveGitLabAuthorizeContext` / `ResolveByServiceProvider`：支持 `tenant-{company_id}` 从 DB 加载
- [ ] 动态 callback `tenant-gitlab` 与 `tenant-{id}` 均可
- [ ] DELETE 级联删 `api_gitoauthappusercredential` where provider_key

### T4 — Catalog 合并
- [ ] taskGitOauth internal GET connection by company_id
- [ ] Django `GitOauthProvidersCatalogView` 合并（扩展现有，无新 path）
- [ ] 前端 providers 请求带 tenantId

### T5 — 前端设置页
- [ ] Sidebar「GitLab 连接」
- [ ] `WorkspaceSettingsGitlabConnection.vue` + router
- [ ] 展示 redirect_uri 复制

### T6 — 意图 / 配置 / 网关
- [ ] `docs/intents/backend/tenant_gitlab_oauth_connection.*`
- [ ] `conf/domain-events/` 主题
- [ ] Gateway 路由（若需显式 upstream 到 :8002）
- [ ] `db/api_route_ownership.yaml`

### T7 — 验证
- [ ] `cd taskGitOauth && go test ./...`
- [ ] Archi loadModel v30.archimate
- [ ] 前端构建相关测（可选 Playwright 冒烟）

## 事件契约任务

- publish `TENANT_GITLAB_OAUTH_CONNECTION_UPSERTED` / `DELETED` 于 PUT/DELETE 成功路径
