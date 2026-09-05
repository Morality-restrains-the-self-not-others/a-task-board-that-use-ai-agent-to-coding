# Fix: SQLite Driver Datetime Format Mismatch in Token Expiry Validation

**Date**: 2026-06-22
**Status**: Fixed — awaiting review & ship

## Problem

用户通过邮箱收到的密码重置链接进行密码重置时，页面提示「获取用户信息失败，请重新请求密码重置」。注册激活链接可能存在相同问题。

### Symptom

- `GET /api/accounts/users/get-reset-user-info/{token}/` 始终返回 `{"error":"无效的重置链接"}`
- `POST /api/accounts/users/reset-password-with-link/{token}/` 同样返回该错误
- 前端 `ResetPassword.vue:getUserIdFromToken()` 收到非 200 响应 → 显示「获取用户信息失败，请重新请求密码重置」

## Root Cause

**`modernc.org/sqlite` 驱动在扫描带有 `datetime` affinity 的列时，将存储的文本值转换为 ISO 8601/RFC 3339 格式。**

```
存储时的格式 (Go → DB):    "2026-06-23 10:37:59.372190"
读取时的格式 (DB → Go):    "2026-06-23T10:37:59.37219Z"
代码尝试解析的格式:          "2006-01-02 15:04:05.999999"  ← 失败
```

`isPasswordResetTokenValid()` 和 `isActivationTokenValid()` 都只尝试使用空格分隔的两种格式解析日期时间字符串，未覆盖 ISO 8601 格式，导致所有 token 过期校验都失败。

### Confirmed via debug logging

```
expires.String="2026-06-23T10:37:59.37219Z"   // 驱动返回 ISO 8601
first parse failed: cannot parse "T10:37:59.37219Z" as " "   // 格式不匹配
both parses failed → valid=false
```

## Fix

**File**: `taskAuth/src/db.go`

### 1. New `parseDateTime()` helper

```go
func parseDateTime(s string) (time.Time, error) {
    formats := []string{
        "2006-01-02 15:04:05.999999",  // Go 原生格式 (微秒)
        "2006-01-02 15:04:05",         // Go 原生格式 (无微秒)
        time.RFC3339Nano,               // ISO 8601 带纳秒 (驱动可能返回)
        time.RFC3339,                   // ISO 8601 秒级
    }
    for _, layout := range formats {
        t, err := time.Parse(layout, s)
        if err == nil {
            return t, nil
        }
    }
    return time.Time{}, fmt.Errorf("parseDateTime: unable to parse %q", s)
}
```

### 2. Updated `isPasswordResetTokenValid()` and `isActivationTokenValid()`

Both functions now delegate time parsing to `parseDateTime()` instead of inline two-format attempts.

## Domain Concept Inventory

- **Bounded Context**: 用户与认证 (user-auth)
- **Key Entities**:
  - `LoginMethod` (`accounts_login_method`) — 登录方式，持有 `password_reset_token` 和 `activation_token`
- **Candidate Aggregates**: `LoginMethod` (token lifecycle)
- **Domain Events**: None (pure infrastructure fix, no new events)

## Value Stream Impact

| Question | Answer |
|----------|--------|
| Which existing streams affected? | `user-auth` → `reset-password` step; `user-auth` → `activate` step |
| New stream needed? | No |
| Fields impact | `task-auth.accounts_login_method.password_reset_token_expires_at` (read path fix); `task-auth.accounts_login_method.activation_token_expires_at` (read path fix) |
| Test impact | Existing tests in `taskAuth/src/auth_password_reset_test.go` pass; no new tests required (bug was in datetime parsing, not in logic) |
| Status changes | None |
| Cross-stream dependencies | None |

## Verification

- ✅ `GET /api/accounts/users/get-reset-user-info/{token}/` → `{"identifier":"author@example.com"}`
- ✅ `POST /api/accounts/users/reset-password-with-link/{token}/` → `{"message":"密码重置成功"}`
- ✅ Through gateway (port 18081): both endpoints work
- ✅ Playwright E2E: password reset page shows form, not error
- ✅ All 27 existing taskAuth unit tests pass
