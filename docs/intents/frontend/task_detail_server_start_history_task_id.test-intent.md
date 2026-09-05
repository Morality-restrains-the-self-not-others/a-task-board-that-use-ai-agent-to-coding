# 测试意图：任务详情历史启动记录使用路由任务ID

- **对应功能意图**: `task_detail_server_start_history_task_id.intent.md`
- **日期**: 2026-08-30

## 用例

| 编号 | 场景 | 前置 | 操作 | 期望 |
|------|------|------|------|------|
| T1 | 路由有 taskId | `props.task={}`，`route.params.taskId=task_881388002226499584` | 拉取历史 | 发起请求且 query 含该 task_id；message 不是「缺少任务ID」 |
| T2 | 仅 pk | `props.task={pk:task_pk_1}` | 拉取历史 | 请求 `task_id=task_pk_1` |
| T3 | props.taskId 优先 | `props.taskId=prop-task`，route 另有 taskId | 拉取历史 | 请求 `task_id=prop-task` |
| T4 | 全缺 | task/taskId/route 均无任务 ID | 拉取历史 | 不请求；message=缺少任务ID；无 data-traceId |

## 自动化落点

- `taskFE/app/src/composables/taskDetail/useServerConfigRuntimeFetch.test.js`
- `taskFE/app/src/utils/serverConfigRouteHelpers.unit.test.js`
- `taskFE/app/src/components/ServerConfig.logic.unit.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailRuntimeSection.test.js`
- `taskFE/app/src/composables/hardwarePanel/useServerConfigHardwarePanel.taskId.unit.test.js`
