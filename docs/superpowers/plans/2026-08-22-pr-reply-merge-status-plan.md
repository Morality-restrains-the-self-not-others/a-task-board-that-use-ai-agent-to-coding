# 实施计划：PR 回复与一键合并

- **日期:** 2026-08-22
- **设计 / NFR / DDD:** 同主题 specs/plans

## 任务清单

### T1 解析 URL（Red→Green）

- [x] `taskGitOauth/domain/merge_request_ref.go` + `_test.go`
- 覆盖 GitHub pull / GitLab `/-/merge_requests/`
- 非法 scheme、非 allowlist host 失败

### T2 评论 DDL + 幂等插入

- [x] `dataMigrate/taskTaskService/013_comment_git_pr.sql`
- [x] SELECT/INSERT/scan 增加 `parent_comment_id`, `git_pr_html_url`, `git_pr_json`
- [x] 同 task+url 返回已有；首次插入 publish `TASK_GIT_PULL_REQUEST_RECORDED`
- [x] Go 测试

### T3 GitOauth status/merge

- [x] 路由 + OpenAPI
- [x] mock HTTP Git API
- [x] 成功/已合并 noop/无 token；审计 `merge_request_merge`
- [x] publish `GIT_MERGE_REQUEST_MERGED`
- [x] SSRF：host 必须匹配 provider website 或 github.com

### T4 前端推送后建评

- [x] `onLayerGraphLayerPush` / submit-and-push 在 html_url 时 POST 评论
- [x] parent = `resolveActiveExecutionCommentId`
- [x] 测例扩展 `taskDetailLayerActions.test.js`

### T5 会话嵌套与卡片

- [x] `nestDisplayCommentsByParent` 嵌套任意 parent
- [x] `CommentGitPrReply.vue`：链接、徽章、一键合并（clickGuard）
- [x] 加载时批量 status（非 poll）
- [x] ConversationFeed 行数若超 500 则拆组件（当前 389，卡片独立文件）

### T6 意图 INDEX + 架构 v94 + 精准重启登记

- [x] INDEX F-094
- [x] architecture 四件套 + VERSION_HISTORY
- [x] `.runall/precise_restart_services.txt`：taskFE taskTaskService taskGitOauth

## 验证命令

```
cd taskGitOauth && go test ./domain ./src -count=1 -timeout 60s
cd taskTaskService && go test ./src -count=1 -timeout 90s -run 'GitPr|CreateComment'
cd taskFE/app && npx vitest run src/composables/taskDetail/taskDetailLayerActions.test.js src/composables/taskDetail/buildDisplayComments.test.js src/components/task-detail/CommentGitPrReply.test.js
gofmt -d ... && go vet
```
