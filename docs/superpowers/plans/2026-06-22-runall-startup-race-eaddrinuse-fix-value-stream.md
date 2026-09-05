# Value Stream: runAll 启动竞态 EADDRINUSE 修复

> Derived from design: `.claude/plans/01-brainstorming-设计文档.md`

## Value Summary

开发/运维用户在启动 runAll 时，DAG 引导与 `/api/start-all` 不再因并发竞态而重复启动同一服务导致 `Address already in use` 错误，所有服务一次启动成功。

## Related Value Streams

- **runall-log-copy-gitoauth-port-conflict-recovery** (2026-05-26): modification — 该流修复了端口冲突的**可恢复性**（检测外部占用→清理→重试），本流修复端口冲突的**根因**（runAll 自身并发启动同一服务）。两者互补：前者处理外部进程冲突，后者防止 runAll 自身制造冲突。
- **runall-stability-first** (2026-05-25): extension — 在 Increment M1/M2 的 preflight 闸门基础上，给 `startAndCheck` 增加 CAS 并发守卫。

## End-to-End Flow

[runAll 启动] → [DAG 引导逐层启动服务] → [startAndCheck 原子守卫] → [单次 launch 执行] → [端口绑定成功] → [健康检查通过] → [用户看到 healthy]

## Value Stages

- **Trigger:** runAll 进程启动（自动触发 DAG 引导）或用户在 UI 点击「全部启动」。
- **Stage 1 (Essential support):** `startAndCheck` 入口增加 CAS 守卫，确保同一服务只有一个 goroutine 进入 launch 阶段。
- **Stage 2 (Essential support):** `killPreviousRunAllProcess` 在杀旧 runAll 前等待其托管服务端口释放。
- **Stage 3 (Core value):** gitOauth 的 `NoReverseDNSWSGIServer` 正确设置 `SO_REUSEADDR`，避免 TIME_WAIT 导致的虚假 EADDRINUSE。
- **Stage 4 (Essential support):** `listenerPIDs` 错误不再被静默吞噬，检测失败时中止启动并报告明确错误。
- **Delivery point:** 用户启动 runAll 后所有服务稳定进入 healthy 状态，无端口冲突错误。

## Value Increments

### Increment 1: startAndCheck CAS 并发守卫 (Thin Slice)
**Value to user:** DAG 引导与 API start-all 不再竞态重复启动同一服务。
**Scope:** `startAndCheck` 入口调用 `transitionServiceToStarting` CAS；新增 `waitForServiceStart` 等待另一个 goroutine 完成启动。
**Depends on:** nothing

### Increment 2: killPreviousRunAllProcess 等待托管服务释放
**Value to user:** 新旧 runAll 实例替换时不再出现孤儿进程占用端口。
**Scope:** 杀旧 runAll 前检测并等待其托管服务的端口释放（最长 15 秒超时）。
**Depends on:** Increment 1

### Increment 3: gitOauth SO_REUSEADDR 修复
**Value to user:** gitOauth 重启时不再因 TIME_WAIT 端口残留而失败。
**Scope:** `NoReverseDNSWSGIServer.server_bind` 增加 `SO_REUSEADDR`；排查其他 Python WSGI 服务。
**Depends on:** Increment 1

### Increment 4: listenerPIDs 错误不静默
**Value to user:** lsof 检测失败时获得明确错误信息，而非静默尝试启动导致端口冲突。
**Scope:** `listenerPIDsForPort` 错误时 `startAndCheck` 标记服务 Failed 并中止。
**Depends on:** Increment 1
