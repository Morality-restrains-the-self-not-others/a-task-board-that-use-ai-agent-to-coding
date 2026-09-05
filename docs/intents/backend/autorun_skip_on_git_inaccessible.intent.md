# Intent: Git/子 Git 不可用时自动运行不启动服务器

## 需求（2026-07-18）

当关联项目无法获取 Git 仓库或子 Git 仓库列表（如「未检测到可用授权」）时，即使任务开启 `auto_run`，也不得异步触发云服务器启动（`start-vm` / `start-vm-auto`）。`auto_run` 标志仍可保存为 true（软跳过）。

## 落点

- `taskTaskService`：`probeGitAccessForAutoRun` + create/update 调度前判定
- 探测：`auto_clone_nested_repos=true` 时 `GET /api/internal/nested-git-repos/`；`false` 时 `POST validate-git-repos` 且 **`probe_access=false`（仅 OAuth token，不打远程 GitLab REST）**
- Git 探测 HTTP 客户端超时 **40s**（`projectGitProbeHTTP`），不得用通用 `projectHTTP` 15s 误杀已授权父仓（2026-08-21 `task_878541740905099264`）

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Git 不可达时跳过自动启服 | — | — | — | — | 应用策略门禁；不新增领域事件。既有 TASK_CREATED 与启服解耦 |

## 设计

`docs/superpowers/specs/2026-07-18-autorun-skip-on-git-inaccessible-design.md`

## 可见性与重试（2026-08-17）

软跳过不得只留在 create/update 响应里：

- 落库 `tasks.auto_run_start_skip_reason`（`dataMigrate/taskTaskService/012_*.sql`）
- GET 任务详情回传 `auto_run_start_skipped` + `auto_run_start_skip_reason`
- 真正 `scheduleTaskAutoRun`（无 StartSkipReason）前清除 skip reason
- 前端 Fork/创建弹窗 + 任务详情横幅「强制重新启动」（`force_auto_run`）

见失败经验：`.ai/09_failure_experience/02_runtime_errors/98_autorun_skip_no_ui_reason.md`

## 软跳过仍发【自动运行】评论（2026-08-17）

Git/子 Git 探测失败（含「探测失败」运输错误）时：

- **仍**调度 `scheduleTaskAutoRun`，但带 `StartSkipReason`
- `triggerTaskAutoRun`：先 `ensureAutoRunAtComment`（正文追加「未启动服务器：{reason}」），再**跳过** `start-vm`
- 不得因软跳过而省略自动执行评论

见：`taskTaskService/src/auto_run.go`（`StartSkipReason`）、`auto_run_at_comment.go`（`composeAutoRunAtCommentContent`）

## GitLab refresh 400（2026-08-22）

子仓探测走 `access-for-user` → `RefreshGitLabToken`。GitLab `/oauth/token` refresh **必须**带与授权时一致的 `redirect_uri` 以及 `client_secret`（官方 OAuth2 契约）；缺失时 Doorkeeper 返回 `400 invalid_grant`，skip_reason 曾泄漏英文 `gitlab refresh http 400`。

- `taskGitOauth`：refresh 表单含 `client_id` / `client_secret` / `grant_type` / `refresh_token` / `redirect_uri`；**空 redirect_uri 禁止发 HTTP**
- `taskGitOauth`：换票走 `issueAccessTokenFromCredential` + `SELECT ... FOR UPDATE`，probe / access-for-user / merge 共用，避免并行 refresh 把 GitLab 旋转后的旧 token 打成 400
- `taskProjectService`：sanitize + nested skip 文案改为「授权已失效，请重新绑定」
- 任务详情 skip 横幅：humanize 存量英文原因；未绑定时提供「去绑定 Git OAuth」真实链接；`user-app-connection` 已 connected 时隐藏该链接（skip_reason 仍落库，须「强制重新启动」清跳过）
