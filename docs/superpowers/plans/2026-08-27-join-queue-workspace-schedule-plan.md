# 实施计划：入队前确认工作空间自动调度

- [ ] 导出 `isWorkspaceAutoScheduleEnabled` / `fetchWorkspaceAutoScheduleEnabled`（Red 单测）
- [ ] Toggle：未启用弹窗且不 PATCH；前往设置 `location.assign`
- [ ] Toggle：已启用 GET + PATCH；GET 失败 data-traceId
- [ ] 更新 click-guard 与既有 join 测例的 GET mock
- [ ] 更新 `comment_queue_serial_execution` 提示文案（可选对齐）
- [ ] 价值流图补测试点；跑 Vitest
