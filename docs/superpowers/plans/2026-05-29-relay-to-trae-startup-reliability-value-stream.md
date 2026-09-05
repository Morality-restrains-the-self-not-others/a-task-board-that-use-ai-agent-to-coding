# Value Stream: relayToTrae 启动可靠性

> 设计：`docs/superpowers/specs/2026-05-29-relay-to-trae-startup-reliability-design.md`

## Related Value Streams

- **task-detail-runtime-relay**：扩展 — 修复 status-push 收敛
- **relay-status-push-timeout-go-relay**：修改 — 路由对齐 taskAgentSupport
- **startup-storm-mitigation**：依赖 — Phase 1 已落地，本流补 503 重试

## Value Summary

开发者通过 relay 直启任务容器时，能稳定完成 reachability 注册并在任务详情看到准确启动日志与 relay 状态。

## End-to-End Flow

[点击启动] → go_relay 换票/拉起 onlineServiceJS → listen 8765 → register-reachability → status-push SSE → [层图可拉取]

## Value Increments

### Increment 1: status-push 路由修复（Thin Slice）
**Value：** go_relay 状态推送不再 404，SSE 与刷新状态一致  
**Scope：** taskAgentSupport `parseCloudInboundPath` + 单测  
**Depends on：** 无

### Increment 2: internal_dispatch 可恢复错误
**Value：** SQLite busy 返回 503 + error_code，日志可诊断  
**Scope：** internal_dispatch + pytest  
**Depends on：** Increment 1

### Increment 3: reachability 重试 + 日志面板回归
**Value：** 瞬时 busy 下容器可完成注册；清理日志后刷新不回填  
**Scope：** reachability.mjs 重试 + relayToTraeUtils 测试（已完成）  
**Depends on：** Increment 2
