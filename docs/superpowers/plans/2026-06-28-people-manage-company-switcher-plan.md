# 实施计划: 公司切换器 + 无权限优雅降级

> 输入: `docs/specs/people-manage-company-switcher-graceful-design.md`

## 任务清单

### Task 1: 后端 `company_members` API — 403 → 200+标记
- [ ] `member_views.py`: `_require_company_admin` → `_check_company_admin`，返回 `(has_permission, None, None)` 或 `(False, message, None)`
- [ ] `company_members` action: 无权限时返回 200 + `members:[]` + `meta.has_permission:false`
- [ ] 其他 action (`invite`, `update_role`, `toggle_status`, `destroy`): 保持 403（这些是写操作）
- **文件**: `accounts/views/member_views.py`

### Task 2: 后端 profile `company_nicknames` — 增加 is_admin/is_creator
- [ ] `user_views.py` `_build_profile_payload`: 每个 company 追加 `is_admin` / `is_creator`
- **文件**: `accounts/views/user_views.py`

### Task 3: 更新现有测试
- [ ] `CompanyMemberViewSet_test.py`: 5 个 403 测试中，`test_regular_member_cannot_view_members` 改为断言 200 + has_permission=false
- [ ] 其余 4 个写操作测试保持 403 断言
- **文件**: `accounts/view_test/CompanyMemberViewSet_test.py`

### Task 4: 前端 `MemberList.vue` — 处理 has_permission=false
- [ ] `fetchMembers`: 检查 `meta.has_permission`，为 false 时显示友好提示
- [ ] 移除 `alert()` 调用，改为设置 `permissionMessage` ref
- [ ] 模板新增无权限提示区块
- **文件**: `front_project/app/src/components/MemberList.vue`

### Task 5: 前端 `PeopleManage.vue` — 公司切换下拉框
- [ ] 从 profile API 获取 `company_nicknames`（含 is_admin/is_creator）
- [ ] 渲染下拉选择器，切换时更新路由 `/tenant/{new_id}/people/manage/`
- [ ] 标注每个公司的权限标签（管理员/成员）
- **文件**: `front_project/app/src/views/PeopleManage.vue`
