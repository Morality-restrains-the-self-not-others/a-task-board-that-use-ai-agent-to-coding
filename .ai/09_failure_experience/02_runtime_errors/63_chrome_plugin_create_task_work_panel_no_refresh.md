# [运行时] Chrome 插件创建任务后 work-panel 不刷新

## 现象

通过 `taskChromePlugin` 悬浮面板创建任务成功后，已打开的
`https://www.daydaymoney.com/tenant/{tenant}/work-panel/` 看板不出现新卡，
需手动刷新页面才可见。

## 根因

跨端创建依赖「领域事件 → work-panel SSE → 前端 resync」链路，但三处断点并存：

1. **发布缺失**：`taskTaskService` 创建/删除成功后只发布了 `TASK_STATUS_CHANGED`（进度变更），未发布 `TASK_CREATED` / `TASK_DELETED`。
2. **消费面不全**：fanout intent 配置仅订阅 `TASK_STATUS_CHANGED`（handler 虽已支持 create/delete，但进程未订阅对应 Kafka topic）。
3. **前端丢弃**：`useWorkPanelTaskStatusSse` 仅处理 `task_status_changed`，收到 `task_created` / `task_deleted` 直接 return。

同页创建仍可通过本机 `tasks-updated` → `fetchTodos` 刷新，故「本页创建正常、插件创建需刷」的差异由此产生。

## 修复

1. `taskTaskService`：create/delete 后发布 `TASK_CREATED` / `TASK_DELETED`。
2. `conf/events/domain-events/task_status_changed/config.yaml`：fanout 订阅三类事件；`KAFKA_TOPICS` / `EventTopic` 登记 `task-created` / `task-deleted`。
3. 前端 SSE：`task_created` / `task_deleted` → `onNeedResync` → `fetchTodos`。
4. 部署：创建 Kafka topic、重启 taskTaskService 与 `2_fanout_work_panel_sse`、SPA `runall-lifecycle.sh build`。

## 验收

- fanout health：`subscribed_events` 含 `TASK_CREATED` / `TASK_DELETED`
- Vitest：`useWorkPanelTaskStatusSse.test.js`（resync on create/delete）
- Go：`TestPublishTaskCreatedOnCreate` / `TestPublishTaskDeletedOnDelete`
- 手工：插件建任务后，已打开 work-panel 无需 F5 即出现新卡

## 关联

- 意图：`docs/intents/frontend/work_panel_task_status_sse.intent.md`
- 代码：`taskTaskService/src/events.go`、`useWorkPanelTaskStatusSse.js`、`workpanelfanout`
- 后续同类：`64_task_created_idempotency_collapses_on_user_id.md`（事件已发但 fanout 幂等键用 `user_id` 吞掉后续 TASK_CREATED）
