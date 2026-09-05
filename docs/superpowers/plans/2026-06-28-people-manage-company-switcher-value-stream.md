# Value Stream: 人员管理 — 公司切换器 + 无权限优雅降级

> Derived from design: `docs/specs/people-manage-company-switcher-graceful-design.md`

## Value Summary

普通成员访问人员管理页面时，不再看到「获取成员列表失败」的错误弹窗，而是看到友好的「没有访问权限」提示；页面顶部新增公司下拉切换器，用户可在所属公司间切换。

## Related Value Streams

- **`2026-06-28-people-manage-permission-isolation`**: **modification** — 将 `member-permission-gate` 步骤的策略从「403 拒绝」调整为「200 + has_permission=false 标记」，同时扩展 `company_nicknames` 字段。前端新增公司切换器 UI。

## End-to-End Flow

[用户访问 people/manage] → [MemberList mount → fetchMembers] → [API 返回 has_permission + members] → [有权限: 列表 | 无权限: 「没有访问权限」]

## Value Increments

### Increment 1: API 语义变更 (Thin Slice)

**Value to user:** 无权限时不再弹 alert 报错

**Scope:**
- `company_members` API: `_require_company_admin` 改为 `_check_company_admin`，返回 `has_permission` 标记而非 403
- `profile` API `company_nicknames`: 增加 `is_admin` / `is_creator`
- `MemberList.vue`: 处理 `has_permission=false`，显示友好提示
- 测试更新: 403 断言改为 200 + has_permission=false

**Depends on:** `2026-06-28-people-manage-permission-isolation`

### Increment 2: 公司切换下拉框 (Enhancement)

**Value to user:** 可在页面顶部下拉切换公司

**Scope:**
- `PeopleManage.vue`: 公司下拉选择器，切换时更新路由
- 数据来源: profile API 的 `company_nicknames`

**Depends on:** Increment 1

## Fields Impact

| Field | Change |
|-------|--------|
| `saas-backend.accounts_companymember.is_admin` | profile API 新增透出此字段 |
| `saas-backend.accounts_company.creator_id` | profile API 新增透出 is_creator 判定 |
| `saas-backend.runtime.company_members_response_has_permission` | 新增 meta 字段 |

## Test Impact

| Test File | Change |
|-----------|--------|
| `accounts/view_test/CompanyMemberViewSet_test.py` | 403 断言 → 200 + has_permission=false |
