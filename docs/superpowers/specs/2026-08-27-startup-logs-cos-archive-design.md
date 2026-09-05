# 评论启动日志 COS 归档

- **Date:** 2026-08-27
- **Status:** accepted（goal-mode 自动采用）
- **Iteration:** ccb-startup-logs-cos-archive
- **Architecture:** v114 (based_on v113)
- **ADR:** ADR-0045

## 当前架构理解

- 基线 **v113 current**：推荐资格动态 scene 码；与本增量无关。
- 工作台「云端开发」评论执行区 `h4`「启动日志」来自 `TaskDetailServerStartStatusPanel`；时间线权威源是 `taskCloudService` 的 `cloud_comment_container_binding_logs_{00..15}`（ADR-0023）。
- 同类对象存储已落地：`step_full.json` 经同一 COS bucket 归档（ADR-0039），客户端 `cos-go-sdk-v5` + SSE-COS AES256，直连禁止环境 Proxy。
- 列表 API 仍返回 `bindings[].logs`；前端按消息去重合并 SSE。

## 目标与成功标准

1. 每条评论启动日志在 MySQL 分片写入成功后，**best-effort** 合并进 COS 对象 `startup_logs.json`（同一评论一份 bundle，按 log `id` 幂等合并）；**Put 成功后删除该分片行**。
2. 冷打开 list API 从 COS 指针表 + 对象还原时间线；分片仅保留 Put 失败残留；前端契约不变。
3. COS 失败不阻断 MySQL 写入、不删除分片行（启动进度仍可见）。
4. 复用 step_full 的 bucket/region/密钥；独立 `startupLogsPathRule`；管理员页可改路径；密钥不回显。
5. 无 Python 新 API（🐍 not_applicable）。

## 方案（采用）

MySQL 分片作为热路径 write-ahead buffer；COS 为评论级 JSON 权威归档（与 step_full 同客户端）。Put 成功后驱逐分片行。

```
insertCCBLogRow (shard, Snowflake id)
  → persistCommentStartupLogBestEffort
       Render key: workspace_{ws}/task_{tid}/comment_{cid}/startup_logs.json
       Get existing bundle → Merge by log id → Put
       UPSERT cloud_comment_startup_log_object 指针
       Put 成功 → DELETE 该 log 的分片行
       publish CommentStartupLogArchived (key = workspace:task:comment:log_id)

listCommentContainerBindingLogsIn
  → 同 task 的指针行 load COS/local payload（优先）
  → shard SELECT 仅补 COS 未覆盖的 id
  → sort created_at,id
```

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| COS 替换 MySQL 为唯一权威 | 热路径每行 RMW 延迟；SSE 实时性差；违背 ADR-0023 分片热表 |
| 仅终态（released/failed）整包上传 | 启动中途刷新/崩溃窗口丢失行；用户要求「也存 COS」覆盖全时间线 |
| 每行一个 COS 对象 | 对象数爆炸、list 贵 |
| 复用 step_full pathRule 文件名 | 与执行全文混 key，覆盖风险 |

## Python 新 API 门禁

not_applicable — 无 Django/Flask 新路由。

## 领域模型（Step 6 摘要）

- **聚合**：`CommentStartupLogBundle`（评论级时间线）
- **实体**：`StartupLogEntry`（id = Snowflake，与分片行同一 id）
- **值对象**：COS object key（pathRule 渲染 + SanitizePathToken）
- **端口**：复用 `StepFullObjectStore` Put/Get
- **热表**：既有 CCB log 分片（不改片键）
- **指针表**：`cloud_comment_startup_log_object` 每评论一行

## 事件

| 业务意图 | 事件名 | Topic | 键 | 消费者 |
|---------|--------|-------|-----|--------|
| 启动日志写入 COS | CommentStartupLogArchived | comment-startup-log-archived | ws:task:comment:log_id | publish-only |
| 管理员改路径（同 COS 页） | StepFullCOSConfigUpdated | 既有 | step-full-cos | publish-only |

## 架构交付物

- `docs/architecture/v114-application-integration-20260827-0200-cursor.puml`
- `docs/architecture/v114-enterprise-landscape-20260827-0200-cursor.puml`
- 伴生 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
