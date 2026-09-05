# /5-ddd 领域模型: runAll 链式启停（Cascade Lifecycle）

## 输入

- Value Stream: `docs/superpowers/plans/2026-05-27-runall-cascade-lifecycle-value-stream.md`
- NFR: `docs/superpowers/plans/2026-05-27-runall-cascade-lifecycle-nfr-clarification.md`
- 设计: `docs/superpowers/specs/2026-05-27-runall-cascade-lifecycle-design.md`

## 限界上下文 (Bounded Context)

- **`service-orchestration`（延续并扩展）**：在单进程 runAll 内编排受管服务的启停顺序；本增量新增「传递闭包 + 拓扑计划」能力，与既有 `ManagedService`、`ServiceStopPolicyService` 共存。
- **不包含**：业务 SaaS 域、valueStream 测试编排域。

## 实体与值对象

### 实体（复用，无新增）

- `ManagedService`：单服务生命周期状态（`stopped` / `healthy` 等）。
- `StartupSession`：runAll 会话与 ownership（链式每步复用，不扩展字段）。

### 值对象（新增）

| 值对象 | 职责 |
|--------|------|
| `ServiceTopologyNode` | 配置图中的节点：`Name`、`GroupName`、`DependsOn` |
| `ServiceLifecyclePlan` | 不可变有序计划：`Operation`（start/stop）+ `OrderedNames` |
| `CascadeExecutionReport` | 失败时快照：`Completed`、`FailedAt` |
| `CascadePlanRequest` | 计划输入：`TargetName` 或 `GroupName` + 运行时状态快照 |

### 值对象（复用）

- `ServiceOwnership`、`ServiceFailureCode`：链中失败时 hint 分类（应用层组装）。

## 聚合与聚合根

本增量**不引入新聚合根**。理由（NFR 一致性 L2 + 单进程内存状态）：

- 链式计划为 **无副作用的纯计算**（值对象 + 领域服务）。
- 执行仍逐步委托既有 `StartServiceWithActor` / `StopServiceWithActor`，状态突变留在 `StatusStore`（基础设施/应用层）。

| 既有聚合 | 本增量关系 |
|----------|------------|
| `ManagedService`（逻辑聚合根） | 计划生成时只读 `Status` 过滤；执行阶段仍由 Runner CAS 更新 |
| `StartupSession` | 链式操作绑定同一 `actorSessionID` |

## 领域服务

### `ServiceCascadeOrchestrationService`（新增）

依赖：`ServiceTopologyRepository`、`ServiceRuntimeContextRepository`、可选 `ServiceStopPolicyService`（stop 计划过滤阻塞下游）。

| 方法 | 说明 |
|------|------|
| `PlanStartCascade(targetName)` | 传递上游闭包 → 正序拓扑 → 仅 `stopped` |
| `PlanStopCascade(targetName)` | 传递下游闭包 → 逆序拓扑 → 仅阻塞状态下游 + 目标 |
| `PlanStartGroup(groupName)` | 组内服务 + 必要组外 stopped 上游 → 正序拓扑 |

**不变量：**

- 计划边集为全图 `depends_on` 的子集，顺序尊重 DAG。
- 启动计划不包含已 `healthy` 节点（幂等跳过）。
- 停止计划在 `cascade=false` 路径不经过本服务（应用层分支）。

### `ServiceStopPolicyService`（延续）

`PlanStopCascade` 使用 `isBlockingStatus` 与 `EvaluateStop` 语义一致，避免重复定义「活跃下游」。

## 仓储接口 (Repository)

| 接口 | 职责 |
|------|------|
| `ServiceTopologyRepository`（新增） | 提供配置 DAG：`ListAll`、`FindByName` |
| `ServiceRuntimeContextRepository`（延续） | 提供各服务当前 `Status` / `GroupName` |

**基础设施实现落点（/6-plans）：** `runnerRuntimeContextRepository` 适配器 + 基于 `Config.Flatten()` 的 `configServiceTopologyRepository`。

## 领域事件 (Domain Events)

扩展 `service_operation_events.go`：

| 事件 | 触发时机 |
|------|----------|
| `ServiceCascadeStartRequested` | UI/API 发起链式启动 |
| `ServiceCascadeStopRequested` | UI/API 发起链式关闭 |
| `ServiceGroupStartRequested` | UI/API 发起启动本组 |
| `ServiceCascadeStepFailed` | 链中某步失败（携带 `Completed`、`FailedAt`） |

## NFR → 模型映射

| NFR 决策 | 模型落点 |
|----------|---------|
| 一致性 L2：计划纯函数 | `ServiceCascadeOrchestrationService` 无写仓储 |
| 容错 L2：fail-fast | `CascadeExecutionReport`；无 Saga 实体 |
| 安全 L2：ownership | 执行留在应用层 `*WithActor`，领域层不绕过 |
| 可观测 L2 | 事件 `ServiceCascadeStepFailed` + 计划 `String()` 日志友好 |
| 可维护 L2：`cascade=false` | 应用层策略分支，领域服务仅服务 cascade 路径 |

## 代码骨架落点

| 文件 | 类型 |
|------|------|
| `service_topology_node_value_object.go` | VO |
| `service_lifecycle_plan_value_object.go` | VO |
| `cascade_execution_report_value_object.go` | VO |
| `service_topology_repository.go` | Repository 接口 |
| `service_cascade_orchestration_service.go` | Domain Service |
| `service_cascade_orchestration_service_test.go` | 单元测试 |
| `service_operation_events.go` | 扩展事件 |

## 实现边界（留给 /6-plans）

- **Increment 1–2：** Runner 调用 `PlanStartCascade` / `PlanStopCascade`；`ui.go` 默认 `cascade=true`。
- **Increment 3：** `PlanStartGroup` + `/api/start-group`。
- **Increment 4：** HTTP 错误映射 `CascadeExecutionReport`；日志 `[cascade]` 前缀。
- **不修改：** `RestartService` 路径；`StopGroup` 可内部复用 `PlanStop` 拓扑 helper。

## 自检 (Hard Gate)

- [x] 新增类型位于 `runAll/src/domain/`
- [x] 领域层无 `net/http`、`syscall`、`os/exec` 导入
- [x] 拓扑与状态通过 Repository 接口注入
- [x] 无新 ORM/外部 SDK
- [x] 单元测试覆盖三节点链 start/stop 顺序
- [x] 领域事件过去式命名
