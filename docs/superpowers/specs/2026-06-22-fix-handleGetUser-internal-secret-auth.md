# Fix: handleGetUser Auth Order — Internal Secret Bypass

**Date**: 2026-06-22
**Status**: Fixed

## Problem

密码重置后无法用新密码登录。API 返回 `{"detail":"user not found"}`。

## Root Cause

`taskAuth/src/auth_users.go:handleGetUser` 认证检查顺序错误：

```go
// BEFORE: token check FIRST, internal secret NEVER reached when no token
tokenUserID, err := resolveTokenUserID(tokenFromRequest(r))
if err != nil {
    writeJSON(w, 401, "authentication required")  // ← returns immediately
    return
}
// internal secret check below is unreachable when token is empty
if tokenUserID != userID && !requireInternalSecret(r) { ... }
```

Django 的 `get_identity_client().get_user(uid)` 调用 `GET /api/accounts/users/{uid}/` 时只携带 `X-TaskAuth-Internal-Secret` header（无 bearer token）→ `resolveTokenUserID("")` → error → 直接 401 → Django `load_principal_from_user_id` 返回 None → `enrich_login` 返回 `"user not found"` → 登录失败。

对比 `handleGetSuperAdmin` 正确地先检查 `requireInternalSecret(r)`。

## Fix

**File**: `taskAuth/src/auth_users.go`

当 `resolveTokenUserID` 返回 error 时，先检查 internal secret；若通过则允许访问。若未通过则返回 401。

## Domain Concept Inventory

- **Bounded Context**: 用户与认证 (user-auth)
- **Key Entities**: `User` (accounts_user), `LoginMethod` (accounts_login_method)
- **Candidate Aggregates**: `User` aggregate (login methods are part of user identity)

## Verification

- ✅ `GET /api/accounts/users/{uid}/` with internal secret → returns user JSON
- ✅ Login after password reset → token issued, user redirected
- ✅ 27/27 tests pass
- ✅ Playwright E2E: reset page loads correctly, form visible
