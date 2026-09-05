# 设计文档：relay 直启预检 — taskCredentialService Token SSOT 与消除双写

**日期：** 2026-07-01
**状态：** 设计中
**替代：** `2026-07-01-relay-precheck-selfcall-deadlock-design.md`（根因分析错误）
**用户指令：** "token 仅有 taskCredentialService 管理，不要双写"

---

## 0. 旧设计文档的错误

`2026-07-01-relay-precheck-selfcall-deadlock-design.md` 声称预检对 `127.0.0.1:8001`（Django）发起 HTTP 自调用导致死锁。**这个根因分析是错误的。**

### 源码事实

`_resolve_relay_precheck_task_api_origin()` 实际返回 **Go taskCredentialService (`:8015`)**：

```python
# relay_to_trae_proxy.py:505-507
def _resolve_relay_precheck_task_api_origin() -> str:
    """预检走 Go taskCredentialService (:8015)，避免单线程 Django 自调用死锁。"""
    return str(settings_manager.get_task_credential_service_url() or "").strip().rstrip("/")
    # → "http://127.0.0.1:8015"
```

```yaml
# conf/core/django/config.yaml:9
taskCredentialServiceBase: http://127.0.0.1:8015
```

不存在 Django 自调用死锁。错误的"进程内直接调用"方案也不可行（Go 是独立进程，无法进程内调用）。

---

## 1. 真正的架构问题：Token 双写 (Dual Write)

### 1.1 token_init — Django 生成 token → SQLite hack 同步到 Go

```
POST /relay-to-trae/token-init/
→ relay_to_trae_token_init()
  → _build_runtime_env_for_relay()
    → build_relay_to_trae_runtime_env()
      → _replace_access_token_placeholder()
        → CloudServerConfig.objects.filter(company_id=..., task_id=...).first()
        → cfg.container_access_token = generate_opaque_token()  ← ❌ Django 生成 token
        → cfg.save()                                            ← ❌ Django 持久化
        → _sync_token_to_credential_service()                   ← ❌ SQLite hack 直写 Go 的 DB
           → sqlite3.connect("db/container/tokens.sqlite3")
           → INSERT INTO container_tokens (...)
```

### 1.2 precheck — Django 查自己的 DB → SQLite hack → 再调 Go

```
POST /relay-to-trae/repo-credentials-precheck/
→ relay_to_trae_repo_credentials_precheck()
  → _issue_relay_access_token()
    → CloudServerConfig.objects.filter(...)       ← ❌ Django 查自己的 DB
    → _sync_credential_service_token()            ← ❌ SQLite hack 同步到 Go
  → requests.post(:8015/.../repo-clone-credentials/)  ← Go 验证自己 DB 中的 token
```

### 1.3 start — 同样的双写路径

```
POST /relay-to-trae/start/
→ relay_to_trae_start()
  → _build_runtime_env_for_relay()               ← ❌ 同 1.1 的路径
```

### 1.4 register-or-reuse — 同样的双写路径

```
POST /relay-to-trae/register-or-reuse/
→ _issue_relay_access_token()                    ← ❌ 同 1.2 的路径
```

### 1.5 双写的罪魁祸首

| 函数 | 位置 | 问题 |
|------|------|------|
| `_replace_access_token_placeholder()` | `mock_run_container.py:390-496` | Django 生成 token + 写 `CloudServerConfig` + 调用 `_sync_token_to_credential_service` |
| `_sync_token_to_credential_service()` | `mock_run_container.py:286-387` | 用 `sqlite3.connect()` 直写 Go 的 `db/container/tokens.sqlite3` |
| `_sync_credential_service_token()` | `relay_to_trae_proxy.py:209-231` | 包装函数，调用 `_sync_token_to_credential_service` |
| `_issue_relay_access_token()` | `relay_to_trae_proxy.py:234-292` | 查 `CloudServerConfig` + 调用 `_sync_credential_service_token` |
| `build_relay_to_trae_runtime_env()` | `mock_run_container.py:674-734` | 调用 `_replace_access_token_placeholder` |

---

## 2. 目标架构：taskCredentialService 作为 Token SSOT

### 2.1 核心原则

- **taskCredentialService 是 container access token 的唯一写入方和持久化存储方**
- **Django 不生成、不持久化、不同步 container access token**
- **Django 需要 token 时，通过 HTTP 调用 Go `/v1/token/init`**

Go 已有完整的 token 能力：

```go
// POST /v1/token/init {tenant_id, workspace_id, task_id}
// → {status: "ok", access_token, refresh_token, expires_at}
token, err := h.svc.Token.IssueToken(scope, nil)
```

`IssueToken` 内部逻辑：已有有效 token → 复用；无 → 生成新 token + 写入 `container_tokens` 表 + 审计事件。

### 2.2 新流程

```
Django (:8001)                          taskCredentialService (:8015)
                                                 │
  ┌─── token_init ───────────────────────────────┤
  │                                              │
  ├─ POST /v1/token/init ──────────────────────►│ IssueToken(scope)
  │   {tenant_id, workspace_id, task_id}        │ → SSOT: container_tokens
  │◄── {access_token, expires_at} ──────────────┤
  │                                              │
  │  返回 {status:"ok", token_initialized:true}  │
  │  (Django 不存 token)                        │
  │                                              │
  ├─── precheck ─────────────────────────────────┤
  │                                              │
  ├─ POST /v1/token/init ──────────────────────►│ IssueToken → 复用同一条
  │◄── {access_token} ──────────────────────────┤
  │                                              │
  ├─ POST /.../repo-clone-credentials/ ─────────►│ ValidateToken ✅
  │   {access_token}                            │ BuildRepoCloneCredentials
  │◄── {repo_clone_credentials, ...} ───────────┤
  │                                              │
  │  返回预检结果 (200/409/502)                  │
  │                                              │
  ├─── start ────────────────────────────────────┤
  │                                              │
  ├─ POST /v1/token/init ──────────────────────►│ IssueToken → 复用同一条
  │◄── {access_token} ──────────────────────────┤
  │                                              │
  │  注入 runtime_env → 发给 go_relayToTrae      │
  └──────────────────────────────────────────────┘
```

**关键：** token_init、precheck、start 各自调用 `/v1/token/init`，Go 的 `IssueToken` 通过 `FindByTaskID` 自动复用在有效期内的 token，保证同一次 relayToTrae 会话使用同一个 token。

---

## 3. Django 侧变更

### 3.1 `relay_to_trae_token_init` — 改为调用 Go

**文件：** `relay_to_trae_proxy.py` 第 397 行

```python
# 旧：_build_runtime_env_for_relay() → _replace_access_token_placeholder() → Django 生成 token + SQLite hack
# 新：调用 taskCredentialService /v1/token/init

def relay_to_trae_token_init(request, tenant_id=None, workspace_id=None, task_id=None):
    ...
    # 调用 Go 签发 token（SSOT）
    credential_svc_url = _task_credential_service_url()
    token_resp = _credential_service_post(
        f"{credential_svc_url}/v1/token/init",
        json={"tenant_id": tenant, "workspace_id": wid, "task_id": tid},
        timeout=5,
    )
    if not token_resp.ok:
        return _relay_error_response(token_resp)
    token_data = token_resp.json()
    access_token = str(token_data.get("access_token") or "")
    if not access_token:
        return Response({"status": "error", "message": "Go 返回空 token"}, status=502)
    # 不存 token，只返回 env_preview
    return Response({
        "status": "ok",
        "task_id": tid,
        "token_initialized": True,
        "env_preview": {
            "TASK_API_ENDPOINT_ORIGIN": settings_manager.get_relay_task_api_base_url(),
            "BUSINESS_API_ENDPOINT_ORIGIN": _default_business_api_endpoint_origin(),
        },
    }, status=200)
```

**删除的依赖：** 不再调用 `_build_runtime_env_for_relay()` / `build_relay_to_trae_runtime_env()` / `_replace_access_token_placeholder()`。

### 3.2 `relay_to_trae_repo_credentials_precheck` — 两阶段 Go 调用

**文件：** `relay_to_trae_proxy.py` 第 510 行

```python
# 旧：_issue_relay_access_token() → 查 CloudServerConfig + SQLite hack → 调 Go repo-clone-credentials
# 新：调 Go token-init → 拿 token → 调 Go repo-clone-credentials

def relay_to_trae_repo_credentials_precheck(request, ...):
    ...
    credential_svc_url = _task_credential_service_url()

    # 1. 从 Go 获取 token（SSOT）
    token_resp = _credential_service_post(
        f"{credential_svc_url}/v1/token/init",
        json={"tenant_id": tenant, "workspace_id": wid, "task_id": tid},
        timeout=5,
    )
    if not token_resp.ok:
        return _relay_error_response(token_resp)
    access_token = str(token_resp.json().get("access_token") or "").strip()
    if not access_token:
        return Response({"status": "error", "message": "无法获取 access token"}, status=502)

    # 2. 用 Go 签发的 token 调用凭证构建（原逻辑保留）
    precheck_url = (
        f"{credential_svc_url}/api/tenant/{tenant}/workspace/{wid}/task/{tid}"
        "/cloud/server-container-token/repo-clone-credentials/"
    )
    resp = _credential_service_post(precheck_url, json={"access_token": access_token}, timeout=8)
    # ... 返回处理（200/409/502 分支不变）
```

**删除的调用：** `_issue_relay_access_token()`。

### 3.3 `relay_to_trae_start` — token 从 Go 获取

**文件：** `relay_to_trae_proxy.py` 第 1015 行

```python
# 旧：_build_runtime_env_for_relay() → Django 生成 token
# 新：调 Go token-init → 注入 runtime_env

def relay_to_trae_start(request, ...):
    ...
    credential_svc_url = _task_credential_service_url()

    # 从 Go 获取 token
    token_resp = _credential_service_post(
        f"{credential_svc_url}/v1/token/init",
        json={"tenant_id": tenant, "workspace_id": wid, "task_id": tid},
        timeout=5,
    )
    if not token_resp.ok:
        return _relay_error_response(token_resp)
    token_data = token_resp.json()
    access_token = str(token_data.get("access_token") or "").strip()
    if not access_token:
        return Response({"status": "error", "message": "无法获取 access token"}, status=502)

    # 构建 runtime_env（不再包含 token 生成逻辑）
    runtime_env = {
        "TASK_API_ENDPOINT_ORIGIN": settings_manager.get_relay_task_api_base_url(),
        "BUSINESS_API_ENDPOINT_ORIGIN": _default_business_api_endpoint_origin(),
        "ACCESS_TOKEN": access_token,
    }
    # ... 其余逻辑不变
```

### 3.4 `register_or_reuse` — token 从 Go 获取

**文件：** `relay_to_trae_proxy.py` 第 860 行

同样将 `_issue_relay_access_token()` 替换为 Go `/v1/token/init` 调用。

### 3.5 新增辅助函数

```python
def _task_credential_service_url() -> str:
    """Go taskCredentialService 的 base URL."""
    return str(settings_manager.get_task_credential_service_url() or "http://127.0.0.1:8015").strip().rstrip("/")

def _credential_service_post(url: str, *, json=None, timeout=8, **kwargs) -> requests.Response:
    """调用 taskCredentialService HTTP API，禁用系统代理。"""
    with requests.Session() as session:
        session.trust_env = False
        return session.post(
            url,
            json=json,
            headers={"Accept": "application/json", "Content-Type": "application/json"},
            timeout=timeout,
            **kwargs,
        )
```

---

## 4. 删除的代码

| 删除项 | 文件 | 原因 |
|--------|------|------|
| `_sync_token_to_credential_service()` | `mock_run_container.py:286-387` | SQLite 直写 hack，违反 SSOT |
| `_sync_credential_service_token()` | `relay_to_trae_proxy.py:209-231` | 同上包装函数 |
| `_issue_relay_access_token()` | `relay_to_trae_proxy.py:234-292` | Django 自生成 token + 查 CloudServerConfig + 调 sync hack |
| `_resolve_relay_precheck_task_api_origin()` | `relay_to_trae_proxy.py:505-507` | 统一使用 `_task_credential_service_url()` |
| `_replace_access_token_placeholder()` 中的 token 生成 + `_sync_token_to_credential_service` 调用 | `mock_run_container.py:449-493` | Django 不再生成 token |
| `build_relay_to_trae_runtime_env()` 中 relayToTrae 路径的调用 | `mock_run_container.py:674-734` | relayToTrae 不再通过此路径获取 token |
| `_build_runtime_env_for_relay()` | `relay_to_trae_proxy.py:382-394` | 不再需要 |
| `TASK2APP_ACCESS_TOKEN_PLACEHOLDER` 在 relayToTrae 路径的使用 | `mock_run_container.py:696` | 不再有占位符替换 |
| 新增的 `import sqlite3, os, uuid`（仅服务于 `_sync_token_to_credential_service`） | `mock_run_container.py` | hack 撤销 |

**注意：** `_replace_access_token_placeholder()` 和 `build_relay_to_trae_runtime_env()` 还被 **mockStart 流程**使用。mockStart 暂不迁移（独立处理），仅移除 relayToTrae 路径的调用。如果函数仅剩 mockStart 路径，保留但清理 relayToTrae 专属逻辑。

---

## 5. 变更文件清单

| 文件 | 变更 | 类别 |
|------|------|------|
| `cloud/services/relay_to_trae_proxy.py` | 新增 `_task_credential_service_url()` + `_credential_service_post()`；修改 `token_init`/`precheck`/`start`/`register_or_reuse` 四个函数改为调 Go；删除 `_sync_credential_service_token`/`_issue_relay_access_token`/`_resolve_relay_precheck_task_api_origin`/`_build_runtime_env_for_relay` | 核心 |
| `cloud/services/mock_run_container.py` | 删除 `_sync_token_to_credential_service()` 及其 `import sqlite3, os, uuid`；`_replace_access_token_placeholder()` 移除 relayToTrae token 生成 + sync 调用 | 清理 |
| `tests/test_relay_to_trae_proxy.py` | mock `_credential_service_post` 替代 mock `CloudServerConfig.objects` / `_issue_relay_access_token`；更新所有预检/启动测试 | 测试 |

**不修改的文件：**
- `taskCredentialService/` — Go 服务已有 `/v1/token/init` 和 `repo-clone-credentials`，无需变更
- 前端 — token-init / precheck / start 三步流程不变，HTTP 契约不变
- `conf/` — `taskCredentialServiceBase` 配置已存在

---

## 6. 关于 `_replace_access_token_placeholder` 的保留

此函数仍被 **mockStart** 流程使用（`mock_run_container.py:917`）。mockStart 是独立流程，本次不做迁移。

处理方式：
- relayToTrae 路径（`build_relay_to_trae_runtime_env`）不再调用此函数
- 函数本身保留，但移除其中的 `_sync_token_to_credential_service()` 调用（因为该函数即将被删除）
- mockStart 路径临时容忍 Django 继续写 `CloudServerConfig`（后续 Phase 单独迁移）

**如果 mockStart 也依赖 token 被同步到 Go**，则 mockStart 也需要改为调 Go。但这超出本次范围，作为独立议题。

---

## 7. Token SSOT 最终状态

```
                     taskCredentialService (:8015)
                     ┌────────────────────────────┐
                     │  container_tokens (SQLite)  │  ← Token SSOT
                     │  - container_access_token   │
                     │  - expires_at               │
                     │  - scope (tenant/task)      │
                     │                             │
                     │  /v1/token/init             │  ← Token 签发（唯一入口）
                     │  /.../repo-clone-credentials│  ← 凭证构建（需 token）
                     │  /.../task-detail           │  ← 任务详情
                     └────────────────────────────┘
                            ▲           ▲
                            │           │
              HTTP (token)  │           │ HTTP (token)
                            │           │
              ┌─────────────┘           └──────────────┐
              │                                        │
     Django (:8001)                          Container (runtime)
     ┌──────────────────┐                    ┌──────────────┐
     │ 不存 token       │                    │ 持有 token   │
     │ 每次请求调 Go     │                    │ 回调换凭证   │
     │ /v1/token/init   │                    │              │
     └──────────────────┘                    └──────────────┘
```

---

## 8. 领域概念（轻量）

- **Bounded Context：** 任务协作 / 容器运行时 / relay 直启
- **实体：** `ContainerToken`（Go 域）、`Todo`、`TaskRepoIdentity`
- **值对象：** `TaskScope`、`AccessToken`
- **领域服务：** `TokenService.IssueToken`（Go）、`CredentialService.BuildRepoCloneCredentials`（Go）
- **应用服务：** `relay_to_trae_token_init` / `precheck` / `start`（Django 编排层）
- **反模式（消除目标）：** Django `CloudServerConfig` 作为 token 第二存储、`_sync_token_to_credential_service` SQLite 跨服务直写

---

## 9. 价值流影响

| 流 | 影响 |
|----|------|
| `task-detail-runtime-relay` | relayToTrae 全链路 token 操作改为 Go SSOT；消除双写 |
| `task-detail-repo-clone-credentials-contract` | 凭证构建逻辑不变（仍在 Go），仅 token 来源从 Django→Go DB hack 变为 Go 原生签发 |

mockStart 流暂不影响。

---

## 10. 测试计划

### 后端

- **更新** `test_relay_to_trae_token_init_*`：mock `_credential_service_post` 替代 mock `_build_runtime_env_for_relay`
- **更新** `test_relay_to_trae_repo_credentials_precheck_*`：mock Go token-init + repo-clone-credentials 两个调用
- **更新** `test_relay_to_trae_start_*`：mock Go token-init
- **新增** `test_token_init_calls_go_service`：验证 token_init 调用 Go `/v1/token/init`
- **新增** `test_precheck_two_phase_go_calls`：验证 precheck 先调 token-init 再调 repo-clone-credentials
- **新增** `test_precheck_handles_go_token_init_failure`：Go 返回 500 → 预检返回 502
- **删除** `test_*_issue_relay_access_token_*`：`_issue_relay_access_token` 不再存在
- **删除** `test_relay_to_trae_repo_credentials_precheck_uses_internal_task_api_origin`：`_resolve_relay_precheck_task_api_origin` 不再存在

### 前端 & E2E

- 不变 — HTTP 契约不变，三步流程不变

---

## 11. 验收标准

1. `token_init` → 调 Go `/v1/token/init`，Django 不写 `cloud_cloudserverconfig.container_access_token`
2. `precheck` → 两阶段 Go 调用（token-init → repo-clone-credentials），不再查 Django `CloudServerConfig`
3. `start` → 调 Go `/v1/token/init`，token 注入 runtime_env 发给 go_relayToTrae
4. `register-or-reuse` → 同上
5. `_sync_token_to_credential_service()` / `_sync_credential_service_token()` / `_issue_relay_access_token()` 已删除
6. 跨语言 SQLite 直写 hack（`sqlite3.connect("db/container/tokens.sqlite3")`）已删除
7. 容器回调 `repo-clone-credentials` / `task-detail` 行为不变
8. `--noreload` 单线程模式 → 全链路正常（Django→Go HTTP 调用，不存在自调用）
9. 所有已有测试更新通过，新增测试覆盖 Go 调用失败场景

---

## 12. 非目标

- mockStart 流程暂不迁移（仍写 `CloudServerConfig`，独立处理）
- 不在此变更引入 Domain Events
- 不修改 `--noreload` 配置
- 前端流程不变
- 不改变容器回调 HTTP 契约
- taskCredentialService Go 代码本次不改

---

## 13. 路线图

```
Phase 1（本次）:
  relayToTrae 流程改为 taskCredentialService Token SSOT
  + 删除 _sync_token_to_credential_service / _sync_credential_service_token / _issue_relay_access_token
  + 删除跨语言 SQLite 直写 hack

Phase 2（后续）:
  mockStart 流程迁移到 taskCredentialService

Phase 3（后续）:
  废弃 CloudServerConfig.container_access_token 字段

Phase 4（远期）:
  Domain Events（ContainerAccessTokenIssued → Kafka → 各 BC 投影）
```

---

## 14. 权限影响分析（Step 2 产出）

**结论：✅ 绿灯 — 纯内部重构，权限模型不变**

### 14.1 变更端点权限矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 变更影响 |
|--------|------|----------|------|----------|----------|
| `POST .../relay-to-trae/token-init` | 认证用户 | Task | read | `IsAuthenticated` (ViewSet级) | 零 — token 来源从 Django→Go |
| `POST .../relay-to-trae/repo-credentials-precheck` | 认证用户 | Task | read | `IsAuthenticated` (ViewSet级) | 零 — 两阶段 Go 调用替代 Django+SQLite hack |
| `POST .../relay-to-trae/start` | 认证用户 | Task | write | `IsAuthenticated` (ViewSet级) | 零 — token 来源从 Django→Go |
| `POST .../relay-to-trae/register` | 认证用户 | Task | write | `IsAuthenticated` (ViewSet级) | 零 — token 来源从 Django→Go |
| `POST /v1/token/init` (Go 内部) | Django 内部服务 | System | write | 仅 localhost 绑定 | ⚠️ 无 secret 验证（已有风险，非本次引入） |

所有端点 URL、HTTP 方法、权限类、认证类 **均不变**。本次变更本质是 token 来源从 Django 切换为 Go SSOT，对 API 消费者完全透明。

### 14.2 安全审查

| 检查项 | 状态 |
|--------|------|
| IDOR 风险 | ✅ 无新增 |
| 权限提升 | ✅ 无新增 |
| 跨租户泄露 | ✅ Go `IssueToken` 按 `(tenant_id, task_id)` 查询，不会跨租户 |
| 敏感操作审计 | ✅ `append_container_token_audit_event` 保留 + Go 内 `recordAudit` |
| Go `/v1/token/init` 无认证 | ⚠️ 已有风险（仅 localhost 绑定），建议独立 Phase 添加 `X-TaskGateway-Internal-Secret` |

### 14.3 新增权限测试

**本次无需新增。** 若后续 Phase 为 Go `/v1/token/init` 添加 internal secret 认证，需补充：无 secret → 401、错误 secret → 401、正确 secret → 200。
