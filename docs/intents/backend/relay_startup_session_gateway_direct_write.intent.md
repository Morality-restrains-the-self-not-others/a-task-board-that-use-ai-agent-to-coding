# Relay token-init / start 直写 Redis startup session

## 意图

Gateway 在 **token-init 成功**与 **start accept 成功**时，经 Cloud internal upsert 将 startup session **直写 Redis**（与 Cloud status-push converge 同 key），减少对 Django `relay-workflow/transition` 写 session 的依赖。

## 方案

**B）Gateway HTTP → Cloud `/api/internal/relay-startup-session/upsert/`**（复用 Cloud `serializeRelaySession` / `relaySessionStore.save`），不在 Gateway 引入 go-redis。

## 验收

1. token-init / start accept 成功路径写入：
   - `relay:startup:wf:{workflow_id}` JSON，TTL 7200
   - `relay:startup:scope:{tid}:{wid}:{task}` → workflow_id
2. JSON 字段与 Cloud `redis_relay_session.go` / Django Redis repo 兼容（`phase`、`token_initialized`、`created_at`/`updated_at` 等）。
3. Django `relay_workflow_transition` 默认 **不写** Redis session（`RELAY_WORKFLOW_TRANSITION_PERSIST_SESSION` 默认 false）；flag=true 可恢复旧写路径。
4. Go 单测覆盖 Cloud upsert 与 Gateway 调用。

## 共存

- **写 session（热路径）**：Gateway → Cloud upsert → Redis。
- **读/收敛**：Cloud status-push 同进程。
- **Django transition**：事件消费仍可调用；session 写可选。

## 变更日期

2026-07-10


## 业务意图 → 事件对照

> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events；同步写路径或非 MQ 副作用在例外理由标注「证据豁免」。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| token-init 成功写入 startup session | RelayTokenInitSucceeded | RELAY_START_ACCEPTED / relay_token_init_succeeded | taskContainerGateway → Cloud upsert | relay 生命周期消费者 | — |
| start accept 成功写入 startup session | RelayStartAccepted | RELAY_START_ACCEPTED | taskContainerGateway → Cloud upsert | relay 生命周期消费者 | — |
