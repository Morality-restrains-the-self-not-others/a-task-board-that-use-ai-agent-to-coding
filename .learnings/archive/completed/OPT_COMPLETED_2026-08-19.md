# Completed OPT Archive — 2026-08-19

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 39 条。
> 归档执行时间：2026-08-25T22:13:52+08:00

## [OPT-20260816-025] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 空闲回收进程内 ticker 已删除，taskEvents workspace_machine_idle timer 接管
- **Created**: 2026-08-16
- **Context**: `startIdleRecycleTicker` 仅当 `IDLE_RECYCLE_TICK_SEC>0` 才启动，conf 未配置该 env，默认关闭。真时钟已是 `task-events-workspace-machine-idle-1-recycle-idle-nodes`（18044）调 `POST /api/internal/cloud/compute/recycle-idle-machines/`。
- **Action**: (1) 从 `taskCloudService/src/main.go` 去掉 `startIdleRecycleTicker()` (2) 删除 `startIdleRecycleTicker` / `idleRecycleTickSeconds`，保留 `handleInternalRecycleIdleMachines` 与 `recycleIdleMachines` (3) 从 `LEGACY_INTERNAL_TICKERS` 去掉 `taskCloudService/src/workspace_machine_recycle.go` (4) 跑门禁与 `workspace_machine_recycle_test.go`
- **Why**: 若有人再设 env，会与 18044 双扫同一批空闲机。
- **How to apply**: 计划 §025；HTTP 已在 `workspace_machine_recycle.go`；timer `taskEvents/internal/handlers/workspacemachineidle/runner.go`
- **2026-08-17 夜间跳过**: `workspace_machine_recycle.go` 有跨文件 WIP（`isMachineNodeStarted`→`machineRuntimeCountsAsStarted` 重命名，依赖 `machine_runtime_status.go`），stash 单文件会破坏编译、整文件提交会卷入无关 WIP。待该批 WIP（idle-reuse 拆除）提交后再做；移除 `os`/`strconv`/`fmt` import 与 `workspace_machine_recycle_test.go` 中 `idleRecycleTickSeconds` 测例。

## [OPT-20260818-047] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: auto_clone_nested_repos=false 时 auto_run 启服探测只校验父仓
- **Created**: 2026-08-18
- **Context**: 项目详情 Git 门禁已在 `auto_clone_nested_repos=false` 时忽略子仓 OAuth。`taskTaskService.probeGitAccessForAutoRun` 仍对每个父仓调用 `nested-git-repos`；该探测验证的是「能否读 .gitmodules」而非子仓 OAuth，父仓已授权时通常成功，但 `.gitmodules` 读取失败仍会软跳过启服。
- **Action**: (1) `probeGitAccessForAutoRun` 经 `loadProjectRepoMeta` 读取开关 (2) false 时改为父仓 validate-git-repo / token 探测，不再要求 nested-git-repos 成功 (3) true 时保持现网 nested 探测 (4) 补 `auto_run_test.go`：false + nested error 仍 start-vm；false + 父仓不可达仍 skip
- **Why**: 与项目详情门禁语义对齐，避免「页面能启用自动运行、创建任务却因读子仓列表失败而不启机」。
- **How to apply**: `taskTaskService/src/auto_run.go` `probeGitAccessForAutoRun`；测例 `auto_run_test.go`；意图 `docs/intents/backend/autorun_skip_on_git_inaccessible.intent.md`

## [OPT-20260818-039] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: oidc_handlers.go 拆分 ≤500 行 + client not found 结构化 WARN
- **Created**: 2026-08-18
- **Context**: GitLab SH-1 SSO `unauthorized_client` 仅有 `http_request` status=400，响应 JSON 无 `client_id` 字段；`handleOidcAuthorize` 拒绝路径未打结构化 WARN。`taskAuth/src/oidc_handlers.go` 已超 500 行，本修复未改该文件以免触发强制拆分阻塞 SSO。
- **Action**: (1) 将 authorize/token/jwks 拆到 ≤500 行文件 (2) client 查找失败时 `slog.WarnContext` 记录 `client_id`+`trace_id`（禁止 secret）(3) 补单测断言日志或错误体仍不含 secret
- **Why**: 下次再缺 client 时 Loki 按 `client_id` 即可定位，不必先查库。
- **How to apply**: `taskAuth/src/oidc_handlers.go` `handleOidcAuthorize`；对照 `oidc authorize store` 已有 slog 字段

## [OPT-20260818-001] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: Fork 带 fork_from 时强制第一进度列（服务端不变量）
- **Created**: 2026-08-18
- **Context**: 本次已在前端 `forkTask` 把 `progress_column_id` 写成工作区进度第一列。其它 API 客户端仍可在 `POST todos` 时复制源任务列。
- **Action**: (1) `handleCreateTask` 在 `fork_from` 非空时忽略客户端 `progress_column_id` (2) 向 taskProjectService 解析工作区 progress-system 第一列并写入 (3) 补 Go 单测
- **Why**: 进度列归属服务端不变量，避免非 SPA 客户端再次把 Fork 任务放到源列。
- **How to apply**: `taskTaskService/src/create_task.go`；进度列查询走既有 project 内部 API

## [OPT-20260817-040] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: runJobAsync close 路径集成级契约测（spawn 可注入）
- **Created**: 2026-08-17
- **Context**: 本次用抽出的 `finalizeJobCloseSideEffects` 单测拦住回归；`runJobAsync` 仍依赖真实 spawn，集成路径未覆盖。
- **Action**: (1) 用可注入 spawn/EventEmitter 的 harness 测 `runJobAsync` close 路径 (2) 断言无 mount 时仍调用 `triggerAutoRunDeliveryForJobAndMirror`
- **Why**: 防止未来有人把交付再次塞回 `if (mountedAgentId)` 块内且单测仍绿。
- **How to apply**: `trae-agent/onlineServiceJS/src/jobsRuntimeRunJob.mjs`

## [OPT-20260818-003] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: OpenTestMySQL cleanup DROP 失败打 WARN（不再静默）
- **Created**: 2026-08-18
- **Context**: 9999「清空全部数据库」后 `dockerInfra/mysql/data` 仍有 52 个 `task_*_test_*` / `test_infra_*` 目录。`db/load/testdb.go` 的 cleanup 用 `_ = runMySQLExec(...)` 丢弃 DROP 错误，测试中断或连接失败时库目录会一直留在共享 MySQL。
- **Action**: (1) cleanup 在 DROP 失败时打 warn 日志（含 db 名与 err）(2) 单测断言失败路径被记录而非静默 (3) 可选：`test_resource_cleanup.sh` 扫描 `SHOW DATABASES` 中 `*_test_*` 残留并提示
- **Why**: 残留测试库会让「清空全部数据库」看起来没改 datadir，并在 tmpfs 上堆积 InnoDB 文件。
- **How to apply**: `db/load/testdb.go` `OpenTestMySQL` cleanup；对照 `db/_infra/mysql_reset.py` 已覆盖的一键 DROP 全部用户库

## [OPT-20260818-006] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: GitHub OAuth 换票失败 / dial demote Loki 告警规则
- **Created**: 2026-08-18
- **Context**: create-project 页出现 `exchange_failed`（trace `6eff4b43bb2221aa64ebd1688f5140f4`）：taskGitOauth 换票打到「TCP 通但 HTTP 挂起」的 github.com 边缘 IP。代码侧已加 demote + 4 轮重试；运维侧仍缺「连续换票超时」主动告警。
- **Action**: (1) 在 AiMonitor 增加 LogQL/指标：统计 `GitHub 回调 authorization_code 换票失败` 与 `dial endpoint demoted` 的 5m 速率；(2) 连续失败阈值触发告警；(3) 文档写入失败经验 90 的运维段。
- **Why**: 边缘 IP 可达性会随路由抖动变化；仅靠用户 Toast 发现过慢，需要主动探测与告警。
- **How to apply**: `AiMonitor/prometheus/rules/` 新增规则；可选 `taskGitOauth` 健康检查暴露最近 demote 计数；对照 `.ai/09_failure_experience/02_runtime_errors/90_github_oauth_exchange_failed_dns_edge_unreachable.md`。

## [OPT-20260817-027] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 终态释放覆盖仅 launch_request_id 的评论 CSC（mark-comment-terminal-released 端点）
- **Created**: 2026-08-17
- **Context**: 进度切到已完成/已取消时，已修复 Starting+instance 的漏杀；但 RunInstances 回包前仅有 `launch_request_id`、尚无 `instance_id` 的评论 CSC 仍 success no-op，晚到的 VM 可能成为孤儿。
- **Action**: (1) 扩展 list-by-task/`ConfigRow` 透出 `launch_request_id`；(2) 终态 handler 对「无 instance 但有 launch_request」行标记 `terminal_released` + `last_runtime_status=Released`；(3) 若云厂商支持按 request 取消则补充取消调用；(4) 单测覆盖。
- **Why**: 否则用户在开机竞态窗口内把任务标完成时，随后出现的 VM 不会被本轮释放链路覆盖。
- **How to apply**: `taskEvents/internal/handlers/taskstatuschanged/`；`taskCloudService/src/list_by_task.go`；对照 `server_orphan_reconcile` / `cloud_csc_reconcile` 是否已部分覆盖。

## [OPT-20260817-037] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: FetchGitHubProfile 对齐 Refresh 的 dial 多轮重试；taskGitOauth 4b876e7 已推送
- **Created**: 2026-08-17
- **Context**: 本次为 `RefreshGitHubToken` 补了无 proxy 时多轮直连重试；`FetchGitHubProfile` 仍只在「直连失败后换 proxy client」时重试，同 client 挂起边缘 IP 不会轮转再试（虽走 `api.github.com`，偶发仍可能超时）。
- **Action**: (1) 将 `FetchGitHubProfile` 的 client 循环改为与 Exchange/Refresh 相同的 attempts 结构 (2) 失败调用 `ReportGithubDialHTTPFailure` (3) 补单测：首轮 timeout、次轮成功
- **Why**: 资料拉取失败会阻断 OAuth 回调落库/展示，与 refresh 同类脆弱点。
- **How to apply**: `taskGitOauth/infrastructure/oauth_clients.go` 的 `FetchGitHubProfile`；参照 `TestRefreshGitHubTokenRetriesAfterHeaderTimeout`

## [OPT-20260818-037] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: gitService Playwright /users/sign_up 302 探测；09b128d06 已推送，本机 127.0.0.1:8012 实测通过
- **Created**: 2026-08-18
- **Context**: 元规则 50 / ADR-0016 的 CI 只检查已提交 conf、compose 与 loader 默认值。GitLab `signup_enabled` 是 DB 级设置，Admin UI 或漏跑 `apply_auth_policy.sh` 仍可能把注册重新打开，静态门禁看不见。
- **Action**: (1) 在 `gitService/playwright/` 增加测例：`GET /users/sign_up` 期望 302 到 `/users/sign_in`，登录页无 password 字段且可见 taskAuth SSO (2) GitLab 未启动时 skip，禁止假失败 (3) 与现有 oidc-sso Playwright 共用基地址
- **Why**: 防止「仓库 conf 全 false、现网 DB 已打开注册」的漂移无人发现。
- **How to apply**: `gitService/playwright/tests/`；`gitService/scripts/apply_auth_policy.sh`

## [OPT-20260818-043] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: gitlab-local/synology provider website 独立化；conf 0048e62 已推送，新增隔离回归测
- **Created**: 2026-08-18
- **Context**: 修 tencent-sh-1 区域 OAuth 时发现 `http-localhost-8012.yaml` 与 `http-synology-gitlab.yaml` 的 `target.website` 仍是 `${scheme}://${subdomains.gitlab}`，与 `daydaymoney-gitlab` 同一 host。catalog/换票会把本应独立的实例绑到默认 CE。
- **Action**: (1) 为 gitlab-local 使用 `localhost:8012`（或独立 subdomain）(2) synology 指向真实 NAS GitLab host (3) 补 resolver 测：两 key 不得与 `gitlab:daydaymoney-gitlab` 抢同一 website
- **Why**: 多实例 token 隔离已按 host 生效；错误 website 会让隔离形同虚设。
- **How to apply**: `conf/auth/task-credential/git-oauth-providers/http-localhost-8012.yaml`、`http-synology-gitlab.yaml`；必要时补 git-oauth/providers 对应 YAML

## [OPT-20260818-008] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 网关 connection refused 清推测性 server_url + demote binding；taskCloudService aab40fc + taskContainerGateway 8bc045a 已推送
- **Created**: 2026-08-18
- **Context**: 存量 CSC 可能已带假 server_url（本修复前写入）。网关 dial refused 时 binding 仍显示 running。
- **Action**: (1) 在 container-gateway / cloud resolve 上游 connection refused 路径调用 clear reachability 或仅清 server_url；(2) demote 对应 comment binding starting；(3) 单测覆盖 refused→starting。
- **Why**: 仅修写入路径无法自愈已污染行；用户仍看到假就绪。
- **How to apply**: `taskContainerGateway` upstream_forward；`clearContainerReachabilityOnConfig`；`ccbTryPromoteStartingToRunning` 对称 demote

## [OPT-20260816-022] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: ticker 迁移总表闭环：allowlist 为空且 check_no_service_internal_poll_loop.py 报 0 legacy；025 空闲回收 7cae2e6 / 026 推荐结算 944729e / 027 orphan 50a83f8 / 028 leaked+cloud_csc_reconcile 9e16146 / 029 微信分账 f27b7a8 / 030 排队调度 0a639e0 / 031 推荐码过期 54ed8ce / 032 relay 00aa672 均已推送
- **Created**: 2026-08-16
- **Context**: 元规则 46 / ADR-0011 禁止业务服务进程内轮询。门禁已对 8 个存量文件 allowlist。迁出须按计划分项落地，禁止一次改 8 处时钟。
- **Action**: 按 `docs/superpowers/plans/2026-08-16-legacy-ticker-migration-plan.md` 顺序执行分项，全部完成后本条才能关：(1) 025 空闲回收删重复 ticker (2) 026 推荐结算 timer (3) 027 orphan API（禁止直连 task-task 库）(4) 028 leaked API + 与 027 共用 `cloud_csc_reconcile` timer (5) 029 微信分账 timer (6) 030 排队调度 timer (7) 031 推荐码过期 timer (8) 032 relay 改为状态变化 push。全部完成后 `LEGACY_INTERNAL_TICKERS` 为空且 `check_no_service_internal_poll_loop.py` 报 0 legacy。
- **Why**: allowlist 只防新增；双时钟会双扫。总表用于盯顺序与收口，真正改代码走 025–032。
- **How to apply**: 计划文档同上；allowlist `db/scripts/ci/check_no_service_internal_poll_loop.py`；模板 `taskEvents/internal/handlers/taskpostexpiryscan/runner.go`

## [OPT-20260817-042] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 网关兜底 remember-pr-html-url：trae-agent d6dc157 + taskContainerGateway 38a4676，双端回归测全绿，已推送。
- **Created**: 2026-08-17
- **Context**: 本次已在 oauth-access-push 成功路径 `rememberLayerPrHtmlUrl`；若 PR 仅由 SaaS follow-up 创建、容器侧未落盘，整页刷新后层图可能仍无 PR 锚点。
- **Action**: (1) 在 gateway/cloud complete 或前端拿到 `html_url` 后调用容器 API 写入 `git_pr_html_url` (2) 补单测：SaaS-only PR 刷新后仍可见 `layer-ztree-pr-btn`
- **Why**: 与 oauth 路径行为一致，刷新不丢 PR 链接。
- **How to apply**: `taskContainerGateway` / `taskCloudService` complete；或 onlineServiceJS 新增 remember 端点

## [OPT-20260819-006] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: INDEX 已增加 B-049e 指向 billing_resource_order_number 意图与 orders_number_test.go
- **Created**: 2026-08-19
- **Context**: 已新增 `docs/intents/backend/billing_resource_order_number.intent.md`，`INDEX.md` 被其他会话 WIP 占用，未写入目录行。
- **Action**: (1) 在 INDEX 后端计费节增加一行指向该意图与 `taskBill/src/orders_number_test.go` (2) 不夹带 INDEX 上无关 diff
- **Why**: 意图金字塔要求目录可检索，否则后续 `/intent-test` 容易漏。
- **How to apply**: `docs/intents/INDEX.md`；意图文件已在 `docs/intents/backend/billing_resource_order_number*.md`

## [OPT-20260818-002] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 夜间 OPT 窗口闭环（已推送）
- **Created**: 2026-08-18
- **Context**: `fetchProgressStatusOptions` 与 `mapProgressColumnsToTaskStatuses` 各写了一份按 order 排序。
- **Action**: 抽到 `workPanelKanbanUtils.js` 共用，两边单测仍覆盖第一列语义。
- **Why**: 两处排序漂移会导致 Fork 第一列与看板第一列不一致。
- **How to apply**: `taskFE/app/src/utils/workPanelKanbanUtils.js`、`taskDetailFetchOptions.js`

## [OPT-20260817-003] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 夜间 OPT 窗口闭环（已推送）
- **Created**: 2026-08-17
- **Context**: 去掉 ztree 点外清空选中后，`TaskDetailTaskLayerAssociationPanel` 仍用手写 `document` 捕获监听关闭模型 `<details>`。仓库已有 `useClickOutside`（WorkspaceSwitcher 等已迁）。
- **Action**: (1) 用 `useClickOutside` 绑 `modelDetailsRef`，仅在 `open` 时关闭 (2) 删 `documentClickBound` / `bindDocumentClick` / `unbindDocumentClick` (3) 保留 click-outside 测例 T4 全绿
- **Why**: 两套点外关闭并存，后续改捕获/冒泡容易漏测。
- **How to apply**: `taskFE/app/src/composables/useClickOutside.js`；`TaskDetailTaskLayerAssociationPanel.vue`

## [OPT-20260818-011] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 夜间 OPT 窗口闭环（已推送）
- **Created**: 2026-08-18
- **Context**: 评论区「将使用镜像 …」已改为 `name:version`。`CommentImageMentionEditor` 的 `@` 下拉仍只渲染 `img.name`，同名不同版本时无法区分。
- **Action**: (1) `mentionImageOptions` 透传 `version`/`tag` (2) 选择菜单项展示 `name:version`（无版本则仅名称）(3) 评论正文 `@mention` 仍只用名称，不改 mentions 契约 (4) 补 `CommentImageMentionEditor` 单测
- **Why**: 用户在点选阶段就应看到将运行的具体版本，避免同名镜像选错。
- **How to apply**: `taskFE/app/src/components/task-detail/CommentImageMentionEditor.vue`、`TaskDetailCommentComposer.vue` 的 `mentionImageOptions`

## [OPT-20260817-028] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 夜间 OPT 窗口闭环（已推送）
- **Created**: 2026-08-17
- **Context**: 辅助信息已展示 `#N`，但复制仍仅针对技术 ID；看板卡片已支持单击复制 `#N`、双击复制完整 ID。
- **Action**: (1) 将 `task-aux-info-display-no` 换为 `TaskCardIdBadge`（或同等交互）(2) 补单测断言单击复制 `#N` (3) 保持无序号时不渲染徽章/`—`
- **Why**: 与看板交互一致，减少「看得见编号却不能一键复制短号」的摩擦。
- **How to apply**: `TaskDetailTaskAuxInfoPanel.vue`；`TaskCardIdBadge.vue`；对照 `TaskDetail.card-ux.test.js`

## [OPT-20260816-056] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 夜间 OPT 窗口闭环（已推送）
- **Created**: 2026-08-16
- **Context**: `TaskDetailCommentsSection.vue` 在接入 `predecessorsFor` / `onFocusPredecessor` 后约 493 行，距 500 行门禁很近。
- **Action**: (1) 把 `cssAttrEscape` 与 `onFocusPredecessor` 迁到 `taskDetailCommentsSectionHelpers.js` (2) 补 helper 单测：给定假 document 能选中 `data-comment-id` (3) 确认 vue 回落到明显低于 500
- **Why**: 再加一小段执行细节接线就会触发强制削文件，打断无关改动。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailCommentsSection.vue`；`taskFE/app/src/composables/taskDetail/taskDetailCommentsSectionHelpers.js`

## [OPT-20260818-004] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: pre-commit 静态阻断顶层 vi.mock 无 VITEST 护栏：taskFE afd443e（部署钩子）+ meta e5553cc3（pre-commit.node SSOT）；全仓现存测试文件均已护栏，无破坏
- **Created**: 2026-08-18
- **Context**: 提交 WIP 批时 `cancelWaitingPreviousBinding.test.js` 顶层 `vi.mock` 在 pre-commit 的 `node --test` 下直接崩溃（Vitest mocker 未初始化）。同类文件已用 `if (!process.env.VITEST) skip` 护栏。
- **Action**: (1) 新增/修改含 `vi.mock` 的 `*.test.js` 必须包在 VITEST 护栏内 (2) 可选：pre-commit 静态扫描顶层 `vi.mock` 且无 `process.env.VITEST` 则阻断
- **Why**: 否则 taskFE 任何含 mock 的新测例都会让 `node --test` 门禁失败，阻断整仓提交。
- **How to apply**: 对照 `taskFE/app/src/components/LayerGraphZtree.events.test.js`；钩子入口见 taskFE `.githooks/pre-commit`

## [OPT-20260819-003] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: taskBill paypal_pay.go paypalCreateOrder 入参改分 + centsToYuanStr；补测 55分→0.55；7c1a3e9 已推送
- **Created**: 2026-08-19
- **Context**: 修微信 Native 按分下单时发现 `paypalCreateOrder` 仍用 `fmt.Sprintf("%d.00", amountYuan)`，不足 1 元会变成 `0.00` 或被调用方先取整。资源订单 PayPal 分支目前只回 `centsToYuanStr`，但充值路径若仍走该函数会与订单分金额漂移。
- **Action**: (1) 入参改为分或直接传 `centsToYuanStr` (2) 补测 55 分 → `0.55` (3) 搜索所有 `paypalCreateOrder` 调用方一并改
- **Why**: 与微信同一类单位错误，任务帖 0.55 元会在 PayPal 侧少收或错收。
- **How to apply**: `taskBill/src/paypal_pay.go` `paypalCreateOrder`；对照 `wechatNativeAmountFen`

## [OPT-20260819-004] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: taskBill wechatCreditFromPending 回调核对 pending.amount_fen vs 订单 total_yuan_cents，拒绝多收；6483133 已推送
- **Created**: 2026-08-19
- **Context**: `ORD-20260818-003` 已有两条 pending（`amount_fen=100`）而订单是 55 分。修复后新码为 55 分；若用户仍扫修复前二维码，回调按 pending 100 分入账会多收 0.45 元。
- **Action**: (1) `wechatCreditFromPending` 在 `OrderID!=0` 时 `loadOrder`，若 `pending.AmountFen != order.TotalYuanCents` 则拒绝入账并打 warn（含 out_trade_no 指纹与两金额）(2) 补测 (3) 评估是否对已支付差额走微信退款
- **Why**: Native `code_url` 最长约 2 小时仍可支付，仅改预下单挡不住旧码。
- **How to apply**: `taskBill/src/wechat_pay.go` `wechatCreditFromPending`；订单 `877596007691485184`

## [OPT-20260818-030] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: taskBill 用量刷新不写 active 行（显式 not_purchased + 零配额视为未购买）；100811c 已推送
- **Created**: 2026-08-18
- **Context**: 修复 GitLab 设置页空下拉时发现：`GET gitlab-resources/?region=` 会先 `refreshTenantDiskUsage`，对从未购买的租户插入 `billing_tenant_gitlab_resource` 空行，列默认 `provisioning_status=active`，随后 `ensureTenantGitlabGroup` 被误触发。
- **Action**: (1) 无配额行时 `refreshTenantDiskUsage` 不得 INSERT 业务行，或插入时显式 `not_purchased` (2) `getTenantGitlabResourceByRegion` 将 disk_gb=0 且 traffic=0 视为未购买 (3) 补测：空租户 `?region=tencent-sh-1` 返回 `not_purchased` 且不写 active 行
- **Why**: 误标 active 会让未购租户看到「已开通」配额区，并对其区域 GitLab 打 Admin API。
- **How to apply**: `taskBill/src/gitlab_resources.go` 的 `reportGitlabDiskUsage` / `refreshTenantDiskUsage`；`taskBill/src/gitlab_region.go` 的 `regionResourceView`

## [OPT-20260818-020] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: taskBill 赠送 GitLab 磁盘后按区域 ensureTenantGitlabGroupForRegion（失败 warn 不回滚）；e1d5ffc 已推送
- **Created**: 2026-08-18
- **Context**: 赠送路径已按 `region` 写入 `billing_tenant_gitlab_resource`，但与购买不同，不触发 `ensureTenantGitlabGroupForRegion`；租户可能看到配额却无对应 GitLab 组。
- **Action**: (1) 在 `adminGrantResources` 提交成功后对本次赠送的每个 slug 调用 `ensureTenantGitlabGroupForRegion`（失败 warn，不回滚配额）(2) 单测 mock 区域 API (3) 无 PAT 时保持可观测 warn
- **Why**: 否则管理员赠送到 `tencent-sh-1` 只加账本、实例侧组/限额可能仍空。
- **How to apply**: `taskBill/src/admin_grant.go`；对照 `gitlab_region.go` `ensureTenantGitlabGroupForRegion`；OPT-20260818-016 PAT

## [OPT-20260819-007] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: taskReferral referral apply internal_error 勿泄露 SQL 原文，结构化日志带 trace_id；cf4eaef 已推送
- **Created**: 2026-08-19
- **Context**: 修复 `reject_reason` Error 1364 时发现 `handleReferralCodeApply` 在 `internal_error` 分支把 `err.Error()`（含 SQL/表字段名）写入 JSON `detail`，页面红字直接展示给用户。
- **Action**: (1) `internal_error` 仅返回通用文案（如「申请失败，请稍后重试」）(2) 完整错误只打结构化日志并带 `trace_id` (3) 补测：DB 失败时响应 body 不含 `HY000` / 字段名
- **Why**: 泄露 schema 细节既不友好也不利于安全；用户只需 `data-traceId` 即可排障。
- **How to apply**: `taskReferral/src/referral_code_handlers.go` `handleReferralCodeApply`；对照元规则 24

## [OPT-20260818-013] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: docs 在实施中 NFR 幂等性审视表头对齐门禁（docs f64af29 已推送）：admin-grant-membership-tier / admin-grant-gitlab-region 已提交；navbar-git-service-region-nav / runall-pending-migrate-badge / billing-transactions-resource-ledger 已在本机对齐表头+规范列但属他会话未提交 WIP（未卷入）。check_nfr_idempotency_table.py --strict 对 5 份均通过。其余存量按「触达时补」。
- **Created**: 2026-08-18
- **Context**: `/5-nfr` 已增加幂等性硬门禁（元规则 48）。门禁脚本默认 warn、存量 200+ 份 `*-nfr-clarification.md` 大多缺表；仅当日 `2026-08-18-pluggable-multi-region-gitservice-nfr-clarification.md` 已示范。
- **Action**: (1) 对仍在实施中的增量文档优先补 `## 幂等性审视` 表 (2) 其余存量按触达时补，不必一次性全改 (3) 新文档用 `check_nfr_idempotency_table.py --strict --files ...`
- **Why**: 缺表的旧 NFR 进入 `/6-ddd` 时仍可能漏掉过粗幂等键（`user_id`/`company_id`）类缺陷。
- **How to apply**: `docs/superpowers/plans/*-nfr-clarification.md`；模板见 `.claude/skills/5-nfr/SKILL.md` 与 `references/idempotency.md`
- **2026-08-19 夜**: 已对齐一份增量文档 — `2026-08-18-tenant-purchase-gitlab-region-nfr-clarification.md` 表头 `幂等性强制审视` → `幂等性审视`（docs `e5f9c1d` 已推送，`check_nfr_idempotency_table.py --strict` 通过）。其余在实施中缺表文档（billing-transactions-resource-ledger / runall-pending-migrate-badge 等）目标文件被他会话 WIP 覆盖，待其提交后按触达续补。

## [OPT-20260818-015] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 全部缺口闭环：消费者键（auth id/comment_id）taskEvents 5470152；cloud_access_key_iam_associations UNIQUE+upsert taskCloudService 6796efa/ec8c3f5；cloud_server_stopped stop_request_id 幂等表 dataMigrate a064dae + taskCloudService 3797b4b + taskEvents 7ad7c9b；cloud_server_started 并发 claim 竞态 → taskCloudService b583448（from_status 守卫原子 claim + 409）+ taskEvents 713cccd（claim 丢失/processing 重放幂等跳过）；container_migrate_await_ready 轻微 attempt 级 event_id 已天然区分。回归测齐全，双仓已推送。
- **Created**: 2026-08-18
- **Context**: 元规则 49 默认 `MemoryStore` 仅进程内去重；资金/配额/云资源（NFR ≥ L3）须在 owner 服务有 DB 唯一约束或幂等表，否则进程重启后可能重复副作用。
- **Action**: (1) 枚举 `taskEvents/cmd/**` 中 billing/cloud 类 intent；(2) 对照 NFR「幂等性审视」表；(3) 缺 DB 兜底的补 owner 侧唯一键或幂等表 + 回归测例。
- **Why**: 元规则 49 只强制共享 runner 与键粒度；L3 最终去重仍依赖持久化层。
- **How to apply**: `.ai/01_project_constraints/54_event_consumer_idempotency.md` §4；`taskBill` `billing_idempotency_key` 为参考实现
- **2026-08-19 夜·审计结论**: 枚举 13 个 billing/cloud intent，**已覆盖**：billing_transaction_created（`billing_transaction.transaction_id` UNIQUE + `billing_idempotency_key`）、profit_sharing/referral_settle（CAS status + 唯一号）、task_post_renewed（`task_post_renewal:<taskID>` 幂等键）、cloud_csc_reconcile/workspace_machine_idle（状态机重扫 no-op）。**消费者键缺口（已修，taskEvents `5470152` 已推送）**：(a) `CLOUD_PLATFORM_AUTHORIZATION_CREATED` 原回退 company_id 致租户内塌缩 → 改按 payload `id`（auth 记录）派生；(b) `BILLING_ORDER_COMMENT_CREATED` 原空 Kafka key + 无 comment_id 字段致全部订单评论塌缩为一个键 → 改按 `comment_id` 派生。双回归测加入 `key_test.go`。**owner DB 兜底缺口（未修，需 dataMigrate/taskCloudService 迁移，目标文件被他会话 WIP 覆盖）**：cloud_server_started/start_auto（无按 event_id 的 DB 唯一/幂等表，仅 MemoryStore + pending 状态过滤）、cloud_server_stopped（`stop_request_id` 从未落库，无唯一列）、cloud_platform_authorization_created（`cloud_access_key_iam_associations` 无 UNIQUE，REPLACE INTO 以新 snowflake 重复插行）、container_migrate_await_ready（轻微，无 DB 幂等表）。待 WIP 批提交后按 §4 补 owner 侧唯一键/幂等表。
- **2026-08-19 凌晨·gap (c) 已闭环**：`cloud_access_key_iam_associations` 加 `UNIQUE(cloud_platform_auth_id, access_key)`（dataMigrate/taskCloudService/027，已应用生产 task_cloud 并推送 6796efa）+ 导入改 `INSERT ... ON DUPLICATE KEY UPDATE`（taskCloudService `ec8c3f5` 已推送，整包测试 217s 绿）——重放的授权事件 upsert 既有行而非重复插行。回归测 `TestCreateAccessKeyIAMAssociationIdempotent` / `TestInternalImportAccessKeyIAMAssociationsDedup`。**其余 owner DB 兜底缺口仍 pending**：cloud_server_started/start_auto（需按 event_id 的幂等表）、cloud_server_stopped（stop_request_id 落库 + 唯一列）、container_migrate_await_ready（幂等表），跨 taskEvents+taskCloudService，留后续窗口。
- **2026-08-19 清晨·gap cloud_server_stopped 已闭环（3 仓已推送）**：`cloud_stop_request` 幂等表（dataMigrate `a064dae`，028，已应用生产 task_cloud）+ taskCloudService `3797b4b`（记录/查重端点 `/api/internal/cloud/stops/processed`、clear-after-stop 成功后 best-effort 落库、启动事件状态端点 `/api/internal/cloud-server-events/status`）+ taskEvents `7ad7c9b`（cloudserverstopped 在 DeleteInstance 前按 stop_request_id 查重，命中即幂等跳过；检查失败 fail-open 不误跳；cloudserverstarted 在 `LatestPendingStartEvent` 未命中时按载荷 event_id 查 DB 状态，success 即幂等跳过不再误投 DLT）。回归测：stop 重放 skip / fail-open、start 重放跳过、clear-after-stop 带 stop_request_id、StopRequestProcessed、CloudServerEventStatusByID。**仍 pending**：cloud_server_started/start_auto 并发 claim 竞态（pending→processing 非原子，重放成功已闭环）、container_migrate_await_ready 幂等表（轻微，attempt 级 event_id 已天然区分）。

## [OPT-20260817-013] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 后端 taskTaskService 7aa09ff + 前端 taskFE 8bf82d6 全部推送：搜索命中评论返回 comment_id → buildTaskDetailHref 附加 ?comment= → 详情页滚动+高亮。回归测：后端搜索 hit comment_id 断言、前端 buildTaskDetailHref 两例 + scrollToCommentById 三例。taskFE 指针待 WIP 清空后随 meta 同步（规则 32）。
- **Created**: 2026-08-17
- **Context**: 本次已支持粘贴 `task_<id>_cmt_<cmt>` / `cmt_<id>` 命中所属任务，但结果「打开任务」仍只进 task-detail 页顶，用户还要在评论列表里再找对应评论。
- **Action**: (1) 搜索 API 命中评论时在 hit 中返回 `comment_id` (2) `buildTaskDetailHref` 附加 comment 锚点或 query (3) 任务详情页挂载后滚到该评论并高亮
- **Why**: 用户从云控制台/日志复制的是评论级标识，只打开任务仍要二次定位。
- **How to apply**: `taskTaskService/src/task_store_search.go`；`taskFE/app/src/utils/navbarTaskSearch.js` `buildTaskDetailHref`；任务详情评论列表滚动/高亮
- **2026-08-19 夜**: **(1) 后端已完成** — taskTaskService `7aa09ff` 已推送：`taskSearchHit` 增 `CommentID`，命中评论标识时 hit 携带 `comment_id`（SQL 已按 comment id 过滤），`handleSearchTasks` JSON 输出 `comment_id`；回归测扩展（容器名/评论 id 搜索均断言 comment_id）。**(2)(3) 前端已完成** — taskFE `8bf82d6` 已推送：`buildTaskDetailHref` 支持 commentId 附加 `?comment=<id>`；`NavbarTaskSearch` taskHref 透传 `hit.comment_id`；`TaskDetailConversationFeed` 评论气泡加 `data-comment-id` + `comment-highlight` 样式；新增 `scrollToCommentById` 助手，`TaskDetailCommentsSection` 挂载后按路由 `?comment=` 滚动+高亮（同一 id 只滚一次）。回归测：navbarTaskSearch buildTaskDetailHref 两例 + scrollToCommentById 三例，全部 vitest 绿。**taskFE 指针待 WIP 清空后随 meta 同步（规则 32）**。

## [OPT-20260818-040] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: runAll 1797865 已推 upstream：BuildMigratePendingReport 4 worker 并发 inspect（结果按 entries 顺序，ctx 取消标 unreachable），补并发重叠+耗时优于串行估计单测，全量 go test ./... 绿；合入前一夜未提交的 migrate-status 徽章 UI（16.js/04.css/label/confirm suffix/manifest）。
- **Created**: 2026-08-18
- **Context**: `GET /api/dev/migrate-status` 对 registry 各 MySQL 串行 `SELECT step_key`。库数十几个时首屏徽章可能接近 8s 超时边界。
- **Action**: (1) 在 `BuildMigratePendingReport` 或 Runner 适配层用最多 4 个 worker 并发 `ListApplied` (2) 保持表缺失=空 applied、连接失败=unreachable (3) 补单测：fake reader 延迟下总耗时小于串行
- **Why**: 降低 9999 打开时「未 migrate」徽章的等待，避免超时把全部标成 unreachable。
- **How to apply**: `runAll/src/domain/migrate_pending.go` 或 `migrate_pending_status.go`；测例放 `migrate_pending_test.go`

## [OPT-20260818-044] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: taskProjectService 8b515c0 已推 upstream：gitlabRESTGet 增 cookieHost 门控（host 匹配才发 _gitlab_session，否则仅 OAuth Bearer），defaultGitLabHost() 从 base.yaml 解析主站 host（sync.Once 缓存）；连带前一夜未提交的区域/自建 gitlab host 不借 gitlab:default provider（matchProvider/fallback）；修复 OPT-20260818-043 独立化后过时的 provider 模板断言；回归测区域 URL+默认 cookie 不覆盖 Bearer、主站 host 匹配时 cookie 生效，全量 go test ./src 绿。
- **Created**: 2026-08-18
- **Context**: `gitlabRESTGet` 在 cookie 非空时只发 `_gitlab_session`、忽略 OAuth Bearer。本次 401 主因是跨实例 token，但若浏览器/网关把默认 GitLab session 带到 taskProjectService，区域仓仍会 401。
- **Action**: (1) cookie 仅当请求 host 与 session 所属 GitLab 实例一致时使用 (2) 否则只用该实例 OAuth token (3) 补测：tencent-sh-1 URL + 默认 session cookie 不得覆盖正确 Bearer
- **Why**: cookie 优先于 token 会让刚修好的实例隔离再次被默认 CE 会话污染。
- **How to apply**: `taskProjectService/src/git_branches.go` `gitlabRESTGet`；调用方 `branch_handlers.go`

## [OPT-20260818-038] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: AiMonitor 15cd46a 已推 upstream：promtail-local.yaml 新增 job git-service-tencent-sh-1（__path__=/var/log/runall/git-service-tencent-sh-1.log，复用 git-service pipeline_stages：runAll 前缀剥除 + slog JSON + plaintext level + trace_id structured_metadata），yaml 校验通过。
- **Created**: 2026-08-18
- **Context**: `gitlab-regions` 组已把上海实例编进 9999；Promtail 仍只有 `job: git-service` → `/var/log/runall/git-service.log`。SH-1 的 SSOT 是 `logs/infra/git-service-tencent-sh-1.log`，Loki 搜不到该区启停失败。
- **Action**: (1) 在 `AiMonitor/promtail/promtail-local.yaml` 增加 `job_name: git-service-tencent-sh-1`，`__path__` 对齐 conf `logging.log_file`（及 runAll tee 若存在）(2) 复用现有 git-service pipeline_stages (3) 用 `python3 -c "import yaml; yaml.safe_load(open('AiMonitor/promtail/promtail-local.yaml'))"` 校验
- **Why**: 上海实例探活/启动失败时无法用 Loki 按 job 过滤，9999 组无法排障。
- **How to apply**: `AiMonitor/promtail/promtail-local.yaml` job `git-service` 旁复制；`conf/infra/git-service-tencent-sh-1/config.yaml` 的 `logging.log_file`

## [OPT-20260818-005] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: runAll d87ff2c 已推 upstream：StopAllApplicationsExcept 增 onProgress 逐服务上报「已停止服务: <name>」+ stopServiceForDevDatabaseClearWithTimeout 90s 单服务超时兜底（阻塞 compose/ssh stop 超时记日志继续）；回归测 domain 层（Clear onProgress 逐服务消息）与 runner 层（exec 探针服务停止路径上报），全量 go test ./... 绿。
- **Created**: 2026-08-18
- **Context**: 在 9999 执行「清空全部数据库」时，日志在「正在停止所有应用服务」后约 3 分钟无新行；实际在串行停 git-service / ai-monitor 等 compose 服务，UI/轮询看似假死。最终仍成功（71 服务停止，MySQL/Redis/Kafka ok）。
- **Action**: (1) 在 `StopAllApplicationsExcept` / `stopServiceForDevDatabaseClear` 每停一个服务 `onProgress` 写出服务名 (2) 对 compose stop 设超时并在超时时记日志 (3) 补单测或手工验收：清库日志出现逐服务行
- **Why**: 停止阶段是清库最长步骤；无进度时运维难判断卡死还是慢停，容易误杀 runAll。
- **How to apply**: `runAll/src/runner.go` `StopAllApplicationsExcept`；`domain/database_platform_reset.go` 的 `onProgress`；对照 `/api/dev/logs?tool=db-clear`

## [OPT-20260817-034] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: runAll f6dac81 已推 upstream：ensureServiceNotReachable/finalizeServiceStop/ensureServicePortsReleased 增加 ctx 参数，内部 checkProbe 不再用 Background；restartService/executeServiceStop 传调用方 ctx，慢响应健康端点随精确重启/停止阶段超时取消。回归测：阻塞端点随 ctx 取消 2s 内返回，全量 go test ./... 绿。
- **Created**: 2026-08-17
- **Context**: 排查精准编译重启卡住时发现 `ensureServiceNotReachable` 使用 `context.Background()` 调用 `checkProbe`。若健康 URL 阻塞（慢响应），停止阶段可无视 precise-restart 的单服务超时继续挂起数秒至数分钟。
- **Action**: (1) 为 `ensureServiceNotReachable` / 相关 stop 探针增加 `ctx context.Context` 参数 (2) `restartService` / `finalizeServiceStop` 传入调用方 ctx (3) 补单测：阻塞探针在 ctx cancel 后迅速返回
- **Why**: 单服务超时只包住 `RestartServiceWithActor`，内部 Background 探针会突破截止时间，进度条仍可能「假死」。
- **How to apply**: `runAll/src/runner.go` 的 `ensureServiceNotReachable`、`checkProbe` 调用链；验收：`go test ./src -run 'EnsureServiceNotReachable.*Context|StopPhase.*Cancel'`

## [OPT-20260817-035] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: runAll c0752ec: src/ui.go 1850行拆分为 ui_helpers/ui_service/ui_build/ui_start_stop/ui_lifecycle/ui_dev/ui_status/ui_proxy/ui_server/ui_logs/ui_observability 各 ≤500 行；registerUIHandlers 保持入口，路由集与 HEAD 完全一致；pre-commit 整包 go test ./src 102s 绿，已推送
- **Created**: 2026-08-17
- **Context**: 本次修改 cancel 签名时 `ui.go` 已约 1849 行，远超源文件 500 行门禁；属遗留巨石，继续堆 cancel/orphan 逻辑会加剧审查成本。
- **Action**: (1) 按 handler 域拆到 `ui_start_all.go` / `ui_stop_all.go` / `ui_build_all.go` / `ui_cancel.go` 等 (2) 保持 `registerUIHandlers` 入口 (3) `go test ./src` 全绿且 `wc -l` 各文件 ≤500
- **Why**: 行数门禁元规则要求超标即削；巨石阻碍后续 bulk/orphan 修复。
- **How to apply**: `runAll/src/ui.go` 及拆出文件；对照 `.ai/01_project_constraints/27_source_file_line_limit_auto_reduce.md`

## [OPT-20260819-009] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 已将 wechat_pay_amount.go / wechat_pay_prepay.go 合入 src；handlers_orders_pay 按分下单 + kycAmountYuanCeil；恢复 handlers_orders_pay_wechat_amount_test；全包 go test 通过。aside 中过时副本已删。后续会话补齐：gitlab_region_deploy 接线 SystemAdmin 列表、stripTenantIDKV 约定路径、adminGrantTransactionDescription 入库具体文案；`_wip_aside/` 目录已删除且移出 `.gitignore`。
- **Created**: 2026-08-19
- **Context**: 为编译本会话退款修复，将冲突的未跟踪 WIP（`wechat_pay_prepay.go`、`wechat_pay_amount.go` 及若干测试）移至 `taskBill/_wip_aside/`。
- **Action**: (1) 完成从 `wechat_pay.go` 抽取到 prepay/amount 文件并删除重复声明 (2) 恢复 aside 测例并修齐符号 (3) 全包 `go test` 绿灯 (4) 清空并删除 `_wip_aside/`
- **Why**: aside 文件不进编译会导致他人 WIP 丢失上下文；长期重复声明会阻断构建；OPT 闭环后停车场目录不得残留
- **How to apply**: 原 `taskBill/_wip_aside/` 内容已落 `taskBill/src/`（deploy / convention path / admin_grant description）

## [OPT-20260819-019] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: 已将 gitlab_region_deploy / admin_grant_description / stripTenantIDKV 等 aside 残留合入 src（taskBill d3347c3）；删除 _wip_aside 与 .gitignore 旁路条目；go test ./src 全绿。
- **Created**: 2026-08-19
- **Context**: OPT-20260819-009 已合并微信按分下单拆分；aside 仍残留与微信无关的未合入文件：`gitlab_region_deploy*.go`、`admin_grant_description_test.go`、`handlers_http_util_test.go`、`gitlab_convention_path_http_test.go`。其中部署路径 API 已被 OPT-20260818-035 等引用为 `src/gitlab_region_deploy.go`，但文件仍在 aside。
- **Action**: (1) 评估 `gitlab_region_deploy.go` 是否应合入 src 并接线 SystemAdmin API (2) `admin_grant_description_test` 对照当前 `admin_grant.go` 描述文案决定合入或删除 (3) `stripTenantIDKV` / convention path 测例若仍缺实现则补齐或废弃 (4) 清空 `_wip_aside/`
- **Why**: 旁路目录继续藏 WIP 会导致「以为已上线」的 OPT/文档与 src 漂移。
- **How to apply**: `taskBill/_wip_aside/` → 评估后合入 `taskBill/src/` 或删除；对照 OPT-20260818-035/036

## [OPT-20260819-021] completed

- **Status**: completed
- **Completed**: 2026-08-19
- **Summary**: Closed 10 stale PRs; registered auth_kyc_* + /api/kyc/ on db main; deleted remote feat branches.
- **Created**: 2026-08-19
- **Context**: ADR-0019 将 ship 改为默认直接合入 main；此前审计仍有约 10 个开放 PR（资源台账 / KYC / CSC meta 等），多数功能已在 main，db#14/#15 ownership 可能需另开基于当前 main 的补登记。
- **Action**: (1) 对 conf#16 docs#54 taskBill#12/#9 taskFE#2 docs#42/#43 ram-work#25 执行 `gh pr close` 并注释 superseded (2) 复核 db#14/#15 是否仍缺 ownership，缺口则新开 PR/直接合入，否则一并关闭 (3) 跑 `delete_merged_feat_branches.py --apply`
- **Why**: 开放幽灵 PR 与新交付路径冲突，易误导后续 merge。
- **How to apply**: 见会话 Canvas `open-prs-audit`；ADR-0019；`gh pr list --state open` 扫 task2money 各仓

## [OPT-20260819-023] completed

- **Status**: completed
- **Created**: 2026-08-19
- **Completed**: 2026-08-19
- **Completion-Note**: 链接改为 `/system-admin/order-records/?tenant_id=&order_id=`；`SystemAdminOrderListPanel` 经 `useSystemAdminOrderListDeepLink` 选租户并对齐展开高亮；vitest 19 项通过。
- **Context**: 退款审批已并入 `/system-admin/order-records/?tab=refund`，但「关联订单」仍链到租户 `/billing/orders/?order_id=`。
- **Action**: (1) 改为管理端订单 Tab + `order_id` 深链（或双链）(2) 为 `SystemAdminOrderListPanel` 补 `useBillingOrderIdDeepLink` 等价能力 (3) 更新意图验收点
- **Why**: 超管在同一入口核对订单与退款，减少跳出租户侧栏。
- **How to apply**: `SystemAdminRefundPanel.vue`；`SystemAdminOrderListPanel.vue`；`billingOrderDeepLink.js`

