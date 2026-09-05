# relayToTrae 直启启动可靠性修复

**日期：** 2026-05-29  
**状态：** 已由 `2026-05-31-relay-to-trae-startup-reliability-design.md` 取代（合并 post-listen 引导与 configFilePath 回归）  
**状态（历史）：** 待审批（0-auto-flow Step 1）

## 背景

任务详情页 `?relayToTrae=true` 直启链路当前存在三类用户可见故障：

| 现象 | 典型日志 |
|------|----------|
| onlineServiceJS 启动后立即 exit 1 | `reachability 失败 … register-reachability/: {"detail":"internal error"}` |
| 启动日志面板行为异常 | 清理后刷新又回填 / 启动失败后刷新无日志（**前端已部分修复，需回归锁定**） |
| relay 状态推送失败 | `status push HTTP 404: 404 page not found` |

根因链：`go_relay` → `onlineServiceJS` listen → `register-reachability` / `exchange-refresh` 经 **taskAgentSupport :8011** 转发 Django internal API；任一环节 500/404 会导致容器进程 `process.exit(1)`，Django 转发层图时 `Connection refused`。

已有缓解见 `docs/superpowers/specs/2026-05-29-startup-storm-mitigation.md`（审计移出 atomic、心跳延迟、bootstrap stagger），但 **internal error 仍无结构化错误面**，且 **status-push 路由 404** 未收敛。

## 目标

1. relay 直启后 `onlineServiceJS` 能在 `register-reachability` 成功后持续监听 8765。
2. `exchange-refresh` / `register-reachability` 失败时返回可诊断的 `error_code` + `detail`，不再统一 `internal error`。
3. 启动日志面板：启动失败后可刷新看到日志；用户清理后刷新不再回填（**已实现，本迭代加测试锁定**）。
4. `go_relay` status-push 能到达 Django 并驱动 SSE 收敛（消除 404）。

## 非目标

- Phase 2 启动风暴（合并 bootstrap batch API、dev 默认 PostgreSQL）— 仅记录 follow-up。
- 去掉 onlineServiceJS watch 模式 — **已完成**。
- Grafana 日志级别从 warning 改 error — 不在本迭代。

## 价值流影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `task-detail-runtime-relay` | `relay-status-convergence` | status-push 404 修复后 SSE 状态与 relay 日志一致 |
| `relay-token-audit-full-chain-eventization` | `relay-register-start-audit` | exchange-refresh 错误需保留审计事件 |
| `relay-status-push-timeout-go-relay` | `relay-status-push-no-proxy-thin-slice` | 路由对齐后回归 |
| `task-detail-relay-debug-agent-observability` | outbound visibility | register-reachability 失败需可追踪 trace_id |

**字段：** `cloud_cloudserverconfig.server_url`、`business_api_endpoint`、`container_access_token`；`cloud_container_token_audit_event.error_code/error_detail`。

## 根因假设（待 Step 7 验证）

1. **internal_dispatch 吞异常**：`task_agent_support/internal_dispatch.py` 将任意 `Exception` 映射为 `500 internal error`，掩盖 SQLite `database is locked`、SSE 发布、或 ORM 竞态。
2. **启动风暴残余**：listen 后 `register-reachability` 与 `exchange-refresh`（relay 换票）仍可能重叠写 `CloudServerConfig`。
3. **status-push URL 漂移**：`go_relay` 仍 POST 公网/旧路径 `.../relay-to-trae/status-push/`，而 inbound 已迁至 taskAgentSupport 或 internal 路由未注册。
4. **reachability 硬失败**：`server.mjs` 在 register 失败时 `process.exit(1)`，无重试，放大瞬时 500。

## 方案对比

### 方案 A — 诊断优先 + 有限重试（推荐）

- internal_dispatch：记录 `exc_type`/`message` 到日志；对 `OperationalError`（SQLite locked）返回 **503 + `RELAY_DOWNSTREAM_BUSY`**，客户端重试。
- onlineServiceJS `registerReachabilityAfterBootstrap`：对 503/502 指数退避重试 3 次（总等待 ≤15s）。
- 修复 status-push 路由：对齐 `go_relay` push URL 与 Django `report_relay_to_trae_status` 实际挂载路径。
- 前端：锁定 `mergeRelayToTraeLogLines` 行为（Vitest 已覆盖，补 Playwright 冒烟可选）。

**优点：** 改动面小，直接解决用户可见失败；与现有 startup-storm 文档一致。  
**缺点：** 不消除风暴根因，仅降级为可恢复。

### 方案 B — register-reachability 移出 exit(1) 关键路径

- listen 后继续运行，后台重试 register；UI 显示「等待业务端点注册」。
- **优点：** 容器 API 可用，层图仍可能失败直到注册成功。  
**缺点：** 与「reachability 失败不回退 127.0.0.1」安全策略冲突；前端状态机更复杂。

### 方案 C — Phase 2 batch internal API

- 单次 internal 调用完成 exchange + reachability + 审计。  
**优点：** 根治写冲突。  
**缺点：** 工作量大，超出本迭代。

**推荐方案 A**：最小可行修复，可测试、可观测，与已落地 startup-storm Phase 1 衔接。

## 详细设计（方案 A）

### 1. Django `internal_dispatch`

```python
# 伪代码
except OperationalError as e:
    if 'locked' in str(e).lower():
        return JsonResponse({'detail': 'database busy', 'error_code': 'RELAY_DOWNSTREAM_BUSY'}, status=503)
    raise
except Exception:
    logger.exception(...)
    return JsonResponse({'detail': 'internal error', 'error_code': 'INTERNAL_DISPATCH_ERROR'}, status=500)
```

补充单元测试：`tests/test_task_agent_support_internal_dispatch.py`（新建）。

### 2. onlineServiceJS `reachability.mjs`

- `postJson` 收到 503/`RELAY_DOWNSTREAM_BUSY` 时重试（默认 3 次，间隔 200/500/1000ms，可 env 覆盖）。
- 最终失败：日志含 `error_code`；仍 `exit(1)`（保持现有安全语义）。

### 3. go_relay status-push 路由

- 核对 `push.go` 构造 URL 与 Django `relay_to_trae_status_views` / taskAgentSupport 转发链。
- 若应走 taskAgentSupport `relay-status-push` action，则改 `TASK_API_ENDPOINT` 前缀或专用 env。
- 测试：`go_relayToTrae/src/push_test.go` + `test_relay_to_trae_status.py`。

### 4. 前端启动日志（回归锁定）

- 已有 `mergeRelayToTraeLogLines` + `awaitingFirstStatusAfterStart` 标志。
- 本迭代不改动逻辑，仅确认 Vitest 39 项通过；可选补一条 Playwright「清理 → 刷新 → 仍为空」。

### 5. 可观测性

- internal_dispatch 异常日志带 `action`、`tenant_id`、`task_id`、`trace_id`。
- Grafana 验收：trace 内可见 register-reachability 失败原因非裸 `internal error`。

## 领域概念清单（供 DDD）

| 概念 | 边界上下文 |
|------|------------|
| `ContainerReachabilityRegistration` | 云平台 / 容器运行时 |
| `RelayStartupSession` | relayToTrae 编排 |
| `TaskAgentSupportInboundDispatch` | taskAgentSupport 网关 |
| 领域事件 | `ContainerEndpointRegistered`、`RelayStatusPushed` |

## 验收标准

1. 给定任务 `847744505890045952`（或等价测试夹具），relay 直启后日志无 `register-reachability … internal error`；8765 可访问 `/api/layers`。
2. 模拟 SQLite busy 时，onlineServiceJS 重试后成功或给出明确 `error_code`。
3. go_relay 日志无 `status push HTTP 404`。
4. 前端：清理日志 → 刷新状态 → 面板仍「暂无日志」；启动失败 → 刷新 → 可见错误日志。
5. `pytest` 相关用例 + `relayToTraeUtils` Vitest 全绿。

## 测试计划

| 层 | 文件 |
|----|------|
| Django internal | `tests/test_task_agent_support_internal_dispatch.py` |
| 容器 token | `tests/test_container_runtime_tokens.py`（reachability busy 分支） |
| go_relay | `push_test.go` |
| onlineServiceJS | `reachability.test.mjs`（重试） |
| 前端 | `relayToTraeUtils.test.js`（已有） |

## 风险与回滚

- 503 重试可能延长启动 1～2s — 可接受。
- status-push URL 变更需与生产 nginx 路由一致 — 仅改 internal 路径时风险低。
- 回滚：还原 dispatch 异常处理与 reachability 重试即可。
