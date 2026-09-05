# Value Stream: relayToTrae Token 审计事件流

> Derived from design: `docs/superpowers/specs/2026-05-20-relay-token-audit-events-design.md`

## Value Summary

当 relayToTrae 发生 token 401 报错时，研发可在分钟级定位是否使用旧 token、以及旧 token 的来源与覆盖点。

## End-to-End Flow

触发（启动/刷新状态/上报状态） → token 生命周期事件写入 → relay 使用事件写入 → status-push 结果事件写入（含 401） → 按任务时间线回放 → 排障结论输出

## Value Stages

- Trigger：用户点击 `直接启动` / `启动` / `刷新状态` 或 relay 定时 status-push
- Stage 1（核心价值）：token 生命周期事件被标准化记录
- Stage 2（核心价值）：relay 使用 token 的关键动作被记录
- Stage 3（核心价值）：status-push 成功/401 被记录并可关联
- Stage 4（交付点）：研发按任务维度查询事件流并定位旧 token 使用点

## Wait / Dependency Points

- 事件写入依赖 task/tenant/workspace 上下文可用
- 401 事件关联依赖 trace_id / seq / task_id 一致性
- 回放查询依赖索引（`task_id + created_at`）

## Value Increments

### Increment 1: 最薄端到端审计闭环（Thin Slice）
**Value to user:** 发生 401 时可以看到最小可用事件链，不再“黑盒”。  
**Scope:**  
- 新增 `container_token_audit_events` 表与基础 repository  
- 在 `status-push` 成功/401 路径写事件  
- 提供按 `task_id` + 时间窗口查询接口（内部或管理命令）  
**Depends on:** 无

### Increment 2: Token 生命周期全链路可追踪
**Value to user:** 能确认 token 是否在 exchange/refresh 阶段被轮换与覆盖。  
**Scope:**  
- 在 `bootstrap/exchange_refresh/refresh_access` 写事件  
- 记录 `prev/new access hash` 与 suffix  
- 增加链路字段 `trace_id`, `seq`, `source_component`  
**Depends on:** Increment 1

### Increment 3: relay 使用点可观测与根因闭环
**Value to user:** 能直接定位“哪个 relay 动作用了旧 token”。  
**Scope:**  
- 在 `relay_register` / `relay_start` 写事件  
- 统一错误码与错误详情截断策略  
- 增加“按 task 回放”查询输出模板  
**Depends on:** Increment 2

### Increment 4: 增强项（稳定性与运维）
**Value to user:** 审计能力可持续运行，不产生安全或存储负担。  
**Scope:**  
- 保留策略（30~90 天）与清理任务  
- 明文泄露防护测试（只存 hash/suffix）  
- 索引与慢查询监控  
**Depends on:** Increment 3

## Classification（按价值类型）

- Core value：Increment 1, 2, 3
- Essential support：索引、trace 关联、错误码标准化
- Enhancement：保留策略可视化、聚合报表
- Future：审计 UI 首版可视化看板
