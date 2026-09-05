# 实施计划：编辑/创建非顶层须选上层交付物

- **日期**: 2026-07-15
- **设计**: `docs/superpowers/specs/2026-07-15-edit-parent-deliverable-on-category-change-design.md`

## Task 1：意图文档

- [ ] 新增/更新 `task2app/docs/intents/frontend/task_detail/026_*.intent.md` + test-intent
- [ ] 更新 024 变更记录（编辑可选上层）
- [ ] 更新 intent_index
- [ ] 007 注明创建+编辑弹窗共用

## Task 2：payload 助手（TDD）

- [ ] `appendParentTaskToCreatePayload` 顶层写入 `parent_task: ''`
- [ ] 单测覆盖顶层清空 / 非顶层写入

## Task 3：CreateTaskModal 编辑态

- [ ] 去掉 `v-if="!editingTask?.id"`
- [ ] 编辑态同样走 `resolveParentDeliverableBlockedReason`
- [ ] 更新 CreateTaskModal.test.js

## Task 4：详情编辑

- [ ] `startEdit` 回填 `parent_task`
- [ ] 拉取 workspace todos（或 prop）供候选
- [ ] IdentityPanel 编辑态渲染 CreateTaskParentDeliverableField
- [ ] `saveEdit` PATCH `parent_task` + 门禁
- [ ] 单测 IdentityPanel + saveEdit

## Task 5：验证

- [ ] vitest 相关用例通过
- [ ] 对照验收清单自检
