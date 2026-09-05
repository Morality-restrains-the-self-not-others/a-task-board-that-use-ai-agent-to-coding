# 工作面板任务状态 SSE 推送

## 意图

工作面板打开后，任务卡片进度列/完成态的变更，以及他端（含 Chrome 插件悬浮面板）创建/删除任务，应通过 SSE 推送到同工作区各浏览器，避免仅本机刷新可见。

## 验收行为

1. 进入 `/tenant/{id}/work-panel/` 时仍先 HTTP 拉取 todos。
2. 拉取完成后建立 `work-panel-events-sse` EventSource。
3. 收到 `task_status_changed` 后，对应任务卡片列位/完成态更新，无需整页刷新。
4. 收到 `task_created` / `task_deleted` 后触发 `fetchTodos` 整板同步，无需手动刷新页面。
5. 切换工作区或离开页面时关闭 SSE。
6. 心跳与 connected 帧不改变看板数据。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 他端改进度列后本机卡片更新 | TaskStatusChanged | TASK_STATUS_CHANGED | taskTaskService（他端 PATCH） | 2_fanout_work_panel_sse → SSE → WorkPanel patch | — |
| 本端拖拽后本地刷新 | TaskStatusChanged | TASK_STATUS_CHANGED | 同上（本端 PATCH） | 本机亦可收 SSE（幂等） | — |
| 插件/他端创建任务后看板出现新卡 | TaskCreated | TASK_CREATED | taskTaskService create | SSE `task_created` → fetchTodos | — |
| 他端删除任务后看板移除 | TaskDeleted | TASK_DELETED | taskTaskService delete | SSE `task_deleted` → fetchTodos | — |
| 初载拉取 todos | — | — | — | — | 无对应事件：纯查询 |
