# Value Stream: runAll 日志面板悬浮 + 左侧拖动

> Derived from design: `docs/superpowers/specs/2026-06-02-runall-floating-log-panel-design.md`

## Value Summary

runAll 开发者在使用 Web UI 查看服务日志时，日志面板悬浮覆盖在服务列表上方，不再挤压左侧布局；左边缘可拖动调整面板宽度。

## Related Value Streams

Greenfield — no existing value streams for this UI layout change. Related runAll UI streams exist (color system, logs scroll clipping) but no overlap or dependency.

## End-to-End Flow

[点击「日志」按钮] → [面板从右侧滑入，覆盖服务列表] → [自动拉取 /api/logs] → [用户查看日志，可拖左边缘调宽] → [Close / ESC 关闭]

## Value Increments

### Increment 1: Floating Log Panel (Thin Slice)
**Value to user:** 日志面板不再挤压服务列表，始终看到完整的服务状态
**Scope:**
- CSS: `.logs-panel` 改为 `position: absolute` overlay
- CSS: 移除 `.pane-divider`，面板左边缘内置拖拽手柄
- JS: 复用现有 `layoutState.resizing` 机制处理左边缘拖动
- JS: 调整 `openLogsPanel` / `closeLogsPanel` / `setLogsPanelWidth`
**Depends on:** nothing

### Increment 2: Polish (动画 + 视觉)
**Value to user:** 面板开关有平滑过渡，拖拽手柄有清晰视觉提示
**Scope:**
- CSS transition: 滑入/滑出动画
- 拖拽手柄视觉优化（紫色渐变 + 指示条）
- 面板阴影增强（悬浮感）
**Depends on:** Increment 1
