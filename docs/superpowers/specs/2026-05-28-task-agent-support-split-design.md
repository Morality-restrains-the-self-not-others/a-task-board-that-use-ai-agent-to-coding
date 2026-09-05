# 设计文档：taskAgentSupport — 容器 inbound 接口独立服务

**日期：** 2026-05-28  
**状态：** Phase 1 已实现（:8011，非设计稿原 :8006——该端口已被 task-events-projects 占用）  
**动机：** 将 `onlineServiceJS` / `go_relayToTrae` 回调 SaaS 的 inbound 流量从 Django `runserver` 剥离，避免与浏览器侧 `cloud/compute/*` 长请求争用单线程，提升容器心跳/换票可靠性。

---

## 背景与问题

### 现状

`trae-agent/onlineServiceJS` 通过环境变量 `TaskApiEndPoint` / `TASK_API_ENDPOINT` 指向：

```
/api/tenant/{tenant}/workspace/{workspace}/task/{task}/cloud/
```

并周期性/事件驱动调用 Django 上 **`server-container-token/*`** 与 **`relay-to-trae/status-push/`** 路由（定义于 `cloud/instance_callback_urls.py`）。

| 路由 | 调用方 | 职责 |
|------|--------|------|
| `exchange-refresh/` | onlineServiceJS 启动 | refresh → access 换票 |
| `refresh-access/` | onlineServiceJS | access 续期 |
| `register-reachability/` | onlineServiceJS | 注册业务 API 地址 |
| `heartbeat/` | onlineServiceJS 每 ~20s | 容器在线 + SSE 通知 |
| `feature-params-yaml/` | bootstrap | 租户 feature 参数 |
| `task-detail/` | bootstrap / OAuth 刷新 | 任务上下文 |
| `repo-clone-credentials/` | bootstrap | 克隆凭据 |
| `git-clone-progress/` | clone 过程 | 进度上报 |
| `layer-changes-push/` | 层 diff | 变更快照 |
| `layer-graph-push/` | zTree 同步 | 层级图快照 |
| `layer-github-oauth-access-tokens/` | 层内 git OAuth | 按仓拉票 |
| `relay-to-trae/status-push/` | go_relayToTrae | relay 状态/审计 |

实现集中在 `task2app/Saas_project/cloud/views/*`，依赖：

- `CloudServerConfig` / `ContainerTokenSession`（ORM + 领域服务）
- gitOauth 内部换票（`layer-github-oauth-access-tokens`）
- taskSSE 发布（heartbeat → 任务详情 UI）
- 容器 token 审计（`ContainerTokenAuditEvent`）

### 痛点（已观测）

1. **Django runserver 单线程拥塞**：浏览器 `container-layer-git-push` 等等待容器最多 120s 时，同进程内 heartbeat / 其它 `/api/*` 排队 pending（见 2026-05-28 push pending 排查）。
2. **职责混杂**：容器 machine-to-machine 回调与用户 session 鉴权的 compute 转发共用一个 WSGI 进程。
3. **扩缩容粒度粗**：无法单独为容器回调配置超时、并发、限流。

### 非目标（用户已确认 scope）

- **不迁移** 浏览器 → Django → onlineServiceJS 的 **`cloud/compute/*`** 出站转发（layer-graph 读取、git-push、jobs 等仍留 `saas-backend`）。
- **不迁移** `repo-reclone/`（用户浏览器触发）、`server-userdata-verify/`（云厂商 userdata 探针）。
- **不在本阶段** 重写全部容器 token 领域逻辑为 Go；优先「流量隔离 + 契约稳定」。

---

## 目标

1. 新建独立服务 **`taskAgentSupport`**，专门接收容器/relay inbound HTTP。
2. **对外 URL 路径保持不变**（或仅改 `TaskApiEndPoint` 端口，路径不变），onlineServiceJS 无需大改。
3. 容器 heartbeat / 换票 **不再经过** Django runserver 主线程队列。
4. 与 taskSSE 侧车、`taskAuth` 拆分模式一致，纳入 `runAll.yaml` + `port_config.json`。

---

## 方案对比

### 方案 A（推荐）：Go 网关 + Django Internal API

```
onlineServiceJS / relayToTrae
        │  POST .../server-container-token/*
        ▼
 taskAgentSupport (Go, :8006)
        │  X-TaskAgentSupport-Internal-Secret
        │  POST /api/internal/task-agent-support/{action}/
        ▼
 saas-backend Django
        │  现有 ContainerTokenLifecycleService / views 逻辑
        ├── SQLite (CloudServerConfig, audit)
        ├── gitOauth :8010
        └── taskSSE publish
```

| 优点 | 缺点 |
|------|------|
| 与 taskAuth/taskBill 编排一致 | 首期需定义 internal API 契约 |
| Go 进程轻量、可多 goroutine 并发 | 多一跳 latency（内网可忽略） |
| Django 领域层复用，迁移风险低 | Go 层需维护路由/鉴权薄壳 |

### 方案 B：Python 侧车（第二 Django/FastAPI 进程）

从 `cloud/views/container_*` 抽包，独立 `manage.py runserver :8006`，共享 ORM。

| 优点 | 缺点 |
|------|------|
| 几乎零业务逻辑搬迁 | 仍是 Python GIL + 若用 runserver 仍有单线程风险 |
| 测试可复用现有 pytest | 两实例共享 DB 迁移/配置复杂 |

### 方案 C：Go 全量重写 token 域

token session、audit、gitOauth 代理全部迁入 Go + 独立 SQLite。

| 优点 | 缺点 |
|------|------|
| 长期最干净 | 工作量巨大，与现有 DDD/审计测试重复 |

**推荐：方案 A**，分阶段交付；方案 B 仅作「若 Go 内部 API 阻力过大」的备选。

---

## 架构（方案 A 详设）

### 组件

| 组件 | 职责 |
|------|------|
| **taskAgentSupport** | 公网/内网 HTTP 入口；校验 `access_token` 格式；限流；透传 `X-Trace-Id`；写 access 日志 |
| **saas-backend internal** | 业务真源：token 生命周期、凭据组装、gitOauth 换票、SSE 通知、审计落库 |
| **onlineServiceJS** | 仅改 `TaskApiEndPoint` host:port（路径不变） |
| **runAll / 反向代理** | 可选：仍走 :8001 由网关按 path 分流到 :8006 |

### 路由与兼容性

**对外路径（保持不变）：**

```
POST /api/tenant/{tid}/workspace/{wid}/task/{tk}/cloud/server-container-token/{action}/
POST /api/tenant/{tid}/workspace/{wid}/task/{tk}/cloud/relay-to-trae/status-push/
```

**对内路径（新增，仅 cluster 内）：**

```
POST /api/internal/task-agent-support/exchange-refresh/
POST /api/internal/task-agent-support/refresh-access/
POST /api/internal/task-agent-support/register-reachability/
POST /api/internal/task-agent-support/heartbeat/
POST /api/internal/task-agent-support/feature-params-yaml/
POST /api/internal/task-agent-support/task-detail/
POST /api/internal/task-agent-support/repo-clone-credentials/
POST /api/internal/task-agent-support/git-clone-progress/
POST /api/internal/task-agent-support/layer-changes-push/
POST /api/internal/task-agent-support/layer-graph-push/
POST /api/internal/task-agent-support/layer-github-oauth-access-tokens/
POST /api/internal/task-agent-support/relay-status-push/
```

Internal 请求体携带：`tenant_id`, `workspace_id`, `task_id`, `body`（原始 JSON）、`trace_id`、客户端 IP（可选）。

响应：**与现有容器 API 完全一致**（含 `error_code`, `trace_id` 字段），保证 onlineServiceJS 无感。

### 鉴权分层

| 层 | 规则 |
|----|------|
| 公网入口 | 现有 `access_token` / `refresh_token` 校验逻辑 **仍在 Django**（internal 调用前不在 Go 重复实现 domain 规则，避免双写） |
| Internal | `X-TaskAgentSupport-Internal-Secret`（`port_config.json` 配置，与 taskAuth bridge 类似） |
| 容器 token | 不引入用户 session cookie；`AllowAny` 行为与现 `container_runtime_token_views` 一致 |

Go 层 **不做** token 业务判定，只做 HTTP 服务与 optional rate limit；token 校验留在 Django internal handler（复用 `ContainerTokenLifecycleService`）。

### 配置

`port_config.json` 新增：

```json
"taskAgentSupport": {
  "host": "127.0.0.1",
  "port": 8006,
  "djangoInternalApiBase": "http://127.0.0.1:8001",
  "internalSecret": "<shared-secret>"
}
```

`runAll.yaml` platform 组新增 `task-agent-support`，`depends_on: [saas-backend]`。

容器环境：

- **推荐**：`TaskApiEndPoint=http://127.0.0.1:8006/api/tenant/.../cloud`（relay 启动脚本注入）
- **兼容期**：:8001 保留路由 alias（Django 307/反向代理到 :8006 或 deprecated 警告）

### 数据与依赖

| 数据/服务 | 归属 | taskAgentSupport 是否直连 |
|-----------|------|---------------------------|
| CloudServerConfig | Django ORM | 否（经 internal API） |
| ContainerTokenAuditEvent | Django ORM | 否 |
| gitOauth | :8010 | 否（Django internal 调用） |
| taskSSE publish | :8798 | 否（Django internal 调用） |
| 容器 run 文件日志 | `logs/container_run/` | 可选：Go 写「收到请求」摘要，Django 仍写业务 outbound 日志 |

taskAgentSupport **无独立业务数据库**（首期）；避免 token 双源。

### 错误处理与超时

| 调用 | Go → Django timeout | 说明 |
|------|---------------------|------|
| heartbeat | 5s | 高频；失败 onlineServiceJS 重试 |
| exchange-refresh / refresh-access | 15s | 启动关键路径 |
| layer-graph-push | 30s | payload 较大 |
| layer-github-oauth-access-tokens | 15s | 含 gitOauth 换票 |

Go 返回 502 时 body 含 `detail` + `trace_id`，与现 Django upstream 错误风格对齐。

### 可观测性

- 继承 `X-Trace-Id`（与 centralized-logs 设计一致）
- Go access log：method, path, status, duration_ms, trace_id, task_id
- Django internal handler 继续写 `ContainerTokenAuditEvent` + `container_machine_api_access_middleware` 等价日志（可迁至 internal 专用 middleware）

---

## 分阶段交付

### Phase 1 — 薄壳 + heartbeat（最小可用）

- 新建 `taskAgentSupport/`（Go，`run.sh`，`/api/health/`）
- 实现 `heartbeat` + `exchange-refresh` + `refresh-access` 三条 internal 转发
- relay 直启改 `TaskApiEndPoint` 指向 :8006
- Playwright：`TaskDetail.relay-to-trae-direct-start` 绿

### Phase 2 — bootstrap 全量

- 迁移 `task-detail`, `repo-clone-credentials`, `feature-params-yaml`, `git-clone-progress`, `register-reachability`
- onlineServiceJS e2e bootstrap 用例绿

### Phase 3 — 运行时同步 + relay 审计

- 迁移 `layer-graph-push`, `layer-changes-push`, `layer-github-oauth-access-tokens`
- 迁移 `relay-to-trae/status-push/`
- Django 公网旧路由标记 deprecated（日志 warn），文档更新

### Phase 4 — 清理

- 删除 Django 公网 `instance_callback_urls` 中已迁移项（或保留 308 到 :8006 一个版本）
- runAll 默认容器只连 taskAgentSupport

---

## 领域概念清单（供 /5-ddd）

| 类型 | 名称 | 说明 |
|------|------|------|
| **Bounded Context** | Container Agent Support | 容器/relay → 平台的 inbound 协作边界 |
| **Bounded Context** | Container Token Lifecycle | token 换票/续期/审计（仍属 saas-backend 核心域） |
| **Entity** | ContainerTokenSession | access/refresh 会话（Django 持久化） |
| **Entity** | CloudServerConfig | 任务级容器配置 |
| **Entity** | ContainerTokenAuditEvent | 审计时间线 |
| **Value Object** | TaskScope | tenant + workspace + task |
| **Value Object** | AccessToken / RefreshToken | 不透明令牌 |
| **Domain Event** | ContainerHeartbeatReceived | heartbeat 触达 |
| **Domain Event** | ContainerTokenExchanged | 换票成功/失败 |
| **Domain Event** | LayerGraphPushed | zTree 快照上报 |

taskAgentSupport 首期仅为 **Adapter / Anti-Corruption Layer**，不新增聚合根。

---

## 价值流影响

| 价值流 | 影响 |
|--------|------|
| `relay-token-audit-observability` | inbound 入口变为 taskAgentSupport；审计字段不变，需新增 `source_component=taskAgentSupport` 可选 |
| `task-detail-runtime-relay` | heartbeat/SSE 路径不变，Django internal 仍 publish taskSSE |
| `relay-status-push-timeout-go-relay` | status-push URL host 改为 :8006；go_relay 配置项更新 |
| `task-detail-relay-debug-agent-observability` | outbound.log 仍 onlineServiceJS；可选 Go access log |
| `layer-oauth-fetch-onlineServiceJS` | OAuth 拉票 URL 指向 taskAgentSupport；gitOauth 仍 Django 调 |

**新价值流（建议 step 3 登记）：** `task-agent-support-inbound-split` — 容器 inbound 与 saas-backend 解耦。

**测试影响：**

- 现有 `test_container_runtime_tokens.py` 等 → 改为经 :8006 打公网路径，或直连 internal API
- 新增 `taskAgentSupport` Go handler 测试 + 契约测试（公网 path ↔ internal path 1:1）
- Playwright relay 直启回归

**Cross-stream：** 与 `2026-05-28-task-sse-sidecar-design` 同类动机（减轻 Django runserver 压力）；与 centralized-logs traceId 设计协同。

---

## 验收标准

1. relay 直启任务：heartbeat 每 20s **200**，且 **不**因同机 `container-layer-git-push` pending 而 heartbeat 10s+ 超时。
2. onlineServiceJS `TaskApiEndPoint` 指向 :8006 时，bootstrap clone + zTree layer-graph-push 行为与现网一致。
3. `ContainerTokenAuditEvent` 时间线完整（换票/401 可追踪）。
4. 公网路径与响应 JSON schema **无 breaking change**（onlineServiceJS 无需改路径拼接逻辑，仅 base URL 可变）。
5. runAll 一键启动含 `task-agent-support`，health check 通过。

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| Internal API 与公网 API 行为漂移 | 契约测试：同一 fixture 请求公网（经 Go）与直连 internal 响应一致 |
| 双入口并行期 token 混乱 | Phase 1–3 仅改 relay 环境；生产切换前冻结 Django 公网 handler 代码 |
| Go/Django  secret 泄露 | 仅 bind 127.0.0.1；secret 来自 port_config，不入库 |
| 增加一跳延迟 | 内网 <1ms；heartbeat 不敏感 |

---

## 实现状态

- [x] Phase 1：`taskAgentSupport/` Go 网关（:8011）+ Django `/api/internal/task-agent-support/*` + `runAll.yaml` `task-agent-support`
- [x] `get_relay_task_api_base_url()` 在 `taskAgentSupport.enabled` 时指向 :8011
- [ ] Phase 2–4：bootstrap 全量迁移、deprecated 公网路由清理
