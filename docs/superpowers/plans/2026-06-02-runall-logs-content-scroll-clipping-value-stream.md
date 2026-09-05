# Value Stream: runAll 日志内容区滚动裁切修复

> Derived from design: `docs/superpowers/specs/2026-06-02-runall-logs-content-scroll-clipping-design.md`

## Value Summary

本地开发者在 `:9999` 状态页打开服务日志后，能滚到最底并阅读完整末行，不被面板裁切遮挡。

## Related Value Streams

- **runall-services-list-scroll-clipping**（`2026-06-02-runall-services-list-scroll-clipping-design.md`）：互补 — 修复 workspace/服务列表裁切；本变更修复日志内容区独立 flex bug。
- **runall-logs-panel-layout-jitter**（`2026-05-29`）：互补 — 打开日志时服务列表 scroll 保持；本变更不涉及。

Greenfield for **logs-content scroll** — 无现有 value-stream.yaml 条目。

## End-to-End Flow

[用户点击「日志」] → [日志面板打开，API 返回 log lines] → [用户在 `#logs-content` 内滚动] → [末行完全可见]

## Value Increments

### Increment 1: Flex 约束修复（Thin Slice）
**Value to user:** 滚到最底时最后一行日志完全可见  
**Scope:** `.logs-panel-header { flex-shrink: 0 }`；`.logs-content { min-height: 0 }` 替换 `min-height: 280px`  
**Depends on:** nothing

### Increment 2: 回归测试
**Value to user:** CI 防止 flex 约束回归  
**Scope:** Playwright 末行可见断言；`ui_test.go` CSS 片段断言  
**Depends on:** Increment 1

## YAML Config

**不写入 `value-stream.yaml`。** 纯前端布局修复，无 `<service>.<table>.<field>` 变更。
