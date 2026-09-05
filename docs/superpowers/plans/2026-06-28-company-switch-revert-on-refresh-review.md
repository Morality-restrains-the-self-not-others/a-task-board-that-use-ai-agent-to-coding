# Code Review: 公司切换后页面刷新回退 — 修复

> Reviewed against: `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-plan.md`
> Design: `docs/specs/company-switch-revert-on-refresh-design.md`

## Review Summary

| Dimension | Verdict |
|-----------|---------|
| Plan conformance | ✅ All 3 tasks implemented as planned |
| Test regression | ✅ `UserViewSet_test.py` passes (1 passed) |
| Code quality | ✅ Clean, minimal, follows existing patterns |
| Backward compatibility | ✅ No change when URL has no tenant |
| Security | ✅ No new attack surface |

## File-by-File

### 1. Navbar.logic.vue (`applyMePayload`, L109-118)
- ✅ `route.params.tenant` checked before API fallback
- ✅ `companies.some()` validates user belongs to URL company
- ✅ Existing fallback chain preserved
- ✅ No variable shadowing issue (function-scoped `const`)

### 2. Sidebar.vue (`initData`, L282-297)
- ✅ Same URL-priority pattern as Navbar
- ✅ `companies.find()` returns matched company for `tenantId`
- ✅ `if (!tenantId)` guard wraps fallback logic cleanly

### 3. user_serializer.py (`get_current_company`, L65-90)
- ✅ `tenant_id` extraction mirrors `get_current_workspace` pattern
- ✅ `company_id=tenant_id` filter precedes `.first()` fallback
- ✅ No new imports — uses existing `CompanyMember`

## Verdict

**PASS** — No critical issues. No important issues. All changes are minimal, correct, and aligned with the design.

## Risk Assessment

| Risk | Level | Mitigation |
|------|-------|-----------|
| Sidebar `matchedCompany.id` may not be string | 🟢 Low | `currentTenant.value` assignment (L299) is type-tolerant |
| `/me` API called from pages without tenant in URL | 🟢 None | Backward compatible — `tenant_id` extraction returns None, falls back to `.first()` |
