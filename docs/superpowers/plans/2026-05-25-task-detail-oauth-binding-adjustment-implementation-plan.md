# Task Detail OAuth Binding Adjustment Implementation Plan (Row-level + repo_url)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development (recommended) or superpowers:executing-plans.  
> Steps use checkbox (`- [ ]`) syntax and are ordered by DDD dependency.

**Goal:** 将任务详情 OAuth 绑定收敛为“仓库行级动作 + `repo_url` 级判定”，并保持启动前预检契约稳定，避免“已绑定仓库仍显示按钮/未绑定仓库无入口”的回归。

**Architecture:** 先冻结领域层“行级绑定动作就绪态”模型（VO/聚合根/仓储接口/领域事件/领域服务），再实现基础设施仓储与应用层编排，最后接入前端行级显示与交互，最终用后端契约、前端单测、e2e 和 DDD 合规闭环。

**Tech Stack:** Python 3 (Django/DRF), Vue 3, pytest, Vitest, Playwright

---

## Skill Notice

I'm using the writing-plans skill to create the implementation plan.

## 输入基线

- design: `.cursor/plans/调整仓库授权方案_4e0a71a5.plan.md`
- value stream: `docs/superpowers/plans/2026-05-25-task-detail-oauth-binding-adjustment-value-stream.md`
- nfr: `docs/superpowers/plans/2026-05-25-task-detail-oauth-binding-adjustment-nfr-clarification.md`
- domain model artifacts:
  - `task2app/Saas_project/cloud/domain/value_objects/task_repo_oauth_connection_coverage.py`
  - `task2app/Saas_project/cloud/domain/entities/task_repo_oauth_row_action_readiness.py`
  - `task2app/Saas_project/cloud/domain/repositories/task_repo_oauth_row_action_repository.py`
  - `task2app/Saas_project/cloud/domain/events/task_repo_oauth_bind_action_required.py`
  - `task2app/Saas_project/cloud/domain/services/task_repo_oauth_row_action_service.py`

## 任务边界文件

- Domain
  - `task2app/Saas_project/cloud/domain/value_objects/task_repo_oauth_connection_coverage.py`
  - `task2app/Saas_project/cloud/domain/entities/task_repo_oauth_row_action_readiness.py`
  - `task2app/Saas_project/cloud/domain/repositories/task_repo_oauth_row_action_repository.py`
  - `task2app/Saas_project/cloud/domain/events/task_repo_oauth_bind_action_required.py`
  - `task2app/Saas_project/cloud/domain/services/task_repo_oauth_row_action_service.py`
- Infrastructure / Application
  - `task2app/Saas_project/cloud/infrastructure/repositories/`（新增 Django 仓储实现）
  - `task2app/Saas_project/accounts/github_app_views.py`
  - `task2app/Saas_project/projects/views/github_task_credential_views.py`（如需对齐 repo_url 语义）
- Frontend
  - `task2app/front_project/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue`
  - `task2app/front_project/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.test.js`
- Tests
  - `task2app/Saas_project/tests/domain/cloud/test_task_repo_oauth_row_action_domain_model.py`
  - `task2app/Saas_project/tests/test_github_task_repo_oauth_binding.py`
  - `task2app/front_project/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.test.js`
  - `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`

## 依赖顺序（必须遵守）

1. Domain layer（已建模，先锁契约）
2. Repository implementation（基础设施实现 domain/repositories ABC）
3. Application orchestration（视图/服务接入新仓储和规则）
4. Frontend row-level interaction（行级按钮显示与点击行为）
5. Regression + compliance

---

### Task 1: 冻结并校验领域契约（行级动作模型）

**Files:**
- Verify: `task2app/Saas_project/cloud/domain/**/task_repo_oauth_*`
- Verify: `task2app/Saas_project/tests/domain/cloud/test_task_repo_oauth_row_action_domain_model.py`

- [ ] **Step 1: 运行新增 domain 单测**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_task_repo_oauth_row_action_domain_model.py -q`
  - Expected: `5 passed`

- [ ] **Step 2: 运行既有 OAuth readiness domain 单测防回归**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_task_oauth_binding_readiness_domain_model.py -q`
  - Expected: PASS

- [ ] **Step 3: 审核 domain 导出入口一致性**
  - Check:
    - `cloud/domain/entities/__init__.py`
    - `cloud/domain/value_objects/__init__.py`
    - `cloud/domain/repositories/__init__.py`
    - `cloud/domain/services/__init__.py`
    - `cloud/domain/events/__init__.py`
  - Expected: 新增类全部导出，且无基础设施导入

---

### Task 2: 实现仓储接口的基础设施层（repo_url 视角）

**Files:**
- Create: `task2app/Saas_project/cloud/infrastructure/repositories/django_task_repo_oauth_row_action_repository.py`
- Modify: `task2app/Saas_project/cloud/infrastructure/repositories/__init__.py`
- Test: `task2app/Saas_project/tests/infrastructure/cloud/test_django_task_repo_oauth_row_action_repository.py`

- [ ] **Step 1: 先写失败测试（期望仓库集合 + 已 OAuth 连接集合）**
  - Run: `cd task2app/Saas_project && pytest tests/infrastructure/cloud/test_django_task_repo_oauth_row_action_repository.py -q`
  - Expected: 因实现缺失 FAIL

- [ ] **Step 2: 实现 Django 仓储（遵循 domain ABC）**
  - Requirement:
    - 实现 `list_expected_repo_urls(scope)`
    - 实现 `list_oauth_connected_repo_urls(scope, actor_id)`
    - 按 `repo_url` 返回 tuple[str, ...]，不在 domain 层暴露 ORM 细节

- [ ] **Step 3: 运行仓储测试确认通过**
  - Run: `cd task2app/Saas_project && pytest tests/infrastructure/cloud/test_django_task_repo_oauth_row_action_repository.py -q`
  - Expected: PASS

---

### Task 3: 应用层接入行级判定服务（保持 API 兼容）

**Files:**
- Modify: `task2app/Saas_project/accounts/github_app_views.py`
- Modify: `task2app/Saas_project/projects/views/github_task_credential_views.py`
- Add/Modify tests:
  - `task2app/Saas_project/tests/test_github_task_repo_oauth_binding.py`
  - `task2app/Saas_project/tests/test_github_app_start_redirect_uri.py`（如涉及 start 参数语义）

- [ ] **Step 1: 增加失败测试（repo_url 判定与 provider_key/connection 对齐）**
  - Run: `cd task2app/Saas_project && pytest tests/test_github_task_repo_oauth_binding.py -k repo_url -v`
  - Expected: 新语义断言 FAIL

- [ ] **Step 2: 实现最小接入**
  - Requirement:
    - 连接状态查询支持 `repo_url` 语义
    - 不破坏原有无 `repo_url` 调用路径（向后兼容）
    - 错误契约保持稳定

- [ ] **Step 3: 回归关键后端测试**
  - Run: `cd task2app/Saas_project && pytest tests/test_github_task_repo_oauth_binding.py -v`
  - Expected: PASS

---

### Task 4: 前端行级按钮与 repo_url 级可见性闭环

**Files:**
- Modify: `task2app/front_project/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue`
- Modify: `task2app/front_project/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.test.js`

- [ ] **Step 1: 先写失败用例（同页已绑定/未绑定混合）**
  - Case:
    - 仅未绑定仓库行显示 `OAuth 绑定`
    - 点击行级按钮仅触发对应 `repo_url` 的检查与跳转

- [ ] **Step 2: 运行前端测试确认失败**
  - Run: `cd task2app/front_project/app && npm run test -- "src/components/task-detail/TaskDetailLinkedProjectsPanel.test.js"`
  - Expected: 新增断言 FAIL

- [ ] **Step 3: 实现最小前端逻辑**
  - Requirement:
    - 移除/禁用全局入口
    - 按 `repo_url` 管理按钮可见性与 loading
    - 已绑定行不显示按钮

- [ ] **Step 4: 运行前端测试确认通过**
  - Run: `cd task2app/front_project/app && npm run test -- "src/components/task-detail/TaskDetailLinkedProjectsPanel.test.js"`
  - Expected: PASS

---

### Task 5: 预检与直启链路回归（用户价值闭环）

**Files:**
- Verify/Modify: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`
- Verify/Modify: `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`

- [ ] **Step 1: 后端预检契约回归**
  - Run: `cd task2app/Saas_project && pytest tests/test_relay_to_trae_proxy.py -k precheck -v`
  - Expected: `missing_repo_credentials` 结构与错误码稳定

- [ ] **Step 2: e2e 场景回归**
  - Run: `cd task2app/playwright && npx playwright test -c front_project/playwright.config.js front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js --grep "预检失败时应阻断 start"`
  - Expected: PASS（阻断 start + 引导可见）

---

### Task 6: 合规与发布前验证

- [ ] **Step 1: 运行 DDD/BDD 合规检查**
  - Run: `cd task2app && python scripts/ci/check_ddd_bdd_compliance.py`
  - Expected: `DDD/BDD 合规检查通过。`

- [ ] **Step 2: 聚合验证命令**
  - Run:
    - `cd task2app/Saas_project && pytest tests/domain/cloud/test_task_repo_oauth_row_action_domain_model.py tests/test_github_task_repo_oauth_binding.py -q`
    - `cd task2app/front_project/app && npm run test -- "src/components/task-detail/TaskDetailLinkedProjectsPanel.test.js"`
  - Expected: 全部 PASS

---

## 自检清单（计划质量）

- [ ] 任务顺序满足 DDD 依赖（domain → infra repo → app → frontend → regression）
- [ ] 每个任务都有明确文件路径与命令
- [ ] 所有变更都可被自动化验证（pytest/vitest/playwright/compliance）
- [ ] `repo_url` 级判定在后端与前端语义一致
- [ ] 无占位符（TBD/TODO）

## 与 value increments 对齐

- Increment 1（仓库行级 OAuth 入口薄切片）→ Task 1 + Task 2 + Task 4
- Increment 2（启动前预检可视化引导）→ Task 3 + Task 5
- Increment 3（契约稳定性回归保护）→ Task 6
