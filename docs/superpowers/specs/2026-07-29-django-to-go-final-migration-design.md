# 设计文档：Django 服务盘点与最终 Go 迁移路线图

**日期：** 2026-07-29  
**状态：** 设计中  
**动机：** 盘点 monorepo 中所有仍使用 Django 的服务，制定将剩余 Django 进程完整迁移到 Go 的分阶段路线图。

---

## 🔍 盘点结果：当前 Django 服务状态

### 活跃 Django 进程

| # | 服务 | 目录 | 端口 | Python 文件 | 状态 |
|---|------|------|------|-------------|------|
| 1 | **saas-backend** | `task2app/Saas_project/` | :8001 | ~1,150 | ✅ **运行中**（gunicorn 2w/4t） |

### 已迁出 Django（Django 代码已删除或死代码）

| # | 服务 | 旧 Django 目录 | 新 Go 服务 | Go 目录 | 状态 |
|---|------|---------------|-----------|---------|------|
| 1 | **git-oauth** | `task2app/gitOauth/` | `taskGitOauth` (:8002) | `taskGitOauth/` | ✅ Django 目录已删除 |
| 2 | **ai-provider** | `task2app/Saas_Ai_Provider/` | `taskAiProvider` (:8010) | `taskAiProvider/` | ✅ Go binary 运行中，Django 97 文件待清理 |
| 3 | **relay-to-trae** | `relayToTrae/` | `go_relayToTrae` | `go_relayToTrae/` | ✅ Python 侧车已删除 |

> **结论：只剩 1 个 Django 进程需要迁移 — `saas-backend`（`task2app/Saas_project/`）。**

---

## 🏛️ saas-backend 剩余接口盘点

### 按 Django App 分类

#### 1. accounts/ — 用户与认证（218 .py 文件）

| 路由前缀 | 代表接口 | 流量特征 | 迁移复杂度 |
|----------|----------|----------|------------|
| `/api/auth/` | login, logout | 低频，短请求 | 低 — 转发 taskAuth |
| `/api/accounts/`、`/api/user/{uid}/accounts/` | users CRUD, profile, avatar, invitations | 中频，读多写少 | 高 — ORM 重，业务规则多 |
| `/api/accounts/sso/` | OIDC SSO 桥接 | 低频 | 中 — 与 taskAuth 协作 |
| `github/gitlab/oauth/start/` | OAuth 发起 | 低频 | 低 — 转发 taskGitOauth |
| `/api/user/{uid}/accounts/{github,gitlab}/app/connection/` | Git 连接状态查询 | 低频 | 低 — 读 taskGitOauth |
| `/api/internal/taskauth/` | enrich-login, forward-login, post-register | **已清空** — urlpatterns = [] | ✅ 已完成 |
| `/api/internal/session/resolve/` | Django session 查询 | 中频 internal | 中 — 需解决 session 存储共享 |
| `/api/internal/git-identities/lookup/` | Git identity 查询 | 中频 internal | 低 — 数据可迁 taskGitOauth |

#### 2. accounts/companies — 组织与租户

| 路由 | 业务 | 迁移复杂度 |
|------|------|------------|
| `companies/`, `members/`, `groups/` | 公司 CRUD, 成员管理 | 高 — 权限耦合深 |
| `admin/tenant-options/` | 超管租户选项 | 低 — 管理面 |

#### 3. projects/ — 项目与工作空间（228 .py 文件）

| 路由 | 业务 | 迁移状态 |
|------|------|----------|
| `/api/tenant/{tid}/projects/` | 项目 CRUD, branches, GitLab 批量 | 🔴 仍 Django |
| `/api/tenant/{tid}/workspaces/` | 工作空间 CRUD | 🟡 已迁 Go taskProjectService；Django 仅 cloud/platforms |
| `workspace-access/` | 权限、协作者 | 🔴 仍 Django |
| `todos/` | 任务 CRUD | 🔴 仍 Django（taskTaskService 存在，但 Django 仍持有 ORM） |
| `tasks/{id}/comments/` | 任务评论 | 🔴 仍 Django |
| `/api/internal/taskproject/*` | 10 个 internal 端点 | 🟡 部分可迁 taskProjectService |
| `/api/internal/task-ai-comment/import/` | 评论导入 | 🟡 可迁 taskAIComment |

#### 4. cloud/ — 云平台（425 .py 文件，最大 App）

| 路由 | 业务 | 迁移状态 |
|------|------|----------|
| `cloud/platforms/` | 云平台授权/配置 | 🔴 仍 Django（仅剩的路由） |
| `cloud/platforms/default-config/` | 默认配置 | 🔴 仍 Django |
| `server-startup-status/` | 启动进度 | 🟡 SSE 已迁 taskSSE |
| `oauth/aliyun/` | 阿里云 OAuth | 🟡 回调在 Go taskCloudService |
| `vpcs/`, `vswitches/`, `security-groups/` | 网络资源查询 | 🔴 仍 Django |
| `images/`, `server-images/` | 镜像管理 | 🔴 仍 Django |
| `.../repo-reclone/` | 重克隆 | 🔴 最后 2 个 container-inbound action |
| `.../server-userdata-verify/` | userdata 探针 | 🔴 |
| `/api/internal/task-agent-support/*` | internal scoped dispatch | 🟡 薄壳转发 |

#### 5. core/task_events/ — 事件意图（Kafka→HTTP Bridge）

| 路由 | 业务 | 迁移复杂度 |
|------|------|------------|
| `/api/internal/task-events/realtime/dispatch/` | SSE 消息分发 | 中 — 可迁 taskEvents |
| `/api/internal/task-events/intents/*` | 16 个 intent 端点 | **高优先** — 每个 intent 应改为 Go Kafka consumer |

#### 6. billing/ + billing_bridge/ — 计费（39 .py 文件）

| 路由 | 业务 | 迁移复杂度 |
|------|------|------------|
| `/api/tenant/{tid}/billing/*` | BillingProxyView → taskBill | 低 — 改 APISIX 直连 |
| `/api/internal/taskbill/*` | emit-billing-event, enrich | 低 — 可迁 taskBill |
| 订阅/支付回调 | Subscriptions, PayPal, SMS | 中 — 涉及外部支付 |

#### 7. cloudSystemAdmin/ + frontend_app/ — 管理面（62 .py 文件）

| 路由 | 业务 | 迁移复杂度 |
|------|------|------------|
| `/api/system-admin/*` | Dashboard, 用户管理 | 低 — 管理面低 QPS |
| `/api/system_admin/*/cloud/` | 云管理 | 低 |
| `manage-deliverable-system/` | 交付体系 | 低 — 可迁 taskProjectService |
| `manage-feature-params/` | 功能参数 | 🟡 部分已迁 |
| `manage-task-panels/` | 任务面板配置 | 低 |
| 隐私条款/服务协议 | 法律文本 | ✅ 已迁 taskBill |

#### 8. core/ — 横切（81 .py 文件）

| 路由 | 业务 | 迁移复杂度 |
|------|------|------------|
| `/api/health/`, `/api/live/` | 探活 | 保留 Django 轻量 — 不迁移 |
| `/api/public/catalog/` | 公开目录 | 低 — 可迁 taskAiProvider |
| Swagger `/swagger/` | API 文档 | 低 — 随接口迁移自然消亡 |

---

## 🎯 迁移策略：Strangler Fig 模式

### 核心原则

1. **不重写，逐步替换** — 每个 Phase 独立交付，不影响现有功能
2. **网关路由切换** — APISIX 优先级路由，新 Go endpoint 就绪后改路由指向，Django 端点保留为回退
3. **数据所有权迁移** — 每迁出一个功能域，将其表的所有权从 `saas` SQLite 迁至对应 Go 服务的数据库
4. **事件驱动解耦** — domain events 的 intent (Kafka→HTTP) 改为原生 Go Kafka consumer
5. **先 internal，后公网** — 先消除服务间 Django HTTP 调用，再动公网 API

### 不可迁移项

| 项目 | 理由 |
|------|------|
| `/api/health/` + `/api/live/` | 极简探活，保留 Django 轻量无意义 |
| 一次性管理脚本/命令 | Django management commands — 重写 ROI 为零 |
| skillList 静态文件服务 | 非 API，可随前端迁移自然消亡 |

---

## 📋 分阶段迁移计划

### Phase 1: 死代码清理（1-2 天）⭐ 即刻可执行

```
目标：删除已迁出但代码仍留在磁盘上的 Django 代码
零风险 — 不改变任何运行中行为
```

| 任务 | 内容 | 影响 |
|------|------|------|
| 1.1 | 删除 `task2app/Saas_Ai_Provider/`（97 文件） | 清理已由 Go taskAiProvider 替代的死代码 |
| 1.2 | 删除 `task2app/gitOauth/` 残留（如存在） | 已由 Go taskGitOauth 替代 |
| 1.3 | 移除 saas-backend 中空 urlpatterns 的 include | `urls_taskauth_internal` 等已清空的模块 |
| 1.4 | 清理 410 Gone stub（确认无调用方后删除） | 减少 dead path 噪音 |
| 1.5 | 更新 `db/api_route_ownership.yaml` baseline | 重新 dump，反映当前真实路由 |

### Phase 2: 事件意图 Kafka 化（1-2 周）⭐⭐ 消除 Kafka→HTTP 回环

```
目标：16 个 intent endpoints → Go Kafka consumers
消除 taskEvents → Django HTTP 的架构反模式
```

| 任务 | Intent | 目标 Go 服务 |
|------|--------|-------------|
| 2.1 | `company-by-creator` | taskTenantService |
| 2.2 | `create-company` | taskTenantService |
| 2.3 | `upsert-user-profile` | taskAuth |
| 2.4 | `set-default-deliverable` | taskProjectService |
| 2.5 | `set-default-progress` | taskProjectService |
| 2.6 | `create-default-workspace` | taskProjectService |
| 2.7 | `handle-workspace-created` | taskProjectService |
| 2.8 | `create-access-key-iam` | taskCloudService |
| 2.9 | `latest-pending-start-event` | taskCloudService |
| 2.10 | `update-cloud-server-event-status` | taskCloudService |
| 2.11 | `update-cloud-server-event-data` | taskCloudService |
| 2.12 | `init-tenant-feature-params` | taskCloudService |
| 2.13 | `grant-initial-resources` | taskCloudService |
| 2.14 | `mark-user-tenant` | taskTenantService |
| 2.15 | `realtime/dispatch/` (SSE) | taskEvents (Go) |

**验收标准：** `POST /api/internal/task-events/intents/*` 零调用；Django `urls_internal.py` 可清空

### Phase 3: Internal API 迁出（2-4 周）⭐⭐ 消除服务间 Django HTTP 调用

```
目标：所有 /api/internal/* Django 端点迁至对应 Go 服务
消除 Django 作为"领域真源 HTTP 服务"的架构角色
```

| 任务 | Internal API | 方案 |
|------|-------------|------|
| 3.1 | `/api/internal/session/resolve/` | taskAuth 直接读写 Django session store（或改用 JWT 自包含） |
| 3.2 | `/api/internal/git-identities/lookup/` | 数据迁入 taskGitOauth DB；taskCredentialService 经 taskGitOauth internal API 查询 |
| 3.3 | `/api/internal/taskproject/*`（10 端点） | 逐个评估：数据 owner 在 Django 还是 Go？迁表或迁逻辑 |
| 3.4 | `/api/internal/task-ai-comment/import/` | taskAIComment 直写自己的 DB，不再回调 Django |
| 3.5 | `/api/internal/feature-params/*` | 剩余 3 端点迁 taskCloudService |
| 3.6 | `/api/internal/task-agent-support/*` | 薄壳转发 → 在 taskAgentSupport Go 中直调目标服务 |

**验收标准：** Django `urls.py` 中所有 `/api/internal/` include 可移除

### Phase 4: 云平台配置迁移（2-4 周）⭐⭐

```
目标：cloud/ app（425 文件）→ taskCloudService
这是 saas-backend 最大的 App
```

| 任务 | 内容 | 目标 |
|------|------|------|
| 4.1 | 云平台授权 CRUD | taskCloudService 已有部分实现，补齐 |
| 4.2 | 网络资源查询（VPC/VSwitch/SecurityGroup） | taskCloudService |
| 4.3 | 镜像管理（images/server-images） | taskCloudService |
| 4.4 | regions/instance-types 查询 | taskCloudService |
| 4.5 | server-startup-status（降级路径） | 删除 Django 实现，taskSSE 为唯一路径 |
| 4.6 | container-inbound 最后 2 个 action | repo-reclone → taskCloudService；server-userdata-verify → taskCloudService |
| 4.7 | 阿里云 OAuth 完整闭环 | 回调已在 Go；login 发起也迁 Go |

**验收标准：** `cloud/urls.py` 可清空或删除

### Phase 5: 计费桥接清理（1 周）⭐

```
目标：billing_bridge → APISIX 直连 taskBill
```

| 任务 | 内容 |
|------|------|
| 5.1 | BillingProxyView → APISIX route 直连 taskBill |
| 5.2 | `/api/internal/taskbill/*` → taskBill 内部消化 |
| 5.3 | 订阅/支付回调 → taskBill |

**验收标准：** `billing/` + `billing_bridge/` urlpatterns 可清空

### Phase 6: 系统管理面迁移（1 周）

```
目标：cloudSystemAdmin + frontend_app 管理 API
```

| 任务 | 内容 | 目标 |
|------|------|------|
| 6.1 | `/api/system-admin/*` dashboard | taskAuth（超管能力）或独立 Go |
| 6.2 | `/api/system_admin/*/cloud/` | taskCloudService |
| 6.3 | 交付体系 manage-deliverable-* | taskProjectService |
| 6.4 | feature-params, task-panels | taskCloudService |
| 6.5 | product-pricing, refund | taskBill |

### Phase 7: 核心域迁移 — accounts + projects（4-8 周）⭐⭐⭐ 最复杂

```
目标：用户、组织、项目 CRUD 从 Django ORM → Go 服务
这是迁移的核心难点 — 业务规则最密集、前端耦合最深
```

| 子阶段 | 内容 | 方案 |
|--------|------|------|
| **7a. 读路径先行** | 为高频读路径（用户列表、项目列表）创建 Go read API | Go → 原 Django DB 直读（临时）→ 逐步迁表 |
| **7b. 写路径逐个** | 按功能域拆：用户注册 → taskAuth；公司创建 → taskTenantService；项目创建 → taskProjectService | 每个迁移包含：表迁移 + handler + APISIX 路由切换 |
| **7c. 用户档案** | profile, avatar, git-identities | taskAuth + taskGitOauth |
| **7d. 公司/成员/组** | Company CRUD, workspace access | taskTenantService |
| **7e. 项目 CRUD** | Project + GitLab 集成 | taskProjectService |
| **7f. 任务/评论** | Todo + Comment CRUD | taskTaskService + taskAIComment |
| **7g. SSO 桥接** | OIDC SSO 端点 | taskAuth |

### Phase 8: 最终清理（1 周）

```
目标：Django 进程下线
```

| 任务 | 内容 |
|------|------|
| 8.1 | 确认所有路由已迁出（APISIX 零 Django upstream 引用） |
| 8.2 | 下线 `saas-backend` 进程 |
| 8.3 | 删除 `task2app/` 目录 |
| 8.4 | 移除 runAll.yaml 中 saas-backend 定义 |
| 8.5 | 更新 `db/table_ownership.yaml`：移除 saas 库 |
| 8.6 | 存档 `docs/architecture/`：Django 最终版 → archived |

---

## ⏱️ 时间线总览

```
Phase 1 ████ (1-2d)  死代码清理
Phase 2 ████████ (1-2w)  事件意图 Kafka 化
Phase 3 ████████████████ (2-4w)  Internal API 迁出
Phase 4 ████████████████ (2-4w)  云平台配置迁移
Phase 5 ████ (1w)  计费桥接清理
Phase 6 ████ (1w)  系统管理面
Phase 7 ████████████████████████████████ (4-8w)  核心域迁移
Phase 8 ████ (1w)  最终清理

总计：12-22 周（约 3-5.5 个月）
```

---

## 🏛️ 架构变更影响

- **迭代版本**: v56 🎯 target
- **迭代名称**: django-final-go-migration
- **作者**: claude
- **设计日期**: 2026-07-29
- **变更明细**:
  - 🔴 [DEPRECATED] `saas-backend` Django — 分阶段退役
  - 🔴 [DEPRECATED] `task2app/Saas_Ai_Provider/` — 死代码清理
  - 🟢 [NEW] taskEvents Go Kafka consumers — 替代 16 intent HTTP endpoints
  - 🟡 [MODIFIED] taskCloudService — cloud platform 配置全量接管
  - 🟡 [MODIFIED] taskTenantService — companies/members 接管
  - 🟡 [MODIFIED] taskProjectService — projects/workspace ACL 接管
  - 🟡 [MODIFIED] taskAuth — 用户档案/internal session 接管
  - 🟡 [MODIFIED] taskBill — 计费桥接/订阅/pricing 接管

---

## 领域概念清单（供 /6-ddd 使用）

| 有界上下文 | 关键实体 | 当前 Owner | 目标 Owner |
|------------|----------|-----------|------------|
| **User & Auth** | User, UserProfile, Session | saas-backend | taskAuth |
| **Company & Tenant** | Company, TenantOption, Member, Group | saas-backend | taskTenantService |
| **Project & Workspace** | Project, Workspace, WorkspaceAccess | saas-backend | taskProjectService |
| **Task & Comment** | Todo, Comment | saas-backend + taskTaskService | taskTaskService + taskAIComment |
| **Cloud Platform** | CloudPlatformAuth, VPC, Image | saas-backend + taskCloudService | taskCloudService |
| **Billing** | Subscription, Payment, Pricing | saas-backend + taskBill | taskBill |
| **Git Identity** | GitIdentity, OAuthToken | saas-backend + taskGitOauth | taskGitOauth |
| **Domain Events** | CompanyCreated, WorkspaceCreated... | Kafka→Django HTTP | Kafka→Go consumers |
| **SSO Bridge** | OIDC State, SSO Session | saas-backend + taskAuth | taskAuth |

---

## 业务意图 → 事件对照（Phase 2 相关）

| 业务意图 | 事件名（过去式） | 当前发布点 | 目标消费者 | 备注 |
|---------|----------------|-----------|-----------|------|
| 用户注册成功 | UserRegistered | taskAuth → Kafka | 待迁：Go consumer (taskTenantService) | 替代 upsert-user-profile intent |
| 公司创建成功 | CompanyCreated | taskTenantService → Kafka | 待迁：Go consumer (taskEvents) | 替代 create-company intent |
| 工作空间创建 | WorkspaceCreated | taskProjectService → Kafka | Go consumer | 替代 create-default-workspace intent |
| Feature params 初始化 | TenantFeatureParamsInited | taskCloudService | Go consumer | 替代 init-tenant-feature-params intent |

---

## 开放问题

1. **Django Session 共享** — taskAuth 当前通过 `/api/internal/session/resolve/` 读 Django session。方案 A：taskAuth 直接读写同一 Redis session store；方案 B：taskAuth 发 JWT 完全自包含。待评估。

2. **SQLite → MySQL 迁移** — saas 库当前是 SQLite。Phase 7 迁表时是否同时切 MySQL？建议与各 Go 服务保持一致（目前已用 MySQL 的：taskGitOauth, taskAiProvider, taskTenantService 等）。

3. **Phase 7 的"大爆炸"风险** — accounts + projects 是核心域，迁移期间需要同时维护 Django 和 Go 两套实现。建议按子阶段 7a-7g 严格串行，每个子阶段完成 + 稳定后再进入下一个。

4. **ai-provider Django 代码清理** — Phase 1 可立即删除。但需确认 `task2app/Saas_Ai_Provider/apps/marketplace/health.py` 不再被任何健康检查引用。

---

## 建议下一步

批准本设计后：

1. **Phase 1 即可开干** — 死代码清理，零风险
2. **Phase 2-3** — 走 `/4-value-stream` 映射 Kafka consumers + internal API 迁移增量
3. **架构文件** — 创建 v56 target 架构（含 Plateau/Gap/WP 架构变迁视图）
