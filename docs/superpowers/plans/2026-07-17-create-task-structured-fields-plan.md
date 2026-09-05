# 实施计划：创建任务结构化可选字段

## Tasks

- [x] 1. 新增 `createTaskDescriptionCompose.js`（字段定义、compose、parse）+ 单元测试
- [x] 2. `buildCreateTaskDraft` 增加空结构化字段
- [x] 3. `CreateTaskBasicFields.vue` 增加可折叠可选字段 UI（依赖三态 select）
- [x] 4. `CreateTaskModal.vue`：提交前 compose；编辑打开时 parse；弹窗可滚动
- [x] 5. 扩展 `CreateTaskModal.test.js` 覆盖标签可见与提交 compose
- [x] 6. 跑 vitest；`bash scripts/runall-lifecycle.sh build` 发布 SPA

## 事件契约任务

不适用（无新增 MQ）。
