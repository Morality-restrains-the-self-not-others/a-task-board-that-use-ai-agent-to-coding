# Value Stream: onlineServiceJS Grafana 日志转发

> Derived from design: `docs/superpowers/specs/2026-05-29-online-servicejs-grafana-log-forwarding-design.md`

## Value Summary

开发与排障人员可在 Grafana 中按 `service=onlineServiceJS` 检索子进程日志，且 Node.js 错误堆栈显示为单条完整记录。

## Related Value Streams

- **platform-centralized-logging**（extension）：在 `increment2-json-log-contract` 基础上补全 go-relay **子进程** JSON 契约
- **task-detail-relay-debug-agent-observability**（间接受益）：DEBUG_AGENT 日志可按服务过滤

## End-to-End Flow

relay 启动 onlineServiceJS → 子进程 stdout/stderr → go-relay 合并多行 + 透传 JSON → runAll tee `go-relay.log` → Promtail → Loki（`service=onlineServiceJS`）→ Grafana Trace Log Journey

## Value Increments

### Increment 1: 子进程 JSON 透传（Thin Slice）
**Value to user:** Loki 出现 `onlineServiceJS` 服务标签；`logJson` 行含正确 `trace_id`
**Scope:** `ForwardChildLine` + `appendSubprocessLog`
**Depends on:** platform-centralized-logging increment1/2

### Increment 2: 多行堆栈合并（Core Value）
**Value to user:** Grafana 单条日志展示完整 error stack
**Scope:** `subprocessLogGrouper` + process scanner 集成
**Depends on:** Increment 1

### Increment 3: 验收与文档（Essential Support）
**Value to user:** runbook 可查；value-stream 有回归测试锚点
**Scope:** 单元测试、runbook 补充、value-stream.yaml 登记
**Depends on:** Increment 2
