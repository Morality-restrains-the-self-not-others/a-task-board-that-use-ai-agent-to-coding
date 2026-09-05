# 实施计划: 人员管理权限隔离

> 输入:
> - 设计文档: `docs/specs/people-manage-permission-isolation-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-28-people-manage-permission-isolation-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-28-people-manage-permission-isolation-nfr-clarification.md`
> - 领域模型: `accounts/domain/services/company_admin_policy.py`

## 任务清单

### Increment 1: 后端 API 权限门禁 (Thin Slice)

#### Task 1.1: 领域服务单元测试 (TDD)
- [ ] 编写 `tests/domain/test_company_admin_policy.py`
  - `test_creator_is_admin`: creator 返回 True
  - `test_admin_member_is_admin`: is_admin=True 的成员返回 True  
  - `test_regular_member_is_not_admin`: is_admin=False 的成员返回 False
  - `test_non_member_is_not_admin`: 非成员返回 False
  - `test_require_raises_for_regular_member`: require 对普通成员抛 PermissionError
- **文件**: `task2app/Saas_project/tests/domain/test_company_admin_policy.py`
- **命令**: `cd task2app/Saas_project && source activate_env.sh && pytest tests/domain/test_company_admin_policy.py -v`

#### Task 1.2: ViewSet 集成 `_require_company_admin`
- [ ] 在 `CompanyMemberViewSet` 新增 `_require_company_admin(self, company_id)` 辅助方法
- [ ] `company_members` action 开头调用 `_require_company_admin`
- [ ] `invite` action 开头调用 `_require_company_admin`
- [ ] `update_role` action 开头调用 `_require_company_admin`
- [ ] `toggle_status` action 开头调用 `_require_company_admin`
- [ ] `destroy` action 开头调用 `_require_company_admin`
- **文件**: `task2app/Saas_project/accounts/views/member_views.py`
- **原则**: creator 拥有最高权限（`company.creator_id == user.id`），admin 其次（`is_admin=True`）

#### Task 1.3: 权限门禁测试
- [ ] 扩展 `accounts/view_test/CompanyMemberViewSet_test.py`
  - `test_regular_member_cannot_view_members`: 普通成员 GET company_members → 403
  - `test_regular_member_cannot_invite`: 普通成员 POST invite → 403
  - `test_regular_member_cannot_update_role`: 普通成员 PATCH update_role → 403
  - `test_regular_member_cannot_toggle_status`: 普通成员 PATCH toggle_status → 403
  - `test_regular_member_cannot_remove_member`: 普通成员 DELETE destroy → 403
  - `test_admin_can_view_members`: admin 仍可正常访问（回归）
  - `test_creator_can_view_members`: creator 仍可正常访问（回归）
- **文件**: `task2app/Saas_project/accounts/view_test/CompanyMemberViewSet_test.py`
- **命令**: `cd task2app/Saas_project && source activate_env.sh && pytest accounts/view_test/CompanyMemberViewSet_test.py -v`

#### Task 1.4: 越权审计日志
- [ ] `_require_company_admin` 拒绝时输出 `logger.warning("user_id={} company_id={} action={} → 403 (not admin)")` 
- **文件**: `task2app/Saas_project/accounts/views/member_views.py`
- **验证**: `grep "not admin" logs/` 可检索越权记录

### Increment 2: 前端 Sidebar 角色感知 (Enhancement)

#### Task 2.1: Sidebar 条件渲染
- [ ] 从 profile/company API 获取当前用户的 `is_admin` 状态
- [ ] "人员管理" 菜单项 (含子菜单) 添加 `v-if="isCompanyAdmin"` 条件
- **文件**: `task2app/front_project/app/src/components/Sidebar.vue`
- **验证**: 普通成员登录后侧边栏不显示"人员管理"入口；admin 仍可见

## 依赖关系

```
Task 1.1 (领域服务测试) → Task 1.2 (ViewSet 集成) → Task 1.3 (权限测试) → Task 1.4 (审计日志)
Task 1.2 完成 → Task 2.1 (前端)
```

## 执行顺序

1. Task 1.1: TDD — 先写领域服务测试（红）
2. Task 1.2: 实现 ViewSet 权限门禁（绿）
3. Task 1.3: 补充 API 层权限测试（绿）
4. Task 1.4: 审计日志
5. Task 2.1: 前端 Sidebar

## 文件变更汇总

| 文件 | 操作 | Task |
|------|------|------|
| `tests/domain/test_company_admin_policy.py` | 新建 | 1.1 |
| `accounts/views/member_views.py` | 修改 | 1.2, 1.4 |
| `accounts/view_test/CompanyMemberViewSet_test.py` | 修改 | 1.3 |
| `components/Sidebar.vue` | 修改 | 2.1 |
