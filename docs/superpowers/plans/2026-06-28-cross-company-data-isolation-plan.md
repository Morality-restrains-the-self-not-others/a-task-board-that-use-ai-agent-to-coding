# Implementation Plan: 跨公司数据隔离 — tenant_id 作用域修复

> Derived from:
> - Design: `docs/superpowers/specs/2026-06-28-cross-company-data-isolation-design.md`
> - Value Stream: `docs/superpowers/plans/2026-06-28-cross-company-data-isolation-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-28-cross-company-data-isolation-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-28-cross-company-data-isolation-ddd.md`
>
> Test strategy: TDD (test first → implement → verify). Each implementation task is preceded by its test task.

## Increment 1: 统一工具函数 + 后端 CRITICAL 视图修复 (Thin Slice)

### Phase 1.1: Domain Layer — TenantMembershipResolver

- [ ] **T1.1.1** Write domain service test: `TenantMembershipResolver.resolve()` — valid, missing company_id, invalid company_id, multi-company user
  - Test file: `accounts/tests/test_workspace_context.py` (new)
  - Command: `cd task2app/Saas_project && python -m pytest accounts/tests/test_workspace_context.py -v`

- [ ] **T1.1.2** Implement `TenantMembershipResolver.resolve(user_id, company_id)` domain service
  - File: `accounts/domain/services/tenant_membership_resolver.py` ✅ (已创建)
  - Export from `accounts/domain/services/__init__.py` ✅ (已更新)
  - Verify: re-run T1.1.1 tests → GREEN

- [ ] **T1.1.3** Write `workspace_context.py` adapter test: `resolve_company_member_for_tenant()` wraps `TenantMembershipResolver`
  - Test file: `accounts/tests/test_workspace_context.py`
  - Verify: tests FAIL (adapter not yet implemented)

- [ ] **T1.1.4** Implement `resolve_company_member_for_tenant(user_id, tenant_id)` in `accounts/workspace_context.py`
  - File: `accounts/workspace_context.py` (enhance existing)
  - Wires `DjangoCompanyMemberRepository` into `TenantMembershipResolver`
  - Verify: re-run T1.1.3 tests → GREEN

### Phase 1.2: workspace_access_views.py — 6 处修复

- [ ] **T1.2.1** Write multi-company isolation test: `workspace_collaborators` with wrong tenant_id → 403
  - Test file: `projects/view_test/WorkspaceAccessViewSet_test.py`
  - Verify: tests FAIL (bug reproduced)

- [ ] **T1.2.2** Fix `workspace_collaborators` (line 211): `.first()` → `resolve_company_member_for_tenant`
  - File: `projects/views/workspace_access_views.py`
  - Verify: re-run T1.2.1 → GREEN

- [ ] **T1.2.3** Write test + fix `workspace_permissions` (line 41)
  - Test: multi-company user → wrong tenant workspace → 403
  - Fix: `.first()` → `resolve_company_member_for_tenant`

- [ ] **T1.2.4** Write test + fix `set_permission` (line 76)
  - Test: multi-company user → set permission on other company workspace → 403
  - Fix: `.first()` → `resolve_company_member_for_tenant`

- [ ] **T1.2.5** Write test + fix `remove_permission` (line 141)
  - Test: multi-company user → remove permission on other company workspace → 403
  - Fix: `.first()` → `resolve_company_member_for_tenant`

- [ ] **T1.2.6** Write test + fix `company_workspaces` (line 176)
  - Test: multi-company user → get workspaces with wrong tenant_id → 403
  - Fix: `.first()` → `resolve_company_member_for_tenant`

- [ ] **T1.2.7** Fix `get_queryset` (line 18): `.first()` → `resolve_company_member_for_tenant`
  - File: `projects/views/workspace_access_views.py`
  - Note: `get_queryset` 无 tenant_id 参数（ViewSet 基类方法），需从 `self.kwargs` 获取

### Phase 1.3: group_views.py — 2 处修复

- [ ] **T1.3.1** Write test + fix `get_queryset` (line 27)
  - Test file: `accounts/view_test/GroupViewSet_test.py`
  - Test: multi-company user → group list scoped to correct company
  - Fix: `.first()` → `resolve_company_member_for_tenant`

- [ ] **T1.3.2** Write test + fix `perform_create` (line 37)
  - Test: multi-company user → create group belongs to URL tenant company
  - Fix: `.first()` → `resolve_company_member_for_tenant`

### Phase 1.4: member_views.py — 3 处修复

- [ ] **T1.4.1** Write test + fix `invite` (line 207)
  - Test file: `accounts/view_test/CompanyMemberViewSet_test.py`
  - Test: multi-company user → invite created in correct company (URL tenant)
  - Fix: `.first()` → `resolve_company_member_for_tenant`

- [ ] **T1.4.2** Write test + fix `pending_invitations` (line 615)
  - Test: multi-company user → only sees invitations for URL tenant company
  - Fix: `.first()` → `resolve_company_member_for_tenant`

- [ ] **T1.4.3** Write test + fix `resend_invitation_link` (line 689)
  - Test: multi-company user → can only resend invitations in URL tenant company
  - Fix: `.first()` → `resolve_company_member_for_tenant`

## Increment 2: 后端 HIGH 风险实例修复

- [ ] **T2.1** Fix login redirect ordering: `_login_redirect_url` (line 120)
  - File: `accounts/taskauth_internal_views.py`
  - Test: `accounts/tests/test_taskauth_bridge.py`
  - Change: add `.order_by('created_at')` or `company_id` ordering

- [ ] **T2.2** Fix principal_loader `companies[0]` (line 76)
  - File: `accounts/taskauth_bridge/principal_loader.py`
  - Add `.order_by('created_at')` to CompanyMember query

- [ ] **T2.3** Fix auth_views login redirect (lines 60, 124)
  - File: `frontend_app/views/auth_views.py`
  - Add deterministic ordering to CompanyMember query

- [ ] **T2.4** Fix user_serializer fallback (lines 85, 110)
  - File: `accounts/serializers/user_serializer.py`
  - Add `.order_by('created_at')` to fallback queries

- [ ] **T2.5** Fix group_serializer queryset (line 38)
  - File: `accounts/serializers/group_serializer.py`
  - Scope to company from request context if available

## Increment 3: 前端查询参数清理

- [ ] **T3.1** Fix `switchCompany` to clear cross-tenant query params
  - File: `front_project/app/src/components/Navbar.logic.vue:196-200`
  - Test: `front_project/app/src/tests/domain/company/company_switch_query_cleanup.test.js` (new)
  - Change: delete `workspace_id` from URL search params before navigation

- [ ] **T3.2** Fix `WorkPanel.initData` to not blindly override API workspace
  - File: `front_project/app/src/views/WorkPanel.vue:339-341`
  - Change: trust API response over URL workspace_id; clean invalid URL param

- [ ] **T3.3** Router guard: strip stale workspace_id on tenant change
  - File: `front_project/app/src/router.js:708-744`
  - Change: `beforeEach` clears `workspace_id` when `to.params.tenant !== from.params.tenant`

## Increment 4: 测试覆盖与验证

- [ ] **T4.1** Run full backend test suite — verify 0 regressions
  - Command: `cd task2app/Saas_project && python -m pytest accounts/view_test/ projects/view_test/ -v --tb=short`

- [ ] **T4.2** Run DDD/BDD compliance check
  - Command: `python scripts/ci/check_ddd_bdd_compliance.py`

- [ ] **T4.3** Write Playwright E2E test: cross-company workspace isolation
  - File: `playwright/front_project/tests/CrossCompany.workspace-isolation.playwright.test.js` (new)
  - Scenario: user in 2 companies → switch to company B → create task → verify workspace belongs to B

## Task Summary

| Phase | Tasks | Files Changed |
|-------|-------|---------------|
| 1.1 Domain | 4 | `domain/services/tenant_membership_resolver.py` (new), `workspace_context.py` |
| 1.2 workspace_access | 7 | `projects/views/workspace_access_views.py` (6 fixes) |
| 1.3 group_views | 2 | `accounts/views/group_views.py` (2 fixes) |
| 1.4 member_views | 3 | `accounts/views/member_views.py` (3 fixes) |
| 2 HIGH fixes | 5 | `taskauth_internal_views.py`, `principal_loader.py`, `auth_views.py`, `user_serializer.py`, `group_serializer.py` |
| 3 Frontend | 3 | `Navbar.logic.vue`, `WorkPanel.vue`, `router.js` |
| 4 Verification | 3 | Test suite, compliance check, E2E |
| **Total** | **27** | **14 files** |
