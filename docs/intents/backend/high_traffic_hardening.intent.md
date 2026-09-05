# 大流量护栏（开发期）

## 意图

在开发模式下为已知容量风险加护栏：网关全局限流、SQLite 池/busy_timeout、SSE 连接上限、AI chunk SSE 批处理、forward-auth 短缓存、relay 生命周期锁超时。结构性项（Postgres、多 relay、HA）登记为上线前。

## 验收

1. `routes-to-apisix.py` 生成的 `global_rules` 含 `limit-req`（`globalPerMinute`）。
2. taskProject/taskTask/taskCloud/taskAIComment/taskCredential 打开 SQLite 时带 busy_timeout 与 MaxOpenConns。
3. Django `PRAGMA busy_timeout=30000`。
4. taskSSE 超 `maxConnections` 返回 503。
5. AI instruct chunk 非每字节一次 HTTP publish。
6. forward-auth 同凭证短窗口内命中内存缓存。
7. go_relay 获取 lifecycle 锁超时返回 503。

## 上线前（不阻塞本迭代）

见 `docs/superpowers/specs/2026-07-15-high-traffic-dev-hardening-design.md` PRE-PROD P1–P6。



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：容量护栏/配置加固，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 大流量护栏（开发期） | — | — | — | — | 容量护栏/配置加固，不产生业务领域事件 |
## 变更记录

- 2026-07-15：初版。
