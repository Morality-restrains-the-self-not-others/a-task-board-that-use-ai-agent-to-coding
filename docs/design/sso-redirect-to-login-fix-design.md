# 设计文档：SSO 跳转到 /auth/login/ 修复 — Gateway Auth Middleware

**日期**: 2026-06-29
**类型**: Bug Fix
**依赖**: SSO 401 修复（已完成）

## 1. 问题描述

Forward-auth 401 修复后，APISIX 正确返回 `X-User-Id` header。但点击「镜像市场管理（SSO）」后跳转到 `/auth/login/` 登录页，而非 AiProvider 管理后台。

## 2. 根因

APISIX forward-auth header 处理逻辑封装在 DRF 认证类 `CustomTokenAuthentication` 中：

```python
# accounts/authentication.py:35-52
class CustomTokenAuthentication(DRFTokenAuthentication):
    def authenticate(self, request):
        if getattr(settings, 'TASK_GATEWAY_TRUST_HEADERS', False):
            verified = request.META.get('HTTP_X_GATEWAY_AUTH_VERIFIED') == '1'
            ...
            principal = load_principal_from_user_id(user_id)
            if principal is not None:
                return (principal, _TokenStub(''))
        return super().authenticate(request)
```

**该逻辑只在以下场景执行：**
1. DRF API views（通过 `REST_FRAMEWORK.DEFAULT_AUTHENTICATION_CLASSES`）
2. `/api/system-admin/*` 路径（通过 `SystemAdminPermissionMiddleware` 手动调用）

**SSO 视图 `sso_ai_provider_admin_redirect` 是纯 Django 函数视图（`HttpResponseRedirect`），路径 `/accounts/sso/ai-provider/admin/` 不匹配任何已有路径前缀，所以 `request.user` 始终是 `AnonymousUser`。**

```
请求链路:
  APISIX → X-User-Id, X-Gateway-Auth-Verified
    → AuthenticationMiddleware → request.user = AnonymousUser (无 session cookie)
    → ApiAuthRedirectMiddleware → 跳过（非 /api/）
    → AuthorizationIdValidationMiddleware → 跳过（非 /api/）
    → SystemAdminPermissionMiddleware → 跳过（非 /api/system-admin/）
    → sso_ai_provider_admin_redirect → request.user.is_authenticated = False → /auth/login/
```

## 3. 修复方案

**新增 `GatewayAuthMiddleware`**，在 `AuthenticationMiddleware` 之后运行，将 forward-auth header 处理逻辑从 DRF 认证类提升到 Django 中间件层，覆盖**所有** Django 视图。

### 设计

```python
# core/middleware.py 新增

class GatewayAuthMiddleware:
    """APISIX forward-auth headers → request.user（所有视图）"""

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        if not request.user.is_authenticated:
            self._authenticate_from_gateway_headers(request)
        return self.get_response(request)

    def _authenticate_from_gateway_headers(self, request):
        from django.conf import settings

        if not getattr(settings, 'TASK_GATEWAY_TRUST_HEADERS', False):
            return

        verified = request.META.get('HTTP_X_GATEWAY_AUTH_VERIFIED') == '1'
        expected = str(getattr(settings, 'TASK_GATEWAY_INTERNAL_SECRET', '') or '').strip()
        provided = str(request.META.get('HTTP_X_TASKGATEWAY_INTERNAL_SECRET', '') or '').strip()
        user_id = str(request.META.get('HTTP_X_USER_ID', '') or '').strip()

        if not (verified and expected and provided == expected and user_id):
            return

        from accounts.taskauth_bridge.principal_loader import load_principal_from_user_id

        try:
            principal = load_principal_from_user_id(user_id)
        except Exception:
            return

        if principal is not None:
            request.user = principal
```

### 中间件顺序

```python
MIDDLEWARE = [
    ...
    'django.contrib.auth.middleware.AuthenticationMiddleware',  # 第9位
    'core.middleware.GatewayAuthMiddleware',                     # 第10位 ← 新增
    'core.middleware.ApiAuthRedirectMiddleware',                 # 原第10位 → 第11位
    ...
]
```

### 安全考量

1. **Session 优先**：已认证用户（有 session）不会被覆盖
2. **Header 验证**：`X-Gateway-Auth-Verified` + `X-TaskGateway-Internal-Secret` 双重校验
3. **用户存在性**：`load_principal_from_user_id` 验证用户存在于 taskAuth
4. **异常安全**：`load_principal_from_user_id` 抛出异常时静默失败，不清空已有认证

## 4. 价值流影响

| Stream | 影响 |
|--------|------|
| `user-auth` → `gateway-forward-auth-token-resolve` (active) | 此 step 现已 active，但仅完成网关头认证；需配合本修复完成 Django 端认证 |
| `user-auth` → `frontend-auth-guard-redirect` | SSO 跨端口跳转现在完整工作 |

## 5. 变更清单

| 文件 | 变更 |
|------|------|
| `core/middleware.py` | 新增 `GatewayAuthMiddleware` 类 |
| `saas_project/settings.py` | MIDDLEWARE 中插入 `GatewayAuthMiddleware`（AuthenticationMiddleware 之后） |

## 6. 测试

- Go 单元测试：forward-auth cookie bridge（已有，6/6 pass）
- Playwright E2E：登录 → 点击 SSO → 验证跳转到 AiProvider（非 /auth/login/）

---

## 变更记录（2026-07-13）

### 复现（production www.daydaymoney.com）

已登录超管在 `/system-admin/users/` 点击「镜像市场管理（SSO）」仍 302 到 `/auth/login/?next=/accounts/sso/ai-provider/admin/`。
日志显示同会话 `/api/user/.../me/` 返回 200，但 `/accounts/sso/...` 请求源 IP 为公网直达 Django（非网关容器 IP）。

### 新增根因

边缘 nginx（`daydaymoney.hk.nginx.example`）仅将 `/api/` 代理到 APISIX；`/accounts/sso/` 落入 `location /` → SPA/Django :8001，**绕过 forward-auth**，`GatewayAuthMiddleware` 收不到 `X-User-Id`。

### 追加修复

1. **Django**：`GatewayAuthMiddleware` 对 `/accounts/sso/` 增加 cookie 回退（`token` → `userId`），与 taskAuth 浏览器导航认证对齐；即使 nginx 未改也能恢复 `request.user`。
2. **Nginx 示例**：新增 `location ^~ /accounts/sso/` → gateway，使生产路径经 forward-auth 注入网关头。

### 测试

- `task2app/Saas_project/tests/test_gateway_auth_middleware_sso_cookie.py`

## 变更记录（2026-07-13 续 — 统一 /api/）

- 规范入口改为 `GET /api/accounts/sso/ai-provider/{admin,vendor}/`（经边缘 `/api/` → APISIX）。
- **清理**旧路径 `/accounts/sso/`（Django 不再挂载；nginx 特例 location 删除）。
- **移除** `GatewayAuthMiddleware` 的 cookie 路径兜底；仅认 forward-auth 头 / Session。
- APISIX `routes.yaml` 新增 `django-accounts-sso`（`/api/accounts/sso/*`，`auth_mode: token`）。
