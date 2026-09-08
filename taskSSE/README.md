# taskSSE

独立 SSE 侧车服务，将任务详情页的 `server-startup-status-sse` 长连接从 Django `runserver` 剥离，避免每个 SSE 连接占用一个 Python 线程导致 API 假死。

## 通信

| 方向 | 机制 |
|------|------|
| Django → taskSSE | HTTP `POST /internal/publish`（开发/内存队列模式必达） |
| Django → taskSSE | Redis pub/sub 频道 `sse:{task_id}`（`transport=redis`） |
| Django → taskSSE | Kafka topic `sse-message`（`transport=kafka`，与 `SSE_MESSAGE` 事件一致） |
| 浏览器 → taskSSE | `GET /api/tenant/.../task/.../cloud/server-startup-status-sse/` |

## 配置

`task2app/conf/port_config.json` → `taskSSE`：

```json
{
  "taskSSE": {
    "host": "127.0.0.1",
    "port": 8798,
    "secret": "dev-secret",
    "transport": "redis",
    "enabled": true,
    "redis": { "host": "127.0.0.1", "port": 6379, "channelPrefix": "sse:" },
    "kafka": { "bootstrapServers": "localhost:9093", "topic": "sse-message", "groupId": "task-sse-consumer" }
  }
}
```

切换 Redis / Kafka：修改 `transport` 为 `redis` 或 `kafka` 后重启 taskSSE。

环境变量覆盖：`TASK_SSE_TRANSPORT`、`TASK_SSE_PORT`、`TASK_SSE_HOST` 等。

## 启动

```bash
cd taskSSE && bash run.sh
```

已纳入 `runAll.yaml` 的 `task-sse` 服务。Vite 开发服务器会将 SSE 路径代理到 taskSSE（`enabled: true` 时）。

## 测试

```bash
npm test
```

## 浏览器 SSE 网关密钥

浏览器路径（任务启动 SSE、充值 SSE）须带 APISIX 注入的 `X-TaskGateway-Internal-Secret`（与 `conf/gateway/task-gateway` 的 `gatewayInternalSecret` 一致）。直连伪造 `X-User-Id` 将被 403。`/health` 与 `/internal/publish` 不受此约束。

环境变量覆盖：`TASK_GATEWAY_INTERNAL_SECRET` 或 `TASK_SSE_GATEWAY_INTERNAL_SECRET`。

## 充值事件 SSE

- 路径：`GET /api/tenant/{tenant_id}/billing/recharge-events-sse/`
- 鉴权：APISIX `auth_mode: token` → forward-auth 注入 `X-User-Id`；proxy-rewrite 注入 `X-TaskGateway-Internal-Secret`（忽略 query `user_id`）
- Redis：`sse:billing:user:{user_id}`
- 网关：`routes.yaml` → `sse-billing-recharge-events` → taskSse

## License

本仓库以 GNU Affero General Public License v3.0 授权，见 [LICENSE](./LICENSE)。

