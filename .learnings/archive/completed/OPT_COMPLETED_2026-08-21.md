# Completed OPT Archive — 2026-08-21

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 35 条。
> 归档执行时间：2026-08-25T22:13:52+08:00

## [OPT-20260820-038] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: 定位被推荐人 `878234267996418048`（手机 13900001111，date_joined 2026-08-20）；taskBill sync-edge 写入 `billing_referral_edge`（渠道 DR2AKvP9J9、eligible=true）；stats 默认渠道 referral_count=1，消费回填 0。根因另修 phone_register 已有号绑边 + taskReferral 加载 bill secret。
- **Created**: 2026-08-20
- **Context**: 生产 `referral_share_code` 中 `DR2AKvP9J9` 属用户 `877397583960502272`，但 `billing_referral_edge` 与 `billing_referral_commission_accrual` 均为 0 行。注册期从未存 accessCode，无法从业务表重建历史边。
- **Action**: (1) 在 nginx/Loki 检索 `/auth/register/?accessCode=DR2AKvP9J9` 及后续注册成功的 user_id (2) 对每个被推荐人 POST bind-from-code（会 sync-edge 并回填消费）(3) 硬刷新 `/profile/referral/` 核对人数与月度收益
- **Why**: 修复只覆盖新注册；用户声称「已经推荐且被推荐人消费了」的存量关系不会自动出现
- **How to apply**: `taskfe-nginx` access.log；Loki `{job=~".+"}`；内部 `POST /api/internal/referral/bind-from-code/`；推荐人码 `DR2AKvP9J9`

## [OPT-20260820-049] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskProjectService 4300e11 已推送：logGitLabAPIFailure 改 tracelog 结构化（op/status/repo/msg/trace_id），4 条调用链透传 outbound trace headers；回归测 3 例（X-Trace-Id/traceparent/无 trace）。
- **Created**: 2026-08-20
- **Context**: 创建任务弹窗拉分支失败（traceId `16911ac8-51fc-4b1f-ba60-e3e042c518f0`）时，`task-project-service.log` 的 `gitlab-api op=list-branches status=400` 行没有 `trace_id`，只有随后的 `http_request` 行带该字段。Loki `{job=~".+"} | json | trace_id=` 因此搜不到 400 根因行。
- **Action**: (1) 把 `taskProjectService/src/gitlab_api_error.go` 的 `logGitLabAPIFailure` 从 `log.Printf` 改为 `tracelog`/`slog` 结构化字段（`op`/`status`/`repo`/`msg`/`trace_id`）(2) 从请求 ctx 或 outbound trace header 传入 trace_id (3) 补单测断言日志 JSON 含 `trace_id`
- **Why**: 排障只能靠同秒 http_request 手工对齐；Loki 按 trace 拉全链路会漏掉 GitLab 400 正文。
- **How to apply**: `logGitLabAPIFailure`；`fetchGitLabBranches` 调用处已有 `outboundTrace` 可下传。

## [OPT-20260820-020] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: conf 40f2a89 已推送：value-stream.yaml 50 处 api_githubappusercredential/api_gitoauthappusercredential → git_oauth_appusercredential，3 处 test_file 指向 taskGitOauth Go 测，ai.md 去 Django 目录表述；provider YAML 复核无重复文档。
- **Created**: 2026-08-20
- **Context**: `/1-brainstorming` as-is 盘点发现：`conf/auth/git-oauth/providers/http-gitlab-tencent-sh-1.yaml` 整文件重复粘贴两段相同 YAML；`conf/value-stream.yaml` 仍用已删 Python 测路径 `gitOauth/api/tests.py` 与旧表名 `api_githubappusercredential`（现表 `git_oauth_appusercredential`）。
- **Action**: (1) 删除 tencent-sh-1 provider YAML 的第二段重复文档，保留单份并 `python3 -c yaml.safe_load` 验收 (2) 将 `gitoauth-binding-state-persistence` 等步骤的 `fields[].name` 改为 `task-git-oauth.git_oauth_appusercredential.*`（三段名）(3) 把仍指向 `gitOauth/api/tests.py` 的 `test_file` 改到对应 `taskGitOauth/**/*_test.go` (4) 更新 `conf/auth/git-oauth/ai.md` 中已不存在的 Django `git-oauth-providers` 目录表述
- **Why**: 重复 YAML 依赖「后一段覆盖」碰巧同内容才不爆；价值流字段/测试路径漂移会让后续 /4-value-stream 对照错表、跑到已删文件。
- **How to apply**: `conf/auth/git-oauth/providers/http-gitlab-tencent-sh-1.yaml`；`conf/value-stream.yaml`；`conf/auth/git-oauth/ai.md`；对照 `dataMigrate/taskGitOauth/001_schema.sql` 与 `docs/superpowers/specs/2026-08-20-git-oauth-as-is-system-brainstorm-design.md` §4.2 / §9

## [OPT-20260820-031] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskAiProvider 1bc06f3 已推送：scanContainerImageRows 补 updated_at（formatCatalogTS），ListContainerImages/Admin 两查询列同步；回归测 2 例非空日期。
- **Created**: 2026-08-20
- **Context**: 主站镜像市场公开 catalog 已返回 `updated_at`。厂商门户 `ListContainerImages` / `scanContainerImageRows` 仍不选不返回该字段，门户卡片若要展示更新时间会对空。
- **Action**: (1) `scanContainerImageRows` SELECT/Scan 增加 `updated_at` (2) JSON 用 `formatCatalogTS` (3) 补门户列表测断言非空日期
- **Why**: 同一镜像实体两套 payload 字段不一致，门户与主站展示会漂移。
- **How to apply**: `taskAiProvider/infrastructure/store_marketplace.go` `scanContainerImageRows`；`store_marketplace` 相关测

## [OPT-20260820-015] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskProjectService d675c00 已推送：/api/projects/ 分发层统一 requireTenantMember（internal 旁路/TaskTenantURL 未配置跳过），回归测 成员200/非成员403/internal放行。
- **Created**: 2026-08-20
- **Context**: OPT-20260727 曾在 taskProjectService 实现 `requireTenantMember`，并声称插在 `/api/tenant/` 分发前。现网路由已迁到 `/api/projects/tenant_id/{tid}/...`，`requireTenantMember` 全仓无调用方（taskTaskService 仍在调）。项目 handlers 只按 URL 租户读表，不校验登录用户是否为该公司成员。
- **Action**: (1) 在 `handleProjectsRoute` / `/api/projects/` 闭包 dispatch 前调用 `requireTenantMember`（internal 旁路保留）(2) 补非成员 403、成员 200、internal 放行测 (3) 确认 convention path 的 tenant_id 与 membership 用同一 id
- **Why**: 前端守卫可被 accessCode/直链绕过；无后端成员门禁时任意已登录用户（含超管）只要知道 tenantId+projId 就能读项目数据。
- **How to apply**: `taskProjectService/src/auth.go` `requireTenantMember`；`taskProjectService/src/main.go` `/api/projects/`；对照 `taskTaskService/src/main.go` 已有插入点

## [OPT-20260820-018] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskBill fe5efac 已推送：markOrderRefunded 后 paid→refunded/pending→cancelled，handleOrderCancel pending→cancelled；抽 closePaymentPendingForOrder；回归测 2 例。
- **Created**: 2026-08-20
- **Context**: 注销阻断已改为只把 `status='pending'` 且关联订单仍为 pending 的支付当作待处理。微信二维码行在订单已退款/已付后仍可能留在 `billing_payment_pending`（从未把该行标为 paid/cancelled/refunded）。
- **Action**: (1) 在 `markOrderRefunded` / 取消订单 / 标记已支付路径同步更新同 `order_id` 的 `billing_payment_pending`（paid→refunded，pending→cancelled）(2) 补 MySQL 回归测，断言退款后该行不再为 pending
- **Why**: 过滤层已挡住误导链接，但残留 pending 行会继续占表、干扰对账与人工排查。
- **How to apply**: `taskBill/src/refund_order.go` `markOrderRefunded`；`taskBill/src/wechat_pay.go` `markWechatPendingPaid`；`taskBill/src/handlers_orders_pay.go` 取消订单

## [OPT-20260820-033] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskGateway db18a46 已推送：routes-to-apisix --check 接入 pre-commit（routes.yaml/生成脚本变更时强制同步）；顺带 routes-apply 修复 order-number-parse 漂移。
- **Created**: 2026-08-20
- **Context**: `routes.yaml` 已登记 `POST /api/system-admin/order-number/parse/`，但 `apisix.yaml` 未 routes-apply，公网落入 spa-catch-all 返回 502 HTML，管理端只显示「解析订单号失败」。本会话已补产物与回归测；CI 仍不跑 `--check`，同类漂移会再漏。
- **Action**: (1) 在 taskGateway pre-commit / 根仓 CI 增加 `python3 taskGateway/scripts/routes-to-apisix.py --check`（2）失败即阻断，提示 `bash taskGateway/run.sh routes-apply`
- **Why**: 只靠人工 routes-apply 无法拦住「routes.yaml 已改、运行中 APISIX 未同步」。
- **How to apply**: `taskGateway/scripts/routes-to-apisix.py --check`；钩子模板 `db/scripts/hooks/templates/` 或 `taskGateway/.githooks/`

## [OPT-20260820-047] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskTaskService eefac41 已推送：task_handlers.go 883→182 行，拆 update/switch/associate/renew、comments、search/list 三文件均 ≤500 行；整包测试全绿。
- **Created**: 2026-08-20
- **Context**: 本次修终态进度跳过 auto_run 门禁时改了 `taskTaskService/src/task_handlers.go`，该文件仍约 880 行，超过行数门禁。
- **Action**: (1) 把 update/switch/associate 等 handler 抽到独立文件 (2) 保持路由入口不变 (3) 复跑 `go test ./src -count=1`
- **Why**: 继续往该文件堆逻辑会再次触发自动削文件，且难审。
- **How to apply**: `taskTaskService/src/task_handlers.go`；参考 runAll `ui.go` 按域拆分。

## [OPT-20260820-039] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskChromePlugin 2c44898 已推送：端点映射合并丢弃已下线 /api/tenant/{companyId}/(workspaces|projects|workspace|installed-images|daydaymoney|manage-deliverable) 键；回归测 4 例。
- **Created**: 2026-08-20
- **Context**: 浮窗工作空间列表已改默认约定路径。`API.init` 仍会把 `Storage.endpointMapping` 覆盖到默认端点上；若用户曾保存整份旧 `/api/tenant/{companyId}/workspaces/` 映射，升级后仍会 404。
- **Action**: (1) 合并 mapping 时丢弃匹配已下线 `/api/tenant/{companyId}/(workspaces|projects|workspace/|installed-images|daydaymoney|manage-deliverable)` 的键 (2) 单测：旧 mapping 不覆盖新默认 (3) 可选：保存时不再写入与 DEFAULT_ENDPOINTS 相同的键
- **Why**: 少数已自定义端点的安装升级后工作空间下拉仍会「加载失败:not found」。
- **How to apply**: `taskChromePlugin/lib/api.js` 的 `init` / `setEndpointMapping`；测例放 `test/api-endpoints.test.js`。

## [OPT-20260820-008] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: runAll 1ac4893 已推送：shouldSkipOrphanCleanup（GracefulShutdownSelf 或无旧 UI）时跳过 orphan SIGKILL 走 adopt；回归测 3 例；ai.md 增补 9999 空窗拉起姿势。
- **Created**: 2026-08-20
- **Context**: 修复 Status UI 进度条时发现 :9999 已无监听，而 :8002/:8018 等托管端口仍在。`bootstrapUIMode` 在 `HadPrevious=false` 时 `GracefulShutdownSelf` 亦为 false，`cleanupOrphanManagedServices` 会等 15s 后对所有仍监听的业务端口 `SIGKILL`。本次用一次性 shutdown-self 桩保住栈，不能每次靠手工桩。
- **Action**: (1) 无 UI listener 时若 `ownership.json` 仍有存活 PID 或业务端口可收养，跳过 orphan 清理并走 adopt (2) 补回归测：9999 空 + 假监听端口不得被 Kill (3) 在 `runAll/ai.md` 写明「9999 空窗拉起」的正确姿势
- **Why**: 误杀会停掉整栈；`no_restart_runall` 正是怕这条路径，导致 Status UI 长期起不来。
- **How to apply**: `runAll/src/main.go` `cleanupOrphanManagedServices` / `killPreviousRunAllProcess`；`ownership.json`；对照 OPT-20260820-006 的 shutdown-self 语义

## [OPT-20260820-012] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskGitOauth 2fc17eb 已推送：writeTokenUseAudit 增 traceID 参数，skip/insert 失败日志带 trace_id；handleAccessForUser 不再丢弃审计错误；回归测 2 例。
- **Created**: 2026-08-20
- **Context**: `site` 改为域名+端口后，`writeTokenUseAudit` 解析不到 website 会返回 error；token-use-report 变成 500，但 access-for-user 仍 `_ = writeTokenUseAudit`，只打无 trace 的 warn。
- **Action**: (1) `handleAccessForUser` 把 `requestTraceID(r)` 写入 skip/insert 失败日志 (2) 补回归测断言日志或可观测字段含 trace_id
- **Why**: 换票成功但审计被跳过时，Loki 无法用前端 data-traceId 对上。
- **How to apply**: `taskGitOauth/src/internal_access_for_user.go`；`src/access_audit_write.go`

## [OPT-20260820-032] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: ImageMarket.vendorStatus.test.js mock 补 response.text，taskFE f6fee8c 已推送
- **Created**: 2026-08-20
- **Context**: `ImageMarket.vendorStatus.test.js` 的 apiFetch mock 只有 `json()`，catalog/installed 走 `readJsonPreservingSnowflakeIds` → `response.text`，测例绿但 stderr 刷 `text is not a function`。
- **Action**: mock 改为同时提供 `json` 与 `text: async () => JSON.stringify(body)`，与 `ImageMarket.card-fields.test.js` 对齐。
- **Why**: 噪音掩盖真实失败，后续改 catalog 加载时容易误判。
- **How to apply**: `taskFE/app/src/views/ImageMarket.vendorStatus.test.js`

## [OPT-20260820-028] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: handleGitCloneProgress 高水位按 comment_id+repo_url 只升不降，taskCloudService 4804642 已推送
- **Created**: 2026-08-20
- **Context**: 评论克隆条卡在 9% 时，容器 UI 已 100%。已在 FE 与容器 POST 串行修复乱序覆盖；Cloud `handleGitCloneProgress` 仍原样转发，旧容器镜像仍可能把迟到 9% 打进 SSE。
- **Action**: (1) 按 comment_id+repo_url 记住最近成功转发的 progress (2) 非重试/失败/重新克隆时忽略更低百分比 (3) 补 Go 测：100 后再 9 不转发
- **Why**: 未重建的任务容器会继续按旧 overall=recv 上报；Cloud 单调可在不换容器镜像时挡住回退。
- **How to apply**: `taskCloudService/src/container_inbound_actions.go` `handleGitCloneProgress`；内存 map 即可，进程重启丢失可接受

## [OPT-20260821-001] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: fetchAIPublicRuntimeUserdataBody 502/503/504/超时 2 次短退避重试，taskCloudService 620268a 已推送
- **Created**: 2026-08-21
- **Context**: 任务详情 `task_878300073396563968` 评论级 start-vm 在 binding 行尚未 INSERT 时因镜像市场 502 失败。本会话已修失败收口与误导文案，502 本身仍是上游瞬时错误。
- **Action**: (1) 在 `finalizeStartVmInGo` / 镜像市场 HTTP 调用对 502/503/超时做 1～2 次短退避重试 (2) 重试耗尽仍走现有 error SSE + failed binding (3) 补单测：首轮 502 次轮 200 则成功
- **Why**: 用户会把瞬时 502 当成配置永久失败；有限重试可消掉多数抖动而不掩盖持续故障。
- **How to apply**: `taskCloudService/src` 镜像解析/市场客户端；禁止无限重试；保持 `startVmErrorStatusData` 带 tenant/workspace。

## [OPT-20260820-014] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: isGuestTenantRoute accessCode 收窄为 task-detail，taskFE b845ea2 已推送
- **Created**: 2026-08-20
- **Context**: 排查 `author@example.com`（`bootstrap-admin`，0 条 `tenant_company_member`）仍能打开 `https://www.daydaymoney.com/tenant/877397588196749312/projects/proj_-2304947540687519745/?accessCode=DR2AKvP9J9`。该码属于租户创建者 `contact@daydaymoney.com` 的推荐分享码，不是租户通行证。`isGuestTenantRoute` 对任意路径只要 query 含 `accessCode` 就跳过成员守卫；Navbar 又把该 query 复制到工作面板/项目等所有租户链接，导致非成员（含超管）可停留在任意 `/tenant/:id/...` SPA。
- **Action**: (1) 将 `isGuestTenantRoute` 的 accessCode 分支收窄为 `task-detail`（及现有 `people/join`/`people/invite`）(2) 补测：`/projects/`、`/work-panel/` 带 accessCode 且用户不在 companies 时须重定向 (3) 任务详情带 accessCode 仍放行
- **Why**: 当前实现把「任务分享 guest」扩成「整租户 URL 免成员校验」；去掉 query 后超管会因 0 公司被赶到 `/onboarding/`，说明页面能打开不是因为有权限。
- **How to apply**: `taskFE/app/src/utils/tenantRouteGuard.js` `isGuestTenantRoute`；`tenantRouteGuard.test.js`；Navbar 保留 accessCode 可继续，但守卫不得因此放行非分享页

## [OPT-20260820-025] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: listJobExecutionEvents 补 workspace_id 过滤与必填校验，taskCloudService b9fd26b 已推送
- **Created**: 2026-08-20
- **Context**: ztree 层图快照 GET 已强制 `workspace_id+task_id+comment_id`。既有 `listJobExecutionEvents` 仍按 `task_id+job_id(+comment_id)` 查询，`handleContainerJobExecutionLogFromDB` 把 workspace 标成未使用。
- **Action**: (1) `listJobExecutionEvents` WHERE 增加 `workspace_id` (2) 补跨 workspace 同 job_id 不泄露测 (3) 空 workspace 返回 400 与层图对齐
- **Why**: 路径已带 workspace，查询不带则 IDOR 面与层图快照不一致。
- **How to apply**: `taskCloudService/src/job_execution_event_store.go` `listJobExecutionEvents`；`job_execution_log_handlers.go`

## [OPT-20260819-036] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: handleCreateTask fork/Idempotency-Key 去重，taskTaskService 75e5dcd 已推送
- **Created**: 2026-08-19
- **Context**: 调查 `task_877835528962076672` / `task_877835529347952640`：同 fork_from、同标题、Kafka 两条 TASK_CREATED（trace 不同，间隔 ~92ms），各扣 1 次 task_post 配额并各起一台 running 容器。前端已把 `isForking` 提前到任何 await 之前；API 客户端/重试仍可打出两次 POST。
- **Action**: (1) `handleCreateTask` 对 `fork_from`+`owner`+短时间窗或客户端 `Idempotency-Key` 做去重 (2) 重复请求返回首次已创建任务 (3) 补 Go 并发 create 回归测
- **Why**: 仅前端互斥挡不住 API 重试/多标签；重复 fork 浪费配额与云资源。
- **How to apply**: `taskTaskService/src/create_task.go`；对照计费 `consumeTaskPostQuota` / `billing_usage.task_id`

## [OPT-20260820-011] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: 抽 isHttpRepoUrl 共享工具，4 处组件改可跳转外链，taskFE c695200 已推送
- **Created**: 2026-08-20
- **Context**: 任务详情关联项目只读/编辑态仓库 URL 已改为 http(s) `<a href>`。评论撰写身份区 `CommentComposerRepoIdentity.vue` 与创建任务 `CreateTaskProjectBranchSection.vue` 仍把仓库地址渲染为 `<p>` 纯文本。
- **Action**: (1) 将上述两处 http(s) 仓库地址改为真实 `<a href>`，`target=_blank` + `rel=noopener noreferrer`，非 http(s) 保持纯文本 (2) 复用或抽出 `isHttpRepoUrl` (3) 补组件单测
- **Why**: 同一页面其它仓库地址已可点击，这两处仍要复制粘贴才能打开 GitLab。
- **How to apply**: `taskFE/app/src/components/task-detail/CommentComposerRepoIdentity.vue`；`taskFE/app/src/components/CreateTaskProjectBranchSection.vue`；参考 `TaskDetailLinkedProjectsViewMode.vue`

## [OPT-20260820-035] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: 安装时 auto_run_steps_md 为空再抽一次回填，taskCloudService 321df6d 已推送
- **Created**: 2026-08-20
- **Context**: 厂商保存与 catalog GET 已会抽取 `/app/autoRunStep.md`。租户安装仍只拷贝 catalog 快照；已安装行若在抽取完成前拷贝，会一直空，需重装才更新。
- **Action**: (1) `handleInstalledImageCollection` POST 在 `auto_run_steps_md` 为空且 `image_url` 非空时调用既有 `extractAutoRunStepsFromImage`；(2) 把结果写入 `tenant_installed_images`；(3) 补安装回退测例。不要改已超 500 行的 `installed_image_handlers.go` 本体，抽到新文件。
- **Why**: 市场页靠 catalog 缓存；已安装列表/创建任务读的是安装快照，漏抽会让勾选自动运行时仍看不到说明。
- **How to apply**: `taskCloudService/src/extract_auto_run_steps.go` 已有抽取实现；安装路径在 `installed_image_handlers.go` POST。

## [OPT-20260821-003] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: binding 表加 workspace_id 列，dataMigrate 7a96d24 + taskCloudService a9f2d18 已推送，032 已应用生产 task_cloud
- **Created**: 2026-08-21
- **Context**: `cloud_comment_container_bindings` 无 `workspace_id` 列；无任务级 CSC 时 `lookupWorkspaceIDByTask` 为空，`insertCCBLogRow` 报 `workspace_id required for log shard`，失败文案写不进评论启动日志。
- **Action**: (1) dataMigrate 给 binding 表加 `workspace_id` (2) INSERT/ensure 从请求或 start-vm statusData 写入 (3) `bindingLogWorkspaceID` 优先读列 (4) 回归测无 CSC 时 server_failed 仍可落库
- **Why**: 日志分片依赖 workspace；缺列时失败路径静默丢日志，runtime-status 只能靠进程内暂存或误导文案。
- **How to apply**: `dataMigrate/taskCloudService/` 新 SQL；`comment_container_bindings_store.go` / `ccb_log_store.go`；走 9999 初始化而非进程启动迁移。

## [OPT-20260820-046] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: CreateTaskMetaFields 下拉共用 formatInstalledImageRunLabel，taskFE e0b22af 已推送
- **Created**: 2026-08-20
- **Context**: 评论 `@镜像` 与关联项目默认镜像已共用 `formatInstalledImageRunLabel`；`CreateTaskMetaFields.vue` 仍内联 `image.name:image.version||tag||latest`。
- **Action**: (1) 下拉 option 文案改为调用共享函数 (2) 补一条与评论区相同的「名称已含版本不重复」测例
- **Why**: 三处文案格式漂移时，用户会在创建任务与详情页看到不同标签。
- **How to apply**: `taskFE/app/src/components/CreateTaskMetaFields.vue`；测例放同目录或 `utils/installedImageLabel.test.js`。

## [OPT-20260820-036] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskAuth d7a91c2 + taskFE e6ce463 已推送：微信 OAuth state 携带 accessCode、findOrCreateWeChatUser 返回 created 标志回调绑定、手机号验证码自动开通回填推荐；Go 全包 + FE 单测全绿；已登记 task-auth 精准重启（taskFE 已在册）。
- **Created**: 2026-08-20
- **Context**: 推荐页绑边已接到手机号/邮箱邀请注册（`access_code` + sessionStorage）。微信 OAuth / 账密自动开通路径不读注册页 sessionStorage，带 `?accessCode=` 的分享链接在这些入口仍不会写 `billing_referral_edge`。
- **Action**: (1) 在 OAuth state 或登录完成回调携带 `accessCode` (2) taskAuth 对应 handler 调 `bindReferralAfterRegister` (3) 补回归测：state 有码则 POST bind-from-code，无码不请求
- **Why**: 现网大量用户走微信登录，只修表单注册仍会漏绑，推荐人数继续为 0
- **How to apply**: `taskAuth` 微信/OIDC 回调；`taskFE` 登录页同样 `captureReferralAccessCodeFromSearch`；禁止解析 `u{userId}`

## [OPT-20260820-045] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskTaskService e82ed9b + taskFE fc9b0ca 已推送：单任务 JSON 水合 container_image{id,name,version,external_image_id}（列表不触发避免 N+1），lookup 404 置 container_image_removed，FE 展示「已卸载」；Go 全包 + FE 单测/组件测全绿；已登记 task-task-service 精准重启（taskFE 已在册）。
- **Created**: 2026-08-20
- **Context**: `taskToJSON` 把 `container_image` 恒写成 `null`，只给 `container_image_id`。任务详情关联项目标题栏解析默认镜像时，若 ID 已不在租户已安装目录（本页 `877435294134071296`），只能回退评论 mention 名称，看不到 version。
- **Action**: (1) 在 taskTaskService 序列化时按 ID 向 Cloud 取已安装镜像快照写入 `container_image`（或任务创建/PATCH 时落库 name+version）(2) 前端 `resolveTaskDefaultImageLabel` 优先用该对象 (3) 目录未命中时文案区分「已卸载」而非仅「已绑定」
- **Why**: 镜像卸载或换新安装后，任务仍绑定旧 ID，用户会以为默认镜像是当前目录里的 trae-agent。
- **How to apply**: `taskTaskService/src/task_store.go` `taskToJSON`；前端 `taskFE/app/src/utils/installedImageLabel.js`。

## [OPT-20260820-040] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskChromePlugin 5c857c1 已推送：getMembers 增加 workspaceId 走 workspace-collaborators（普通成员可见），缺省回退 company_members；loadMembers 缓存按工作空间隔离 + user 字段兼容；service-worker 透传 workspaceId；node --test 全包 298 绿（T5b/T5c 回归）。
- **Created**: 2026-08-20
- **Context**: 本次修 404 时把成员接口从空的 `.../members/` 改到 `company_members/`。该接口对非租户管理员返回空列表，work-panel 创建任务实际用 `/api/projects/workspace-access/workspace-collaborators/tenant_id/{tid}/?workspace_id=`。
- **Action**: (1) `getMembers` 增加 workspaceId，改打 collaborators (2) 解析与 work-panel 相同的 payload (3) 浮窗与 DevTools 负责人/协作人下拉回归测
- **Why**: 普通成员选完工作空间后负责人仍可能空白，无法对齐工作面板建任务。
- **How to apply**: 对照 `taskFE/app/src/utils/workPanelDataFetch.js` 的 `fetchCollaborators`；改 `taskChromePlugin/lib/api.js` 与 `content.js` / `panel/lib/workspace.js` 的 `getMembers` 调用。

## [OPT-20260820-010] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: runAll df13f47 已推送：docker-uninstall 启动提示改走 ./run.sh 常驻入口（调试前台另注明）；README 与 conf/runAll.yaml.ai.md 均已标注 ./bin/runAll 仅调试前台；crontab 无前台启动记录（watchdog --no-restart-runall）。
- **Created**: 2026-08-20
- **Context**: `runAll/run.sh` 已改为 `setsid -f nohup` 独立会话。2026-08-20 两次 `detail=terminated` 都来自 `screen -S run` 里直接 `./bin/runAll` 当前台作业（pid=pgid）。文档示范仍大量写 `./bin/runAll --config`。
- **Action**: (1) 搜 `screen -S run` / crontab / Agent 启动记录，把常驻入口改成 `runAll/run.sh` (2) 其余 `./bin/runAll` 示例标明仅调试前台 (3) 现网 `:9999` 宕机时用 `run.sh` 拉起，勿在 screen 里前台跑二进制
- **Why**: 只改脚本不够，screen 里继续 `./bin/runAll` 仍会被点名 SIGTERM，watchdog 默认还不重启 runAll。
- **How to apply**: `runAll/run.sh`；`runAll/README.md`；`conf/runAll.yaml.ai.md`；`screen -ls` 中 `run` 会话的 hardcopy / 启动历史

## [OPT-20260820-027] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskAiProvider 8738679 + docs ced67ad 已推送：新增 SupportedSkillVersionSet，公开目录过滤 sunset 版本镜像、审批 approve 兜底拒绝；versions.yaml 补 sunset 策略说明；Go 全包 15.4s 绿；已登记 ai-provider 精准重启。
- **Created**: 2026-08-20
- **Context**: 当前校验允许 current/deprecated 写入；sunset 仅拒新写。v2 发布并把 v1 标 sunset 后，存量已上架 v1 镜像仍可被选用。
- **Action**: (1) 在 `versions.yaml` 增加 sunset 策略说明 (2) 审核通过/公开目录过滤 sunset 声明 (3) 厂商编辑已上架镜像时强制改选仍支持的版本
- **Why**: 破坏性契约切换后若仍分发旧实现，容器会打到已删除的无 cid 路径。
- **How to apply**: `docs/skills/saas-container/versions.yaml`；`taskAiProvider` submit/approve 与 `ListApprovedCatalog`

## [OPT-20260820-016] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskTaskService 9834a51 + taskCloudService b636260 已推送：新增 POST /api/internal/git-identities/lookup/（ownership 校验），realFetchUserCompanyGitIdentity 改 HTTP，删除 task_task.* 跨库 SQL 与 gitIdentitiesTableFQN/gitIdentityLookupSQL；回归测 httptest 模拟 + 全仓扫描禁 task_task.*git_identit* 残留，双仓整包测试全绿。task-cloud-service/task-task-service 已在精准重启登记。
- **Created**: 2026-08-20
- **Context**: 任务详情「提交成功但推送失败」根因是 `taskCloudService` 跨库 SQL 仍查已改名前的表。本次已把 FQN 改为 `task_task.task_git_identities` 恢复推送，但跨库直连仍违反单表单服务所有权。
- **Action**: (1) 在 taskTaskService 增加内部 lookup（id+user_id+company_id → name/email）(2) taskCloudService `realFetchUserCompanyGitIdentity` 改为 HTTP，删除 `task_task.*` SQL (3) 保留 1146/归属回归测
- **Why**: 表再改名或切库时 cloud 会再次 502；直连也无法走 owner 侧鉴权与审计。
- **How to apply**: `taskCloudService/src/git_repo_identities_prepare.go`；`taskTaskService/src/git_identity_ensure.go` 的 `loadGitIdentityByID`；元规则 19

## [OPT-20260820-034] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: 生产只读核对通过（task_bill）：全部 3 个 task_post_quota>0 租户账实一致——877397588196749312 gift 81 + purchase 1 = quota 82（order 877909223214710784）；877834131067662336 / 878234273079914496 gift 100 = quota 100。无「只购买无 grant」老租户（惰性回填已生效），无孤儿 grant，无需回填脚本。
- **Created**: 2026-08-20
- **Context**: 任务帖购买批次改为 `billing_resource_grant.source_kind=purchase`。`ensureTaskPostPurchaseLots` 在 quotas/订单 GET 时按租户惰性回填。大租户首次请求可能扫已支付订单；账户 `task_post_quota` 与未过期 gift remaining 的差额若与历史购买数量不一致，回填 remaining 会截断。
- **Action**: (1) 上线后抽查若干混合赠送/购买租户：`SUM(remaining) WHERE source_kind=gift` + purchase remaining = `billing_account.task_post_quota`；(2) 对只购买、无 grant 的老租户确认 GET quotas 后出现 purchase lot；(3) 若存在超大租户首次延迟，再补一次性运维回填脚本（按 tenant_id 批跑，避免请求路径扫全表）。
- **Why**: 惰性回填正确但不透明；生产对账能尽早发现账户计数与批次 remaining 漂移。
- **How to apply**: `taskBill/src/task_post_quota_lots.go` `ensureTaskPostPurchaseLots`；对照 `task_post_quota_source_test.go` TestEnsureTaskPostPurchaseLotsBackfill；只读 SQL 即可先验收。

## [OPT-20260820-023] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: 生产核对分片行数 1993 ≥ 遗留表 1410；dataMigrate `2b1bf46` 新增 033_drop_legacy_ccb_logs.sql（幂等 RENAME → cloud_comment_container_binding_logs_deprecated_20260821）已应用生产 task_cloud（data_migrate_log 33/33）；taskCloudService `c618f1a` 隔离回归测与清理清单改指 deprecated 表名。应用已确认无遗留表写入。
- **Created**: 2026-08-20
- **Context**: `030_ccb_logs_workspace_shards.sql` 把存量从 `cloud_comment_container_binding_logs` 回填到 16 张分片；应用已停止写入遗留表。表仍占空间，且容易被误用为查询入口。
- **Action**: (1) 生产核对分片行数 ≥ 遗留表中能解析到 workspace 的行数；(2) 新增 `dataMigrate/taskCloudService/031_drop_legacy_ccb_logs.sql`：`RENAME` 过渡一周或直接 `DROP TABLE`；(3) 删除代码/文档中对遗留表名的写入暗示。
- **Why**: 遗留表继续存在会让排障走错表，也会让「应用已停止写入」的约束靠约定而非物理不可写。
- **How to apply**: 先 `SELECT COUNT(*)` 对比遗留表 vs `UNION ALL` 16 片；确认 taskCloudService 无 `FROM cloud_comment_container_binding_logs`（无 `_NN` 后缀）后再 DROP。

## [OPT-20260820-042] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskFE 59b3f7f 已推送：SystemAdminRefundPanel 操作列新增「查看消耗/收起消耗」，行内复用 useAdminRefundOrderConsumption + OrderResourceConsumption 展示已消耗/剩余，无 order_id 行不渲染；inline-consumption 回归测 2 例 + 既有 12 例全绿，anti-replay gate ok
- **Created**: 2026-08-20
- **Context**: 超管退款 Tab 已在批准/拒绝弹层展示订单任务帖已发放/已消耗/剩余。扫待审批列表时仍须点开弹层才能看消耗。
- **Action**: (1) 在退款申请表增加展开或摘要列（已消耗/剩余）(2) 复用 `useAdminRefundOrderConsumption` (3) 补 vitest：列表即可看到 remaining
- **Why**: 批量审批时反复开关弹层成本高，一眼看到已消耗可加快拒绝「已用完仍申请全额退」的单。
- **How to apply**: `taskFE/app/src/components/system-admin/SystemAdminRefundPanel.vue`；数据仍走 `GET /api/tenant/{tid}/billing/orders/{id}/`

## [OPT-20260820-041] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskBill c59dad8 + dataMigrate 9bee7cb + taskFE 93dd91e 已推送：admin_grant 赠送批次落 region（规范化 slug）；053 迁移已应用生产 task_bill（region 列+回填，仅唯一区域批次回填）；regionResourceView 增 disk/traffic_gifted_gb+purchased_gb；BillingDashboard 区域卡片展示「赠送 X · 购买 Y」。Go 整包 98s 绿 + FE vitest 4 文件 8 例绿。task-bill 已在精准重启登记，待 runAll 恢复后重启生效。
- **Created**: 2026-08-20
- **Context**: 任务帖配额已在账单页拆成赠送/购买剩余，并在订单上归属消耗。GitLab 磁盘与流量仍只有合计，购买与后台赠送无法在同一卡片区分。
- **Action**: (1) 评估 `billing_resource_grant` 对 gitlab_disk/gitlab_traffic 是否已有批次 (2) 配额 GET 增加 gifted/purchased 或按区域拆分 (3) BillingDashboard 卡片展示 (4) 补 Go/vitest 回归
- **Why**: 租户无法判断磁盘/流量剩余来自赠送还是订单，续费与退款沟通成本高。
- **How to apply**: 对照 `taskBill/src/task_post_quota_lots.go` 与 `GET /billing/quotas/`；GitLab 有区域维度，拆分字段需带 region。

## [OPT-20260820-022] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskCloudService b3d4857 + dataMigrate 2f1b02a 已推送：jobExecutionEventShard CRC32(workspace_id)%16 选片，insert/list/latestJobIDForComment(带 workspaceID) 改走分片表；034 迁移已应用生产 task_cloud（16 表+回填，核验 124 行全入 06 片）；回归测 TestJobExecutionEventShardCrossWorkspaceRouting 跨片隔离 + 整包 287s 绿。task-cloud-service 已在精准重启登记，待 runAll 恢复后重启生效。
- **Created**: 2026-08-20
- **Context**: 评论启动日志已按 ADR-0023 拆成 `cloud_comment_container_binding_logs_{00..15}`。同库时间累积表 `cloud_job_execution_event`（及可选 `cloud_server_events`）仍是单表，工作空间膨胀后会再次出现全租户混扫。
- **Action**: (1) 复用 ADR-0023：`CRC32(utf8(workspace_id)) % 16`、Snowflake PK、表名仅 `%02d`；(2) 新增 `dataMigrate/taskCloudService/` 迁移预创建 16 张表并回填；(3) 改 taskCloudService 读写选表；(4) 补跨 workspace 隔离回归测。
- **Why**: 启动日志只是评论执行区的一块；作业执行事件同样按 workspace 膨胀，不拆则会把热点重新打回单表。
- **How to apply**: 对照 `taskCloudService/src/ccb_log_shard.go` 与 `dataMigrate/taskCloudService/030_ccb_logs_workspace_shards.sql`；先冻结向量再改写路径。`cloud_server_events` 已有时间分区，是否再哈希分表需按年增量评估。

## [OPT-20260820-021] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: HTTP fallback 评论作者查询闭环：taskTaskService aad702a 新增 GET /api/internal/comments/{id}/created-by 独立解析 created_by_id（owner-service 无 taskID 上下文无法复用 container-snapshot 缓存）；taskCredentialService cf0081e HTTP fallback FetchCommentCreatedByUserID 改走该 API，回归测 EnrichReposForCommentAuthor discovery_user=评论作者（不再落任务级/匿名）。FetchUserGitIdentities 仍为 nil（HTTP 降级路径评论身份已 FromComment，作者全量身份 fallback 极少触发，按条目 Action 范围不扩）。双仓整包绿。
- **Created**: 2026-08-20
- **Context**: 容器提交者身份已改为评论 JSON（SQLite `FetchRepoIdentities(task, comment)`）。HTTP fallback 虽能映射 snapshot 的 name/email，但 `FetchCommentCreatedByUserID` / `FetchUserGitIdentities` 仍恒返回空，嵌套仓发现在直连 MySQL 不可用时仍可能用错 user_id。
- **Action**: (1) 给 container-snapshot 增加 `created_by_id`（或独立 comment-author 内部 API）(2) HTTPBusinessRepository 映射该字段 (3) 回归测：HTTP 路径 EnrichReposForCommentAuthor 的 discovery_user 等于评论作者
- **Why**: 生产虽走 SQLite，HTTP 是 owner-service 降级路径；作者查不到时嵌套仓 OAuth 会落到任务级/匿名，私有子仓克隆失败难排查。
- **How to apply**: `taskTaskService/src/container_snapshot.go`；`taskCredentialService/infrastructure/http_business.go` `FetchCommentCreatedByUserID`；对照 `application/nested_repos_enrich.go` `lookupCommentAuthorUserID`

## [OPT-20260820-019] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: taskTenantService 08d8cbc + taskAuth 89cb1d2 已推送：末位成员注销执行后由 taskTenantService internal API cleanup-invites 失效无剩余活跃成员租户的 pending 邀请（回拨 invitation_token_expires_at 过期，handleJoin/handlePendingInvitations 天然过滤）；best-effort 不阻断注销。回归测 taskTenantService（4 例）与 taskAuth（execute-due 断言 cleanup 调用 + 失败不阻断）全绿，pre-commit 通过。
- **Created**: 2026-08-20
- **Context**: 账号注销 precheck 已不再把 `TENANT_INVITE_PENDING` 当硬门禁（邀请属租户级，且过期行不会出现在人员管理页）。执行日归档末位成员后，`tenant_invitation` 未接受行可能残留。
- **Action**: (1) 在 `execute-due` 且该租户无剩余活跃成员时，由 taskTenantService internal API 将 `is_accepted=0` 的邀请标为失效/删除 (2) 补回归测：末位成员注销后 pending 邀请不可再 join
- **Why**: 不清理则过期链接仍可能被误用，或继续出现在运营排查的「未接受」统计里。
- **How to apply**: `taskAuth` execute-due 编排；`taskTenantService` `tenant_invitation`；对照 `invite_handlers.go` 的 pending 列表条件

## [OPT-20260821-010] completed

- **Status**: completed
- **Completed**: 2026-08-21
- **Summary**: Cloud GET 空快照 live fallback + 代理注入 X-TaskContainerGateway-Internal-Secret；公网任务关联已渲染可写层 2 + 项目文件树 ram-work
- **Created**: 2026-08-21
- **Context**: 任务详情「任务关联」ztree 空白修复后，hydrate 走 `GET container-layer-graph`（Cloud FromDB）。若 `cloud_layer_graph_snapshot` 尚无行（容器未成功 persist），前端会一直 loading；CDP 直打同路径曾 401，而 `container-task-ui-context` 200。
- **Action**: (1) Cloud GET 空快照且 binding running 时，best-effort 转发 Gateway live layer-graph 一次并 persist (2) 对比 APISIX/Cloud 对 `container-layer-graph` 与 `container-task-ui-context` 的鉴权差异，消除 401 (3) 补回归：空库 + live 200 → 树非空且下次 GET 走 DB
- **Why**: 只修空白/gate 后，无快照的运行中容器仍看不到 ztree，用户会以为功能坏了。
- **How to apply**: `taskCloudService/src/layer_graph_snapshot_handlers.go`；`isContainerOutboundComputeSub` / `proxyContainerGatewayRequest`；APISIX `/api/cloud/compute/container-layer-graph`

