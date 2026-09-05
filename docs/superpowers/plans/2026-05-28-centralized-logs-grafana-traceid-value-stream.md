# 价值流：多服务日志集中收集与 Grafana traceId 查询

> 设计：`docs/superpowers/specs/2026-05-28-centralized-logs-grafana-traceid-design.md`

## Value Summary

开发者在本地 runAll 全栈运行时，可在 Grafana 输入一次 `traceId`，检索跨服务（至少 saas-backend）的应用日志，无需逐服务翻 runAll 日志面板。

## Related Value Streams

- **runall-log-copy-gitoauth-port-conflict-recovery** — 并存：runAll UI 单服务日志复制仍保留；本流提供跨服务 Loki 视图。
- **relay-token-audit-observability** — 扩展：audit DB 外的应用日志可按 `trace_id` 在 Loki 检索。
- **task-detail-relay-debug-agent-observability** — 扩展：DEBUG_AGENT 详细日志与常规日志一并进入 Loki（无 Promtail 过滤隔离）。
- **runall-stability-first** / `service-runtime-observability` — 扩展：平台级集中日志为运行时可观测性增量。

## End-to-End Flow

[开发者发起带 X-Trace-Id 的 API 请求] → [各服务 stdout 写日志含 trace_id] → [runAll tee 到 RUNALL_LOG_ROOT] → [Promtail tail + 提取 label] → [Loki 存储] → [Grafana Explore `{trace_id="..."}`] → [开发者看到跨服务日志]

## Value Increments

### Increment 1: Loki MVP 薄切片（Thin Slice）
**Value to user:** 在 Grafana 手动输入 traceId，能查到 saas-backend 对应请求日志行。  
**Scope:** AiMonitor 增 Loki/Promtail/Grafana datasource；runAll 文件 tee；Promtail regex 提取 Django `[trace_id=...]`。  
**Depends on:** 现有 AiMonitor、TraceIdMiddleware、runAll 编排。

### Increment 2: 跨服务 JSON 契约 + Dashboard
**Value to user:** 一次 relay 启动链路，同一 traceId 下可见 saas-backend + go-relay + 相关 Go 服务日志。  
**Scope:** Go 中间件、JSON slog/Formatter、Promtail JSON pipeline、Grafana「Trace Log Journey」仪表盘。  
**Depends on:** Increment 1。

### Increment 3: 体验与运维（Future）
**Value to user:** runAll UI 跳转 Grafana、Loki 7 天保留策略。  
**Scope:** 深链、retention、文档。  
**Depends on:** Increment 2。
