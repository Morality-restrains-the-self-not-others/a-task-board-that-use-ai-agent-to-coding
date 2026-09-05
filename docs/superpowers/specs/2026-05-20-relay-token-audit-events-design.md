# relayToTrae Token 审计事件流设计

## 背景

在 relayToTrae 链路中，`exchange-refresh` / `refresh-access` 会轮换容器令牌。若运行中组件继续使用旧 token，上报 `status-push` 会出现 `401 无效的 access_token`。当前系统以 `CloudServerConfig` 保存“当前态”，缺少可追溯历史，导致排障时难以确认“谁在何时写入/使用了哪个 token”。

## 目标

- 建立 append-only 的 token 审计事件流，支持按任务维度回放令牌生命周期。
- 在不暴露明文 token 的前提下，支持快速比对“是否使用旧 token”。
- 保持现有 `CloudServerConfig` 作为当前态，不改变在线路径主逻辑。

## 非目标

- 不把历史追溯能力耦合到 UI 首版交互。
- 不将所有组件全面改造为事件溯源架构。

## 方案概述

新增一张只追加审计表 `container_token_audit_events`，记录 token 生命周期及关键调用事件，核心链路写入如下事件：

1. `bootstrap`（初始 access 签发）
2. `exchange_refresh`（access -> refresh）
3. `refresh_access`（refresh -> 新 access）
4. `relay_register`（relay 登记使用 token）
5. `relay_start`（relay 启动使用 token）
6. `status_push_ok`（状态上报成功）
7. `status_push_401_invalid_token`（状态上报 401）

排障时按 `tenant/workspace/task + 时间` 回放事件，即可定位“旧 token 使用点”和“覆盖点”。

## 数据模型

表：`container_token_audit_events`

- `id`
- `tenant_id`
- `workspace_id`
- `task_id`
- `event_type`
- `access_token_sha256`（可空）
- `refresh_token_sha256`（可空）
- `access_token_suffix`（可空，末 6~8 位）
- `refresh_token_suffix`（可空，末 6~8 位）
- `prev_access_token_sha256`（可空）
- `new_access_token_sha256`（可空）
- `trace_id`（可空）
- `seq`（可空）
- `source_component`（django / relay / onlineServiceJS）
- `error_code`（可空，如 401）
- `error_detail`（可空，截断）
- `created_at`

索引建议：

- `(task_id, created_at desc)`
- `(event_type, created_at desc)`
- `(trace_id)`

## 安全约束

- 不落盘明文 access/refresh token。
- 仅存 hash + suffix，suffix 仅用于人工比对，不作为鉴权依据。
- 审计表保留期建议 30~90 天，支持周期清理。

## 写入时机

- Django token 生命周期接口：写 `bootstrap/exchange_refresh/refresh_access`。
- relay 代理入口：写 `relay_register/relay_start`。
- status-push 接口：写 `status_push_ok` 或 `status_push_401_invalid_token`。

## 验证策略

- 单元测试：事件写入正确性、字段完整性、无明文泄露。
- 回归测试：模拟旧 token 上报，断言出现 `status_push_401_invalid_token` 且可关联前序 token 轮换事件。

## Domain Model

- Bounded Context：`Container Token Lifecycle & Observability`
- Aggregates：
  - `ContainerTokenSession`（当前态，已存在）
  - `TokenAuditStream`（新增，只追加）
- Domain Events：
  - `TokenBootstrapped`
  - `RefreshTokenExchanged`
  - `AccessTokenRefreshed`
  - `TokenRegisteredByRelay`
  - `StatusPushRejected401`
- Repository：
  - `ContainerTokenSessionRepository`（已存在）
  - `TokenAuditEventRepository`（新增 append-only）
