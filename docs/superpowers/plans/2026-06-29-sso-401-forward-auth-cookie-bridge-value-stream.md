# Value Stream: SSO 401 Fix — Forward-Auth Cookie Bridge

> Derived from design: `docs/design/sso-401-fix-design.md`
> Date: 2026-06-29
> Type: Bug Fix (completes planned step)

## Value Summary

超级管理员点击「镜像市场管理（SSO）」能正常跳转到 Ai Provider 管理后台，而非看到 401 错误。

## Related Value Streams

- **`user-auth` → `gateway-forward-auth-token-resolve`** (planned → active): 此修复使该 planned step 成为 active。前次 OIDC SSO 修复为 `resolveTokenUserIDFromRequest` 添加了 `userId` cookie 第4认证方式，但 `handleGatewayForwardAuth` 未同步使用，导致网关级 forward-auth 仍只认 Token header。

## End-to-End Flow

```
[管理员点击 SSO 链接] → [APISIX :18081 forward-auth]
  → [taskAuth: resolveTokenUserIDFromRequest(r)]
  → [4层回退: Token header → token cookie → Bearer JWT → userId cookie ✓]
  → [200 + X-User-Id header] → [Django sso_ai_provider_admin_redirect]
  → [生成 bridge JWT] → [302 → Ai Provider admin]
```

## Value Increments

### Increment 1: Forward-Auth userId Cookie Fallback (Single Increment)

**Value to user:** 管理员通过浏览器点击 SSO 链接即可访问 Ai Provider 管理后台，无需额外登录。
**Scope:** 修改 `handleGatewayForwardAuth` 使用 `resolveTokenUserIDFromRequest(r)` 替代 `tokenFromRequest(r)` + `resolveTokenUserID(tokenKey)`。
**Depends on:** nothing (existing `resolveTokenUserIDFromRequest` already supports 4 auth methods)
**Test:** Go unit test + Playwright E2E

## YAML Update

Modify existing step in `conf/value-stream.yaml`:

```yaml
- name: gateway-forward-auth-token-resolve
  status: active  # was: planned
  test_file: taskAuth/src/gateway_forward_auth_test.go  # was: tests/test_domain_events_port_config.py
  fields:
    - name: task-auth.accounts_customtoken.key
      description: 网关 forward-auth 解析 token（auth.db），注入 X-User-Id 头；enrich_login 同步 token 至本地 DB
    - name: task-auth.accounts_user.id
      description: forward-auth 返回 user_id 供后端信任（含 userId cookie fallback）
    - name: task-auth.accounts_user.is_active
      description: forward-auth 校验用户激活状态
    - name: saas-backend.runtime.lifecycle_status
      description: Django TASK_GATEWAY_TRUST_HEADERS 信任网关头短路认证
```

**Change:** `status: planned` → `active`, `test_file` updates to real Go test path.
