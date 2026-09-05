# Task Detail Relay 启动稳定性设计

日期：2026-05-23  
状态：待评审  
范围：`task-detail` 页面中 `relayToTrae` 直接启动全链路稳定性治理（体验优先）

## 1. 背景与问题陈述

当前链路在“点击启动”后存在首轮 `exchange-refresh` 偶发失败、后续重试成功的现象。用户侧会看到不稳定状态，严重时出现 HTTP 500 页面级失败。  
本设计目标是在不做大规模重构前提下，建立可恢复、可观测、可回归的稳定链路。

## 2. 目标与非目标

### 2.1 目标

1. 用户点击启动后，页面不出现 500 级失败提示，状态可收敛到“运行中”或可解释失败。
2. token 相关业务失败统一为可预期 4xx，并返回稳定 `error_code` 与 `trace_id`。
3. 补齐后端与 Playwright 自动化回归，阻断同类问题再次上线。

### 2.2 非目标

1. 不在本轮做 relay 启动架构重写（不引入全新编排引擎）。
2. 不改动与当前问题无关的通用认证体系。
3. 不扩大到其他业务页面的统一错误治理（仅覆盖 task-detail 启动链路）。

## 3. 现状链路与目标链路

## 3.1 现状链路

`task-detail 点击启动`  
-> `env-prepare/token-init`  
-> `exchange-refresh`  
-> `refresh-access`  
-> `relay start/register/status-push`  
-> `SSE 状态回传`

风险点：业务异常有机会以 500 泄漏；前端对错误语义依赖文案，不够稳定；失败链路自动化覆盖不足。

### 3.2 目标链路

保持现有主干流程，但引入：

1. **后端错误契约收敛**：业务异常统一 4xx + `error_code` + `trace_id`。
2. **前端状态机收敛**：仅按状态码+错误码迁移状态，不做文案推断。
3. **审计与回归加固**：失败事件可追踪，关键路径有测试门禁。

## 4. 设计方案（推荐方案A：契约收敛优先）

## 4.1 状态机分层

### 后端状态（逻辑态）

`INIT` -> `TOKEN_READY` -> `EXCHANGED` -> `ACCESS_REFRESHED` -> `RELAY_STARTING` -> `RUNNING`  
任一步骤失败 -> `FAILED(error_code)`

### 前端状态（展示态）

`idle` / `starting` / `running` / `degraded` / `failed`

迁移规则：

1. `starting` 阶段禁止重复点击触发并发启动。
2. 收到可恢复错误码时进入 `degraded` 并提供重试。
3. 收到不可恢复错误码时进入 `failed` 并提示刷新上下文或人工处理。

## 4.2 统一错误契约

token 相关接口在业务失败时返回统一格式：

```json
{
  "detail": "access_token 已过期，请调用 refresh-access 续期",
  "error_code": "TOKEN_ACCESS_EXPIRED",
  "trace_id": "..."
}
```

首批错误码：

- `TOKEN_ACCESS_MISSING` (400)
- `TOKEN_ACCESS_INVALID` (401)
- `TOKEN_ACCESS_EXPIRED` (401)
- `TOKEN_REFRESH_MISSING` (400)
- `TOKEN_REFRESH_INVALID` (401)
- `TOKEN_SCOPE_MISMATCH` (403)
- `TOKEN_EXCHANGE_ALREADY_DONE` (403)
- `BUSINESS_API_ENDPOINT_INVALID` (400)
- `RELAY_DOWNSTREAM_UNAVAILABLE` (502/503)
- `RELAY_START_DISPATCH_FAILED` (500)

约束：

1. 业务可预期错误禁止返回 500。
2. 500 仅用于不可恢复系统错误，且必须带 `trace_id` 并落审计失败事件。

## 4.3 前端映射策略

`task-detail` 启动面板新增错误码映射表：

- `TOKEN_ACCESS_EXPIRED` / `TOKEN_ACCESS_INVALID` -> “凭证失效，正在重新准备启动环境”
- `TOKEN_SCOPE_MISMATCH` -> “任务上下文不一致，请刷新页面后重试”
- `TOKEN_EXCHANGE_ALREADY_DONE` -> 自动转补偿分支（走 refresh-access）
- `BUSINESS_API_ENDPOINT_INVALID` -> 配置错误提示并阻断启动
- `RELAY_*` 系统错误 -> “服务暂不可用”，展示 `trace_id` 便于排查

兼容策略：若历史接口无 `error_code`，按 status code 做降级映射并打埋点。

## 4.4 可观测性

1. 启动请求分配统一 `trace_id`，贯穿前端日志、relay 日志、审计事件。
2. 审计事件至少覆盖 attempted/succeeded/failed 三态。
3. 建立最小指标：启动失败率、补偿成功率、500 次数。

## 5. 测试与回归设计

## 5.1 后端 pytest

1. 契约测试：断言 `status + error_code + trace_id`。
2. 脏数据回归：历史异常字段值不再触发 500 泄漏。
3. 幂等测试：重复 exchange 行为符合预期错误码。
4. 审计测试：失败链路事件齐全且 `trace_id` 可串联。
5. 异常映射测试：领域异常被正确映射为业务错误响应。

## 5.2 Playwright E2E

1. 启动成功流：点击启动后进入运行中，页面无 HTTP 500。
2. 首轮失败补偿流：首轮 exchange 失败后可恢复并最终运行。
3. 不可恢复错误流：显示明确提示，不出现无意义重试循环。
4. SSE 收敛流：状态从 starting 合理迁移，不悬挂。

## 5.3 CI 门禁

1. 必跑 token 生命周期 pytest 子集。
2. 必跑 task-detail relay 启动 Playwright 子集。
3. 出现新增 500 或无 `error_code` 业务失败即阻断合并。

## 6. 分阶段实施切片

### Slice 1（止血）

- 异常映射兜底，阻断业务 500 泄漏。
- 前端展示统一可恢复失败提示。

### Slice 2（契约）

- 引入并统一错误码，前端接入映射表。

### Slice 3（回归）

- 完成 pytest + Playwright 防回归矩阵并接入 CI。

### Slice 4（观测）

- 指标化与事件链可视化，支持持续运营排查。

## 7. 回滚与风险控制

1. 每个 Slice 独立发布，避免大批量回滚。
2. 前端映射以开关控制，异常时先回退展示策略。
3. 后端契约调整维持向后兼容窗口（旧客户端可按 status code 退化处理）。

## 8. Domain Concept Inventory（供 DDD 步骤输入）

### Bounded Contexts

- 任务协作（task-detail 状态与交互）
- 云平台与资源（container token / relay 启动）
- 可观测审计（token lifecycle events）

### Key Entities

- `CloudServerConfig`
- `ContainerTokenAuditEvent`
- `RelayTaskRegistration`

### Candidate Aggregates

- `ContainerTokenSession`（token 生命周期一致性边界）
- `RelayTaskRegistration`（任务与状态推送绑定）

### Domain Events

- `exchange_refresh`
- `refresh_access`
- `relay_start_attempted`
- `relay_start_succeeded`
- `relay_start_failed`

## 9. Value Stream Impact Analysis

本需求不是全新绿地功能，影响现有 value streams：

1. `task-detail-runtime-relay`（直接影响）
2. `relay-token-audit-observability`（直接影响）
3. `relay-token-audit-full-chain-eventization`（直接影响）
4. `relay-status-push-timeout-go-relay`（关联影响）
5. `internal-api-timeout-governance`（关联影响）

字段影响（重点）：

- `saas-backend.cloud_cloudserverconfig.container_access_token`
- `saas-backend.cloud_cloudserverconfig.container_refresh_token`
- `saas-backend.cloud_cloudserverconfig.business_api_endpoint`
- `saas-backend.cloud_container_token_audit_event.event_type`
- `saas-backend.cloud_container_token_audit_event.error_code`
- `saas-backend.cloud_container_token_audit_event.trace_id`
- `saas-backend.cloud_container_token_audit_event.seq`

测试影响：

- 需增强 token 生命周期测试、relay proxy 测试、task-detail Playwright 启动回归测试。

状态变化：

- 当前不新增 stream，仅在现有 active steps 上增强契约与回归。

跨流依赖变化：

- 强化了 `task-detail-runtime-relay` 与审计 streams 的协同关系（通过 trace_id 串联）。

