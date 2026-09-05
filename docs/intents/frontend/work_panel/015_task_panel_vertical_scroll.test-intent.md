# 测试意图：工作面板任务区纵向可滚动

对应：`015_task_panel_vertical_scroll.intent.md`

| ID | 场景 | 期望 | 覆盖 |
|----|------|------|------|
| T1 | 源码契约 | `#task-panel-container` 含 `overflow-y-auto`，不含 `overflow-hidden` | `TaskPanel.fillViewport.test.js` |
| T2 | CSS 高度链 | `.task-column` 无 `min-height:360px`；`.task-cards-container` 含 `min-height:0` | 同上 |
| T3 | 贴底冒烟 | `#task-panel-board-other` 底边贴近视口底（短过滤栏） | `WorkPanel.kanban-fill-viewport.playwright.test.js` |
