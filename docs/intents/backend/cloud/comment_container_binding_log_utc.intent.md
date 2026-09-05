# 意图：评论 binding 启动日志 created_at 按 UTC 读写

## 背景与目标

`cloud_comment_container_binding_logs.created_at` 以 UTC 墙钟写入 naive DATETIME。读回只用 RFC3339，驱动 `loc=Local` 会给同一墙钟打上 `+08:00`，list JSON 让前端显示成 15:xx，与 SSE 的 23:xx CST 并排重复。

目标：读回把 `YYYY-MM-DD HH:MM:SS` 当作 UTC；JSON 输出 RFC3339 `Z`。

## 范围与边界

- 范围内：binding / binding_log 的 created_at、updated_at 字符串解析与 list JSON。
- 范围外：改 MySQL 列类型为 TIMESTAMP；停写 `trace_id=` 日志后缀。

## 约束与风险

- 写入路径已是 `time.Now().UTC().Format("2006-01-02 15:04:05")`，不得改成本地墙钟否则存量错位。
- 解析须同时接受空格 DATETIME、`T`、`Z`、错误 `+08:00` 标签（墙钟仍当 UTC）。

## 验收标准

1. `parseCloudUTCDateTime("2026-08-13T15:30:59+08:00")` 等于 UTC 15:30:59（不是 07:30）。
2. list `logs[].created_at` 为 `…Z` RFC3339，且与插入时刻的 UTC 墙钟一致。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|-----------|--------|--------------|---------|
| 纠正启动日志时间解析 | — | — | — | — | 既有 SSE_MESSAGE 持久化附属字段，无新领域事实 |

## 实施计划

1. `parseCloudUTCDateTime` / `formatCloudUTCJSON`。
2. store Scan 与 log JSON 改用上述函数。
3. Go 单测表驱动 + list API `created_at` 断言。

## 变更记录

- 2026-08-13：初版
