# Task Detail API DDD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 基于已批准 value stream，完成 task-detail 主链路在 `cloud` 模块的 DDD 分层收敛，并保证 `task-detail-thin-slice`、`container-runtime-context`、`relay-status-convergence` 三个步骤可验证通过。

**Architecture:** 先锁定领域契约（事件、仓储接口、领域服务与不变量），再落地应用层编排与接口层接入，最后补齐基础设施实现和装配。领域层持续保持纯业务语义，不引入 ORM/HTTP SDK；基础设施依赖只能出现在 infrastructure 或服务适配层。

**Tech Stack:** Python 3, Django/DRF, pytest, DDD layered architecture (`domain` / `services` / `views` / `infrastructure`)

---

## Skill Notice

I'm using the writing-plans skill to create the implementation plan.

## 输入基线

- value stream: `docs/superpowers/plans/2026-05-21-task-detail-api-ddd-value-stream.md`
- ddd design: `docs/superpowers/specs/2026-05-21-task-detail-api-ddd-design.md`
- stream registry: `value-stream.yaml` (`task-detail-runtime-relay`)
- 当前代码现状（已存在）：
  - 领域仓储接口：`task2app/Saas_project/cloud/domain/repositories/container_runtime_context_repository.py`
  - 领域事件：`task2app/Saas_project/cloud/domain/events/container_ui_context_refreshed.py`
  - 应用服务：`task2app/Saas_project/cloud/services/task_container_runtime_context_app_service.py`
  - 视图接入：`task2app/Saas_project/cloud/services/get_container_task_ui_context.py`
  - relay 收敛：`task2app/Saas_project/cloud/services/relay_to_trae_status.py`

## 依赖顺序（必须遵守）

1. **Domain 阶段（先做）**：事件、仓储接口、领域服务、不变量、领域测试  
2. **Application/Interface 阶段（后做）**：应用服务编排、view/service 接口接入、value stream 对应 API 行为  
3. **Infrastructure 阶段（最后）**：Django ORM 仓储实现、装配与依赖注入、回归验证

## DDD 约束 Hard Gates

- [ ] 仓储接口与领域事件先于实现层改动提交（Domain commit 在前）
- [ ] `cloud/domain/**` 不导入 `cloud.models`、`django.*`、`requests`、外部 SDK
- [ ] 基础设施实现仅依赖领域接口，不反向污染领域层
- [ ] 应用服务只依赖领域抽象（接口/值对象/领域服务），不直接承载业务规则

## DDD 结构校验（/4-plans 要求）

- [x] **分层结构符合要求**：计划明确区分 `domain`、`services/views`（应用/接口）与 `infrastructure`。
- [x] **任务顺序符合要求**：Task 1-2 先冻结领域契约与不变量，Task 3-4 再接应用/接口层，Task 5 最后处理基础设施下沉。
- [x] **接口先于实现**：`ContainerRuntimeContextRepository`、领域事件在 Domain 阶段先定义/收敛，`DjangoContainerRuntimeContextRepository` 在 Infrastructure 阶段后置。
- [x] **端到端可验证**：Task 6 对应 value stream 三个 step 的测试入口与字段观测点，满足增量可验证。

---

### Task 1: Domain Contract Freeze（事件+仓储接口）

**Files:**
- Modify: `task2app/Saas_project/cloud/domain/events/task_detail_patched.py`
- Modify: `task2app/Saas_project/cloud/domain/events/container_ui_context_refreshed.py`
- Modify: `task2app/Saas_project/cloud/domain/events/__init__.py`
- Modify: `task2app/Saas_project/cloud/domain/repositories/container_runtime_context_repository.py`
- Test: `task2app/Saas_project/tests/domain/cloud/test_task_detail_runtime_context_domain_model.py`

- [ ] **Step 1: 写失败测试（领域事件与仓储契约一致）**
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/domain/cloud/test_task_detail_runtime_context_domain_model.py -k "task_detail_patched or refreshed_event or repository" -v`
  - Expected: 断言失败（字段归一化或接口契约不满足）
- [ ] **Step 3: 最小实现领域契约**
  - `TaskDetailPatched` 统一 `patched_fields` 归一化规则
  - `ContainerUiContextRefreshed` 字段保持最小必要载荷
  - `ContainerRuntimeContextRepository.find_latest_by_scope(scope)` 保持纯抽象接口
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/domain/cloud/test_task_detail_runtime_context_domain_model.py -k "task_detail_patched or refreshed_event or repository" -v`
  - Expected: PASS
- [ ] **Step 5: Commit（Domain first）**
  - Run: `git add task2app/Saas_project/cloud/domain/events/task_detail_patched.py task2app/Saas_project/cloud/domain/events/container_ui_context_refreshed.py task2app/Saas_project/cloud/domain/events/__init__.py task2app/Saas_project/cloud/domain/repositories/container_runtime_context_repository.py task2app/Saas_project/tests/domain/cloud/test_task_detail_runtime_context_domain_model.py && git commit -m "feat: freeze task-detail runtime domain contracts"`

### Task 2: Domain Invariants for Relay Convergence

**Files:**
- Modify: `task2app/Saas_project/cloud/domain/entities/relay_startup_session.py`
- Modify: `task2app/Saas_project/cloud/domain/services/relay_two_step_startup_service.py`
- Modify: `task2app/Saas_project/cloud/domain/value_objects/relay_status_snapshot.py`
- Test: `task2app/Saas_project/tests/domain/cloud/test_relay_two_step_startup_domain_model.py`

- [ ] **Step 1: 写失败测试（负序列号拒绝、状态收敛不变量）**
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/domain/cloud/test_relay_two_step_startup_domain_model.py -k "negative_seq or converge or token_init" -v`
  - Expected: 至少一个不变量断言失败
- [ ] **Step 3: 最小实现不变量**
  - `seq >= 0`
  - token-init 完成后才能 accept-start
  - status convergence 仅更新聚合状态，不写基础设施
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/domain/cloud/test_relay_two_step_startup_domain_model.py -k "negative_seq or converge or token_init" -v`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add task2app/Saas_project/cloud/domain/entities/relay_startup_session.py task2app/Saas_project/cloud/domain/services/relay_two_step_startup_service.py task2app/Saas_project/cloud/domain/value_objects/relay_status_snapshot.py task2app/Saas_project/tests/domain/cloud/test_relay_two_step_startup_domain_model.py && git commit -m "feat: enforce relay convergence domain invariants"`

### Task 3: Application Service for Container Runtime Context

**Files:**
- Modify: `task2app/Saas_project/cloud/services/task_container_runtime_context_app_service.py`
- Modify: `task2app/Saas_project/cloud/services/get_container_task_ui_context.py`
- Test: `task2app/Saas_project/tests/domain/cloud/test_task_detail_runtime_context_domain_model.py`
- Test: `task2app/Saas_project/tests/test_ai_task_comment.py`

- [ ] **Step 1: 写失败测试（应用服务 payload 与快照契约一致）**
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/domain/cloud/test_task_detail_runtime_context_domain_model.py tests/test_ai_task_comment.py -k "get_task_ui_context or container_task_ui_context" -v`
  - Expected: payload 字段或回退逻辑断言失败
- [ ] **Step 3: 最小实现应用层编排**
  - 应用服务通过 `ContainerRuntimeContextRepository` 读取快照
  - 调用 `TaskContainerRuntimeService.refresh_context()`
  - 输出兼容字段：`status/has_server_config/container_endpoint_registered/container_page_url/container_vscode_url`
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/domain/cloud/test_task_detail_runtime_context_domain_model.py tests/test_ai_task_comment.py -k "get_task_ui_context or container_task_ui_context" -v`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add task2app/Saas_project/cloud/services/task_container_runtime_context_app_service.py task2app/Saas_project/cloud/services/get_container_task_ui_context.py task2app/Saas_project/tests/domain/cloud/test_task_detail_runtime_context_domain_model.py task2app/Saas_project/tests/test_ai_task_comment.py && git commit -m "feat: stabilize task container runtime application payload"`

### Task 4: Interface Integration for Relay Status Convergence

**Files:**
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_status.py`
- Modify: `task2app/Saas_project/cloud/services/relay_startup_app_service.py`
- Test: `task2app/Saas_project/cloud/view_test/test_relay_token_audit_status_push_view.py`

- [ ] **Step 1: 写失败测试（status-push 收敛与审计链一致）**
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest cloud/view_test/test_relay_token_audit_status_push_view.py -k "status_push_ok or invalid_token" -v`
  - Expected: `event_type/seq/trace_id/error_code` 断言失败
- [ ] **Step 3: 最小实现接口层收敛**
  - `relay_to_trae_status` 调用共享 startup app service
  - status push 成功与失败路径都写审计事件
  - 不改变外部协议，仅调整内部收敛路径
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest cloud/view_test/test_relay_token_audit_status_push_view.py -k "status_push_ok or invalid_token" -v`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add task2app/Saas_project/cloud/services/relay_to_trae_status.py task2app/Saas_project/cloud/services/relay_startup_app_service.py task2app/Saas_project/cloud/view_test/test_relay_token_audit_status_push_view.py && git commit -m "feat: converge relay status push through shared app service"`

### Task 5: Infrastructure Repository Extraction（从 services 中下沉实现）

**Files:**
- Create: `task2app/Saas_project/cloud/infrastructure/__init__.py`
- Create: `task2app/Saas_project/cloud/infrastructure/repositories/__init__.py`
- Create: `task2app/Saas_project/cloud/infrastructure/repositories/django_container_runtime_context_repository.py`
- Modify: `task2app/Saas_project/cloud/services/task_container_runtime_context_app_service.py`
- Modify: `task2app/Saas_project/cloud/services/get_container_task_ui_context.py`
- Test: `task2app/Saas_project/tests/test_ai_task_comment.py`

- [ ] **Step 1: 写失败测试（装配改为 infrastructure 仓储实现后行为不变）**
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_ai_task_comment.py -k "container_task_ui_context" -v`
  - Expected: 导入路径/装配失败或行为断言失败
- [ ] **Step 3: 最小实现基础设施下沉**
  - 将 `DjangoContainerRuntimeContextRepository` 从 `services` 提取到 `cloud/infrastructure/repositories`
  - 应用服务仅保留接口编排，不直接持有 ORM 查询实现
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_ai_task_comment.py -k "container_task_ui_context" -v`
  - Expected: PASS
- [ ] **Step 5: Commit（Infrastructure last）**
  - Run: `git add task2app/Saas_project/cloud/infrastructure/__init__.py task2app/Saas_project/cloud/infrastructure/repositories/__init__.py task2app/Saas_project/cloud/infrastructure/repositories/django_container_runtime_context_repository.py task2app/Saas_project/cloud/services/task_container_runtime_context_app_service.py task2app/Saas_project/cloud/services/get_container_task_ui_context.py task2app/Saas_project/tests/test_ai_task_comment.py && git commit -m "refactor: move runtime context repository to infrastructure layer"`

### Task 6: Value Stream Step Verification（task-detail-runtime-relay）

**Files:**
- Test: `task2app/Saas_project/projects/view_test/TodoViewSet_test.py`
- Test: `task2app/Saas_project/tests/test_ai_task_comment.py`
- Test: `task2app/Saas_project/cloud/view_test/test_relay_token_audit_status_push_view.py`

- [ ] **Step 1: 验证 step1 `task-detail-thin-slice`**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest projects/view_test/TodoViewSet_test.py -k "todo and detail" -v`
  - Expected: 涉及 `progress_column_id/title/updated_at` 的断言通过
- [ ] **Step 2: 验证 step2 `container-runtime-context`**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_ai_task_comment.py -k "container_task_ui_context" -v`
  - Expected: `server_url/business_api_endpoint/container_vscode_url/container_access_token` 投影相关断言通过
- [ ] **Step 3: 验证 step3 `relay-status-convergence`**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest cloud/view_test/test_relay_token_audit_status_push_view.py -k "status_push" -v`
  - Expected: `event_type/seq/trace_id/error_code` 相关断言通过
- [ ] **Step 4: 执行最小回归组合**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/domain/cloud/test_task_detail_runtime_context_domain_model.py tests/domain/cloud/test_relay_two_step_startup_domain_model.py tests/test_ai_task_comment.py cloud/view_test/test_relay_token_audit_status_push_view.py projects/view_test/TodoViewSet_test.py -v`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add -A && git commit -m "test: verify task-detail runtime relay value stream end-to-end"`

---

## 执行前后校验清单

- [ ] 每次进入新阶段前检查上一阶段提交已完成（Domain -> App/Interface -> Infrastructure）
- [ ] 执行 `rg "from cloud\\.models|import django" task2app/Saas_project/cloud/domain` 结果为空
- [ ] `task2app/Saas_project/cloud/domain/repositories/*` 仅含 ABC 接口，无 ORM 查询
- [ ] 任何基础设施实现文件位于 `task2app/Saas_project/cloud/infrastructure/**`
- [ ] `value-stream.yaml` 中 `task-detail-runtime-relay` 三个步骤至少各有一个对应用例通过

## 与 value stream 对齐说明

- `task-detail-thin-slice`：由 `TodoViewSet_test.py` + `TaskDetailPatched` 契约与主链路回归覆盖。
- `container-runtime-context`：由 `TaskContainerRuntimeContextAppService` + `test_ai_task_comment.py` 容器上下文用例覆盖。
- `relay-status-convergence`：由 `relay_to_trae_status.py` 收敛路径 + `test_relay_token_audit_status_push_view.py` 覆盖。

