# Test Intent: Git/子 Git 不可用时自动运行不启服

| ID | 场景 | 期望 |
|----|------|------|
| T1 | create `auto_run=true`，nested-git-repos 返回未授权 error | HTTP 201；`auto_run=true`；`auto_run_start_skipped=true`；cloud start 次数=0；仍调用 ensureAutoRunAtComment 且 `StartSkipReason` 含授权文案 |
| T1b | T1 后 GET 同一任务（冷打开） | `auto_run_start_skipped=true`；`auto_run_start_skip_reason` 含探测失败原因（已落库） |
| T1c | create `auto_run=true`，nested-git-repos HTTP 非 200（探测失败） | skip_reason 含「探测失败」；ensure 评论 1 次；cloud start=0 |
| T2 | create `auto_run=true`，nested error 空 | cloud start ≥1；skip reason 清空 |
| T3 | update `force_auto_run` + nested 父仓不可见 | 跳过 start；响应含 skip_reason；落库可冷打开回放；仍发评论（StartSkipReason） |
| T4 | 关联项目无 git_repos | 不因 Git 探测阻断（保持原启服行为） |
| T5 | 真正 schedule 成功后 | `auto_run_start_skip_reason` 为空 |
| T6 | `triggerTaskAutoRun` 带 `StartSkipReason` | ensure 评论成功；不调用 cloud start-vm |
| T7 | `composeAutoRunAtCommentContent` / ensure 带 skip | 评论正文含「未启动服务器：」+ reason |
| T8 | `auto_clone_nested_repos=false` 父仓探测 | `validate-git-repos` body `probe_access=false`；token 可用则 skip_reason 空 |
| T9 | 父仓探测客户端超时 | skip_reason 含「超时」而非笼统「探测失败」 |
| T10 | Git 探测 HTTP 超时 | `projectGitProbeHTTP.Timeout` ≥ 40s（覆盖观测到的 24s GitLab REST） |
| T11 | GitLab refresh 缺 redirect_uri/client_secret | `RefreshGitLabToken` 表单含官方字段 |
| T12 | GitLab refresh HTTP 400 invalid_grant | 错误含 `invalid_grant`，不含 refresh token 明文 |
| T13 | nested skip `gitlab refresh http 400` | 文案含「重新绑定」，不含原始英文 status |
| T14 | 任务详情 skip 横幅存量英文原因 | `auto-run-start-skipped-reason` 人性化；有 repoUrl 且尚未绑定时出现 `auto-run-skip-oauth-bind` |
| T15 | skip 横幅 OAuth 已绑定回流 | `user-app-connection` connected=true 时隐藏 `auto-run-skip-oauth-bind`；横幅与强制重启仍在 |
| T16 | GitLab refresh 空 redirect_uri | `RefreshGitLabToken` 在发 HTTP 前失败，错误含 `redirect_uri` |
| T17 | probe 与 access-for-user 并发 miss cache | GitLab refresh 只执行 1 次（行锁 `FOR UPDATE`） |
| T18 | `GetCredentialByIDForUpdate` | 第二事务在第一事务 commit 前阻塞 |

实现：`taskTaskService/src/auto_run_test.go`、`auto_run_at_comment_test.go`、`auto_run_parent_probe_test.go`、`taskGitOauth/infrastructure/oauth_clients_test.go`、`taskGitOauth/src/credential_lock_test.go`、`taskProjectService/src/gitoauth_client_test.go`、`taskFE/app/src/components/task-detail/TaskDetailAutoRunSkipBanner.test.js`。
