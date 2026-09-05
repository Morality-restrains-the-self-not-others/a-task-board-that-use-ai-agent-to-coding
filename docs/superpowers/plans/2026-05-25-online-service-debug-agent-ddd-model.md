# DDD 模型：onlineServiceJS DEBUG_AGENT 调试链路

> 输入价值流：`docs/superpowers/plans/2026-05-25-online-service-debug-agent-value-stream.md`  
> 输入 NFR：`docs/superpowers/plans/2026-05-25-online-service-debug-agent-nfr-clarification.md`

## 1. 限界上下文（Bounded Context）

- **Task Detail 启动上下文**：负责 `relayToTrae=true` 直启参数组装与默认开关注入。
- **Relay Runtime 编排上下文**：负责 env 透传、onlineServiceJS 生命周期编排。
- **OnlineService Observability 上下文**：负责请求生命周期调试日志的领域规则与事件契约。

## 2. 实体与值对象

### 实体（Entity）
- `OnlineServiceDebugLogEntry`：调试日志聚合根，承载一次入站/出站请求的完整上下文。

### 值对象（Value Object）
- `DebugAgentFlag`：开关语义封装，统一解析 `1/true/yes/on`。
- `HttpDebugPayload`：请求/响应载荷快照，不做脱敏与截断，仅做结构合法性约束。

## 3. 聚合与聚合根

- 聚合：`OnlineServiceRequestLogAggregate`
  - 聚合根：`OnlineServiceDebugLogEntry`
  - 内含值对象：`DebugAgentFlag`、`HttpDebugPayload`

一致性边界：一次请求生命周期（请求 + 响应 + 状态码 + trace_id）作为同一聚合记录。

## 4. 领域服务

- `OnlineServiceDebugLogService`
  - 职责：按方向（inbound/outbound）构建聚合根并生成领域事件。
  - 依赖：`OnlineServiceDebugLogRepository`（抽象接口）。

## 5. 仓储接口

- `OnlineServiceDebugLogRepository`
  - `save(entry)`：保存聚合根
  - `find_by_trace_id(trace_id)`：按 trace 查询

## 6. 领域事件

- `OnlineServiceInboundRequestHandled`（过去式）
- `OnlineServiceOutboundRequestCompleted`（过去式）

## 7. 目录落位

- `task2app/Saas_project/cloud/domain/entities/online_service_debug_log_entry.py`
- `task2app/Saas_project/cloud/domain/value_objects/debug_agent_flag.py`
- `task2app/Saas_project/cloud/domain/value_objects/http_debug_payload.py`
- `task2app/Saas_project/cloud/domain/repositories/online_service_debug_log_repository.py`
- `task2app/Saas_project/cloud/domain/services/online_service_debug_log_service.py`
- `task2app/Saas_project/cloud/domain/events/online_service_inbound_request_handled.py`
- `task2app/Saas_project/cloud/domain/events/online_service_outbound_request_completed.py`
