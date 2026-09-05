# Value Stream: 公司切换后项目列表为空 — 修复

> Derived from design: `docs/specs/company-switch-project-list-empty-design.md`
> Date: 2026-06-28
> Type: Bug fix — extends previous `company-switch-revert-on-refresh` fix with `/me` API tenant context propagation

## Value Summary

用户切换公司后点击「创建任务」，项目列表正确显示该公司的项目，而非空列表。

## Related Value Streams

- **`2026-06-28-company-switch-revert-on-refresh`**: **extension** — 上一个修复解决了 Navbar/Sidebar 的 `currentTenant` 回退问题（前端 URL 优先），但未解决 `/me` API 本身缺少 `tenant_id` 上下文的问题。本修复补齐剩余部分：将 `tenant_id` 通过 query param 传入 `/me` API，让后端 `get_current_workspace` 能正确解析当前公司的工作空间。
- **`project-workspace` / `workspace-crud`**: **modification** — `get_current_workspace` 增加 query param fallback，workspace 解析依赖 `tenant_id` 的传递链完整化。

## End-to-End Flow

```
[用户切换公司] → URL = /tenant/<new_id>/work-panel
  → WorkPanel.initData()
    → GET /api/user/{uid}/accounts/users/me/?tenant_id=<new_id>  ← 新增 query param
      → get_current_workspace(tenant_id=<new_id>)  ← 正确解析新公司 workspace
        → 返回 new_workspace (属于新公司)
          → currentWorkspace = new_workspace ✅
            → fetchProjects():
              GET /api/tenant/<new_id>/projects/?workspace_id=<new_workspace_id>
                → 返回新公司的项目列表 ✅ (修复前为空 ❌)
```

## Value Increments

### Increment 1: `/me` API tenant 上下文传播 (Thin Slice, ONLY)

**Value to user:** 公司切换后项目列表正确显示（不再为空）

**Scope:**
- `Navbar.logic.vue`: `/me` 调用增加 `?tenant_id=` query param
- `Sidebar.vue`: 同上
- `WorkPanel.vue` `initData()`: 同上
- `user_serializer.py` `get_current_company`: 增加 `request.GET.get('tenant_id')` fallback
- `user_serializer.py` `get_current_workspace`: 同上

**Depends on:** `2026-06-28-company-switch-revert-on-refresh` (前端 URL tenant 优先逻辑)

## Fields Impact

| Field | Change |
|-------|--------|
| `taskFE.runtime.me_api_tenant_param` | `/me` API 调用增加 `?tenant_id=` query param |
| `saas-backend.accounts_companymember.company_id` | `get_current_company` / `get_current_workspace` 增加 query param 作为 tenant_id 来源 |
| `saas-backend.projects_workspace.id` | `get_current_workspace` 在 tenant_id 可用时优先解析该 tenant 的 workspace |

## Test Impact

| Test File | Change |
|-----------|--------|
| `accounts/view_test/UserViewSet_test.py` | 新增：`?tenant_id=` query param 场景下的 `current_company` / `current_workspace` 正确性 |
| Navbar/Sidebar/WorkPanel 组件测试 | 新增：`/me` 调用包含 `?tenant_id=` 参数 |
