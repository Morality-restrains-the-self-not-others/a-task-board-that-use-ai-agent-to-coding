# DDD：任务详情读路径与 GitLab 探活

- 日期：2026-08-31
- NFR：`docs/superpowers/plans/2026-08-31-task-detail-gitlab-probe-split-nfr-clarification.md`

## Bounded contexts

- 任务协作（taskTaskService）：Task + TaskProjectSnapshot（stored_repo_address）
- 项目（taskProjectService）：Project.git_repos（DB）
- Git OAuth：探活只读

## 读模型

`GitRepoProbeStatus { repo_url, token_status }` — 非聚合，不落库。

## 业务意图 → 事件

纯查询，无事件（意图文档 B-086 已写例外）。

## 端口

不新增端口。删除任务读路径对 Project 全量 GET 的同步依赖。
