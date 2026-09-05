# 测试意图：创建/配置项目是否自动克隆子仓库

- **对应功能意图**: `project_auto_clone_nested_repos.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| T1 | Create 不传字段 | DB/响应 `auto_clone_nested_repos=true` |
| T2 | Create 传 false | 持久化为 false |
| T3 | Update true→false | GET 为 false |
| T4 | Merge nested，项目 false | 不追加子仓 URL |
| T5 | Merge nested，项目 true | 追加子仓 |
| T6 | CreateProject UI 有 URL | checkbox 可见且默认 checked |
| T7 | 提交取消勾选 | POST body 含 `auto_clone_nested_repos: false` |
| T8 | 详情关开关空态 | 提示已关闭自动克隆（若有发现列表则列表仍显示） |
| T9 | 任务详情 auto_clone=false | composer 身份行 `task-nested-repos-auto-clone-toggle` 为关；渲染 `task-nested-repos-auto-clone-off-hint`；不渲染 `task-nested-repos-clone-status` |
| T10 | SQLite FetchTaskRepos 读 project_entries | `auto_clone_nested_repos=0` → snapshot false；表名必须是 `project_entries` |
| T11 | 镜像 collectRepoCloneJobs flag=false | 跳过带 `parent_repo_url` 的子仓，仅保留父仓 |
| T12 | identities 为空 / userID=0 且 auto_clone=true | MergeNested 仍发现并合并子仓（匿名 GitHub Contents） |
| T13 | 任务级子仓状态：bootstrapCloneDone 但无该仓进度/已移入日志 | 标 idle，禁止把未克隆子仓显示为已完成 |
| T14 | 私有父仓 + 任务 identity 属于他人 | nested fetch 与克隆凭证使用 `task_comments.created_by_id`，不得用 owner_id |
| T15 | 详情 auto_clone=false，父仓 token_available，子仓 token_error | `auto-run-git-gate.blocked=false`，不展示「授权异常无法启动」 |
| T16 | 详情 auto_clone=false，父仓已授权，nestedError 非空 | 不发出 AUTO_RUN_NESTED_REPOS_UNAVAILABLE |
| T17 | 详情 auto_clone=true，子仓 token_error | 仍发出 AUTO_RUN_GIT_AUTH_ERROR |
| T18 | 详情 auto_clone=false，父仓 token_error | 仍发出 AUTO_RUN_GIT_AUTH_ERROR |
| T19 | 详情 auto_clone true→false（setProps） | 子仓 token_error 时门禁从 blocked 变为放行 |
| T20 | Playwright：auto_clone=false + 子仓 token_error | 无 `project-auto-run-git-gate-hint`，自动运行文案不是「无法启动」 |
