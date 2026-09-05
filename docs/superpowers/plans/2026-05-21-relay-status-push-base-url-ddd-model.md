# DDD Model: Relay Status Push Base URL

## 1) 需求边界

- 输入需求：修复 relay 状态上报目标 URL 选择错误，避免 `onlineServiceJS` 实际已启动但任务页显示“未启动”。
- 本次改动范围：`core/config` 配置选择逻辑 + 测试 + runbook。
- 结论：该需求属于应用配置连通性修复，不引入新的业务概念，不需要新增领域层模型文件。

## 2) 限界上下文 (Bounded Context)

- **Relay 启动与状态收敛上下文**：负责 `token-init -> start dispatch -> status converge` 过程状态机与事件。
- **容器 UI 上下文**：负责任务详情页容器可达性与 UI 快照读取。

本次需求影响“Relay 启动与状态收敛上下文”的上游配置选择，不改变上下文边界。

## 3) 实体与值对象

### 实体 (Entity)

- `RelayStartupSession`（聚合根，已有）
  - 职责：维护 relay 启动阶段、失败原因、最近 status seq 与快照。
  - 文件：`task2app/Saas_project/cloud/domain/entities/relay_startup_session.py`

### 值对象 (Value Object)

- `TaskScope`（已有）
  - 文件：`task2app/Saas_project/cloud/domain/value_objects/task_scope.py`
- `RelayStatusSnapshot`（已有）
  - 文件：`task2app/Saas_project/cloud/domain/value_objects/relay_status_snapshot.py`

本次需求不新增实体与值对象。

## 4) 聚合与聚合根

- 聚合：Relay 启动收敛聚合
- 聚合根：`RelayStartupSession`
- 一致性边界：
  - `token_initialized` 与 `phase` 迁移一致性
  - `last_status_seq` 单调不回退
  - `failure_reason` 与状态错误一致

本次需求不改变聚合规则，仅修复配置入口选择。

## 5) 领域服务

- `RelayTwoStepStartupService`（已有）
  - 文件：`task2app/Saas_project/cloud/domain/services/relay_two_step_startup_service.py`
  - 作用：编排 token-init、start 受理、status 收敛。

本次需求不新增领域服务。

## 6) 仓储接口

- `RelayStartupSessionRepository`（已有）
  - 文件：`task2app/Saas_project/cloud/domain/repositories/relay_startup_session_repository.py`
- `ContainerRuntimeContextRepository`（已有，邻域引用）
  - 文件：`task2app/Saas_project/cloud/domain/repositories/container_runtime_context_repository.py`

本次需求不新增仓储接口。

## 7) 领域事件

已有事件契约已覆盖本链路（如 `RelayStartAccepted`、`RelayStatusConverged`、`RelayTokenInit*` 等）。

- 事件导出：`task2app/Saas_project/cloud/domain/events/__init__.py`

本次需求不新增领域事件，仅复用现有事件流。

## 8) 领域模型文件生成结论

- 新增 `domain/` 文件：无
- 原因：本次是配置选择逻辑修复，不涉及新业务语义、聚合规则或跨上下文新契约。

## 9) Hard Gate 自检

- [x] 所有实体/值对象/仓储文件位于 `domain/` 目录下（复核已有文件）
- [x] 领域层无 ORM 导入（本次未新增 domain 代码）
- [x] 领域层无外部服务导入（本次未新增 domain 代码）
- [x] 仓储接口使用 ABC 抽象（现有接口满足）
- [x] 领域事件使用过去式命名（现有事件满足）
- [x] 每个聚合根有对应仓储接口（`RelayStartupSession` 已有）
- [x] 领域服务通过构造函数注入仓储依赖（`RelayTwoStepStartupService` 已有）
- [x] 无持久化字段声明渗入领域层（本次未新增 domain 代码）

---

领域模型复核完成：`RelayStartupSession` 聚合、`TaskScope/RelayStatusSnapshot` 值对象、`RelayTwoStepStartupService` 与相关事件契约均已就绪；本次需求无需新增领域层文件，可直接进入 TDD/回归验证阶段。
