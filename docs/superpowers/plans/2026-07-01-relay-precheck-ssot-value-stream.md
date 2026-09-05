# Value Stream: relay 直启预检 Token SSOT 消除双写

> Derived from design: `docs/superpowers/specs/2026-07-01-relay-precheck-ssot-in-process-design.md`

## Value Summary

relayToTrae 全链路（token-init / precheck / start / register）不再由 Django 自生成 container access token + SQLite hack 同步到 Go，改为直接调用 Go taskCredentialService `/v1/token/init`，使 Go 成为 Token 唯一数据源（SSOT）。

## Related Value Streams

- **`relay-precheck-local-origin`** (`2026-05-27-relay-precheck-local-origin-value-stream.md`): extension — 本流是该流的第二步。第一步将 precheck origin 从公网网关改为内网地址；本步消除 Django→Go DB 双写 hack，完成架构归一化。
- **`task-detail-repo-clone-credentials-decoupling`** (`2026-05-25-*-decoupling-value-stream.md`): dependency — 凭证构建已迁至 Go，本流依赖其 Go endpoint 正常工作。
- **`relay-token-audit-observability`** (`conf/value-stream.yaml`): alignment — 审计表 `task-credential-service.token_audit_events` 和 `task-credential-service.container_tokens` 已是 Go 域字段。

## End-to-End Flow

[用户点击直启「启动」] → [token-init: Django → Go /v1/token/init 签发 token] → [precheck: Django → Go token-init 复用 token → Go repo-clone-credentials 校验凭证] → [start: Django → Go token-init 复用 token → 注入 runtime_env → 发给 go_relayToTrae] → [容器持有 Go 签发的 token 回调换凭证]

## Value Increments

### Increment 1: token-init 改调 Go (Thin Slice)
**Value to user:** token-init 返回的 token 由 Go 统一签发，不再依赖 Django CloudServerConfig + SQLite hack。  
**Scope:** `relay_to_trae_token_init()` 不再调用 `_build_runtime_env_for_relay()` → `_replace_access_token_placeholder()`，改为 `POST /v1/token/init` 获取 token；新增 `_task_credential_service_url()` + `_credential_service_post()` 辅助函数。  
**Depends on:** Go taskCredentialService `/v1/token/init` 已就绪。

### Increment 2: precheck + start + register 改调 Go (Core Value)
**Value to user:** 预检和启动全链路 token 由 Go SSOT 签发，消除 SQLite 跨语言直写 hack。预检不再因 Django→Go DB 同步失败而报 502。  
**Scope:** `relay_to_trae_repo_credentials_precheck()` 两步调用（token-init → repo-clone-credentials）；`relay_to_trae_start()` 调 token-init 替代 `_build_runtime_env_for_relay()`；`register_or_reuse` 调 token-init 替代 `_issue_relay_access_token()`。  
**Depends on:** Increment 1。

### Increment 3: 删除双写 hack 代码 (Cleanup)
**Value to user:** 代码库清理，消除跨语言耦合。  
**Scope:** 删除 `_sync_token_to_credential_service()`、`_sync_credential_service_token()`、`_issue_relay_access_token()`、`_resolve_relay_precheck_task_api_origin()`、`_build_runtime_env_for_relay()`；从 `mock_run_container.py` 移除 `import sqlite3, os, uuid`（仅服务于 sync hack）。  
**Depends on:** Increment 2。

### Increment 4: 测试更新与回归保护 (Essential Support)
**Value to user:** 行为长期稳定。  
**Scope:** 更新 `test_relay_to_trae_proxy.py` — mock `_credential_service_post` 替代 mock `CloudServerConfig.objects` / `_issue_relay_access_token`；新增 Go 调用失败场景测试；删除已废弃函数的测试。  
**Depends on:** Increment 1–3。

## Fields Impact

| Field | Change |
|-------|--------|
| `task-credential-service.container_tokens.container_access_token` | **写入方归一化** — 仅 Go `IssueToken` 写入，删除 Django SQLite hack 写入路径 |
| `task-credential-service.container_tokens.container_access_token_expires_at` | 同上 |
| `task-credential-service.container_tokens.container_refresh_token` | 同上 |
| `saas-backend.cloud_cloudserverconfig.container_access_token` | **不再写入** — relayToTrae 路径停止写入此字段（mockStart 暂保留） |

## Affected Existing Streams

- `relay-precheck-local-origin` — 本流完成其第二步（原标注为"非目标"的进程内调用 → 现为 Go SSOT 调用）
