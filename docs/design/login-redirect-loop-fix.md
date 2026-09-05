# 设计文档：登录跳转循环修复 — 网关统一认证

## 1. 问题复现 (Playwright 验证)

| 步骤 | 请求 | 状态 | 关键发现 |
|------|------|------|----------|
| 1 | `GET /auth/login/` | 200 | SPA 登录页正常加载 |
| 2 | `POST /api/auth/` | 200/502 | 登录 API 不稳定，成功时返回 token + user + redirect_url |
| 3 | `GET /api/accounts/users/profile/` | **403** | APISIX forward-auth 拒绝（`server: APISIX/3.11.0`，空 body） |
| 4 | → 跳回 `/auth/login/` | — | Auth guard 清除 localStorage token，死循环 |

登录成功时返回体（200）：
```json
{
  "redirect_url": "/system-admin/",
  "token": "8b8c47ce96a1d19bba450366c26517d42f785881",
  "user": { "id": "850249621660790784", "is_active": true, "is_superuser": true }
}
```

## 2. 根因分析

### 2.1 现有架构：网关 forward-auth 机制

项目**已经实现**了用户期望的网关统一认证模式：

```
                          ┌─────────── 当前架构 ───────────┐
                          │                                 │
  浏览器                   APISIX Gateway                   taskAuth (Go)          Django
  ──────                   ──────────────                   ─────────────          ──────
  GET /api/.../profile/    │                               │                      │
  Authorization: Token xx  │                               │                      │
      ───────────────────▶ │  transformer 插件:             │                      │
                           │    ① 剥离客户端 X-User-Id      │                      │
                           │    ② 注入 X-TaskGateway-       │                      │
                           │       Internal-Secret          │                      │
                           │                               │                      │
                           │  forward-auth 插件:            │                      │
                           │    POST /gateway/forward-auth/ │                      │
                           │    ───────────────────────────▶│ handleGatewayFwd()  │
                           │    X-TaskAuth-Internal-Secret   │   resolveToken()    │
                           │    Authorization: Token xx     │   SELECT customtoken│
                           │                               │   → user_id         │
                           │    ◀───────────────────────────│   200 + X-User-Id   │
                           │    ③ 注入 X-User-Id,           │                      │
                           │       X-Gateway-Auth-Verified  │                      │
                           │                               │                      │
                           │    ④ 转发到上游 (含所有头)     │                      │
                           │    ──────────────────────────────────────────────────▶│
                           │                               │   CustomTokenAuth     │
                           │                               │   ⑤ 检查网关头:       │
                           │                               │   X-Gateway-Auth-     │
                           │                               │   Verified=1 ✓        │
                           │                               │   X-User-Id ✓         │
                           │                               │   Internal-Secret ✓   │
                           │                               │   → 直接信任，跳过    │
                           │                               │     token 解析        │
                           │    ◀──────────────────────────────────────────────────│
      ◀─────────────────── │  200                         │  200                  │
```

### 2.2 问题定位：forward-auth 返回 403

Playwright 抓包显示 profile 请求的 403 **来自 APISIX**（`server: APISIX/3.11.0`），空 body。这说明 forward-auth 子请求（APISIX → taskAuth）返回了非 2xx，APISIX 直接拒绝请求，**从未到达 Django**。

taskAuth 的 `handleGatewayForwardAuth`（`gateway_forward_auth.go`）在以下情况返回非 2xx：

| 条件 | 返回 |
|------|------|
| `X-TaskAuth-Internal-Secret` 缺失或与 `cfg.InternalSecret` 不匹配 | **403** |
| 请求无 `Authorization` 头或 token 为空 | **401** |
| token 在 `accounts_customtoken` 表中未找到 | **401** |
| 用户 `is_active=false` | **401** |

**最可能原因：secret 不匹配导致 403。**

APISIX 配置中硬编码了 `X-TaskAuth-Internal-Secret: taskauth-local-dev-secret-do-not-use-in-prod`。如果 taskAuth 的 `cfg.InternalSecret`（来自环境变量 `TASKAUTH_INTERNAL_SECRET` 或 YAML 配置）与此不匹配，`requireInternalSecret()` 返回 false → 403。

> 注：若 `cfg.InternalSecret == ""`，检查会被跳过（返回 true），token 解析会继续。此时若无 token 或 token 无效，返回 **401** 而非 403。观察到的 403 说明 secret **已配置但不匹配**。

### 2.3 次要问题

| 问题 | 位置 | 影响 |
|------|------|------|
| `delegate_post` 丢弃响应头 | `delegate.py:39` (`_ = forward_to_taskauth(...)`) | taskAuth 返回的 `Set-Cookie` 丢失（虽 token 认证不需 session，但其他功能可能依赖）|
| `TASKAUTH_ENABLED=False` 时 token 解析只查内存 registry | `principal_loader.py:45-51` | 网关 header 短路失败时，fallback 不可用 |
| 登录路由 `/api/auth/` 无 forward-auth | `routes.yaml` `auth_mode: none` | 登录请求不走网关认证（这是正确的），但响应中无 session cookie |

## 3. 设计方案：完善网关统一认证

### 3.1 核心思路

用户提出的思路与现有架构完全一致：

> taskGateway 根据 token 向 taskAuth 获取用户 ID，然后把用户 ID 传给后端服务，后端服务不再自行解析 token

**这套机制已经实现**（见 2.1 架构图），修复方向是**补齐配置缺口**和**加固 fallback**。

### 3.2 改动清单

### 改动 1：统一 Secret 配置（关键修复）

**问题**：APISIX forward-auth 发送的 `X-TaskAuth-Internal-Secret` 与 taskAuth `cfg.InternalSecret` 不匹配，导致 forward-auth 拒绝所有请求。

**修复**：在 taskAuth 的环境变量或 YAML 配置中，设置与 APISIX 一致的 secret。

```yaml
# taskAuth YAML 配置 (auth/task-auth)
internalSecret: "taskauth-local-dev-secret-do-not-use-in-prod"
```

或设置环境变量：
```bash
export TASKAUTH_INTERNAL_SECRET="taskauth-local-dev-secret-do-not-use-in-prod"
```

同时确保 Django 的 `TASK_GATEWAY_INTERNAL_SECRET` 与 APISIX transformer 注入的值匹配：
```bash
export TASK_GATEWAY_INTERNAL_SECRET="task-gateway-local-dev-secret-do-not-use-in-prod"
```

> **注意**：这些是开发环境默认值。生产环境应使用独立的安全 secret。

**涉及文件**：无代码改动，仅配置/环境变量。配置文件位于 `conf/` 目录或环境变量。

### 改动 2：修复 `delegate_post` 传播响应头

**问题**：`delegate.py:39` 将 taskAuth 响应头丢弃（`_`），包括可能的 `Set-Cookie`。

**修复**：保留并传播响应头。

```python
# delegate.py
def delegate_post(path: str, request, *, token: str | None = None) -> Response | None:
    if not taskauth_enabled():
        return None
    headers = {}
    if request.META.get('HTTP_COOKIE'):
        headers['Cookie'] = request.META['HTTP_COOKIE']
    headers = with_trace_headers(headers, request=request, ...)
    try:
        status, data, response_headers = forward_to_taskauth(  # 不再丢弃
            path,
            body=dict(request.data) if hasattr(request, 'data') else {},
            headers=headers,
            token=token,
        )
    except Exception:
        return None
    return _response_from_taskauth(status, data, response_headers)  # 传入 headers


def _response_from_taskauth(status: int, data, response_headers: dict | None = None) -> Response:
    resp = Response(status=204) if status == 204 else Response(data, status=status)
    if response_headers:
        for key, value in response_headers.items():
            if key.lower() in ('set-cookie',):
                resp[key] = value
    return resp
```

**涉及文件**：`Saas_project/accounts/taskauth_bridge/delegate.py`

### 改动 3：加固 `TASKAUTH_ENABLED=False` 时的 fallback

**问题**：当网关 header 验证失败且 `TASKAUTH_ENABLED=False` 时，`load_principal_from_token` 只查内存测试 registry，真实 token 永远返回 None。这导致即使 Django 收到了请求（网关放行），也无法认证。

**修复**：增加本地 token 解析路径。当 `TASKAUTH_ENABLED=False` 时，除了检查内存测试 registry，还检查 Django 的 `CustomToken` 表。同时，登录成功时将 token 同步写入此表。

**a) `delegate.py` — 登录时写回 token**
```python
def delegate_login(request) -> Response | None:
    resp = delegate_post('/api/accounts/users/login/', request)
    if resp is not None and resp.status_code == 200:
        _persist_token_locally(resp.data)
    return resp

def _persist_token_locally(data: dict) -> None:
    from accounts.taskauth_bridge.token_registry import upsert_local_token
    token = data.get('token')
    user = data.get('user') or {}
    user_id = user.get('id')
    if token and user_id:
        upsert_local_token(key=token, user_id=str(user_id))
```

**b) `token_registry.py` — 新增本地持久化**
```python
def upsert_local_token(key: str, user_id: str) -> None:
    from accounts.models.token import CustomToken
    from django.contrib.contenttypes.models import ContentType
    from django.contrib.auth import get_user_model
    User = get_user_model()
    ct = ContentType.objects.get_for_model(User)
    CustomToken.objects.update_or_create(
        key=key,
        defaults={'object_id': user_id, 'content_type': ct},
    )

def resolve_local_token(key: str) -> str | None:
    from accounts.models.token import CustomToken
    ct = CustomToken.objects.filter(key=key).first()
    return ct.object_id if ct else None
```

**c) `principal_loader.py` — 增加本地 fallback**
```python
if not taskauth_enabled():
    uid = resolve_test_token(key)       # 1) 内存测试 registry（现有）
    if uid:
        return load_principal_from_user_id(uid)
    uid = resolve_local_token(key)      # 2) 本地 DB（新增）
    if uid:
        return load_principal_from_user_id(uid)
    return None                         # 3) 无更多回退
```

**涉及文件**：
- `Saas_project/accounts/taskauth_bridge/delegate.py`
- `Saas_project/accounts/taskauth_bridge/token_registry.py`
- `Saas_project/accounts/taskauth_bridge/principal_loader.py`

### 3.3 认证流程（修复后）

```
                          正常路径 (forward-auth 成功)
                          ─────────────────────────
  浏览器 → APISIX → forward-auth → taskAuth resolve → 200 + X-User-Id
                 → 转发到 Django（携带 X-User-Id, X-Gateway-Auth-Verified）
                 → Django CustomTokenAuth 信任网关头 → 直接 load user → ✅

                          降级路径 (forward-auth 失败/无网关头)
                          ───────────────────────────────────
  浏览器 → APISIX → forward-auth 失败/未配置 → 请求可能被拒绝或放行
         如果放行 → Django CustomTokenAuth:
           TASKAUTH_ENABLED=True  → HTTP 调用 taskAuth resolve → ✅
           TASKAUTH_ENABLED=False → 内存 registry → 本地 DB → ✅ (新增)
```

### 3.4 后端服务的职责简化

修复后的后端服务（Django）认证逻辑：

```python
# authentication.py — 已成型的逻辑，无需改动
def authenticate(self, request):
    if TASK_GATEWAY_TRUST_HEADERS:
        # 网关头有效 → 直接信任，不再解析 token
        if verified and secret_match and user_id:
            return (load_principal_from_user_id(user_id), stub)
    # 网关头无效/不存在 → fallback 自解析 token
    return super().authenticate(request)
```

**设计原则**：网关是认证的唯一决策点。后端服务信任网关传递的 `X-User-Id`，不关心 token 格式。仅当网关未部署或故障时，后端才自行解析 token。

## 4. 影响范围

| 文件 | 改动 | 风险 |
|------|------|------|
| `accounts/taskauth_bridge/delegate.py` | `delegate_post` 传播响应头 + `delegate_login` 写回 token | 低 |
| `accounts/taskauth_bridge/token_registry.py` | 新增 `upsert_local_token` / `resolve_local_token` | 低 |
| `accounts/taskauth_bridge/principal_loader.py` | `load_principal_from_token` 增加本地 DB 回退 | 低 |
| taskAuth 配置 | 确保 `TASKAUTH_INTERNAL_SECRET` 与 APISIX 匹配 | 配置变更，需重启 |
| Django 配置 | 确保 `TASK_GATEWAY_INTERNAL_SECRET` 与 APISIX 匹配 | 配置变更，需重启 |

**不改变**：
- APISIX 路由配置（forward-auth 插件配置已完成）
- taskAuth Go 服务代码
- 前端代码
- `CustomTokenAuthentication.authenticate()` 网关 header 逻辑

## 5. 领域概念

- **Bounded Context**: 用户认证 (Auth)
- **Entity**: User (id=`850249621660790784`)
- **Value Object**: Token (40-char hex key), GatewayTrustHeaders
- **Repository**: `token_registry`（内存 + DB 双实现）
- **Gateway**: APISIX forward-auth → taskAuth `handleGatewayForwardAuth`（统一认证入口）
- **受影响的 Value Stream**: `user-auth` → `login` 步骤

## 6. 测试策略

| 场景 | 测试方法 |
|------|----------|
| 网关 forward-auth 成功 → Django 信任网关头 | 单元测试：模拟网关头 → `authenticate()` 返回 principal |
| 网关 forward-auth 失败/无网关头 → fallback 到本地 token 解析 | 单元测试：无网关头 + `TASKAUTH_ENABLED=False` → 查本地 DB |
| 登录成功 → token 本地持久化 | 单元测试：mock taskAuth 响应 → 验证 `upsert_local_token` |
| 端到端：登录 → 访问需认证页面 | Playwright：登录后访问 `/system-admin/`，验证不跳回 |
