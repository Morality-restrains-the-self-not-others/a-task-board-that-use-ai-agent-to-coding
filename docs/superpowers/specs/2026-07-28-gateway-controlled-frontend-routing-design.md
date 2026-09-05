# Design: Gateway-Controlled Frontend Routing (taskGateway 统管所有路径)

- **状态**: 🎯 target（待审批）
- **作者**: claude
- **日期**: 2026-07-28
- **版本**: v55
- **类型**: 架构重构 — 路由层

## 问题陈述

### 现状

当前 `www.daydaymoney.com` 的流量分流在 **Nginx 层** 完成：

```
Nginx (:443)
├── location /api/ → APISIX (:18081)  # API 流量走 Gateway
└── location /     → Django (:8001)   # SPA 页面直接打 Django
```

| 路径 | 当前路由 | 问题 |
|------|---------|------|
| `/auth/login` | Nginx `/` → Django → `login` 视图 | Django 需要 `_serve_spa_shell()` 间接提供 SPA HTML |
| `/auth/register/` | Nginx `/` → Django → `spa_catch_all` | Django 需要 `_serve_spa_shell()` 间接提供 SPA HTML |
| `/projects/` | Nginx `/` → Django → `spa_catch_all` | Django 承担了不属于它的职责 |
| `/*` (所有非 API) | Nginx `/` → Django → `spa_catch_all` | taskGateway 不掌握完整路由拓扑 |

**核心问题**: taskGateway（APISIX）只掌管 `/api/*` 路径，前端页面渲染路径被 Nginx 直接转发到 Django。Django 是 API 服务器，不应该承担 SPA 页面渲染职责。

### 期望

> taskGateway 应该掌握所有路径。前端渲染页面直接经过 taskGateway 后打到 taskFE。

## 目标架构

```
                         Internet (HTTPS)
                              │
                    ┌─────────▼─────────┐
                    │  Nginx (:443)     │
                    │  daydaymoney.com    │
                    │                   │
                    │  location /       │
                    │    → APISIX       │  ← 所有流量统一进入 Gateway
                    └─────────┬─────────┘
                              │
                    ┌─────────▼─────────┐
                    │  APISIX (:18081)  │  ← taskGateway (Docker)
                    │                   │
                    │  Routes:          │
                    │  /api/* → Go svc  │  (unchanged)
                    │  /gateway/* → docs│  (unchanged)
                    │  .well-known → auth│ (unchanged)
                    │  /* → taskFE      │  ← NEW: SPA catch-all
                    └───┬───────┬───────┘
                        │       │
              ┌─────────▼──┐ ┌──▼──────────┐
              │ Go Services│ │  taskFE     │
              │ :8001-8025 │ │  :3000      │ ← NEW upstream
              └────────────┘ └─────────────┘
```

### 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| taskFE 上游 | Vite dev (:3000) / Nginx static (prod) | dev 复用已有 Vite 进程；prod 需新增 Nginx 容器 |
| SPA catch-all auth | `auth_mode: none` | SPA 页面通过 Vue Router `beforeEach` + API forward-auth 双层鉴权 |
| 保留 auth-login-page 路由? | ❌ 移除 | 被 SPA catch-all 覆盖 |
| 保留 django-default (priority 0)? | ❌ 替换为 spa-catch-all | 语义明确 |
| Django `_serve_spa_shell()` | ❌ 移除 | Django 不再承担 SPA 渲染职责 |

## 详细变更

### 1. Nginx 配置 (`daydaymoney.sh.nginx.active.conf`)

```diff
-    location / {
-        proxy_pass http://daydaymoney_spa;       # :8001 = Django
+    location / {
+        proxy_pass http://daydaymoney_gateway;    # :18081 = APISIX
```

> ⚠️ `api.daydaymoney.com` 的 `location /` 已经指向 APISIX，无需变更。

### 2. APISIX 路由 (`taskGateway/routes/routes.yaml`)

#### 新增 upstream

```yaml
  taskFE:
    host: 127.0.0.1
    port: 3000          # Vite dev server; prod 改为 Nginx static server
    docs: false
```

#### 新增 SPA catch-all 路由

```yaml
  # SPA shell catch-all — 所有非 API GET/HEAD → taskFE
  # Vue Router 处理客户端路由与认证
  - id: spa-catch-all
    priority: 10
    uri: /*
    methods: [GET, HEAD]
    upstream: taskFE
    auth_mode: none
```

#### 移除路由

```yaml
  # REMOVE: auth-login-page (priority 800) — 被 spa-catch-all 覆盖
  # REMOVE: django-default (priority 0) — 替换为 spa-catch-all
```

### 3. Django 简化 (`auth_views.py`)

移除 `_find_spa_index()`, `_serve_spa_shell()` 函数，`spa_catch_all` 回归返回 JSON 405：

```python
def spa_catch_all(request, path=None):
    """兜底视图：Django 仅处理 API 路径。非 API 路径由 Gateway → taskFE 处理。"""
    if request.method in ('GET', 'HEAD'):
        return JsonResponse({
            'detail': 'This path is handled by the frontend via Gateway. '
                      'If you see this, the Gateway route may be misconfigured.',
            'path': request.path,
        }, status=404)
    return JsonResponse({
        'detail': 'Method not allowed for non-API path',
        'path': request.path,
    }, status=405)
```

`login` 视图 GET 处理器恢复返回 JSON：

```python
def login(request):
    if request.method == 'GET':
        frontend_domain = settings_manager.get_frontend_domain()
        if request.user.is_authenticated:
            # 保留 OIDC resume 流程
            oidc_resume = sanitize_oidc_resume_next(request.GET.get('next'))
            if oidc_resume:
                target = resolve_oidc_resume_target(oidc_resume)
                response = HttpResponse(status=302)
                response['Location'] = target
                _attach_sso_bridge_cookies(response, str(request.user.pk), request)
                return response
            redirect_url = _default_frontend_redirect_url(request.user, frontend_domain)
            response = HttpResponse(status=302)
            response['Location'] = redirect_url
            _attach_sso_bridge_cookies(response, str(request.user.pk), request)
            return response
        return JsonResponse({
            'status': 'unauthenticated',
            'redirect_url': f'{frontend_domain}/auth/login/',
        })
```

### 4. taskFE 静态文件服务（生产环境）

新增一个轻量 Nginx 容器挂载 `taskFE/static/` 目录：

```yaml
# taskGateway/docker-compose.yml 或独立 compose
taskFE:
  image: nginx:1.27-alpine
  volumes:
    - ../../taskFE/static:/usr/share/nginx/html:ro
    - ./taskfe-nginx.conf:/etc/nginx/conf.d/default.conf:ro
  ports:
    - "127.0.0.1:3000:80"
```

```nginx
# taskfe-nginx.conf
server {
    listen 80;
    root /usr/share/nginx/html;
    location / {
        try_files $uri $uri/ /index.html;  # SPA fallback
    }
}
```

> 💡 开发环境直接复用 Vite dev server (:3000)，无需额外容器。

## 请求流对比

### Before (current)

```
GET /auth/login
  → Nginx location / → Django :8001
  → Django spa_catch_all → _serve_spa_shell()
  → 返回 taskFE/static/assets/index.html
```

### After (target)

```
GET /auth/login
  → Nginx location / → APISIX :18081
  → APISIX spa-catch-all (priority 10) → taskFE :3000
  → taskFE Nginx serve index.html (SPA fallback)
  → Vue Router 匹配 /auth/login/ → Login.vue
```

```
POST /api/auth/
  → Nginx location / → APISIX :18081
  → APISIX taskauth-login (priority 850) → taskAuth :8003
  → taskAuth 验证凭据 → 返回 token
```

## 架构变更影响

### 组件变更

| 变更 | 组件 | 说明 |
|------|------|------|
| 🟢 NEW | `taskFE` (upstream) | APISIX 新增 taskFE 上游 |
| 🟢 NEW | `taskFE-static` (container) | 生产环境 Nginx 容器提供 SPA 静态文件 |
| 🟡 MODIFIED | Nginx config | `location /` 从 Django:8001 → APISIX:18081 |
| 🟡 MODIFIED | APISIX routes.yaml | 新增 spa-catch-all，移除 auth-login-page + django-default |
| 🟡 MODIFIED | Django auth_views.py | 移除 `_serve_spa_shell()`，spa_catch_all 回归 JSON |
| 🔴 DEPRECATED | Django SPA shell serving | Django 不再承担页面渲染 |

### 回滚路径

1. 恢复 Nginx `location /` → `daydaymoney_spa` (Django:8001)
2. 恢复 APISIX routes.yaml（移除 spa-catch-all，恢复 auth-login-page + django-default）
3. 恢复 Django `_serve_spa_shell()` 函数

## 业务意图 → 事件对照

本次变更不涉及新业务意图，无新增领域事件。

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| — | — | — | — | 纯基础设施重构，无业务状态变更 |

## 价值流影响

- **受影响流**: 无新增/修改 value stream，纯路由层重构
- **测试影响**: APISIX routes 单元测试需更新（移除两条路由，新增一条）；Nginx 配置需 reload 验证；Playwright E2E 登录流应仍通过

## 🐍 Python 新增接口清单

本次变更**不新增** Python 接口。相反，Django 的 `spa_catch_all` 和 `login` GET 处理器从"提供 SPA shell"简化为"返回 JSON 状态"，减少 Django 的职责范围。

## 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| taskFE 不可达时所有页面 502 | 高 | taskFE 容器 health check + 快速回滚 |
| SPA catch-all `auth_mode: none` 暴露页面源码 | 低 | API 层仍有 forward-auth 保护；页面源码非敏感 |
| OIDC SSO 流程断裂 | 中 | `login` POST + `next` 参数由 SPA 处理，已验证前端支持 |
| 生产构建未就绪 | 中 | `taskFE/static/assets/index.html` 须在部署前构建 |
