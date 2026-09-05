# DDD Model: relayToTrae 停止后刷新状态（前端限界上下文）

> 输入: value-stream + NFR 澄清。本增量为 **前端 UI 状态机**，无后端 domain 代码变更。

## 限界上下文

**TaskDetail Relay Direct Start** — task-detail 面板内 relay 侧车与 onlineServiceJS 复合运行时状态。

## 值对象

### RelayReachability
- `serviceOnline: boolean` — relay 侧车 HTTP 可达（health 200 或 status 200 回退）
- `probedAt: number` — 探测序号（generation）

### OnlineServiceRuntime
- `running: boolean`
- `onlineServiceUp: boolean`
- `orphanPort: boolean`
- `uiUrl: string`
- `error: string`

## 领域服务（前端 composable 逻辑）

### RelayStatusRefreshService
- `refresh(options?: { preserveRelayOnline?: boolean })` — 带 generation 的状态刷新
- `applyStatusPayload(data)` — 合并 onlineService 字段，不降级 relay 可达性（当 preserve）

## 领域事件（UI 层）

- `RelayOnlineServiceStopped` — stop 成功，本地写入 stopped 态
- `RelayStatusRefreshed` — refresh 完成

## 仓储

无（状态为组件 ref，数据来自 Django proxy API）。
