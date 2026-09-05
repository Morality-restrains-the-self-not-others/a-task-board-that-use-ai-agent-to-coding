# 测试意图：层图 push 换票须识别 GitLab 评论 L2

对应功能意图：`043_layer_oauth_gitlab_binding.intent.md`

| ID | 场景 | 前置 | 步骤 | 期望 |
|----|------|------|------|------|
| T1 | 站点级空 URL L2 不被丢 | 评论 JSON `[{repo_url:"",oauth_gitsite:"gitlab-tencent-sh-1.daydaymoney.com"}]` | `FetchRepoIdentities(task, comment)` | 1 行；`OauthGitsite` 为该 host；`FromComment` |
| T2 | HTTPS 仓 + SSH 身份 | 任务仓 `https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git`；身份 scp URL + `oauth_gitsite` + UserID | `ResolveLayerOauthTokens` 带 comment token 与 match key | `OK`；match key 图有 token |
| T3 | 仅站点级 L2 | 身份 `repo_url` 空、有 `oauth_gitsite`、UserID=0；评论作者 42 | 同上 | `OK`；`FetchAccessToken(42, gitlab-tencent-sh-1.daydaymoney.com)` |
| T4 | 真缺绑定 | 无身份 | 同上 | `BINDING_MISSING`；detail 不含「关联项目」；detail_safe 含「Git 授权」不含单独「GitHub」 |
| T5 | zTree 芯片 | `last_push_error` 含 HTTP 409 JSON `BINDING_MISSING` | `layerPushErrorFields` | `pushErrorLabel==='未绑定 Git 授权'`；`pushErrorKind==='binding'` |

可执行测试：

- `taskCredentialService/application/layer_oauth_test.go`
- `taskCredentialService/infrastructure/sqlite_business_comment_author_test.go`
- `taskCredentialService/infrastructure/http_business_test.go`
- `taskFE/app/src/utils/layerZtreePushError.test.js`
