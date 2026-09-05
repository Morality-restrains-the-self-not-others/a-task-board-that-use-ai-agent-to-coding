# 实施计划: 登录跳转循环修复 — 网关统一认证

> 来源:
> - 设计: `docs/design/login-redirect-loop-fix.md`
> - 价值流: `docs/superpowers/plans/2026-06-23-login-redirect-loop-gateway-auth-fix-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-23-login-redirect-loop-gateway-auth-fix-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-23-login-redirect-loop-gateway-auth-fix-ddd.md`

## 概述

- **总任务数**: 6
- **涉及文件**: 3 个 Python 文件 + 配置
- **预计改动行数**: ~60 行（新增/修改）

---

## Task 1: 新增本地 token 持久化函数

**文件**: `Saas_project/accounts/taskauth_bridge/token_registry.py`
**状态**: pending

在现有内存测试 registry 基础上，新增基于 Django DB 的 token 存储和查询函数。

```python
def upsert_local_token(key: str, user_id: str) -> None:
    """本地持久化 token，支持无 session 解析。"""
    ...

def resolve_local_token(key: str) -> str | None:
    """从本地 DB 解析 token，返回 user_id 或 None。"""
    ...
```

**验证**: 导入成功，函数签名正确
**依赖**: 无

---

## Task 2: 登录成功时本地写回 token

**文件**: `Saas_project/accounts/taskauth_bridge/delegate.py`
**状态**: pending

修改 `delegate_login`，登录成功后调用 `upsert_local_token` 将 token 同步写入本地 DB。
同时修改 `delegate_post` 和 `_response_from_taskauth`，保留并传播 taskAuth 返回的响应头（Set-Cookie）。

```python
# delegate_post: 保留 response_headers
status, data, response_headers = forward_to_taskauth(...)  # 不再丢弃
return _response_from_taskauth(status, data, response_headers)

# _response_from_taskauth: 接受并传播 headers
def _response_from_taskauth(status, data, response_headers=None) -> Response:
    resp = ...
    if response_headers:
        for key, value in response_headers.items():
            if key.lower() in ('set-cookie',):
                resp[key] = value
    return resp

# delegate_login: 登录成功时写回 token
def delegate_login(request) -> Response | None:
    resp = delegate_post(...)
    if resp is not None and resp.status_code == 200:
        _persist_token_locally(resp.data)
    return resp
```

**验证**: 单元测试 mock taskAuth 响应 → 验证 `upsert_local_token` 被调用
**依赖**: Task 1

---

## Task 3: `load_principal_from_token` 增加本地 DB 回退

**文件**: `Saas_project/accounts/taskauth_bridge/principal_loader.py`
**状态**: pending

在 `TASKAUTH_ENABLED=False` 路径中，测试 registry 未命中时增加本地 DB 查询：

```python
if not taskauth_enabled():
    uid = resolve_test_token(key)       # 1) 内存测试 registry
    if uid:
        return load_principal_from_user_id(uid)
    uid = resolve_local_token(key)      # 2) 本地 DB (新增)
    if uid:
        return load_principal_from_user_id(uid)
    return None                         # 3) 无更多回退
```

**验证**: 单元测试 — 写入 token → 调用 `load_principal_from_token` → 返回正确 User
**依赖**: Task 1

---

## Task 4: 确认 Secret 配置一致

**涉及**: 环境变量 / `conf/port_config.json` / YAML 配置
**状态**: pending

1. 确认 taskAuth 的 `TASKAUTH_INTERNAL_SECRET`（或 YAML `internalSecret`）与 APISIX forward-auth 配置的 `X-TaskAuth-Internal-Secret` 一致（当前 APISIX 硬编码值: `taskauth-local-dev-secret-do-not-use-in-prod`）
2. 确认 Django 的 `TASK_GATEWAY_INTERNAL_SECRET` 与 APISIX transformer 注入的 `X-TaskGateway-Internal-Secret` 一致（当前 APISIX 硬编码值: `task-gateway-local-dev-secret-do-not-use-in-prod`）
3. 如 taskAuth `InternalSecret` 为空字符串，则 `requireInternalSecret` 检查被跳过，forward-auth 仍可工作（但会返回 401 而非 403 当 token 无效时）

**验证**: `curl -H "Authorization: Token <valid_token>" http://183.250.1.132:18081/api/accounts/users/profile/` 返回 200
**依赖**: 无（可与 Task 1-3 并行）

---

## Task 5: 编写单元测试

**文件**: `Saas_project/tests/test_token_local_persistence.py`（新建）
**状态**: pending

覆盖场景：
- `upsert_local_token` 写入 → `resolve_local_token` 读出 → user_id 匹配
- `resolve_local_token` 查询不存在的 token → 返回 None
- `load_principal_from_token` with `TASKAUTH_ENABLED=False` → 本地 DB 命中 → 返回 User
- `delegate_post` 传播 Set-Cookie 响应头

**验证**: `pytest tests/test_token_local_persistence.py -v` 全部通过
**依赖**: Task 1-3

---

## Task 6: Playwright 端到端验证

**脚本**: `playwright/front_project/tests/LoginRedirectFix.e2e.test.js`（新建）
**状态**: pending

覆盖场景：
- 登录成功 → 跳转到 `/system-admin/` → 不跳回 `/auth/login/`
- 登录后 profile API 返回 200（网关 forward-auth 通过或本地 fallback 生效）

**验证**: `npx playwright test LoginRedirectFix` 通过
**依赖**: Task 1-5

---

## 依赖关系图

```
Task 1 (token_registry) ──┬── Task 2 (delegate) ──┐
                          ├── Task 3 (principal_loader) ──┤
                          │                              ├── Task 5 (unit tests) ── Task 6 (E2E)
Task 4 (secret config) ───┘                              │
                                                         └── (独立并行)
```

## 执行顺序

1. Task 4 (配置 — 独立，可先做)
2. Task 1 (token_registry — 被 Task 2/3 依赖)
3. Task 2 + Task 3 (并行 — 都只依赖 Task 1)
4. Task 5 (单元测试 — 依赖 Task 1-3)
5. Task 6 (E2E — 依赖全部)
