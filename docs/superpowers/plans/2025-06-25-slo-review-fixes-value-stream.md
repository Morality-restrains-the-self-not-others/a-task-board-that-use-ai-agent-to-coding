# Value Stream: SLO 代码审查修复

> Derived from design: `docs/superpowers/specs/2025-06-25-slo-review-fixes-design.md`

## Value Summary

修复 OIDC SLO 登出同步的代码审查发现：关闭开放重定向安全漏洞，增强 GitLab 会话清除可靠性，清理死代码，修复测试覆盖缺口。

## Related Value Streams

- **oidc-slo-logout-sync** (2026-06-25): **modification** — 修复该价值流实现中的审查发现。原 4 个 `planned` 步骤更新为 `active`，新增 1 个安全加固步骤。
  - Increment 1 (EndSession): 新增 redirect_uri 白名单验证
  - Increment 2 (主应用登出): 新增异常日志 + cookie 属性完善
  - Increment 3 (前端 SLO): 修复 URL 构造 + cookie Domain 清除
  - Increment 4 (Playwright E2E): 修复 SLO-003 假阳性
  - **新增** Increment 5: 死代码清理 + 路由加固

## End-to-End Flow

```
[用户点击退出]
  → [主应用 POST /logout → Django 销毁 session + 记录异常日志]
  → [返回 slo_redirect_url → 前端用 URL API 安全构造 SLO URL]
  → [浏览器跳转 taskAuth EndSession]
  → [EndSession 验证 post_logout_redirect_uri 白名单]
  → [清除 userId + _gitlab_session cookies (SameSite=Lax)]
  → [302 重定向至白名单内的登录页]
  → [拒绝外部 URL → 回退至默认登录页]
```

## Value Increments

### Increment 1: EndSession 安全加固（🔴 阻塞项）
**Value to user:** 防止开放重定向攻击，确保 SLO 重定向仅指向信任的内部服务
**Scope:**
- `handleOidcEndSession` 新增 `isValidPostLogoutRedirectURI` 白名单验证
- 白名单来源：`cfg.GatewayPublicBase` + `cfg.GitServicePublicBase`
- 仅允许 http/https scheme，拒绝 `javascript:` / `data:` 注入
- host 匹配拒绝跨域重定向
- 新增 Go 单元测试：`TestHandleOidcEndSession_RejectsExternalRedirectURI`
- 更新旧测试期望值匹配新行为
**Depends on:** oidc-slo-logout-sync Increment 1

### Increment 2: Cookie 安全属性 + 异常日志（🟡 中风险）
**Value to user:** 提高 SLO cookie 清除的跨浏览器兼容性和运维可观测性
**Scope:**
- taskAuth EndSession: Set-Cookie 新增 `SameSite=Lax` 属性
- Django logout: `except Exception` 新增 `logger.warning(..., exc_info=True)`
- Django logout: `delete_cookie` 新增 `samesite='Lax'` 参数
- 前端 `document.cookie` 清除使用一致的 Domain 设置
**Depends on:** Increment 1

### Increment 3: 代码质量 — 死代码清理 + 去重 + URL 修复（🟡+⚪）
**Value to user:** 减少维护负担，消除误导性配置和接口
**Scope:**
- 删除 `taskAuth/domain/session_termination.go` + `_test.go`（52+行死代码）
- 删除 `taskAuth/domain/customtoken_repository.go`（未实现的接口）
- 前端：提取共享 `resetAuthState()`，消除 `clearLocalAuthState`/`setLoggedOutUser` 重复
- 前端：URL 构造改用 `new URL()` + `searchParams.set()` 替代字符串拼接
- APISIX: `django-logout` 路由增加 `methods: [POST]`
**Depends on:** Increment 2

### Increment 4: 测试修复（🟡）
**Value to user:** 确保回归测试覆盖真实 SLO 路径
**Scope:**
- Playwright SLO-003: 新增 GitLab SSO 步骤，确保 `_gitlab_session` 在清除前已存在
- Go 测试 `TestHandleOidcEndSession_NoGitServiceBase_FallsBackToGateway` 重命名
- 更新 `TestHandleOidcEndSession_RedirectsToGitLab` 期望值
- 更新 `TestHandleOidcEndSession_WithPostLogoutRedirect` 期望值
**Depends on:** Increment 3
