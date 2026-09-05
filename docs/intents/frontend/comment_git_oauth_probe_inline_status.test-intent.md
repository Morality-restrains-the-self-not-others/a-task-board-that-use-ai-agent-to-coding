# 测试意图：任务详情打开时 Git OAuth AccessToken 探测失败以内联徽标展示

## 对应功能意图

`docs/intents/frontend/comment_git_oauth_probe_inline_status.intent.md`

## 用例

| ID | 场景 | 前置 | 步骤 | 期望 |
|----|------|------|------|------|
| T1 | 探测超时不弹窗 | user-app-connection probe AbortError，文案含「超时」，带 traceId | 任务详情评论区触发一次 probe | 不调用 `showRequestError`；readiness 含 `checkFailedRepoUrls` 与 `probeTraceId` |
| T2 | 徽标检查超时 | readiness.checkFailedRepoUrls 含 GitHub URL | 渲染 CommentExecutionGitOauthBadge | `data-kind=check_failed`；文案「Git OAuth · 检查超时」；`data-traceId` 等于探测 trace；无绑定链接 |
| T3 | 非超时探测失败 | probeError 不含「超时」 | 摘要函数 | kind=check_failed，文案「Git OAuth · 检查失败」 |
| T4 | 未绑定优先于检查失败 | unbound 与 checkFailed 同时存在 | 摘要函数 | kind=unbound |
| T5 | 按仓捕获超时 | fetchOutcome reject 超时 Error | probeCommentGitOauthAccessTokensOnce | 不抛；返回 checkFailedRepoUrls + probeTraceId |

## 可执行测试

- `taskFE/app/src/utils/commentExecutionGitOauth.test.js`
- `taskFE/app/src/utils/probeCommentGitOauthAccessTokensOnce.test.js`
- `taskFE/app/src/components/task-detail/CommentExecutionGitOauthBadge.test.js`
- `taskFE/app/src/composables/taskDetail/useCommentGitOauthAccessTokenProbe.test.js`
