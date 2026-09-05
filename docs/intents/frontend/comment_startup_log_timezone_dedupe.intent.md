# 意图：评论启动日志本地时钟一致且同一步骤不重复

## 背景与目标

任务详情评论「启动日志」把同一 UserData 步骤显示两次，时钟相差 8 小时（例如 23:30:59 与 15:30:58）。根因：SSE 用浏览器本地时间，list API 把 UTC DATETIME 墙钟当成本地；去重未忽略 `trace_id=` 后缀。

目标：同一行为只显示一行；时钟与 Loki/浏览器本地时区一致（CST 为 23:xx 而非 15:xx）。

## 范围与边界

- 范围内：`mergeBackendBindingLogs` / `appendBindingLogLine` / `mergeBindingAndServerStartupLogs` 去重键；`created_at` 无时区则按 UTC 解析。
- 范围外：不改 UserData `report_progress` 上报；不删除落库 `trace_id=` 后缀。

## 约束与风险

- 去重不得误吞不同步骤（「安装完成」vs「服务已就绪」）。
- 冲突时保留带 `trace_id=` 的行，便于 Grafana 对照。

## 验收标准

1. SSE 行与后端 `… trace_id=` 行规范化后相同 → 面板只留一行。
2. `created_at=2026-08-13T15:30:59Z` 与 naive `2026-08-13 15:30:59` 展示同一本地时钟。
3. 阶段日志（排队/启动实例）与 UserData 行仍按时间线共存。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|-----------|--------|--------------|---------|
| 启动日志去重与时钟对齐 | — | — | — | — | 纯前端展示；后端时间解析见 `comment_container_binding_log_utc` |

## 实施计划

1. `startupLogDedupeKey` + `upsertStartupLogLine` + `parseBindingLogDate`。
2. vitest 覆盖 8 小时重复样例。
3. 后端 RFC3339 `Z` 见对应 cloud 意图。

## 变更记录

- 2026-08-13：初版（Loki `bb1157e1b936e3d8eb7e7b05`）
