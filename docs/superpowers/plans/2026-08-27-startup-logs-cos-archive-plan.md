# 实施计划：评论启动日志 COS 归档

- **Date:** 2026-08-27
- **Design:** `docs/superpowers/specs/2026-08-27-startup-logs-cos-archive-design.md`

## 切片

- [ ] S1 Domain key + bundle merge（红绿）
- [ ] S2 DDL 指针表 `042_cloud_comment_startup_log_object.sql`
- [ ] S3 insert 后 COS persist + list hydrate
- [ ] S4 conf `startupLogsPathRule` + admin GET/PATCH + 事件 Topic
- [ ] S5 管理页表单项 + 单测
- [ ] S6 意图/ADR/架构/value-stream.yaml/INDEX

## 事件任务

- 契约：`CommentStartupLogArchived` → `comment-startup-log-archived`
- publish：`persistCommentStartupLogBestEffort` 成功后
- 消费者：无（publish-only）

## 验证

```bash
cd taskCloudService && go test ./domain -count=1
cd taskCloudService && go test ./src -count=1 -run 'CCBLog|StartupLog|StepFullCOS'
cd taskEvents && go test ./config -count=1 -run EventTopicCloud
```
