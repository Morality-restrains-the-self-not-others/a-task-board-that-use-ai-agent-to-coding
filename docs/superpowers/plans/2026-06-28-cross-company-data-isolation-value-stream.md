# Value Stream: 跨公司数据隔离 — tenant_id 作用域修复

> Derived from design: `docs/superpowers/specs/2026-06-28-cross-company-data-isolation-design.md`
> Date: 2026-06-28
> Type: Security fix — modifies existing `workspace_access_views.py`, `group_views.py`, `member_views.py`, and frontend company-switch logic

## Value Summary

多公司用户切换公司后，所有 API 请求严格作用于 URL 中 `tenant_id` 对应的公司，防止跨公司数据泄露（workspace collaborators、权限、分组、邀请等数据不再混入其他公司）。

## Related Value Streams

- **`2026-06-28-company-switch-revert-on-refresh`**: **extension** — 该流修复了 Navbar/Sidebar 的 `currentTenant` URL-first 解析。本修复补齐后端视图层的 tenant_id 作用域缺失：11 处 `CompanyMember.objects.filter(user_id=...).first()` 无一使用 tenant_id 过滤。
- **`2026-06-28-company-switch-project-list-empty`**: **extension** — 该流修复了 `/me` API 缺少 `?tenant_id=` 问题。本修复补充了前端 `workspace_id` 查询参数跨公司污染问题，以及后端视图层缺失 tenant 作用域的根本缺陷。
- **`2026-06-28-people-manage-permission-isolation`**: **related but distinct** — 该流为 PeopleManage API 增加 admin 权限门禁。本修复为 workspace/group/member 视图增加 tenant_id 作用域约束。两者互补：权限门禁控制"谁能操作"，tenant 作用域控制"操作哪个公司的数据"。
- **`company-management` / `company-switch-url-context`** (conf/value-stream.yaml, status: planned): **modification** — 该步骤描述 `get_current_company` 增加 tenant_id aware filter。本修复将此步骤从 planned 推进到 active，并扩展到 11 处视图层调用点，以及新增统一的 `resolve_company_member_for_tenant` 工具函数。

## End-to-End Flow

```
[用户切换公司 B] → switchCompany(B) → 清除跨租户查询参数 (workspace_id)
  → URL = /tenant/B/work-panel/ (无旧 workspace_id 污染)
    → WorkPanel.initData() → GET /me/?tenant_id=B
      → currentWorkspace = B_ws (API 返回，不受 URL 污染)
        → CreateTaskModal → GET /api/tenant/B/.../workspace-collaborators/?workspace_id=B_ws
          → backend: resolve_company_member_for_tenant(user, tenant_id=B)
            → company_member = B 的成员记录 ✅
            → workspace = Workspace.objects.get(id=B_ws)
            → workspace.company == company_member.company ✅
              → 返回 B 的 workspace collaborators ✅
```

## Value Increments

### Increment 1: 统一工具函数 + 后端 CRITICAL 视图修复 (Thin Slice)

**Value to user:** 切换公司后，workspace collaborators、权限、分组、邀请等数据不会泄露其他公司的信息

**Scope:**
- `accounts/workspace_context.py`: 新增 `resolve_company_member_for_tenant(user_id, tenant_id)` 统一工具函数
- `projects/views/workspace_access_views.py`: 6 处 `.first()` → `resolve_company_member_for_tenant`
- `accounts/views/group_views.py`: 2 处 `.first()` → `resolve_company_member_for_tenant`
- `accounts/views/member_views.py`: 3 处 `.first()` → `resolve_company_member_for_tenant`

**Depends on:** nothing（工具函数自包含，tenant_id 已从 URL 传入各视图）

### Increment 2: 后端 HIGH 风险实例修复

**Value to user:** 登录后不再随机跳转到错误公司的页面；序列化器返回正确的公司数据

**Scope:**
- `accounts/taskauth_internal_views.py`: `_login_redirect_url` 确定性排序
- `accounts/taskauth_bridge/principal_loader.py`: companies 确定性排序
- `frontend_app/views/auth_views.py`: 登录重定向确定性排序
- `accounts/serializers/user_serializer.py`: `.first()` fallback 排序
- `accounts/serializers/group_serializer.py`: queryset 作用域

**Depends on:** Increment 1

### Increment 3: 前端查询参数清理

**Value to user:** 公司切换后不再看到「旧公司 workspace 在新公司页面中」的错误状态

**Scope:**
- `Navbar.logic.vue` `switchCompany`: 切换公司时清除 `workspace_id` 等跨租户查询参数
- `WorkPanel.vue` `initData`: 不盲信 URL `workspace_id`，API 返回值优先
- `router.js` `beforeEach`: 租户变更时清理本地 workspace 状态

**Depends on:** Increment 1

### Increment 4: 测试覆盖

**Value to user:** 回归保护 — 后续变更不会重新引入跨公司数据泄露

**Scope:**
- `accounts/tests/test_workspace_context.py`: `resolve_company_member_for_tenant` 单元测试
- `projects/view_test/WorkspaceAccessViewSet_test.py`: 多公司交叉访问
- `accounts/view_test/CompanyMemberViewSet_test.py`: 多公司隔离
- Playwright E2E: 跨公司切换 workspace 隔离验证

**Depends on:** Increment 1, 2, 3

## Fields Impact

| Field | Change |
|-------|--------|
| `saas-backend.accounts_companymember.company_id` | `resolve_company_member_for_tenant` 强制过滤；新增 `company_id=tenant_id_int` 条件 |
| `saas-backend.projects_workspace.company_id` | 现有字段，workspace.company 校验依赖 tenant 作用域已正确约束的 company_member |
| `saas-backend.accounts_company.id` | 确定性排序（登录/重定向场景） |
| `taskFE.runtime.company_switch_query_cleanup` | Navbar switchCompany 清除跨租户查询参数 |
| `taskFE.runtime.workspace_id_resolution` | WorkPanel 不盲信 URL workspace_id |

## Test Impact

| Test File | Change |
|-----------|--------|
| `accounts/tests/test_workspace_context.py` | **新增**: `resolve_company_member_for_tenant` 正确/缺失 tenant_id/无效 tenant_id/多公司 |
| `projects/view_test/WorkspaceAccessViewSet_test.py` | 增强: 多公司用户交叉访问 workspace_collaborators/workspace_permissions 返回 403 |
| `accounts/view_test/CompanyMemberViewSet_test.py` | 增强: invite/pending_invitations/resend 多公司隔离 |
| `accounts/view_test/GroupViewSet_test.py` | 增强: 多公司分组隔离 |
| `accounts/view_test/UserViewSet_test.py` | 增强: `get_current_company` tenant_id fallback (衔接 existing planned step) |
| `playwright/front_project/tests/` | **新增**: 跨公司切换后 workspace_id 清理 + workspace collaborator 正确性 E2E |
