# 实施计划：终态按评论 CSC 释放

- 日期：2026-08-14
- 设计 / NFR / DDD：同前缀 docs

## 任务

- [ ] T1 红灯：`TestInternalListByTask` — 模板+评论，comments 不含模板
- [ ] T2 绿灯：`listCommentConfigsForTask` + `GET list-by-task` + openapi-internal
- [ ] T3 红灯：`TestDispatchNoRunningResourceSuccess` — 空模板/无评论 → success
- [ ] T4 改 `TestDispatchNoConfigRetry` 期望为 success
- [ ] T5 `cloudconfig.ListByTask` + ConfigRow.CommentID
- [ ] T6 handler 按评论循环；停机信封带 comment_id/instance_id/region_id
- [ ] T7 双评论两条 STOPPED；Starting 无 instance no-op
- [ ] T8 跑 `go test` taskCloudService + taskEvents/taskstatuschanged

## 事件任务

- 复用 TaskStatusChanged / CloudServerStopped；更新意图对照表（已写）
- 无新事件类型

## 验证命令

```bash
cd taskCloudService && go test ./src -count=1 -timeout 120s -run 'ListByTask|InternalList'
cd taskEvents && go test ./internal/handlers/taskstatuschanged -count=1 -timeout 60s
```
