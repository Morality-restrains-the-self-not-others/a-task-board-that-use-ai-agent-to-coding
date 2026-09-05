# [运行时] 任务详情 server-startup-status-sse 在 runAll 重启窗口报 nginx 502

## 现象

- 页面：任务详情 `…/task-detail/task_*`
- 请求：`GET /api/tenant/…/workspace/…/task/…/cloud/server-startup-status-sse/`
- 响应：`502 Bad Gateway`，`server: nginx/1.24.0 (Ubuntu)`，`content-type: text/html`，耗时约 **200ms**
- 典型时刻：平台刚执行 runAll 全量启动前的短暂窗口

## 调用链

```
Browser EventSource
  → 边缘 nginx（www.daydaymoney.com）
  → APISIX（taskGateway，route sse-startup-status → up-taskSse:8798）
  → taskSSE（Node，长连接 text/event-stream）
```

APISIX 路由见 `taskGateway/routes/routes.yaml` `sse-startup-status`；Django 在 `TASK_SSE_ENABLED` 时对该路径直接返回 **503 JSON**（已迁侧车），浏览器生产路径**不走** Django。

## 根因

1. **502 是边缘 nginx 上游不可达**，不是业务「启动状态」校验失败。taskSSE 存活时会返回 `200` + `data: {"status":"connected",…}`；鉴权失败为 APISIX/taskSSE 的 **401/403 JSON**，连接数满为 **503 JSON**——均不会变成 nginx HTML 502。
2. 本案时间线（UTC）：
   - `17:59:12` 浏览器打 SSE → 502
   - `18:00:05` runAll 日志：`start requested (previous status=stopped)`，taskSSE `listening on :8798`
   - `18:01:xx` APISIX/openresty 进程起来；同期 gateway 日志出现对其它上游的 `Connection refused`
3. 即：**runAll 处于 stopped / 正在拉起 DAG 时，边缘 nginx 连不上 APISIX 或 APISIX 连不上 taskSSE** → 快速 502。

## 前端「前置」实际做了什么

`TaskDetail.onMounted` / `effectiveTaskId` watch 在具备 `tenantId`+`workspaceId`+`taskId` 后**立即** `establishSSEConnection`（见 `establishSSEConnection.js`）。

**刻意不闸门**于：

- `isServerStarting` / `runtime_hydrate` / 云 Running
- `fetchContainerTaskUiContext` 完成
- taskSSE `/health` 探活

原因：SSE 与服务器生命周期独立（意图 027）；冷打开需尽早挂上通道以收 `container_heartbeat` / clone 进度等（意图 030）。

因此「为什么没有先执行前置判断」：业务侧**没有**「服务器已启动才建 SSE」的前置；基础设施侧**也没有**在发 EventSource 前探测网关/taskSSE 就绪。502 窗口内任何页面打开都会撞上同一代理失败。

## 正常响应所需条件

| 层级 | 条件 | 失败形态 |
|------|------|----------|
| 边缘 | nginx 上游（APISIX）可达 | HTML 502（本案） |
| 网关 | APISIX 路由 `sse-startup-status`；forward-auth（Cookie）通过 | 401/403 |
| 侧车 | taskSSE `:8798` listen；`X-TaskGateway-Internal-Secret` 匹配 | 403 JSON / 连接拒绝→502 |
| 容量 | `connections < maxConnections` | 503 JSON |
| 前端 | `tenantId`/`workspaceId`/`taskId` 非空 | 不发请求 |

## 验证

```bash
# 侧车直连
curl -sS http://127.0.0.1:8798/health
# 公网（需登录 Cookie；应见 connected 事件，勿期望短连接立刻关闭）
curl -sSN --max-time 3 \
  "https://www.daydaymoney.com/api/tenant/<tid>/workspace/<wid>/task/<task>/cloud/server-startup-status-sse/" \
  -H 'Accept: text/event-stream' -H 'Cookie: …'
# 对照 runAll：若 previous status=stopped 且刚 start，502 属预期瞬态
```

## 预防

1. 运维：runAll 全量重启时，前端/用户侧会出现短暂 API/SSE 502；优先看 `logs/task-sse.log` 是否已有 `listening`，以及 `logs/task-gateway.log` 的 `Connection refused`。
2. 产品：勿把「SSE 未连接 / 502」解读为「服务器启动失败」（意图 027）。
3. 前端在 SSE `onerror` 后**异步探测**确认为 HTML 502/5xx 时，在 SSE 行提示「状态推送服务暂不可用」并退避重连（`scheduleSSEReconnect`）；**禁止**仅凭短耗时失败就置位（易与「服务器启动状态=已停止」并排误解为云机在重启）。不必改为「仅启动中才连 SSE」。

## 相关

- `.ai/09_failure_experience/02_runtime_errors/05_edge_nginx_localhost_bind_502.md`
- `task2app/docs/intents/frontend/task_detail/027_sse_vs_server_lifecycle_status_labels.intent.md`
- `taskFE/app/src/composables/taskDetail/establishSSEConnection.js`
