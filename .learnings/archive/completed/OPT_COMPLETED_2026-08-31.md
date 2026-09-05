# Completed OPT Archive — 2026-08-31

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 40 条。
> 归档执行时间：2026-09-02T03:01:17+08:00

## [OPT-20260829-005] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: AiMonitor/docker-compose.yaml promtail `restart: unless-stopped`；`runall-local-promtail.sh` 优先刮 `$DEPLOY_ROOT/logs`。`docker inspect` RestartPolicy=unless-stopped；挂载 `/tmp/ram-deploy/logs`；`loki_ingester_memory_streams{tenant="fake"}=37`；`/loki/api/v1/label/job/values` 含 `task-cloud-service`/`task-task-service`；`{job="task-task-service"} | trace_id="33a8447afbbc18929e4aa5f4"` 1 stream。python3 runAll/scripts/tests/test_resolve_runall_log_root.py 2 passed。
- **Created**: 2026-08-29
- **Context**: 排查启动 TraceId `d7654c836d4365f957da8398` 时 Loki `http://10.2.150.68:3100` 可达但 `loki_ingester_memory_chunks=0`、无业务 job；历史 OPT 已多次拉起 promtail 后又空。本机 `logs/task-cloud-service.log` 还被整点 truncate。TraceId 门禁只能退回 MySQL。
- **Action**: (1) `AiMonitor/docker-compose.yaml` 把 promtail `restart:` 从 `no` 改为 `unless-stopped`（或 runAll 监控栈启动强制 `runall-local-promtail.sh up`）；(2) 拉起后 `curl` Loki `{job="task-cloud-service"}` 非空；(3) 用新 X-Trace-Id 打一笔 start-vm 断言 `{job=~".+"} |= "<id>"` 命中。
- **Why**: 无 ingest 时无法用页面 TraceId 重建启动失败路径，只能靠会被 truncate 的本地日志和混时区 DATETIME。
- **How to apply**: `bash runAll/scripts/runall-local-promtail.sh up`；`curl -sS http://10.2.150.68:3100/loki/api/v1/label/job/values`；对照已完成 OPT-20260824/0827/0828 promtail 恢复记录。

## [OPT-20260831-010] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Completion-Note**: 已跟踪 `conf/` YAML 密钥抽到 gitignored `conf-local/`；HOST_SECRETS 登记册撤销（ADR-0054）。验收：`python3 db/scripts/ci/test_check_conf_local_secrets.py` exit 0；`python3 db/scripts/ci/check_conf_local_secrets.py` exit 0。`db/registry.yaml` 口令另见 OPT-20260831-011。
- **Created**: 2026-08-31
- **Context**: SSO JWT、Git OAuth `client_secret`、PayPal、Stripe 等曾在已跟踪 YAML。
- **Action**: 抽取到 `conf-local/`，`conf/` 留空骨架，CI 禁止非空机密键。
- **Why**: clone 不得再带上业务密钥。
- **How to apply**: `python3 db/scripts/ci/check_conf_local_secrets.py`

## [OPT-20260830-019] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: GitHub Release tag deploy-20260831 已上传 17 个 ELF（~245MB）。envs/current/releases.yaml 钉 github://…@deploy-20260831；sha 为内容 sha256。deploy-sync 证明：干净目录拉下 runAll+valueStream，sha256 与 pin 一致（/tmp/daydaymoney-gh-sync-proof）。Fetcher 用 Release tag 而非内容哈希查 API。本机 /tmp/ram-deploy 仍可用 file://。taskEvents/taskFE dist 未进该 Release。
- **Created**: 2026-08-30
- **Context**: P4 本机 `/tmp/ram-deploy` 已用 `file://` pins + deploy-sync 跑通；跨机部署仍需 GitHub Release 资产。17 个 ELF 约 245MB。
- **Action**: (1) `bash scripts/publish-deploy-artifacts.sh <tag> /tmp/ram-deploy/artifacts task2money/daydaymoney-deploy`；(2) 把 `releases.yaml` 改成 `github://task2money/daydaymoney-deploy/<asset>@<tag>` 并推配置仓；(3) 在干净目录 `DEPLOY_MODE=1 deploy-sync` 验证能拉下来。
- **Why**: `file://` 不能给第二台无源码主机用。
- **How to apply**: `scripts/publish-deploy-artifacts.sh`；`gh release list --repo task2money/daydaymoney-deploy`。

## [OPT-20260831-008] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Completion-Note**: GitHub tag deploy-20260831 assets: taskEvents-bin.tar.gz 783668123B sha 6d1b1f0f…, taskFE-dist.tar.gz 3152649B sha 232f564a…, runAll clobber sha 909656f45d9097dbb4303b4a22a78ebecb2d4efa4ecbb804c0a4621cbef901b8. `go test ./src -run TestSyncPinnedArtifacts` 0.015s ok（含 RestoresRelativeSymlink、UnpacksTarGzToDest）。pytest `runAll/scripts/tests/test_up_from_config_repo.py` 4 passed。GitHub deploy-sync 空目录 `/tmp/daydaymoney-opt008-gh-proof`（未设 TASK_EVENTS_BIN/TASKFE_PUBLIC）exit 0：54 个可执行 worker；`html` → `releases/20260830145020-2529203`；`test -f html/index.html`。pins `dest`+`unpack` 在 `envs/current/releases.yaml`。TASK_* 仅 sync 后 overlay。live `/tmp/ram-deploy` 仍 `file://`。https://github.com/task2money/daydaymoney-deploy/releases/tag/deploy-20260831
- **Summary**: GitHub tag deploy-20260831 已含 taskEvents-bin.tar.gz 与 taskFE-dist.tar.gz；干净目录 deploy-sync 不解压依赖 TASK_* overlay。
- **Created**: 2026-08-31
- **Context**: 标签 `deploy-20260831` 已含 17 个 Go ELF。`taskEvents/bin` 本机约 1.5G、`taskFE/app/public` 未上传；跨机 `up.sh` 仍需 `TASK_EVENTS_BIN` / `TASKFE_PUBLIC`。
- **Action**: (1) 评估 GitHub 单文件 2GB 上限与压缩分包；(2) 上传并在 `releases.yaml` 增加 pin；(3) 干净目录 `deploy-sync` 拉下后 `test -x taskEvents/bin/...` 与 `test -f taskFE/app/public/index.html`。
- **Why**: 否则独立配置仓仍缺事件 worker 与前端静态资源，无法整栈冷启动。
- **How to apply**: `scripts/publish-deploy-artifacts.sh` 或分资产上传；`gh release view deploy-20260831 --repo task2money/daydaymoney-deploy`。

## [OPT-20260830-003] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: clickGuard busy 改为 Vue ref；模板 :disabled=isBusy() 在 await 后解除。验收：npx vitest run src/utils/clickGuard.test.js exit 0；taskAiProvider node --test tests/clickGuard.unit.test.js exit 0。Search: isBusy/createClickGuard across taskFE+taskAiProvider — 两处 clickGuard.js 均改为 ref。
- **Created**: 2026-08-30
- **Context**: 历史版本入口曾把 `:disabled` 绑到非响应式的 `openGuard.isBusy()`，GET 结束后 `busy=false` 不触发重渲染，按钮卡在 disabled，面板无法收起。本次已在 EntityRevisionPanel 去掉该绑定；其它仅绑 `xxxGuard.isBusy()` 而无伴随 ref 的按钮仍有同类风险。
- **Action**: (1) 给 `createClickGuard` 的 busy 使用 Vue `ref` 或对外暴露 computed；(2) 搜 `Guard.isBusy()` 且模板无并列响应式条件的按钮；(3) 补测：await 结束后 `disabled` 必须解除。
- **Why**: 非响应式 busy 会在异步结束后留下永久 disabled，纯 UI 开关会被锁死。
- **How to apply**: 改 `taskFE/app/src/utils/clickGuard.js` 与 `clickGuard` 单测；`npx vitest run src/utils/clickGuard.test.js` exit 0。

## [OPT-20260830-004] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: GET 任务 403 改用 errMsgForbiddenRead「您没有权限查看此任务」。验收：go test ./src -count=1 -run Forbidden\|GetTask 于 taskTaskService exit 0。Search: errMsgForbidden GET handlers — list/comments/revision/queued 读路径一并改。
- **Created**: 2026-08-30
- **Context**: 诊断 `task_881388002226499584` 关联项目空态时，非成员对 GET `/api/tasks/todos/.../{taskId}/` 也返回 `errMsgForbidden`（「您没有权限修改此任务」）。读接口用写语义文案，排障时会误判为 PATCH 失败。
- **Action**: (1) 在 `taskTaskService/src/api_errors.go` 增加读路径文案（如「您没有权限查看此任务」）；(2) GET handler 改用该常量；(3) 补 Go 单测断言 GET 403 body 不含「修改」。
- **Why**: 错误文案与 HTTP 方法不一致时，前端/Loki 检索会把只读失败当成写失败。
- **How to apply**: `go test ./src -count=1 -run 'Forbidden|GetTask'` 于 `taskTaskService`；exit 0。

## [OPT-20260830-009] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: Home.vue 页脚邮箱改为 VITE_CONTACT_EMAIL。验收：npx vitest run src/views/Home.footer-links.test.js exit 0；源码无 author@example.com。
- **Created**: 2026-08-30
- **Context**: `/faq/` 已从 `conf/frontend/vue/config.yaml` 的 `contactEmail` 注入展示地址；`Home.vue` 页脚「联系我们」仍写死 `author@example.com`。改 conf 后首页与 FAQ 会不一致。
- **Action**: (1) `Home.vue` 用 `import.meta.env.VITE_CONTACT_EMAIL` 渲染页脚邮箱；(2) 补 Vitest 断言页脚文本等于该 env、源码无裸 `author@example.com`。
- **Why**: 同一联系邮箱两处 SSOT 会在改 conf 后只改 FAQ、首页仍旧。
- **How to apply**: 改 `taskFE/app/src/views/Home.vue` 与对应单测；`npx vitest run src/views/Home.test.js` 或现有 Home 测例 exit 0。

## [OPT-20260831-012] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: compose/run.sh/脚本去掉 do-not-use-in-prod 默认；load_gitservice_config 从 yaml/conf-local 导出 oidcClientSecret 与 trafficGateInternalSecret。验收：python3 gitService/scripts/test_load_gitservice_config.py；rg -n do-not-use-in-prod conf gitService 无命中；python3 db/scripts/ci/check_no_hardcoded_secrets.py exit 0。Search: do-not-use-in-prod across conf+gitService — 全部去掉。
- **Created**: 2026-08-31
- **Context**: `conf/` YAML 机密键已抽到 `conf-local/`。`conf/infra/git-service-tencent-sh-1/docker-compose.yml` 仍用 `${VAR:-本地开发密钥}` 插值；`gitService/run.sh` 也有同类 fallback。门禁因 `${` 前缀不报。
- **Action**: (1) compose 与 run.sh 的 `*SECRET` 默认改为空；(2) 从 confload 合并后的 `internalSecret` / OIDC client secret 导出；(3) 单测断言脚本不再含 `do-not-use-in-prod` 字面量。
- **Why**: 已跟踪文件里仍有可复制的本地密钥默认值，和「conf 仅非机密」不一致。
- **How to apply**: `rg -n 'do-not-use-in-prod' conf gitService` 无命中；`python3 db/scripts/ci/check_no_hardcoded_secrets.py` exit 0。

## [OPT-20260831-006] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: loadProjectRepoMeta/getProjectRepoURLs 增加 ctx，创建/更新任务传 r.Context()。验收：go test ./src -run LoadProjectRepoMeta|GetProjectRepoURLs exit 0。
- **Created**: 2026-08-31
- **Context**: 任务详情 GET 已不再调用 `loadProjectRepoMeta`。创建/更新任务仍用 `context.Background()` 打 GET project，出站不带入站 `x-trace-id`。
- **Action**: (1) 给 `loadProjectRepoMeta` / `getProjectRepoURLs` 增加 `ctx`；(2) 从 HTTP handler 传入 `r.Context()`；(3) 单测断言出站带同一 trace。
- **Why**: 写路径若再撞 GitLab 探活，Loki 仍对不上这条任务请求。
- **How to apply**: `taskTaskService/src/project_client.go` `loadProjectRepoMeta`；创建任务 handler。

## [OPT-20260831-009] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: downloadGitHubAsset 每 64MB/30s 打 bytes= 进度。验收：go test ./src -run TestGitHubAssetDownloadLogsProgress exit 0，日志不含 token。
- **Created**: 2026-08-31
- **Context**: OPT-008 从 GitHub 拉 `taskEvents-bin.tar.gz`（~748MB）时，`github release lookup` 与 `github asset stored` 之间约 11 分钟无日志，排障无法区分卡住与慢下。
- **Action**: (1) `downloadGitHubAsset` 按已读字节打结构化进度（如每 64MB 或每 30s）；(2) 单测用 stub Reader 断言日志含 `bytes=` 且不含 token。
- **Why**: 跨机冷启动最大资产无进度时，运维只能干等或误杀进程。
- **How to apply**: `runAll/src/deploy_fetch.go` `downloadGitHubAsset`；`go test ./src -run TestGitHubAssetDownloadLogsProgress` 退出 0。

## [OPT-20260831-003] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: crg-daemon add --alias taskChromePlugin；code-review-graph register+update。search queryRequestList 命中 Function lib/request-list-query.js；search applyRequestFilters 命中 panel 契约测；code-review-graph repos 列出 taskChromePlugin。
- **Created**: 2026-08-31
- **Context**: `/goal` DevTools 请求过滤排序时 `code-review-graph search applyRequestFilters` 返回 0 nodes。根图未 register `taskChromePlugin`，设计只能靠 grep/读文件。
- **Action**: (1) `crg-daemon add /tmp/ram-work/taskChromePlugin --alias taskChromePlugin`（或等价 register）；(2) 对该子仓 `code-review-graph update`；(3) 确认能搜到 `queryRequestList` / `applyRequestFilters`。
- **Why**: 后续插件改动无法用 CRG 做影响面，和流水线硬门禁不一致。
- **How to apply**: `.claude/skills/1-brainstorming-design-docs/references/code-review-graph.md` 子仓 register 节；`code-review-graph repos`。

## [OPT-20260831-002] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: Popup 改用 RequestListQuery.filterRequests/sortRequests，popup.html 增加 #reqSort。验收：node --test test/popup-request-list.test.js test/request-list-query.test.js exit 0。
- **Created**: 2026-08-31
- **Context**: DevTools 单请求列表已抽出 `lib/request-list-query.js` 并支持列头排序。Popup `popup.js` 仍手写 filter + `capturedAt` 时间倒序，无用户可控排序。
- **Action**: (1) Popup 改为调用 `filterRequests`/`sortRequests`；(2) 在窄布局加排序下拉（时间/状态/URL），不必做五列表头；(3) 单测覆盖 popup 过滤与默认时间倒序不回归。
- **Why**: 同类列表两套逻辑会漂移；用户在弹窗预览错误请求时同样需要排序。
- **How to apply**: `taskChromePlugin/popup/popup.js` `renderRequestList`；`popup.html` 增加 `reqSort`；`test/` 新增或扩展 popup 契约测。

## [OPT-20260830-012] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskProject/Cloud/Task/Tenant/AIComment/AIEndPoint/Referral/Credential findMonorepoRoot 委托 confload.FindMonorepoRoot。验收：各仓 go test -run TestFindMonorepoRoot exit 0。Search: func findMonorepoRoot walk conf/base.yaml — 上列服务已委托。
- **Created**: 2026-08-30
- **Context**: `FindMonorepoRoot` 已委托 `FindConfigRoot`，但 `taskProjectService`/`taskCloudService`/`taskTaskService`/`taskTenantService`/`taskAIComment`/`taskAIEndPoint`/`taskReferral`/`taskCredentialService` 等仍本地 walk `conf/base.yaml` 或 `db/registry.yaml`，无源码主机上会忽略 `CONF_ROOT`。
- **Action**: (1) 各服务 `findMonorepoRoot` 改为 `return confload.FindMonorepoRoot()`（或 `FindConfigRoot`）；(2) 删除重复 walk；(3) 测例 unset/设置 `CONF_ROOT`。
- **Why**: 部署机无 `.gitmodules` 时这些进程仍按 cwd 找不到配置。
- **How to apply**: 上列服务 `config.go`/`main.go`；`taskAuth`/`taskBill` 已是委托范例。

## [OPT-20260830-022] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: go test ./src -run ReloadConfigFromDisk_SkipsUnchangedMtime -count=1 exit 0; applyDeployLayout logs once per svc+detail
- **Created**: 2026-08-30
- **Context**: `/api/status` 轮询会 `reloadRunAllConfigBestEffort` → `applyDeployLayout`，`DEPLOY_MODE=1` 下每条服务打 `deploy layout:`，把 `runall-console.log` 打爆。
- **Action**: (1) 配置热加载改为 mtime/hash 变化才 `LoadConfig`；(2) `applyDeployLayout` 的 per-service `log.Printf` 改为 debug 或只在首次/变更时打。
- **Why**: 状态页打开时磁盘与 Loki 被编排器自刷淹没，排障看不到 start/stop。
- **How to apply**: 打开 9999 30s 后 `rg -c 'deploy layout:' /tmp/ram-deploy/logs/runall-console.log` 增量应接近 0；`go test ./src -run Reload` 覆盖 mtime 短路。

## [OPT-20260830-020] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: go test ./src -run KillPreviousRunAllProcess_SkipsOccupied9999DuringGoTest -count=1 exit 0; testing.Testing() skips :9999 shutdown-self
- **Created**: 2026-08-30
- **Context**: P4 切到 `/tmp/ram-deploy/bin/runAll` 后，runAll 子仓 `go test ./src/...`（pre-commit）把生产编排器 `shutdown-self` 掉了；task-auth 仍在、`:9999` 空、value-stream 需重拉。日志里出现测试进程的 `deploy layout:` 行。
- **Action**: (1) 所有启动 UI 的测例改用临时端口/`t.TempDir` 配置，禁止听 `9999`；(2) `killPreviousRunAllProcess` 在 `go test` 下不得对已占用的生产 9999 发 `/api/shutdown-self`；(3) 加回归：先起假 9999 再 `go test` 不得把它杀掉。
- **Why**: 部署机与开发机同跑 `go test` 时会拆掉编排器空窗，违反 ADR-0035。
- **How to apply**: `runAll/src/main.go` `killPreviousRunAllProcess`；`go test ./src -count=1` 前后 `ss -ltnp | rg ':9999\b'` PID 不变。

## [OPT-20260830-024] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: go test ./src -run AdoptListeningServices_SkipsSourceTreeELFInDeployMode -count=1 exit 0; DEPLOY_MODE leftover ram-work ELF not adopted
- **Created**: 2026-08-30
- **Context**: P4 切到 `/tmp/ram-deploy` 后 runAll adopt 把仍在跑的 `/tmp/ram-work/<svc>/bin/<svc>` 标成 healthy；`start-all` 跳过它们。本次手工 kill leftover 后才从 `$DEPLOY_ROOT/bin` 拉起。
- **Action**: (1) adopt 时比较 `/proc/<pid>/exe` 是否在 `$DEPLOY_ROOT/bin` 或配方 cwd；(2) `DEPLOY_MODE=1` 下 exe 落在源码树则视为 not-owned，纳入 start 计划；(3) 回归：先起 ram-work ELF 再 `DEPLOY_MODE=1 start-all`，最终 exe 必须是 deploy bin。
- **Why**: 否则编排器显示全绿但进程仍读源码树 conf/cwd，切流不完整。
- **How to apply**: `runAll/src` adopt 路径；`DEPLOY_MODE=1` 单测。

## [OPT-20260831-005] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: go test ./src -run TestHandleProjectGitRepoDiskSizesEnrichesInternal -count=1 exit 0; npx vitest run useProjectDetailGitRepos.disk-size.test.js exit 0
- **Created**: 2026-08-31
- **Context**: B-086 从 `handleGetProject` 去掉 `enrichProjectGitRepoDiskSizes`。项目详情页磁盘数字不再随 GET 出现。
- **Action**: (1) 新增或复用独立 GET/POST 拉 `disk_size_bytes`；(2) 项目详情阶段 B 加载；(3) GitLab hang 时 GET 项目仍快。
- **Why**: 探活已独立，磁盘统计同样打 GitLab，不应悄悄回到数据 GET。
- **How to apply**: `taskProjectService/src/repo_disk_size.go`；`taskFE/app/src/composables/useProjectDetailGitRepos.js`。

## [OPT-20260831-001] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: go test taskTaskService -run TestDetachTaskProjectsByProjectIDRemovesRows exit 0; taskEvents projectdeleted TestReplaySameProjectIDSkipsSecondDispatch exit 0; python3 db/scripts/ci/check_task_events_runall_health_ports.py exit 0
- **Created**: 2026-08-31
- **Context**: `handleDeleteProject` 只删 `project_entries`/`project_repos`/`project_workspaces`。任务表 `task_projects` 仍指向已删 ID。2026-08-28 三个 somanyad 项目从目录消失后，任务详情仍渲染可点链接，详情 GET 404。昨夜 dump 恢复无法找回（8 月 29 起 dump 已无这些行）。
- **Action**: (1) 删除项目时发领域事件 `PROJECT_DELETED`（project_id）；(2) taskTaskService 消费者按 project_id 删除或标记 `task_projects`；(3) 单测：删除后 GET 任务 projects 不再含该 id；(4) 可选巡检 SQL：`task_projects.project_id NOT IN project_entries`。
- **Why**: 否则任务详情会把 Snowflake ID 当项目名可点，用户感觉「详情页丢失」。
- **How to apply**: `taskProjectService/src/project_handlers.go` `handleDeleteProject`；`taskTaskService` 消费者；`go test` 覆盖删除后关联行数为 0。`npx vitest run src/utils/taskProjectsWithDetails.test.js` 已覆盖目录缺失展示。

## [OPT-20260829-002] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: go test ./src -run PublicUnsubscribe\|Resubscribe exit 0; npx vitest run UnsubscribeConfirm.test.js 5 passed
- **Created**: 2026-08-29
- **Context**: 本轮交付了邀请邮件退订列表与跳过 SMTP，确认页只有「已退订」说明，没有把邮箱移出列表的入口。误点退订或后续想再收邀请邮件的用户只能走运维改库。
- **Action**: (1) 在 `/auth/unsubscribe/` 增加带 HMAC token 的「重新接收邀请邮件」按钮（写操作须 clickGuard + Idempotency-Key）；(2) 增加 `DELETE` 或 `POST /api/public/email-resubscribe/` 幂等删除 `auth_email_unsubscription` 行；(3) 发布 `EMAIL_RESUBSCRIBED` 并补单测。
- **Why**: 退订是单向集合写入，没有对等恢复路径会导致支持成本上升，也与常见 List-Unsubscribe 产品预期不一致。
- **How to apply**: `taskAuth/src/auth_email_unsubscription.go`、`taskFE/app/src/views/UnsubscribeConfirm.vue`、`dataMigrate` 无需改表（删行即可）；权限上保持公开 token 与退订同一 HMAC。机器验收：`go test ./src -run 'PublicUnsubscribe|Resubscribe'`（taskAuth）与 `npx vitest run src/views/UnsubscribeConfirm.test.js`（taskFE）均 exit 0。

## [OPT-20260830-025] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: handle_corrupt_datadir_test.sh 11 passed; empty shell start exits 1 unless MYSQL_ALLOW_EMPTY_REINIT=1; check_p4 fails ram-work mysql data symlink
- **Created**: 2026-08-30
- **Context**: 本机 21:33 重启后 ramsync `ensure_data_dir` 把磁盘上无 `ibdata1` 的空壳目录当成「快照」拷回 RAM 并打 restored。P4 `materialize-p4-root.sh` 把 `$DEPLOY_ROOT/dockerInfra/mysql/data` 链到该空壳；`run.sh start` 的 `detect_corrupt_data.sh` 判定损坏后 `mv` 的是符号链接，再 `mkdir` 空目录冷启动，随后 9999 `init-databases` 只种了 bootstrap。用户能登录但业务行丢失。21:27 dump 与 21:16 物理副本仍在。
- **Action**: (1) `detect_corrupt_data.sh` 命中时默认拒绝启动，除非显式 `MYSQL_ALLOW_EMPTY_REINIT=1`；(2) `ensure_data_dir` 必须检查 `ibdata1`+`mysql/`，空壳不得报 restored，应改从 `$MYSQL_BACKUP_DIR/mysql-dump-latest.sql.gz` 提示恢复；(3) P4 物化不要 symlink 已判定损坏的 datadir。
- **Why**: 切流+重启叠加会把「能登录的空种子库」当成成功，真实用户数据只留在 dump/快照里。
- **How to apply**: `dockerInfra/mysql/run.sh`、`detect_corrupt_data.sh`、`/home/ljy/bin/ramsync-daemon.sh` `ensure_data_dir`；回归：空壳 datadir 启动必须非 0 退出。

## [OPT-20260830-021] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: pytest test_export_deploy_payload.py asserts Dockerfile/src/buildDocker.sh; test_check_p4_deploy_root requires those files
- **Created**: 2026-08-30
- **Context**: go-relay 用 `trae-agent/onlineServiceJS/run.sh` 作 monorepo 标记；P4 只拷了这一个文件。镜像构建与 agent 运行时还需要同目录 Dockerfile/依赖。
- **Action**: (1) 审计 go-relay 与 `DOCKER_PUSH=1 ./buildDocker.sh` 实际读取的路径；(2) 把必要文件写入 `materialize-p4-root.sh`；(3) 给 checker 加存在性断言。
- **Why**: 缺文件时 go-relay 启动失败，UI 还可能展示过期 chdir 日志。
- **How to apply**: `test -x /tmp/ram-deploy/trae-agent/onlineServiceJS/run.sh`；`go test ./src -run FindMonorepoRoot`；按审计清单 `test -f` 其余文件。

## [OPT-20260830-015] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: bash dockerInfra/scripts/resolve_infra_conf_test.sh 5 passed; redis/kafka/portainer/mysql run.sh source resolve_infra_conf
- **Created**: 2026-08-30
- **Context**: `/tmp/ram-deploy` 切over 时已改 `dockerInfra/mysql/run.sh` 读 `CONF_ROOT`/`DEPLOY_ROOT`（日志 `conf=/tmp/ram-deploy/conf/infra/mysql/config.yaml`）。`redis`/`kafka`/`portainer` 的 `run.sh` 仍主要靠 compose 本地文件，未统一同一解析函数。
- **Action**: (1) 抽出与 mysql 相同的 `resolve_*_conf` 到 `dockerInfra` 公共片段；(2) redis/kafka/portainer 若读 YAML 则走 `CONF_ROOT`；(3) 补 shell 测：只有 conf 树时仍能解析。
- **Why**: Docker 基础设施配方是 ADR-0052 范围内的交付面，Go 根解析改了但配方脚本未跟。
- **How to apply**: `dockerInfra/*/run.sh` 与对应 `*_test.sh`；`bash -n` + 现有 run.sh 测例。

## [OPT-20260829-008] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: go test ./src -run TestRejectPendingZeroAmountInvoiceApplicationsClearsPending exit 0; TestApproveInvoiceZeroAmountRejected still green
- **Created**: 2026-08-29
- **Context**: 本轮禁止了零额申请/审批，但上线前若库中已有 `billing_invoice_application.status=pending` 且订单 `total_yuan_cents<=0`，租户会一直看到「开票处理中」，管理员点已开具会被 400。
- **Action**: (1) 查出 pending 申请 JOIN 订单金额 ≤0 的行；(2) 用既有 reject 路径批量拒绝并写 note「零额不可开票」；(3) 补一条只读盘点 SQL/单测夹具，禁止手改 status。
- **Why**: 否则零额赠送单会卡在 pending，客服无法从 UI 闭环。
- **How to apply**: `taskBill` `rejectInvoiceApplication` + 一次性运维脚本或 admin 过滤；机器验收：`go test ./src -run TestApproveInvoiceZeroAmountRejected` 仍绿，并新增盘点查询测例行数为 0 或拒绝后 pending=0。

## [OPT-20260831-011] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: python3 db/scripts/ci/check_conf_local_secrets.py exit 0; python3 db/scripts/ci/test_check_conf_local_secrets.py ok; go test db/load -run TestResolveMySQLDSNUsesConfLocalPasswordOverlay; rg password: db/registry.yaml is password: ""
- **Created**: 2026-08-31
- **Context**: conf YAML 密钥已抽到 `conf-local/`。`db/registry.yaml` 口令仍在 conf 之外的已跟踪文件；进入 Git 历史的 PayPal/Stripe/SSO 等值按第 56 条仍须废弃轮换。
- **Action**: (1) 口令改为 overlay 或环境注入，tracked YAML 留空骨架；(2) 轮换已进入 Git 的凭据，不要 filter-repo。
- **Why**: clone 仍会拿到 registry 口令；历史中的 conf 密钥仍有效。
- **How to apply**: `python3 db/scripts/ci/check_conf_local_secrets.py` 保持 exit 0；registry 迁出后 `rg -n 'password:' db/registry.yaml` 无非空业务口令。

## [OPT-20260830-011] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: wc -l hardwarePanel/*.js all <=500 (max createHardwarePanelCtx.js 469); vitest 3 files 11 tests passed
- **Created**: 2026-08-30
- **Context**: 本会话为任务 ID 解析改动该文件，`wc -l` 仍约 2608 行，远超源文件 500 行门禁；整文件拆分超出本次「缺少任务ID」切片。
- **Action**: (1) 按启动 VM / 地域实例 / 模版摘要拆 composable；(2) 每文件 `wc -l` ≤500；(3) 复跑现有 hardwarePanel 单测。
- **Why**: 行数门禁已触发，继续堆功能会无法审查。
- **How to apply**: `taskFE/app/src/composables/hardwarePanel/useServerConfigHardwarePanel.js`；`python3 task2app/scripts/ci/check_frontend_component_line_limit.py` 或 `wc -l` 验收。

## [OPT-20260829-004] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: go test ./src -run TestCommentCreateAndListCreatedAtRFC3339Z; go test ./src -run TestCSCEventBindingCreatedAtUnixAlignedAndJSONZ — RFC3339 Z and unix diff < 2
- **Created**: 2026-08-29
- **Context**: 排查 task_881353145278558208 评论时间 vs 阿里云 CreationTime 时发现 `task_comments.created_at` / CSC `created_at` 写入的是本机墙钟 naive DATETIME，而 `cloud_server_events` / `cloud_comment_container_bindings` 用 UTC。MySQL `SYSTEM` tz 为 UTC，导致 `NOW()` 与应用本地时间混用，时间线要对齐必须手工换算。
- **Action**: (1) 盘点 `taskTaskService`/`taskCloudService` 写入 DATETIME 的路径，统一为 UTC（`UTC_TIMESTAMP()` 或 `time.Now().UTC()`）；(2) API 输出一律 RFC3339 带 `Z`；(3) 补迁移说明与单测，禁止 naive 本地墙钟再入库。
- **Why**: 混用 naive 本地与 UTC 会把真实延迟（本例约 75 分钟）误判成时区显示问题，也让 Loki/DB 对照成本升高。
- **How to apply**: `taskTaskService` 评论 INSERT；`taskCloudService` CSC/event/binding `created_at`；机器验收：`go test` 断言序列化为 RFC3339 `Z`，且同一墙钟事件在两表的 unix 秒差 &lt; 2。

## [OPT-20260829-015] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: runAll/playwright/tests/runall-dep-indicator.playwright.test.js：fixture :19999 打开 / 等 .service；每个 .dep[data-dep-name] 与目标行 :scope>.dot class 一致；拦截 /api/status 把 depends_on.status 打空后 healthy 仍为 green。./node_modules/.bin/playwright test tests/runall-dep-indicator.playwright.test.js --config=playwright.config.js 2 passed (2.9s).
- **Created**: 2026-08-29
- **Context**: 9999 状态页在精准编译重启/热加载 YAML 后曾把 `depends_on.status` 打回 pending，依赖点全灰；已在 StatusStore 叠 live 状态并让 UI 按同 payload 服务行上色。仍缺页面级回归。
- **Action**: (1) 在 `runAll/playwright/tests/` 增加用例：打开 `/`，等 `.service` 渲染；(2) 对每个 `.dep[data-dep-name]` 读取目标行 `.dot` class，断言与依赖点 class 一致（healthy→green）；(3) 覆盖「depends_on.status 为空但目标行 healthy」仍为 green。
- **Why**: 单测覆盖 store/helper，热替换后的 embed UI 与收养时序仍可能让点色和行状态短暂分叉。
- **How to apply**: `runAll/playwright/tests/runall-dep-indicator.playwright.test.js`；机器验收：`npm run test:e2e -- runall-dep-indicator` 或现有 Playwright 入口对该文件 exit 0。

## [OPT-20260830-007] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/Faq.playwright.test.js：GET /faq/ 200；无 faq-doc-nav；无「我们便不会主动提起诉讼」「账号与登录」「{{contactEmail}}」；faq-doc-panel-usage|billing 可见、无 account；正文含 conf contactEmail author@example.com；faq-back-home-link 到 pathname=/。会话失效收口跳过匿名 FAQ（requestErrorDisplay.isAnonymousPublicPage）。playwright.verify.config.js 1 passed (1.1s)。
- **Created**: 2026-08-30
- **Context**: FAQ 已去掉文档侧栏、「知识产权处理方法」与「账号与登录」，剩余 usage/billing 纵向铺开；md 已移出 `src/public` 以免静态 `/faq/` 目录导致 nginx 403。Vitest 已覆盖，公网硬刷新需 APISIX + nginx try_files 接线。
- **Action**: (1) 在 `taskFE/tests/` 增加 Playwright：打开 `/faq/` 断言 HTTP 200、无 `[data-testid=faq-doc-nav]`、无「我们便不会主动提起诉讼」、无「账号与登录」、无 `{{contactEmail}}`；(2) 断言 `faq-doc-panel-usage|billing` 存在且无 `faq-doc-panel-account`，正文含 `conf/frontend/vue/config.yaml` 的 `contactEmail`；(3) 点击 `faq-back-home-link` 到达首页。
- **Why**: 单元测不到 hashed 入口 JS、SPA fallback 与目录 403 回归。
- **How to apply**: `taskFE/tests/` 新增 Faq.playwright 测例；机器验收对应文件 `npx playwright test` exit 0。

## [OPT-20260829-010] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/TaskDetail.comment-execution-git-oauth-summary.playwright.test.js 增「已绑定 + 层快照 Permission denied」：mock user-app-connection connected + container-layer-graph.git_remote.last_push_error；断言 comment-execution-git-oauth data-kind=bound_no_write 且 bind href 含 github-start-from-gateway。playwright.verify.config.js 该文件 2 passed（含原 T17 unbound）。
- **Created**: 2026-08-29
- **Context**: 本轮用 Vitest 覆盖了 T17b（OAuth 已绑定 + `Permission to … denied` →「无写权限」+ 换账号授权；zTree「push 无权限」）。公网任务详情页层快照经 SSE 注入，尚无 Playwright 把 `last_push_error` 打进 mock 层图并断言 summary 芯片。
- **Action**: (1) 在 `taskFE/tests/TaskDetail.comment-execution-git-oauth-summary.playwright.test.js` 增加已绑定 + 层快照 permission denied 用例；(2) mock `user-app-connection` 为 connected 且 layer graph 含 `last_push_error`；(3) 断言 `comment-execution-git-oauth` `data-kind=bound_no_write` 且 bind href 含 start-from-gateway。
- **Why**: 单元测不到 SSE/层图槽把 `last_push_error` 传到摘要行的真实接线。
- **How to apply**: `taskFE/tests/TaskDetail.comment-execution-git-oauth-summary.playwright.test.js`；机器验收：`npm run test:e2e -- TaskDetail.comment-execution-git-oauth-summary`。

## [OPT-20260829-013] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/ProjectDetail.oauth-token-status-button.playwright.test.js：mock GET 他人 GitHub git_repos + POST validate-git-repos token_status=token_error is_accessible=false；/me 含 tenant 避免 onboarding。断言 git-repo-oauth-status-0「授权异常」、重试按钮、不得「已授权」。playwright.verify.config.js -g 他人 GitHub 1 passed (1.1s)。
- **Created**: 2026-08-29
- **Context**: 项目详情 Git 行「已授权」已改为：有 GitHub token 但 `permissions.push=false`（如 `test-ruandao/helloworld`）时后端返回 `token_error`。现有 Playwright 只 mock `git_repos_status`，未走真实 validate-git-repos probe 契约。
- **Action**: (1) 在 `taskFE/tests/ProjectDetail.oauth-token-status-button.playwright.test.js` 增加用例：项目 `git_repos` 为他人 GitHub URL，`validate-git-repos` mock `token_status=token_error` + `is_accessible=false`；(2) 断言 `[data-testid=git-repo-oauth-status-0]` 为「授权异常」且出现「重试」按钮，不得为「已授权」。
- **Why**: 单元测覆盖 Go probe 与 composable，但详情页从 GET project 到徽章的完整接线仍缺 E2E。
- **How to apply**: `taskFE/tests/ProjectDetail.oauth-token-status-button.playwright.test.js`；机器验收：`npm run test:e2e -- ProjectDetail.oauth-token-status-button` exit 0。

## [OPT-20260829-024] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/TaskDetail.comment-git-oauth-probe-once.playwright.test.js T18c: unreachable 无 bind href；skipped_intranet 保持 bound。15 文件套件 15 passed。
- **Created**: 2026-08-29
- **Context**: 执行细节芯片「Git OAuth · 已绑定」在非内网 GitLab 宕机时已改为「网络不可达」（Go/Vitest）。公网页仍依赖精准编译重启 task-git-oauth 与 taskFE 后才能对 `http://115.29.110.74` 看到 `data-kind=unreachable`。
- **Action**: (1) 9999 精准编译重启 `task-git-oauth` 与 `taskFE`；(2) 硬刷新任务详情评论执行细节；(3) 未标内网且 GitLab 不可达时断言 `[data-testid=comment-execution-git-oauth][data-kind=unreachable]`；(4) 租户标记内网后保持 `data-kind=bound`。
- **Why**: 单元测不到 APISIX、user-app-connection probe 与 SPA 构建产物接线。
- **How to apply**: `taskFE/tests/TaskDetail.comment-git-oauth-probe-once.playwright.test.js` 增 T18c mock `network_status=unreachable`；公网验收对应页面 exit 0。

## [OPT-20260829-016] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/TaskDetail.layer-ztree-push-error-copy.playwright.test.js：复制写入失败原文与 traceId。15 passed。
- **Created**: 2026-08-29
- **Context**: 层节点「push 失败」芯片旁已增加「复制」按钮，Vitest 覆盖了 `formatPushErrorClipboardText` 与组件点击写入 title+traceId。公网任务详情层图经 SSE 注入，尚无 Playwright 把 `last_push_error` 打进 mock 层图并断言复制。
- **Action**: (1) 在 `taskFE/tests/` 增加或扩展 TaskDetail 层图用例，mock 层快照含 `git_remote.last_push_error` 与 `last_push_error_trace_id`；(2) 断言 `[data-testid=layer-ztree-push-error-copy]` 可见；(3) 点击后 `navigator.clipboard.writeText` 收到失败原文且含 `traceId:`。
- **Why**: 单元测不到评论执行细节展开后 zTree 节点从 SSE 快照到剪贴板的完整接线。
- **How to apply**: `taskFE/tests/` 下 TaskDetail 层图 Playwright；机器验收：`npm run test:e2e -- layer-ztree-push-error-copy` 或对应文件 exit 0。

## [OPT-20260829-026] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/PeopleInvite.expiration.playwright.test.js：invite-expiration-days 含 90/180/365 默认 90。15 passed。
- **Created**: 2026-08-29
- **Context**: 邀请链接有效期下拉已扩到 365 天（Vitest/Go 已绿）。公网页需登录后切到「复制邀请链接」才能看见 `data-testid=invite-expiration-days`。
- **Action**: (1) 9999 精准编译重启 `taskFE` 与 `task-tenant-service`；(2) 登录后打开 `/tenant/{tid}/people/invite/`；(3) 选「复制邀请链接」；(4) 断言 select 含 90/180/365 且默认 90。
- **Why**: 单元测不到公网 SPA hash、登录态与原生 select 渲染。
- **How to apply**: `taskFE/tests/` 下 people-invite Playwright；机器验收对应文件 exit 0。

## [OPT-20260829-014] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: CreateTask.grant-ticket + ProjectDetail.oauth-l2-badges Playwright：无 ticket 提交禁用、回流可提交、L2 徽章三态。15 passed。
- **Created**: 2026-08-29
- **Context**: L2 资源标记已落地：创建/Fork 自动运行看本会话 `grant_ticket`，项目/评论徽章为需要授权 / 已授权 / 授权异常。Vitest 已覆盖 composable 与 session helper，Playwright 仍按旧 L1 `connected` 或仅 mock `git_repos_status`。
- **Action**: (1) 创建任务 Playwright：auto_run 时 L1 connected 但无 session ticket 仍禁用提交；(2) 回流 `grant_ticket=` 后可提交；(3) 项目详情：无 L2 显示「需要授权」，L2+probe 可写为「已授权」，L2 但 probe 失败为「授权异常」。
- **Why**: 公网路径依赖 sessionStorage ticket 与后端 L2 字段，单元测不到弹窗到 POST 的完整接线。
- **How to apply**: `taskFE/tests/` 下 CreateTask / ProjectDetail / TaskDetail OAuth Playwright；机器验收：对应 `npm run test:e2e` 文件 exit 0。

## [OPT-20260830-001] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/EntityRevision.playwright.test.js：任务/项目详情 open 空态后 close。15 passed。
- **Created**: 2026-08-30
- **Context**: 任务/项目标题与正文已落不可变快照，Vitest 覆盖打开才 GET、点选再拉详情与 data-traceId。公网页仍依赖 9999 迁移 + 精准编译重启后才能看到入口。
- **Action**: (1) 9999「初始化全部数据库」应用 `019_task_revision` / `014_project_revision` / `051_entity_revisions_resource_members`；(2) 精准编译重启 `taskTaskService` `taskProjectService` `taskEvents` `taskFE`；(3) 打开任务详情点 `entity-revision-open`，断言列表请求且空态或行渲染，再点 `entity-revision-close` 面板消失；(4) 项目详情同样断言。
- **Why**: 单元测不到 APISIX 前缀、utf8mb4 分区表与生产 SPA hash 接线。
- **How to apply**: `taskFE/tests/` 新增 Playwright；机器验收对应文件 exit 0。

## [OPT-20260830-002] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/TenantFeedbackNav.playwright.test.js：tenant-feedback-toggle 展开含 a[target=_blank]。15 passed。
- **Created**: 2026-08-30
- **Context**: 超管配置链接组、租户 GET 按累计消耗过滤已有 Go/Vitest。公网页需 9999 迁移 `074`/`052` 并精准编译重启 task-bill 与 taskFE 后才能看到侧栏。
- **Action**: (1) 9999「初始化全部数据库」应用 `dataMigrate/taskBill/074_feedback_link_groups.sql` 与 `dataMigrate/taskAuth/052_nav_feedback_page.sql`；(2) 精准编译重启 `task-bill` `taskFE`；(3) 超管打开 `/system-admin/feedback-links/` 保存一组 https 空阈值链接；(4) 租户控制台断言 `tenant-feedback-toggle` 与真实 `a[target=_blank]`。
- **Why**: 单元测不到 APISIX 租户前缀、PDP region 注入与生产 SPA hash。
- **How to apply**: `taskFE/tests/` 新增 Playwright；机器验收对应文件 exit 0。

## [OPT-20260829-023] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/GitlabConnection.reachability.playwright.test.js：reachability-down 与 skipped_intranet。15 passed。
- **Created**: 2026-08-29
- **Context**: 已用 Go/Vitest 覆盖探测与内网跳过。公网页仍依赖 taskFE SPA 构建、`008_intranet` 迁移与 task-git-oauth 重启后，才能对已关闭的自建 GitLab 看到红字或勾选内网后的预期文案。
- **Action**: (1) 9999 执行 git-oauth 迁移并精准编译重启 `task-git-oauth` 与 `taskFE`；(2) 硬刷新 `/tenant/{tid}/settings/gitlab-connection/`；(3) 未勾选内网时断言 `[data-testid=gitlab-reachability-down]`；(4) 勾选并保存后断言 `[data-status=skipped_intranet]`。
- **Why**: 单元测不到 APISIX 前缀、脱敏 GET 与公网 SPA hash 的接线。
- **How to apply**: `taskFE/tests/` 下 gitlab-connection Playwright；机器验收：对应 `npm run test:e2e` 文件 exit 0。

## [OPT-20260829-017] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: taskFE/tests/TaskDetail.comment-grant-ticket.playwright.test.js：无 ticket 提交禁用、回流 POST 含 grant_ticket。15 passed。
- **Created**: 2026-08-29
- **Context**: 云端开发曾出现「提交成功但推送失败：尚未完成 Git OAuth 使用授权」。根因是 composer / 层图只查 L1 `user-app-connection`，评论创建未消耗 `grant_ticket`，推送才查评论 L2。现已在 FE 提交前、taskTaskService POST、层图 commit 前拦截，但 Vitest 测不到公网 task-detail 回流 `grant_ticket` 与已有评论补授权。
- **Action**: (1) 在 `taskFE/tests/` 增加 TaskDetail 用例：关联 GitHub HTTPS 仓、mock L1 `connected: true`、无 session `grant_ticket` 时 `@镜像` 后提交并运行按钮 disabled；(2) 模拟 OAuth 回流 `grant_ticket=` 后按钮可点且 POST body 含 `grant_ticket`；(3) 已发出运行评论无 `oauth_gitsite` 时层图「提交并推送」不发 git-commit，提示使用授权。
- **Why**: 公网路径依赖 sessionStorage ticket、callback query 与评论 JSON L2，单测测不到弹窗到 POST/层图的完整接线。
- **How to apply**: `taskFE/tests/` 下 TaskDetail comment OAuth Playwright；机器验收：对应 `npm run test:e2e` 文件 exit 0。

## [OPT-20260831-004] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: aimonitor-promtail 刮 /tmp/ram-deploy/logs；restart unless-stopped；loki_ingester_memory_streams{tenant=fake}=37；{job=task-task-service}|trace_id=33a8447afbbc18929e4aa5f4 命中。python3 runAll/scripts/tests/test_resolve_runall_log_root.py 2 passed。
- **Created**: 2026-08-31
- **Context**: 任务详情 15s 排障时 Loki 7 天 `{job=~".+"}` 为 0；进程日志在 `/tmp/ram-deploy/logs/`。Tempo 只有 task-auth span。
- **Action**: (1) 部署 filelog/promtail 刮 `/tmp/ram-deploy/logs/*.log`；(2) 确认 `loki_ingester_memory_streams` 非 0；(3) 用已知 `x-trace-id` 能查出 task-task-service 行。
- **Why**: 有 traceId 却只能 grep 磁盘，无法用 Grafana 跨服务对齐。
- **How to apply**: `conf/runAll.yaml` loki_url；AiMonitor otel/promtail 配置。

## [OPT-20260831-014] completed

- **Status**: completed
- **Completed**: 2026-08-31
- **Summary**: layout.sh clone-as-root writes gitignored conf-local/runAll.yaml logging.file_root to $DEPLOY_ROOT/logs without sed-ing tracked YAML; pytest runAll/scripts/tests/test_layout_from_config_repo.py 8 passed.
- **Created**: 2026-08-31
- **Context**: 新节点 clone-as-root 后 `conf/runAll.yaml` 仍可能写死 `file_root: /tmp/ram-work/logs`；`layout.sh` 仅在 `DEPLOY_ROOT != CONFIG_REPO` 时 sed。新机目录不是 `/tmp/ram-work` 时日志/9999 会指错路径。
- **Action**: (1) clone-as-root 也把 `file_root` / `RUNALL_LOG_ROOT` 写成 `$DEPLOY_ROOT/logs`；(2) 单测覆盖 clone 根即部署根；(3) README 一句说明。
- **Why**: 只拷 conf-local 仍可能因硬编码源机路径导致新节点日志与 9999 找不到。
- **How to apply**: `runAll/scripts/layout-from-config-repo.sh` + `test` 同目录；`rg -n 'file_root: /tmp/ram-work' .daydaymoney-deploy-seed`

---

