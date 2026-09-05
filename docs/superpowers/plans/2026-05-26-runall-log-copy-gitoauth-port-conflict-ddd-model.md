# /5-ddd 领域模型: runAll 日志复制与 git-oauth 端口冲突恢复启动

## 输入

- Value Stream: `docs/superpowers/plans/2026-05-26-runall-log-copy-gitoauth-port-conflict-value-stream.md`
- NFR: `docs/superpowers/plans/2026-05-26-runall-log-copy-gitoauth-port-conflict-nfr-clarification.md`
- 设计: `docs/superpowers/specs/2026-05-25-runall-stability-first-design.md`

## 限界上下文 (Bounded Context)

- `orchestration-lifecycle`（延续）：统一 UI 手动启动与批量启动的 preflight 端口治理。
- `service-runtime-observability`（延续）：结构化失败 hint、日志可复制文本、端口冲突清理事件。

## 实体与值对象

### 实体

- 无新增实体。复用既有 `ManagedService`、`StartupSession`、`ServiceStartupAttempt`。

### 值对象（新增）

- `PortConflictSnapshot`：某服务在某端口上的外部监听冲突快照（port + foreign_pids）。
- `PortConflictCleanupResult`：清理尝试结果（terminated_pids + remaining_foreign_pids）。
- `FailureHint`：结构化失败提示（failure_code + port + pids + message），可渲染为 UI/日志文本。
- `ServiceLogClipboardSnapshot`：某服务日志行的纯文本快照，供前端复制与后端测试共用格式。

### 值对象（复用）

- `ServiceFailureCode`：`PRECHECK_PORT_CONFLICT` 等稳定分类。
- `ServiceOwnership`：owned PID 判定边界。
- `LogEntry`：日志行渲染输入。

## 聚合与聚合根

- `ManagedServiceAggregate`（聚合根：`ManagedService`，延续）
  - 新增不变量：Launch 前必须通过 `ServicePreflightDomainService.EnsurePortsReadyForLaunch`
  - 手动启动与批量启动共用同一 preflight 语义（NFR 一致性 L2）

- `StartupSessionAggregate`（聚合根：`StartupSession`，延续）
  - 端口清理事件挂载到会话可观测链路（内存事件，不要求持久化）

## 领域服务

- `ServicePreflightDomainService`（新增）
  - `DetectPortConflicts(serviceName, ports)`：探测 foreign listener
  - `ResolvePortConflicts(conflicts)`：终止 foreign PID 并重扫
  - `EnsurePortsReadyForLaunch(serviceName, ports, autoCleanup)`：Launch 前统一入口
  - `BuildPortConflictFailureHint(conflict, cleanup)`：生成可复制诊断 hint

- `ServiceOwnershipGuardService`（延续）
  - 与 preflight 协作：owned PID 永不进入 terminate 列表

## 仓储接口 (Repository)

- `PortListenerProbeRepository`（新增）：抽象 `lsof`/端口监听探测
- `ForeignProcessTerminationRepository`（新增）：抽象 SIGTERM/SIGKILL 清理
- `OwnedProcessRegistryRepository`（新增）：抽象 runAll 当前 session owned PID 集合
- `ServiceLogRepository`（延续）：日志 tail/clear，不新增 copy API

## 领域事件 (Domain Events)

- `PortConflictCleanupAttempted`（新增）
- `PortConflictCleanupSucceeded`（新增）
- `PortConflictCleanupFailed`（新增）

延续事件：`ServicePreflightFailed`、`ServiceStartRejectedByForeignProcess`。

## NFR → 模型映射

| NFR 决策 | 模型落点 |
|----------|---------|
| 一致性 L2：手动/批量 preflight 统一 | `ServicePreflightDomainService.EnsurePortsReadyForLaunch` 作为 Launch 唯一 preflight 入口 |
| 安全 L2：仅清理 foreign PID | `FilterForeignPIDs` + `OwnedProcessRegistryRepository` |
| 可观测 L2：可复制诊断 | `FailureHint.Render()` + `ServiceLogClipboardSnapshot.PlainText()` |
| 可用性 L2：自动恢复 | `ResolvePortConflicts` + cleanup 事件三元组 |

## 代码骨架落点

- `runAll/src/domain/port_conflict_snapshot_value_object.go`
- `runAll/src/domain/port_conflict_cleanup_result_value_object.go`
- `runAll/src/domain/failure_hint_value_object.go`
- `runAll/src/domain/service_log_clipboard_snapshot_value_object.go`
- `runAll/src/domain/port_listener_probe_repository.go`
- `runAll/src/domain/foreign_process_termination_repository.go`
- `runAll/src/domain/owned_process_registry_repository.go`
- `runAll/src/domain/service_preflight_domain_service.go`
- `runAll/src/domain/service_lifecycle_domain_events.go`（扩展 cleanup 事件）
- `runAll/src/domain/port_conflict_domain_model_test.go`
- `runAll/src/domain/service_preflight_domain_service_test.go`

## 实现边界（留给 /6-plans）

- Increment 1（日志复制按钮）：主要改 `status.html`；后端仅复用 `ServiceLogClipboardSnapshot` 作为格式契约/测试辅助，不新增 API。
- Increment 2/3：在 `runner.go`/`preflight.go` 基础设施层实现三个 Repository，并让 `startService()` 调用 `ServicePreflightDomainService`。
- Increment 4：UI 消费 `FailureHint.Render()` 等价文案，不扩展 `/api/status` 字段契约。

## 自检 (Hard Gate)

- [x] 实体/值对象/仓储位于 `runAll/src/domain/`
- [x] 领域层无 ORM 导入
- [x] 领域层无外部服务 SDK/HTTP 客户端/`lsof`/`syscall` 导入
- [x] 仓储为抽象接口，不含持久化/系统调用实现
- [x] 领域事件采用过去式命名
- [x] 领域服务通过构造函数注入仓储依赖
- [x] 无数据库字段声明细节
- [x] 领域单元测试覆盖 foreign PID 过滤、preflight 清理、日志 clipboard 渲染

---

领域模型已生成到 `runAll/src/domain/`。仓储接口和领域服务接口已就绪，可进入 `/6-plans-实施计划`。
