# Domain Model: taskAgentSupport Internal Dispatch

> 薄切片 — 应用层适配，无新持久化实体

## Bounded Context

**容器运行时令牌**（既有）— internal API 为 taskAgentSupport 网关的入站适配。

## 概念

| 类型 | 名称 | 职责 |
|------|------|------|
| 应用服务 | `TaskAgentSupportInternalDispatch` | 解析 envelope，构造 HttpRequest，委派 inbound 视图 |
| 值对象 | `InternalInboundEnvelope` | tenant_id, workspace_id, task_id, body, trace_id |
| 端口 | `ContainerInboundView` | exchange-refresh / register-reachability 等（既有 DRF 视图） |

## 不变量

1. 传给 DRF `@api_view` 的必须是 `django.http.HttpRequest`，由视图自行包装 DRF `Request`。
2. 转发失败时：业务错误透传 status/body；仅未预期异常返回 500 + `INTERNAL_DISPATCH_ERROR`。
3. `OperationalError`（locked）→ 503 + `RELAY_DOWNSTREAM_BUSY`。

## 事件

无新领域事件；既有 `ContainerTokenAudit` 在视图内继续写入。
