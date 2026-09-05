# taskContainerGateway 可观测性 — Grafana/Loki 补齐（原 Vite Proxy 篇，已修订）

**日期：** 2026-05-31  
**状态：** 已实现（2026-05-31）；**2026-07-05 修订** — API 路由真源为 taskGateway，Vite proxy 观测 hop 降为可选/废弃  
**关联：**  
- `2026-05-31-task-container-gateway-go-design.md`（验收：同 trace_id ≥3 条）  
- `2026-05-31-relay-git-commit-pending-trace-observability-design.md`  
- `2026-05-28-centralized-logs-grafana-traceid-design.md`  
- 现场排障：`container-layer-git-commit` 502 `django internal unreachable`（Django 挂死时网关无 request 级日志）

---

## 1. 现象与目标

### 1.1 现象

用户在 Grafana **Trace Log Explore**（`job=runall` + `trace_id`）排查 `container-layer-git-commit` 时：

| 跳 | 期望 | 实际（2026-05-31 排障时） |
|----|------|---------------------------|
| 浏览器 → **taskGateway** | access log + trace | Inc 1 时部分环境仍经 Vite proxy，**无**统一 edge 日志 |
| taskGateway → **taskContainerGateway** `:8014` | `http_request` + `forward_stage` | **无**；Loki `service` 下拉无 `task-container-gateway` |
| Gateway → Django internal | `validate_session` / `resolve_container_target` | 仅 saas-backend 侧（且 Django 挂死时无完成日志） |
| Gateway → onlineServiceJS | `http_request` | ✅ 常有 |

**运行时证据（2026-05-31）：**

- `/tmp/runall-logs/task-container-gateway.log` 仅 2 行启动 `log.Printf`，无 per-request 行。
- Loki `label/service/values`：`go-relay`, `onlineServiceJS`, `saas-backend`, `task-auth` — **不含** `task-container-gateway`、`taskFE`。
- 文本搜索 `taskContainerGateway` 可得启动行，但 `service=*`、`trace_id` 无值，**无法**与浏览器 `X-Trace-Id` 关联。
- `go-relay` 使用 `tracelog.Middleware` + JSON `http_request`；`taskContainerGateway` **未**接入同类设施。

### 1.2 目标

1. **同一 `trace_id`** 在 Loki 至少可见：**taskContainerGateway** 入站 + 各转发阶段 +（已有）onlineServiceJS + saas-backend internal/访问日志。
2. Grafana `service` 变量能选中 **`task-container-gateway`**（与 runAll 服务名一致）。
3. **开发环境**可选（**已废弃为主路径**）：Inc 1 曾考虑 Vite 对单条路由打 `proxy_forward` hop 日志；**现 API 不经 Vite**，应以 **taskGateway access log** 为第一 hop。
4. 不改动 Promtail/Loki 基础架构；复用现有 JSON 契约与 `trace-log-explore` 仪表盘。

### 1.3 非目标

- 生产 nginx/ingress 访问日志（另案；本地 dev 以 **taskGateway + Go 网关** 为主观测点）。
- 完整 OpenTelemetry span 树（沿用现有 Tempo 增量，本设计仅补 **日志**）。
- ~~将 Vite proxy 全量 API 日志化~~（**不再适用**：`vite.config.js` `proxy: {}`）。
- Grafana 仪表盘结构大改。

---

## 2. 价值流影响

| 流 | 影响 |
|----|------|
| **platform-centralized-logging** | **扩展** — 新增 `task-container-gateway` JSON 契约；**task-gateway** access log |
| **task-container-gateway** | **扩展** — `gateway-git-commit-forward` 验收含 Loki trace 可检索 |
| **task-detail-runtime-relay** / relay 排障 | 间接 — git-commit 502/pending 可定位在 validate / resolve / upstream 哪一段 |
| **increment2-grafana-trace-dashboard** | 无结构变更；`service` 下拉随新 JSON `service` 字段自动出现 |

**字段（value-stream 登记建议）：**

| name | description |
|------|-------------|
| `task-container-gateway.stdout_json.trace_id` | Go JSON slog `http_request` / `forward_stage` |
| `task-container-gateway.stdout_json.forward_stage` | `inbound` \| `django_validate` \| `django_resolve` \| `upstream_forward` \| `error` |
| `task-gateway.stdout_json.trace_id` | APISIX / taskGateway 入站 access（推荐第一 hop） |
| `taskFE.stdout_json.trace_id` | ~~可选 Vite dev proxy hop~~ **deprecated** — API 不再经 Vite |

---

## 3. 领域概念清单（供 /5-ddd）

| 概念 | 边界 | 说明 |
|------|------|------|
| **ContainerGatewayAccessLog** | 容器网关 | 单次浏览器入站 HTTP 的 method/path/status/duration_ms/trace_id |
| **ContainerForwardStageLog** | 容器网关 | 网关内阶段事件（Django internal、upstream OSJS） |
| **DevProxyHopLog** | ~~前端 dev~~ | **deprecated** — 由 taskGateway 入站 log 替代 |
| **TraceCorrelation** | 平台可观测性 | `X-Trace-Id` 在 Promtail pipeline 中提升为 Loki label |

**领域事件（日志语义，非持久化）：**

- `GatewayRequestReceived` / `GatewayForwardCompleted` / `GatewayForwardFailed`

---

## 4. 根因分析

### 4.1 taskContainerGateway

- P1 实现仅 `log.Printf` 启动信息，**无** HTTP middleware。
- 设计文档要求 `structured log → Loki`（`task-container-gateway-auth-design.md`），**未在代码中落地**。
- Promtail 从 JSON payload 提取 `service`、`trace_id` label；纯文本行 → `service=*`、`trace_id` 空 → Trace Log Explore 过滤不到。

### 4.2 taskGateway / Vite（历史）

- **当前：** 浏览器 `fetch` 直连 taskGateway（`apiBaseUrl`）；Vite **`server.proxy: {}`**，不承担 API 路由。
- **历史（Inc 1）：** 曾用 Vite 内置 `server.proxy` 将部分 `/api` 转到 `:8014`；该路径已废弃，避免与 `routes.yaml` 双轨。
- 架构上 **taskGateway 是 dev/prod 统一的 edge**；Vite 仅页面/HMR。

### 4.3 与 git-commit 502 的关系

- 网关 `djangoPost` 30s 超时返回 `django internal unreachable` 时，**若无** `forward_stage` 日志，Grafana 仅见 onlineServiceJS 或空白，误判为「请求没到后端」。
- 补齐网关阶段日志后，可按 trace 看到 `django_validate` 超时/失败，与 Django 挂死根因（如 Session 线程竞态，见 relay-git-commit 设计）**串联**。

---

## 5. 方案对比

| 方案 | 摘要 | 优点 | 缺点 | 推荐 |
|------|------|------|------|------|
| **A — Go tracelog + 阶段日志（推荐）** | 复用 `go_relayToTrae/tracelog` 模式迁入 `taskContainerGateway`；handler 内打 `forward_stage` | 满足终态验收；与 monorepo 一致；生产/dev 同代码 | 需抽共享包或复制薄层 | **是** |
| **B — 仅 runAll tee 解析访问日志** | 不改 Go，靠正则从 stderr 猜 | 改动小 | 无 trace_id、无阶段、不可靠 | 否 |
| **C — Vite 全量 proxy 日志** | 所有 `/api` 代理都打 JSON | dev 可见性最大 | 噪声大、非生产路径、维护成本高 | 否（改为 **opt-in 单路由**） |
| **D — 跳过 Vite，文档说明** | API 经 taskGateway | 与终态一致 | — | **已采纳为默认** |

**推荐组合：A（必须）+ D（默认）+ ~~C′ Vite hop~~（deprecated，除非 legacy 环境仍用 Vite proxy）。**

---

## 6. 详细设计（方案 A + 可选 C′）

### 6.1 共享 tracelog（Go）

**选项 6.1a（推荐）：** 新建 `taskContainerGateway/tracelog/`，从 `go_relayToTrae/src/tracelog` **复制**最小子集（`Init`, `Middleware`, `Emit`, `Header`），`service` 固定为 `task-container-gateway`（与 runAll 服务名、Loki 文件名一致）。

**选项 6.1b：** 后续 monorepo 抽 `shared/tracelog` 供 taskAgentSupport / taskAIEndPoint 跟进 — **本迭代不阻塞**。

**`main.go`：**

```go
tracelog.Init("task-container-gateway")
handler := tracelog.Middleware(corsMiddleware(mux))
```

**`Middleware` 产出（与 go-relay 对齐）：**

```json
{
  "time": "...",
  "level": "INFO",
  "msg": "http_request",
  "service": "task-container-gateway",
  "trace_id": "9d2f3ff4-d861-48c4-8a11-d5be82997d94",
  "method": "POST",
  "path": "/api/tenant/.../container-layer-git-commit/",
  "status": 502,
  "duration_ms": 30003
}
```

- 透传请求头 `X-Trace-Id`；缺失则生成（与 go-relay 相同规则）。
- 响应头回写 `X-Trace-Id`。

### 6.2 转发阶段日志（`handlers.go` / `django_client.go`）

在现有流程打点（**不记录 token/cookie**）：

| forward_stage | 时机 | 额外字段 |
|---------------|------|----------|
| `django_validate` | `djangoValidateSession` 前后 | `django_status`, `duration_ms` |
| `django_resolve` | `djangoResolveTarget` 前后 | `django_status`, `duration_ms` |
| `upstream_forward` | `forwardToOnlineService` 前后 | `upstream_url`（path 级，无 query token）, `upstream_status`, `duration_ms` |
| `error` | 提前返回 4xx/502 | `detail`（短字符串） |

实现方式：`tracelog.Emit(ctx, level, "forward_stage", map[string]any{...})` 或 `slog.InfoContext` 与 `http_request` 相同 JSON handler。

**敏感信息：** `Authorization`、`Cookie`、`access_token` **禁止**写入日志。

### 6.3 Django internal 侧（薄增量，可选 P1.5）

`validate_session` / `resolve_container_target_view` 入口各打一条 `container_gateway_internal` JSON（`trace_id` 来自 `X-Trace-Id`），便于 Loki 中 **saas-backend** 行与网关 `forward_stage` 对齐。  
**非必须**若网关阶段日志已足够；可作为 P1.5。

### 6.4 ~~Vite dev proxy hop~~（deprecated — 可选 C′）

> **2026-07-05：** API 已统一经 **taskGateway**（`vite.config.js` `proxy: {}`）。第一 hop 观测应使用 **taskGateway access log**，本节仅作 Inc 1 历史记录。

~~**文件：** `task2app/front_project/app/vite.config.js`~~

~~对 `container-layer-git-commit` 的 proxy 项增加 `configure` hook…~~

**现行替代：** 确保 `taskGateway/logs/taskgateway-access.log`（或 APISIX access log）含 `X-Trace-Id`；Promtail 解析 `service=task-gateway`。

### 6.5 Promtail / Grafana

- **无需改** `promtail.yaml` pipeline（已支持 JSON `service` + `trace_id`）。
- 新 JSON 行写入后，`label/service/values` 自动出现 `task-container-gateway`（及可选 `taskFE`）。
- Trace Log Explore 查询不变：`{job="runall", service=~"$service", level=~"$level"} |= "$trace_id" | json`

### 6.6 测试

| 层 | 文件 | 覆盖 |
|----|------|------|
| Go 单元 | `taskContainerGateway/tracelog/tracelog_test.go` | trace 规范化、Middleware 写 JSON |
| Go 单元 | `taskContainerGateway/src/handlers_observability_test.go` | mock Django/upstream，断言 stderr 含 `forward_stage` + `trace_id` |
| Promtail | 扩 `AiMonitor/scripts/test_promtail_config.py`（可选） | 文档化示例行可解析 |
| Vite | ~~container-vite-proxy 测试~~ | **deprecated** — 改测经 taskGateway 的路径 |

---

## 7. 验收标准

- [x] 对任意带 `X-Trace-Id` 的 `POST …/container-layer-git-commit/`，Loki 同 trace_id **≥2 条** `service=task-container-gateway`（`http_request` + 至少一个 `forward_stage`）。
- [x] Grafana Trace Log Explore 的 `service` 下拉可选 **task-container-gateway**。
- [x] Django 不可达场景：可见 `forward_stage=django_validate` + `django_status=502` 或 `duration_ms≈30000`，**无需**猜测「请求未到达网关」。
- [x] （~~可选 dev taskFE proxy_forward~~）taskGateway 入站 log 可关联同一 trace_id。
- [x] `cd taskContainerGateway && go test ./...` 全绿。

---

## 8. 分阶段交付

| Phase | 内容 | 工期粗估 |
|-------|------|----------|
| **P1** | taskContainerGateway tracelog + Middleware + forward_stage | 0.5–1d |
| **P1.5** | Django internal 两行 JSON（可选） | 0.25d |
| **P2** | ~~Vite proxy_forward~~ taskGateway access JSON（可选） | 0.25d |
| **P3** | taskAgentSupport / taskAIEndPoint 同模式（跟进债） | 另案 |

---

## 9. 风险与回滚

| 风险 | 缓解 |
|------|------|
| 日志量增大 | 仅 info；路径已截断；无 body |
| 复制 tracelog 漂移 | 测试锁定 JSON 字段；后续抽 shared |
| Vite 日志噪声 | **N/A** — API 不经 Vite |

回滚：移除 Middleware，恢复纯 `log.Printf`。

---

## 文档修订（2026-07-05）

- API 路由真源为 **taskGateway**（`vite.config.js` `proxy: {}`）；Vite proxy hop 观测 **deprecated**。
- 第一 hop 日志：`task-gateway` access + `task-container-gateway` forward_stage。
- 关联修订：`2026-05-31-task-container-gateway-l0-job-stream-design.md`、`auth-design.md`、plan/value-stream。

---

## 10. 审批后下一步

- `/3-value-stream-价值流` 更新 `platform-centralized-logging` 与 `task-container-gateway` 步骤  
- `/6-plans` 生成实施计划（TDD：先 failing test 断言 JSON 行）  
- 与 **Django Session thread-local 热修**（relay-git-commit 设计）可并行，互不阻塞
