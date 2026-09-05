# 实施计划 — 创建任务可选加入自动调度队列

- 日期：2026-08-27

## 切片

- [x] **S1** 纯函数 + 单测：`isWorkspaceAutoScheduleEnabled`、`shouldShowCreateTaskQueuedAutoRun`、`appendQueuedAutoRunToCreatePayload`、`resolveQueuedAutoRunHint`
- [x] **S2** `useCreateTaskAutoRun` GET 一次 + 嵌套勾选 UI + 组件测
- [x] **S3** draft/edit/submit 带 `queued_auto_run`
- [x] **S4** Go `deferImmediateAutoRunStart` + 创建测：queued 无 start-vm；仅 auto_run 仍有
- [x] **S5** 价值流图测试点 + INDEX

## 文件

- `taskFE/app/src/utils/workspaceAutoScheduleEnabled.js` (+test)
- `taskFE/app/src/utils/createTaskQueuedAutoRun.js` (+test)
- `taskFE/app/src/composables/useCreateTaskAutoRun.js` (+test)
- `taskFE/app/src/components/CreateTaskAutoRunSection.vue` (+test)
- `taskFE/app/src/composables/useWorkPanelDeliverableForm.js`
- `taskTaskService/src/queued_schedule.go` / `create_task.go` / `task_handlers_update.go` / `auto_run_test.go`
