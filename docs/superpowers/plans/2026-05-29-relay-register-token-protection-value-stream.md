# Value Stream: relay register 换票窗口保护

> 设计：`docs/superpowers/specs/2026-05-29-relay-register-token-protection-design.md`

## Related Value Streams

- **task-detail-runtime-relay**（`value-stream.yaml`）：**修改** — 直启链路在 register 阶段不得破坏 token 态；`relay-stop-refresh-status-ui` 回归扩展
- **relay-token-audit-full-chain-eventization**：**扩展** — 新增 `token_bootstrap_blocked` / `relay_register_reused_state` 审计
- **relay-token-exchange-no-proxy-consistency**（`relay-status-push-timeout-go-relay`）：**依赖** — 换票两步仍须在同一 DB 态下完成
- **2026-05-29-relay-to-trae-startup-reliability**：**互补** — 该流修路由/503/reachability；本流修 register 与 bootstrap 竞态
- **startup-storm-mitigation**（spec）：**背景** — 启动风暴使换票窗口更窄，register 竞态更易触发

## Value Summary

开发者在任务详情 `?relayToTrae=true` 点击直启后，换票（exchange-refresh → refresh-access）不被 register 打断，任务详情日志不再出现 `无效的 access_token`。

## End-to-End Flow

```text
[用户点击直启]
  → token-init（bootstrap access）
  → precheck
  → start 202 → go_relay 同步换票（exchange-refresh → refresh-access）
  → onlineServiceJS 启动（TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE）
  → register-reachability / status-push
  → [任务详情 SSE 显示正常日志，无无效 token]
```

**当前断点：** start 202 后 / health 刷新时 `register(issueTokenOnServer)` 进入 bootstrap，在 TEIP 窗口清空 refresh。

**TEIP（Token Exchange In Progress）：** `refresh_token` 非空且 `access_token` 为空且 `expires_at` 为空。

## Value Increments

### Increment 1: Django TEIP 硬保护（Thin Slice）

**Value to user：** 直启不再因 register 打断换票而出现 `无效的 access_token`（即使前端仍发 register）。

**Scope：**

- `_is_token_exchange_in_progress(cfg)` 判定
- `_replace_access_token_placeholder`：TEIP 时禁止清空 refresh / 重签 access
- `_issue_relay_access_token` / `relay_to_trae_register`：TEIP 时 register-only（不 server-issue）或 409
- 审计：`token_bootstrap_blocked`
- 单测：`tests/test_relay_register_token_exchange_guard.py`（或 `test_relay_to_trae_proxy.py` 新用例）
- 回归：`test_exchange_refresh_then_refresh_access_updates_db`

**Depends on：** 无

**Fields：**

- `saas-backend.cloud_cloudserverconfig.container_refresh_token`
- `saas-backend.cloud_cloudserverconfig.container_access_token`
- `saas-backend.cloud_container_token_audit_event.event_type`
- `saas-backend.cloud_container_token_audit_event.error_code`

### Increment 2: 前端 register 语义收敛

**Value to user：** 减少无意义 register 请求；启动路径更清晰。

**Scope：**

- 删除 `startRelayToTrae` 末尾冗余 `registerRelayToTraeTask({ issueTokenOnServer: true })`
- `fetchRelayToTraeServiceStatus`：register 使用 `register_only` / 不再 `issueTokenOnServer`
- Playwright：`TaskDetail.relay-to-trae-direct-start` 断言 start 后无二次 bootstrap 审计

**Depends on：** Increment 1（后端已安全，前端为减载）

**Fields：** 无新增 DB 字段

### Increment 3: 关联加固（Enhancement）

**Value to user：** 换票失败时快速失败，不出现长时间误报在线。

**Scope：**

- go_relay：`exchange-refresh` 成功后禁止 fallback 到 original token
- Django：`exchange_refresh` / `refresh_access` 后失效 status-push cfg 缓存
- 审计：`refresh_access` 成功事件可串联 trace

**Depends on：** Increment 1

**Fields：**

- `saas-backend.cloud_container_token_audit_event.new_access_token_sha256`
- `go-relay.runtime.access_token`（日志侧，非 DB）

### Future（本价值流不交付）

- TEIP 超时自动清理（go_relay 崩溃后 refresh 残留）
- go_relay 实现 `skip_token_exchange` 与 Django 对齐
- 合并 token-init + start 为单次 internal batch API

## 验收标准（按增量）

| 增量 | 验收 |
|------|------|
| 1 | `exchange_refresh` 后立刻 `relay_register(placeholder)` → DB refresh 不变；`refresh_access` 仍 200 |
| 1 | 审计出现 `token_bootstrap_blocked` 或 `relay_register_reused_state`，无 `无效的 refresh_token` |
| 2 | Playwright 直启：日志不含 `无效的 access_token` |
| 3 | go_relay 换票失败时进程不携带作废 access 启动子进程 |
