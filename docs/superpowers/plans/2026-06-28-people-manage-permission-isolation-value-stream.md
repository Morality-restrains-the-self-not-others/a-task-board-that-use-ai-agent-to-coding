# Value Stream: 人员管理权限隔离

> Derived from design: `docs/specs/people-manage-permission-isolation-design.md`

## Value Summary

确保公司人员管理操作（查看成员列表、邀请、修改角色、启用/禁用、移除）仅对公司创建者和管理员开放，普通成员无法越权操作。

## Related Value Streams

- **`company-management` / `member-crud`** (conf/value-stream.yaml): **modification** — 现有成员 CRUD 操作缺乏 admin/creator 权限校验，本次在相同的 API endpoint 上新增权限门禁。不改变 API 契约，仅增加 403 拒绝路径。

## End-to-End Flow

**[非管理员用户访问人员管理 API]** → [_require_company_admin 检查] → [admin/creator: 正常返回 | 普通成员: 403 Forbidden]

## Value Increments

### Increment 1: 后端 API 权限门禁 (Thin Slice)

**Value to user:** 普通成员访问人员管理 API 时收到 403，防止越权查看/操作

**Scope:**
- `CompanyMemberViewSet` 新增 `_require_company_admin` 方法
- 应用到 5 个 action: `company_members`, `invite`, `update_role`, `toggle_status`, `destroy`
- 新增测试用例：普通成员访问被拒绝

**Depends on:** nothing

### Increment 2: 前端 Sidebar 角色感知 (Enhancement)

**Value to user:** 非管理员用户在侧边栏看不到"人员管理"入口，提升 UX

**Scope:**
- Sidebar 根据 profile API 返回的 `is_admin` 条件渲染"人员管理"菜单
- 直接 URL 访问仍由后端门禁保护

**Depends on:** Increment 1

## Fields Impact

| Field | Change |
|-------|--------|
| `saas-backend.accounts_companymember.is_admin` | 已有字段，运行时检查逻辑加强 |
| `saas-backend.accounts_company.creator_id` | 已有字段，纳入权限判断（creator = 最高权限） |

## Test Impact

| Test File | Change |
|-----------|--------|
| `accounts/view_test/CompanyMemberViewSet_test.py` | 新增: 普通成员调用各管理 action 返回 403 |
