# Value Stream: 公司切换后页面刷新回退到初始公司 — 修复

> Derived from design: `docs/specs/company-switch-revert-on-refresh-design.md`
> Date: 2026-06-28
> Type: Bug fix — modifies existing `company-management.user-company-assoc` step, no new steps

## Value Summary

用户通过 Navbar 公司下拉框切换公司后，页面刷新不再回退到初始公司；Navbar 和 Sidebar 的公司选择器与 URL 中的 `/tenant/:tenant` 保持一致。

## Related Value Streams

- **`company-management` / `user-company-assoc`**: **modification** — `get_current_company` 增加 URL `tenant_id` 感知（匹配而非 `.first()`），前端 Navbar/Sidebar 的 `currentTenant` 优先使用 `route.params.tenant`。现有的 `accounts/view_test/UserViewSet_test.py` 需新增 `get_current_company` tenant-aware 断言。
- **`2026-06-28-people-manage-company-switcher`**: **related but distinct** — 该流为 PeopleManage 页面新增本地公司切换器，本修复针对的是**全局 Navbar** 公司切换器在页面刷新后状态回退的 bug。两者共享 `user-company-assoc` 步骤中的 `current_company` 语义。
- **`2026-06-28-workspace-switcher-navbar-consolidation`**: **pattern reference** — 该流将 WorkspaceSwitcher 移至 Navbar 并建立了「URL 是上下文真源」的模式。本修复将同一模式应用于公司切换器。

## End-to-End Flow

```
[用户切换公司] → switchCompany(B) → window.location.href = /tenant/B/...
  → 页面加载 → onMounted()
    → Navbar.logic: applyMePayload → route.params.tenant = "B" → currentTenant = "B" ✅
    → Sidebar: initData → route.params.tenant = "B" → currentTenant = "B" ✅
    → WorkPanel: initData → URL path → tenantId = "B" ✅ (already correct)
  → Navbar 下拉框显示公司 B ✅ (修复前显示公司 A ❌)
```

## Value Increments

### Increment 1: URL Tenant 优先 — 公司上下文一致性 (Thin Slice, ONLY)

**Value to user:** 公司切换后 Navbar 和 Sidebar 显示正确的公司，与 URL 一致

**Scope:**
- `Navbar.logic.vue` `applyMePayload`: `currentTenant` 优先用 `route.params.tenant`（验证用户属于该公司）
- `Sidebar.vue` `initData`: 同上
- `user_serializer.py` `get_current_company`: 可选增强 — 检查 request resolver_match 中的 `tenant_id` kwarg，优先匹配

**Depends on:** nothing（前端路由参数已存在，仅未被使用）

## Fields Impact

| Field | Change |
|-------|--------|
| `saas-backend.accounts_companymember.company_id` | `get_current_company` 新增 tenant_id 过滤（selective filter） |
| `taskFE.runtime.currentTenant_resolution` | Navbar/Sidebar 的 `currentTenant` 来源从 API-only 改为 URL-first |

## Test Impact

| Test File | Change |
|-----------|--------|
| `accounts/view_test/UserViewSet_test.py` | 新增：`get_current_company` 在 `tenant_id` kwarg 存在时返回正确公司 |
| Navbar/Sidebar 单元测试 | 新增：URL 有 tenant 时 `currentTenant` 使用 URL 值；无 tenant 时 fallback API |
| Playwright E2E: `WorkPanel` 公司切换 | 新增：切换公司 → 验证 Navbar 下拉框和 Sidebar 显示正确公司 |
