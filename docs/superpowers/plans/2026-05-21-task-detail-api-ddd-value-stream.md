# Value Stream: Task Detail API DDD

> Derived from design: `docs/superpowers/specs/2026-05-21-task-detail-api-ddd-design.md`

## Value Summary
任务协作者可以稳定查看/更新任务详情，并实时看到容器运行态与 relay 状态收敛结果，减少“页面有入口但链路不可用”的排障成本。

## Capability Classification
- Core value
  - task detail 主链路读写一致（详情读取、PATCH 后回读一致）
  - relay status push 能进入统一收敛语义并可观测
- Essential support
  - 容器 runtime 上下文由应用服务统一投影，避免视图层直连 ORM
  - 领域事件契约（`TaskDetailPatched`、`ContainerUiContextRefreshed`、`RelayStatusConverged`）保持边界清晰
- Enhancement
  - 增强错误可观测性（序列号、trace、error_code 对齐）
  - runtime 上下文空快照语义一致化（更易排障）
- Future
  - `TaskContainerRuntimeSession` 聚合从最小切片继续演进为完整聚合行为
  - 更细粒度跨上下文事件回放与审计聚合

## End-to-End Flow
[Trigger] 用户打开任务详情 / 容器页面触发状态上报  
→ [Stage 1] `task-detail` API 返回任务读模型并支持 patch  
→ [Stage 2] runtime 应用服务读取上下文快照并发布 UI 上下文事件  
→ [Stage 3] relay status push 进入收敛服务并写入审计事件  
→ [Delivery] 前端获得稳定任务状态 + 容器可达信息 + relay 收敛结果

## Wait & Dependency Points
- runtime 上下文依赖 `CloudServerConfig` 最新快照；若无配置应返回空快照而非异常。
- relay 收敛依赖启动会话状态机（token-init 完成后才能 start-dispatch）。
- 审计事件链依赖 status push 的 seq/trace/error 字段完整上报。

## Value Increments

### Increment 1: Task Detail Thin Slice
**Value to user:** 用户可稳定读取任务详情并执行基础更新，回读结果可验证。  
**Scope:** 任务详情主链路（查询 + patch）最小端到端闭环。  
**Depends on:** nothing.

### Increment 2: Runtime Context Projection
**Value to user:** 用户在任务详情可看到一致的容器 runtime 上下文（是否可达、页面 URL、VSCode URL）。  
**Scope:** `TaskContainerRuntimeContextAppService` + repository 投影 + 空快照语义。  
**Depends on:** Increment 1.

### Increment 3: Relay Status Convergence
**Value to user:** relay 状态推送不再“有上报无收敛”，可在链路中观测成功/失败语义。  
**Scope:** status push → 收敛服务 → 审计事件一致性。  
**Depends on:** Increment 1, Increment 2.

## Stream Mapping (for valueStream YAML)
- stream name: `task-detail-runtime-relay`
- domain: `任务协作`
- description: `task-detail 主链路与容器 runtime/relay 状态收敛`
- step order:
  1. `task-detail-thin-slice`
  2. `container-runtime-context`
  3. `relay-status-convergence`
