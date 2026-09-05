# /5-ddd 领域模型: runAll 稳定性优先

## 输入

- Value Stream: `docs/superpowers/plans/2026-05-25-runall-stability-first-value-stream.md`
- NFR: `docs/superpowers/plans/2026-05-25-runall-stability-first-nfr-clarification.md`
- 设计: `docs/superpowers/specs/2026-05-25-runall-stability-first-design.md`

## 限界上下文 (Bounded Context)

- `orchestration-lifecycle`：编排生命周期、所有权约束、阶段化状态机。
- `service-runtime-observability`：失败分类、修复提示、会话追踪与可观测输出。

## 实体与值对象

### 实体

- `ManagedService`（已有）：受管服务根信息（name/group/status/depends_on）。
- `StartupSession`（新增）：一次 runAll 编排会话，持有 config 指纹与会话状态。
- `ServiceStartupAttempt`（新增）：会话内某服务某阶段的一次尝试结果。

### 值对象

- `ServiceOwnership`：`service + owner_session + pid + config_hash` 的所有权快照。
- `ServiceLifecyclePhase`：`preflight/launch/readiness/completed`。
- `ServiceFailureCode`：稳定失败分类（`PRECHECK_*`, `LAUNCH_*`, `READINESS_*`）。
- `ConfigFingerprint`：生效配置路径与哈希。

## 聚合与聚合根

- `ManagedServiceAggregate`（聚合根：`ManagedService`）
  - 一致性边界：服务状态 + 所有权校验 + stop/restart 权限约束
  - 关键不变量：单服务同一时刻仅允许一个 owner session
- `StartupSessionAggregate`（聚合根：`StartupSession`）
  - 一致性边界：一次会话内阶段推进与失败分类
  - 关键不变量：失败必须具备 `phase + failure_code + hint`

## 领域服务

- `ServiceOwnershipGuardService`
  - 职责：校验 actor session 对目标服务是否可操作
  - 输入：`service_name`, `actor_session_id`
  - 输出：可操作/拒绝（要求显式 takeover）

## 仓储接口 (Repository)

- `ServiceRuntimeContextRepository`（已有）
- `ServiceOwnershipRepository`（新增）
- `StartupSessionRepository`（新增）
- `ServiceStartupAttemptRepository`（新增）

## 领域事件 (Domain Events)

- `ServiceOwnershipAcquired`
- `ServiceStartRejectedByForeignProcess`
- `ServicePreflightFailed`
- `ServiceReadinessFailed`
- `ServiceBecameHealthy`

以上事件均为过去式，作为上下文间解耦契约。

## 代码骨架落点

- `runAll/src/domain/startup_session_entity.go`
- `runAll/src/domain/service_startup_attempt_entity.go`
- `runAll/src/domain/service_ownership_value_object.go`
- `runAll/src/domain/service_lifecycle_phase_value_object.go`
- `runAll/src/domain/service_failure_code_value_object.go`
- `runAll/src/domain/config_fingerprint_value_object.go`
- `runAll/src/domain/service_ownership_guard_service.go`
- `runAll/src/domain/service_ownership_repository.go`
- `runAll/src/domain/startup_session_repository.go`
- `runAll/src/domain/service_lifecycle_domain_events.go`

## 自检 (Hard Gate)

- [x] 实体/值对象/仓储位于 `runAll/src/domain/`
- [x] 领域层无 ORM 导入
- [x] 领域层无外部服务 SDK/HTTP 客户端导入
- [x] 仓储为抽象接口，不含持久化实现
- [x] 领域事件采用过去式命名
- [x] 聚合根均有对应仓储接口
- [x] 领域服务通过构造函数注入仓储依赖
- [x] 无数据库字段声明细节

---

领域模型已生成到 `runAll/src/domain/`。仓储接口和领域服务接口已就绪，可进入 `/6-plans-实施计划`。
