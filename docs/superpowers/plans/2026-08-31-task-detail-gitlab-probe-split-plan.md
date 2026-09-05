# 实施计划：任务详情立即返回 + GitLab 探活独立请求

- 日期：2026-08-31
- 设计：`docs/superpowers/specs/2026-08-31-task-detail-get-15s-timeout-design.md`

## 事件任务

纯查询：无契约 / publish / 消费者。已在 B-086 写例外。

## Tasks

- [x] **T1** GET project 不调用 GitLab enrich（红→绿）
  - 测：`taskProjectService/src/project_get_db_only_test.go`
  - 改：`handleGetProject` 去掉 `enrichProjectGitReposStatus` / `enrichProjectGitRepoDiskSizes`
- [x] **T2** GET task `loadProjects` 不 GET project（红→绿）
  - 测：`taskTaskService/src/task_get_detail_no_project_hang_test.go`；更新 mismatch 测例为 fail-open
  - 改：`task_store.go` `loadProjects`
- [x] **T3** 任务详情关联仓库独立探活 UI
  - 测：composable + LinkedProjectsViewMode
  - 改：watch 仓库 URL → POST validate-git-repos；不 await 于 `fetchTaskDetail`
- [x] **T4** `projectHTTP` `Transport.Proxy=nil`（与 probe client 一致）
