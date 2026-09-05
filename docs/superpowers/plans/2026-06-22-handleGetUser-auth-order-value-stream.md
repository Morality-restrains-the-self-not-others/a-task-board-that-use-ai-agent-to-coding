# Value Stream: Fix handleGetUser Internal Secret Auth Order

> Derived from: `docs/superpowers/specs/2026-06-22-fix-handleGetUser-internal-secret-auth.md`

## Value Summary

用户密码重置后能正常登录——修复 taskAuth `handleGetUser` 认证检查顺序，使 Django identity client 携带 internal secret 的调用不被错误拒止。

## Related Value Streams

- **user-auth → login** (existing): modification — 修复 `handleGetUser` 被 identity client 调用时的认证逻辑
- **user-auth → reset-password** (existing): dependency — 密码重置依赖登录流程正确工作

## End-to-End Flow

```
Django identityClient.get_user(uid)
  → GET /api/accounts/users/{uid}/ + X-TaskAuth-Internal-Secret
  → handleGetUser: resolveTokenUserID("") → error
  → BEFORE: 401 "authentication required" ← BUG
  → AFTER: requireInternalSecret(r) → true → proceed → return user ✅
  → Django enrich_login succeeds → login completes
```

## Value Increments

### Increment 1: Fix auth order in handleGetUser (single increment)

**Value to user:** Internal service calls succeed; login after password reset works.
**Scope:** `taskAuth/src/auth_users.go:handleGetUser` — ~5 line change
**Depends on:** nothing
**Test file:** `taskAuth/src/auth_users_test.go` (existing)
