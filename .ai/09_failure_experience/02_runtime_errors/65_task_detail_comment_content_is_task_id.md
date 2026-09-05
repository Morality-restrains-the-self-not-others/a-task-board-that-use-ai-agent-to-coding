# 任务详情评论正文显示为任务编号

## 现象

评论 Feed（`#comments-container .conversation-feed p.mt-1.text-sm`）正文为 `task_<snowflake>`，用户期望看到真实评论文案。

## 根因

- **不是前端误渲染字段**：`TaskDetailConversationFeed` 正确绑定 `c.content`。
- 本地库 `comments.content` 与 `task_id` 完全相同；日志显示客户端 `POST .../comments/` 以该字符串为 body 写入（本例两次，约创建任务后 1 分钟）。
- 代码库中无「自动把 task_id 写成评论」路径；高概率为误粘贴任务详情 URL 末段或「任务辅助信息」里的任务 ID。

## 修复

1. 创建评论时拒绝 `content == task_id`（Go `errMsgCommentContentIsTaskID` + 前端 `isCommentContentTaskIdEcho`）。
2. 列表/详情/Feed 过滤历史 echo 评论；本机已 `DELETE FROM comments WHERE content = task_id`。
3. 用户需重新提交真实评论文案；任务标题「写一个 hello world」仍在任务身份区，不会自动变成评论。

## 验收

```bash
go test -run TestCreateCommentRejectsContentEqualToTaskID ./src
npm test -- --run src/composables/taskDetail/buildDisplayComments.test.js
sqlite3 data/task_task.db "SELECT COUNT(*) FROM comments WHERE content=task_id;"  # 0
```
