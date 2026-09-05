# Intent: 全量应用服务监听 0.0.0.0

## 需求（2026-07-12）

对 runAll 编排的**应用服务**统一：HTTP/TCP **监听**地址为 `0.0.0.0`（非仅 `127.0.0.1`），便于边缘 nginx / LAN 探活与转发。

## 明确不改（连接目标，非监听）

| 配置 | 原因 |
|---|---|
| `conf/infra/{redis,kafka,portainer,ai-monitor}` | 客户端连隧道/远端的 **连接 host** |
| `docker-infra.yaml` fragments | 同上 |
| `services.*.host: 127.0.0.1`（Cloud/Project/Task 等） | 同机 **出站** 调兄弟服务 |
| `djangoInternalApiBase` / `internalApiBase` | 必须 loopback |
| git-oauth provider `host` | Provider 匹配用主机名，非本进程监听 |

## 已改监听

task-bill、django conf host、vue、git-service conf、mock-trae-worker、domain-events intents（18020+）、及此前 aiendpoint/credential/relay/mock。

## 验收

runAll `stop-all` → `start-all` 后，`ss -tlnp` 关键端口非仅 `127.0.0.1`；健康检查通过。


## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：部署监听配置变更，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: 全量应用服务监听 0.0.0.0 | — | — | — | — | 部署监听配置变更，不产生业务领域事件 |
