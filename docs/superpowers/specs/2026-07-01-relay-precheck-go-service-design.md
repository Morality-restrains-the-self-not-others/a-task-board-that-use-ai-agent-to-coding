# 设计文档：容器令牌接口 + 数据库表 独立为 Go 服务

**日期：** 2026-07-01  
**状态：** 设计中  
**页面：** `http://183.250.1.132:4000/tenant/.../task-detail/.../?relayToTrae=true`

---

## 现象

用户点击「启动」后：
> 仓库克隆凭证预检失败。任务 API 不可达或网关异常；请确认 Django 服务（如 8001）已运行。

## 根因

`runall-saas-backend.sh` 以 `--noreload` 启动单线程 Django。`relay_to_trae_repo_credentials_precheck()` 对自身发起同步 HTTP 调用 `http://127.0.0.1:8001/.../repo-clone-credentials/`，单线程死锁，8 秒超时返回 502。

---

## 方案：容器运行时基础设施整体迁移到 Go 服务

**核心决策：** 不仅迁移端点，**连数据库表也迁移**。Go 服务成为容器令牌与运行时状态的唯一 owner，Django 清理全部相关代码。

### 数据库表迁移

#### 表 1：`cloud_cloudserverconfig` → Go `container_tokens`

这是容器运行时的核心表，存储容器令牌与服务端点配置。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | TEXT PK | 雪花 ID |
| `task_id` | TEXT NOT NULL | 关联任务 |
| `company_id` | TEXT | 租户 ID |
| `workspace_id` | TEXT | 工作空间 ID |
| `container_access_token` | TEXT | 容器访问令牌 |
| `container_access_token_expires_at` | DATETIME | 令牌过期时间 |
| `container_refresh_token` | TEXT | 刷新令牌 |
| `server_url` | TEXT | 容器服务地址 |
| `business_api_endpoint` | TEXT | 业务 API 端点 |
| `container_vscode_url` | TEXT | VS Code URL |
| `authorization_id` | TEXT | 云平台授权 ID |
| `image_id` | TEXT | 镜像 ID |
| `instance_type` | TEXT | 实例类型 |
| `region_id` | TEXT | 区域 |
| `zone_id` | TEXT | 可用区 |
| `created_at` / `updated_at` | DATETIME | 时间戳 |

**迁移方式：** Go 服务用 `database/sql` + `modernc.org/sqlite` 创建独立 DB 文件 `db/container/tokens.sqlite3`，通过 `db/registry.yaml` 注册。

**Django 侧：** 删除 `CloudServerConfig` Model、删除对应 migration、删除所有 `CloudServerConfig.objects.*` 查询。

#### 表 2：`cloud_container_token_audit_event` → Go `token_audit_events`

令牌审计事件表。Phase 1 仅做 schema 定义，数据迁移后续。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | TEXT PK | 雪花 ID |
| `task_id` | TEXT | 关联任务 |
| `event_type` | TEXT | 事件类型枚举 |
| `access_token_sha256` | TEXT | 令牌哈希（审计用，不存明文） |
| `prev_access_token_sha256` | TEXT | 前值哈希 |
| `new_access_token_sha256` | TEXT | 新值哈希 |
| `refresh_token_sha256` | TEXT | 刷新令牌哈希 |
| `source_component` | TEXT | 来源组件 |
| `trace_id` | TEXT | 链路追踪 ID |
| `error_code` | TEXT | 错误分类 |
| `error_detail` | TEXT | 错误详情 |
| `seq` | INTEGER | 事件序号 |
| `created_at` | DATETIME | 创建时间 |

#### 表 3：`cloud_cloudserverconfighistory` → Go `server_config_history`

配置历史表（如有引用则需要迁移）。

#### Django 保留（Go 只读）的表

这些是业务表，Django 继续持有写权限，Go 服务只读查询：

| 表 | 用途 | Go 访问方式 |
|----|------|-----------|
| `projects_todo` | 任务主体 | 查 title/description/parameters |
| `projects_taskproject` | 任务-项目关联 | 查 project_id |
| `projects_projectrepo` | 仓库 URL | 查 repo_url |
| `projects_taskrepoidentity` | 仓库身份绑定 | 查 repo_url + git_identity_id |
| `accounts_gitidentity` | Git 身份 | 查 user_id |
| `projects_taskgithubrepooauthbinding` | GitHub OAuth 绑定 | 查 github_user_id + repo_slug |
| `projects_tenant_feature_params` | 功能参数 | 查 agent_max_steps 等 |

---

### 端点迁移全景

全部挂载于 `/api/tenant/{tid}/workspace/{wid}/task/{tid}/cloud/`，按迁移批次分四组：

#### 组 A：数据查询（只读 saas.sqlite3，调 gitOauth）

| # | 端点 | 当前 Django View（待删除） | 调用方 |
|---|------|--------------------------|--------|
| A1 | `server-container-token/repo-clone-credentials/` | `fetch_container_repo_clone_credentials` | Django self-call ← 死锁、onlineServiceJS、taskAgentSupport |
| A2 | `server-container-token/task-detail/` | `fetch_container_task_detail` | onlineServiceJS、taskAgentSupport |
| A3 | `server-container-token/layer-github-oauth-access-tokens/` | `fetch_container_layer_github_oauth_access_tokens` | onlineServiceJS、taskAgentSupport |

#### 组 B：令牌生命周期（写 tokens.sqlite3）

| # | 端点 | 当前 Django View（待删除） | 调用方 |
|---|------|--------------------------|--------|
| B1 | `server-container-token/exchange-refresh/` | `exchange_server_container_refresh_token` | Go relay、onlineServiceJS、taskAgentSupport |
| B2 | `server-container-token/refresh-access/` | `refresh_server_container_access_token` | Go relay、onlineServiceJS、taskAgentSupport |
| B3 | `POST /v1/token/init`（新增） | `_issue_relay_access_token()` in relay_to_trae_proxy.py | Django relay_to_trae_token_init |

#### 组 C：状态上报（写 tokens.sqlite3 或专属表）

| # | 端点 | 当前 Django View（待删除） |
|---|------|--------------------------|
| C1 | `server-container-token/register-reachability/` | `register_container_reachability` |
| C2 | `server-container-token/heartbeat/` | `report_container_heartbeat` |
| C3 | `server-container-token/git-clone-progress/` | `report_container_git_clone_progress` |
| C4 | `server-container-token/layer-changes-push/` | `report_container_layer_changes` |
| C5 | `server-container-token/layer-graph-push/` | `report_container_layer_graph` |
| C6 | `relay-to-trae/status-push/` | `report_relay_to_trae_status` |

#### 组 D：其他

| # | 端点 | 调用方 |
|---|------|--------|
| D1 | `server-container-token/feature-params-env/` | onlineServiceJS、taskAgentSupport |
| D2 | `model-budget-usage/` | taskAIEndPoint |
| D3 | `repo-reclone/` | onlineServiceJS、CloudComputeViewSet 转发 |
| D4 | `server-userdata-verify/<secret>/` | VM cloud-init |

#### 不迁移的端点

**`compute/relay-to-trae/*`（8 个浏览器端 relay 代理）** — 浏览器通过 Django 会话鉴权，非容器令牌，保留在 Django：

| 端点 | 说明 |
|------|------|
| `compute/relay-to-trae/env-prepare/` | 环境准备（读配置） |
| `compute/relay-to-trae/health/` | Go relay 健康检查代理 |
| `compute/relay-to-trae/register/` | 注册 relay 任务 |
| `compute/relay-to-trae/token-init/` | **改为调用 Go /v1/token/init** |
| `compute/relay-to-trae/repo-credentials-precheck/` | **改为调用 Go A1** |
| `compute/relay-to-trae/start/` | 启动代理（转发到 Go relay） |
| `compute/relay-to-trae/stop/` | 停止代理（转发到 Go relay） |
| `compute/relay-to-trae/status/` | 状态查询代理 |

---

### Go 服务架构

```
taskCredentialService/
├── build.sh
├── go.mod
├── src/
│   ├── main.go               # findMonorepoRoot → 配置 → 启动 HTTP server (:8015)
│   ├── config.go              # port, gitoauth_base, db paths
│   ├── handlers.go            # 路由注册 + 全部 HTTP handlers
│   ├── db.go                  # DB 初始化：打开 tokens.sqlite3 (rw) + saas.sqlite3 (ro)
│   ├── token.go               # 令牌签发/校验/交换/刷新
│   ├── credentials.go         # 仓库克隆凭证构建（调 gitOauth）
│   ├── task_detail.go         # 任务详情查询
│   ├── layer_oauth.go         # 层 OAuth token 查询
│   ├── status.go              # 状态上报（heartbeat / reachability / clone-progress / layer-push）
│   ├── feature_params.go      # 功能参数查询
│   ├── gitoauth.go            # gitOauth HTTP 客户端
│   ├── models.go              # 数据结构定义（token / credential / task / audit）
│   └── tracelog/              # OTel 集成
├── migrations/
│   └── 001_create_tables.sql  # DDL：container_tokens + token_audit_events
└── bin/
```

**端口：** `8015`（`TASK_CREDENTIAL_SERVICE_PORT`）  
**DB 文件：** `db/container/tokens.sqlite3`（Go 自有，读写）、`db/saas/saas.sqlite3`（Django 业务库，只读）

**`db/registry.yaml` 新增：**
```yaml
databases:
  container:
    path: db/container/tokens.sqlite3
    owner: task-credential-service
```

**`conf/container/task-credential-service/config.yaml`：**
```yaml
host: 127.0.0.1
port: 8015
gitoauth_base: http://127.0.0.1:8002
gitoauth_timeout_seconds: 5
access_token_expiry_seconds: 3600
```

---

### Django 清理清单

#### 删除的 Model（及对应 migration）

| Model | 文件 |
|-------|------|
| `CloudServerConfig` | `cloud/models.py` |
| `CloudServerConfigHistory` | `cloud/models.py` |
| `ContainerTokenAuditEvent` | `cloud/models.py` |

#### 删除的 View 文件

| 文件 | 说明 |
|------|------|
| `cloud/views/container_runtime_token_views.py` | exchange-refresh / refresh-access / register-reachability / heartbeat |
| `cloud/views/container_task_detail_views.py` | task-detail / repo-clone-credentials |
| `cloud/views/container_layer_github_oauth_views.py` | layer-github-oauth-access-tokens |
| `cloud/views/container_feature_params_views.py` | feature-params-env |
| `cloud/views/container_git_clone_progress_views.py` | git-clone-progress / layer-changes-push / layer-graph-push |
| `cloud/views/container_repo_reclone_views.py` | repo-reclone |
| `cloud/views/task_model_budget_usage_views.py` | model-budget-usage |
| `cloud/views/relay_to_trae_status_views.py` | relay-to-trae/status-push |
| `cloud/views/userdata_verify_views.py` | server-userdata-verify |

#### 修改的文件

| 文件 | 变更 |
|------|------|
| `cloud/instance_callback_urls.py` | **清空** — 所有 15 个 URL pattern 删除 |
| `cloud/urls.py` | 移除 `instance_callback_urls` 的 include |
| `cloud/services/relay_to_trae_proxy.py` | `_issue_relay_access_token()` → 调用 Go `POST /v1/token/init`；`_resolve_relay_precheck_task_api_origin()` → 改为 `get_task_credential_service_url()`；删除 `_is_token_exchange_in_progress()` 等内部逻辑 |
| `cloud/services/mock_run_container.py` | 替换 `CloudServerConfig.objects.*` 为 Go API 调用 |
| `cloud/views/cloud_compute_views.py` | `relay-to-trae/env-prepare` 注入 `TASK_CREDENTIAL_SERVICE_ORIGIN` 环境变量 |
| `cloud/container_machine_api_access_middleware.py` | 更新日志匹配正则（如有需要） |
| `core/config/settings_manager.py` | 新增 `get_task_credential_service_url()` |
| `conf/core/django/config.yaml` | 新增 `taskCredentialServiceBase: http://127.0.0.1:8015` |
| `saas_project/settings.py` | 移除 `cloud_cloudserverconfig` 等 app 引用（如独立 app） |

#### 删除的测试文件

| 文件 | 说明 |
|------|------|
| `tests/test_container_runtime_tokens.py` | Token 端点测试 → 迁移到 Go `handlers_test.go` |
| `tests/test_relay_to_trae_proxy.py`（部分） | 预检/Token 相关测试 → 迁移到 Go |
| `tests/test_relay_to_trae_status.py` | 状态推送测试 |
| `tests/test_container_token_audit_integration.py` | 审计集成测试 |
| `tests/test_container_token_audit_retention.py` | 审计保留测试 |
| `tests/test_relay_register_token_exchange_guard.py` | 换票保护测试 |

#### `cloud/domain/` 清理

| 文件/目录 | 说明 |
|-----------|------|
| `cloud/domain/entities/task_detail_read_model.py` | 读模型 → Go 侧等效结构 |
| `cloud/domain/events/task_detail_patched.py` | 如无用则删 |
| `cloud/domain/repositories/task_detail_read_model_repository.py` | Repository → Go 侧 |
| `cloud/domain/services/repo_clone_credentials_fetch_service.py` | 凭证服务 → Go `credentials.go` |
| `cloud/domain/services/task_repo_clone_credentials_guard_service.py` | Guard → Go |
| `cloud/domain/value_objects/access_token.py` | Token VO → Go |
| `cloud/domain/value_objects/container_token_context.py` | Context VO → Go |

---

### 调用链路变化

```
Before (Django 全包，单线程死锁):
═══════════════════════════════════════
Browser → 网关 → Django (:8001)
  ├─ relay-to-trae/token-init     → 写 CloudServerConfig
  ├─ relay-to-trae/precheck       → self-call repo-clone-credentials → DEADLOCK
  └─ relay-to-trae/start          → 转发 Go relay

Go relay (:8797)
  → exchange-refresh → Django (:8001) → 写 CloudServerConfig

onlineServiceJS
  → taskAgentSupport (:8011) → Django (:8001) → 读写 DB

After (Go 服务，无死锁):
════════════════════════════════
Browser → 网关 → Django (:8001)
  ├─ relay-to-trae/token-init     → Go (:8015) POST /v1/token/init
  ├─ relay-to-trae/precheck       → Go (:8015) POST .../repo-clone-credentials/
  └─ relay-to-trae/start          → 转发 Go relay

Go relay (:8797)
  → exchange-refresh → Go (:8015)

onlineServiceJS
  → taskAgentSupport (:8011) → Go (:8015)

Go (:8015)
  ├─ tokens.sqlite3 (rw)     ← 自有数据库
  └─ saas.sqlite3 (ro)       ← Django 业务数据
```

---

### runAll 集成

```yaml
# conf/runAll.yaml 新增
- name: task-credential-service
  conf_app: container/task-credential-service
  build_command: "./build.sh"
  start_command: "./bin/taskCredentialService"
  stop_command: "bash -c 'lsof -ti:8015 | xargs kill -9 2>/dev/null || true'"
  working_dir: taskCredentialService
  depends_on: [saas-backend, git-oauth]
  health_check:
    health_path: /health
    timeout: 30
    retries: 10
```

`depends_on: [saas-backend]` 确保 Django 先启动（完成 saas.sqlite3 的 migration），Go 服务后启动（只读打开 saas.sqlite3）。

`conf/runAll.yaml` `saas-backend` 条目的 `depends_on` 需移除 `task-sse`（删掉不存在的依赖，或保留为无影响）。

---

### 迁移策略

#### Phase 1（解决死锁）：组 A + 组 B3（token init）

- Go 服务实现 `tokens.sqlite3` 建表 + token 签发
- Go 服务实现 A1/A2/A3 三个查询端点
- Django 改为调 Go 服务做 token init + precheck
- **死锁消除** — Django 不再 self-call
- Go relay / onlineServiceJS / taskAgentSupport **暂不切换**（仍走 Django）

#### Phase 2（token 端点统一）：组 B1/B2

- Go 服务实现 exchange-refresh / refresh-access（写 tokens.sqlite3）
- Go relay → Go 服务（不再经 Django）
- onlineServiceJS → taskAgentSupport → Go 服务
- Django 清理 `container_runtime_token_views.py` + `CloudServerConfig` Model

#### Phase 3（状态上报迁移）：组 C + 组 D

- 全部 15 个端点由 Go 服务承载
- Django 删除全部 9 个 container view 文件
- Django 清空 `instance_callback_urls.py`
- 最终：Django 中不再有任何容器令牌相关代码和表

---

### 价值流影响

| 流 | 影响 |
|----|------|
| **新增** `task-credential-service-go` | Go 服务独立部署、独立 DB、独立健康检查 |
| `relay-precheck-internal-origin` | 预检 origin 从 `internalApiBase` → `taskCredentialServiceBase`；不再依赖 `django.internal_api_base` |
| `relay-token-audit-observability` | 审计表从 Django → Go，审计事件由 Go 发布 |
| `relay-token-audit-full-chain-eventization` | token 签发/交换/刷新全链路由 Go 管辖 |
| `task-detail-repo-clone-credentials-contract` | URL 契约不变，实现全部在 Go |
| `task-detail-oauth-binding-guidance` | 前端不变 |
| `task-detail-runtime-relay` | 直启死锁根除 |
| `relay-status-push-timeout-go-relay` | 状态推送直接到 Go 服务 |
| `relay-register-token-exchange-guard` | TEIP 保护逻辑迁移到 Go |
| `internal-api-timeout-governance` | 内网调用目标从 Django→Django 变为 Django→Go（新增服务间调用路径）|

---

### 领域概念

- **Bounded Context：** 容器运行时（Container Runtime）— 独立于 SaaS 业务
- **聚合根：** `ContainerToken`（tokens.sqlite3）— 令牌生命周期
- **实体：** `TaskTokenBinding`、`TokenAuditEvent`、`ServerConfigSnapshot`
- **值对象：** `AccessToken`、`RefreshToken`、`TaskScope`
- **领域服务：** `TokenIssuanceService`、`RepoCloneCredentialsService`、`LayerOAuthTokenService`
- **领域事件：** `TokenIssued`、`TokenExchanged`、`TokenRefreshed`、`CredentialsFetchFailed`
- **基础设施：** SQLite（tokens.sqlite3 rw + saas.sqlite3 ro）、gitOauth HTTP client

---

### 测试计划

#### Go 侧（新增）

| 测试文件 | 覆盖 |
|----------|------|
| `handlers_test.go` | 全部 HTTP handlers：200/401/403/409/502 |
| `token_test.go` | Token 签发/校验/过期/交换/刷新 |
| `credentials_test.go` | 凭证构建：正常/缺 identity/token 换发失败 |
| `db_test.go` | DB 初始化、migration、CRUD |
| `gitoauth_test.go` | gitOauth HTTP mock |
| `integration_test.go` | 全链路：token init → precheck → start |

#### Django 侧（修改/新增）

| 测试文件 | 变更 |
|----------|------|
| `test_relay_to_trae_proxy.py` | 验证预检调 Go 服务 URL |
| `test_port_config_merge.py` | 新增 `get_task_credential_service_url()` 测试 |
| 新增 `test_task_credential_service_integration.py` | Django↔Go 集成测试 |

#### E2E

| 测试 | 说明 |
|------|------|
| `TaskDetail.relay-to-trae-direct-start.playwright.test.js` | 直启全链路回归 |
| 新增 playwright 测试 | Go 服务不可达时的降级行为 |

---

### 风险

| 风险 | 缓解 |
|------|------|
| Go 服务 SQLite 写与 Django SQLite 读并发 | WAL 模式已启用；Go 写自己的 `tokens.sqlite3`，不写 `saas.sqlite3` |
| Django 删除 CloudServerConfig 后其他模块引用报错 | 全量 grep `CloudServerConfig` 确认所有引用点；Phase 2 才删除 Model |
| Go 服务未就绪时 relay 启动全链路不可用 | runAll `depends_on: [saas-backend, git-oauth]` 确保启动顺序；health check 阻塞后续 |
| taskAgentSupport 路由切换遗漏 | 逐个切换、逐批验证 |

---

### 验收标准

1. `db/container/tokens.sqlite3` 由 Go 服务自动创建（migration），表结构与 Django `CloudServerConfig` 语义等价
2. Go `POST /v1/token/init` 签发 token 并持久化到 tokens.sqlite3，返回 `access_token`
3. Phase 1 三个查询端点 (A1/A2/A3) 返回与当前 Django 等价 JSON
4. Django 预检调用 Go `:8015` 而非自身 `:8001` — 死锁消除，响应 <1s
5. Django 删除全部 container view 文件（9 个）和 `CloudServerConfig` 等 Model
6. `instance_callback_urls.py` 清空（或完全删除）
7. 已有 pytest 测试（排除已迁移部分）全部通过
8. Go 测试覆盖率 ≥ 80%