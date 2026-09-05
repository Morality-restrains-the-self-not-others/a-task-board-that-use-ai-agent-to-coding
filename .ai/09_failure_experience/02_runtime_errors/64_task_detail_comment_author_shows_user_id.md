# 任务详情评论作者显示裸 user_id

## 现象

任务详情页评论时间线（`#comments-container .conversation-feed`）作者名显示雪花 `user_id`（如 `850256676127797248`），而非公司成员昵称。

## 根因

1. `taskTaskService` 人类评论序列化将 `created_by.username` 直接设为 `created_by_id`（auth user_id），未走已有的 `batch-resolve-task-owners`。
2. 前端 `TaskDetailConversationFeed` 直接渲染 `created_by.username`；协作映射 `collaboratorNameById` 仅按公司成员 `id` 索引，无法用 user_id 反查昵称。

## 修复

- 后端：评论列表 / 创建 / 任务详情三路 feed 调用 `enrichCommentsCreatedBy`（复用 `batchResolveTaskOwners` → 公司 `member_name`）。
- 前端：`buildCollaboratorNameById` 同时按 `id` 与 `user`/`user_id` 索引；ConversationFeed 用 `resolveCommentAuthorDisplayName` 展示。

## 验收

- Go：`go test -run TestEnrichCommentsCreatedByResolvesCompanyNickname ./src`
- 前端：`npm test -- --run src/utils/taskCardPeopleDisplay.test.js`
- 页面：评论作者 span 显示成员昵称，不再是裸数字 ID
