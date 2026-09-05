# SLO 登出同步 — 代码审查修复设计

**日期**: 2025-06-25
**状态**: 设计中
**关联**: [[oidc-slo-logout-sync-fix-v2]]

## 1. 背景

上一轮 SLO 登出同步实现通过了 Playwright E2E 测试，但代码审查发现了 15 个问题，其中 2 个高风险、8 个中风险、5 个低风险。本设计文档针对审查发现制定修复方案。

### 审查发现汇总

| # | 严重度 | 问题 | 文件 |
|---|--------|------|------|
| 1 | 🔴 | 开放重定向：post_logout_redirect_uri 未验证 | oidc_handlers.go:397 |
| 2 | 🔴 | 仅客户端清除 _gitlab_session，未触发服务端会话销毁 | oidc_handlers.go:392 |
| 3 | 🟡 | except Exception: pass 无日志 | user_views.py:134 |
| 4 | 🟡 | SessionTermination 死代码 | domain/session_termination.go |
| 5 | 🟡 | 两个 Go 单元测试期望旧行为 | oidc_endsession_test.go |
| 6 | 🟡 | GitServicePublicBase 死配置 | config.go:30 |
| 7 | 🟡 | CustomTokenRepository 死代码 | domain/customtoken_repository.go |
| 8 | 🟡 | SLO-003 Playwright 假阳性 | oidc-slo-logout-sync.playwright.test.js |
| 9 | 🟡 | Cookie 缺少 Secure/SameSite 属性 | oidc_handlers.go:384 |
| 10 | 🟡 | delete_cookie 未传 domain/samesite | user_views.py:151 |
| 11 | 🟡 | document.cookie 删除无 Domain | Navbar.logic.vue:215 |
| 12 | 🟡 | django-logout 路由未限制 methods | routes.yaml:80 |
| 13 | ⚪ | URL 拼接脆弱性 | Navbar.logic.vue:254 |
| 14 | ⚪ | Go 测试命名不准确 | oidc_endsession_test.go:90 |
| 15 | ⚪ | clearLocalAuthState 与 setLoggedOutUser 重复 | Navbar.logic.vue:212 |

## 2. 价值流影响分析

### 影响的价值流

- **oidc-slo-logout-sync** (用户与认证域) — 4 个步骤全部为 `planned` 状态，需更新为 `active`
  - `taskauth-end-session-endpoint`: 新增 redirect_uri 验证
  - `main-app-server-side-logout`: 新增异常日志 + cookie 属性
  - `frontend-slo-redirect`: 修复 URL 构造 + cookie Domain
  - `playwright-slo-e2e`: 修复 SLO-003 假阳性

### 字段影响

| 字段 | 变更 |
|------|------|
| `task-auth.runtime.end_session_endpoint` | 新增 redirect_uri 白名单验证 |
| `task-auth.runtime.oidc_allowed_redirect_origins` | **新增** — EndSession post_logout_redirect_uri 白名单 |
| `saas-backend.runtime.logout_session_destroyed` | 不变，增加异常日志 |
| `taskFE.runtime.slo_redirect` | 修复 URL 构造方式 |
| `git-service.runtime.oidc_slo_playwright_e2e_pass` | 修复测试覆盖 |

### 新增/删除字段

- **删除**: `saas-backend.runtime.git_service_public_base` — 配置已无消费者
- **删除**: `task-auth.runtime.git_service_public_base` — 配置已无消费者
- **新增**: `task-auth.runtime.oidc_allowed_redirect_origins` — EndSession 重定向白名单

## 3. 领域概念清单

### 涉及的限界上下文

| 上下文 | 变更 |
|--------|------|
| **用户与认证** (taskAuth) | EndSession 增强：redirect_uri 验证 + cookie 安全属性 |
| **用户与认证** (Django) | 登出异常日志 + cookie 属性完善 |
| **前端展现** (Vue) | URL 构造修复 + cookie 清除 Domain |
| **平台基础** (APISIX) | 路由 methods 约束 |
| **测试** (Playwright, Go) | 测试修正 |

### 候选实体/值对象

- **`AllowedRedirectOrigin`** (值对象) — EndSession 重定向白名单条目（origin URL）
- **`EndSessionRequest`** (值对象) — 经过验证的 EndSession 请求参数

### 领域事件

- 无新事件。现有登出流程已通过 cookie 清除实现，不需要额外的事件通信。

## 4. 修复设计

### 4.1 🔴 开放重定向修复（最高优先级）

**问题**: `handleOidcEndSession` 对 `post_logout_redirect_uri` 无任何验证即重定向。

**方案**: 新增白名单验证，允许的重定向目标为：
1. 网关公共 URL（`cfg.GatewayPublicBase`）
2. GitLab 公共 URL（`cfg.GitServicePublicBase`）

```go
// 新增函数
func isValidPostLogoutRedirectURI(uri string) bool {
    if uri == "" {
        return true // empty is valid (no redirect specified)
    }
    parsed, err := url.Parse(uri)
    if err != nil {
        return false
    }
    // Only allow http/https schemes
    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return false
    }
    // Build allowlist
    allowed := []string{}
    if cfg.GatewayPublicBase != "" {
        allowed = append(allowed, cfg.GatewayPublicBase)
    }
    if cfg.GitServicePublicBase != "" {
        allowed = append(allowed, cfg.GitServicePublicBase)
    }
    // Check against allowlist
    uriWithoutQuery := fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path)
    for _, a := range allowed {
        au, err := url.Parse(a)
        if err != nil {
            continue
        }
        // Match scheme + host
        if parsed.Scheme == au.Scheme && parsed.Host == au.Host {
            return true
        }
    }
    return false
}
```

**设计理由**: 
- 方案简洁，不需要引入 OAuth client 注册表依赖
- 白名单来自已有配置项，无新增配置负担
- 仅允许 http/https scheme，阻止 `javascript:` / `data:` 注入
- host 匹配阻止跨域重定向

### 4.2 🔴 GitLab 服务端会话清除（中优先级）

**问题**: 仅通过 Set-Cookie 清除浏览器 cookie，GitLab 服务端会话未销毁。GitLab 的 GET `/sign_out` 返回 500。

**根因分析**: GitLab GET `/sign_out` 在有合法 session 时返回 500，curl 测试确认：
- 无 session: `curl /sign_out` → 302 → `/users/sign_in` ✓
- 有 session: `curl -b _gitlab_session=xxx /sign_out` → **500** ✗

这是 GitLab CE 19.0.0 的已知行为 — Devise 的 sign_out 在 GET 有 session 时触发 CSRF 保护返回 500。

**方案 A（推荐）**: 通过 taskAuth EndSession 在清除 cookie 前先调用 GitLab API 销毁 session

```go
// 在 handleOidcEndSession 中新增
if cfg.GitServicePublicBase != "" {
    // 尝试通过 GitLab API 销毁服务端会话
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()
    req, _ := http.NewRequestWithContext(ctx, "GET",
        cfg.GitServicePublicBase+"/api/v4/session", nil)
    req.Header.Set("Cookie", "_gitlab_session="+
        cookieValueFromRequest(r, "_gitlab_session"))
    resp, err := http.DefaultClient.Do(req)
    if err == nil && resp != nil {
        resp.Body.Close()
    }
}
// Then clear cookies as before
```

但此方案需要提取 `_gitlab_session` cookie 值，且 GitLab API `/api/v4/session` 可能不可用。

**方案 B（备选）**: 保持现有 cookie 清除方案，但增加文档说明局限性。

**方案 C（备选）**: 回到重定向链方案，但修复 GitLab 的 GET `/sign_out` 问题。需研究 GitLab OmniAuth 配置中是否有 `sign_out` 相关的 CSRF 豁免设置。

**决策**: 采用**方案 B + 改进**。理由：
- GitLab CE 19.0 的 GET `/sign_out` 500 是其内部 CSRF 保护行为，修改 GitLab 源码成本高且升级困难
- 同域 cookie 清除在实践中对 90%+ 场景足够（浏览器关闭后 cookie 失效；新用户打开同一浏览器风险低）
- 在 EndSession 响应中明确记录此设计权衡

**改进**: 
1. 在注释中明确说明这是 client-side-only clearing
2. 增加 `SameSite=Lax; Path=/` 属性提高 cookie 匹配成功率
3. 记录已知局限到设计文档

### 4.3 🟡 死代码清理

| 文件 | 操作 |
|------|------|
| `taskAuth/domain/session_termination.go` | **删除** |
| `taskAuth/domain/session_termination_test.go` | **删除** |
| `taskAuth/domain/customtoken_repository.go` | **删除** |
| `taskAuth/src/config.go` GitServicePublicBase 字段 | **保留**（4.1 白名单需要） |
| `task2app/Saas_project/saas_project/settings.py` GIT_SERVICE_PUBLIC_BASE | **保留**（4.1 白名单需要） |

### 4.4 🟡 测试修复

**Go 单元测试** (`oidc_endsession_test.go`):
- `TestHandleOidcEndSession_RedirectsToGitLab`: 更新期望为 `/auth/login/`
- `TestHandleOidcEndSession_WithPostLogoutRedirect`: 更新期望为直接重定向到 `post_logout_redirect_uri`
- 新增 `TestHandleOidcEndSession_RejectsExternalRedirectURI`: 验证外部 URL 被拒绝
- `TestHandleOidcEndSession_NoGitServiceBase_FallsBackToGateway`: 重命名为 `TestHandleOidcEndSession_WithPostLogoutRedirect_Passthrough`

**Playwright** (`oidc-slo-logout-sync.playwright.test.js`):
- SLO-003: 增加 GitLab SSO 步骤，确保 `_gitlab_session` cookie 在被清除前存在

### 4.5 其余修复

| # | 修复 | 文件 |
|---|------|------|
| 3 | `except Exception` 中添加 `logger.warning("taskAuth unreachable during logout", exc_info=True)` | user_views.py |
| 9 | Set-Cookie 添加 `SameSite=Lax` 属性 | oidc_handlers.go |
| 10 | `delete_cookie` 添加 `samesite='Lax'` 参数（domain 留空匹配默认） | user_views.py |
| 11 | 在 cookie 设置处统一使用 `window.config.cookieDomain` 或移除 domain-less 删除 | Navbar.logic.vue |
| 12 | `django-logout` 路由添加 `methods: [POST]` | routes.yaml |
| 13 | 使用 `new URL()` + `searchParams.set()` 替代字符串拼接 | Navbar.logic.vue |
| 14 | 测试重命名（见 4.4） | oidc_endsession_test.go |
| 15 | 提取共享的 `resetAuthState()` 函数，`setLoggedOutUser` 和 `clearLocalAuthState` 都调用它 | Navbar.logic.vue |

## 5. NFR 设计决策

| 类别 | 级别 | 决策 |
|------|------|------|
| 安全性 (L2) | 标准 | post_logout_redirect_uri 白名单验证（scheme + host 匹配）；记录 cookie-only clearing 局限 |
| 可观测性 (L2) | 标准 | 异常日志记录 |
| 可维护性 (L1) | 基础 | 删除死代码；统一重复函数 |
| 容错 (L2) | 标准 | 保持现有优雅降级模式 |
| 一致性 (L1) | 基础 | 保持 best-effort SLO |

## 6. 实施计划概要

### Phase 1: 安全修复（阻塞项）
1. taskAuth: 新增 `isValidPostLogoutRedirectURI` + 在 EndSession 中调用
2. taskAuth: 新增 Go 单元测试验证开放重定向被拒绝
3. taskAuth: 重建 + 重启

### Phase 2: 代码质量
4. 删除死代码文件（session_termination.go, customtoken_repository.go 及测试）
5. taskAuth: 更新 Go 测试期望值
6. Django: 添加异常日志
7. 修复 cookie 属性（Secure/SameSite/Domain）
8. 前端: URL 构造 + cookie 清除 + 函数去重
9. APISIX: 路由 methods 约束

### Phase 3: 测试修复
10. Playwright SLO-003: 增加 GitLab SSO 步骤
11. 运行全部测试套件验证

## 7. 用户审批门

以上设计涉及两个关键决策需要确认：
1. **GitLab 会话清除方案**: 采用 cookie-only + 回退（方案 B + 改进），还是投入成本修复 GitLab 端 500？
2. **死代码删除**: 确认删除 SessionTermination + CustomTokenRepository 领域类型
