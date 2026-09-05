# DDD：PR 回复与一键合并

- **日期:** 2026-08-22
- **限界上下文:** Task（评论）+ GitOauth（凭证与远端 Git 写）

## 实体 / 值对象

**TaskComment（既有）扩展字段：**

- `ParentCommentID` optional
- `GitPR` optional value object: `HTMLURL`, `Provider`

**MergeRequestRef（GitOauth domain）：**

- Provider: github | gitlab
- Host, ProjectPath, Number/IID
- 从 html_url 解析；非法则拒绝

## 聚合

- 评论仍属 Task 聚合边缘（现有 comment 表，不拆新聚合）
- Merge 不在 Task 内执行；GitOauth 应用服务编排 token + 远端 API

## 端口

- `MergeRequestPort.GetStatus(ctx, ref, token) (state, error)`
- `MergeRequestPort.Merge(ctx, ref, token) error`
- 适配器：GitHub REST / GitLab REST；`Proxy=nil`

## 领域事件

见意图对照表。幂等键与 NFR 表一致。

## 目录落点

- `taskGitOauth/domain/merge_request_ref.go`
- `taskTaskService` 评论 handler 扩展（不强制新 domain 包以免过度拆分）
