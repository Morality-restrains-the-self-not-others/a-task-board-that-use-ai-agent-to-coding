# relayToTrae Token 审计事件流 Implementation Plan

> **For agentic workers:** 推荐使用 `subagent-driven-development` 或 `executing-plans` 按任务逐项执行。本文使用 `- [ ]` 可勾选任务格式。  
> **Derived from value stream:** `docs/superpowers/plans/2026-05-20-relay-token-audit-events-value-stream.md`  
> **Spec:** `docs/superpowers/specs/2026-05-20-relay-token-audit-events-design.md`

**Goal:** 建立 append-only token 审计事件流，在不落明文 token 的前提下，支持定位 relayToTrae 401 是否由旧 token 引发。

**Architecture (DDD-first):**
- 领域层先定义：`Entity / ValueObject / Repository Interface / Domain Service / Domain Events`
- 基础设施后实现：Django Model + Repository Impl + Migration
- 应用层最后接入：token 生命周期、relay register/start、status-push 成功/401

**Tech Stack:** Django 4.x + DRF + pytest

---

## Task 1: 领域层契约（先行，禁止基础设施依赖）

**Files:**
- Create: `task2app/Saas_project/cloud/domain/entities/container_token_audit_event.py`
- Create: `task2app/Saas_project/cloud/domain/value_objects/token_digest.py`
- Create: `task2app/Saas_project/cloud/domain/repositories/container_token_audit_event_repository.py`
- Create: `task2app/Saas_project/cloud/domain/services/container_token_audit_service.py`
- Create: `task2app/Saas_project/cloud/domain/events/token_audit_events.py`
- Modify: `task2app/Saas_project/cloud/domain/entities/__init__.py`
- Modify: `task2app/Saas_project/cloud/domain/value_objects/__init__.py`
- Modify: `task2app/Saas_project/cloud/domain/repositories/__init__.py`
- Modify: `task2app/Saas_project/cloud/domain/services/__init__.py`
- Modify: `task2app/Saas_project/cloud/domain/events/__init__.py`
- Create: `task2app/Saas_project/tests/domain/cloud/test_container_token_audit_domain_model.py`

- [ ] **Step 1: 写领域失败测试（纯 Python，无 ORM）**
  - 断言 `TokenDigest` 仅接受 hash/suffix，拒绝明文 token。
  - 断言 `ContainerTokenAuditEvent` 必填上下文与 `event_type`。
  - 断言 Repository interface 为 ABC 抽象。

- [ ] **Step 2: 运行测试确认失败**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/domain/cloud/test_container_token_audit_domain_model.py -v
```
Expected: FAIL（缺少领域模型）

- [ ] **Step 3: 实现领域模型与接口**
  - `container_token_audit_event.py`：实体只包含业务字段与校验，不含 ORM 字段声明。
  - `token_digest.py`：值对象封装 `sha256` 与 `suffix` 规则。
  - `container_token_audit_event_repository.py`：定义 `append()`、`list_by_task()` 接口。
  - `container_token_audit_service.py`：统一构造审计事件命令对象。

- [ ] **Step 4: 再跑领域测试**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/domain/cloud/test_container_token_audit_domain_model.py -v
```
Expected: PASS

---

## Task 2: 基础设施层（审计表 + Repository 实现）

**Files:**
- Create: `task2app/Saas_project/cloud/models/container_token_audit_event.py`
- Modify: `task2app/Saas_project/cloud/models/__init__.py`
- Create: `task2app/Saas_project/cloud/infrastructure/persistence/django_container_token_audit_event_repository.py`
- Modify: `task2app/Saas_project/cloud/infrastructure/persistence/__init__.py`
- Create: `task2app/Saas_project/cloud/migrations/0044_container_token_audit_event.py`（编号按本地实际顺延）
- Create: `task2app/Saas_project/tests/infrastructure/test_django_container_token_audit_event_repository.py`

- [ ] **Step 1: 写持久化失败测试**
  - 覆盖 append 成功、按 task+时间逆序查询、字段截断策略（error_detail）。
  - 覆盖不存明文 token（仅 hash/suffix）。

- [ ] **Step 2: 运行测试确认失败**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/infrastructure/test_django_container_token_audit_event_repository.py -v
```
Expected: FAIL（模型/仓储实现不存在）

- [ ] **Step 3: 实现模型与仓储**
  - 模型字段对齐设计：`tenant_id/workspace_id/task_id/event_type/.../created_at`
  - 增加索引：`(task_id, created_at)`、`(event_type, created_at)`、`trace_id`
  - 仓储实现只做映射，不放业务决策。

- [ ] **Step 4: 生成并检查 migration**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- python manage.py makemigrations cloud
../activate_env.sh unit -- python manage.py sqlmigrate cloud 0044
```
Expected: SQL 包含新表与索引

- [ ] **Step 5: 跑持久化测试**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/infrastructure/test_django_container_token_audit_event_repository.py -v
```
Expected: PASS

---

## Task 3: 应用层接入（生命周期 + relay + status-push）

**Files:**
- Modify: `task2app/Saas_project/cloud/services/server_config_container_tokens.py`
- Modify: `task2app/Saas_project/cloud/views/container_runtime_token_views.py`
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py`
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_status.py`
- Create: `task2app/Saas_project/cloud/services/container_token_audit_app_service.py`
- Create: `task2app/Saas_project/tests/test_container_token_audit_integration.py`

- [ ] **Step 1: 写集成失败测试（按事件类型断言）**
  - `bootstrap` / `exchange_refresh` / `refresh_access`
  - `relay_register` / `relay_start`
  - `status_push_ok` / `status_push_401_invalid_token`

- [ ] **Step 2: 运行测试确认失败**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_container_token_audit_integration.py -v
```
Expected: FAIL（尚未接入写审计）

- [ ] **Step 3: 实现接入**
  - 应用服务统一接收上下文与 token 原值，内部仅计算 hash/suffix 后入库。
  - 401 路径必须写审计（即使 token 无法匹配 CloudServerConfig）。
  - 统一 `source_component`（`django`/`relay`）与 `error_code`（如 `401`）。

- [ ] **Step 4: 回归现有 relay 相关测试**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py tests/test_relay_to_trae_status.py -v
```
Expected: PASS（无行为回退）

- [ ] **Step 5: 跑新增集成测试**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_container_token_audit_integration.py -v
```
Expected: PASS

---

## Task 4: 时间线查询与运维清理

**Files:**
- Create: `task2app/Saas_project/cloud/views/container_token_audit_views.py`
- Modify: `task2app/Saas_project/cloud/views/cloud_compute_views.py`（挂调试查询 action）
- Create: `task2app/Saas_project/cloud/management/commands/cleanup_container_token_audit_events.py`
- Create: `task2app/Saas_project/tests/test_container_token_audit_timeline_api.py`
- Create: `task2app/Saas_project/tests/test_container_token_audit_retention.py`

- [ ] **Step 1: 写 API/清理失败测试**
  - 查询接口支持 `task_id + since + limit`
  - 返回不含明文 token
  - 清理命令按保留期删除旧事件

- [ ] **Step 2: 运行测试确认失败**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_container_token_audit_timeline_api.py tests/test_container_token_audit_retention.py -v
```
Expected: FAIL

- [ ] **Step 3: 实现查询与清理**
  - 查询接口用于排障（可先放在内部调试 action）。
  - 清理命令参数示例：`--days 90`。

- [ ] **Step 4: 跑测试通过**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_container_token_audit_timeline_api.py tests/test_container_token_audit_retention.py -v
```
Expected: PASS

---

## Task 5: 全量验证与价值流对齐

- [ ] **Step 1: 运行本次相关测试合集**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest \
  tests/domain/cloud/test_container_token_audit_domain_model.py \
  tests/infrastructure/test_django_container_token_audit_event_repository.py \
  tests/test_container_token_audit_integration.py \
  tests/test_container_token_audit_timeline_api.py \
  tests/test_container_token_audit_retention.py -v
```

- [ ] **Step 2: 运行 DDD/BDD 合规脚本**
```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
python3 scripts/ci/check_ddd_bdd_compliance.py
```

- [ ] **Step 3: 验证 valueStream 配置可加载**
```bash
cd valueStream
go test ./...
```

---

## DDD Gate Self-Review

- [ ] 领域层先于基础设施层实现（Task 1 在 Task 2 前）
- [ ] `domain/` 无 ORM / HTTP / requests / boto3 依赖
- [ ] 仓储接口先定义于 `domain/repositories`，实现后置于 `infrastructure/persistence`
- [ ] 领域事件命名为过去式
- [ ] 应用层不落明文 token，仅传递 hash/suffix

---

## Spec Coverage Matrix

| 设计要求 | 对应任务 |
|---|---|
| append-only 审计表 | Task 2 |
| hash/suffix 替代明文 | Task 1, 2, 3 |
| 生命周期事件覆盖 | Task 3 |
| relay 使用点覆盖 | Task 3 |
| status-push 401 必落审计 | Task 3 |
| 时间线回放 | Task 4 |
| 保留策略与清理 | Task 4 |

