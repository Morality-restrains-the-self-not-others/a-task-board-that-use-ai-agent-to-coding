# 设计文档：taskAgentSupport Phase 2 — 路径化 Internal API + Bootstrap 全量转发

**日期：** 2026-05-30  
**状态：** Phase 2 设计已批准；路径化 internal API + 12 action 注册已实现  
**前置：** Phase 1 已实现（Go :8011 + Django internal 5 action）；`2026-05-30-task-agent-support-internal-dispatch-request-fix` 已修复 Request 类型 500  
**触发故障：** relay 直启 bootstrap 调用 `task-detail` 返回 404，`path` 显示 `/api/internal/task-agent-support/task-detail/`（无 scope 段，且路由未注册）

---

## 1. 背景与问题

### 1.1 用户可见故障

任务详情 `?relayToTrae=true` 点击「启动」后，onlineServiceJS 容器日志：

```
bootstrap (post-listen) error: HTTP 404 .../cloud/server-container-token/task-detail/
{"detail":"未找到请求的 API 路径...","path":"/api/internal/task-agent-support/task-detail/"}
```

公网路径 **正确**（含 tenant/workspace/task），Go 网关 **正确**解析 scope；Django internal 侧 **404** 有两层原因：

| # | 原因 | 说明 |
|---|------|------|
| A | **路由未实现** | Phase 1 仅注册 5 个 action，`task-detail` / `repo-clone-credentials` 等未加入 `_VIEW_BY_ACTION` |
| B | **scope 不在 URL** | Internal 契约把 `tenant_id/workspace_id/task_id` 放在 JSON envelope；404 中间件只打印 URL path，排障时像「丢了 ID」 |

### 1.2 当前数据流

```
onlineServiceJS
  POST :8011/api/tenant/T/workspace/W/task/K/cloud/server-container-token/task-detail/
       body: { access_token }
    ↓ taskAgentSupport Go
  POST :8001/api/internal/task-agent-support/task-detail/
       body: { tenant_id, workspace_id, task_id, body: { access_token } }
    ↓ Django
  404 — urls 无 task-detail/；即使加了路由，path 仍不可读
```

### 1.3 用户诉求（本轮）

1. **把 ID 放到 Internal API 路径中** — 与公网 cloud 路径对齐，日志/404/Grafana 可直接看到 scope。
2. **解决其余问题** — 补齐 bootstrap 及运行时依赖的全部 inbound action，使 relay 直启端到端可用。

---

## 2. 目标与非目标

### 2.1 目标

1. Internal API 路径携带 `tenant_id / workspace_id / task_id`。
2. 请求体 **仅** 保留容器原始 payload（`access_token` 等），`trace_id` 走 Header。
3. 注册 **全部** 容器 inbound action（Phase 2 + Phase 3），复用现有 Django 视图，无业务逻辑重写。
4. Go 网关 `forwardToDjango` 同步改路径；**对外** cloud 路径不变，onlineServiceJS **零改动**。
5. 更新 pytest + Go handler 测试；relay 直启 bootstrap 绿。

### 2.2 非目标

- 不修改 onlineServiceJS 路径拼接逻辑。
- 不迁移 `cloud/compute/*` 出站转发。
- 不迁移 `repo-reclone/`、`server-userdata-verify/`（浏览器 / 云厂商探针，仍走 Django 公网）。
- 不在 Go 层实现 token 领域校验（仍在 Django 视图）。
- 不提供旧 envelope-only Internal URL 的长期兼容（仅本仓库内 Go + pytest 消费者，可一次性切换）。

---

## 3. 方案对比

### 方案 A（推荐）：路径化 scope + 扁平 action 注册

Internal URL：

```
POST /api/internal/task-agent-support/tenant/{tid}/workspace/{wid}/task/{tk}/{action}/
Header: X-TaskAgentSupport-Internal-Secret, X-Trace-Id
Body:   { "access_token": "...", ... }   ← 与公网容器 API 相同
```

| 优点 | 缺点 |
|------|------|
| 与公网 `/api/tenant/.../task/.../cloud/...` 结构对称 | 需一次性改 Go + Django urls + 测试 |
| 404/访问日志直接含 scope | 旧 envelope 契约废弃 |
| 单条 Django url 模式覆盖全部 action | — |

### 方案 B：保留 envelope，仅补注册 action

只加 `task-detail` 等到 `_VIEW_BY_ACTION`，路径仍为 `/api/internal/task-agent-support/{action}/`。

| 优点 | 缺点 |
|------|------|
| 改动最小 | **不满足**用户「ID 放路径」诉求 |
| — | 404 排障仍困难 |

### 方案 C：Internal 完全镜像公网路径

```
POST /api/internal/tenant/{tid}/workspace/{wid}/task/{tk}/cloud/server-container-token/{action}/
```

| 优点 | 缺点 |
|------|------|
| 与公网 1:1 | 路径过长；`relay-to-trae/status-push` 需特殊规则 |
| — | 与现有 `api/internal/task-agent-support/` 命名空间不一致 |

**推荐方案 A**：在 internal 命名空间下引入 scope 段，action 名保持扁平（与 envelope 时代一致）。

---

## 4. 详细设计（方案 A）

### 4.1 Internal 路由契约（新）

**基址：** `{djangoInternalApiBase}/api/internal/task-agent-support/`

**模板：**

```
POST .../tenant/<tenant_id>/workspace/<workspace_id>/task/<task_id>/<action>/
```

**action 全集（12 项）：**

| action | Django 视图 | 阶段 | 调用方 |
|--------|-------------|------|--------|
| `exchange-refresh` | `exchange_server_container_refresh_token` | P1 ✓ | onlineServiceJS 启动 |
| `refresh-access` | `refresh_server_container_access_token` | P1 ✓ | onlineServiceJS 续期 |
| `register-reachability` | `register_container_reachability` | P1 ✓ | onlineServiceJS |
| `heartbeat` | `report_container_heartbeat` | P1 ✓ | onlineServiceJS ~20s |
| `relay-status-push` | `report_relay_to_trae_status` | P1 ✓ | go_relayToTrae |
| `task-detail` | `fetch_container_task_detail` | **P2** | bootstrap / OAuth 刷新 |
| `repo-clone-credentials` | `fetch_container_repo_clone_credentials` | **P2** | bootstrap |
| `feature-params-env` | `fetch_tenant_feature_params_env_for_container` | **P2** | bootstrap |
| `git-clone-progress` | `report_container_git_clone_progress` | **P2** | clone 过程 |
| `layer-graph-push` | `report_container_layer_graph` | **P3** | zTree 同步 |
| `layer-changes-push` | `report_container_layer_changes` | **P3** | 层 diff |
| `layer-github-oauth-access-tokens` | `fetch_container_layer_github_oauth_access_tokens` | **P3** | 层内 git OAuth |

> 注：原设计稿写 `feature-params-yaml`，实现与 `instance_callback_urls.py` 一致为 **`feature-params-env`**。

### 4.2 Django 改动

**`cloud/urls_task_agent_support_internal.py`** — 单条 scope 路由替代多条 flat 路由：

```python
urlpatterns = [
    path(
        'tenant/<str:tenant_id>/workspace/<str:workspace_id>/task/<str:task_id>/<str:action>/',
        dispatch_task_agent_support_internal_scoped,
        name='task-agent-support-internal-scoped',
    ),
]
```

**`internal_dispatch.py`** 调整：

```python
def dispatch_task_agent_support_internal_scoped(
    request, tenant_id: str, workspace_id: str, task_id: str, action: str
) -> HttpResponse:
    # 1. check_task_agent_support_secret
    # 2. view_func = _VIEW_BY_ACTION.get(action) → 404 unknown action
    # 3. 校验 tenant_id/workspace_id/task_id 非空（path 捕获，通常已有）
    # 4. body = json.loads(request.body) — 直接作为容器 payload
    # 5. trace_id = request.headers.get('X-Trace-Id')
    # 6. view_func(http_request, tenant_id=..., workspace_id=..., task_id=...)
```

删除 envelope 解析（`tenant_id/workspace_id/task_id/body` 嵌套结构）。

**`_VIEW_BY_ACTION`** 扩展至上述 12 action。

### 4.3 Go 网关改动

**`django_client.go`：**

```go
url := fmt.Sprintf(
    "%s/api/internal/task-agent-support/tenant/%s/workspace/%s/task/%s/%s/",
    cfg.DjangoInternalAPI, tenantID, workspaceID, taskID, action,
)
// Body: 直接 marshal 容器原始 body，不再包 inboundEnvelope
```

**`config.go` / `port_config.json`** — 补充 action 超时（可选，沿用 `defaultSec: 30` 亦可）：

| action | 建议 timeout |
|--------|-------------|
| heartbeat | 5s |
| exchange-refresh / refresh-access | 15s |
| task-detail / repo-clone-credentials / feature-params-env | 15s |
| layer-github-oauth-access-tokens | 15s |
| git-clone-progress / layer-changes-push | 15s |
| layer-graph-push | 30s |

### 4.4 对外兼容性

| 层 | 是否变更 |
|----|----------|
| onlineServiceJS → taskAgentSupport 公网路径 | **不变** |
| taskAgentSupport → Django internal | **变更**（仅集群内） |
| Django 公网 `instance_callback_urls` | **不变**（`taskAgentSupport.enabled=false` 时仍可直接打 :8001） |

### 4.5 错误响应

- **unknown action** → `404 {"detail":"unknown action: xxx"}`
- **Django url 未匹配** → 现有 `ApiJson404Middleware` → `{"detail":"未找到...","path":".../tenant/T/workspace/W/task/K/unknown/"}`
- scope 与 token 不匹配 → 视图层 **403**（行为不变）

404 path 将 **包含完整 scope**，满足排障诉求。

---

## 5. 分步交付（建议单 PR）

| 步骤 | 内容 | 验收 |
|------|------|------|
| 1 | Django：路径化 url + dispatch 重构 + 12 action 注册 | pytest internal 全绿 |
| 2 | Go：`forwardToDjango` 改 URL + body | `handlers_test.go` 绿 |
| 3 | 更新现有 `test_task_agent_support_internal*.py` 路径 | CI 绿 |
| 4 | 新增 `task-detail` / `repo-clone-credentials` smoke | 401/200 非 404 |
| 5 | 手工：relay 直启任务 bootstrap 完成 clone | 无 bootstrap 404 |

---

## 6. 领域概念清单（供 /5-ddd）

| 类型 | 名称 | 说明 |
|------|------|------|
| **Bounded Context** | Container Agent Support | Go 网关 + internal 适配层 |
| **Bounded Context** | Container Token Lifecycle | token / 凭据 / 审计（Django 真源） |
| **Value Object** | TaskScope | tenant + workspace + task（路径化后为一等公民） |
| **Adapter** | TaskAgentSupportInternalDispatch | path-scoped 转发至 inbound 视图 |
| **Domain Event** | ContainerBootstrapCompleted | 依赖 task-detail + repo-clone-credentials 链 |

无新聚合根；仍为 Anti-Corruption Layer。

---

## 7. 价值流影响

| 价值流 | 影响 |
|--------|------|
| `task-detail-runtime-relay` | bootstrap `task-detail` 404 消除；relay 直启可拉任务上下文 |
| `task-detail-repo-clone-credentials-decoupling` | `repo-clone-credentials` internal 转发可用 |
| `task-detail-repo-clone-credentials-contract` | 契约测试需覆盖经 :8011 路径 |
| `relay-token-audit-observability` | internal 访问日志含 scope path |
| `layer-oauth-fetch-onlineServiceJS` | Phase 3 action 注册后 OAuth 拉票经网关 |
| `task-agent-support-inbound-split`（建议 step 3 登记） | Phase 2+3 完成，inbound 拆分实质可用 |

**字段：** 无 schema 变更；读写仍经 `cloud_cloudserverconfig`、`cloud_container_token_audit_event`。

**测试影响：**

| 文件 | 变更 |
|------|------|
| `tests/test_task_agent_support_internal.py` | 路径改为 scoped |
| `tests/test_task_agent_support_internal_dispatch.py` | 路径 + smoke 扩展 |
| `taskAgentSupport/src/handlers_test.go` | 可选：forward URL 断言 |
| `trae-agent/onlineServiceJS/e2e/bootstrap-clone-*.spec.mjs` | 无需改（打 mock 公网路径） |
| `playwright/.../TaskDetail.relay-to-trae-direct-start` | 回归直启 |

---

## 8. 验收标准

1. `POST :8011/.../server-container-token/task-detail/` → **200**（合法 token）或 **401/403**（鉴权失败），**非 404**。
2. bootstrap 日志无 `task-detail` / `repo-clone-credentials` / `feature-params-env` 404。
3. Internal 404（若故意打错 action）的 `path` 含 `tenant/.../workspace/.../task/...`。
4. Phase 1 action（heartbeat、exchange-refresh）经新路径仍正常。
5. `taskAgentSupport.enabled=false` 时，直打 Django :8001 公网路径行为不变。

---

## 9. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Go/Django 不同步部署导致 404 | 单 PR 同时改；runAll 重启两服务 |
| `feature-params-env` 依赖 DB 慢 | timeout 15s；现有 bootstrap 重试 |
| repo-clone-credentials 409/502 | 业务错误，非路由问题；UI 已有 OAuth 引导 |
| 路径 segment 含特殊字符 | tenant/workspace/task 均为 snowflake 数字 ID，无编码问题 |

---

## 8. 自检

- [x] 根因 A（未注册）与 B（scope 不可见）均已覆盖  
- [x] 用户诉求「ID 放路径」→ 方案 A  
- [x] onlineServiceJS 零改动  
- [x] 价值流 `task-detail-runtime-relay` 等已标注  
- [x] Phase 2 + Phase 3 一次补齐，避免 bootstrap 后连环 404
