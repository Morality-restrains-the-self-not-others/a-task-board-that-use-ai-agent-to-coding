# relay register 换票窗口保护设计

**日期：** 2026-05-29  
**状态：** 已评审修订（方案 C′）  
**关联排障：** trace `c194f3c872464dfa960a8a3073036385`（`无效的 access_token` / `无效的 refresh_token`）

## 评审修订（2026-05-29，/8-review 后）

**方案 C′ = 已实现 C + 下列修订：**

| 修订项 | 内容 |
|--------|------|
| **go_relay register 契约** | `handleRegister` 在校验前调用 `resolveAccessTokenForRegisterLocked`；body 无 access 时复用 `state.AccessToken`；二者皆空则 **200 deferred**（仅确认 scope，不写入空 token） |
| **precheck/token-init** | TEIP 时 HTTP **409** + `TOKEN_EXCHANGE_IN_PROGRESS`，不再 400 |
| **TEIP 窗口 register** | 禁止 Django 新签 access；go_relay 不得用作废 state token 覆盖 `RegisteredTask` |
| **DDD** | 应用层统一经 `ContainerTokenBootstrapPolicy`（删除三处 inline TEIP if） |

原方案 C 第 3 节「handleRegister 已支持省略 access」与代码不符，以 C′ 为准。

## 背景与问题

任务详情页 `?relayToTrae=true` 直启链路为：

`token-init` → `repo-credentials-precheck` → `start`（202）→ go_relay 内 `exchange-refresh` → `refresh-access` → onlineServiceJS 启动。

审计日志显示：在 `exchange_refresh` 成功写入 `refresh_token` 后约 **4ms**，又出现一次 `relay_register`，且 access 后缀从 `V8_GYXu3` 变为 `jW_t6wnC`；随后 `refresh-access` 返回 `401 无效的 refresh_token`，go_relay fallback 到已作废的预埋 access，最终触发 `无效的 access_token`。

根因不是「refresh-access 未返回新 access」，而是 **register 路径在换票窗口内再次调用了 bootstrap 签发逻辑，清空了刚写入的 refresh_token**。

## 为什么 token-init/start 之后还会触发 relay register？

### 前端调用点（`ServerConfig.logic.vue`）

| 时机 | 函数 | `issueTokenOnServer` | 说明 |
|------|------|----------------------|------|
| 页面挂载 / 切换 relay 面板 / 任务变更 | `fetchRelayToTraeServiceStatus` | **true** | health 成功后 **必定** `registerRelayToTraeTask` |
| 用户点击「直接启动」且 `start` 返回 202 后 | `startRelayToTrae` 末尾 | **true** | 启动受理后 **再次** register |
| 用户手动刷新状态 | `@refresh-status` | **true** | 同上 |

`registerRelayToTraeTask({ issueTokenOnServer: true })` 会把 `ACCESS_TOKEN` 设为占位符 `__TASK2APP_ACCESS_TOKEN__` 发给 Django `relay-to-trae/register/`。

### Django register 路径（`relay_to_trae_proxy.py`）

```text
relay_to_trae_register
  → _access_token_needs_server_issue(placeholder) == true
  → _issue_relay_access_token
      → build_relay_to_trae_runtime_env
          → _replace_access_token_placeholder
```

### 危险的 bootstrap 逻辑（`mock_run_container.py`）

当 `CloudServerConfig.container_refresh_token` 非空时（**exchange-refresh 刚写完的状态**）：

```python
if cfg.container_refresh_token:
    cfg.container_refresh_token = ""      # 清空 refresh
    access = generate_opaque_token()      # 签发新预埋 access
    cfg.container_access_token = access
```

设计意图：每次 mock/relay 直启应用 **新预埋 access** 走 `exchange-refresh`；若库中仍有 refresh，复用旧 access 会导致 exchange 返回 403。

副作用：在 **go_relay 正在执行 refresh-access 的换票窗口** 内，任何再次进入 `_replace_access_token_placeholder` 的 register 都会 **销毁未消费的 refresh**，打断两步换票。

### 时序（实测 trace 还原）

```text
13:24:17.294  token-init     → 签发预埋 access (V8_GYXu3)
13:24:17.345  start 202       → 异步 /v1/start，env 含同一 access
13:24:17.358  exchange_refresh → access 作废，refresh=8pUbzcsI 写入 DB
13:24:17.362  relay_register   → issueTokenOnServer，清空 refresh，签发 jW_t6wnC  ← 问题点
13:24:17.537  refresh-access   → 401 无效的 refresh_token（8pUbzcsI 已被清空）
13:24:17.565  go_relay        → fallback 使用已作废的 V8_GYXu3
```

**结论：** register 在 start 之后触发是 **前端刻意设计**（保证 go_relay 侧 status-push 登记），但 **与 go_relay 同步换票重叠**，且 register 的 server-issue 路径未感知「换票进行中」状态。

### 各 register 调用的必要性

| 调用 | 是否必要 | 备注 |
|------|----------|------|
| 挂载时 health→register | 部分必要 | 需在 relay 在线时登记 task_id 供 status-push；但 **不应在换票窗口重签 token** |
| start 202 后 register | **冗余** | go_relay `handleStart` 已在内部 `registerTaskLocked`；Django 再 register 且 `issueTokenOnServer` 有害 |
| 手动刷新 register | 同挂载 | 应复用已有 token，禁止 bootstrap |

## 领域概念清单（供 `/5-ddd`）

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | Container Token Lifecycle | exchange-refresh / refresh-access / bootstrap |
| Bounded Context | Relay Startup Orchestration | token-init → precheck → start → register |
| Aggregate | `ContainerTokenSession`（`CloudServerConfig` 映射） | 当前 access/refresh 态 |
| Aggregate | `RelayStartupSession` | 两步启动工作流（已有 `RelayTwoStepStartupService`） |
| Domain Event | `exchange_refresh` | access→refresh，access 作废 |
| Domain Event | `refresh_access` | refresh→新 access |
| Domain Event | `token_bootstrap_blocked` | **新增**：换票窗口内拒绝重签 |
| Domain Event | `relay_register_attempted` | 已有审计 |

## 价值流影响（`value-stream.yaml`）

| 流 | 影响 |
|----|------|
| `task-detail-runtime-relay` | 直启后不得因 register 打断换票；status 收敛仍依赖 register 登记 |
| `relay-token-audit-observability` | 新增 `token_bootstrap_blocked` / `relay_register_reused_token` 事件 |
| `relay-token-exchange-no-proxy-consistency` | 保护 `container_refresh_token` 在换票窗口不被清空 |
| `relay-token-audit-full-chain-eventization` | register 审计需区分「复用 token」与「重签 token」 |

**字段影响：**

- `saas-backend.cloud_cloudserverconfig.container_access_token`
- `saas-backend.cloud_cloudserverconfig.container_refresh_token`
- `saas-backend.cloud_container_token_audit_event.event_type` / `error_code`

**测试影响：**

- 新增：`tests/test_relay_register_token_exchange_guard.py`（或并入 `test_relay_to_trae_proxy.py`）
- 更新：`TaskDetail.relay-to-trae-direct-start.playwright.test.js`（断言 start 后无二次 bootstrap）
- 回归：`tests/test_container_runtime_tokens.py::test_exchange_refresh_then_refresh_access_updates_db`

## 设计目标

1. **Django 硬保护**：换票窗口内（refresh 已写入、新 access 尚未签发）禁止 `_replace_access_token_placeholder` 清空 refresh 或重签 access。
2. **register 语义收敛**：register 只负责「把 task 登记到 go_relay」，**不是**第二次 token-init。
3. **可观测**：阻断时写审计事件，error_code 可追踪。
4. **不破坏合法重启**：用户显式重新 env-prepare / 新一次 token-init 后仍可获得新预埋 access。

## 换票窗口判定（推荐）

定义 **Token Exchange In Progress（TEIP）** 状态：

```text
TEIP(cfg) :=
  container_refresh_token != ""
  AND container_access_token == ""
  AND container_access_token_expires_at IS NULL
```

这与 `exchange_server_container_refresh_token` 成功后的 DB 态一致；`refresh_server_container_access_token` 成功后会写入新 access，TEIP 结束。

可选加强：结合 `RelayStartupSession` 中 `start_dispatch_attempted` 且未完成 `relay_start_succeeded` 的时间窗（默认 60s TTL），避免陈旧 TEIP 误伤。

## 方案对比

### 方案 A：仅 Django 保护（推荐为必做底线）

在 `_issue_relay_access_token` / `_replace_access_token_placeholder` 入口：

- 若 `TEIP(cfg)`：  
  - **register / precheck 场景**：返回现有 refresh 对应的「等待 refresh-access」错误或 **复用 workflow 已签发的末次 access**（若内存/审计可关联）；  
  - **禁止**清空 `container_refresh_token`、禁止 `generate_opaque_token()`。
- register 若收到占位符且 TEIP：改为 **仅转发 register 到 go_relay，不重新签发**（body 不带 access 或带 `register_only: true`）。

| 优点 | 缺点 |
|------|------|
| 单点修复，覆盖所有调用方（含未来） | 需定义 register 无 token 时 go_relay 行为 |
| 与 trace 根因直接对齐 | TEIP 判定需单测覆盖边界 |

### 方案 B：仅前端去重

- 删除 `startRelayToTrae` 末尾 `registerRelayToTraeTask({ issueTokenOnServer: true })`。
- `fetchRelayToTraeServiceStatus` 中 register 改为 `issueTokenOnServer: false`，仅在占位符且无 TEIP 时由用户显式 env-prepare。

| 优点 | 缺点 |
|------|------|
| 减少无效请求 | 无法防止其他客户端/竞态 |
| 改动面小 | 挂载 register 仍可能在其他路径撞窗口 |

### 方案 C′：C + go_relay 契约 + 409 契约（**当前推荐**）

在方案 C 基础上补齐跨组件契约（见「评审修订」）。

**推荐 C′**：Django TEIP 硬保护 + 前端减载 + go_relay register 复用 state / deferred + precheck 409。

## 详细设计（方案 C′）

### 1. Django：`_replace_access_token_placeholder` 保护

```python
def _is_token_exchange_in_progress(cfg: CloudServerConfig) -> bool:
    return bool(
        str(cfg.container_refresh_token or "").strip()
        and not str(cfg.container_access_token or "").strip()
        and cfg.container_access_token_expires_at is None
    )
```

当 `TEIP` 且调用来源为 `relay_register` / `relay_precheck` / `relay_start`（通过 `caller` 参数区分）：

- **不修改** `container_refresh_token` / `container_access_token`。
- 若调用方需要 access（precheck）：返回 `409 TOKEN_EXCHANGE_IN_PROGRESS`，detail 提示「换票进行中，请稍后重试」。
- 若调用方为 register：走 **register-only** 分支（见下）。

写入审计：`token_bootstrap_blocked`，`error_code=TOKEN_EXCHANGE_IN_PROGRESS`。

`token-init` / 显式 `env-prepare`：**仍允许**在 TEIP 时返回错误，引导用户等待当前 start 完成，避免人工双启。

### 2. Django：`relay_to_trae_register` 语义拆分

新增请求意图（向后兼容）：

| 字段 | 含义 |
|------|------|
| `register_only: true` | 只登记 task/scope 到 go_relay，**禁止** server-issue |
| 缺省（旧行为） | 占位符时 server-issue；TEIP 时改为 register-only 或 409 |

TEIP 时推荐行为：

1. 不调用 `_issue_relay_access_token`。
2. 向 go_relay `/v1/register` 发送 `tenant/workspace/task/origin`，**省略 access_token**（或传空，go_relay 使用 `state.AccessToken` 已有值）。
3. 审计：`relay_register_reused_state`，`source_component=django`。

### 3. go_relay：`/v1/register` 可选 access（C′ 修订）

```go
// handleRegister 伪代码
stateMu.Lock()
resolved := resolveAccessTokenForRegisterLocked(bodyAccessToken)
if taskAPIOrigin == "" { return 400 }
if resolved == "" {
    writeJSON(w, 200, {"status":"deferred","task_id":taskID})
    return  // 不调用 registerTaskLocked，等待 handleStart 内 register
}
registerTaskLocked(..., resolved)
stateMu.Unlock()
```

- body 有 access → 与 state 合并后 `resolveAccessTokenForRegisterLocked`（**state 优先**）
- body 无 access、state 有 token → 用 state 登记
- 二者皆空 → **200 deferred**（health 探测在 start 前可接受）

### 4. 前端：`ServerConfig.logic.vue`

| 变更 | 说明 |
|------|------|
| 删除 `startRelayToTrae` 末尾 register | go_relay `handleStart` 已 register |
| `fetchRelayToTraeServiceStatus` | register 使用 `register_only: true` 或 `issueTokenOnServer: false` |
| 仅在 token-init 成功后缓存 `lastBootstrapAccessSuffix`（可选） | 供调试对比，不用于鉴权 |

### 5. status-push 缓存失效（关联修复，可同 PR）

`exchange_refresh` / `refresh_access` 成功后，失效 `_CFG_CACHE` 中对应 access 条目，避免 120s 内误报 `status_push_ok`。

### 6. go_relay：取消危险 fallback（关联修复，可同 PR）

`exchange-refresh` 成功后若 `refresh-access` 失败，**不得** `using original token`；应 fail-fast 启动。

## 错误契约

| error_code | HTTP | 场景 |
|------------|------|------|
| `TOKEN_EXCHANGE_IN_PROGRESS` | 409 | TEIP 期间请求 bootstrap / precheck 需 access |
| `TOKEN_BOOTSTRAP_BLOCKED` | 409 | 显式 env-prepare 与进行中的换票冲突 |
| `TOKEN_ACCESS_INVALID` | 401 | 不变 |

响应均含 `trace_id`。

## 测试计划

### 后端单测

1. `exchange_refresh` 后立刻调用 `_issue_relay_access_token` → 不断言 refresh 被清空；返回 TEIP 错误或 register-only 成功。
2. `relay_to_trae_register` + 占位符 + TEIP → go_relay mock 收到无 access 的 register；审计 `token_bootstrap_blocked` 或 `relay_register_reused_state`。
3. 完整链路：`exchange_refresh` → `refresh_access` 仍成功（回归现有测试）。

### 集成 / Playwright

1. 直启：register 调用次数中，**start 202 之后不得再触发 server-issue**。
2. 日志/SSE 不得出现 `无效的 access_token`（已有回归用例扩展）。

## 非目标（本设计不做）

- 合并 token-init 与 start 为单次 API（Phase 2 startup storm）。
- 全面实现 go_relay `skip_token_exchange`（可另开任务，但与 TEIP 保护正交）。

## 开放问题（评审时确认）

1. **TEIP 超时**：若 go_relay 崩溃在 exchange 之后、refresh 之前，refresh 永久残留导致无法 bootstrap——是否需要 TTL（如 5 分钟）自动清理 TEIP？建议：首版仅保护窗口 + 文档说明「停止后重试」；TTL 作为 follow-up。
2. **register_only 默认**：是否在 TEIP 时自动降级为 register_only，还是对旧客户端返回 409？建议：**自动降级**（更安全），并写审计。

## 评审检查清单

- [x] TEIP 判定与 `exchange_refresh` 写库字段一致
- [x] register 不再在 start 后重签 token
- [ ] precheck 在 TEIP 返回明确 409 而非 400（C′）
- [x] 审计事件可串联 trace 排障
- [ ] go_relay register-only 契约测试绿（C′）
- [ ] 价值流 `task-detail-runtime-relay` 回归通过
