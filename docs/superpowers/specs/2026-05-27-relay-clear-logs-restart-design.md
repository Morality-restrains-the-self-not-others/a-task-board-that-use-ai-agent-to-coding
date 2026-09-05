# Task Detail relayToTrae 清理日志后再次启动回填旧日志

日期：2026-05-27  
状态：已批准（随 /0-auto-flow 执行）

## 1. 问题陈述

在 task-detail `relayToTrae=true`「直接启动」面板：

1. 点击「启动」→ 产生启动日志  
2. 点击「清理日志」→ UI 清空  
3. 再次点击「启动」→ **旧日志重新出现**

「清理日志后刷新状态」已有 Playwright 回归且通过；**清理后再次启动**未覆盖。

## 2. 根因

1. **`startRelayToTrae` 重置 `relayToTraeLogSnapshot`**，使 dedup 前缀失效。  
2. 在 Django → go_relay `start/` 真正执行（`state.Logs = nil`）之前，**SSE 或 `fetchRelayToTraeServiceStatus` 仍返回 relay 内存中的全量旧日志**。  
3. `appendRelayToTraeLogs` 在 snapshot 为空时将全量日志写入 UI。  
4. **`registerTaskLocked` 将 `LogCursor` 置 0**，SSE 推送会重放历史日志。

## 3. 设计方案

### 3.1 前端：启动/清理后的日志抑制窗口

- 新增 `relayToTraeLogSuppressedAfterClear` 标志。  
- `clearRelayToTraeLogOutput`：保持 UI 清空，**不重置 snapshot**（保留刷新 dedup），并置抑制标志。  
- `startRelayToTrae` 开头：清空 UI、重置 snapshot、**置抑制标志**。  
- `appendRelayToTraeLogs`：抑制期间只更新 snapshot 基线、不写入 UI；收到空日志或后端 buffer 重置后解除抑制并正常追加。

### 3.2 后端：register 不重置 LogCursor 到 0

- 新登记：`LogCursor = len(state.Logs)`，避免重放已有 buffer。  
- 重登记：保留已有 `LogCursor`。

### 3.3 测试

Playwright：`清理日志后再次启动不应回填历史日志` — mock 启动 → 清日志 → 再启动 → 断言旧 log line 不可见。

## 4. Value Stream Impact

- **task-detail-relay-debug-agent-observability**：日志面板 UX  
- 无 DB 字段变更
