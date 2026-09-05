# Value Stream: relayToTrae 停止后刷新状态

> Derived from design: `docs/superpowers/specs/2026-05-27-relay-stop-refresh-status-design.md`

## Value Summary

开发与排障人员在 task-detail「直接启动」面板停止 onlineServiceJS 后刷新状态，能稳定看到「relay 在线 + onlineService 未启动」，不再误判 relay 侧车离线。

## End-to-End Flow

用户 stop onlineServiceJS → 点击「刷新状态」→ 前端 health/status 探测（带序号防竞态）→ 合并 stopped 态 → 面板展示 relay 在线、onlineService 未启动。

## Value Increments

### Increment 1: 防竞态 + relay 可达性判定（Thin Slice）
**Value to user:** stop 后刷新不再偶发显示「relay 未连接」。  
**Scope:** `fetchRelayToTraeServiceStatus` generation token；status HTTP 200 即视为 relay 在线；stop 后 preserve relay online。  
**Depends on:** nothing

### Increment 2: stopped 态文案与 Playwright 回归
**Value to user:** 状态文案可理解；自动化覆盖 stop→refresh。  
**Scope:** `applyRelayToTraeStatusPayload` 文案；Playwright 用例。  
**Depends on:** Increment 1
