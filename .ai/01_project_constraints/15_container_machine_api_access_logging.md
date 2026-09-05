# 机器容器 API 访问日志（元规则）

## 基本信息

- **权威接口清单**：以 [`docs/skills/saas-container/saas-machine-container.md`](../../docs/skills/saas-container/saas-machine-container.md) 为准（路径、请求/响应字段随版本变更；**不要**在本文件重复罗列 URL）。容器 inbound 清单见 `trae-agent/onlineServiceJS/skill.md`。索引：[`docs/skills/saas-container/README.md`](../../docs/skills/saas-container/README.md)。
- **适用范围**：skill 中约定的、任务级前缀下的**容器令牌与相关 API**（挂载在 `api/tenant/<tenant_id>/workspace/<workspace_id>/task/<task_id>/comment/<comment_id>/cloud/` 下、路径含 `server-container-token/` 或 scoped inbound）。容器 env `TaskApiEndPoint` **必须**含 `/comment/{cid}/`，禁止旧 `…/task/{task_id}/cloud`。若 skill 增删改路由，须同步调整实现，并更新 skill §7 变更清单。
- **历史**：Django `Saas_project/skillList/machine_container.md` 与 `container_machine_api_access_middleware.py` 已随 `task2app` 退役。

## 日志原则（固定形态）

相对项目根目录（与 `paths.conf` 中 `LOG_DIR` 解析结果一致），若仍写文件访问日志，路径形态为：

```text
logs/container_run/tenantId_<tenant_id>/workspaceId_<workspace_id>/taskId_<task_id>.log
```

说明：

- `tenant_id`、`workspace_id`、`task_id` 取自请求 URL 路径中与任务上下文一致的片段。
- **禁止**在日志中写入请求 JSON 体内的 `access_token`、`refresh_token` 等敏感字段；记录元数据（时间、HTTP 方法、路径、响应状态码、客户端 IP、`trace_id`）。
- 当 HTTP 状态码为 **4xx/5xx** 时，可追加脱敏后的响应 `detail`；响应里若出现 `*_token` 等敏感键名则跳过或脱敏。
- 当前 Go 实现优先用 `tracelog` / Loki（`X-Trace-Id`），不必再写 Django 中间件文件日志；新增实现须遵守上述脱敏原则。
