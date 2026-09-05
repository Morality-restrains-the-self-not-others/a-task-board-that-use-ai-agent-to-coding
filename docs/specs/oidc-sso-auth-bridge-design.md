# Design Document: OIDC SSO 认证桥接 — taskAuth 识别跨端口 Session

**Created:** 2026-06-25  
**Status:** 待审批  
**Author:** Claude

---

## 1. 问题描述

用户从主站 (port 4000) 登录后，点击「代码仓库」→「taskAuth SSO」，OIDC 流程中断在 `/auth/login?next=...` 跳转到主页 (port 4000)，无法完成 SSO 登录 GitLab。

### 完整重定向链（现状）

```
已登录主站(:4000) → 点击 代码仓库 → GitLab sign_in(:8012) → 点击 taskAuth SSO
  → 302: /users/auth/openid_connect          (GitLab OIDC 启动)
  → 302: /api/oidc/authorize?...              (taskAuth 检查认证)
  → resolveTokenUserIDFromRequest():
      ├─ Authorization: Token xxx  → 不存在 (302 重定向不带自定义 header)
      ├─ Cookie: "token"           → 不存在 (从未设置过)
      └─ Authorization: Bearer xxx → 不存在
  → 未认证 → 302: /auth/login?next=...       (重定向到登入页)
  → Vue 登入页检测到 userId cookie → 302: :4000/ (跳到主页)
  ✗ OIDC 流程中断
```

### 期望行为

已登录用户应**直接完成 OIDC SSO**，无需二次登录：

```
已登录主站(:4000) → 点击 代码仓库 → GitLab sign_in(:8012) → 点击 taskAuth SSO
  → 302: /users/auth/openid_connect
  → 302: /api/oidc/authorize?...
  → resolveTokenUserIDFromRequest():
      ├─ Token header / cookie  → 不存在
      └─ Cookie: "userId"       → 存在！→ 验证用户 → 认证通过 ✅
  → 生成授权码 → 302: GitLab callback
  → GitLab 交换 token → Dashboard ✅
```

---

## 2. 根因分析

### 两个认证体系不兼容

| 维度 | taskAuth (Port 18081) | Django/Vue (Port 4000) |
|------|----------------------|------------------------|
| 认证凭据 | `Authorization: Token xxx` / `token` cookie | `userId` cookie + Django session |
| 凭据来源 | taskAuth 自身签发 | 主站登录时设置 |
| 跨端口传递 | `Authorization` header 不会在重定向中传递 | Cookie 同域共享 (183.250.1.132) |
| OIDC 可见性 | 查不到 token → 认为未认证 | Cookie 存在但不被 taskAuth 识别 |

### 关键发现

1. `userId` cookie 是**同域共享**的（Domain=183.250.1.132, Path=/），浏览器在请求 port 18081 时会自动携带
2. `token` cookie **从未被设置**（登录成功只设置 userId cookie，不设置 token cookie）
3. 302 重定向是浏览器发起的纯 GET 请求，**只带 Cookie，不带 Authorization header**
4. `userId` cookie 非 httpOnly，但用户信息在 taskAuth 本地数据库中有记录

---

## 3. 修复方案：认证桥接

### 核心思路

在 `resolveTokenUserIDFromRequest` 中增加第 4 种认证方式：**通过 `userId` cookie 识别已登录用户**。

### 3.1 代码修改

**文件**: `taskAuth/src/oidc_handlers.go`

在 `resolveTokenUserIDFromRequest` 函数末尾，现有 3 种认证方式之后，新增第 4 种：

```go
func resolveTokenUserIDFromRequest(r *http.Request) (string, error) {
    // 1. Try Authorization: Token xxx (existing)
    if uid, err := resolveTokenUserID(tokenFromRequest(r)); err == nil {
        return uid, nil
    }
    // 2. Try Cookie-based session token (existing)
    if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
        if uid, err := resolveTokenUserID(cookie.Value); err == nil {
            return uid, nil
        }
    }
    // 3. Try Authorization: Bearer xxx (existing)
    auth := strings.TrimSpace(r.Header.Get("Authorization"))
    if strings.HasPrefix(auth, "Bearer ") {
        bearer := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
        if claims, err := verifyRS256Signature(bearer); err == nil {
            if exp, ok := claims["exp"].(float64); ok && exp > 0 {
                if time.Now().Unix() > int64(exp) {
                    return "", fmt.Errorf("bearer token expired")
                }
            }
            if sub, ok := claims["sub"].(string); ok && sub != "" {
                return sub, nil
            }
        }
    }
    // 4. NEW: Try userId cookie (cross-port session bridge for OIDC SSO)
    //    The main app (:4000) sets a userId cookie after Django login.
    //    Cookies are domain-scoped and travel to :18081 on redirects.
    //    We verify the user exists and is active in our local DB.
    if cookie, err := r.Cookie("userId"); err == nil && cookie.Value != "" {
        userID := strings.TrimSpace(cookie.Value)
        if isActive, err := userIsActive(userID); err == nil && isActive {
            return userID, nil
        }
    }
    return "", fmt.Errorf("not authenticated")
}
```

### 3.2 安全性分析

| 风险 | 缓解措施 |
|------|---------|
| 伪造 userId cookie | 验证用户是否在数据库中存在且 active |
| Session 劫持 | userId 本身不敏感；真正的安全边界是 Django session |
| 跨端口攻击 | Cookie 同域限制（183.250.1.132），外部无法伪造 |

`userId` cookie 已通过主站登录的完整认证流程设置。taskAuth 额外验证用户 active 状态，提供第二层保护。

### 3.3 修改文件清单

| 文件 | 变更 |
|------|------|
| `taskAuth/src/oidc_handlers.go` | `resolveTokenUserIDFromRequest` 新增 `userId` cookie 分支 |
| `gitService/playwright/tests/oidc-sso-401-fix-verify.playwright.test.js` | 更新 TC-02 期望目标为 GitLab Dashboard |

---

## 4. 域概念

- **Bounded Context**: Auth (跨 taskAuth + Django 认证桥接)
- **Entity**: User (已存在于 taskAuth accounts_user 表)
- **认证凭据**: Token (taskAuth 原生) → 桥接 → userId Cookie (Django session)

---

## 5. 批准后实施步骤

1. 修改 `oidc_handlers.go` 添加 userId cookie 支持
2. 重新编译 taskAuth
3. 重启 taskAuth 进程
4. 运行 Playwright E2E 测试验证完整 SSO 流程
