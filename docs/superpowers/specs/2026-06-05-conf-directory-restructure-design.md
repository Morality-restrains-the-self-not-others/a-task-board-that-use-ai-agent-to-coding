# conf/ 目录结构调整为 `conf/<职能>/<服务>/` 设计

> 日期：2026-06-05  
> 状态：**已批准** — 9 职能方案，所有命名已确认  
> 触发：用户要求将 conf 从扁平结构 `conf/<服务>/` 调整为二级结构 `conf/<职能>/<服务>/`  
> 前身：`2026-06-02-port-config-split-monorepo-conf-design.md`（Phase 3 已完成）  
> 用户选择：**9 职能分组** / django 组命名 **core** / mock 服务**独立成组**

---

## 1. 当前状态分析

### 1.1 现状：扁平结构

```
conf/                              # 独立 Git 仓库
├── README.md
├── runAll.yaml                    # runAll 编排入口
├── .gitignore
├── .git/                          # conf 是独立 git repo
├── .runall/                       # runAll 内部数据
├── logs/                          # 运行日志（非配置）
├── ai-provider/                   # 有 sync 四件套
├── django/                        # 有 sync 四件套 + 多碎片
├── docker-infra/                  # 仅 config.yaml
├── domain-events/                 # 有 sync + per-event 子目录
├── git-oauth/                     # 有 sync + providers/
├── git-service/                   # 仅 config.yaml
├── mock-run-container/            # 仅 config.yaml
├── mock-trae-worker/              # 仅 config.yaml
├── relay-to-trae/                 # 仅 config.yaml
├── stripe/                        # 仅 config.yaml
├── task-agent-support/            # 仅 config.yaml
├── task-ai-endpoint/              # 仅 config.yaml
├── task-auth/                     # 有 sync
├── task-bill/                     # 仅 config.yaml
├── task-container-gateway/        # 仅 config.yaml
├── task-gateway/                  # 仅 config.yaml
├── task-sse/                      # 有 sync
└── vue/                           # 有 sync
```

### 1.2 现有服务分类（按 runAll.yaml 编排组）

| runAll Group | Services |
|---|---|
| **infrastructure** | docker-redis, docker-kafka, docker-portainer, ai-monitor, git-service |
| **platform** | task-auth, task-bill, git-oauth, ai-provider, saas-backend(django), task-gateway, task-agent-support, task-ai-endpoint, task-container-gateway, task-sse, taskFE(vue) |
| **domain-events-intents** | task-events-* (18 processors) |
| **container-stack** | go-run-container(mock-run-container), go-relay(relay-to-trae) |
| **value-stream** | valueStream (独立，非 conf 管理) |

### 1.3 Sync 四件套分布

| Service | config.yaml | sync.manifest.yaml | sync.sh | 依赖碎片 |
|---------|-------------|--------------------|---------|----------|
| ai-provider | ✓ | ✓ | ✓ | django.yaml |
| django | ✓ | ✓ | ✓ | task-auth.yaml, domain-events.yaml, docker-infra.yaml |
| domain-events | ✓ | ✓ | ✓ | django.yaml, docker-infra.yaml |
| git-oauth | ✓ | ✓ | ✓ | django.yaml |
| task-auth | ✓ | ✓ | ✓ | django.yaml |
| task-sse | ✓ | ✓ | ✓ | docker-infra.yaml |
| vue | ✓ | ✓ | ✓ | django.yaml |
| docker-infra | ✓ | — | — | — |
| git-service | ✓ | — | — | — |
| 其余 8 个 | ✓ | — | — | — |

## 2. 分组方案（已确认：9 职能）

### 2.1 最终方案

| # | 职能 | 服务 | 中文含义 | 分组依据 |
|---|------|------|----------|----------|
| 1 | **auth** | task-auth, git-oauth | 认证授权 | 身份服务，所有其他服务依赖 |
| 2 | **core** | django | 核心后端 | 主 SaaS 平台（saas-backend），依赖 auth + events + infra |
| 3 | **gateway** | task-gateway, task-container-gateway, task-sse | 网关与实时通信 | API 网关 + 容器网关 + SSE 实时推送 |
| 4 | **ai** | ai-provider, task-ai-endpoint, task-agent-support | AI 服务 | AI 服务集群，均依赖 django |
| 5 | **events** | domain-events | 领域事件 | 事件处理，有独立 per-event 子结构 |
| 6 | **infra** | docker-infra, git-service, relay-to-trae | 基础设施 | Docker/Git/中继 |
| 7 | **billing** | task-bill, stripe | 计费 | 计费与支付 |
| 8 | **frontend** | vue | 前端 | Vite SPA |
| 9 | **mock** | mock-run-container, mock-trae-worker | Mock 服务 | 本地开发 mock，非生产服务 |

**统计**：9 个职能，21 个服务。单服务组 3 个（core / events / frontend），2-3 服务组 5 个，无超大组。

### 2.2 关键决策记录

| 决策 | 选项 | 选择 | 理由 |
|------|------|------|------|
| 粒度 | 4 / 7 / 9 组 | **9 组** | 用户偏好；django 独立为核心、mock 独立、SSE 归入网关 |
| django 组命名 | backend / core / platform | **core** | 用户选择 |
| task-sse 归属 | platform / gateway / realtime | **gateway** | SSE 本质是推送网关，与 task-gateway 功能内聚 |
| mock 归属 | infra / 独立 | **mock 独立** | 用户选择；mock 是非生产服务，独立分组更清晰 |

### 2.3 备选方案（已淘汰）

<details>
<summary>7 职能方案（推荐但用户未选）</summary>

| 职能 | 服务 |
|------|------|
| auth | task-auth, git-oauth |
| platform | django, task-gateway, task-container-gateway, task-sse |
| ai | ai-provider, task-ai-endpoint, task-agent-support |
| events | domain-events |
| infra | docker-infra, git-service, relay-to-trae, mock-run-container, mock-trae-worker |
| billing | task-bill, stripe |
| frontend | vue |
</details>

## 3. 目标目录结构

```
conf/
├── README.md                         # 更新路径引用
├── runAll.yaml                       # 更新 conf_app 路径
├── .gitignore                        # 不变
├── .git/                             # 不变（Git 跟踪重命名）
├── .runall/                          # 不变
├── logs/                             # 不变（运行日志）
│
├── auth/                             # 认证授权
│   ├── task-auth/
│   │   ├── config.yaml
│   │   ├── sync.manifest.yaml        # 更新碎片 from 路径
│   │   ├── sync.sh
│   │   └── django.yaml               # GENERATED
│   └── git-oauth/
│       ├── config.yaml
│       ├── sync.manifest.yaml
│       ├── sync.sh
│       ├── django.yaml               # GENERATED
│       └── providers/                # OAuth 供应商配置（子目录保留）
│           ├── github.yaml
│           ├── gitlab-daydaymoney.yaml
│           └── ...
│
├── core/                             # 核心后端
│   └── django/                       # saas-backend
│       ├── config.yaml
│       ├── config.example.yaml
│       ├── config.test.yaml
│       ├── config.local.yaml         # gitignored
│       ├── sync.manifest.yaml        # 更新碎片 from 路径
│       ├── sync.sh
│       ├── task-auth.yaml            # GENERATED
│       ├── domain-events.yaml        # GENERATED
│       ├── docker-infra.yaml         # GENERATED
│       └── vue.yaml                  # GENERATED
│
├── gateway/                          # 网关与实时通信
│   ├── task-gateway/
│   │   └── config.yaml
│   ├── task-container-gateway/
│   │   └── config.yaml
│   └── task-sse/
│       ├── config.yaml
│       ├── sync.manifest.yaml
│       ├── sync.sh
│       └── docker-infra.yaml         # GENERATED
│
├── ai/                               # AI 服务
│   ├── ai-provider/
│   │   ├── config.yaml
│   │   ├── sync.manifest.yaml
│   │   ├── sync.sh
│   │   └── django.yaml               # GENERATED
│   ├── task-ai-endpoint/
│   │   └── config.yaml
│   └── task-agent-support/
│       └── config.yaml
│
├── events/                           # 领域事件
│   └── domain-events/
│       ├── config.yaml               # 全局 transport/redis/kafka
│       ├── sync.manifest.yaml
│       ├── sync.sh
│       ├── django.yaml               # GENERATED
│       ├── docker-infra.yaml         # GENERATED
│       ├── billing_transaction_created/   # per-event 子目录
│       ├── user_created/
│       ├── company_created/
│       ├── ...                        # 其余 event slug
│       ├── sse_message/
│       ├── task_completed/
│       └── workspace_created/
│
├── infra/                            # 基础设施
│   ├── docker-infra/
│   │   └── config.yaml
│   ├── git-service/
│   │   └── config.yaml
│   └── relay-to-trae/
│       └── config.yaml
│
├── billing/                          # 计费
│   ├── task-bill/
│   │   └── config.yaml
│   └── stripe/
│       └── config.yaml
│
├── frontend/                         # 前端
│   └── vue/
│       ├── config.yaml
│       ├── sync.manifest.yaml
│       ├── sync.sh
│       └── django.yaml               # GENERATED
│
└── mock/                             # Mock 服务（非生产）
    ├── mock-run-container/
    │   └── config.yaml
    └── mock-trae-worker/
        └── config.yaml
```

## 4. 影响范围分析

### 4.1 需要更新的路径引用

| 影响项 | 文件 | 变更说明 |
|--------|------|----------|
| sync.manifest.yaml `from` | 7 个 manifest | `from: ../task-auth/config.yaml` → `from: ../../auth/task-auth/config.yaml`（跨职能引用变深一层）|
| sync.sh 工作目录 | 7 个 sync.sh | 相对路径以 sync.sh 所在目录为基准，函数内 `cd` 需确认不受影响 |
| runAll.yaml `conf_app` | `conf/runAll.yaml` | `conf_app: task-auth` → `conf_app: auth/task-auth`，共约 10 处 `conf_app` |
| scripts/conf-read.py | `scripts/conf-read.py` | 路径解析 `<app>` → `<职能>/<服务>` |
| scripts/conf-sync-all.sh | `scripts/conf-sync-all.sh` | glob `conf/*/sync.sh` → `conf/*/*/sync.sh` |
| scripts/ci/check_conf_sync.sh | `scripts/ci/check_conf_sync.sh` | 同上 |
| FindMonorepoRoot 标记 | `task2app/paths.conf` 等 | `conf/core/django/config.yaml` → `conf/core/django/config.yaml` |

### 4.2 不影响的部分

| 项目 | 说明 |
|------|------|
| config.yaml 内容 | 服务端口、密钥等业务值**完全不变** |
| sync 碎片内容 | GENERATED 文件内容不变（仅源路径变更） |
| sync 协议 | `sync.manifest.yaml` 格式、`pick` 语义、`conf-sync.py` 逻辑**不变** |
| .gitignore 规则 | `config.local.yaml` 忽略规则不变 |
| config.test.yaml 机制 | 测试配置合并逻辑不变 |
| 端口号 | 所有服务端口号不变 |
| domain-events 子结构 | `domain-events/<event>/config.yaml` 每事件一文件不变 |

### 4.3 特殊处理

#### 4.3.1 conf/ 是独立 Git 仓库

`conf/` 内含 `.git/`，是独立仓库。使用 `git mv` 重命名目录可保留 commit 历史。

#### 4.3.2 domain-events 的 per-event 子目录

`domain-events/` 包含 `sync.sh`、`config.yaml` **和** 18 个 per-event 子目录。迁移后：`events/domain-events/` 依然是同一个服务目录，per-event 子目录是内部实现细节。

#### 4.3.3 logs/ 目录

`conf/logs/` 是运行日志（`conf-sync.log` 等），非服务配置。保持 `conf/logs/` 不变。

#### 4.3.4 task-sse 归属 gateway

task-sse 是 SSE 实时推送服务。虽然与 API 网关不同，但都属于"通信/分发"层。与 task-gateway（HTTP API 路由）和 task-container-gateway（容器 WebSocket 路由）同组，构成完整的"平台入口层"。

## 5. 价值流影响

### 5.1 受影响的价值流和步骤

| Value Stream | 步骤 | 影响 |
|---|---|---|
| user-auth | runall-task-auth-orchestration | `conf_app: task-auth` → `conf_app: auth/task-auth` |
| (所有 stream) | runAll 编排步骤 | 所有带 `conf_app` 的服务路径更新 |
| (N/A) | CI conf-sync 检查 | `check_conf_sync.sh` glob 路径 |

### 5.2 新 Stream？

**不需要新 value stream** — 这是配置目录结构的内部重组。如需跟踪进度，可在"系统管理与策略" domain 下加入一个简短的迁移步骤。

### 5.3 字段影响

- `fields[].name` 无需变更 — 数据库字段、业务字段均不受影响
- `value-stream.yaml` 中涉及的 `conf/` 路径引用需更新 description 文本

## 6. 领域概念清单（供 `/5-ddd`）

| 概念 | 说明 |
|------|------|
| **Bounded Context** | 平台运行时配置（Platform Runtime Config）— 新增 `ConfigFunction` 分层 |
| **Entity** | `ServiceConfig`（每服务的 `config.yaml`） |
| **Value Object** | `ConfigFunction`（职能标签：auth/core/gateway/ai/events/infra/billing/frontend/mock） |
| **Aggregate** | `FunctionConfigBundle`（一个职能下所有服务的配置集合） |

## 7. 迁移策略

### 7.1 Git 原生重命名（保留历史）

```bash
# 在 conf/ 仓库内执行
cd conf

# 创建职能目录
mkdir -p auth core gateway ai events infra billing frontend mock

# 批量移动
git mv task-auth    auth/task-auth
git mv git-oauth    auth/git-oauth

git mv django       core/django

git mv task-gateway           gateway/task-gateway
git mv task-container-gateway gateway/task-container-gateway
git mv task-sse               gateway/task-sse

git mv ai-provider         ai/ai-provider
git mv task-ai-endpoint     ai/task-ai-endpoint
git mv task-agent-support   ai/task-agent-support

git mv domain-events    events/domain-events

git mv docker-infra    infra/docker-infra
git mv git-service     infra/git-service
git mv relay-to-trae   infra/relay-to-trae

git mv task-bill   billing/task-bill
git mv stripe      billing/stripe

git mv vue   frontend/vue

git mv mock-run-container   mock/mock-run-container
git mv mock-trae-worker     mock/mock-trae-worker
```

### 7.2 实施分期

| Phase | 内容 | 风险 | 变更文件数 |
|-------|------|------|-----------|
| **1. Git 迁移** | `git mv` 所有 21 个服务到新路径 | 低：纯文件移动，Git 保留历史 | 21 moves |
| **2. Sync 路径更新** | 更新 7 个 `sync.manifest.yaml` 的 `from`；执行 `conf-sync-all.sh` 重新生成碎片 | 中：碎片生成需验证一致性 | ~14 files |
| **3. 外部引用更新** | `runAll.yaml`、`scripts/conf-read.py`、`scripts/conf-sync-all.sh`、CI 脚本 | 中：需 CI 验证 | ~10 files |
| **4. 运行时验证** | runAll 全栈启动；conf-sync CI 检查 | 低：Phase 2-3 正确则自动通过 | — |
| **5. 文档更新** | `conf/README.md`、`value-stream.yaml` description、相关 `.ai.md` | 低 | ~5 files |

### 7.3 回滚方案

1. `git revert` 迁移 commit（`git mv` 支持 Git 原生 revert）
2. 恢复 `conf/` 到迁移前状态
3. 外部引用如已提交，需单独 revert（建议分 Phase 提交）

## 8. 自检清单

- [x] 分组方案已确认：9 职能（用户选择）
- [x] 所有 21 个服务目录均已分配职能
- [x] sync.manifest.yaml 的 from/to 路径已规划
- [x] runAll.yaml conf_app 路径已规划
- [x] CI 检查路径已规划
- [x] conf/ 独立 Git 仓库的迁移方式已明确（git mv）
- [x] 回滚方案已定义
- [x] 无业务配置值（端口/密钥等）变更
- [x] domain-events per-event 子结构保持

## 9. 最终确认（全部通过 ✅）

1. ✅ **task-sse 放在 gateway** — SSE 推送与 API 网关、容器网关同组
2. ✅ **命名确认** — `auth` / `core` / `gateway` / `ai` / `events` / `infra` / `billing` / `frontend` / `mock` 全部确认

---

## 10. 下一步（流程）

设计批准后 → 可选 **Git Worktree** / **价值流映射** / **直接实施**。
