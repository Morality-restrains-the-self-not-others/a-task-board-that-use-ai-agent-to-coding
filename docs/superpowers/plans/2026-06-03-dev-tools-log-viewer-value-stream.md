# Value Stream: 开发工具统一日志查看

> Derived from design: `docs/superpowers/specs/2026-06-03-dev-tools-log-viewer-design.md`

## Value Summary

开发者在 runAll Web UI 一键查看「生成配置副本」「同步配置副本」「清空全部数据库」「初始化全部数据库」「清空 Grafana 可观测数据」五个开发工具的操作日志，并可清空日志，替代当前仅靠 `alert()` 弹窗的碎片化反馈。

## Related Value Streams

- **[2026-05-31-dev-database-reset-value-stream](2026-05-31-dev-database-reset-value-stream.md)** — extension：在已有清空/初始化数据库按钮之上增加日志查看能力
- **[2026-05-31-runall-grafana-clear-all-observability-value-stream](2026-05-31-runall-grafana-clear-all-observability-value-stream.md)** — extension：在已有一键清空可观测数据按钮之上增加日志查看能力

本流是纯 UI 增强 + 文件日志基础设施，不改变现有开发工具的业务逻辑。

## End-to-End Flow

[开发者点击「日志查看」] → [下拉选择工具] → [GET /api/dev/logs 返回文件尾 N 行] → [日志面板实时显示] → [开发者执行开发工具操作] → [操作过程中逐行写入日志文件] → [2s 自动刷新看到新日志] → [点击「清空日志」→ confirm → 文件清空]

## Value Increments

### Increment 1: 后端 — DevToolLogRecorder + API（Thin Slice）
**Value to user:** 五个开发工具的操作日志写入持久化文件，可通过 API 读取和清空
**Scope:**
- `domain/dev_tool_log_recorder.go` — 接口定义 + 常量
- `infrastructure/dev_tool_log_recorder.go` — 文件实现（Append/Tail/Clear/ClearAll）
- `ui.go` — `GET /api/dev/logs` + `POST /api/dev/logs/clear` handler
- `runner.go` — Runner 持有 recorder，五个操作中嵌入日志写入
- `domain/dev_tool_log_recorder_test.go` — 接口契约测试
- `infrastructure/dev_tool_log_recorder_test.go` — 文件读写测试
- `ui_test.go` — API 端点测试
**Depends on:** nothing

### Increment 2: 前端 — 日志查看按钮 + 面板复用
**Value to user:** 在开发工具栏点击「日志查看」，下拉切换五个工具，实时看日志，一键清空
**Scope:**
- `status.html` — 新增按钮、下拉选择器、面板模式切换、JS 逻辑
- `ui_test.go` — UI 片段测试（按钮、下拉、五个 tool 选项）
**Depends on:** Increment 1
