# Domain Model: taskContainerGateway

## Bounded Context: ContainerGateway

网关限界上下文：浏览器/容器 HTTP 边缘，不含 OAuth/PR 领域规则。

### 聚合根: ContainerForwardSession

一次 browser→OSJS 转发会话。

| 属性 | 类型 | 说明 |
|------|------|------|
| trace_id | string | X-Trace-Id |
| action | ContainerForwardAction | 路由动作 |
| scope | TaskScope | tenant/workspace/task |
| auth | GatewayAuthResult | validate 结果 |
| target | ContainerTarget | base + token |
| stage | ForwardStage | inbound/validate/outbound_start/outbound_done |

**不变量：**
- auth.scope_ok 为 false 时禁止 outbound
- target.token 非空才允许 forward

### 值对象

- **TaskScope** — tenant_id, workspace_id, task_id
- **ContainerTarget** — base_url, access_token
- **GatewayAuthResult** — user_id, auth_method, scope_ok
- **ContainerForwardAction** — layer_git_commit, layer_graph, …
- **ForwardStage** — 枚举

### 领域服务

- **ContainerForwardOrchestrator** — validate → resolve → forward（Go 实现，Django 提供 auth/target 端口）

### 仓储接口（防腐层）

- **SessionValidatorPort** — validate(cookie, token, scope) → GatewayAuthResult
- **ContainerTargetResolverPort** — resolve(scope, override_url) → ContainerTarget
- **OnlineServiceClientPort** — forward(action, target, body) → response

### 领域事件

- **ContainerForwardCompleted** — trace_id, action, upstream_status, duration_ms
- **ContainerForwardFailed** — trace_id, action, error_code

## Bounded Context: SaaS Auth（Django 真源）

- **GatewaySessionValidation** — 应用服务，复用 DRF authenticate + 租户成员校验
