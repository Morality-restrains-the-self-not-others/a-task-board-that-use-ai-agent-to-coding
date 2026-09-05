# 意图：评论级「服务器内容」Tab

## 背景与目标

任务详情页任务级「服务器内容」卡片在无评论作用域时展示「缺少评论ID」，与「执行细节 / 服务器运行状态」已按评论隔离的心智不一致。`GET /api/cloud/compute/server-content` 必须带 `comment_id`（path kv），内容属于该评论 CSC。

目标：把「服务器内容」迁入每条评论执行细节 Tab，与「服务器运行状态」并列；任务级仅保留迁出提示。

## 范围与边界

- 范围内：taskFE 任务详情评论执行细节 Tab、评论级 server-content 快照、任务级 ServerConfig 迁出提示。
- 范围外：不改 Cloud `server-content` 契约；不新增后台轮询；历史启动记录仍在任务级。

## 约束与风险

- 拉取必须带该评论 `comment_id`；无 ID 不发请求。
- 两评论并发刷新不得互相覆盖。
- 禁止 `setInterval` / 挂载自动轮询 Describe；切到「服务器内容」Tab 的一次性 GET 视为用户操作。

## 验收标准

1. 评论执行细节 Tab 栏在启用运行态 Tab 时出现「服务器内容」，`data-testid=comment-execution-tab-server-content`。
2. 点击该 Tab 后 `#server-content` 插槽内出现 `data-testid=server-content-section` 且 `data-comment-id` 为该评论 id。
3. 刷新请求 URL 含 `/comment_id/{id}/`；无 comment_id 时文案「缺少评论ID」且不发请求。
4. 任务级原内容面板不再拉取，改为 `data-testid=server-content-moved-hint`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 按评论拉取服务器内容 | — | HTTP GET server-content | 用户点 Tab / 刷新内容 | 展示该评论 CSC 页面 HTML | 纯查询、一次性 |

## 实施计划

1. 评论级 content 快照 store + `bindCommentServerContentPanel`。
2. 执行细节增加「服务器内容」Tab 与插槽；CommentsSection 挂载原内容组件。
3. 任务级改为迁出提示；去掉 mount 自动 fetch。

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-20 | 任务级卡片改为评论 Tab | 用户选择：服务内容应为评论级别 |
