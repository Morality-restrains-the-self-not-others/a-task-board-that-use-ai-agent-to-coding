# Value Stream: relay status-push timeout（Go relay）

> Derived from design: `docs/superpowers/specs/2026-05-22-relay-status-push-timeout-go-relay-design.md`

## Value Summary

任务详情页用户在「直接启动/刷新状态」后可以持续看到稳定的 relay 在线状态，不再因代理链路超时被误判为停止。

## End-to-End Flow

用户触发启动/刷新状态 → `go_relayToTrae` 采集状态快照并发起 status-push → Django `relay-to-trae/status-push` 收敛并返回 ACK → relay 清零 missed-ack → 前端持续收到正确状态。

## Value Stages

- Trigger：用户在任务详情页点击「直接启动」或「刷新状态」
- Stage 1（核心）：`go_relayToTrae` 周期执行 status-push
- Stage 2（核心）：请求经 no-proxy 直连任务 API，避免代理超时
- Stage 3（核心）：Django 返回 ACK，relay 保持注册状态
- Delivery：前端持续显示 onlineServiceJS 已启动且状态一致

## Wait / Dependency Points

- status-push 对 Django ACK 的等待（HTTP 头与响应体）
- 若依赖代理链路，可能导致等待超时并触发 missed-ack 累加

## Value Increments

### Increment 1: no-proxy status-push thin slice（Thin Slice）
**Value to user:** 状态刷新不再频繁超时，任务不会被误注销。  
**Scope:** 仅修复 `push.go` status-push 的 HTTP client 出站代理策略为 no-proxy。  
**Depends on:** nothing

### Increment 2: token-exchange 出站一致性
**Value to user:** 启动阶段换票链路与状态上报链路网络行为一致，减少隐性链路差异故障。  
**Scope:** `token.go` 的 exchange-refresh/refresh-access 统一 no-proxy transport。  
**Depends on:** Increment 1

### Increment 3: 回归测试与可维护性增强
**Value to user:** 后续版本不易回归到代理超时问题，稳定性可持续。  
**Scope:** 新增/更新 Go 单测，覆盖 no-proxy client 行为与关键请求链路。  
**Depends on:** Increment 2

## Stage Classification

- Core value
  - status-push no-proxy 直连
  - ACK 成功后的注册保持
- Essential support
  - token-exchange no-proxy 一致性
  - 单测保障
- Enhancement
  - 更细粒度观测指标（本次不做）
- Future
  - 动态可配置 proxy 策略与多环境开关（本次不做）

## Affected Existing Value Streams

- `task-detail-runtime-relay` / step `relay-status-convergence`
- `relay-token-audit-observability` / 与 status-push 失败率相关

## Field Impact (for YAML mapping)

- 直接字段变化：无新增数据库字段
- 关联观测字段（语义受影响但结构不变）：
  - `saas-backend.cloud_container_token_audit_event.event_type`
  - `saas-backend.cloud_container_token_audit_event.seq`
  - `saas-backend.cloud_container_token_audit_event.trace_id`
  - `saas-backend.cloud_container_token_audit_event.error_code`

## Proposed YAML Step Mapping

建议在 `task-detail-runtime-relay` 下新增一个 active step：

- step name: `relay-status-push-no-proxy-timeout-fix`
- test file: `tests/test_relay_to_trae_proxy.py`（先复用，后续可拆专用 Go/集成测试映射）
- fields:
  - `saas-backend.cloud_container_token_audit_event.event_type`
  - `saas-backend.cloud_container_token_audit_event.seq`
  - `saas-backend.cloud_container_token_audit_event.trace_id`
  - `saas-backend.cloud_container_token_audit_event.error_code`
