# 设计：relay 清理启动日志 — 服务端清空（作用域三元组）

- **日期**: 2026-07-09 20:10
- **状态**: approved（goal 自主批准）
- **作者**: claude
- **迭代**: relay clear-logs server-side（path 作用域）
- **python_api_approval**: approved — Gateway 主路径 + Django thin forward 兜底（2026-07-09）
- **overall_design_approval**: approved（2026-07-09，/goal 零交互落地）
- **相关意图**:
  - `task2app/docs/intents/frontend/task_detail/007_relay_clear_logs_no_refill.intent.md`（前端抑制）
  - `task2app/docs/intents/frontend/task_detail/008_relay_clear_logs_server_side.intent.md`（服务端清空）
- **前置问题**: 用户要求 `clear-logs` 携带完整 TenantID / WorkspaceID / TaskID，且**作用域落在 path（非 query）**以便路由调度

---

## 1. 架构现状理解（基线）

根据 `docs/architecture/` 与 `VERSION_HISTORY.md`：

| 项 | 内容 |
|----|------|
| 视图 | `enterprise-landscape`、`application-integration`（v1 标 current；最新交付态为 **v11 shipped**） |
| 应用层关键组件 | Vue(:4000) → task-gateway/APISIX → **taskContainerGateway** → **go-relay(:8797)**；saas-backend Django 为 accounts/company/internal 真源 |
| 技术层 | go-relay 持有本机 onlineServiceJS 生命周期与启动日志缓冲；status-push 经 Django/SSE 回前端 |
| 版本历史 | v9–v11 云域/凭证 Go SSOT；relay 生命周期已由 Gateway L0 代理（health/status/register/start/stop/token-init/…） |

**本次需求**在既有「Vue → Gateway → go-relay」链路上增加 **clear-logs**，不新增独立微服务。

📋 架构版本历史（节选）：
- v10 ✅ shipped — Cloud Platform CPA + Aliyun Query Go SSOT
- v11 ✅ shipped — Vendor Cloud Test Credentials SSOT
- 本次拟在 **v11** 基础上设计 **v12 target**（application-integration 为主）

---

## 2. 问题与修正动机

### 2.1 原草案缺陷

| 项 | 原草案 | 问题 |
|----|--------|------|
| go-relay | `POST /v1/clear-logs?task_id=` | 作用域在 query，不利于按路径段做路由/鉴权匹配；缺 tenant/workspace |
| 中间修订 | JSON body 三元组 | 作用域不在 path，与公网 URL 双源；不利于网关按 URI 调度 |
| 校验 | 缺 task_id → 400 | 无法校验「该 task 是否属于声明的 tenant/workspace」 |

### 2.2 目标

用户点击「清理日志」时：

1. 前端清空面板 + 保持抑制（007 已交付）
2. **同步**调用服务端清空 go-relay 缓冲（真相源）
3. **完整 TaskScope 全部落在 path**（`tenant` / `workspace` / `task`），**禁止**用 query 承载作用域，便于 APISIX / Gateway / go-relay 按路径段做路由调度与鉴权匹配

---

## 3. 推荐方案（已选定）

### 3.1 端到端调用链

```text
Vue「清理日志」
  → POST /api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/relay-to-trae/clear-logs/
  → APISIX (relay-to-trae-proxy，按 path 段匹配)
  → taskContainerGateway handleRelayClearLogs（从 URL path 解析 t/w/task）
  → POST go-relay /v1/tenant/{t}/workspace/{w}/task/{task}/clear-logs
  → clearLogsForScopeLocked(tenant, workspace, task)
```

Gateway 关闭 / Django forward 兜底时：同公网 path 由 `CloudComputeViewSet` → `relay_to_trae_clear_logs` →  
`_relay_http_request("POST", f"/v1/tenant/{t}/workspace/{w}/task/{task}/clear-logs")`（**无 query、无 scope body**）。

### 3.2 go-relay API（path 作用域）

**`POST /v1/tenant/{tenant_id}/workspace/{workspace_id}/task/{task_id}/clear-logs`**

| 项 | 约定 |
|----|------|
| Auth | 与现有 `/v1/*` 相同：`X-Relay-To-Trae-Secret`（若配置） |
| Path（必填） | `tenant_id`、`workspace_id`、`task_id` 三段均非空 |
| Query | **禁止**承载作用域；本接口不定义任何必需 query |
| Body | **不要求**；可为空。不以 JSON 作为作用域真源（避免与 path 双源） |
| 路由实现 | Go 1.24 `ServeMux` 模式：`POST /v1/tenant/{tenant_id}/workspace/{workspace_id}/task/{task_id}/clear-logs` + `r.PathValue(...)` |

**废弃 / 不接受的契约：**

- ❌ `POST /v1/clear-logs?task_id=...`
- ❌ `POST /v1/clear-logs` + JSON `{tenant_id,workspace_id,task_id}` 作为唯一作用域来源

**校验规则：**

1. path 三段 trim 后均非空，否则 `400` + `message: missing tenant_id|workspace_id|task_id`
2. 若 `registeredTasks[task_id]` 存在：其 `TenantID`/`WorkspaceID` 必须与 path 一致，否则 `403` + `message: scope mismatch`
3. 若任务尚未登记：仍允许按 path `task_id` 清空该任务缓冲；path 中的 tenant/workspace 写入审计日志，不因「未登记」而 404
4. 清空成功后：`taskLogs[task]=[]`；若 `ActiveTaskID==task` 则 `state.Logs=[]`；该 task 的 `LogCursor=0`

**成功响应 `200`：**

```json
{
  "status": "ok",
  "cleared": true,
  "tenant_id": "...",
  "workspace_id": "...",
  "task_id": "...",
  "cleared_live": true
}
```

### 3.3 Gateway / APISIX / Django

| 层 | 变更 |
|----|------|
| APISIX `routes.yaml` + `apisix.yaml` | 增加 `.../relay-to-trae/clear-logs*` 两条 URI（与现有 stop/status 同形，作用域已在更前 path 段），upstream 仍为 taskContainerGateway |
| taskContainerGateway | 从公网 path 解析 t/w/task；转发 `POST /v1/tenant/{t}/workspace/{w}/task/{task}/clear-logs`（**不**拼 query、**不**依赖 body 作用域） |
| Django `CloudComputeViewSet` | `POST relay-to-trae/clear-logs` + `require_django_forward('relay-to-trae-clear-logs')`；用 URL kwargs 拼 go-relay path 同步转发 |
| `django_forward_guard.MIGRATED_OUTBOUND_ACTIONS` | 加入 `relay-to-trae-clear-logs` |

### 3.4 前端

`clearRelayToTraeLogOutput`：

1. 先本地清空 + `suppressedAfterClear=true`（现有）
2. `apiFetch(relayToTraeApiUrl('clear-logs/'), { method:'POST' })` — 作用域已在 `relayToTraeApiUrl` 的 tenant/workspace/task path 中，**无需**再传 query/body 作用域
3. 失败：`relayToTraeMessage` 提示「服务端日志清理失败」，**不**回滚本地清空

### 3.5 领域概念（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | Task Runtime / Relay Lifecycle |
| Value Object | `TaskScope(tenant_id, workspace_id, task_id)` |
| 行为 | `ClearStartupLogs(scope)` — 清空该 scope 的启动日志缓冲与 push cursor |
| 非目标 | 不清 Loki；不停 onlineServiceJS |

---

## 4. 🐍 Python 新增接口清单与 Go 替代评估

### 拟新增接口

| # | 方法 | 路径 | 归属服务 | 公网/internal | 流量特征 | 说明 |
|---|------|------|----------|---------------|----------|------|
| 1 | POST | `/api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/relay-to-trae/clear-logs/` | **主路径：taskContainerGateway (Go)**；Django 仅 Gateway 关闭时的 thin forward | 公网（经 APISIX token） | 短、低频 | 清空 go-relay 启动日志缓冲；作用域在 path |
| 2 | POST | `/v1/tenant/{t}/workspace/{w}/task/{task}/clear-logs` | go-relay (Go) | internal（secret） | 短 | 真相源清空；作用域在 path，无 query |

### Go 替代方案

| # | 方案 | 目标 | 优点 | 缺点 | 推荐 |
|---|------|------|------|------|------|
| 1 | Gateway L0 直连 go-relay | taskContainerGateway | 与 stop/status 一致；无 Django 拥塞 | 需改 APISIX + Gateway | ⭐ |
| 2 | 仅 Django proxy | saas-backend | 实现快 | 扩大 Python 公网面；与 target「relay 已迁 Gateway」冲突 | ❌ |
| 3 | 复用 stop | — | 零新 API | 语义错误（会停进程） | ❌ |

### 选型结论

- **最终选择**: **Go 主路径（Gateway → go-relay）**；Django 仅保留与 `stop`/`status` 同模式的 **thin forward 兜底**（Gateway 未启用时），不在 Django 内实现业务逻辑。
- **Python 面**: 若落地 Django `@action`，属「与既有 relay-* 对称的 1 个 thin proxy」；推荐审批为 **批准 Python thin forward（低频）** 或 **scoped：仅 Go，Django 不新增**（Gateway 常开环境可接受）。
- **Swagger**: 交付时须登记 `POST .../clear-logs/` 的 request/response schema 与 400/403/502。

> `python_api_approval`: **待用户确认**（见下方审批选项）

---

## 5. 价值流影响（输入给 step 3）

| 流 | 影响 |
|----|------|
| `task-detail-runtime-relay` / `relay-stop-refresh-status-ui` | 新增步骤：清理日志 → 服务端 clear-logs → 刷新无历史 |
| `tests/test_relay_to_trae_proxy.py` | 新增 clear-logs 转发测例 |
| go_relay handlers_test | 作用域必填 / mismatch / 成功清空 |
| Playwright `TaskDetail.relay-to-trae-direct-start` | 断言发起 clear-logs 且 body 含三元组 |

---

## 6. 🏛️ 架构变更影响

- **迭代版本**: v12 🎯 target
- **迭代名称**: relay clear-logs server-side (path scope)
- **作者**: claude
- **设计日期**: 2026-07-09 20:15
- **新增文件**（application-integration 三类伴生）:
  - 🆕 `docs/architecture/v12-application-integration-20260709-2015-claude.puml`
  - 🆕 `docs/architecture/v12-application-integration-20260709-2015-claude.archimate`
  - 🆕 `docs/architecture/v12-application-integration-20260709-2015-claude.mermaid.md`
- **已有文件（未修改）**:
  - `docs/architecture/v11-*-*.puml` 及伴生
- **变更明细**: 🟡 go-relay / Gateway / APISIX / Vue / Django thin forward

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v11** | Vendor credentials SSOT 基线 |
| **Plateau v12** | path-scoped clear-logs |
| **Gap** | UI 清理不触达服务端缓冲 |
| **WorkPackage** | clear-logs path 契约全链路 |
| **视图** | 沿用 v11 连线结构；mermaid 补充 Vue→CGW→go-relay clear-logs 流 |

---

## 7. 测试计划（摘要）

| 层 | 用例 |
|----|------|
| go-relay | path 缺段 → 404/400；registered scope mismatch → 403；匹配 → logs/cursor 清空；带 `?task_id=` 的旧契约不作为合法主路径 |
| Gateway | 从公网 path 解析 t/w/task，转发至 go-relay **同形 path**（无 query） |
| Django proxy | 用 kwargs 拼 go-relay path；worker 未配置 → 503 |
| Playwright | 清理后请求 URL 含 tenant/workspace/task/clear-logs；刷新不回填 |

---

## 8. 非目标

- 不清空 Loki/Grafana
- 不停止 onlineServiceJS / 不清 container reachability
- 不改变 007 前端抑制逻辑（双保险保留）

---

## 9. 变更记录

| 日期 | 变更 |
|------|------|
| 2026-07-09 | 初版意图仅 `task_id` query |
| 2026-07-09 | 修正为 JSON body 三元组 |
| 2026-07-09 | **再修正**：作用域全部改 **path**（公网与 go-relay 同形）；禁止 query/body 作为作用域真源，便于路由调度 |
