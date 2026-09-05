# Value Stream: taskAgentSupport internal_dispatch Request 修复

> 设计：`docs/superpowers/specs/2026-05-30-task-agent-support-internal-dispatch-request-fix-design.md`

## Related Value Streams

- **task-detail-runtime-relay**：修复 — relay 直启换票链路
- **relay-to-trae-startup-reliability**：补充 — Increment 2 未覆盖的 AssertionError 根因

## Value Summary

relayToTrae 经 `:8011` 调用 `exchange-refresh` 等 inbound API 时，Django internal 转发不再因 Request 类型错误返回裸 `500 internal error`。

## End-to-End Flow

[直启启动] → go_relay token-exchange → taskAgentSupport :8011 → Django internal_dispatch → exchange-refresh 视图 → [200/401 可诊断]

## Value Increments

### Increment 1: HttpRequest 转发修复（Thin Slice）
**Value：** 消除 500 AssertionError，换票可进入业务逻辑  
**Scope：** `internal_dispatch.py` + `test_task_agent_support_internal_dispatch.py`  
**Depends on：** 无

### Increment 2: OperationalError → 503（同 PR）
**Value：** SQLite busy 可重试，日志含 error_code  
**Scope：** `internal_dispatch.py` 异常分支  
**Depends on：** Increment 1
