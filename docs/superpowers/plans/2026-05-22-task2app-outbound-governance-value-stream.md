# Value Stream: task2app 出站调用治理

> Derived from design: `.cursor/plans/调用治理与异步化_aab3449c.plan.md`

## Value Summary
平台调用方可以稳定获得“内网同步调用 1 秒超时保护 + 外网异步执行与事件回传”的一致体验，降低阻塞与超时不可见问题。

## Capability Classification
- Core value
  - 内网同步调用统一进入 `internal_sync` 策略并强制 1 秒超时。
  - 外网调用改为 `external_async`，主流程立即返回 `job_id`，结果由事件推送回传。
- Essential support
  - 统一异常契约（timeout/network/upstream）与可观测字段。
  - 统一策略路由（`internal_sync` / `external_async` / `streaming`）与最小治理网关。
- Enhancement
  - 多模块（cloud-integration、billing）逐步迁移，消除散落直连调用。
  - 任务状态可视化与更细粒度错误分类。
- Future
  - OAuth 外网链路全量异步化（从 planned 到 active 的完整闭环）。
  - 全仓库出站调用自动扫描与治理守卫。

## End-to-End Flow
用户触发业务操作（如层级推送后自动建 PR）  
→ API 入口按策略路由识别调用类型  
→ 内网调用走 1 秒同步保护 / 外网调用进入异步任务队列  
→ Worker 执行外网调用并更新任务状态  
→ 通过 SSE 事件推送结果  
→ 用户在前端看到成功/失败状态与错误语义。

## Wait / Dependency Points
- `external_async` 依赖任务执行器与事件总线（SSE_MESSAGE）。
- 业务模块迁移依赖统一网关可复用后再逐条替换。
- `relay-status-push-timeout-go-relay` 与新规则在 ACK 时序上存在耦合风险，需要在后续增量统一约束。

## Value Increments

### Increment 1: Internal Timeout Thin Slice (Thin Slice)
**Value to user:** 内网同步调用发生超时时可在 1 秒内快速失败，不再长时间阻塞。  
**Scope:** 建立 `internal_sync` 策略、1 秒硬超时、统一异常契约，并接入 gitoauth 内网调用链路。  
**Depends on:** nothing

### Increment 2: External Async Thin Slice
**Value to user:** 外网 PR 创建不阻塞主请求，操作即时返回并可收到异步结果。  
**Scope:** 引入最小异步任务模型（queued/running/succeeded/failed）与事件推送；迁移 GitHub PR 创建链路。  
**Depends on:** Increment 1

### Increment 3: Governance Expansion
**Value to user:** 更多云平台/外网调用获得一致异步体验与错误可见性。  
**Scope:** 扩展到 cloud-integration、billing 等外网链路；清理散落 timeout 与绕过入口。  
**Depends on:** Increment 2

### Increment 4: Full Coverage and Hardening
**Value to user:** 规则覆盖更全面，流式与长任务边界清晰，回归风险降低。  
**Scope:** OAuth async 全量化、回归套件完善、治理守卫与配置校验。  
**Depends on:** Increment 3

