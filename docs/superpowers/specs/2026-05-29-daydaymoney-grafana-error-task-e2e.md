# DaydaymoneyGrafana 报错 → localhost:4000 自动建任务 E2E 核对

**日期：** 2026-05-29  
**状态：** 已核对（自动化测试 + 配置清单）

## 链路

1. **Grafana 插件** `DaydaymoneyGrafana`：Loki 轮询 + 全局 `window.onerror` → POST `errorReportUrl`
2. **默认 URL**：`http://localhost:4000/api/grafana-errors/`（`AppConfig.tsx`）
3. **Vite :4000**：`/api/*` 代理到 Django `:8001`（`vite.config.js`）
4. **Django**：`POST /api/grafana-errors/` → `create_error_task_from_grafana_payload` 创建 `Todo` 任务

## 插件必填配置

| 字段 | 说明 |
|------|------|
| `errorReportUrl` | 保持 `http://localhost:4000/api/grafana-errors/` |
| `tenantId` | 与 task2app 租户 ID 一致 |
| `workspaceId` | 与 workspace ID 一致 |
| `ingestToken` | 与 `GRAFANA_ERROR_INGEST_TOKEN` / `port_config` 一致（若启用） |
| `lokiPollEnabled` | 需 Loki 错误自动建任务时设为 true |

## 自动化验收

```bash
cd task2app/Saas_project
pytest tests/test_grafana_error_ingest.py -q
```

覆盖：服务层建任务、60s 去重、HTTP 201、`task_url` 指向 `http://localhost:4000/tenant/...`。

## 手工 E2E（可选）

1. runAll 启动 `saas-backend` + `taskFE`（:4000）
2. Grafana 安装并启用 DaydaymoneyGrafana，填写 tenant/workspace
3. 在 Explore 触发一条带 `level=error` 的 Loki 日志，或打开含 JS 错误的 Dashboard
4. 在 task2app 对应 workspace 任务列表中出现标题含报错摘要的新任务

## 常见失败

| 现象 | 原因 |
|------|------|
| 403 | `ingestToken` 与 Django `GRAFANA_ERROR_INGEST_TOKEN` 不一致 |
| 400 tenant/workspace | 插件未填 tenantId/workspaceId 且未配置环境变量 |
| 402 | 租户余额不足（`InsufficientBalanceError`） |
| 无任务 | `lokiPollEnabled=false` 且未触发全局 JS 错误上报 |
