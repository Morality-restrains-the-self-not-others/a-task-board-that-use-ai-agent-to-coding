# 实施计划: 公司切换后项目列表为空 — 修复

## 任务清单

- [ ] **Task 1**: Navbar.logic.vue — `/me` 调用增加 `?tenant_id=`
  - 文件: `front_project/app/src/components/Navbar.logic.vue`
  - 位置: `fetchCurrentUser()` 中 `meResponse` 调用
  - 改动: URL 追加 `?tenant_id=${route.params.tenant}` query param

- [ ] **Task 2**: Sidebar.vue — `/me` 调用增加 `?tenant_id=`
  - 文件: `front_project/app/src/components/Sidebar.vue`
  - 位置: `initData()` 中 apiFetch 调用
  - 改动: URL 追加 query param

- [ ] **Task 3**: WorkPanel.vue — `/me` 调用增加 `?tenant_id=`
  - 文件: `front_project/app/src/views/WorkPanel.vue`
  - 位置: `initData()` 中 apiFetch 调用 (line ~302)
  - 改动: URL 追加 `?tenant_id=${tenantIdFromUrl}` query param

- [ ] **Task 4**: user_serializer.py — query param fallback
  - 文件: `Saas_project/accounts/serializers/user_serializer.py`
  - `get_current_company`: 在 `resolver_match` 后增加 `request.GET.get('tenant_id')` fallback
  - `get_current_workspace`: 同上

- [ ] **Task 5**: 运行测试确保无回归
  - `pytest accounts/view_test/UserViewSet_test.py -x`

## 依赖

```
Task 1,2,3 (并行) + Task 4 (并行) → Task 5
```
