# Value Stream: Login Redirect to Owner Company

> Derived from design: `/debug` session analysis — 403 on wrong tenant

## Value Summary

用户登录后自动跳转到自己创建的公司（creator_id），而非任意第一个成员公司，避免误入无权限租户导致 403。

## Related Value Streams

- **2026-06-04-enrich-login-fallback-value-stream**: extension — 同一 `_login_redirect_url` 函数，本次修改其公司匹配逻辑
- **2026-06-23-login-redirect-loop-gateway-auth-fix-value-stream**: related — 同一登录重定向链路

## End-to-End Flow

[用户登录] → [taskAuth 验证凭证] → [Django enrich-login 组装响应] → [_login_redirect_url 确定目标公司] → [前端跳转] → [用户看到自己的项目列表]

## Value Increments

### Increment 1: 修复 _login_redirect_url 优先使用 creator_id（Thin Slice）

**Value to user:** 登录后始终进入自己创建的公司，不会误入无权限租户

**Scope:**
- 修改 `accounts/taskauth_internal_views.py:_login_redirect_url`
- 优先 `Company.objects.filter(creator_id=user_id).first()`
- 兜底 `CompanyMember.objects.filter(user_id=user_id).first()`
- 返回 `/tenant/<company_id>/projects/` 带租户前缀

**Depends on:** nothing

### Increment 2: 前端 403 容错重定向

**Value to user:** 即使误入无权限租户页面，前端自动跳回用户自己的公司

**Scope:**
- 修改 `Projects.vue:loadProjects` 增加 403 处理
- 403 时获取用户实际租户并 `router.replace`

**Depends on:** Increment 1
