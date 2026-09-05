# ztree 层级落库 + step_full.json COS 归档

- **Date:** 2026-08-23
- **Status:** accepted（/goal 自动采用）
- **Iteration:** ztree-step-full-cos-archive
- **python_api_approval:** not_applicable（全 Go + Vue + onlineServiceJS；无新增 Python endpoint）
- **Architecture change:** 是（目标 **v104 target**，based_on **v103 current**）
- **Page:** 任务详情评论执行细节 — ztree 任务关联 + 执行日志面板
- **ADR:** ADR-0039

---

## 根据当前架构，系统现状

- 共有 2 个 **current** 架构视图：`enterprise-landscape` v103、`application-integration` v103
- **业务层：** Developer / Task Management / Cloud Resource
- **应用层：** taskFE、APISIX、taskCloudService、taskContainerGateway、onlineServiceJS、taskEvents
- **技术层：** MySQL `task_cloud`、Kafka、Redis/SSE、腾讯云 COS（现仅厂商证照桶 `ai-provider-1259712831`）
- 上次 **current** 版本是 **v103**。本次在 v103 基线上设计 **v104**。

📋 架构版本历史（最近）：v103 impersonation inbox ✅ current；v102 模拟登录 archived。

---

## 🔍 Trace 日志分析

用户输入无 `data-traceId` / `trace_id`。跳过 Loki 强制检索。`skipped_no_traceid`。

---

## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `.code-review-graph/graph.db` 存在；`update --brief` 成功 |
| 关键发现 | 图偏前端/脚本，未索引 `taskCloudService` Go 符号 |
| 决策影响 | 爆炸半径由静态检索：`cloud_layer_graph_snapshot`、`handleContainerJobExecutionLogFromDB`、`jobsRuntimeRunJob` close、`AdminVendorDocsStorage` COS 模式 |
| skip 理由 | Go 服务未入图；结论以源码为准 |

---

## 1. 问题

用户在任务详情多选了两块 UI：

1. **ztree 层图** — 「ztree 层级需要保留到数据库中」
2. **执行日志** — 各层执行完成后把 `step_full.json` 提交到 COS；刷新优先读 COS；COS 参数在管理员后台配置

现网已有：

- `cloud_layer_graph_snapshot`（层图 JSON 快照，layer-graph-push UPSERT）
- `cloud_job_execution_event`（步骤摘要，**不含** `agent_step_full.json` 全文）
- 容器磁盘 `runtime/job_logs/trae_agent_json/{jobId}/step_N/agent_step_full.json` — **随容器释放消失**
- 厂商证照 COS（taskAiProvider，另一桶/路径规则，不可直接当执行日志桶）

缺口：执行完成后的 **完整 step_full** 没有对象存储；关容器后复查只能看到 023 表摘要。

---

## 2. 选定方案

### 2.1 层图继续 MySQL 快照（需求 1）

**不另建层级表。** `cloud_layer_graph_snapshot.graph_json` 已是评论级 last-write-wins 树。本迭代：

- 保持 UPSERT；job close 已有 `mirrorLayerGraphToTaskCloudSSE`
- GET `container-layer-graph` 继续 **先读 DB**，无树才 live 回退
- 快照 `extra` 可挂 `step_full_object_key` 便于对账（可选）

### 2.2 step_full 走腾讯云 COS（需求 2）

Owner：**taskCloudService**。容器 **没有** COS 密钥。

写路径（无 ticker）：

```
job close（completed/failed/interrupted）
  onlineServiceJS 收集该 job 的 agent_step_full.json
  POST server-container-token/job-step-full-push/
  Cloud：渲染 object key → PutObject（或 backend=local 写 MEDIUMTEXT）
       → UPSERT cloud_job_step_full_object
       → 发布 JobStepFullArchived
```

默认 object key（用户指定；占位符可在后台改）：

```
workspace_{workspaceId}/task_{taskId}/comment_{commentId}/layer_{layerId}/step_full.json
```

对象 key 必须含 `{layerId}`：任务各层独立归档，**一评论一层的 bundle**，避免多层写同一对象互相覆盖（同 job 重跑后写覆盖前写，旧层日志丢失）。同一评论同一层多 job：文件为 **bundle**（按 `job_id` 合并 `jobs[]`），后写覆盖同 job，不同 job 追加。若 pathRule 含 `{jobId}` 则一 job 一对象。pathRule 含 `{layerId}` 时，归档首写的同层回退合并按 `layer_id` 过滤，禁止跨层合并。

读路径（需求 2 刷新优先 COS）：

```
GET container-job-execution-log
  1) 有 COS/local 对象 → source=saas_cos，agent_steps 来自 step_full
  2) 否则 023 事件表 → source=saas_db
  3) 不因容器 502 挡复查
```

### 2.3 管理员后台配置 COS（需求 3）

系统管理新页 `/system-admin/step-full-cos/`：

| 字段 | GET | PATCH |
|------|-----|-------|
| backend | cos / local | 是 |
| bucket, region | 明文 | 是 |
| keyPrefix, pathRule | 明文 | 是 |
| secretId / secretKey | 仅 `secret_configured: bool` | 写则落 `step-full-cos.local.yaml`（gitignore） |

非密钥写回 `conf/taskCloudService/step-full-cos.yaml` 并热加载。密钥永不回显、不进 git。出站 Client `tracelog.DirectClient`（禁止环境 Proxy）。SSE-COS AES256。

权限：`authz.IsPlatformStaff`（super_admin / employee）。

### 2.4 明确不做

- 不把克隆日志上 COS
- 不让容器直连 COS
- 不新建微服务 / Python API
- 不替换 023 表；COS 是全文，023 是直播摘要

---

## 3. 方案对比

| # | 方案 | 结论 |
|---|------|------|
| 1 | 仅 MySQL MEDIUMTEXT 存全文 | 大 blob 压热库；否决作为权威 |
| 2 | COS + 指针表 + local 回退 | ⭐ 关容器可复查；测试/无密钥可 local |
| 3 | 浏览器预签名 PUT | 容器已有 JSON，多一跳 CORS；否决 |
| 4 | 复用证照桶同一 pathRule | 混 PII 与执行日志；否决（可同 bucket 不同 prefix） |

---

## 4. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外 |
|---------|--------|--------|--------|------|
| 层图上报 | LayerGraphSnapshotPersisted | 既有 | 无（publish-only） | — |
| job 完成归档 step_full | JobStepFullArchived | `handleJobStepFullPush` | 无（publish-only） | — |
| 管理员改 COS 规则 | StepFullCOSConfigUpdated | admin PATCH | 无 | — |
| 页面 hydrate 日志 | — | GET | 读 COS/DB | 纯查询 |

幂等键：`(workspace_id, task_id, comment_id, job_id)` UPSERT。禁止用 tenant/user 当键。

---

## Domain Concept Inventory

- **BC:** Cloud（日志 owner）/ Container runtime（临时执行者）
- **Aggregates:** LayerGraphSnapshot `(ws,task,comment)`；JobStepFullArchive `(ws,task,comment,job)`
- **VO:** StepFullObjectKey（pathRule 渲染结果）
- **Port:** `StepFullObjectStore` { Put, Get } — COS / memory / MySQL-local

---

## 改动文件清单（预览）

- `dataMigrate/taskCloudService/040_cloud_job_step_full_object.sql`
- `taskCloudService/domain/step_full_*.go`
- `taskCloudService/src/step_full_*.go`、admin handler、inbound、hydrate
- `conf/taskCloudService/config.yaml` + `step-full-cos.yaml`
- `trae-agent/onlineServiceJS` job close 上报
- `taskFE` 系统管理页 + 路由 + 侧栏
- `taskGateway` APISIX 路由
- `taskEvents` EventTopic
- `taskAgentSupport` inbound allowlist
