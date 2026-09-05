# 实施计划：任务子树状态与终态门禁

## 任务清单

- [x] 1. Go：`ResolveTerminalKind` / `IsTerminalState`（对齐 taskEvents）
- [x] 2. Go：`listDescendants` BFS + `buildSubtreeResponse(maxDepth)`
- [x] 3. Go：`GET .../subtree/` 路由与 handler
- [x] 4. Go：`DescendantTerminalGate` 接入 PATCH 与 `/switch`
- [x] 5. Go：单测 T1–T4、T6（stub 列名映射）
- [x] 6. 前端：`TaskDetailSubtreeStatusPanel.vue` + fetch
- [x] 7. 前端：`onProgressStatusChange` 解析 409 + data-traceId
- [x] 8. 前端单测 T5；接线 TaskDetail
- [x] 9. 更新 value-stream 测试点
- [x] 10. Review + PR

## 事件契约任务

- [x] 成功终态继续 publish `TASK_STATUS_CHANGED`（既有）
- [x] 门禁拒绝不 publish
