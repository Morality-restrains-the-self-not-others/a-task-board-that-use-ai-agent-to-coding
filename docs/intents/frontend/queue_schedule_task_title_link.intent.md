# 意图：自动调度安排排队任务标题可点击进入任务详情

## 背景与目标

`/tenant/:tenant/queue-schedule/`（页面标题「自动调度安排」，浏览器壳标题「云端开发」）排队任务表把任务名渲染为不可点击的 `span.font-medium.text-text`。用户看到标题（如「将当前时间，硬编码进 now.md 文件中」）期望能打开对应任务详情。

## 范围与边界

- 范围内：`WorkspaceQueueSchedule.vue` 排队任务列表「任务」列标题改为真实 `<a href>`，目标为既有任务详情路由。
- 范围外：不改后端 queue-schedule API；不改任务 ID 旁注样式；不拦截点击、不 `router.push`。

## 约束与风险

- 导航须遵守元规则 44：真实 `a[href]`，禁止 `@click.prevent` + `router.push`。
- 纯导航链接：`Anti-Replay-OK: 真实 a[href] 打开任务详情，无写请求`。
- 复用 `buildTaskDetailHref`，与工作面板搜索「打开任务」同路径：`/tenant/{tid}/workspace/{wid}/task-detail/{taskId}/`。
- `tenantId` / `workspaceId` / `task_id` 任一缺失时不渲染死链 `#`，退回 `span`。

## 验收标准

1. 有完整 ID 的排队任务标题是 `<a>`，可见文本仍为标题（空标题为「（无标题）」）。
2. `href` 为 `/tenant/{tenant}/workspace/{workspaceId}/task-detail/{taskId}/`。
3. 中键 / Ctrl+点击可新标签打开（浏览器默认行为，无 preventDefault）。
4. 缺少跳转所需 ID 时标题仍为 `span`，不出现 `href="#"`。

## 实施计划

1. 单测断言标题节点为 `a[href]`。
2. 模板用 `buildTaskDetailHref` 渲染链接。
3. 浏览器打开排队调度页核对标题可点。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 点击排队任务标题打开详情 | — | — | 前端 `<a href>` | 浏览器导航至任务详情 | 纯前端导航，无服务端状态变更，无对应事件 |

## 变更记录

- 2026-08-27：新增。排队任务标题由 `span` 改为真实任务详情链接。
