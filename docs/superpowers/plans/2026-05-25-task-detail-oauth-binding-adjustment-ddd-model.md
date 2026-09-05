# DDD Model: 任务详情 OAuth 绑定方案调整

> 输入：
> - `docs/superpowers/plans/2026-05-25-task-detail-oauth-binding-adjustment-value-stream.md`
> - `docs/superpowers/plans/2026-05-25-task-detail-oauth-binding-adjustment-nfr-clarification.md`

## 限界上下文

- **账号 OAuth 上下文（accounts/gitOauth）**：提供账号级 OAuth 身份与 refresh token 生命周期（本次不改动核心发证流程）。
- **任务仓库授权上下文（cloud/task-detail）**：维护任务下仓库授权覆盖率与“是否可启动”判定。
- **relay 启动编排上下文（cloud/relayToTrae）**：消费预检结果，决定启动阻断或展示引导。

## 实体与值对象

- **实体（聚合根）**
  - `TaskOauthBindingReadiness`：表示某个 `task_scope + actor` 在某时刻的 OAuth 仓库绑定就绪态。
- **值对象**
  - `TaskOauthBindingCoverage`：期望仓库集合与已授权仓库集合的覆盖快照，负责缺失集合计算。

## 聚合与聚合根

- 聚合：`TaskOauthBindingReadiness` 聚合
  - 聚合根：`TaskOauthBindingReadiness`
  - 包含值对象：`TaskOauthBindingCoverage`
  - 一致性边界：单次预检判定内的覆盖率一致性（L2，读己之写，允许短暂最终一致）

## 领域服务

- `TaskOauthBindingReadinessService`
  - 职责：基于 repo 集合评估就绪态；从仓储读取后统一产出就绪实体与引导事件。
  - 依赖注入：`TaskOauthBindingReadinessRepository`（构造函数注入）

## 仓储接口（ABC）

- `TaskOauthBindingReadinessRepository`
  - `list_expected_repo_urls(scope)`
  - `list_authorized_repo_urls(scope, actor_id)`

## 领域事件

- `TaskOauthBindingGuidanceRequired`（过去式语义：已判定需要引导）
  - 载荷：`scope`、`actor_id`、`trace_id`、`missing_repo_urls`、`occurred_at`
  - 用途：为应用层和前端提供结构化“为何不能启动”的契约

## NFR 驱动的建模决策映射

- 一致性 L2 → 小聚合，不做跨上下文强一致事务。
- 安全性 L2 → 领域服务显式接收 `actor_id + scope`，避免隐式上下文。
- 可观测性 L2 → 引导事件必须携带 `trace_id + missing_repo_urls`。
- 可维护性 L2 → 通过仓储接口抽象读取来源，领域层不依赖 ORM/HTTP/SDK。

## 生成的领域文件

- `task2app/Saas_project/cloud/domain/entities/task_oauth_binding_readiness.py`
- `task2app/Saas_project/cloud/domain/value_objects/task_oauth_binding_coverage.py`
- `task2app/Saas_project/cloud/domain/services/task_oauth_binding_readiness_service.py`
- `task2app/Saas_project/cloud/domain/repositories/task_oauth_binding_readiness_repository.py`
- `task2app/Saas_project/cloud/domain/events/task_oauth_binding_guidance_required.py`
