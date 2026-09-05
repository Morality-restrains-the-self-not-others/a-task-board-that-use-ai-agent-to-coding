# 设计：容器网关 → onlineServiceJS 上游路径补齐 tenant/workspace/task

- **日期**: 2026-07-09 20:35
- **状态**: 已采用（goal-mode 自动决策）
- **作者**: claude
- **相关**: `taskContainerGateway` L0 转发、`onlineServiceJS` 入站路由

## 1. 问题

任务详情「启动」后，用户在 **onlineServiceJS** 启动日志中看到：

```text
GET /api/layers/{layer_id}/files?max_files=3000
GET /api/layers
GET /api/jobs
GET /api/repos/clone-log/{layer_id}
GET /api/repos/bootstrap-clone-log?...
```

这些路径**没有** `tenantId` / `workspaceId` / `taskId`。

对照 **task-container-gateway** 日志，同一 `trace_id` 的**入站**路径已是完整作用域：

```text
GET /api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/container-layer-files/
→ upstream_url=http://127.0.0.1:8765/api/layers/{layer_id}/files
```

即：浏览器 → 网关（有三段 ID）正确；网关 → 容器（裸 `/api/...`）导致容器侧日志与排障缺少任务作用域。

## 2. 目标（完成标准）

1. 凡经 **taskContainerGateway** 转发到 onlineServiceJS 的 HTTP，上游 URL 须含  
   `/api/tenant/{tenantId}/workspace/{workspaceId}/task/{taskId}/…`
2. onlineServiceJS **兼容**旧裸路径（容器 UI、e2e、直连）与新 scoped 路径。
3. 容器 `http_request` 日志的 `path`（`originalUrl`）在网关转发场景下可见三段 ID。
4. 既有 L0 单测与网关构建通过。

## 3. 方案（采用）

与 go-relay `clear-logs` 的 path 作用域一致：**作用域落在 path，不靠 query/body 补全**。

| 层 | 变更 |
|----|------|
| Gateway L0 / layer-graph / job-stream / git-commit | 上游根改为 `{base}/api/tenant/{t}/workspace/{w}/task/{task}`，其后接原 `/layers`、`/jobs`、`/repos/...` |
| onlineServiceJS | 入站 rewrite：将 `/api/tenant/.../task/.../(rest)` 归一为 `/api/(rest)`，**保留** `originalUrl` 供访问日志 |
| 浏览器公网路径 | **不变**（仍 `…/cloud/compute/container-*`） |

示例：

```text
旧 upstream: http://127.0.0.1:8765/api/layers/L1/files?max_files=3000
新 upstream: http://127.0.0.1:8765/api/tenant/T/workspace/W/task/TASK/layers/L1/files?max_files=3000
```

## 4. 非目标

- 不改 onlineServiceJS 业务路由 handler 签名。
- 不强制 Django thin-forward（网关关闭时）同步改路径（仍走裸 `/api`；OSJS 双兼容）。
- 不改容器内 Web UI 对裸 `/api` 的调用。

## 5. 风险

- 若 OSJS 未部署 rewrite 而网关已发 scoped URL → 404。须同批发布网关与 onlineServiceJS（或先 OSJS 后网关）。
