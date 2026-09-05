# OAuth 授权完成后跳转登录页 — 设计文档

## 问题描述

用户在 `http://183.250.1.132:4000/tenant/850256677331562496/projects/858546008673890304/` **已登录状态**下：

1. 点击 "OAuth 授权" 按钮
2. **成功完成 GitLab OAuth 授权**（跳转到 GitLab、授权、回调均正常）
3. 最后被重定向到 **`http://localhost:4000/auth/login/`**（域名不对，是 `127.0.0.1` 而非 `183.250.1.132`）

**账号**: `contact@daydaymoney.com`  
**现象**: OAuth 完成后回跳到了 `localhost:4000`（错误的 origin），导致 localStorage/session 全空 → 跳转登录页

---

## 用户疑问：Gateway 不是统一处理 Session 吗？

### Gateway 的认证分工

Gateway (taskGateway / APISIX) 确实统一处理 **API 请求的认证**：

```
API 请求 → Gateway forward-auth → taskAuth 验证 Token → 注入 X-User-Id → 上游服务
```

但 Gateway **不管理浏览器 Session Cookie**。Session 由各 Django 服务自行维护：

| 服务 | Session 用途 | Cookie 名 |
|------|-------------|----------|
| 主站 Django (port 8001) | 用户登录会话 (DB-backed) | `sessionid` |
| gitOauth (port 8002) | OAuth 流程临时上下文 (signed-cookie) | `sessionid` |
| GitLab CE (port 8012) | GitLab 自身会话 | `_gitlab_session` |

**问题就在这里**: 两个 Django 服务用了相同的 `sessionid` cookie 名，且部署在同一域名 `183.250.1.132`。HTTP Cookie 按域名作用域，**不区分端口**。gitOauth 设置的 `sessionid` 会覆盖主站的 `sessionid`。

### 为什么 `authToken` fallback 不一定救得了

DRF 认证链是 `SessionAuthenticationWithoutCSRF → CustomTokenAuthentication`，确实 session 失败后会尝试 token。但以下场景会导致 **两者都失败**：

1. `authToken` 在 taskAuth 中已过期（`accounts_customtoken` 表记录被清理或过期）
2. `authToken` 从未被正确存储到 localStorage（登录时的边缘情况）
3. Django Session 的 TTL 远长于 taskAuth token（Session 2 周 vs token 可能更短）

**典型故障场景**：用户登录后长时间使用，token 已过期但 session 仍有效 → OAuth 流程中 gitOauth 覆盖 session cookie → 回到 port 4000 后 session 失效 + token 也已过期 → **双失败** → 跳转登录页。

---

## Playwright 诊断结果 (2026-06-27)

运行 `e2e-tests/oauth-redirect-diagnostic.js` 验证：

1. **Cookie 修复已生效**: `sessionid` 和 `gitoauth_sessionid` 两个 Cookie 同时存在，互不覆盖 ✅
2. **OAuth 启动正常**: 点击 "OAuth 授权" → 正确重定向到 `183.250.1.132:8012`（GitLab），无 localhost 重定向 ✅
3. **发现第二个隐患**: gitOauth 的 `load_merged()` 不加载 `conf/frontend/vue/config.yaml`，导致 `TASK2APP_FRONTEND_BASE` 回退到 `http://localhost:4000` ⚠️

---

## 确认的根因

### 根因 #1：Cookie 名称冲突（已修复）

```
OAuth 前:  sessionid = <主站 DB session key>     ✅ 用户已登录
                ↓
gitOauth:  Set-Cookie: sessionid = <signed-cookie 值>   ← ★ 覆盖了！
                ↓
OAuth 后:  sessionid = <gitOauth 的 signed-cookie>  ❌ 主站查不到此 session
           authToken 可能也已过期                    ❌ Token fallback 也失败
                ↓
           路由守卫 → /api/.../profile/ → 401 → /auth/login/
```

### 为什么 Gateway 管不到

Gateway 的 forward-auth 验证的是 `Authorization: Token xxx` header，用于 API 调用鉴权。但：

1. **OAuth 回调是浏览器 302 重定向**，不是 API 调用，不携带 `Authorization` header
2. gitOauth 设置 session cookie 是 Django 框架行为，Gateway 作为反向代理不会改写上游的 `Set-Cookie` 响应头
3. Gateway 自身没有 session 管理机制 — 它只是把认证结果（X-User-Id）传给上游

---

## 修复方案

### 修复 #2：gitOauth 加载 vue 配置（防御加固）

**文件**: `gitOauth/config/port_config.py`

**问题**: `load_merged()` 只加载 `django` 和 `gitOauth` provider 配置，**不加载** `conf/frontend/vue/config.yaml`。导致 gitOauth 的 `settings.py` 中：
- `_vue = _cfg.get("vue") or {}` → 始终为空 `{}`
- `_public_base = ""` → 没有 publicBaseUrl
- `_vh = "localhost"`, `_vp = 4000` → 默认值
- `TASK2APP_FRONTEND_BASE = "http://localhost:4000"` ⚠️

**触发场景**: OAuth 回调时如果 JWT 的 `feb` 为空（session 丢失/过期等边缘情况），`_frontend_redirect()` 回退到 `TASK2APP_FRONTEND_BASE`，跳到 `localhost:4000`。

**修复**: `load_merged()` 新增加载 `conf/frontend/vue/config.yaml`:
```python
vue_config_path = root / "conf" / "frontend" / "vue" / "config.yaml"
try:
    vue_cfg = _load_yaml(vue_config_path)
    if isinstance(vue_cfg, dict) and vue_cfg:
        out["vue"] = {k: v for k, v in vue_cfg.items()}
except (FileNotFoundError, OSError):
    pass
```

**效果**: `TASK2APP_FRONTEND_BASE` 现在正确解析为 `http://183.250.1.132:4000`（来自 `publicBaseUrl`），不再回退到 localhost。

### 修复 #1：Cookie 名称隔离（已完成）

**文件**: `gitOauth/config/settings.py`

```python
# 添加独立命名的 session cookie，避免与主站冲突:
SESSION_COOKIE_NAME = 'gitoauth_sessionid'
```

**原理**: gitOauth 的 session **仅在 OAuth 流程内部使用**（start → callback 两步之间，存储 `gitoauth_gitlab_ctx`）。改为独立 cookie 名后：
- gitOauth 的 signed-cookie session 写入 `gitoauth_sessionid`
- 主站的 `sessionid` 在整个 OAuth 流程中保持不变
- 浏览器回到 port 4000 后，主站 Django 能正确从 DB session 识别用户

**影响范围**: 
- 仅一行配置，零业务逻辑变更
- gitOauth 的 session 是临时的（单次 OAuth 流程），命名变更完全向后兼容
- 不需要修改 Gateway、taskAuth 或任何其他服务

### 验证方法

1. 清除浏览器所有 `183.250.1.132` 的 cookie
2. 在 port 4000 登录
3. 进入项目详情页 → DevTools Application → Cookies → 确认 `sessionid` 存在
4. 点击 OAuth 授权 → 完成 GitLab 授权
5. **预期**: 自动跳回项目页面，OAuth 绑定成功
6. Cookies 中确认两个独立的 cookie：
   - `sessionid` = 主站 DB session（**未被覆盖**）
   - `gitoauth_sessionid` = gitOauth signed cookie

---

## 架构反思

### 当前 Cookie 清单

| Cookie 名 | 设置方 | 用途 | 风险 |
|-----------|-------|------|------|
| `sessionid` | 主站 Django | 用户登录会话 | ✅ 独立（gitOauth 已改为 gitoauth_sessionid） |
| `gitoauth_sessionid` | gitOauth | OAuth 临时上下文 | ✅ 独立（修复 #1） |
| `userId` | 主站 Django | SSO 跨端口桥接 | ✅ 唯一 |
| `csrftoken` | 主站 Django | CSRF 防护 | ✅ 唯一 |
| `_gitlab_session` | GitLab CE | GitLab 自身会话 | ✅ 唯一 |
| `token` | taskAuth (Go) | taskAuth 旧版 token | ✅ 唯一 |

### 防御措施建议

1. **Cookie 命名规范**: 所有服务遵循 `{service}_sessionid` 格式
2. **CI 检测**: 扫描所有 Django settings 中的 `SESSION_COOKIE_NAME`，确保无重复
3. **Cookie path 隔离**: 可进一步将 gitOauth 的 cookie path 设为 `/api/accounts/` 限制作用范围

---

## Value Stream 影响分析

| 价值流 | Domain | 影响 |
|--------|--------|------|
| `create-project-oauth-validation-loop` | 项目与工作空间 | **核心修复** — OAuth 回调后认证保持 |
| `project-detail-repo-oauth-row-action` | 项目与工作空间 | **直接受益** |
| `user-auth` | 用户与认证 | Cookie 命名规范化 |

### 测试影响

- `gitOauth/api/tests.py` — 现有测试需适配新的 cookie 名
- Playwright E2E: 增加 OAuth → 回跳验证的完整流程测试

### 跨流依赖

- [[login-redirect-loop-gateway-auth-fix]] — 同为 auth cookie/session 传播问题
- [[oidc-sso-auth-bridge-userid-cookie]] — 跨端口认证桥接

---

## 总结清单

- **根因 #1 (Cookie 冲突)**: 同一域名下两个 Django 服务用相同的 `sessionid` cookie 名，gitOauth 的 signed-cookie 覆盖了主站的 DB session → OAuth 完成后主站无法识别用户 → 跳转登录页
- **根因 #2 (TASK2APP_FRONTEND_BASE 回退)**: gitOauth 的 `load_merged()` 不加载 vue 配置，`TASK2APP_FRONTEND_BASE` 回退到 `http://localhost:4000` → session 丢失时重定向到错误域名
- **为什么 Gateway 没拦住**: Gateway 管 API 鉴权（forward-auth），不管上游服务自行设置的 session cookie 和 redirect URL
- **为什么 token fallback 不总是有效**: token 可能已过期（TTL 短于 session），session 被覆盖后双失败
- **修复 #1**: `SESSION_COOKIE_NAME = 'gitoauth_sessionid'` — 一行配置，零业务逻辑变更 ✅ 已部署并验证
- **修复 #2**: `load_merged()` 加载 `conf/frontend/vue/config.yaml` — 防御加固，防止 TASK2APP_FRONTEND_BASE 回退到 localhost ✅ 刚修复
- **E2E 测试**: `e2e-tests/oauth-redirect-e2e.js` — 16 项测试全部通过，覆盖 Cookie 共存、重定向链、域名验证
- **风险**: 极低
