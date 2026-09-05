# Value Stream: runAll Web UI 按钮交互色彩

> Derived from design: `docs/superpowers/specs/2026-05-31-runall-ui-color-system-design.md`

## Value Summary

运维人员在 `:9999` 状态页点击服务操作按钮时，能立即通过 **背景 + 边框 accent + 光晕** 确认点击已触发，减少误操作与重复点击。

## Related Value Streams

- **runall-explicit-lifecycle-commands**：扩展 — 本变更不改变生命周期命令语义，仅增强同一 UI 上的点击反馈。
- **runall-grafana-clear-all-observability**：扩展 — 修复 `obs-clear-all` 按钮点击反馈（`flashButtonClick` → `pulseClickFeedback`）。

Greenfield for **presentation/color tokens** — 无现有 value-stream.yaml 条目对应 UI 色彩规则。

## End-to-End Flow

[用户点击按钮] → [CSS `:active` + JS `.is-clicked` 脉冲] → [用户感知 accent 反馈] → [原有 API 调用继续执行]

## Value Increments

### Increment 1: 设计令牌 + 默认按钮反馈（Thin Slice）
**Value to user:** 任意 `.action-btn` / `.logs-panel-btn` 点击可见灰蓝 accent 反馈  
**Scope:** `:root` CSS 变量；default `:active` / `.is-clicked`；`clickFeedbackDurationMs = 300`  
**Depends on:** nothing

### Increment 2: 语义 Variant accent
**Value to user:** 重启/日志/清空按钮分别显示蓝/紫/橙按下态  
**Scope:** `.restart-btn`、`.logs-btn`、`.clear-logs-btn`、`.obs-clear-all-btn` 规则  
**Depends on:** Increment 1

### Increment 3: Bugfix + 回归测试
**Value to user:** 可观测栏「清空 Grafana」按钮点击无 JS 报错；CI 锁定色彩规则片段  
**Scope:** `pulseClickFeedback` 修复；`ui_test.go` 片段断言  
**Depends on:** Increment 1

## YAML Config

**不写入 `value-stream.yaml`。** 设计文档已声明：纯前端 UX，无 `<service>.<table>.<field>` 变更。
