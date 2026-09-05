# Value Stream: Trace Log Explore level 过滤

> Derived from design: `docs/superpowers/specs/2026-05-30-trace-log-explore-level-filter-design.md`

## Value Summary

开发者在 Grafana Trace Log Explore 可按日志级别（info/warn/error 等）过滤，与 service、trace_id 组合排查链路问题。

## Related Value Streams

- **platform-centralized-logging**（`value-stream.yaml`）：**扩展** — 在 `increment2-grafana-trace-dashboard` 之上增加 level 模板变量与 LogQL。

## End-to-End Flow

[开发者打开 Trace Log Explore] → [选择 service / level / trace_id] → [Loki label + 行过滤] → [日志面板仅显示匹配级别]

## Value Increments

### Increment 1: Dashboard level 变量 + LogQL（Thin Slice）
**Value to user:** 下拉选择 level，面板即时过滤  
**Scope:** `trace-log-explore.json` + dashboard JSON 测试  
**Depends on:** Promtail 已有 `level` label（已交付）
