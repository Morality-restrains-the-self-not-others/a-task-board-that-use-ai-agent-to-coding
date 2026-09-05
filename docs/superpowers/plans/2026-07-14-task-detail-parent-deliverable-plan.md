# 实施计划：交付物详情上层交付物

- **日期**: 2026-07-14
- **设计**: `docs/superpowers/specs/2026-07-14-task-detail-parent-deliverable-design.md`

## Tasks

- [x] T1: 意图文档 `024_交付物详情展示上层交付物.intent.md` + `.test.intent.md`
- [x] T2: RED — `TaskDetailTaskIdentityPanel` 单测：顶层隐藏 / 非顶层展示
- [x] T3: GREEN — Panel UI + props（parent route / title / loading）
- [x] T4: GREEN — `useTaskDetail` 拉取 parent title + 构造 `parentDeliverableRoute`
- [x] T5: 透传 `TaskDetail.vue`；更新 smoke stub props
- [x] T6: 跑 vitest 相关用例；必要时更新 value-stream 测试点引用
- [ ] T7: Review / Ship（PR）

## 验证命令

```bash
cd task2app/front_project/app && npx vitest run src/components/task-detail/TaskDetailTaskIdentityPanel.auto-run.test.js
```
