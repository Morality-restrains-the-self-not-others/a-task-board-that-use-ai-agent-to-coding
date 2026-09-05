# 实施计划: 功能参数多层级配置

> 来源:
> - 设计: `docs/designs/feature-params-hierarchy.md`
> - 价值流: `docs/superpowers/plans/2026-06-30-feature-params-hierarchy-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-30-feature-params-hierarchy-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-30-feature-params-hierarchy-ddd.md`

---

## Increment 0: 修复现有权限缺陷

### 0.1 公司配置 POST 增加 admin 校验

- [ ] **0.1.1** 修改 `frontend_app/views/feature_views.py:manage_feature_params` POST 分支，在 `update_or_create` 前增加:
  ```python
  if not CompanyMember.objects.filter(
      user_id=request.user.id, company_id=company.id, is_admin=True, is_active=True
  ).exists():
      return JsonResponse({'message': '仅租户管理员可修改功能参数配置'}, status=403)
  ```
- [ ] **0.1.2** 更新 `tests/test_manage_feature_params.py`: 新增 `test_non_admin_cannot_update_feature_params` (非管理员 POST → 403)
- [ ] **0.1.3** 运行 `pytest tests/test_manage_feature_params.py -v` 确认全部通过

---

## Increment 1: 数据模型 + 配置解析服务

### 1.1 Models & Migrations

- [ ] **1.1.1** 修改 `projects/models/workspace.py` — Workspace 增加 `allow_personal_feature_params` 字段
  ```python
  allow_personal_feature_params = models.BooleanField(default=False)
  ```
- [ ] **1.1.2** 创建 `projects/models/workspace_feature_params.py` — WorkspaceFeatureParams 模型
- [ ] **1.1.3** 创建 `projects/models/personal_feature_params_config.py` — PersonalFeatureParamsConfig 模型
- [ ] **1.1.4** 创建 `projects/models/task_feature_params_snapshot.py` — TaskFeatureParamsSnapshot 模型
- [ ] **1.1.5** 修改 `projects/models/todo.py` — Todo 增加 `feature_params_source` + `personal_feature_params_config_id`
- [ ] **1.1.6** 生成 migration: `python manage.py makemigrations projects`
- [ ] **1.1.7** 执行 migration: `python manage.py migrate projects`

### 1.2 Domain Entities & Value Objects

- [ ] **1.2.1** 创建 `projects/domain/feature_params/value_objects/feature_params_source.py` ✅ (DDD 步骤已产出)
- [ ] **1.2.2** 创建 `projects/domain/feature_params/value_objects/providers_summary.py` ✅
- [ ] **1.2.3** 创建 `projects/domain/feature_params/value_objects/snapshot_meta.py` ✅
- [ ] **1.2.4** 创建 `projects/domain/feature_params/entities/tenant_feature_params.py` ✅
- [ ] **1.2.5** 创建 `projects/domain/feature_params/entities/workspace_feature_params.py` ✅
- [ ] **1.2.6** 创建 `projects/domain/feature_params/entities/personal_feature_params_config.py` ✅
- [ ] **1.2.7** 创建 `projects/domain/feature_params/entities/task_feature_params_snapshot.py` ✅

### 1.3 Domain Service — Resolver

- [ ] **1.3.1** 创建 `projects/domain/feature_params/ports/repositories/` 下 5 个端口接口 ✅
- [ ] **1.3.2** 创建 `projects/domain/feature_params/services/feature_params_resolver.py` ✅
- [ ] **1.3.3** 创建 `projects/domain/feature_params/events/feature_params_resolved.py` ✅
- [ ] **1.3.4** 创建 `projects/domain/feature_params/services/feature_params_application_service.py` ✅

### 1.4 Infrastructure — Django Repository Adapters

- [ ] **1.4.1** 创建 `projects/infrastructure/adapters/persistence/django_tenant_feature_params_repository.py` — 实现 `TenantFeatureParamsRepository`
- [ ] **1.4.2** 创建 `projects/infrastructure/adapters/persistence/django_workspace_feature_params_repository.py` — 实现 `WorkspaceFeatureParamsRepository`
- [ ] **1.4.3** 创建 `projects/infrastructure/adapters/persistence/django_personal_feature_params_config_repository.py` — 实现 `PersonalFeatureParamsConfigRepository`
- [ ] **1.4.4** 创建 `projects/infrastructure/adapters/persistence/django_task_feature_params_snapshot_repository.py` — 实现 `TaskFeatureParamsSnapshotRepository`
- [ ] **1.4.5** 创建 `projects/infrastructure/adapters/persistence/django_workspace_governance_adapter.py` — 实现 `WorkspaceGovernancePort`

### 1.5 Container Env Fetch — 接驳 Resolver

- [ ] **1.5.1** 修改 `cloud/views/container_feature_params_views.py:fetch_tenant_feature_params_env_for_container`:
  - 将 `TenantFeatureParams.objects.filter(company_id=...).first()` 替换为调用 `FeatureParamsApplicationService.resolve_and_snapshot()`
  - 从 Todo 读取 `feature_params_source` + `personal_feature_params_config_id`
- [ ] **1.5.2** 确保快照写入失败不阻塞 env 返回（try/except + log warning）

### 1.6 Unit Tests — Resolver

- [ ] **1.6.1** 创建 `tests/test_feature_params_resolver.py`:
  - `test_resolve_company_source` — source=company → 返回公司配置
  - `test_resolve_workspace_with_custom_config` — source=workspace + 有自定义 → 返回工作空间配置
  - `test_resolve_workspace_inherit_company` — source=workspace + use_company_default → 返回公司配置
  - `test_resolve_personal_success` — source=personal + 治理允许 + config 存在 → 返回个人配置
  - `test_resolve_personal_blocked_by_governance` — 治理禁止 → 回退公司 + warning
  - `test_resolve_personal_config_deleted` — 个人配置已删除 → 回退公司 + warning
  - `test_resolve_personal_wrong_user` — 个人配置不属于该用户 → 回退公司
  - `test_resolve_company_not_found_raises` — 公司配置不存在 → ValueError
- [ ] **1.6.2** 运行 `pytest tests/test_feature_params_resolver.py -v` 确认 8 条全部通过

---

## Increment 2: 管理 API

### 2.1 Workspace Feature Params API

- [ ] **2.1.1** 创建 `frontend_app/views/workspace_feature_params_views.py`:
  - `GET /api/tenant/{t}/workspace/{w}/feature-params/` — 读工作空间配置 + 返回公司配置作为参考
  - `POST /api/tenant/{t}/workspace/{w}/feature-params/` — 写工作空间覆盖
  - 权限: `has_workspace_access(user, workspace_id)` + 写需要 `WorkspaceAccess.role='admin'`
- [ ] **2.1.2** 注册路由 `frontend_app/urls.py`
- [ ] **2.1.3** 扩展 workspace settings API: `PATCH /api/tenant/{t}/workspace/{w}/` 支持 `allow_personal_feature_params` 字段

### 2.2 Personal Feature Params Configs API

- [ ] **2.2.1** 创建 `frontend_app/views/personal_feature_params_views.py`:
  - `GET /api/personal/feature-params-configs/` — 列出当前用户的所有 personal configs（filter by user_id）
  - `POST /api/personal/feature-params-configs/` — 创建（`user_id = request.user.id` 硬编码）
  - `GET /api/personal/feature-params-configs/{id}/` — 读单个（`IsPersonalConfigOwner` 校验）
  - `PUT /api/personal/feature-params-configs/{id}/` — 改（`IsPersonalConfigOwner` 校验）
  - `DELETE /api/personal/feature-params-configs/{id}/` — 删（`IsPersonalConfigOwner` 校验）
- [ ] **2.2.2** 创建 `frontend_app/permissions.py` → `IsPersonalConfigOwner` DRF Permission 类
- [ ] **2.2.3** 注册路由 `frontend_app/urls.py`

### 2.3 Task Feature Params Binding API

- [ ] **2.3.1** 修改 `projects/views/todo_views.py:TodoViewSet.create` — 新增字段处理:
  - 从请求体读取 `feature_params_source` 和 `personal_feature_params_config_id`
  - 选 personal 时三重校验: workspace.allow_personal + config 属于用户 + config.company == workspace.company
- [ ] **2.3.2** 创建 `projects/views/task_feature_params_views.py`:
  - `PATCH /api/tasks/{task_id}/feature-params/` — 修改配置来源（`IsTodoMutationOwner` + 三重校验）
- [ ] **2.3.3** 注册路由

### 2.4 Snapshots Query API

- [ ] **2.4.1** 创建 `projects/views/task_feature_params_snapshot_views.py`:
  - `GET /api/tasks/{task_id}/feature-params-snapshots/` — 脱敏列表（需 workspace view 权限）
  - 响应仅含 `providers_summary`，不含 `resolved_env`
- [ ] **2.4.2** 注册路由

### 2.5 API Tests

- [ ] **2.5.1** 创建 `tests/test_workspace_feature_params_api.py` — CRUD + 权限 (T4-T10)
- [ ] **2.5.2** 创建 `tests/test_personal_feature_params_config_api.py` — CRUD + IDOR (T11-T17)
- [ ] **2.5.3** 创建 `tests/test_task_feature_params_binding.py` — 绑定 + 三重校验 (T18-T25)
- [ ] **2.5.4** 创建 `tests/test_task_feature_params_snapshot_api.py` — 脱敏 + 权限 (T26-T28)
- [ ] **2.5.5** 运行全部 API 测试: `pytest tests/test_*feature_params*.py -v`

---

## Increment 3: 前端 UI

### 3.1 Workspace Feature Params Page

- [ ] **3.1.1** 创建 `front_project/app/src/views/WorkspaceFeatureParamsSettings.vue`
  - "使用公司默认" / "自定义" 切换
  - 公司配置只读参考区
  - Provider Editor + Model Selector 复用现有组件
  - "允许成员使用个人配置" 开关
- [ ] **3.1.2** 注册路由: `/tenant/:tenant/settings/workspace/:workspace/feature-params/`
- [ ] **3.1.3** 在 Sidebar/WorkspaceSettings 中添加导航入口

### 3.2 Personal Configs Page

- [ ] **3.2.1** 创建 `front_project/app/src/views/PersonalFeatureParamsConfigs.vue`
  - 配置列表（Card 布局，展示 name + 摘要信息）
  - 新建/编辑弹窗（复用 FeatureParamsProvidersEditor）
  - 复制操作
  - 仅在 `allow_personal_feature_params=true` 的工作空间可见入口
- [ ] **3.2.2** 注册路由: `/personal/feature-params-configs/`
- [ ] **3.2.3** 在 UserCenter/Navbar 中添加入口

### 3.3 Task Form — Config Selector

- [ ] **3.3.1** 修改 `TaskDetailContent.logic.vue` 或任务创建表单组件
  - 新增"功能参数"下拉选择器
  - 选项动态生成: 公司默认 / 工作空间默认（有自定义时）/ 分隔线 / 个人配置列表（仅 workspace 允许时）
  - 默认选中「工作空间默认」（有自定义时）或「公司默认」
- [ ] **3.3.2** 修改任务详情展示当前使用的配置（只读标签 + source_display_name）

### 3.4 Task Detail — Snapshot History

- [ ] **3.4.1** 在任务详情页增加「运行记录」面板
  - 调用 `GET /api/tasks/{id}/feature-params-snapshots/`
  - 时间线列表展示每次运行的 source_display_name + 摘要
  - 点击展开查看 providers_summary（脱敏）

---

## Increment 4: 集成测试 + 数据兼容

### 4.1 Backward Compatibility

- [ ] **4.1.1** 验证: 现有任务的 `feature_params_source` 默认 'company'，容器 env 拉取行为不变
- [ ] **4.1.2** 验证: 未设置 WorkspaceFeatureParams 的工作空间，env 拉取正常回退到公司配置
- [ ] **4.1.3** 验证: 未设置 `allow_personal_feature_params` 的工作空间（默认 false），个人配置列表 API 返回空

### 4.2 E2E Tests (Playwright)

- [ ] **4.2.1** 创建 `playwright/front_project/tests/FeatureParamsHierarchy.e2e.test.js`
  - 完整流程: 管理员配 workspace → 开启 allow_personal → 用户创建 personal config → 创建任务选 personal → 查看快照
- [ ] **4.2.2** 运行 E2E: `cd task2app && npx playwright test FeatureParamsHierarchy`

### 4.3 Fallback Path Tests

- [ ] **4.3.1** 测试: 删除已绑定任务的 personal config → 容器重启 → 回退到公司默认 + 快照记录回退原因
- [ ] **4.3.2** 测试: workspace 关闭 allow_personal → 已绑定 personal config 的存量任务 → 回退到公司默认

---

## 依赖关系

```
Inc 0 (fix admin guard)
  └─ Inc 1 (models + resolver)
       └─ Inc 2 (management APIs)
            └─ Inc 3 (frontend UI)
                 └─ Inc 4 (E2E + compat)
```

每个 Increment 内的任务可按顺序执行，部分任务可并行（如 models 1.1.x 和 domain entities 1.2.x）。

## 测试清单汇总

| 测试文件 | 用例数 | 覆盖 |
|----------|--------|------|
| `tests/test_manage_feature_params.py` | +1 | 非管理员 403 |
| `tests/test_feature_params_resolver.py` | 8 | 6条路径 + 治理 + 回退 |
| `tests/test_workspace_feature_params_api.py` | 7 | CRUD + admin/viewer 权限 |
| `tests/test_personal_feature_params_config_api.py` | 7 | CRUD + IDOR + user_id 注入 |
| `tests/test_task_feature_params_binding.py` | 8 | 绑定 + 三重校验 |
| `tests/test_task_feature_params_snapshot_api.py` | 3 | 脱敏 + 权限 |
| `FeatureParamsHierarchy.e2e.test.js` | 1 | 完整流程 |
| **Total** | **35** | |
