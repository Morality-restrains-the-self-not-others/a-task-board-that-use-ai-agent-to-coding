# Intent: 边缘 nginx 上游绑定 0.0.0.0 + 同机调用 loopback

## 背景

边缘 nginx upstream 指向宿主机 LAN IP（如 `183.250.1.132:PORT`）。若服务仅绑 `127.0.0.1`，边缘连不上 → **502 Bad Gateway**。  
此前 `credential` 已用 Django/TCG loopback 规避同机调用；公网子域仍依赖上游可被 LAN 访问。

## 需求（2026-07-12）

对 nginx 注释列出的同类服务统一：

1. **监听**：`host: 0.0.0.0`（与 `task-agent-support` 一致）
2. **同机调用**：TCG/Django → `http://127.0.0.1:PORT`（勿经公网子域）
3. **注入容器的 Origin**（`taskApiEndpointOrigin` / `businessApiEndpointOrigin` / `publicBaseUrl`）保持公网子域

## 范围服务

| 服务 | 端口 | conf |
|---|---|---|
| task-ai-endpoint | 8013 | `conf/ai/task-ai-endpoint/config.yaml` |
| task-credential-service | 8015 | `conf/container/task-credential-service/config.yaml` |
| go-relay (relay-to-trae) | 8797 | `conf/infra/relay-to-trae/config.yaml` |
| go-run-container (mock) | 8796 | `conf/mock/mock-run-container/config.yaml` |



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：边缘 Nginx 绑定配置，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: 边缘 nginx 上游绑定 0.0.0.0 + 同机调用 loopback | — | — | — | — | 边缘 Nginx 绑定配置，不产生业务领域事件 |
## 变更记录

| 日期 | 差异 | 原因 |
|---|---|---|
| 2026-07-12 | 上述服务 host→0.0.0.0；TCG 同机 URL→loopback；runAll health 将 0.0.0.0 规范为 127.0.0.1 | 消除边缘 502 同类问题 |
