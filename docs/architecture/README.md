# AI Dev Platform — 系统架构文档

> 状态: current | 版本: 1 | 更新: 2026-07-01 | 由 `/goal` 自动生成
>
> ⚠️ **本文为 v1 历史快照**（2026-07-01，Django 时代）。服务拓扑、端口、存储引擎均已演进：存储已全部 MySQL 化（SSOT：[`db/registry.yaml`](../../db/registry.yaml)），Django 于架构 v57（2026-07-30）退役。**权威现状以 `db/registry.yaml` + [table-to-owner.md](./table-to-owner.md) + [VERSION_HISTORY.md](./VERSION_HISTORY.md) + `conf/runAll.yaml` 为准**。SQLite 表述已按运行事实修正（docs-cleanup 2026-08-24）。
>
> 全站网址 / 接口 / 核心路径目录：[service-url-api-catalog.md](./service-url-api-catalog.md)（Grafana 浏览面 http://10.2.150.68:3000/d/service-url-api-catalog/service-url-api-catalog ；Grafana 不是 URL SSOT，见 [设计说明](../superpowers/specs/2026-08-27-service-url-api-catalog-design.md)）。

## 1. 系统概览

AI Dev Platform 是一个 AI 驱动的 SaaS 开发平台（Monorepo），支持用户创建项目、通过 AI Agent 自动完成开发任务、在容器中运行代码，并通过 Git OAuth 集成管理代码仓库。

### 关键指标

| 维度 | 数值 |
|------|------|
| 总服务数 | 39 个运行时组件 |
| 编程语言 | Go (23), Python/Django (4), Node.js (1), Vue.js (1) |
| 领域事件消费者 | 18 个 Go intent 级消费者 |
| 数据库 | 12 个 MySQL 库（task_task/task_ai_comment/task_auth/…，SSOT: `db/registry.yaml`） |
| 基础设施组件 | 6 个 Docker 服务 |
| 运行时端口范围 | 4000, 8001-8015, 8797, 9998, 18020-18037, 18080-18081 |

---

## 2. 分层架构

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend Layer                        │
│  Vue Frontend (:4000)                                    │
├─────────────────────────────────────────────────────────┤
│                    Gateway Layer                         │
│  task-gateway / APISIX (:18081)                          │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Forward Auth → task-auth                          │  │
│  │  Route Proxy → saas-backend, task-bill, task-sse   │  │
│  │  OIDC SSO (Logout sync)                            │  │
│  └───────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────┤
│                  Platform Services                       │
│  ┌──────────────┬──────────────┬──────────────────────┐ │
│  │ saas-backend │ ai-provider  │ task-auth (OIDC)     │ │
│  │ Django :8001 │ Django :8010 │ Go :8003             │ │
│  │ (已退役 v57) │ (已退役 v57) │ [MySQL task_auth]    │ │
│  ├──────────────┼──────────────┼──────────────────────┤ │
│  │ git-oauth    │ task-bill    │ task-sse             │ │
│  │ Go :8002     │ Go :8009     │ Node.js :8007        │ │
│  │ [MySQL]      │ [MySQL]      │                      │ │
│  ├──────────────┼──────────────┼──────────────────────┤ │
│  │ agent-support│ ai-endpoint  │                      │ │
│  │ Go :8011     │ Go :8013     │                      │ │
│  └──────────────┴──────────────┴──────────────────────┘ │
├─────────────────────────────────────────────────────────┤
│                    Container Stack                       │
│  ┌──────────────────┬──────────────────────────────┐   │
│  │ container-gw     │ credential-service           │   │
│  │ Go :8014         │ Go :8015 [MySQL container]   │   │
│  ├──────────────────┼──────────────────────────────┤   │
│  │ go-relay :8797   │ go-run-container             │   │
│  └──────────────────┴──────────────────────────────┘   │
├─────────────────────────────────────────────────────────┤
│              Domain Events (18 Go Consumers)             │
│  :18020-18021 Billing+SSE                               │
│  :18022-18026 Email+Invitation+Activation               │
│  :18025-18030 Registration Chain (User→Company→Ws)      │
│  :18027-18029 Company Fan-out                            │
│  :18030-18037 Cloud Server + AI Reply                   │
├─────────────────────────────────────────────────────────┤
│            Infrastructure (Docker)                       │
│  Redis | Kafka | GitLab (:8012) | Grafana+Loki           │
└─────────────────────────────────────────────────────────┘
```

---

## 3. 服务详细清单

### 3.1 Frontend Layer

| 服务 | 技术 | 端口 | 描述 |
|------|------|------|------|
| taskFE | Vue.js (Vite) | 4000 | SPA 前端，含 Vite proxy 转发 |

### 3.2 Gateway Layer

| 服务 | 技术 | 端口 | 描述 |
|------|------|------|------|
| task-gateway | APISIX (Docker) | 18081 | API 网关，forward-auth → task-auth，路由转发 |

### 3.3 Platform Services

| 服务 | 技术 | 端口 | DB | 描述 |
|------|------|------|-----|------|
| saas-backend | Django | 8001 | —（v57 退役） | 核心业务后端，用户/公司/项目/任务 CRUD |
| ai-provider | Go taskAiProvider | 8010 | MySQL ai_provider | AI Provider 管理，模型配置 |
| task-auth | Go | 8003 | MySQL task_auth | 认证服务 (OIDC Provider)，用户身份真源 |
| git-oauth | Go taskGitOauth | 8002 | MySQL git_oauth | Git OAuth 授权 (GitHub/GitLab 多 provider) |
| task-bill | Go | 8009 | MySQL task_bill | 计费服务 |
| task-sse | Node.js | 8007 | — | SSE 实时推送服务 |
| task-agent-support | Go | 8011 | — | AI Agent 辅助服务 |
| task-ai-endpoint | Go | 8013 | — | AI 端点服务 |

### 3.4 Container Stack

| 服务 | 技术 | 端口 | DB | 描述 |
|------|------|------|-----|------|
| task-container-gateway | Go | 8014 | — | 容器网关，验证/转发容器请求 |
| task-credential-service | Go | 8015 | MySQL container | 令牌管理，15 容器回调端点，审计事件 |
| go-relay | Go | 8797 | — | Relay to Trae，容器到 IDE 中继 |
| go-run-container | Go | dynamic | — | 容器生命周期管理 |

### 3.5 Domain Events (18 Go Consumers)

| 事件域 | 端口范围 | 消费者数 | 描述 |
|--------|----------|----------|------|
| Billing + SSE | :18020-18021 | 2 | 账单处理、SSE 消息 |
| Notifications | :18022-18026 | 5 | 邮件、邀请、激活通知 |
| Registration Chain | :18025-18030 | 6 | USER→COMPANY→WORKSPACE 级联 |
| Company Fan-out | :18027-18029 | 3 | 公司创建后的扇出处理 |
| Cloud + AI | :18030-18037 | 8 | 云服务器启停、AI 回复持久化 |

### 3.6 Infrastructure (Docker)

| 服务 | 端口 | 用途 |
|------|------|------|
| Redis | 6379 | SSE pub/sub、读旁路/版本戳（membership_rev）、短暂会话；不作新领域事件总线 |
| Kafka | 9092, UI:18080 | 领域事件总线 (domainEvents transport) |
| GitLab CE | 8012 | 自托管 Git 服务 |
| Grafana | 3000 | 监控面板 (Trace Log Journey/Explore) |
| Loki | 3100 | 日志聚合，按 trace_id 检索 |
| Promtail | — | 日志采集 → Loki |

---

## 4. 数据流与依赖关系

### 4.1 请求链路

```
Browser → Vue (:4000) → APISIX (:18081) → saas-backend (:8001)
                                              ├── task-auth (:8003)     [认证]
                                              ├── git-oauth (:8002)     [Git 授权]
                                              ├── task-sse (:8007)      [实时推送]
                                              └── task-credential (:8015) [令牌]
```

### 4.2 认证链路

```
APISIX forward-auth → task-auth (:8003)
  ├── Token 验证 (MySQL task_auth)
  ├── userId Cookie 回退 (第4认证方式)
  ├── OIDC Provider (OIDC SSO)
  └── → 注入 X-User-Id header → saas-backend
```

### 4.3 领域事件流

```
Django saas-backend              Go task-events consumers
  │ publish event                      │ consume from Kafka
  │ (USER_CREATED, etc.)              │
  ├──────────────────────────────────►│ USER_CREATED→0: create_company (:18025)
  │                                   │ COMPANY_CREATED→1: deliverable (:18027)
  │                                   │ COMPANY_CREATED→2: progress (:18028)
  │                                   │ COMPANY_CREATED→3: workspace (:18029)
  │                                   │ WORKSPACE_CREATED→1: process (:18030)
```

### 4.4 容器启动链路

```
task-detail UI
  → saas-backend (precheck via task-credential :8015)
    → task-credential-service (token init → MySQL container)
      → go-run-container (container lifecycle)
        → go-relay (:8797) (relay to Trae IDE)
          → task-credential-service (token audit events)
```

### 4.5 Git OAuth 链路

```
Project Detail → git-oauth (:8002)
  ├── GitHub App OAuth → GitHub API
  ├── GitLab OAuth → GitLab (:8012)
  ├── 多 provider 路由 (conf/auth/git-oauth/providers/)
  └── bind_status: pending → active → failed
```

---

## 5. 数据存储

> 存储引擎已全部 MySQL 化（2026-05~08 渐次完成；`db/registry.yaml` 全部 `driver: mysql`）。以下为摘要，**库名/owner 全量 SSOT 见 [`db/registry.yaml`](../../db/registry.yaml) + [table-to-owner.md](./table-to-owner.md)**。

| MySQL 库（database_key） | owner 服务 | 关键表（示例） |
|--------|------|--------|
| task_task | taskTaskService | todos/tasks 元数据、comments（人类评论） |
| task_ai_comment | taskAIComment | ai_task_comments（AI 指令评论）、容器 Agent 评论 |
| task_auth | taskAuth | accounts_user, login_method, customtoken |
| task_bill | taskBill | billing_* |
| task_project / task_cloud / task_tenant / task_referral / task_budget / git_oauth / ai_provider / container | 各 Go 服务 | 见 `db/table_ownership.yaml` |

> 设计原则: 用户身份真源仅在 task_auth 库，saas 侧不存储 User 表（通过 task-auth bridge 访问）；评论/任务「双库」分库模式保留（task_task + task_ai_comment，见 `docs/superpowers/specs/2026-07-15-task-comments-sql-vs-nosql-scale-design.md`）

**表 → owner 完整对照**（含 Go SSOT 库与 CI）：见 [table-to-owner.md](./table-to-owner.md)；机器可读 SSOT：`db/table_ownership.yaml`。

---

## 6. 业务领域映射

| 业务域 | 核心服务 | 价值流 |
|--------|----------|--------|
| 用户与认证 | task-auth, task-gateway | user-auth (注册/登录/激活/SSO) |
| 组织与成员 | saas-backend | company-management (CRUD/邀请/权限) |
| 项目与工作空间 | saas-backend | project-workspace, project-detail-oauth |
| 任务协作 | saas-backend, task-events | task-management, task-detail-runtime-relay |
| 云平台与资源 | container-stack, git-oauth | cloud-integration, oauth-token-fetch |
| 计费 | task-bill, task-events | billing (transaction processing) |
| 可观测性 | Grafana, Loki, Promtail | platform-centralized-logging |

---

## 7. 编排与部署

- **编排器**: runAll (Go binary, 端口动态)
- **配置文件**: `conf/runAll.yaml` — 定义 6 个服务组、依赖关系、健康检查
- **地址管理**: `conf/base.yaml` — SSOT 域名+协议寻址（`scheme`/`BASE_DOMAIN` + `${subdomains.*}`）
- **日志聚合**: runAll 日志 tee → Loki → Grafana (按 trace_id 检索)
- **远程部署**: screen session 管理 (`screen -S <name> -dm bash -c '...'`)

---

## 8. 文件索引

| 文件 | 格式 | 状态 | 用途 |
|------|------|------|------|
| `table-to-owner.md` | Markdown | ✅ current | 表→owner 对照与跨服务直连债务 |
| `v12-enterprise-landscape-20260805-1800-claude.puml` | PlantUML | ✅ current | 企业级全景图 v12（基线回填：全量 Go 服务 + Django 退役） |
| `v64-application-integration-20260805-1753-claude.puml` | PlantUML | ✅ current | 应用组件架构 v64（微信身份统一） |
| `v64-application-integration-20260805-1753-claude.archimate` | ArchiMate XML | — | v64 应用组件架构 (标准交换格式) |
| `v64-application-integration-20260805-1753-claude.mermaid.md` | Mermaid (Markdown) | — | v64 应用组件架构 (Mermaid 图表) |
| `v12-enterprise-landscape-20260805-1800-claude.archimate` | ArchiMate XML | — | v12 企业级全景图 (标准交换格式) |
| `v12-enterprise-landscape-20260805-1800-claude.mermaid.md` | Mermaid (Markdown) | — | v12 企业级全景图 (Mermaid 图表) |
| `VERSION_HISTORY.md` | Markdown | — | 架构版本演进时间线 |

> 文件命名: `v<N>-<视图名>-<YYYYMMDD-HHMM>-<作者>.puml` — 版本号前置，同版本多视图自然聚合；每个版本独立文件，老版本只读保留。

### 自动归档机制

- 主目录只保留每视图**最近 5 个版本**；提交新视图版本时由 `docs/.githooks/pre-commit` 自动把旧版本移入 `archive/`（只移动不删除，历史完整，详见 [VERSION_HISTORY.md](./VERSION_HISTORY.md)）
- 手动执行/预览: `bash architecture/scripts/archive-old-views.sh [--keep N] [--dry-run]`
- hook 安装: `bash architecture/scripts/install-arch-hooks.sh`（docs 子模块内提交时触发）

## 9. 架构版本历史

完整的版本演进记录见 [VERSION_HISTORY.md](./VERSION_HISTORY.md)。

当前架构版本:
- **enterprise-landscape v12** (2026-08-05 18:00, by claude) — 基线回填：全量 Go 微服务拓扑 + Django 退役 + RBAC/微信身份
- **application-integration v64** (2026-08-05 17:53, by claude) — 微信身份统一（target，待交付）

历史版本文件（只读）: 归档于 [archive/](./archive/)（保留最近 5 版，自动归档），完整时间线见 [VERSION_HISTORY.md](./VERSION_HISTORY.md)

---

## 10. 关键架构决策 (ADR)

1. **用户身份分离**: MySQL `task_auth` 库作为用户身份真源，saas 服务不持有 User 表
2. **领域事件异步化**: Django 发布事件 → Kafka → Go consumers 异步处理（Redis 不作领域事件总线）
3. **Token 审计完整链路**: relay register/start → task-credential-service → token_audit_events
4. **Gateway forward-auth 4 层认证回退**: Token → OIDC → Session → userId Cookie
5. **域名+协议 SSOT**: `conf/base.yaml` 驱动 `${scheme}` / `${subdomains.*}`；`PUBLIC_SCHEME`（默认 https）与 `BASE_DOMAIN` 可覆盖
6. **Go 服务迁移**: 令牌管理、凭证服务、领域事件从 Python 迁移至 Go 独立服务
