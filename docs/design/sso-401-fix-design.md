# 设计文档：镜像市场管理（SSO）401 修复

**日期**: 2026-06-29
**类型**: Bug Fix
**严重级别**: P1 — 超级管理员无法访问镜像市场管理端

---

## 1. 问题描述

超级管理员在 `http://183.250.1.132:4000/system-admin/` 点击「镜像市场管理（SSO）」，
浏览器新标签页打开 `http://183.250.1.132:18081/accounts/sso/ai-provider/admin/`，
页面返回 **HTTP 401 Unauthorized**，body 为空。

## 2. Playwright 调查结果

### 2.1 登录流程

| 步骤 | 结果 |
|------|------|
| 访问 /system-admin/ | 302 → `/auth/login/` |
| POST /api/auth/ (邮箱+密码) | 200，返回 DRF Token |
| 登录后 cookie | `userId=850249621660790784` (domain=183.250.1.132, path=/, sameSite=Lax) |
| localStorage | `authToken=eecc68d36f7cdda41e4efabfa87beb3bfb960106` |

### 2.2 SSO 跳转链路

```
[用户点击] target="_blank" → http://183.250.1.132:18081/accounts/sso/ai-provider/admin/
                                     │
                                     ▼
                              [APISIX :18081]
                              route: django-default (uri: /*, auth_mode: token)
                                     │
                                     ▼
                              [forward-auth → taskAuth]
                              handleGatewayForwardAuth()
                              → 检查 Authorization: Token xxx
                              → ❌ 浏览器导航不携带自定义 Header
                              → 返回 401
                                     │
                                     ▼
                              [APISIX → 浏览器]
                              HTTP 401, body: <html><head></head><body></body></html>
                                     │
                              [Django view sso_ai_provider_admin_redirect]
                              ❌ 从未到达
```

### 2.3 认证机制对比

| | 主站 API 调用 (正常) | SSO 浏览器导航 (异常) |
|---|---|---|
| 认证方式 | `Authorization: Token xxx` (JS fetch header) | 浏览器默认请求头 |
| Cookie 携带 | 随 fetch 发送 | `userId` cookie (可能) |
| APISIX forward-auth | ✅ Token → user_id | ❌ 无 Token → 401 |

### 2.4 关键发现

1. **APISIX forward-auth 硬依赖 Token header**：`gateway_forward_auth.go` 第 17-20 行只检查 `tokenFromRequest(r)` (即 `Authorization: Token xxx`)
2. **`userId` cookie bridge 未被 forward-auth 使用**：此前为 OIDC SSO 添加的第 4 认证方式 (`resolveTokenUserIDFromRequest`) 只在 OIDC handlers 中使用，`handleGatewayForwardAuth` 仍使用旧的 `tokenFromRequest` + `resolveTokenUserID` 两段式调用
3. **Same-context fetch 也返回 401**：即使浏览器发送了 cookie，服务器也不认
4. **Django view 被网关挡住**：`sso_ai_provider_admin_redirect` 中的 `request.user.is_authenticated` 检查永远不会执行

## 3. 根因

**`handleGatewayForwardAuth` 没有复用 `resolveTokenUserIDFromRequest`**，后者已支持 4 层认证回退（Token header → token cookie → Bearer JWT → userId cookie），但 forward-auth handler 仍只用 Token header。

```go
// gateway_forward_auth.go — 当前代码（有问题）
tokenKey := tokenFromRequest(r)  // 只看 Authorization: Token xxx
if tokenKey == "" {
    w.WriteHeader(http.StatusUnauthorized)  // ← 浏览器导航没有这个 header
    return
}
```

应当使用：
```go
userID, err := resolveTokenUserIDFromRequest(r)  // 4 层回退，含 userId cookie
```

## 4. 修复方案

### 方案 A（推荐）：forward-auth 使用 `resolveTokenUserIDFromRequest`

修改 `taskAuth/src/gateway_forward_auth.go`，将：
```go
tokenKey := tokenFromRequest(r)
if tokenKey == "" {
    w.WriteHeader(http.StatusUnauthorized)
    return
}
userID, err := resolveTokenUserID(tokenKey)
```
替换为：
```go
userID, err := resolveTokenUserIDFromRequest(r)
```

**优点**：
- 最小改动（2 行变 1 行）
- 复用已验证的 4 层认证回退逻辑
- 修复所有 `auth_mode: token` 路由的 Cookie 认证问题（不仅 SSO）
- 无安全退化 — `userId` cookie 验证仍检查用户存在且活跃

**安全性分析**：
- `resolveTokenUserIDFromRequest` 的 `userId` cookie 检查会调用 `userIsActive(userID)` 验证用户状态
- `handleGatewayForwardAuth` 还额外调用 `loadUserAuthFlags(userID)` 检查组/管理员状态
- 两层验证确保安全

### 方案 B（备选）：为 `/accounts/sso/*` 添加 `auth_mode: none` 路由

在 `routes.yaml` 中新增专有路由，绕过 forward-auth，由 Django session 自行验证。

**缺点**：
- Django session 需要 `sessionid` cookie，当前 token-based login 不设置此 cookie
- 实际上不生效，仍会 `redirect → /auth/login/` → 死循环
- 未解决根本问题：其他需要 cookie 认证的页面也会遇到同样问题

### 方案 C（不推荐）：前端在 SSO URL 中传递 Token

将 `authToken` 拼接到 URL query 参数中。

**缺点**：
- Token 出现在 URL/日志中，安全风险
- 需要前端 + 后端 + 网关三处修改
- 仅为 SSO 单点修复

## 5. 价值流影响分析

### 影响的已有 Value Stream

| Stream | 影响 |
|--------|------|
| `user-auth` → `gateway-forward-auth-token-resolve` (planned) | 此 step 正是要解决网关头认证的 Token→Cookie 桥接。本修复使其实际生效 |
| `user-auth` → `frontend-auth-guard-redirect` | SSO 跨端口跳转依赖网关认证 |
| `system-admin-phone-login-recharge-policy` | 系统管理员页面组，SSO 是其中一环 |

### 测试影响

| 测试文件 | 操作 |
|----------|------|
| `taskAuth/src/gateway_forward_auth_test.go` | 新增：userId cookie 认证测试 |
| `playwright/.../SystemAdmin.sso-ai-provider-admin.playwright.test.js` | 新增：E2E 验证 SSO 跳转成功 |

### 字段影响

无数据库字段变更。

## 6. 域概念清单

| 概念 | 类型 | 所属上下文 |
|------|------|-----------|
| Gateway Forward-Auth | Domain Service | 用户与认证 |
| Token Resolution Chain | Domain Service | 用户与认证 |
| userId Cookie Bridge | Value Object | 用户与认证 |
| SSO Bridge JWT | Value Object | 用户与认证 |
| APISIX Route | Infrastructure | API 网关 |

## 7. 实施步骤

1. **修改 `taskAuth/src/gateway_forward_auth.go`**：使用 `resolveTokenUserIDFromRequest`
2. **编写 Go 单元测试**：userId cookie 作为唯一凭据时 forward-auth 返回 200
3. **编译 taskAuth** 二进制
4. **Playwright E2E 验证**：模拟完整登录→SSO 跳转→镜像市场页面
5. **回归测试**：确认 Token header 路径不受影响

---

*设计文档完成，待审批后进入实施。*
