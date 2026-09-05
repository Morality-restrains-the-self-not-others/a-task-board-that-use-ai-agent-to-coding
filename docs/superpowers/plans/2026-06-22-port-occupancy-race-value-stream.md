# Value Stream: Fix Port Occupancy Race in runAll startAndCheck

> Derived from design: `docs/superpowers/specs/2026-06-22-port-occupancy-race-analysis.md`

## Value Summary

runAll 服务启停不再因 TOCTOU 竞态导致消费者永久丢失——邮件/通知等异步消费者不会因端口检查时机误差而停止运行。

## Related Value Streams

- **runall-startup-race-eaddrinuse-fix** (existing, `value-stream.yaml`): extension — builds on Increment 1-3 (CAS guard, orphan cleanup, SO_REUSEADDR for gitOauth). This stream adds Increment 4 (port-release wait loop in startAndCheck) and Increment 5 (SO_REUSEADDR for taskEvents Go consumers).

## End-to-End Flow

```
runAll 停止旧进程 (SIGTERM)
  → 轮询等待端口释放 (每 500ms, 最多 30s)  ← NEW
  → 端口空闲或超时
  → 启动新进程 (带 SO_REUSEADDR)           ← NEW for taskEvents
  → 健康检查通过
  → 服务正常运行
```

## Value Increments

### Increment 1: Port-Release Wait Loop in startAndCheck (Core)
**Value to user:** runAll 重启服务时不再因旧进程清理延迟而跳过启动
**Scope:** `runAll/src/runner.go` — 在 `listenerPIDs` 返回非空时，轮询等待端口变为空闲（500ms 间隔，最多 30s），超时后才跳过。替换当前一次性检查+跳过的逻辑。
**Depends on:** runall-startup-race-eaddrinuse-fix (existing)
**Test file:** `runAll/src/runner_test.go` (existing, extend `TestStartAndCheckPortAlreadyListening`)

### Increment 2: SO_REUSEADDR for taskEvents Consumers (Essential Support)
**Value to user:** taskEvents 消费者在 TIME_WAIT 窗口后可立即重新绑定端口
**Scope:** 修改 `taskEvents/eventbin` HTTP listener 创建逻辑，设置 `SO_REUSEADDR`。覆盖全部 17 个 taskEvents 消费者（email_sent, invitation_created, user_activated, user_created/*, company_created/*, etc.）
**Depends on:** Increment 1
**Test file:** `taskEvents/eventbin/` (现有启动测试)
