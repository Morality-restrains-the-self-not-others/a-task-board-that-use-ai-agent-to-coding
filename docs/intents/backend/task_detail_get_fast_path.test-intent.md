# 测试意图：任务/项目数据 GET 与 GitLab 探活分离

- 日期：2026-08-31
- 意图：`docs/intents/backend/task_detail_get_fast_path.intent.md`
- 设计：`docs/superpowers/specs/2026-08-31-task-detail-get-15s-timeout-design.md`

## 可执行测试（落地后）

| 场景 | 文件（预期） | 断言 |
|------|-------------|------|
| GET project 时 GitLab hang | `taskProjectService` `project_handlers` 测 | **<200ms** 200；不调用 `validateGitReposForUser` |
| GET task / `loadProjects` | `taskTaskService` | **不**请求 `GET /api/projects/.../{id}`；含 `stored_repo_address` |
| 任务详情 loading vs 探活 | `taskFE` `taskDetailFetchFns` / project-repo 测 | `fetchTaskDetail` resolve 后 loading false；探活 POST 另发 |
| 探活失败 | 前端单测 | 任务数据保留；错误节点 `data-traceId` |
| mismatch 仍可用 catalog | `taskProjectsWithDetails.test.js` | 工作区 `git_repos` vs stored 仍能标红 |

## 手工对照

- GitLab `115.29.110.74` 不可达：任务详情首屏快开；探活 badge 随后失败/未知，页面不转圈 15s。

## 非目标

- 不要求 Loki 有日志才能验收。
