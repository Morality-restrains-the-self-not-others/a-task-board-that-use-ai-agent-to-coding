# 功能意图：任务详情历史启动记录使用路由任务ID

- **日期**: 2026-08-30
- **状态**: 已实施
- **页面**: `/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/`

## 背景与目标

任务详情「历史服务器启动记录」面板（`data-testid=server-start-history-message`）在 URL 已含 `taskId` 时仍展示「缺少任务ID」。根因是 compute 拉取只读 `props.task.id`，未走 `resolveTaskRouteIds`（props.taskId → task.id|pk → route.params.taskId）。

## 范围与边界

- 范围内：ServerConfig 历史/内容/运行态/Workbench/停机请求的 taskId 解析；relay 上下文与硬件启动同源回退。
- 范围外：不改后端契约；任务身份面板展示仍以已加载 `localTask` 为准。

## 约束与风险

- 禁止再手写 `props.task?.id` 作为 compute API 的唯一 taskId。
- 纯前端校验「缺少任务ID」不得伪造 `data-traceId`。
- 无对应服务端状态变更，不发领域事件。

## 验收标准

1. URL 含 `/task-detail/:taskId/` 时，点击历史记录不展示「缺少任务ID」（有 ID 则发请求）。
2. `task` 仅有 `pk` 或仅有 `props.taskId` 时同样发起请求。
3. 三者皆无时仍展示「缺少任务ID」且不请求。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 查看历史服务器启动记录 | — | — | — | — | 纯前端只读拉取，无服务端状态变更 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-30 | compute taskId 改走 resolveTaskRouteIds，含路由回退 |
