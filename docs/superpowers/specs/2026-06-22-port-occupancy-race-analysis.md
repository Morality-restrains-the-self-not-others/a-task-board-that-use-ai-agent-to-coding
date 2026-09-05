# 为什么进程停止运行但端口仍然被占用

**Date**: 2026-06-22
**Status**: 根因分析

## 现象

`task-events-email-sent-1-send-email` 消费者进程已停止，邮件无法发送。runAll 日志显示「服务端口 18022 已被监听，跳过启动 (PID=[1585794])」—— 端口检查时发现旧 PID，跳过启动。但随后旧进程退出，端口释放，runAll 不再重试。

## 根因分析

### 三类机制

三个独立但可叠加的机制导致了「进程已死，端口仍被占用」的现象：

#### 1. TOCTOU 竞态窗口 (根本原因)

runAll 的启停流程存在检查-执行时间差：

```
runAll:  stop旧进程 (发送 SIGTERM)
         │
         ├─ 旧进程收到信号，开始清理（socket 仍 LISTEN）
         │
runAll:  listenerPIDs("18022") → 找到 PID 1585794
         → 「端口已被监听，跳过启动」  ← 致命决策
         │
         ├─ 旧进程清理完毕, close(socket), 退出
         │
         端口 18022 空闲... 但 runAll 已不会再尝试
```

关键代码在 `runAll/src/runner.go:478-522`：
```go
// 端口占用 → 直接 return nil，不再启动
if len(pids) > 0 {
    log.Printf("port %s is already being listened to (PID=%v), skipping start", port, pids)
    // ...
    return nil  // ← 一次判断，永久跳过
}
```

`listenerPIDs()` 使用 `lsof -t -iTCP:18022 -sTCP:LISTEN` 检查，只判断 `LISTEN` 状态。旧进程清理期间 socket 仍在 LISTEN → 检查通过 → 跳过启动 → 旧进程退出 → 端口空闲但无人启动。

#### 2. TCP TIME_WAIT（辅助因子）

TCP 连接关闭后，端口进入 TIME_WAIT 状态，持续 2*MSL（典型 60-120 秒）。期间 `bind()` 同端口返回 `EADDRINUSE`。

但在此场景中，`lsof -sTCP:LISTEN` **不匹配** TIME_WAIT 状态的 socket（因为 TIME_WAIT 不是 LISTEN）。所以 TIME_WAIT 不是主因——问题出在旧进程退出前的 LISTEN 窗口。

当进程设置 `SO_REUSEADDR` 后，可跳过 TIME_WAIT 限制直接 bind。taskEvents 消费者**未设置** SO_REUSEADDR。

#### 3. 僵尸进程（不适用于本次）

僵尸进程 (defunct/Z) 保留在进程表中但不持有 socket。`lsof -t -iTCP` 不会返回僵尸 PID。当前环境存在僵尸进程（旧 taskAuth PID 1573896），但不影响端口检测。

### 完整时序图

```
10:55:15  email consumer 处理最后一个 EMAIL_SENT 事件
10:55:41  runAll 重启 email consumer
10:55:41  → listenerPIDs("18022") → [1585794] → 「端口已被监听，跳过启动」
10:55:41  → 旧进程仍在 LISTEN（清理中）
~10:55:5x 旧进程退出，端口 18022 释放
------ 此后 8 小时无人消费 EMAIL_SENT ------
19:00:xx 手动重启 consumer → 立即消费积压事件
```

## 现有防护机制及其盲区

`runall-startup-race-eaddrinuse-fix` 已修复了三类并发场景：

| 机制 | 覆盖场景 | 本次是否生效 |
|------|---------|------------|
| CAS 启动守卫 | DAG + API 并发启动同一服务 | ❌ 不适用（非并发启动） |
| 孤儿清理 (15s 等待) | 旧 runAll 进程残留的托管服务 | ❌ 不适用（消费者由当前 runAll 管理） |
| SO_REUSEADDR | TIME_WAIT 假占用 | ❌ taskEvents 未设置 |
| lsof 错误 fail-fast | 端口检测异常 | ❌ 检测本身成功（返回了 PID） |

**盲区**：`startAndCheck` 中的端口检查是一次性的。如果端口在检查时被占用但随后释放，runAll 不会重试。缺少：

1. **等待旧进程释放端口**：发送 stop 信号后，应轮询等待端口变为空闲（带超时）
2. **端口释放后的重试**：如果检测到旧 PID 且随后端口释放，应启动新进程
3. **taskEvents Go 服务的 SO_REUSEADDR**：Go net.Listen 默认不设 SO_REUSEADDR，需显式配置

## Domain Concept Inventory

- **Bounded Context**: 平台与本地开发 (platform-dev)
- **Key Entities**: `ManagedService`（生命周期状态机）、`PortProbe`（端口检测）
- **Candidate Aggregates**: `ServiceLifecycle`（start → healthy → stopping → stopped）

## 修复方向

### 短期（防御性）

1. **taskEvents 消费者添加 SO_REUSEADDR**：修改 `eventbin` 的 HTTP listener 创建，设置 `net.ListenConfig.Control` 中的 `SO_REUSEADDR`
2. **健康检查持续监控**：runAll 的定期健康检查如果发现端口空闲 + 状态 healthy，应触发自动重启

### 长期（根治）

3. **startAndCheck 增加端口释放等待循环**：
   ```
   stop旧进程 → 轮询等待端口空闲（最多 30s）
   → 端口空闲 → 启动新进程
   → 超时 → 强制清理 + 启动
   ```
4. **startAndCheck 的「跳过启动」不应是终态**：检测到端口占用后，验证占用进程是否为已知旧 PID。如果是，等待其退出后启动；如果不是（外部进程），才跳过。
