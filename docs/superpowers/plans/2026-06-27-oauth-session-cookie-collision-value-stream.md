# Value Stream: OAuth 授权 Session Cookie 冲突修复

> Derived from design: `docs/specs/oauth-redirect-loop-port-4000/design.md`

## Value Summary

用户从项目详情页发起 GitLab OAuth 授权并完成后，能正常回到发起页面（而非被踢回登录页），session cookie 不再因跨服务同名冲突而丢失认证状态。

## Related Value Streams

- **create-project-oauth-validation-loop** (`conf/value-stream.yaml`): **extension** — 本修复确保 OAuth 回调后认证状态保持，使 OAuth 验证闭环能正常工作。修改 gitOauth 的 session cookie 命名，不影响现有 OAuth 业务逻辑。
- **login-redirect-loop-gateway-auth-fix** (`docs/superpowers/plans/2026-06-23-...`): **related** — 同为认证状态传播问题，该修复解决 Gateway forward-auth secret 不匹配，本修复解决同域名下 Django session cookie 名称冲突。
- **session-userid-cookie-resilience** (`docs/superpowers/plans/2026-05-28-...`): **related** — Cookie 跨端口传播的通用方案，本修复补充了 Cookie 命名维度的隔离。

## End-to-End Flow

```
用户已登录 port 4000 (sessionid = <主站 DB key>)
  → 点击 OAuth 授权 → API 调用成功 → 跳转 gitOauth
  → gitOauth Set-Cookie: gitoauth_sessionid (修复后，不再覆盖 sessionid)
  → GitLab 授权完成 → gitOauth 回调 → 302 回 port 4000 项目页
  → Router guard → /api/.../profile/ → sessionid 仍为主站 DB session ✅
  → 用户回到项目页，OAuth 绑定成功
```

## Value Increments

### Increment 1: Session Cookie 命名隔离 (Thin Slice — 完整修复)

**Value to user:** OAuth 授权完成后正常回到发起页面，无需重新登录。

**Scope:**
1. 配置：`gitOauth/config/settings.py` — 添加 `SESSION_COOKIE_NAME = 'gitoauth_sessionid'`
2. 验证：gitOauth 现有测试确认 session 机制仍然正常（cookie 名变更不影响功能）

**Depends on:** nothing (单行配置修复)
