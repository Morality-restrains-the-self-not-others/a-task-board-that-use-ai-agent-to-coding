# 实施计划：Git OAuth 资源使用标记

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-git-oauth-resource-grant-marker-design.md`
- **DDD**: `docs/superpowers/plans/2026-08-29-git-oauth-resource-grant-marker-ddd.md`

每个任务：先写失败测试 → 最小实现 → 跑测。

## Task 1: Grant 匹配纯函数（GitOauth domain）

- [ ] `taskGitOauth/domain/grant.go`：`GitsiteFromRepoURL`、`GrantKey`
- [ ] `grant_test.go`：github.com 与带端口 gitlab host
- **Verify**: `cd taskGitOauth && go test ./domain -count=1`

## Task 2: grant_ticket 表 + Issue/Consume

- [ ] `dataMigrate/taskGitOauth/007_grant_ticket.sql` utf8mb4 + 服务前缀
- [ ] Issue/Consume 单测（二次消费失败）
- **Verify**: `go test ./... -count=1 -timeout 120s`（包级）

## Task 3: OAuthBrowserState 增加 GrantKind/GrantID

- [ ] 旧 state 无字段仍可解码
- [ ] start-from-gateway 读 query
- **Verify**: 现有 oauth_state 测试 + 新用例

## Task 4: 项目 L2 DDL + HasGrant/Upsert

- [ ] `dataMigrate/taskProjectService/013_git_oauth_grant.sql`
- [ ] token_status：无 L2 不得 `token_available`
- **Verify**: `git_oauth_grant_test.go` + 现有 validate 测

## Task 5: 回调 MarkGrant + 事件

- [ ] GitHub/GitLab callback 成功后调 Project/Task internal
- [ ] `PROJECT_GIT_OAUTH_GRANTED` / ticket 路径
- **Verify**: callback 单测 fake HTTP

## Task 6: 评论 JSON grant 字段 + 消费 ticket

- [ ] ParseRepoIdentities 读 oauth_* 
- [ ] 发评/ensureAutoRun 消费 ticket
- [ ] `COMMENT_GIT_OAUTH_GRANTED`
- **Verify**: comment_repo_identities 测 + auto_run 测

## Task 7: 换票前查 L2

- [ ] taskProjectService fetchGitAccessToken 前
- [ ] taskCredentialService ResolveLayerOauthTokens / buildCredentials
- [ ] taskCloudService git_push_oauth
- **Verify**: 各服务现有 oauth 测扩展「无 grant 拒绝」

## Task 8: 排队 UserID = 评论作者

- [ ] `startQueuedMembership` 禁止 `t.OwnerID` 覆盖
- [ ] 测：Owner≠created_by 时 UserID=作者
- **Verify**: `queued_schedule_dispatch_oauth_actor_test.go`

## Task 9: FE 徽章与创建门禁

- [ ] start href 带 grant_kind/grant_id 或 pending
- [ ] createTaskOauthGate：connected 不够，需要 session ticket 或 grant query
- [ ] 徽章：需要授权 / 已授权 / 授权异常
- **Verify**: createTaskOauthGate.test.js、gitRepoOAuthStatusUtils.test.js

## Task 10: 意图与 OpenAPI

- [ ] 更新 `create_task_oauth_bind.intent.md`、034、项目详情 oauth 意图
- [ ] gitOauth/project OpenAPI 新参数
- [ ] ADR-0049 → accepted（随 ship）

## Task 11: 架构 ship 时 v118 → current

- [ ] `/10-ship` 改 VERSION_HISTORY；不在 build 改 current 文件

## 回滚

- 关掉 L2 检查 feature？本增量不打 flag（语义正确性）；回滚=还原调用方跳过 HasGrant + 删表（数据可丢）。
