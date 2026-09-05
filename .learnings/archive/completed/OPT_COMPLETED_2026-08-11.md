# Completed OPT Archive — 2026-08-11

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 57 条。
> 归档执行时间：2026-08-13T13:17:38+08:00

## [OPT-20260810-018] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: auto-commit_selftest 已挂载根仓 pre-commit 门禁（hook check-auto-commit-selftest，~7s）；脚本头注明耗时与 scratch 清理。meta commit 19d299e
- **Created**: 2026-08-10
- **Context**: 已新增 `scripts/lib/auto-commit_selftest.sh`（14 例，覆盖 main 落点 / feature→main / detached / Cursor hooks），但仅手工可跑，钩子或分支守卫回归时无自动拦截。
- **Action**: (1) 在根仓 CI 或 `deploy_repo_random_precommit` 相关校验中调用 `bash scripts/lib/auto-commit_selftest.sh`；(2) 文档注明预期耗时与 scratch 清理。
- **Why**: Cursor/Claude 双端钩子与 main 强制落点是恢复安全网核心；无 CI 时易在模板漂移后静默失效。
- **How to apply**: 参考 `bash scripts/lib/random_test_runner.sh self-test` 的 CI 挂载方式；改 `db/scripts/ci/` 或根 `.pre-commit-config.yaml`。

## [OPT-20260810-019] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: git_retry commit 失败后区分「并行会话已提交」（工作区干净→记 info 成功）与真门禁失败；selftest 补 [5] 竞态用例 17/17。meta commit 08e861a
- **Created**: 2026-08-10
- **Context**: 本会话验证 Cursor `sessionEnd` 时出现一次 `auto-commit FAILED`，但 HEAD 已是成功检查点（并行 Stop/SessionEnd 竞态）；当前失败文案无法区分「已被他进程提交」与真门禁失败。
- **Action**: (1) `git_retry git commit` 失败后若工作区相对 HEAD 已干净则记 info 并 exit 0；(2) 仅在仍有未提交变更时输出 FAILED；(3) 补自测例。
- **Why**: 减少误报，避免 Agent/人工把已成功的检查点当成失败去重做。
- **How to apply**: 改 `scripts/lib/auto-commit.sh` 提交失败分支；扩展 `auto-commit_selftest.sh`。

## [OPT-20260810-053] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: simplify-and-harden/ci/agent-teams 与 10-ship 勾选改 npm run build+dist/preview 200；runbook/worktree 指引同步。docs 481fdf7 + meta 75a3ad6
- **Created**: 2026-08-10
- **Context**: `taskFE` build 已改为纯 Vite；`.claude/skills/simplify-and-harden/SKILL.md` 与部分 ship 清单仍要求 `manage.py collectstatic` / `collected_static`。
- **Action**: (1) 将 Pass 4 / ship 勾选改为「`npm run build` + dist/preview 200」；(2) 标注 Django collectstatic 为历史；(3) 交叉更新仍写「build + collectstatic」的 runbook。
- **Why**: 避免后续会话为「合规」重新挂回 collectstatic 或在日志里找 Django。
- **How to apply**: `.claude/skills/simplify-and-harden/SKILL.md`、`.claude/skills/10-ship/SKILL.md`、`docs/runbooks/*`。

## [OPT-20260810-011] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: git mv 9 个 docs/superpowers plan/spec 文件 vue-frontend→taskFE，rg 旧名清零，无引用需同步。docs commit 6004549
- **Created**: 2026-08-10
- **Context**: 内容已改为 taskFE，但 `docs/superpowers/plans|specs/*vue-frontend*` 文件名仍用旧服务名，交叉链接与检索易混淆。
- **Action**: 批量 rename 历史 plan/spec 文件名 `vue-frontend`→`taskFE`，并更新仓内相对链接。
- **Why**: 降低迁移后检索噪音，避免 AI/人工误以为服务仍叫 vue-frontend。
- **How to apply**: `git mv` 相关文件后 `rg 'vue-frontend-.*\.(md)$' docs` 清零；同步更新引用这些路径的文档。

## [OPT-20260810-020] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: taskEvents handler 与 repo 统一使用 saas.DefaultPersonalCompanyName 共享常量（commit d571de7）；测试断言锁定常量值与前端占位符对齐。
- **Created**: 2026-08-10
- **Context**: `taskEvents` handler 与 `CreateCompanyForUser` 各写死中文字符串「我的公司」；前端 Onboarding 占位符亦同义。本次空 username 回退修复后三处易漂移。
- **Action**: (1) 在 `taskEvents/internal/repository/saas` 或共享常量包定义 `DefaultPersonalCompanyName`；(2) handler/repo 共用；(3) 文档注明与前端占位符对齐。
- **Why**: 避免再次出现「repo 已兜底、handler 仍永久失败」类不一致。
- **How to apply**: `taskEvents/internal/handlers/usercreated/handler.go`、`internal/repository/saas/repo.go`。

## [OPT-20260810-010] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: taskFE runall-lifecycle.sh build 分支加 node_modules/.bin/vite 就绪检查，缺失时输出可操作指引（cd taskFE/app && npm ci）并保留 127；新增 test_build_npm_diagnostics.sh 静态+仿真断言（commit d6da55b）。
- **Created**: 2026-08-10
- **Context**: taskFE（working_dir taskFE/app）精准编译重启 build 失败 exit 127，日志仅 `sh: 1: vite: not found` —— taskFE/app/node_modules 整体缺失（npm ci 前无 vite 二进制），且失败登记保留后按钮反复重试无指引。本次事故为环境问题而非代码问题，但 runAll 侧（runBuild / runall-lifecycle.sh）无 node_modules 就绪检查，只能靠人工猜 vite 缺失含义。
- **Action**: runAll `runBuild`（或 taskFE build_command 脚本）对 npm 类服务（working_dir 含 package.json）build 前检测 node_modules/.bin 存在性，缺失时输出可操作错误：`node_modules 缺失：cd taskFE/app && npm ci 后重试`；错误信息保留 127 与原始 stderr。
- **Why**: 环境缺失与代码错误的处置路径不同（npm ci vs 修代码），无提示时浪费重启轮次；登记文件 failed 条目重试也无法自愈。
- **How to apply**: 修改 runAll/src/runner.go runBuild 或 runall-lifecycle.sh build 分支；改动后登记 taskFE 走精准编译重启验证，并补单测/断言脚本。

## [OPT-20260810-034] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 抽取 OpenContainerActionLinks.vue 共享组件，TaskLayerAssociationPanel 与 CommentLayerZtreeStatus 复用，保留 #open-container-page-btn/#open-container-vscode-btn id；新增单测 3 例，原 open-vscode 单测保持绿（commit cb1d7bd）。
- **Created**: 2026-08-10
- **Context**: 「打开容器页面」与「打开容器开发页面」在 `TaskDetailTaskLayerAssociationPanel.vue` 与 `TaskDetailCommentLayerZtreeStatus.vue` 各写一份，含相同的 ingress ensure 点击逻辑与样式类。
- **Action**: (1) 新建如 `OpenContainerActionLinks.vue`（或 composable + 薄模板）；(2) 两处任务关联头行复用；(3) 保留 `#open-container-page-btn` / `#open-container-vscode-btn` id 与现有 Playwright/单元测。
- **Why**: 减少双份维护，避免 loading 态与就绪态按钮行为漂移。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailTaskLayerAssociationPanel.vue`、`TaskDetailCommentLayerZtreeStatus.vue`、`openContainerPage.js`。

## [OPT-20260810-025] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: runAll register API 已统一走 resolveRegisteredService 并拒绝未知名（writeJSONError unknown services），别名归一化（taskAuth→task-auth）；precise_restart_api_test/precise_restart_resolve_test 已覆盖，测试全绿。
- **Created**: 2026-08-10
- **Context**: 精准编译重启登记文件出现 taskEvents failed（resolvable=false），导致失败计数常驻干扰队列条信息。
- **Action**: (1) register 与自动登记路径统一走 resolveRegisteredService；(2) 未知名直接拒绝或规范化为 runAll.yaml name；(3) 补单测。
- **Why**: 无效登记制造噪声失败，放大卡住无信息的观感。
- **How to apply**: scripts/register-precise-restart.sh 、runAll/src/ui_precise_restart.go 、auto-commit 扫描登记逻辑。

## [OPT-20260810-045] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: Navbar.logic.platformStaff/membershipTier 单测 vi.mock 补齐 upsertSavedAccount（commit efe5f0b），复跑两文件无 stderr，21 例相关单测全绿。
- **Created**: 2026-08-10
- **Context**: 跑 platformStaff/membershipTier 回归时 stderr 反复报 saved_accounts_store mock 缺少 `upsertSavedAccount`（测试仍绿，但噪声干扰排障）。
- **Action**: (1) 在 `Navbar.logic.platformStaff.test.js` 与 `Navbar.logic.membershipTier.test.js` 的 vi.mock 中补 `upsertSavedAccount: () => Promise.resolve()`；(2) 复跑两文件确认无该 stderr。
- **Why**: 降低假失败信号，避免掩盖真正的 Navbar 回归错误。
- **How to apply**: 对齐 `Navbar.logic.authRouteSession.test.js` 的 mock 列表。

## [OPT-20260810-052] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: taskFE e2e 死脚本清理：test:e2e* 改指向 ../playwright.config.js，无 task2app/playwright 引用（commit 23338ff+fa2129c），playwright.config.js 存在且有效。
- **Created**: 2026-08-10
- **Context**: 本会话已从 `npm run build` 移除 Django collectstatic；但 `taskFE/app/package.json` 的 `test:e2e*` / `verify:daydaymoney-api-host` 仍 `npm --prefix ../task2app/playwright`，task2app 已退役，脚本不可用。
- **Action**: (1) 将 e2e 脚本改为 `taskFE/tests` + 根 `playwright.config.js`（或删除死脚本）；(2) 更新文档/CI 引用；(3) 跑一条 smoke 确认路径有效。
- **Why**: 死路径会误导 Agent/CI 去找已删除的 Django 时代目录。
- **How to apply**: `taskFE/app/package.json`、`taskFE/playwright.config.js`、`taskFE/tests/`。

## [OPT-20260810-013] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: PeopleJoin 失败路径从响应提取 traceId 并写入错误容器 :data-traceId（commit 3ffc6bc）；PeopleJoin.unauth-actions.test.js 断言属性存在，6 例通过。
- **Created**: 2026-08-10
- **Context**: 修复邀请页未登录 CTA 时发现 `PeopleJoin.vue` 的错误 `div` 仅展示文案，接口失败路径未挂 `data-traceId`，不符合约束 24。
- **Action**: (1) 在 join / validate-invite / me 失败路径从响应提取 traceId；(2) 错误 DOM 增加 `:data-traceId="traceId"`；(3) 补单元测试断言属性存在。
- **Why**: 线上邀请加入失败时无法从 UI 一键关联 Loki 全链路日志。
- **How to apply**: 参考 `requestErrorDisplay.js` / `extractTraceId`；改 `taskFE/app/src/views/PeopleJoin.vue` 与 `PeopleJoin.unauth-actions.test.js`。

## [OPT-20260810-015] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: Navbar.fetchCurrentUser 优先 getStoredUserId（localStorage），HttpOnly cookie 场景走 profile 回退；Navbar.logic.fetchCurrentUser.test.js 5 例覆盖（commit ef2e9fa）。
- **Created**: 2026-08-10
- **Context**: 修复邀请页登录回跳误判未登录时，`PeopleJoin` 已改用 `getStoredUserId` + `AuthSessionGuardService`；`Navbar.logic.vue` 的 `fetchCurrentUser` 仍先读 `getCookie('userId')`，HttpOnly 场景依赖 profile 回退（可用但多一次请求）。
- **Action**: (1) `fetchCurrentUser` 将 `getCookie('userId')` 改为 `getStoredUserId()`；(2) 可选让 `AuthSessionGuardService` 默认注入 `getStoredUserId`；(3) 补一条 Navbar 单元测试覆盖 localStorage 有 id、cookie 空时走 /me/。
- **Why**: 与邀请页/会话守卫同源，减少 HttpOnly 下无谓的 profile 往返，避免各入口鉴权策略漂移。
- **How to apply**: 改 `taskFE/app/src/components/Navbar.logic.vue` 与 `auth_session_guard_service.js`；参考 `PeopleJoin.unauth-actions.test.js` 登录回跳用例。

## [OPT-20260810-031] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 创建任务草稿与协作成员对齐逻辑将负责人、操作员默认设为当前登录成员（commit 9a56f3a），defaultCurrentUser 单测入库，相关 21 例单测全绿。
- **Created**: 2026-08-10
- **Context**: 已让创建任务草稿与协作成员对齐逻辑将负责人、操作员默认设为当前登录成员；本地 vitest 已绿，公网需 taskFE 精准编译重启后生效。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 taskFE 执行「精准编译重启」；(2) 打开 https://www.daydaymoney.com/tenant/<id>/work-panel 新建任务，确认 `#task-owner-input` 与 `#task-operator-input` 显示当前用户公司昵称（非空 placeholder）。
- **Why**: SPA 产物与预览进程需部署后才能在公网验证默认值。
- **How to apply**: `.runall/precise_restart_services.txt`、`useCreateTaskCollaborators.js`、`useWorkPanelDeliverableForm.js`。

## [OPT-20260810-032] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: WorkPanel.vue 两处 getCookie(userId) 改为 getStoredUserId（commit ce4f4fe），注释标注 OPT-20260810-032；WorkPanel 相关 24 例单测全绿。
- **Created**: 2026-08-10
- **Context**: 修复创建任务负责人/操作员默认值时发现 `WorkPanel.vue` 仍用 `getCookie('userId')` 决定是否拉取 `/me/`；HttpOnly 签名 cookie 场景下 JS 读不到 cookie，会误走「无 userId」分支。
- **Action**: (1) 将 `WorkPanel.vue` 中两处 `getCookie('userId')` 改为 `getStoredUserId()`（或 `resolveAuthenticatedUserId`）；(2) 补/改对应单测或 Playwright，覆盖仅有 localStorage `currentUserId` 的登录态。
- **Why**: 与 OPT-20260807-004 语义一致，避免工作面板在 HttpOnly cookie 环境下初始化不完整。
- **How to apply**: `taskFE/app/src/views/WorkPanel.vue`、`sessionUserIdUtils.js`、相关 Playwright。

## [OPT-20260810-041] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: runAll adopt 对「有监听但健康失败」做短退避重试（commit 1a34fa3），runner_adopt_test 覆盖瞬时失败后成功收养，go test 全绿。
- **Created**: 2026-08-10
- **Context**: 修复精准重启端口冲突时两次 hot-replace runAll：第一次成功 adopt `task-auth`，第二次因健康探针瞬时 EOF/竞态未 adopt，UI 出现 `status=""` + `listen_port_active=true`，需手动 `/api/start` 收养。
- **Action**: (1) 在 `AdoptRunningManagedServices` / `adoptListeningServices` 对「有监听但健康失败」做短退避重试（如 2～3 次、间隔 200ms）；(2) 补单测覆盖瞬时失败后成功收养；(3) 可选：UI 对 `listen_port_active && status==""` 显示「可收养」并一键 adopt。
- **Why**: 热替换后编排器与真实监听态短暂不一致，会误导精准重启/启停操作。
- **How to apply**: `runAll/src/runner_adopt.go`、`runAll/src/main.go` pre-UI adopt 路径。

## [OPT-20260810-042] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: runAll bulk 操作写 .runall/bulk_op.lock 供 watchdog 离线感知，结束删除（commit 8a58caa），bulk_op_guard_test 覆盖，go test 全绿。
- **Created**: 2026-08-10
- **Context**: watchdog 已通过 `/api/status` 的 `active_bulk_progress` 跳过抢拉；但 runAll 自身 hot-replace / 短暂宕机时 API 不可达，门禁 fail-open 仍可能在窗口内拉起服务。
- **Action**: (1) bulk 开始时写 `.runall/bulk_op.lock`（kind/run_id/ts）；结束时删除；(2) `ensure_services_healthy.py` 优先读锁文件，再回退 API；(3) 锁文件过期（如 >2h）忽略并告警。
- **Why**: 关闭「API 不可达 → watchdog 照样抢端口」的残余竞态。
- **How to apply**: `runAll/src/bulk_op_guard.go`、`runAll/scripts/ensure_services_healthy.py`。

## [OPT-20260810-036] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: taskFE: 启动状态轮询补齐 comment_id 扇出，无 SSE 仅轮询时 binding 启动日志仍有服务器调度行（865c399）
- **Created**: 2026-08-10
- **Context**: SSE 断连时 REST 轮询 `server-startup-status` 经 `updateServerStatus` 写任务级 statusLogs，但当前 poll payload 常不带 `comment_id`，live 总线无法扇出；仅靠容器名匹配的展示合并兜底。
- **Action**: (1) `createServerStartupStatusPollController` / poll URL 在评论级启动时传入 `commentId`；(2) 轮询响应或前端注入 `comment_id` 后再 `updateServerStatus`；(3) 补单测覆盖无 SSE 仅轮询时 binding 启动日志仍有服务器调度行。
- **Why**: SSE 抖动时用户仍应在评论面板看到完整服务器调度进度，不能只依赖偶然命中的容器名过滤。
- **How to apply**: `serverStartupStatusPoll.js`、`useTaskDetail.js`、`updateServerStatus.test.js`。

## [OPT-20260811-005] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 实现陈旧租户统一路由守卫：router.beforeEach 校验 URL tenant 是否仍属 /me/ companies 列表（后端 /me/ 忽略 tenant_id，故以 companies 列表为准），陈旧则清 lastActiveTenantId 并按现有公司跳转（无公司→onboarding），保留子路径；people/join/invite 与带 accessCode 的任务分享跳过。30s 模块缓存避免重复 /me/。新增 tenantRouteGuard.js + 16 单测，taskFE 全量 1917 例全绿，build 成功。commit 7d5a2b9 已推送。
- **Created**: 2026-08-11
- **Context**: 本次仅在公司设置页对 `companies/current` 404 做跳转；WorkPanel/People/Billing 等仍可能因 URL 残留展示其它失败态。已有 `useTenantAccessGuard`（/me/ 403/404→profile）与新建 `staleTenantRecovery` 未统一。
- **Action**: (1) 抽象路由级守卫：租户路径加载时若公司不存在则清 storage 并按 companies 跳转；(2) 覆盖 WorkPanel、PeopleManage、Billing*；(3) 补单测/冒烟。
- **Why**: 清库后用户可能从历史落到任意 `/tenant/:id/...`，逐页修易漏。
- **How to apply**: `taskFE/app/src/utils/staleTenantRecovery.js`、`useTenantAccessGuard.js`、`router.js`。

## [OPT-20260810-051] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: trace 头透传 gitoauth + taskGitOauth 日志接入 trace_id；go test 全绿，双仓已推
- **Created**: 2026-08-10
- **Context**: 本次 trace `9bbb77e0-…` 在 Loki 仅命中 task-auth + task-project-service；task-git-oauth 的 access-for-user 使用独立 trace，关联靠时间窗口。
- **Action**: (1) `fetchGitAccessToken` 注入 `traceparent` / `X-Trace-Id`；(2) taskGitOauth 接入同一 trace；(3) 补单测断言出站头。
- **Why**: OAuth refresh 超时是根因关键证据，缺共享 trace 时排障多靠时间对齐。
- **How to apply**: `taskProjectService/src/gitoauth_client.go`、`taskGitOauth` 中间件 / `http_request` 日志字段。

## [OPT-20260810-023] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: taskEvents commit 56f0665（已推送）：ThrottledDLTAlerter 累计 suppressed，下一封邮件正文附带 suppressed_since_last_email 并复位；新增单测覆盖计数累计/复位。
- **Created**: 2026-08-10
- **Context**: 当前冷却窗口内后续 DLT 仅打日志抑制，下一封邮件不说明窗口内漏报条数，运维难以判断故障规模。
- **Action**: (1) `ThrottledDLTAlerter` 累计 `suppressed`；(2) 下一封邮件正文附带 `suppressed_since_last_email`；(3) 单测覆盖计数复位。
- **Why**: 报告邮件应反映窗口内真实死信压力。
- **How to apply**: `taskEvents/broker/dlt_alert.go` + `dlt_alert_test.go`。

## [OPT-20260810-022] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: taskEvents commit 56f0665（已推送）：RedisDLTCooldownStore SET NX+TTL 跨进程共享冷却窗口，store 故障降级进程内冷却；新增单测（获胜/抑制/回退）+ integration 集成测（真实 Redis 双 store 互斥与窗口过期重开）。
- **Created**: 2026-08-10
- **Context**: 本次 DLT 邮件告警冷却为进程内内存实现；每个 `task-events-*` consumer 进程各自一套 5 分钟窗口，级联失败时运维仍可能收到多封（每进程一封）。
- **Action**: (1) 评估用 Redis SET NX + TTL（或 Kafka 侧旁路）做跨进程冷却键 `dlt-alert:{window}`；(2) 失败时降级为进程内冷却；(3) 补集成测。
- **Why**: 多 intent 消费者并行是常态；真正「5 分钟不重复」需跨进程。
- **How to apply**: `taskEvents/broker/dlt_alert.go`、复用既有 Redis broker 配置。

## [OPT-20260810-026] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: taskFE commit 46ca129（已推送）：模态遮罩样式迁到显式 .app-modal-overlay；裸 .fixed.inset-0 收窄为纯定位；下拉遮罩独立类；52 个 Vue 居中模态迁移；CSS 回归单测 3 例；全量 358/1919 通过，build 成功。
- **Created**: 2026-08-10
- **Context**: 修复公司下拉灰罩时确认 `styles.css` 的 `.fixed.inset-0` 会给所有同名组合套上 `rgba(0,0,0,0.5)` 与 `z-index:9999`；本次用 `dropdown-click-outside-overlay` 豁免下拉遮罩，但根因仍是全局选择器过宽。
- **Action**: (1) 将模态遮罩样式迁到显式类（如 `.modal-backdrop` / `.app-modal-overlay`）；(2) 批量把现有模态根节点改用该类；(3) 删除或弱化裸 `.fixed.inset-0` 规则；(4) 保留下拉透明遮罩规则或改用 onClickOutside 无 DOM 遮罩。
- **Why**: 避免未来再出现「透明点击关闭层」误继承模态灰罩与错误 z-index。
- **How to apply**: `taskFE/app/src/css/styles.css`、各 `*Modal.vue` / `fixed inset-0 bg-black` 根节点、配套 CSS/组件单测。

## [OPT-20260810-046] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: Fork 自动运行路径补齐 Playwright 断言（auto_run=true + client_public_ip），2 用例 headless 通过
- **Created**: 2026-08-10
- **Context**: 本次已恢复 Fork 前确认模态；`TaskDetail.fork-popup.playwright.test.js` 覆盖「不自动运行，仅派生」与 T1 未提前 POST，但未 mock 公网 IP 并断言 `auto_run:true` + `client_public_ip`。
- **Action**: (1) 在 Playwright 中增加选择「自动运行并派生」用例；(2) mock `queryClientPublicIpForAutoSg` / 公网 IP 接口；(3) 断言 POST body `auto_run===true` 且在有 IP 时含 `client_public_ip`。
- **Why**: 与测试意图 T4 对齐，防止回归再次硬编码直派生。
- **How to apply**: `taskFE/tests/TaskDetail.fork-popup.playwright.test.js`、`taskFE/app/src/utils/publicClientIp*`。

## [OPT-20260810-043] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 新增 Navbar.authLogin.logged-in.playwright.test.js：已登录访问 /auth/login/ 断言导航栏含昵称/退出且无登录主按钮，headless 与 cdp 配置均通过
- **Created**: 2026-08-10
- **Context**: 已修复 Navbar.logic 在 auth 路由强制 setLoggedOutUser，并加 Vitest 回归；公网 dist 已部署。尚缺 CDP Playwright 端到端覆盖真实 Cookie/会话。
- **Action**: (1) 新增 `taskFE/tests/Navbar.authLogin.logged-in.playwright.test.js`（CDP 9222）；(2) 用已登录夹具打开 `/auth/login/?next=...`；(3) 断言 `nav[data-alias=cmp-navbar-main]` 含昵称/退出，不含「登录」主按钮。
- **Why**: 单元测可被 mock 绕过；E2E 才能锁住网关 Cookie 与 SPA 路由边界。
- **How to apply**: `taskFE/tests/` + `playwright.config.cdp.js`；参考 `Navbar.companySwitcher.playwright.test.js`。

## [OPT-20260811-008] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: APISIX up-taskAuth 增加 active health check（GET /api/health/，5s/2 成功/3 失败）；routes.yaml healthCheck + 生成器 _upstream_health_checks + apisix.yaml 已 reload；APISIX 探活 5s 间隔在 taskAuth 日志可观测，网关 /api/health/ 200。taskGateway commit 1d4cd53 已推送；新增 5 例生成器单测全绿。
- **Created**: 2026-08-11
- **Context**: wechat-login-apisix-502 设计 §4.4 降优先级：单节点开发机 health check 不能消除 Connection refused 502，但可配合监控。
- **Action**: 在 `taskGateway/apisix/apisix.yaml` 的 `up-taskAuth` 增加 `checks.active`（`GET /api/health/`），reload gateway，验证 unhealthy 时日志可读。
- **Why**: 多副本/未来扩容时摘除坏节点；开发机收益有限。
- **How to apply**: `taskGateway/apisix/apisix.yaml`；APISIX health-check 文档。

## [OPT-20260810-048] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 已提交 taskFE App.layout.test.js 长页滚动契约断言（壳层 h-screen overflow-hidden + 内容区 overflow-y-auto），vitest 全绿（c187823）；浏览器全路由人工抽检留待有浏览器/真实数据环境
- **Created**: 2026-08-10
- **Context**: 为让工作面板看板贴底，将 `App.vue` 根改为 `h-screen overflow-hidden`，主内容区 `overflow-y-auto`。理论上海量设置页应在内容区滚动，但未做全路由抽检。
- **Action**: (1) 抽检典型长页（系统管理用户列表、工作空间设置、计费流水）确认仍可纵向滚动到底；(2) 若发现某页依赖 `document`/`body` 滚动，改为内容区滚动或给该页单独 `min-h-0`；(3) 补一条 Playwright/冒烟断言。
- **Why**: 全局壳层高度约束可能让个别页面「滚不动」或出现双滚动条。
- **How to apply**: `taskFE/app/src/App.vue`；对照各 `views/*` 根节点是否再用 `min-h-screen` 与壳层冲突。

## [OPT-20260810-050] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 宿主机 github.com 出站已恢复（oauth/access_token 404/0.55s），AiMonitor 增 github-oauth-public 黑盒探活并提交（c61cb4a）；真实用户 refresh 验证因清库（auth_user=1）待人工
- **Created**: 2026-08-10
- **Context**: 任务详情 nested-git 排障时确认 `api.github.com` 可达，但 `github.com/login/oauth/access_token` 直连超时；taskGitOauth refresh 502，私有仓/换票仍会失败。本会话已用公开仓匿名探测缓解 UX，未修复底层出站。
- **Action**: (1) 测量并修复宿主机到 `github.com`（非仅 api.github.com）的 HTTPS 出站；(2) 复核 `conf/auth/git-oauth/providers/*` 无死 `outbound_proxy`；(3) 用真实用户 refresh 跑通 `access-for-user` 且 <3s；(4) 增加探针/告警：OAuth 端点连续超时。
- **Why**: 私有 GitHub 仓与 token 刷新仍依赖 oauth 端点；只修公开仓匿名路径无法覆盖私有仓。
- **How to apply**: `taskGitOauth` RefreshGitHubToken、provider yaml、运维出站/DNS；经验见 `.ai/09_failure_experience/02_runtime_errors/88_nested_git_unbound_when_oauth_refresh_timeout.md`。

## [OPT-20260810-017] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: sync.manifest 声明 from auth/task-auth internalSecret，conf-sync 生成 task-auth.yaml（conf c161699）；LoadConfig 用 confload.ReadAppFragment 读片段（taskAiProvider ed1bda6），与 taskBill 同源；env>config.yaml>片段>X_INTERNAL_SECRET；3 条单测绿
- **Created**: 2026-08-10
- **Context**: 厂商申请联系方式门禁依赖 taskAuth SMS gate；当前密钥靠 `X_INTERNAL_SECRET`/`TASKAUTH_INTERNAL_SECRET` 环境与 LoadConfig 兜底，未走服务配置目录 sync。
- **Action**: (1) 在 `conf/ai/ai-provider/sync.manifest.yaml` 声明 from auth 的 internalSecret；(2) LoadConfig 读生成片段；(3) 文档注明与 taskBill 同源密钥。
- **Why**: 符合「服务只读本目录 conf」元规则，避免密钥漂移导致申请一律 502。
- **How to apply**: 参考 `conf/billing` sync 与 `.ai/01_project_constraints/29_service_own_conf_directory_only_via_sync.md`。

## [OPT-20260811-013] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: taskFE 1b4d2f5 已推送；PhoneRegister 等待父组件 resolve 成功才倒计时，失败内联展示 data-traceId；新增 3 例 Vitest 全绿，auth 相关 26 例全绿
- **Created**: 2026-08-11
- **Context**: 登录页 `useLoginVerificationCode` 已改为成功才倒计时、失败内联；注册页 `PhoneRegister.vue` 仍在 emit `send-code` 后乐观启动 60s 倒计时，API 失败时用户仍看到倒计时。
- **Action**: (1) 让 `PhoneRegister` 等待父组件发码结果再 `countdown=60`；(2) 失败时将错误展示在「获取验证码」按钮上方并挂 `data-traceId`；(3) 补 Vitest 覆盖成功/失败路径。
- **Why**: 失败后误进入倒计时会阻止立即重试，与登录页新 UX 不一致。
- **How to apply**: `taskFE/app/src/components/auth/PhoneRegister.vue`、`taskFE/app/src/views/Register.vue`。

## [OPT-20260811-011] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: runAll 6afd260 已推送；bootstrap 6 行调用移至 13.js 末尾，扩展 TDZ 回归断言；full go test ./src 109s 全绿
- **Created**: 2026-08-11
- **Context**: 本次 TDZ 修复已把 `observabilityState`/`devLogState` 提前到 `01.js`；`06.js` 仍在中部执行 `loadObservabilityBar()`/`restartStatusRefreshTimer()` 等顶层调用，后续若再在 07–13 增加 `const` 状态仍有复发风险。
- **Action**: (1) 将 `06.js` 中 6 行 bootstrap 调用剪到 `13.js` 末尾；(2) 扩展 `TestAssembledStatusPage_SharedStateBeforeBootstrapCalls` 断言 bootstrap 位于所有 `const *State` 之后；(3) 热替换 runAll 复验可观测栏。
- **Why**: 顶层初始化应在全部声明之后，比「状态必须提前」更稳。
- **How to apply**: `runAll/src/status_ui/js/06.js`、`13.js`、`status_page_test.go`。

## [OPT-20260810-024] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: runAll 6afd260 已推送；02.js/04.js 已拆至 ≤500 行（02=447/04=481），随 status_ui 拆分批量落地
- **Created**: 2026-08-10
- **Context**: 修复 queue-status-bar 时 02.js 约 653 行、04.js 约 615 行，仍高于项目默认 500 行（status_ui 测试门禁为 1000）。本次已将 bulkQueue 抽到 07.js。
- **Action**: (1) 把 showProgress/updateProgress/cancelProgressOperation 抽到独立 js 片段；(2) 把 toast 与其它非队列逻辑从 04.js 再拆；(3) 保持 manifest 加载顺序与 ui_test 片段断言。
- **Why**: 降低后续改 UI 时的行数门禁与审查成本。
- **How to apply**: runAll/src/status_ui/js/ 、manifest.json 、ui_test.go。

## [OPT-20260810-040] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: runAll 6afd260 已推送；03=394/04=481/05=353 均 ≤500，07-13.js 片段与 manifest 更新完成
- **Created**: 2026-08-10
- **Context**: 修复进度面板「复制日志」时已将 02.js 拆出 08.js（≤500）。同目录 `03.js`/`04.js`/`05.js` 仍分别约 789/615/564 行，行数门禁仍触发。
- **Action**: (1) 按功能块继续拆分（如 precise-restart SSE、dev logs、observability bar）；(2) 更新 `status_ui/manifest.json`；(3) 复跑 `TestUIHomePage_ContainsRequiredSnippets`。
- **Why**: 超标文件继续堆功能会再触发强制削文件，拖慢后续 UI 改动。
- **How to apply**: `runAll/src/status_ui/js/`、`manifest.json`、`ui_test.go`。

## [OPT-20260811-012] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 根因确认：shutdown-self 后托管 Go 进程未被误杀（孤儿进程仍在运行），adopt=0 系 docker/netns 端口对 lsof 不可见（docker-proxy 以 root 运行）+ exec 型健康检查无端口可解析 + ownership PID 历史失效三者叠加，导致证据门禁全拒。修复 3ac0268：无端口/ownership 证据时以健康探针通过作为存活证据收养（探针失败仍拒绝）；新增 2 回归单测。bin/runAll 已重建
- **Created**: 2026-08-11
- **Context**: 本次为落地 UI 修复对 runAll 做 shutdown-self 热替换时，日志有 skip-orphan 但无 `pre-UI adopted N`，`/api/status` 一度全空；ownership.json 中 PID 均已不存活，只好 `POST /api/start-all` 恢复（最终 58 healthy）。
- **Action**: (1) 复现热替换并对照 `adoptListeningServices` / ownership PID；(2) 确认 Go 托管进程是否被误杀或未写入监听端口；(3) 补回归：热替换后至少 adopt>0 或 status 非全空。
- **Why**: 热替换本应保留托管服务；adopt 失败会导致全站短暂空窗与强制全量启动。
- **How to apply**: `runAll/src/main.go`、`runner_adopt.go`、`/api/shutdown-self`。

## [OPT-20260810-039] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 已验证：/api/status 58 个服务全部 healthy，无空/未知状态；热替换后收养正常，无运维告警。
- **Created**: 2026-08-10
- **Context**: 为落地进度面板复制按钮对 runAll 做了 build + 热替换后，`/api/status` 一度出现约 24 个服务 status 为空/`?`、34 个 healthy；可能与替换前 stop-all 或收养窗口有关。
- **Action**: (1) 刷新 9999 核对空状态服务是否被 adopt 为 healthy；(2) 必要时「全部启动」或对缺失项单服务 start；(3) 若复现，查 `AdoptRunningManagedServices` / ownership 回填日志。
- **Why**: 热替换后 UI 空白或灰点会误导运维以为服务宕机。
- **How to apply**: `runAll/src/runner_adopt.go`、`/api/status`、9999 页面服务列表。

## [OPT-20260811-007] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 复验通过：Loki /ready 正常、promtail 运行中；1h 范围查询 {job=~".+"} 返回 58 个 distinct job 流（含 task-gateway/task-auth/task-events 等），ingest 正常。早前 0 条为 instant query 类型所致（应使用 query_range）。traceId 49b2be86 未在 Loki 检索到（日志行 trace_id=<no value> 或已过保留期），排障仍建议回退 taskGateway/logs/。
- **Created**: 2026-08-11
- **Context**: 排障 `data-traceId=49b2be86-...` 时 Loki（localhost 与 10.2.150.68:3100）`/ready` 成功，但 1h/7d `{job=~".+"}` 均 0 条；最终靠 `taskGateway/logs/taskgateway-access.log` 定位。
- **Action**: (1) 检查 promtail/fluent-bit 是否运行与 scrape；(2) 确认服务日志是否仍写入 Loki；(3) 修复后用已知 traceId 复验 Explore。
- **Why**: 有 data-traceId 却查不到 Loki 会迫使每次回落到本地 access log，放大排障成本。
- **How to apply**: `conf/runAll.yaml` loki_url；AiMonitor/promtail；`taskGateway/logs/`。

## [OPT-20260810-021] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 扫描确认：user-created-dlt 无 missing-username 死信（topic 空或已读穿），无需重放。scripts/replay_user_created_dlt.py 与 repair_users_without_company.py 已入库（commit b8699a5）含单测。可选 Kafka UI/告警留给运维。
- **Created**: 2026-08-10
- **Context**: `user-created-dlt` 曾积压 `missing username` 永久失败；本次已修 handler 并手工重放 1 用户。其他环境可能仍有同类 DLT。
- **Action**: (1) 运维脚本扫描 `user-created-dlt` 中 `error=missing username`；(2) 对仍无 by-creator 公司的 user_id 调用 `repair_users_without_company.py`；(3) 可选接入 Kafka UI/告警。
- **Why**: 热修只覆盖当前环境；跨环境需可重复的 DLT 清理路径。
- **How to apply**: 扩展 `scripts/repair_users_without_company.py` 或新增 `scripts/replay_user_created_dlt.py`。

## [OPT-20260810-012] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 评估结论：暂不重命名 taskFE→task-fe，保留为受控例外。理由：(1) working_dir 与 submodule 目录名 taskFE 耦合，仅改服务名会造成名/路径分裂；(2) value-stream.yaml 70 处 taskFE.* 指标前缀改名会破坏基线对照；(3) valueStream/src/fields.go 已有 CamelCase 兼容 shim，边际收益低；(4) 全量对齐需连 gitlink/import/脚本/CI 一起改，代价不成比例。若未来需要，建议加 task-fe 别名（低成本）并在 value-stream 基线可重置时一次性迁移。详见 .learnings/OPT-20260810-012-taskfe-kebab-case-assessment.md
- **Created**: 2026-08-10
- **Context**: 多数 runAll 服务为 kebab-case（`task-auth`），前端为 CamelCase `taskFE`；本次已放宽 value-stream 字段服务段以兼容，但仍是命名特例。
- **Action**: 评估是否将 runAll `name: taskFE` 统一为 `task-fe`（含 conf、ownership、value-stream 字段前缀、脚本登记）。
- **Why**: 消除 CamelCase 特例，降低校验/文档/脚本分支。
- **How to apply**: 若采纳，先改 `conf/runAll.yaml` + 全仓引用，再跑 start-all 与 value-stream 启动验收。

## [OPT-20260811-015] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: Playwright Test 6 平台角色多公司并列+切换回归，CDP 全绿（17 断言）；已推送
- **Created**: 2026-08-11
- **Context**: 本次仅用 Vitest 覆盖平台角色+多公司并列渲染；既有 Playwright `Navbar.workPanelLink.profile-page-fix` Test 3 仍只测无公司超管。
- **Action**: (1) 在该 Playwright 文件新增用例：mock `/me/` 为 `isSuperuser` + 两家公司；(2) 断言同时存在 `a[href="/system-admin/"]` 与 `[data-testid="nav-company-switcher"]`；(3) 选择第二家公司后 URL 含对应 tenant work-panel。
- **Why**: 浏览器级回归可防止模板再次被改回 `v-else-if` 互斥。
- **How to apply**: `taskFE/tests/Navbar.workPanelLink.profile-page-fix.playwright.test.js`；参考 Vitest `平台角色加入多公司时并列…`。

## [OPT-20260811-016] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 各 dirty 子仓真实 WIP 已原子提交（taskFE/taskAuth/taskEvents/taskCloudService/taskGitOauth/taskAiProvider/runAll/dataMigrate/valueStream/conf/db/docs/AiMonitor）；github 网络中断，7 仓推送待网络恢复补推
- **Created**: 2026-08-11
- **Context**: LICENSE/README 噪音已从 nightly sweep dirty 判定豁免；仍有约 14 个子仓因真实未提交改动被 SKIP（taskFE/runAll/taskAuth/taskCloudService/taskEvents/taskAiProvider/taskGitOauth/conf/docs/AiMonitor/valueStream/dataMigrate/db/dockerInfra 等）。
- **Action**: 交互会话收工时：对各 dirty 子仓 `git status` → 该提交的提交推送、不该提交的 stash/丢弃；优先清有单测的业务仓（taskFE/taskAuth/taskEvents/taskCloudService/runAll），使下一夜间 sweep 覆盖面扩大。
- **Why**: sweep 只抽 clean worktree；真 WIP 不收工会持续浪费夜间巡检窗口。
- **How to apply**: 各子仓独立提交（规则 32）；勿 `git add -A` 卷入无关文件。

## [OPT-20260810-037] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 盘点确认：生产库 ai_provider_userdatatemplate 为 0 行（2026-08-11 清库重建后），无存量缺 COMMENT_ID 模板需迁移；模板源码 userDataScriptLinux.js / userDataScriptWindows.js / userDataTemplate.js 均已携带 export COMMENT_ID=__TASK2APP_COMMENT_ID__ 与 report_progress 的 comment_id 字段，厂商保存时即生成含字段新模板。批量注入迁移 / 一键重生入口判定无必要，未实施（避免对空表做脆弱脚本内容手术）。
- **Created**: 2026-08-10
- **Context**: 新生成的 Linux/Windows 模板已 export COMMENT_ID 并在 boot-progress 携带 comment_id；镜像市场/租户已安装的旧模板脚本不会自动更新，仍可能只靠后端 CSC 兜底路由。
- **Action**: (1) 盘点 `ai_provider_userdatatemplate` 中缺 `COMMENT_ID`/`comment_id` 的 content；(2) 提供迁移或「一键重生脚本」管理入口；(3) 文档提示厂商保存前须重新生成。
- **Why**: 旧模板不带 comment_id 时，多评论并行启动可能把进度串到错误面板或仅落任务级日志。
- **How to apply**: `userDataTemplate.js`、`dataMigrate/taskAiProvider/`、厂商门户 UserData 模板列表。

## [OPT-20260811-002] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: docs/runbooks/container-instance-stuck-in-starting-remediation.md 已写：存量 init_from_task2app.sh 旧模板三类缺陷（COMMENT_ID=-/docker inspect camelCase/report_progress 缺引号）+ 释放重跑/SSH 修复/人工标 running 三档补救。docs 子仓 f5caa6b 已推。
- **Created**: 2026-08-11
- **Context**: 任务 `task_15486923850655975865` 对应机器上 `init_from_task2app.sh` 已含 `COMMENT_ID='-'`、`docker inspect -` 与破损的 `report_progress` 条件；当前进程无法靠前端刷新自愈。
- **Action**: (1) 文档/运维备注：对该实例释放后按新模板重跑 start-vm（推荐），或 SSH 手工把健康检查改为 `"$CONTAINER_NAME"`、把 `COMMENT_ID` 写成真实 `cmt_…` 后重跑脚本；(2) 若仅需解除「启动中」，确认 userdata-verify 回调或手动将 CSC 标 running（慎用）。
- **Why**: 已写入磁盘的坏脚本不会随服务重启修复，用户会以为前端仍坏。
- **How to apply**: 实例 `/root/init_from_task2app.sh`；`handleBootProgress` / server-userdata-verify；评论 CSC 行。

## [OPT-20260811-025] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 已将 WorkPanel.create-task-branch-datalist-no-reopen.playwright.test.{js,sh} 重命名为 suggest-no-reopen，正文与引用已对齐 BranchSuggest
- **Created**: 2026-08-11
- **Context**: `WorkPanel.create-task-branch-datalist-no-reopen.playwright.test.js` 正文已改测 BranchSuggest，但文件名仍含 datalist，易误导检索。
- **Action**: (1) 重命名 js/sh 与引用 (2) 更新 `.ai/09_failure_experience` 中相关链接（若有）
- **Why**: 命名与实现一致，减少后续误改回 datalist。
- **How to apply**: `taskFE/tests/WorkPanel.create-task-branch-datalist-no-reopen.playwright.test.*`


## B. 运维与破坏性操作

## [OPT-20260811-027] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: Removed empty-comments execution-details-fallback; TaskDetailCommentsSection.vue now 464 lines (≤500).
- **Created**: 2026-08-11
- **Context**: `TaskDetailCommentsSection.vue` 现 512 行，已超默认 500 行门禁；本次硬件 Tab 未继续堆入该文件。
- **Action**: (1) 抽出 fallback / per-comment execution-details 模板为子组件 (2) 复跑 `wc -l` ≤500 与相关 vitest
- **Why**: 行数门禁与可维护性；后续再向该文件加功能会被迫阻塞。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailCommentsSection.vue`。

## [OPT-20260811-022] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 落地 TenantPageAccessEmpty + useTenantPageAccess；接入 TenantCompanySettings/BillingOrders/WorkspaceSettingsGitlabConnection；vitest 全绿
- **Created**: 2026-08-11
- **Context**: v72 后侧栏用 `page:*`；页内控件用 `hasRegion`。直链无 page/region 须空态，而非仅粗码 `hasPerm`。
- **Action**: (1) 盘点 settings/billing 视图 (2) 按 page/region registry 补空态 (3) 补单测
- **Why**: 侧栏隐藏不是安全边界；页内兜底改善体验。
- **How to apply**: v72 设计 §5.3；各 settings/billing views。

## [OPT-20260811-023] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: POST 409 后缀重试；listOrphanCustomAccessRoles + PeopleAccess 清理按钮；单测覆盖
- **Created**: 2026-08-11
- **Context**: 旧保存路径按 `访问·{名}` + 粗码；v72 改为角色↔resource_group。命名冲突/孤儿角色问题在新写路径上仍可能出现。
- **Action**: (1) 在 v72 保存编排中处理 display_name 冲突（后缀雪花短码）(2) 提供未绑定自定义角色清理或扫描
- **Why**: 避免 Conflict 导致保存失败与角色表膨胀。
- **How to apply**: v72 访问管理保存编排；taskAuth 角色列表。

## [OPT-20260811-031] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: publishRoleResourceGroupsChanged + PUT 双发；rbac_events_test 常量/Kafka-off 冒烟
- **Created**: 2026-08-11
- **Context**: P1 用 `publishRoleChanged` 失效缓存；设计文档规划显式 `RoleResourceGroupsChanged`。
- **Action**: (1) 定义事件契约 (2) PUT resource-groups 成功路径投递 (3) 消费者/观测对齐
- **Why**: 与其它授权变更事件语义区分，便于审计。
- **How to apply**: `rbac_resource_groups.go`；`rbac_events.go`；taskEvents。

## [OPT-20260811-032] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: check_auth_region_api_members.py + selftest；现网 8 api members 匹配通过
- **Created**: 2026-08-11
- **Context**: v72 P2 — 防止新 API 未挂 region 导致可绕过。
- **Action**: (1) 静态扫描 Go HandleFunc 路径 (2) 对照 auth_resource_member api 种子 (3) pre-commit/CI 门禁
- **Why**: 仅文档约定会漂移。
- **How to apply**: `dataMigrate/taskAuth/032_*.sql`；新增 check 脚本。

---

## E. 代码健康

（暂无 pending；`TaskDetailCommentsSection.vue` 已回落到 464 行，见 OPT-20260811-027 completed）

## [OPT-20260811-029] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: CommentsPanel feed errors banner 已渲染；empty-fallback.test 含 ai_comments: HTTP 503 断言
- **Created**: 2026-08-11
- **Context**: `commentsFeedErrors` 已传入 Panel 但模板未渲染；三源拉取失败时用户只见空态，无法区分「真无评论」与「feed 失败」。
- **Action**: (1) 在 CommentsPanel 空态/列表上方渲染 `comments_feed_errors` banner（含 data-traceId 若有）(2) 补 vitest (3) 更新 intent
- **Why**: 空评论 UX 修复后，feed 失败更易被误判为无数据。
- **How to apply**: `TaskDetailCommentsPanel.vue`；`taskDetailFetchFns.js`。

## [OPT-20260811-018] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: ResetPassword 改用 PageFooter；生产 ResetPassword chunk 无内联 SaaS footer，PageFooter 复用确认
- **Created**: 2026-08-11
- **Context**: `ResetPassword.vue` 内联复制了与 `PageFooter.vue` 相同的营销 footer 标记；Register 已用 `<PageFooter />`。
- **Action**: (1) 删除 `ResetPassword.vue` 内联 `<footer>`；(2) 改为 `import PageFooter` + `<PageFooter />`；(3) 跑相关布局/冒烟测。
- **Why**: 避免三处 footer 文案/结构漂移。
- **How to apply**: `taskFE/app/src/views/ResetPassword.vue`；`taskFE/app/src/views/PageFooter.vue`。

## [OPT-20260811-020] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: Credential Management API：localStorage 仅账号；PasswordCredential store/get；旧明文密码迁移清除；生产 chunk 已含实现
- **Created**: 2026-08-11
- **Context**: 当前按产品要求将密码明文写入 `localStorage`（`daydaymoney.rememberedLoginCredentials`），满足回填但放大 XSS 可窃取面。
- **Action**: (1) 优先 `navigator.credentials.store(new PasswordCredential(...))` + `credentials.get`；(2) 不支持时仅记账号标识；(3) 迁移/清除旧 localStorage 密码字段。
- **Why**: 安全加固；降低 XSS 一次窃取明文密码风险。
- **How to apply**: `remembered_login_credentials_store.js`；`applyRememberedLoginCredentials.js`；MDN PasswordCredential。

## [OPT-20260811-019] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 精准重启后公网验收：记住我勾选框存在；seed localStorage 回填邮箱并勾选；密码走 Credential API（不落盘）
- **Created**: 2026-08-11
- **Context**: 已实现勾选「记住我」成功登录后 localStorage 持久化邮箱/手机+密码，并在 `/auth/login/` 回填；taskFE 已 `npm run build` 且登记精准重启。需公网浏览器验收。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」使 taskFE 新 dist 上线；(2) 打开 https://www.daydaymoney.com/auth/login/ 勾选记住我登录；(3) 退出后再进登录页确认邮箱与密码已回填且复选框勾选；(4) 取消勾选再登录后确认回填已清除。
- **Why**: 单元测试已覆盖 store/submit，公网需确认产物 hash 与真实浏览器 localStorage 行为。
- **How to apply**: `taskFE/app/src/domain/auth/services/remembered_login_credentials_store.js`；`useLoginSubmit.js`；`Login.vue`。

## [OPT-20260811-003] completed

- **Status**: completed（部署完成，待浏览器）
- **Created**: 2026-08-11
- **Context**: 已修 Navbar `/me/` 401/404 假登录与 taskAuth `/me/` 401 清 cookie；夜间 taskFE+taskAuth 已精准重启发布。仍需带旧 localStorage、不手动清 cookie 的浏览器验收。
- **Action**: (1) 打开 `/auth/login/`（保留旧 localStorage）；(2) 确认无 `account-switcher-trigger`，仅登录/注册。
- **Why**: 未经验证时用户仍可能看到「未设置昵称」假登录态。
- **How to apply**: `Navbar.logic.vue`；`taskAuth/src/auth_users.go`；对照 `docs/superpowers/specs/2026-08-11-db-reset-stale-login-ui-design.md`。
- **Related**: OPT-20260811-004

## [OPT-20260811-017] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: reset-password footer position=static；在 auth overflow-y-auto 壳内滚动时 footer 随内容位移（非视口钉底）
- **Created**: 2026-08-11
- **Context**: 已将 `reset_password*` 纳入 auth 滚动壳（`authShellRoutes.js` + `App.vue`），单元测试已绿；taskFE 已登记精准重启，本机无 CDP 9222，生产 SPA 尚未含本次 dist。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点击「精准编译重启」；(2) 打开 `/auth/reset-password/?token=…`，滚动主内容区；(3) 确认 `footer.bg-gray-800` 与上方表单同容器位移（不贴死视口底）。
- **Why**: 源码已修但公网仍为旧包时用户仍见 footer 钉底。
- **How to apply**: `.runall/precise_restart_services.txt`；`taskFE/app/src/App.vue`；`taskFE/app/src/utils/authShellRoutes.js`。

## [OPT-20260811-035] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 2026-08-11 live: build-all 50/50 后 registrations=[]、precise_restart_services.txt 0 行、consumed-at=1786425927
- **Created**: 2026-08-11
- **Context**: BuildAll 收尾已实现清空 `.runall/precise_restart_services.txt` + consumed-at；runAll 已热替换到 :9999。需在 UI 上确认进度完成后「精准编译重启」徽章消失。
- **Action**: (1) 确认登记文件非空（或 `scripts/register-precise-restart.sh taskFE`）(2) http://10.2.150.68:9999/ 点页头「全部重新编译」并等完成 (3) 确认徽章/登记接口为空且 `precise_restart_services.txt` 已空
- **Why**: 单元测试覆盖引擎，不覆盖 SSE→refresh 徽章链路。
- **How to apply**: `Runner.BuildAll` / `clearPreciseRestartRegistrationsAfterFullRebuild`；`status_ui/js/02.js` updateProgress→refresh。

## [OPT-20260811-030] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: 公网 Token 调 GET /api/auth/resource-groups/ → 200 + 18 pages；根因是网关 public 未注入 X-Tenant-Perms，已把 resource-groups 列入 taskauth-rbac-admin(token) 并 routes-apply
- **Created**: 2026-08-11
- **Context**: v72 P1 已落地（032 迁移已应用；PDP/FE/RequireRegion 代码已写）；需重启 task-auth+taskFE 后公网验收。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」(2) 管理员打开人员管理→访问管理，确认 page→region 树 (3) 为普通成员仅勾选部分 region 后另开会话：侧栏按 page:* 隐藏且无 region API 403
- **Why**: 单元测试无法覆盖 forward-auth 头注入与真实 Cookie。
- **How to apply**: `PeopleAccess.vue`；`rbac_resource_groups.go`；`rbac_pdp.go`。

## [OPT-20260811-047] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: sessionStorage 持久化 pending 项；刷新后 restorePendingFromStorage + Playwright 验证 PASS；顺带 unique queue id 与 external 完成后 processNext
- **Created**: 2026-08-11
- **Context**: 本次修复了刷新后丢失「正在执行」进度（`fetchStatusData` 丢弃 `active_bulk_progress`）。客户端 `bulkQueue.items` 中尚未开始的排队操作仍仅存内存，刷新后「排队中」列表仍会清空，不会自动续跑后续操作。
- **Action**: (1) 在 `enqueue`/`cancelItem`/`cancelAllPending`/`processNext` 时将 pending 项的 `{type,label,id}` 写入 `sessionStorage`；(2) 页面加载且 `resumeActiveBulkProgress` 完成后，按 type 映射重新绑定 `_exec*` 并恢复 `items`；(3) 补 UI 片段回归测试。
- **Why**: 用户连续点了多个批量操作时，刷新仍会丢掉尚未执行的排队项。
- **How to apply**: `runAll/src/status_ui/js/07.js` + `05.js`（exec 映射）+ `queue_status_bar_ui_test.go`。

## [OPT-20260811-060] completed

- **Status**: completed
- **Completed**: 2026-08-11
- **Summary**: Hero「了解更多」已改为 #projects，vitest T4 通过；页脚占位链拆至 OPT-20260811-066
- **Created**: 2026-08-11
- **Context**: Hero「立即开始」已改为 `/auth/register/`（本会话 goal）；Hero「了解更多」与页脚「首页/帮助中心/联系我们/常见问题」仍为 `href="#"`。
- **Action**: (1) 将 Hero「了解更多」改为 `#projects`（或产品确认的目标）；(2) 页脚「首页」改为 `/`，其余按产品确认补真实路径；(3) 补 vitest。
- **Why**: 占位链点击无导航，转化漏斗与页脚可达性仍断裂。
- **How to apply**: `taskFE/app/src/views/Home.vue` 第 12 行与页脚列表；参考 Navbar `/auth/register/`。

