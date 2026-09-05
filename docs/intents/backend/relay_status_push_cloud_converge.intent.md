# Relay status-push converge 迁入 taskCloudService

## 意图

将 `relay-to-trae/status-push` 的 **converge 编排**从 Django `relay-status-push-effects` 迁到 `taskCloudService`，使 Cloud 与 Redis session 同进程完成收敛。

## 验收

1. Cloud `handleRelayStatusPush` 鉴权后本地：读 Redis session → converge → 写回 → publish SSE；立即 `{status:ok, task_id, ack}`。
2. Redis key 与 Django `RedisRelayStartupSessionRepository` 一致：`relay:startup:wf:{id}`、`relay:startup:scope:{tid}:{wid}:{task}`。
3. Django `relay-status-push-effects` 与 `publish-relay-status-sse` internal HTTP **已删除**；热路径仅 Go（Kafka `SSE_MESSAGE`）。
4. Cloud Redis：`REDIS_HOST`/`REDIS_PORT`/`REDIS_DB` 或 `conf/taskCloudService/config.yaml` redis / domain-events fragment。
5. Go 单测覆盖 converge 成功与 missing session。

## 共存

- **写 session（热路径）**：Gateway → Cloud `/api/internal/relay-startup-session/upsert/` → Redis（见 `relay_startup_session_gateway_direct_write.intent.md`）。
- **读/收敛**：Cloud status-push 同进程。
- **SSE**：Cloud / Gateway → Kafka `SSE_MESSAGE`（经 task-events → task-sse）；同 `task_id` 400ms debounce（Cloud）；Django internal `publish-relay-status-sse` **已删除**。
- **Django transition**：默认不再写 session（可选 flag 恢复）。



## 业务意图 → 事件对照

> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events；同步写路径或非 MQ 副作用在例外理由标注「证据豁免」。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| relay status-push 收敛成功 | RelayStatusConverged | relay_status_converged / SSE_MESSAGE | taskCloudService handleRelayStatusPush | Kafka SSE_MESSAGE → task-events → task-sse | — |
## 变更记录

- 2026-07-10：初版 — converge 迁 Cloud；写仍经 Django。
- 2026-07-10：写路径改为 Gateway→Cloud upsert；Django transition session 写可选。
- 2026-07-10：删除 Django `relay-status-push-effects` compat 壳；热路径仅 Go。
- 2026-07-10：修复 Cloud→Django `publish-relay-status-sse` 错用 task-gateway secret/头名导致持续 403，进而打满 gunicorn、拖垮 `container-clone-log` 的 `django_validate`（502 timeout）；并加 3s 超时与并发上限 2。
- 2026-07-10：SSE 迁出 Django（Kafka `publishSSEMessage`）+ per-task 400ms debounce；gunicorn 默认 threads 2→4。
- 2026-07-10：Gateway `publishRelayStatusSSE` 改 Kafka；删除 Django `publish-relay-status-sse` 路由/视图；遗留 Django helper 亦改 Kafka。
