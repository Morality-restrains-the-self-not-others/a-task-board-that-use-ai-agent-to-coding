# Intent: task-cloud-service 切断 saas SQLite 直连

将 `taskCloudService` 产品路径对 `saas.sqlite3` / `SAAS_SQLITE_PATH` 的直连改为经 `saas-backend` internal HTTP；表 owner 仍为 saas-backend。

## 验收

1. 生产 `main` 不再 `openSaasDB`；源码无 `SAAS_SQLITE_PATH` / `db/saas/saas.sqlite3` 产品引用
2. tenant-member / git-identities / feature-params store / budget-permissions / snapshot / api_key_usage 均经 HTTP
3. 容器 `feature-params-env` 与 budget gate 仍可用
4. `known_cross_service_access` 删除 task-cloud-service→saas
5. ownership checker 不再 WARN 该项



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：切断 SQLite 直连的 HTTP cutover，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: task-cloud-service 切断 saas SQLite 直连 | — | — | — | — | 切断 SQLite 直连的 HTTP cutover，无新增业务事件 |
## 变更记录

- 2026-07-14：落地 HTTP 客户端 + Django internal store API；移出 known-debt
