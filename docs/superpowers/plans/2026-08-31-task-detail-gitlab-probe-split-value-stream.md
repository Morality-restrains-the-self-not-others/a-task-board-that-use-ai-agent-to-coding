# 价值流：任务详情立即返回 + GitLab 探活独立请求

- 日期：2026-08-31
- 设计：`docs/superpowers/specs/2026-08-31-task-detail-get-15s-timeout-design.md`

Mapping the approved design into a value stream.

## Related Value Streams

- `task-management` / `todo-crud`：任务 GET
- `task-detail-oauth-*`：OAuth 徽章；本增量把探活从数据 GET 拆出，不改授权协议
- `2026-08-23` 合并请求 30s 超时：同一 GitLab 不可达族；本增量修详情 GET 同步探活

## 用户价值

打开任务详情立刻看到标题/描述/仓库地址；GitLab 宕机不再转圈 15s。探活标记随后出现。

## 增量（单一 MVP）

1. GET project 去掉同步 GitLab enrich
2. GET task `loadProjects` 只读任务表
3. 任务详情关联仓库独立 POST validate-git-repos 显示探活标记

无第二期（磁盘占用独立 GET、lite project URL）。

## 步骤与测试点

| 步骤 | 测试文件 | 测试点 |
|------|----------|--------|
| GET project 不打 GitLab | `project_get_db_only_test.go` | 带 userId + git_repos 时 gitoauth 命中 0 |
| GET task 不等 project hang | `task_get_detail_no_project_hang_test.go` | hang GET project 时详情 <1s |
| 前端 loading 不等探活 | `taskDetailFetchFns` / LinkedProjects 测 | 探活 POST 不在 fetchTaskDetail 内 await |
| mismatch catalog | 既有 `taskProjectsWithDetails.test.js` | 工作区 git_repos vs stored |
