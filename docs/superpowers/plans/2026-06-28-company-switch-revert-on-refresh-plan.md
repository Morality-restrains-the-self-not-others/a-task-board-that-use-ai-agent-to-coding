# 实施计划: 公司切换后页面刷新回退 — 修复

> 输入:
> - 设计文档: `docs/specs/company-switch-revert-on-refresh-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-nfr-clarification.md`
> - DDD: 已跳过（无新领域概念）
>
> 变更类型: Bug fix — 3 files, 纯前端 + 可选后端增强

## 任务清单

### Phase 1: 前端修复 (核心 — P0)

- [ ] **Task 1.1**: 修改 `Navbar.logic.vue` `applyMePayload` — URL tenant 优先
  - 文件: `task2app/front_project/app/src/components/Navbar.logic.vue`
  - 位置: `applyMePayload` 函数（约 97-126 行）
  - 改动: 在设置 `currentTenant.value` 前，检查 `route.params.tenant`，验证是否在 `userInfo.companies` 中；若匹配则优先使用
  - 验证: 单元测试 — URL 有 `tenant=B` 且用户属于 B → `currentTenant = "B"`；URL 无 tenant → fallback 到 `current_company`

- [ ] **Task 1.2**: 修改 `Sidebar.vue` `initData` — URL tenant 优先
  - 文件: `task2app/front_project/app/src/components/Sidebar.vue`
  - 位置: `initData` 函数（约 265-311 行）
  - 改动: 在设置 `currentTenant.value` 前，检查 `route.params.tenant`，验证是否在 `userData.companies` 中；若匹配则优先使用
  - 验证: 同上 pattern

### Phase 2: 后端增强 (可选 — P1)

- [ ] **Task 2.1**: 修改 `UserSerializer.get_current_company` — tenant_id URL 感知
  - 文件: `task2app/Saas_project/accounts/serializers/user_serializer.py`
  - 位置: `get_current_company` 方法（约 65-77 行）
  - 改动: 从 `request.resolver_match.kwargs` 提取 `tenant_id`；若存在则优先 `filter(company_id=tenant_id).first()`
  - 验证: 现有 `accounts/view_test/UserViewSet_test.py` 新增断言 — `tenant_id` kwarg 存在时返回正确公司

### Phase 3: 验证

- [ ] **Task 3.1**: 运行现有前端测试确保无回归
  - 命令: 前端单元测试 (vitest/jest)
  - 预期: 全部通过

- [ ] **Task 3.2**: 运行现有后端测试确保无回归
  - 命令: `cd task2app/Saas_project && source activate_env.sh && python -m pytest accounts/view_test/UserViewSet_test.py -x`
  - 预期: 全部通过

- [ ] **Task 3.3**: 端到端验证 — 公司切换后 Navbar 正确显示
  - 方式: Playwright E2E 或手动验证
  - 场景: 用户属于公司 A 和 B → 加载 `/tenant/A/work-panel` → 切换至 B → 验证 Navbar 下拉框显示 B

## 依赖关系

```
Task 1.1 ──┐
            ├──→ Task 3.1 (前端测试)
Task 1.2 ──┘
            │
Task 2.1 ───→ Task 3.2 (后端测试)
            │
            └──→ Task 3.3 (E2E)
```

Task 1.1 和 1.2 可并行。Task 2.1 独立可选。Task 3.x 在所有对应实现完成后执行。

## 预期文件变更

| 文件 | 操作 | 行数估计 |
|------|------|---------|
| `front_project/app/src/components/Navbar.logic.vue` | 修改 ~6 行 | `applyMePayload` 内增加 URL tenant 检查 |
| `front_project/app/src/components/Sidebar.vue` | 修改 ~6 行 | `initData` 内增加 URL tenant 检查 |
| `Saas_project/accounts/serializers/user_serializer.py` | 修改 ~8 行 | `get_current_company` 增加 tenant_id filter |
