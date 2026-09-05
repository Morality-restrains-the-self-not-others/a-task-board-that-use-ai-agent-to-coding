# Value Stream: 容器令牌基础设施 Go 服务迁移

> Derived from design: `docs/superpowers/specs/2026-07-01-relay-precheck-go-service-design.md`

## Value Summary

runAll 单线程模式下直启 relay 不再死锁超时；容器令牌签发/校验/交换/凭证构建全部由独立 Go 服务承载，Django 清理全部容器运行时代码与表，职责边界清晰。

## Related Value Streams

**Modifications (table ownership changes — `saas-backend.*` → `task-credential-service.*`):**

| Stream | Change |
|--------|--------|
| `relay-precheck-internal-origin` | **修改** — `saas-backend.cloud_cloudserverconfig.*` → `task-credential-service.container_tokens.*`；删除 `saas-backend.django.internal_api_base` |
| `relay-token-audit-observability` | **修改** — `saas-backend.cloud_container_token_audit_event.*` → `task-credential-service.token_audit_events.*`；`saas-backend.cloud_cloudserverconfig.*` → `task-credential-service.container_tokens.*` |
| `relay-token-audit-full-chain-eventization` | **修改** — 同上，审计表 ownership 迁移 |
| `relay-register-token-exchange-guard` | **修改** — TEIP 保护逻辑从 Django → Go，token 字段 ownership 迁移 |
| `relay-status-push-timeout-go-relay` | **修改** — 状态推送端点从 Django 迁移到 Go |
| `task-detail-repo-clone-credentials-contract` | **修改** — endpoint 实现从 Django → Go，实现语言变更，URL 契约不变 |
| `task-detail-runtime-relay` | **修改** — relay 直启链路中 token/credential 部分迁移到 Go |

**New:**

| Stream | Description |
|--------|-------------|
| `task-credential-service-go` | Go 凭证服务：独立端口 (:8015)、独立 DB (tokens.sqlite3)、15 个容器回调端点 |

**Unaffected:**
- `task-detail-oauth-binding-guidance` — 前端引导文案不变
- `internal-api-timeout-governance` — 内网超时策略不变（调用目标从 Django→Django 变为 Django→Go）

## End-to-End Flow

```
[用户点击直启「启动」]
  → Django: relay-to-trae/token-init → Go POST /v1/token/init (签发 token → tokens.sqlite3)
  → Django: relay-to-trae/repo-credentials-precheck → Go POST .../repo-clone-credentials/ (读 saas.sqlite3 + 调 gitOauth)
  → Django: relay-to-trae/start → Go relay (:8797) 拉起 onlineServiceJS
  → Go relay: exchange-refresh → Go (:8015) (换 refresh_token)
  → onlineServiceJS: task-detail / credentials / heartbeat / ... → Go (:8015)
  → [用户看到容器启动成功]
```

## Value Increments

### Increment 1: Go 服务骨架 + token init（Thin Slice）
**Value to user:** Go 服务新进程存在，token 签发由 Go 负责，Django token-init 改为调用 Go。  
**Scope:**
- `taskCredentialService/` 目录 + Go 项目结构
- `db/container/tokens.sqlite3` 自动建表（DDL migration）
- `POST /v1/token/init` — token 签发端点
- Django `_issue_relay_access_token()` → HTTP 调 Go
- `conf/runAll.yaml` 新增 service 条目
- runAll 启动顺序：saas-backend → task-credential-service
**Depends on:** saas-backend（需 saas.sqlite3 就绪）

### Increment 2: 组 A 查询端点迁移（Core Value — 死锁消除）
**Value to user:** 直启预检不再死锁（8s → <1s），task-detail + layer-oauth 也统一由 Go 服务。  
**Scope:**
- Go: repo-clone-credentials / task-detail / layer-github-oauth-access-tokens 三个 handler
- Go: 只读打开 saas.sqlite3，查 Todo / TaskRepoIdentity / GitIdentity
- Go: gitOauth HTTP 客户端
- Django: `relay_to_trae_repo_credentials_precheck()` → 调 Go :8015
- taskAgentSupport: A1/A2/A3 转发目标切换到 Go
**Depends on:** Increment 1 (token init)

### Increment 3: 组 B token 端点迁移
**Value to user:** Token 生命周期全由 Go 管辖，Go relay 不再经 Django 换票。  
**Scope:**
- Go: exchange-refresh / refresh-access handler（写 tokens.sqlite3）
- go_relayToTrae: 换票目标从 Django :8001 → Go :8015
- onlineServiceJS: exchange-refresh 调用走 Go
- Django: 删除 `container_runtime_token_views.py`
**Depends on:** Increment 2

### Increment 4: 组 C+D 状态上报 + 杂项端点迁移（收尾）
**Value to user:** 容器运行时基础设施全面 Go 化，Django 零容器代码残留。  
**Scope:**
- Go: 全部剩余 9 个端点 handler
- Django: 删除 9 个 view 文件 + 3 个 Model + `instance_callback_urls.py` 清空
- Django: `cloud/domain/` 清理 6 个文件
- 测试文件迁移（6 个 pytest → Go test）
**Depends on:** Increment 3

### Increment 5: 前端错误提示精准化（Enhancement）
**Value to user:** 5xx 错误不再显示"请确认 Django 服务（如 8001）已运行"，改为精准提示。  
**Scope:** `ServerConfig.logic.vue` 第 1735-1737 行文案更新  
**Depends on:** Increment 2

### Increment 6: 回归保护（Essential Support）
**Value to user:** 全链路行为经完整测试回归。  
**Scope:** Go test suite + pytest 更新 + Playwright E2E 回归  
**Depends on:** Increment 1-5

## Affected Existing Streams (Reconciliation)

| Stream | Action |
|--------|--------|
| `relay-precheck-internal-origin` | 字段 `saas-backend.*` → `task-credential-service.*`；新增 Phase 1 steps |
| `relay-token-audit-observability` | 字段 ownership 迁移 |
| `relay-token-audit-full-chain-eventization` | 同上 |
| `relay-register-token-exchange-guard` | 同上 |
| `relay-status-push-timeout-go-relay` | 同上 |
| `task-detail-repo-clone-credentials-contract` | 端点实现迁移，URL 契约不变 |
| `task-detail-runtime-relay` | relay 直启链路迁移 |
