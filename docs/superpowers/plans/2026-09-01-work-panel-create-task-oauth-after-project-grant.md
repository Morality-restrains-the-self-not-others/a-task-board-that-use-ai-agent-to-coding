# 实施计划 — 项目 L2 种下创建任务自动运行

- **Date:** 2026-09-01
- **Goal:** 项目详情已授权的同一用户，在该项目工作面板/Fork 创建 `auto_run=true` 任务时不再被 OAuth 门禁拦截；【自动运行】评论写入评论 L2。

## Task 1: taskProjectService GET grant

- [ ] RED: `lookupProjectGitOAuthGrant` + GET handler 单测（有行 / 他用户 / 他项目 / 缺参 / 无 secret）
- [ ] GREEN: 同路径 GET 返回 `{has_grant, remote_user_id}`；POST 行为不变
- [ ] `openapi.yaml` + `db/api_route_ownership.yaml`

验证：`go test` 包内 `TestLookup*` `TestHandleInternal*GitOAuthGrant*`

## Task 2: taskTaskService seed

- [ ] RED: `applyProjectL2SeedToIdentities`（命中 stamp、他项目不 stamp、已有 ticket 不覆盖、lookup 失败不阻断）
- [ ] GREEN: `ensureAutoRunAtComment` 在 ticket 之后调用 seed；发 `COMMENT_GIT_OAUTH_GRANTED` `via=project_l2_seed`
- [ ] 日志 `event=auto_run_project_l2_seed` / `_lookup_failed`

验证：`go test` `TestApplyProjectL2Seed*` `TestEnsureAutoRunAtComment_SeedsFromProjectL2`

## Task 3: taskFE 门禁

- [ ] RED: `collect*OAuthRepoProjectIds`；bound = session ticket OR `token_available`；仅 L1 仍拦截；文案含项目详情
- [ ] GREEN: `useCreateTaskRepoOAuth` 调 validate-git-repos `probe_access:false` + `project_id`
- [ ] CreateTaskModal / Fork header 传入 `tenantId`

验证：vitest `createTaskOauthGate.test.js` + `useCreateTaskRepoOAuth.test.js`

## Task 4: Review + Ship

- [ ] 日志/意图事件对照
- [ ] 子仓 commit+push，meta 指针，v126 → current
- [ ] 登记精准编译重启 taskFE / task-project-service / task-task-service
