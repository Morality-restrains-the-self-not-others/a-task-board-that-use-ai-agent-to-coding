# Value Stream: runAll 一键清空 Grafana 可观测数据

> Derived from design: `docs/superpowers/specs/2026-05-31-runall-grafana-clear-all-logs-design.md`

## Value Summary

开发者在 runAll Web UI 一键清空日志、Trace、指标历史，使 AI/人工在 Grafana 查询时仅看到当前会话数据。

## Related Value Streams

- **platform-centralized-logging**（extension）：在 increment3 深链/retention 之上增加 increment4 全栈 reset
- **2026-05-28-centralized-logs-grafana-traceid-value-stream**：基础 Loki MVP，本流在其上扩展

## End-to-End Flow

[开发者点击「清空 Grafana 可观测数据」] → [confirm] → [runAll API 清内存+tee] → [AiMonitor 脚本 reset 4 volumes] → [Grafana 查询无历史] → [新日志/trace/指标正常写入]

## Value Increments

### Increment 1: 本地日志源清空 + API 骨架（Thin Slice）
**Value to user:** 一键后 runAll logs panel 全部为空，tee 文件截断  
**Scope:** `Truncate`/`Clear` 修复 + `POST /api/observability/clear-all` 仅本地段  
**Depends on:** nothing

### Increment 2: Loki + Promtail volume reset
**Value to user:** Grafana Loki Explore 无历史日志  
**Scope:** `reset_observability_storage.sh` 处理 loki_data + promtail_data  
**Depends on:** Increment 1

### Increment 3: Tempo + Prometheus volume reset
**Value to user:** Trace Explore / 指标面板无历史  
**Scope:** 脚本扩展 tempo_data + prometheus_data，完整 stop/start 顺序  
**Depends on:** Increment 2

### Increment 4: UI 按钮 + 验收测试
**Value to user:** observability bar 一键操作 + 确认对话框  
**Scope:** status.html + ui_test + value-stream.yaml 登记  
**Depends on:** Increment 3
