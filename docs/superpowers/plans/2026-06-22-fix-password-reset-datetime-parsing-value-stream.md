# Value Stream: Fix Password Reset Datetime Parsing

> Derived from design: `docs/superpowers/specs/2026-06-22-fix-password-reset-datetime-parsing.md`

## Value Summary

用户能正常使用邮箱密码重置链接和注册激活链接——修复 SQLite 驱动日期时间格式转换导致的 token 过期校验全部失败问题。

## Related Value Streams

- **user-auth → reset-password** (existing, `value-stream.yaml`): modification — bug fix in `isPasswordResetTokenValid()` datetime parsing
- **user-auth → activate** (existing, `value-stream.yaml`): modification — same bug fix in `isActivationTokenValid()` datetime parsing

No new streams needed. Existing active stream entries remain unchanged — this fix corrects the runtime behavior underlying the already-mapped steps.

## End-to-End Flow

```
User clicks email link → Vue Router redirect (/auth/reset-password/:token/ → /auth/reset-password/?token=XXX)
  → ResetPassword.vue: getUserIdFromToken(token)
  → apiFetch GET /api/accounts/users/get-reset-user-info/{token}/
  → taskAuth: handleGetResetUserInfo → isPasswordResetTokenValid → parseDateTime ✅
  → returns {"identifier":"author@example.com"}
  → User sees password reset form → enters new password → POST reset-password-with-link
  → Password reset successful ✅
```

**BEFORE fix**: `parseDateTime` returned error → `valid=false` → "无效的重置链接" → user sees "获取用户信息失败"

## Value Increments

### Increment 1: Fix Datetime Parsing (Single Increment)

**Value to user:** Password reset links and activation links work correctly.
**Scope:** Add `parseDateTime()` helper in `taskAuth/src/db.go` supporting all 4 datetime formats (Go-native microsecond, Go-native second, RFC 3339 Nano, RFC 3339). Update `isPasswordResetTokenValid()` and `isActivationTokenValid()` to use it.
**Depends on:** nothing — pure code fix.
**Test file:** `taskAuth/src/auth_password_reset_test.go` (existing), `taskAuth/src/db.go` (no new test needed — existing tests cover token validation paths)
