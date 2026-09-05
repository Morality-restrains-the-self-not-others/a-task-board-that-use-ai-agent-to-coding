# 任务详情评论头像显示 initials SVG 而非真实头像

## 现象

任务详情页评论时间线（`#comments-container .conversation-feed`）作者 `<img class="w-9 h-9 rounded-full">` 的 `src` 为 `data:image/svg+xml...`（姓名首字占位图），与用户在导航栏/个人资料中的真实头像不一致。

## 根因

1. `TaskDetailConversationFeed` 始终用 `initialsAvatarDataUri(authorLabel)`，从不读取 `created_by.avatar_url`。
2. `batch-resolve-task-owners` 只返回 `username` / `user_id` / `company_member_id`，评论 enrich 链路无头像字段。
3. 工作区协作者序列化也不带 `avatar_url` / `member_avatar_url`（公司头像迁租户后多为空）。

## 修复

- Django：`batch_resolve_task_owners_internal` 批量查 `UserProfile.avatar`，响应增加 `avatar_url`（相对 `/media/...`）。
- Go：`resolvedOwner.AvatarURL` → `created_by.avatar_url`（`enrichCommentsCreatedBy` / `createdByObject`）。
- 前端：`resolveCommentAuthorAvatar` + ConversationFeed `authorAvatar`（真实 URL 优先，否则 initials）；可选 `collaboratorAvatarById` 双索引回退。

## 验收

- Django：`pytest tests/test_batch_resolve_task_owners_internal.py`
- Go：`go test -run TestEnrichCommentsCreatedByResolvesCompanyNickname ./src`
- API：任务详情 `comments[].created_by.avatar_url` 非空（有个人头像时）
- 页面：评论 `<img src>` 为 `/media/user_avatars/...`（或绝对 URL），而非 `data:image/svg+xml`

## 后续（OPT-20260721-006）

公司成员头像已在 taskTenantService 落地：DB `member_avatar` 存相对路径，公开 URL  
`/api/tenant/{cid}/accounts/members/{mid}/avatar`；协作列表与 batch-resolve **优先公司头像**，再回退个人 `UserProfile`。
