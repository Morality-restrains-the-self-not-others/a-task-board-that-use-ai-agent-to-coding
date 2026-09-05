# Task Detail relayToTrae 停止后刷新状态误判设计

日期：2026-05-27  
状态：已实现

## 1. 问题陈述

在 task-detail 页面 `relayToTrae=true` 的「直接启动」面板中，用户操作流程：

1. 点击「启动」→ onlineServiceJS 运行中  
2. 点击「停止」→ onlineServiceJS 已停止  
3. 点击「刷新状态」

期望：`relayToTrae 服务：在线`，`onlineServiceJS：未启动`  
实际：`relayToTrae 服务：未连接`，`onlineServiceJS：未启动`

**onlineServiceJS 状态正确**；**relay 侧车可达性被误判为离线**，导致用户以为本机 relay 服务已挂掉。

## 2. 根因分析

状态刷新逻辑位于 `ServerConfig.logic.vue` 的 `fetchRelayToTraeServiceStatus`：

```
health/ → 失败? → 回退 status/ → 仍失败? → 重置 relayToTraeServiceOnline=false（显示「未连接」）
       → 成功 → register + status/ → applyRelayToTraeStatusPayload
```

### 2.1 并发竞态（主因）

`fetchRelayToTraeServiceStatus` **无请求序号/互斥**。以下路径会并发触发：

- `stopRelayToTrae` 末尾自动调用  
- 用户点击「刷新状态」  
- 切换「直接启动」Tab  
- `props.task` 轮询更新

典型竞态：

1. 请求 A：health 超时 → `relayToTraeServiceOnline=false`  
2. 请求 B：health 成功 → status 返回 stopped → `relayToTraeServiceOnline=true`  
3. 请求 A 回退 status 也失败 → **再次写入 offline，覆盖 B 的正确结果**

停止后用户立刻点「刷新状态」，极易触发此竞态。

### 2.2 离线判定过宽（次因）

health 失败且 status 回退失败时，会将 **relay 可达性** 与 **onlineServiceJS 运行态** 一并清零。  
即使 `stopRelayToTrae` 刚将 `relayToTraeServiceOnline=true` 写入，一次 transient health 失败也会覆盖为「未连接」。

已有「health 超时回退 status」修复（Playwright: `刷新状态时 health 短暂失败不应误判 relay 未连接`），但：

- 未覆盖 **stop → refresh** 场景  
- 未防止 **并发 stale 响应覆盖**  
- `isRelayStatusPayloadLike` 对 payload 形状要求过严，Django/空响应边缘情况可能漏判

### 2.3 stop 后多余的全量刷新

`stopRelayToTrae` 已在本地写入正确 stopped 态（`relayToTraeServiceOnline=true`，`onlineServiceUp=false`），再调用全量 `fetchRelayToTraeServiceStatus` 可能因上述竞态/瞬态失败**破坏**已正确的 UI。

### 2.4 手动刷新与 task 轮询未 preserve（生产仍复现）

已实现 `preserveRelayOnline` **仅**用于 `stopRelayToTrae` 末尾自动刷新。用户点击「刷新状态」与 `props.task` 详情轮询（`watch` → `fetchRelayToTraeServiceStatus()`）均**不带** preserve。

在 stopped 收敛态（`relayToTraeServiceOnline=true` 且 `onlineService` 未运行）下，若 health 与 status 回退同时失败，仍会调用 `applyRelayToTraeOfflineState()`，面板显示「未连接」并写入 `relayToTraeMessage`（`start.sh` 提示）。用户再点「启动」时，该 message 可能仍挂在面板上，造成「点启动也提示无法连接」的错觉（启动链路本身走 Django proxy，未必真的连不上 8797）。

Playwright 用例 `停止后刷新状态时 health 失败仍应保持 relay 在线` 已通过（mock 仅 health 失败、status 200），**未覆盖** status 也失败 + task 轮询并发。

## 3. 目标与非目标

### 目标

1. stop 后手动「刷新状态」时，relay 侧车在线、onlineServiceJS 未启动——状态稳定、可预期。  
2. health 偶发失败时，只要 status 可达，仍显示 relay 在线。  
3. 并发刷新不会互相覆盖；后发请求结果优先。  
4. Playwright 回归覆盖 stop → refresh 路径。

### 非目标

1. 不改动 go_relayToTrae 停止语义或 Django async stop 架构。  
2. 不重构 SSE 状态推送主路径。  
3. 不处理 relay 真正未启动（`start.sh` 未运行）的场景——仍应显示「未连接」。

## 4. 设计方案

### 4.1 请求序号（generation token）

在 `ServerConfig.logic.vue` 为 `fetchRelayToTraeServiceStatus` 增加单调递增 `relayStatusFetchGeneration`：

- 每次调用递增 generation  
- 异步步骤返回前检查 generation 是否仍为当前值；过期则丢弃写入  
- 防止 stale 失败响应覆盖较新的成功结果

### 4.2 拆分「relay 可达」与「onlineService 运行态」

引入 `resolveRelayReachable(healthOk, statusResponse)`：

| health | status HTTP | 判定 |
|--------|-------------|------|
| 200    | —           | relay 在线 |
| 失败   | 200         | relay 在线（回退） |
| 失败   | 非 2xx      | relay 离线 |

仅当 **health 与 status 均不可达** 时才设置 `relayToTraeServiceOnline=false` 并展示「无法连接本机 relayToTrae…」。

`isRelayStatusPayloadLike` 保留用于是否 `applyRelayToTraeStatusPayload`，但 **不再作为 relay 可达性的唯一依据**——status HTTP 200 即视为可达，即使 body 为 `{running:false}` 或 Django 包装体。

### 4.3 stop 后的刷新策略（扩展 preserve 条件）

`stopRelayToTrae` 成功时：

- 保留本地 stopped 态写入（现有逻辑）  
- 末尾 `fetchRelayToTraeServiceStatus({ preserveRelayOnline: true })`（已实现）

**新增：** 下列路径在探测失败时同样 **preserve** relay 在线（不调用 `applyRelayToTraeOfflineState`）：

- 手动「刷新状态」  
- `props.task` 轮询触发的 `fetchRelayToTraeServiceStatus()`  
- 切换「直接启动」Tab  

条件（满足其一即可 preserve）：

```text
preserveRelayOnline === true
OR (relayToTraeServiceOnline && !relayToTraeOnlineServiceUp && !isRelayToTraeRunning)
```

即：已确认 relay 可达且 onlineService 处于 stopped 收敛态时，不因瞬态 health+status 双失败降级为「未连接」。

启动中（`isRelayToTraeStarting`）仍允许降级，避免掩盖真实 relay 宕机。

用户手动「刷新状态」仍走 generation 防竞态；offline 判定遵循 4.2，且受上述 preserve 保护。

### 4.4 stopped 态文案

`applyRelayToTraeStatusPayload` 在 `running=false && !orphan_port && !error` 时：

- 若当前 message 为「已停止 onlineServiceJS」或为空，设为「onlineServiceJS 已停止，relay 服务在线」  
- 不改动 `relayToTraeServiceOnline`

### 4.5 测试

**Playwright**（`TaskDetail.relay-to-trae-direct-start.playwright.test.js` 或新用例）：

1. mock：启动 → SSE running → stop → **第二次 health 超时、status 200 `{running:false}`** → 点击刷新  
2. 断言：`relayToTrae 服务：在线`，`onlineServiceJS：未启动`，无「无法连接本机 relayToTrae」

**并发回归**（可选 vitest/逻辑单测）：

- 模拟 A 慢失败 + B 快成功，验证最终 `relayToTraeServiceOnline=true`

## 5. Domain Concept Inventory（供 DDD）

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | 任务协作 / relay 运行时 | task-detail 直接启动面板 |
| Entity | RelayDirectStartSession | 面板内 relay + onlineService 复合状态 |
| Value Object | RelayReachability | health/status 探测结果 |
| Value Object | OnlineServiceRuntime | running / orphan_port / ui_url |
| Domain Event | relay_online_service_stopped | stop 成功后的收敛态 |

## 6. Value Stream Impact

影响现有 value streams：

1. **task-detail-runtime-relay**（直接修改：stop 后状态刷新 UX）  
2. **relay-status-push-timeout-go-relay**（关联：状态字段语义一致）  
3. **task-detail-relay-debug-agent-observability**（无行为变更，测试矩阵需补一条）

字段：无 DB 字段变更；前端状态机字段 `relayToTraeServiceOnline` / `relayToTraeOnlineServiceUp` 语义澄清。

测试：增强 `TaskDetail.relay-to-trae-direct-start.playwright.test.js`；无需新后端 pytest（纯前端状态机修复）。

## 7. 实施切片

| Slice | 内容 | 价值 |
|-------|------|------|
| 1 | generation token 防竞态 | 消除 stale 覆盖 |
| 2 | relay 可达性判定放宽 + stop preserve | 修复 stop→refresh 误判 |
| 3 | Playwright 回归 | 防止复发 |

## 8. 风险与回滚

- 风险：真正 offline 时延迟显示「未连接」（需 health+status 均失败才判定）——可接受，与现有回退策略一致。  
- 回滚： revert `ServerConfig.logic.vue` 单文件即可。
