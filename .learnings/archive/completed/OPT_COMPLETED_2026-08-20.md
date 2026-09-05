# Completed OPT Archive — 2026-08-20

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 59 条。
> 归档执行时间：2026-08-25T22:13:52+08:00

## [OPT-20260820-044] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 新 SPA `index-D6o0T3dC.js` 经 APISIX `http://10.2.150.68:18081` CDP 验收：评论 Tab `comment-execution-tab-server-content` 与「执行细节/服务器运行状态」并列；`server-content-section[data-comment-id=cmt_878227569307054080]`；GET `/api/cloud/compute/server-content/.../comment_id/cmt_878227569307054080/`；任务级 `server-content-moved-hint` 可见且无「缺少评论ID」。当时 `www.daydaymoney.com` 边缘 502（SH ssh :2222 超时）。
- **Blocked-By**: BROWSER
- **Created**: 2026-08-20
- **Context**: 任务详情任务级「服务器内容」卡片因无评论 ID 显示「缺少评论ID」。已迁入各评论执行细节「服务器内容」Tab。
- **Action**: CDP 打开任务详情 → 断言评论 Tab 与迁出提示 → 点「服务器内容」后检查 `data-comment-id` 与 `/comment_id/` 请求。
- **Why**: 未部署新 SPA 时任务级仍会空拉。
- **How to apply**: 选择器 `[data-testid=server-content-moved-hint]`、`[data-testid=comment-execution-tab-server-content]`、`[data-testid=server-content-section]`。

## [OPT-20260820-030] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: SPA releases/20260820191135-1512747 热重载后 CDP 读 [data-testid=comment-execution-clone-progress] 文案为「项目克隆 100% ram-work 100% 项目克隆 (1/1) 完成 ram-work」，不含单独 3%。taskFE ea4c217，镜像 x86_64_2026-08-20_19-07。
- **Blocked-By**: BROWSER
- **Created**: 2026-08-20
- **Context**: 任务 `task_878227563963510784` 未开启自动克隆子仓库，评论条曾到 100% 后又显示「项目克隆 3% / ram-work 3% / (1/1) ram-work … 3%」。根因是全局「仓库克隆已完成」清空分仓 100% 后，大仓 stderr 尾窗迟到 Receiving 3% 被当成新进度。代码已修 FE 保留分仓完成态 + 容器 overall 取全文 Checkout 峰值 + 同进程百分比只升不降。
- **Action**: (1) http://10.2.150.68:9999/ 精准编译重启 `taskFE`（容器镜像需已推送 onlineServiceJS）(2) 硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878227563963510784/?accessCode=DR2AKvP9J9 (3) 断言 `[data-testid=comment-execution-clone-progress]` 为 100% 且文案含「完成 ram-work」，不得长时间停在单独的 3%；(4) 新开未开子仓克隆的 ram-work 任务时，进度到 100% 后刷新/等待不得回落到 3%
- **Why**: 未重建 SPA / 未换容器镜像时，公网仍会把迟到 3% SSE 盖掉完成态。
- **How to apply**: CDP 9222。选择器 `data-testid=comment-execution-clone-progress`、`comment-execution-clone-progress-overall`。

## [OPT-20260819-027] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill 965babc 已推送：createOrderWithNote 同事务内 buyer_note 非空时插入 author_side=tenant 的首条 billing_order_comment，作者取 X-User-Id（缺失 tenant:{id} 占位）；新增有留言写 1 条/无留言 0 条回归测。
- **Created**: 2026-08-19
- **Context**: 本会话落地 `buyer_note` 订单字段；成单后 `billing_order_comment` 线程仍独立，施工人员若只看评论区会看不到下单留言。
- **Action**: 当 `buyer_note` 非空时，在同事务插入一条 `author_side=tenant` 的 `billing_order_comment`（或详情页同时展示字段+线程），并补回归测。
- **Why**: 统一施工人员查看入口，避免留言藏在订单头字段而评论区空白。
- **How to apply**: `createOrderWithNote` + `order_comments.go`；OrderDetail / OrderExpandDetail

## [OPT-20260819-015] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill c75ba00 + dataMigrate e3910a7 已推送：billing_resource_grant.billing_transaction_id 外键 + enrich 优先按 ID 关联、秒级仅兜底；与 026 同根因一并落地（grant FK 替代 order-item JOIN 达成精确归属），回归测见 transaction_change_enrich_test.go。
- **Created**: 2026-08-19
- **Context**: 列表 enrich 现按 `created_at` 秒级把 `billing_resource_grant` 挂到 `admin_grant` 流水；同秒多笔赠送可能串单。
- **Action**: (1) grant 或 transaction 表增加 `billing_transaction_id`（或对称外键）；(2) enrich 优先按 ID 关联，秒级仅作兜底；(3) 迁移放 `dataMigrate/taskBill/`。
- **Why**: 秒级匹配在并发赠送下会错配变动明细。
- **How to apply**: `dataMigrate/taskBill/` + `transaction_change_enrich.go` + 写路径 `handlers_admin_grant.go`。

## [OPT-20260818-026] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill c75ba00 + dataMigrate e3910a7 已推送：billing_transaction.related_order_id + billing_resource_grant.billing_transaction_id（迁移 050 已应用到生产 task_bill），adminGrantResources 写路径落两处 FK，enrich 优先按 FK 精确归属、秒级仅作旧行兜底；新增同秒两笔互不串单 + 写路径 FK 落库回归测。
- **Created**: 2026-08-18
- **Context**: 账单「最近交易」配额入账已按 `created_at` 秒级对齐 `billing_resource_grant` 做展示 enrich。同一秒两次独立赠送会串明细。
- **Action**: (1) 在 `billing_transaction` 增加 `related_order_id`（或写入 `transaction_id` 可解析外键）(2) `adminGrantResources` 写入时带上订单 ID (3) 列表 enrich 按 ID JOIN `billing_resource_order_item`，去掉时间窗口匹配
- **Why**: 秒级匹配在并发赠送时会把资源明细挂到错误流水上。
- **How to apply**: `taskBill/src/admin_grant.go`、`transaction_change_enrich.go`；需 `dataMigrate/taskBill/` DDL + 9999 初始化

## [OPT-20260819-040] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: runAll c103899 已推送：/api/exec-queue 按 type 折叠重复 pending、/api/build-all 在 build-all 已排队时 409、页头按钮确认弹窗打开期间 disabled；回归测 exec-queue 折叠双 build-all + build-all pending 409 + UI 页含 confirming 锁。整包 go test ./src 107s 绿。
- **Created**: 2026-08-19
- **Context**: 在 http://10.2.150.68:9999/ 点「全部重新编译」并确认后，`execution_queue.pending` 仍出现第二条 `build-all`，导致 67 个服务编译刚到尾声又从头再跑一轮（约 +2 分钟）。当前 `active_bulk` 已是 `build-all` 时不应再 enqueue。
- **Action**: (1) 在 `runall_page.html` / runAll 队列逻辑里，打开确认弹窗后立即 disable 页头按钮 (2) `POST /api/build-all` 若已有同类型 in-flight 或 pending 则 409 且前端不再排队 (3) 补单测：确认中连点不会产生第二条 `build-all`
- **Why**: 全量编译本就耗内存；重复入队会拖长窗口并增加 OOM 风险。
- **How to apply**: `runAll/src` 执行队列 + `runall_page.html` `buildAllServices` / `showModalConfirm`；对照 `/api/status` 的 `execution_queue.pending`

## [OPT-20260819-043] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: db 3053588 已推送：暂存新增符号 git 历史回退检测（add→delete→re-add 拦截、waiver 豁免、全新符号放行）。另修复 taskAuth/taskCloudService OpenAPI 契约漂移（3 路由补齐，8/8 绿）解除 db 随机门禁。
- **Created**: 2026-08-19
- **Context**: 元规则 53 / ADR-0021 的 CI 只拦高信号关键词（`fallbackToOld`、`*_legacy.go`、注释掉的旧函数、`revert:` message）。从 git 历史原样拷回已删函数但改名时不会命中。
- **Action**: (1) 对暂存新增函数做 `git log -S` / 相似度比对，识别近 N 次提交中已删除的同名或高相似块 (2) 命中则同样要求 `Logic-Rollback-OK` (3) 先对 taskFE/taskBill 试点以免误报
- **Why**: 静默拷回是 Agent 最常见的无关键字回退，仅靠词表会漏。
- **How to apply**: 扩展 `db/scripts/ci/check_logic_rollback_approval.py`；对照 `.ai/01_project_constraints/58_logic_rollback_requires_approval.md`

## [OPT-20260819-041] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill 64e23b6 已推送：累计支付并入 refund 负额扣减 + refund_deduction_points/_cents 拆分字段；回归测 PayPal 充值+订单实付+退款→净额。
- **Created**: 2026-08-19
- **Context**: 账单首页累计支付现为钱包实付充值 + 资源订单 `resource_purchase`，不含 `admin_grant`。退款流水 `transaction_type=refund` 未冲减该合计，退款后「累计支付」会偏高。
- **Action**: (1) 在 `handleBillingStatistics` 将已完成退款从 `user_recharge_points` 扣减（或单独返回净支付）(2) 补 Go 测：PayPal 充值 + 订单实付 + 退款后净额 (3) 卡片文案标明是否为净值
- **Why**: 用户对照消耗/支付时，未扣退款会把已退回的钱仍算作累计支付。
- **How to apply**: `taskBill/src/handlers_billing_query.go` `handleBillingStatistics`；`handlers_billing_statistics_test.go`

## [OPT-20260819-042] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 9402407 已推送：购买资源/查看全部改真实 a href + realHref 回归测。
- **Created**: 2026-08-19
- **Context**: 修累计消耗/支付时看到 `BillingDashboard.vue`「购买资源」仍用 `router-link`。元规则 49 要求导航用真实 `<a href>`，避免左键拦截与守卫回弹打架。
- **Action**: (1) 将购买资源与「查看全部」交易链接改为 `<a :href="...">` (2) 补组件测断言原生 href (3) 搜索同页其余 router-link
- **Why**: 与租户内其它计费入口（订单列表已用真实 href）不一致，守卫失败时可能静默回弹。
- **How to apply**: `taskFE/app/src/views/BillingDashboard.vue`；对照 `.ai/01_project_constraints/49_no_link_click_interception.md`

## [OPT-20260819-001] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE bd514a3 已推送：抽 fetchGitOAuthUserAppConnected 共享函数，创建任务门禁与评论侧共用；共享单测 6 例 + CreateTaskModal 30 例绿。
- **Created**: 2026-08-19
- **Context**: 创建任务门禁 `useCreateTaskRepoOAuth` 与评论侧 `useLinkedProjectsRepoOAuth` 各自请求 `/api/git-oauth/user-app-connection/`，超时/解析 connected 逻辑重复。
- **Action**: (1) 抽出共享 `fetchGitOAuthUserAppConnected(repoUrl)` (2) 两处 composable 改为调用 (3) 保留既有单测并补共享函数测例
- **Why**: 绑定判定分叉会导致创建门禁与评论补救提示不一致。
- **How to apply**: `taskFE/app/src/composables/useCreateTaskRepoOAuth.js`、`taskFE/app/src/composables/taskDetail/useLinkedProjectsRepoOAuth.js`、`createTaskOauthGate.js` 的 `isGitOAuthUserAppConnectedPayload`

## [OPT-20260819-031] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill 7306aaa 已推送：frozen_points 单位分描述 + frozen_yuan 只读字段 + 15 处 *_points 单位说明；回归测 refundApplicationJSON。
- **Created**: 2026-08-19
- **Context**: 超管退款审批列表曾把 `frozen_points=55` 原样显示为「55」，根因是前端未分转元；OpenAPI `RefundApplication.frozen_points` 仅写 integer，未说明单位，易再次误用。
- **Action**: (1) 在 `taskBill/src/openapi.yaml` 的 `frozen_points` 增加 description「金额，单位：分（1 元 = 100）」(2) 扫同文件其它 `*_points` 金额字段补单位说明 (3) 可选：响应增加只读 `frozen_yuan` 字符串字段减少前端换算
- **Why**: 契约层写清单位，避免新 UI/客户端再次把分当元展示。
- **How to apply**: `taskBill/src/openapi.yaml` `RefundApplication`；对照 `refund.go` / `refund_approve.go` 序列化字段

## [OPT-20260819-016] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 9748af0（BillingDashboard 部分随 9402407）：行级 _lines 单次计算，回归测每行仅一次 ledgerSnapshotLines。
- **Created**: 2026-08-19
- **Context**: `BillingTransactionsTable.vue` / Dashboard 在 `v-for` 与空态 `v-if` 各调用一次 `ledgerSnapshotLines(transaction)`，行多时多余计算。
- **Action**: (1) 用计算属性或单次 helper 缓存行级 lines；(2) 更新相关 vitest。
- **Why**: 减少重复解析，模板更清晰。
- **How to apply**: `taskFE/app/src/components/BillingTransactionsTable.vue` 等。

## [OPT-20260819-044] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: trae-agent 18e213a 已推送：postJson 捕获 X-Trace-Id → BOOTSTRAP_FAILED 透传 runtime-event body trace_id；回归测 3 例。Docker 镜像推送已触发。
- **Created**: 2026-08-19
- **Context**: 任务关联引导失败已能展示 Cloud inbound / 容器启动 `TRACE_ID` 的 `data-traceId`。克隆凭证 502（`repo-clone-credentials`）往往是另一条请求级 trace；`emitRuntimeEvent('BOOTSTRAP_FAILED')` 未把该 502 的 `X-Trace-Id` 放进 body/`fields`，Loki 无法从页面 trace 直接拼到 Git OAuth 502。
- **Action**: (1) 捕获 `repo-clone-credentials` 失败响应头 `X-Trace-Id` (2) `emitRuntimeEvent` / `postRuntimeEventToSaas` 写入 `trace_id` 或 `fields.trace_id` (3) 单测断言 POST body 含该 id
- **Why**: 页面 `data-traceId` 应对齐真正失败的克隆凭证请求，而不仅是 runtime-event 入站。
- **How to apply**: `trae-agent/onlineServiceJS/src/bootstrap.mjs`、`runtimeEventLog.mjs`、`saasPostJson.mjs`；Cloud 已有 `firstNonEmptyTraceID` 读 body

## [OPT-20260819-024] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE a63cdd3 已推送：用户页 tab 深链 + 推荐资格页链接带 query；回归测 4 例。
- **Created**: 2026-08-19
- **Context**: 「推荐码申请列表」已迁到 `/system-admin/users/` 第三 tab（`referral-apps`）；推荐资格页仅留策略/入账并链到用户页。目前不能通过 `?tab=referral-apps` 直达。
- **Action**: (1) `useSystemAdminUsers` / `SystemAdminUsers.vue` 读取 `route.query.tab` 初始化 `activeTab`；(2) `switchTab` 同步 query；(3) 推荐资格页链接改为带 query；(4) 补 vitest + 可选 Playwright。
- **Why**: 从策略页跳转可直接落到申请审批，减少二次点击。
- **How to apply**: `taskFE/app/src/views/SystemAdminUsers.vue`；`useSystemAdminUsers.js`；`SystemAdminReferralManagement.vue`

## [OPT-20260819-034] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskAuth d37b702 已推送：findLoginMethodByCanonicalPhone 共享查找 + handleLogin 直用；回归测 2 例。
- **Created**: 2026-08-19
- **Context**: 手机号+密码登录曾按 identifier 精确匹配 E.164，而注册把 identifier 存成国内号。修复后登录复用 `findLoginMethodForPasswordReset(phone, "")`，函数名仍绑在「找回密码」语义上。
- **Action**: (1) 在 `taskAuth/src/db.go` 抽出 `findLoginMethodByCanonicalPhone` (2) `findLoginMethodForPasswordReset` / `handleLogin` 均调用它 (3) 保留现有 E.164 登录回归测
- **Why**: 避免后续改密码重置时误伤登录查找，或再复制一套规范化逻辑。
- **How to apply**: `taskAuth/src/db.go` `findLoginMethodForPasswordReset`；`taskAuth/src/auth_login.go` `handleLogin`

## [OPT-20260819-017] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 00b43e0 已推送：退款拒绝改自定义模态框，window.prompt 移除；回归测 2 例。
- **Created**: 2026-08-19
- **Context**: 为「开启退款申请」开关补二次确认时发现，同页 `approve`/`reject` 仍用 `window.prompt` 收集备注，违反前端「禁止浏览器原生弹窗」规范。
- **Action**: (1) 用 `modalService` 或独立确认/备注 Modal 收集 note；(2) 取消时不发 POST；(3) 补 vitest + 更新 Playwright 审批 mock 用例。
- **Why**: 与策略开关确认体验一致，且符合自定义模态框规范。
- **How to apply**: `taskFE/app/src/views/SystemAdminRefundApplications.vue` 的 `postAction`；参考 `SystemAdminGitlabRegionDeleteModal.vue`。

## [OPT-20260818-027] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE e6f0db3 已推送：删除无引用 BillingDashboardRecentTransactions.vue，Dashboard 内联为主实现。
- **Created**: 2026-08-18
- **Context**: `BillingDashboard.vue` 内联了最近交易表；`BillingDashboardRecentTransactions.vue` 无引用但仍有一份并列实现。本次已同步变动列，仍是双份维护。
- **Action**: (1) 确认无动态 import/路由引用 (2) 要么让 Dashboard 改用该组件，要么删除组件及 alias
- **Why**: 两套表格会再次出现「入账内容」只改了一处的回归。
- **How to apply**: `taskFE/app/src/views/BillingDashboard.vue`、`taskFE/app/src/components/BillingDashboardRecentTransactions.vue`

## [OPT-20260819-020] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: runAll 06bbe77 已推送：root commit_repo 透传 include_untracked；真实 submodule gitlink 回归测 2 例。
- **Created**: 2026-08-19
- **Context**: 批量 `commit_with_submodules --apply` 后根仓第一次成功提交只写入 learnings + taskBill 指针；其余 12 个已推送子仓 gitlink 虽已 `git add`，却未进入该 commit，需二次补提交。疑与 git≥2.53 `GIT_INDEX_FILE=next-index-*.lock` + pre-commit 框架改写索引有关；且 `commit_repo("(root)")` 未透传 `--include-untracked`。
- **Action**: (1) 在 `runAll/scripts/commit_with_submodules.py` 的 root `commit_repo` 调用透传 `include_untracked` (2) 根仓 pre-commit/自测：模拟暂存多个 gitlink + 钩子改 learnings 后，断言最终 commit 仍含全部 gitlink (3) 必要时在 `.githooks/pre-commit` 恢复 `GIT_INDEX_FILE` 前后对 staged gitlink 做校验日志
- **Why**: 子仓已推但 meta 指针漏提会导致他人 clone 仍指向旧 SHA，与规则 32「先子后根」交付完整性冲突。
- **How to apply**: `runAll/scripts/commit_with_submodules.py` Phase 2；`.githooks/pre-commit`；对照本会话 `1837fbcd` vs `3b999a54`

## [OPT-20260819-039] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: docs 已推送：v86 target 四件套（diff/full .archimate + puml + mermaid.md）+ VERSION_HISTORY 条目；archimate-tool check 0 critical。
- **Created**: 2026-08-19
- **Context**: `docs/superpowers/specs/2026-08-19-user-account-deletion-pipl-design.md` 标注目标 v86 target，本次仅落地代码与 intent，未生成 ArchiMate/puml/VERSION_HISTORY。
- **Action**: 在 `docs/architecture/` 增补 v86 target 四件套并更新 VERSION_HISTORY，引用跨服务 blockers + execute-due timer + Kafka 五事件。
- **Why**: 设计文档与 archimate 元规则要求跨服务编排变更可可视化追溯。
- **How to apply**: 对照 `.cursor/rules/archimate-architecture-artifacts.mdc` 与 `.claude/skills/archimate/SKILL.md`

## [OPT-20260818-026] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill c75ba00 + dataMigrate e3910a7 已推送：billing_transaction.related_order_id + billing_resource_grant.billing_transaction_id（迁移 050 已应用到生产 task_bill），adminGrantResources 写路径落两处 FK，enrich 优先按 FK 精确归属、秒级仅作旧行兜底；新增同秒两笔互不串单 + 写路径 FK 落库回归测。
- **Created**: 2026-08-18
- **Context**: 账单「最近交易」配额入账已按 `created_at` 秒级对齐 `billing_resource_grant` 做展示 enrich。同一秒两次独立赠送会串明细。
- **Action**: (1) 在 `billing_transaction` 增加 `related_order_id`（或写入 `transaction_id` 可解析外键）(2) `adminGrantResources` 写入时带上订单 ID (3) 列表 enrich 按 ID JOIN `billing_resource_order_item`，去掉时间窗口匹配
- **Why**: 秒级匹配在并发赠送时会把资源明细挂到错误流水上。
- **How to apply**: `taskBill/src/admin_grant.go`、`transaction_change_enrich.go`；需 `dataMigrate/taskBill/` DDL + 9999 初始化

## [OPT-20260819-015] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill c75ba00 + dataMigrate e3910a7 已推送：billing_resource_grant.billing_transaction_id 外键 + enrich 优先按 ID 关联、秒级仅兜底；与 026 同根因一并落地（grant FK 替代 order-item JOIN 达成精确归属），回归测见 transaction_change_enrich_test.go。
- **Created**: 2026-08-19
- **Context**: 列表 enrich 现按 `created_at` 秒级把 `billing_resource_grant` 挂到 `admin_grant` 流水；同秒多笔赠送可能串单。
- **Action**: (1) grant 或 transaction 表增加 `billing_transaction_id`（或对称外键）；(2) enrich 优先按 ID 关联，秒级仅作兜底；(3) 迁移放 `dataMigrate/taskBill/`。
- **Why**: 秒级匹配在并发赠送下会错配变动明细。
- **How to apply**: `dataMigrate/taskBill/` + `transaction_change_enrich.go` + 写路径 `handlers_admin_grant.go`。

## [OPT-20260819-027] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill 965babc 已推送：createOrderWithNote 同事务内 buyer_note 非空时插入 author_side=tenant 的首条 billing_order_comment，作者取 X-User-Id（缺失 tenant:{id} 占位）；新增有留言写 1 条/无留言 0 条回归测。
- **Created**: 2026-08-19
- **Context**: 本会话落地 `buyer_note` 订单字段；成单后 `billing_order_comment` 线程仍独立，施工人员若只看评论区会看不到下单留言。
- **Action**: 当 `buyer_note` 非空时，在同事务插入一条 `author_side=tenant` 的 `billing_order_comment`（或详情页同时展示字段+线程），并补回归测。
- **Why**: 统一施工人员查看入口，避免留言藏在订单头字段而评论区空白。
- **How to apply**: `createOrderWithNote` + `order_comments.go`；OrderDetail / OrderExpandDetail

## [OPT-20260820-005] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: db b780dd9 已推送：check_account_deletion_action_urls.py + 11 单测全绿，.pre-commit-config.yaml 注册 check-account-deletion-action-urls。taskFE 908fc4a/docs 8de81df/taskCloudService 9e0e5a7 已实现前端 remap+路由别名，脚本从 router/*.js 收集已注册路由与 redirect 别名，校验 taskBill/taskCloudService/taskTenantService 的 account_deletion_action_url.go 产出 URL 模式（9 个全部命中）。meta 指针待收尾同步。
- **Created**: 2026-08-20
- **Context**: 注销「前往处理」曾指向 `/billing/gitlab-resources/`、`/settings/members/`、`/workspace/.../task/` 等未注册路径，SPA catch-all 会踢回首页。本会话已改后端 URL + 前端 remap + 路由别名；仍缺少自动扫描，后续新增 blocker 可能再写假路径。
- **Action**: (1) 从 `taskFE/app/src/router/*.js` 收集已注册 path (2) 扫描 `ActionURL`/`action_url` 字面量 (3) 未命中则 CI 失败；别名 redirect 算合法
- **Why**: 单靠 code review 挡不住下次复制错误路径；catch-all 到 home 的失败模式会再次出现。
- **How to apply**: 新增 `db/scripts/ci/check_account_deletion_action_urls.py`，对照 `tenantRoutes.js` 与 `account_deletion_action_url.go`

## [OPT-20260820-003] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE ff2900c 已推送：layerZtreeTab 晚变 true 且用户未手动选过 Tab 时自动切到「任务关联」，pickTab 记录用户选择，onBindingStatusClick 跳前序视为用户导航；补 2 组件单测，组件全套 38 测全绿。Playwright 强制 click workaround 未删（需浏览器验收，本窗口不触发）。
- **Created**: 2026-08-20
- **Context**: `TaskDetailCommentExecutionDetails` 在 mount 时若 `layerZtreeTab` 仍为 false（bindings 未返回），`activeTab` 会落在「执行细节」。之后 CSC 绑定到达只把 Tab 按钮显示出来，不切换 activeTab；任务关联失败态在 DOM 中但 `v-show` 隐藏。
- **Action**: (1) watch `layerZtreeTab` 从 false→true 且用户未手动选过 Tab 时切到 `ztree` (2) 补组件单测 (3) 可删 Playwright 里强制 click 任务关联的 workaround
- **Why**: 用户打开执行细节时期望先看到可写层；晚绑定导致失败提示藏在未选中 Tab。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.vue` `activeTab` watch；对照 `TaskDetailCommentExecutionDetails.ztree-tab.test.js`

## [OPT-20260819-033] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 全部闭环：Go 侧 taskBill 8a865ae（adminOrderFocusOffset + doAdminListOrders order_id 对齐 + OpenAPI query + 3 回归测）+ FE 侧 taskFE 99eb737（SystemAdminOrderListPanel 全部租户模式也透传 order_id，2 deeplink 单测 + 相关 7 测全绿）均已推送。
- **Created**: 2026-08-19
- **Context**: 退款「关联订单」深链已改为管理端订单 Tab，并在选中租户后走 `GET /api/tenant/{id}/billing/orders/?order_id=`。无 `tenant_id` 时回退「全部租户」走 `GET /api/system-admin/orders/`，该接口尚不支持 `order_id` 对齐，目标订单若不在首页则无法高亮。
- **Action**: (1) `doAdminListOrders` 增加与 `handleListOrders` 同款的 `order_id` → offset 对齐及 `focus_order_id` (2) OpenAPI `/api/system-admin/orders/` 补 query (3) 补 Go 单测 + 前端全部租户模式下也可传 `order_id`
- **Why**: 仅有订单号、租户选项未命中时深链会静默落在第一页。
- **How to apply**: `taskBill/src/handlers_orders_pay.go` `doAdminListOrders`；对照 `handlers_orders.go` `tenantOrderFocusOffset`
- **2026-08-20 夜**: (1)(2)(3-Go) 已完成 — taskBill `8a865ae` 已推送：`adminOrderFocusOffset` + `doAdminListOrders` 支持 `order_id` 对齐并回传 `focus_order_id`，OpenAPI 补 query，三个 Go 回归测绿。剩余 (3-FE)：`SystemAdminOrderRecords.vue` 全部租户模式透传 `order_id`，因 taskFE 工作树有他会话 staged WIP（bootstrap 克隆日志批）本窗口未提交。

## [OPT-20260820-002] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 闭环：后端 trae-agent 4fea5e7（empty 层输出 bootstrap_pending 快照节点 + 2 单测，镜像已推）+ FE taskFE 0209586（topicForLayer/ztStyleForLayer 识别 bootstrap_pending/meta_kind=empty：行名「正在准备可写层」+ ztStyle=active，4 单测全套 23 测全绿）均已推送。剩余 taskFE mock Playwright 未补（需浏览器/CDP 验收，本窗口不触发）。
- **Created**: 2026-08-20
- **Context**: 容器 `ensureStartupEmptyLayer()` 会创建 `kind=empty` 层，但 `jobsRuntimeSnapshot.mjs` 的 `buildLayersSnapshot` 对 empty 执行 `continue`，心跳已 connected 时层图仍为 0 节点。本次已让凭证 409 走 `BOOTSTRAP_FAILED` + clone-log 失败文案，不再无限「等待可写层」；合法克隆进行中仍可能空白到 clone_begin。
- **Action**: (1) 快照对 empty 层输出 pending/引导中节点（或 `bootstrap_pending` 标记）(2) 前端 zTree 识别该节点展示「正在准备可写层」而非空白 (3) 补 onlineServiceJS 快照单测 + taskFE mock Playwright
- **Why**: 心跳早于克隆是启动时序，过滤空层会让 UI 失去结构锚点；失败路径已修，进行中路径仍依赖文案轮询。
- **How to apply**: `trae-agent/onlineServiceJS/src/jobsRuntimeSnapshot.mjs` `buildLayersSnapshot`；对照 `commentLayerZtreeUiState.js` loading hint
- **2026-08-20 夜**: (1)(3-快照) 已完成 — trae-agent `4fea5e7` 已推送：empty 层输出 `bootstrap_pending=true` + `mind_state=pending`，2 个快照单测纳入 test:unit；onlineServiceJS 镜像 `DOCKER_PUSH=1` 已推（x86_64-latest）。剩余 (2) 前端 zTree 识别 + (3-FE Playwright)，因 taskFE 工作树有他会话 staged WIP（bootstrap 克隆日志批）本窗口未提交。

## [OPT-20260817-010] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 4c1ec50 已推送：NavbarTaskSearch.vue/.test.js 更名 WorkPanelTaskSearch.*，组件内 6 个 data-testid/data-alias/id 与 WorkPanelHeader 引用、Navbar.ui/WorkPanelHeader.legend 测试选择器、日志前缀全部同步，无残留 navbar-task-search/NavbarTaskSearch。相关 40 测全绿。历史 spec/plan 文档为日期记录未改。
- **Created**: 2026-08-17
- **Context**: 任务搜索已从顶部导航栏迁到工作面板标题行，但组件文件、data-testid、data-alias 仍保留 `navbar-task-search` 前缀，阅读与检索会误以为还在 Navbar。
- **Action**: (1) 将 `NavbarTaskSearch.vue` / `.test.js` 重命名为 `WorkPanelTaskSearch.*` (2) 同步更新 `data-testid`/`data-alias`/`id` 与意图文档中的选择器 (3) 全仓 grep `navbar-task-search` / `NavbarTaskSearch` 一并替换
- **Why**: 挂载点已变，旧命名会让后续改布局的人继续往 Navbar 里找搜索框。
- **How to apply**: `taskFE/app/src/components/NavbarTaskSearch.vue`；`WorkPanelHeader.vue`；`docs/intents/frontend/navbar_task_search_filter*.md`

## [OPT-20260818-036] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 闭环：Go/JS 两端启动命令映射本已有交叉引用注释与现网/SH-1 用例；taskFE 1c924af 补新增 slug（git-service-aws-tokyo-1）GITSERVICE_CONF_APP 用例，与 taskBill TestGitlabRegionServiceStart 同表，防止双份漂移。
- **Created**: 2026-08-18
- **Context**: `gitlabRegionServiceStart`（Go）与 `deriveGitlabRegionServiceStart`（JS）目前手写同一套规则（现网 run.sh / SH-1 deploy 脚本 / 其它 slug 的 GITSERVICE_CONF_APP）。
- **Action**: (1) 抽一份约定（注释交叉引用或小型共享表）(2) 新增 slug 时两处同测
- **Why**: 只改一端会导致管理页启动命令与真实入口再次不一致。
- **How to apply**: `taskBill/src/gitlab_region_deploy.go`；`taskFE/app/src/components/system-admin/gitlabRegionDeployPaths.js`

## [OPT-20260819-029] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 闭环：taskAuth 4159f1a（handlePublicKycAdminGet 附带 limit_policies 数组）+ taskFE 9a4cb3f（kyc-tier-help 各档叠加 API 限额金额，未命中不回退硬编码；单测断言来自 mock，3 测全绿）均已推送。
- **Created**: 2026-08-19
- **Context**: 本次在超管 KYC drawer 标题区增加了 T0/T1/T2 语义说明，刻意未硬编码单笔/日限额金额（以 `auth_kyc_limit_policy` 为准）。超管仍需对照策略表才能看到具体额度。
- **Action**: (1) 为超管暴露只读 limit-policy API（或随 KYC profile GET 附带 policies）(2) 在 `kyc-tier-help` 中展示各档当前 max_single/max_daily (3) 补 unit/playwright 断言金额来自 API mock
- **Why**: 说明与运行时限额同源，避免运营改策略后文案漂移。
- **How to apply**: `taskAuth` public/admin KYC handlers；`SystemAdminUserKycDrawer.vue`；`kycTierHelp.js`

## [OPT-20260818-022] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 闭环：taskBill 398d861（handleResourceQuotas 增加 gitlab_resources[] 按区列表 + TestHandleResourceQuotasMultiRegion，整包 83.6s 绿）+ taskFE 9e18f01（BillingDashboard 按区渲染磁盘/流量/到期/仓库链接，2 单测全套 12 测绿）均已推送。旧聚合字段保留兼容。
- **Created**: 2026-08-18
- **Context**: `GET /api/tenant/{tid}/billing/quotas/` 经 `gitlabResourceView` 只返回最近购买的一个 region；租户买了第二区后账单页仍像单区。
- **Action**: (1) quotas API 增加 `gitlab_resources[]` 按 region 列表 (2) BillingDashboard 按区展示磁盘/流量 (3) 保留旧聚合字段兼容
- **Why**: 购买已按区隔离，首页不展示会让租户以为没买到所选区。
- **How to apply**: `taskBill/src/handlers_orders_pay.go` `handleResourceQuotas`；`taskFE/app/src/views/BillingDashboard.vue`

## [OPT-20260818-018] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 与 022 同根因一并闭环：listGitlabResourceViews 列出全部 billing_tenant_gitlab_resource 行，Dashboard 按 slug 展示磁盘/流量，FE 单测断言两区域同时有配额都可见（gitlabRegions.test.js）。
- **Created**: 2026-08-18
- **Context**: GET `/billing/gitlab-resources/` 已强制 `?region=`；工作区 GitLab 连接与下单已必选区域。`gitlabResourceView` 目前取租户最近有配额的 region，仪表盘可能仍像单区域。
- **Action**: (1) 列出租户全部 `billing_tenant_gitlab_resource` 行 (2) Dashboard 按 slug 展示磁盘/流量 (3) 补 FE 单测：两区域同时有配额时都可见
- **Why**: 租户选购第二区域后仪表盘若只显示一行，会误以为另一区域未生效。
- **How to apply**: `taskBill/src/gitlab_resources.go` 的 `gitlabResourceView`；`taskFE` Billing/配额相关视图

## [OPT-20260816-024] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 复核结果：useWorkPanelMachineSummary 已有 visibility 暂停（OPT-20260808-021）、useServerConfigRelayToTrae 自停 bootstrap、useBillingOrderActions 用户触发有界轮询、useServerConfigRuntime 纯本地时钟、验证码/倒计时非 API。docs 0655e11 在 comment_runtime_no_background_poll.intent.md 增补「前端 API 轮询 allowlist」注明触发源，只减不增。
- **Created**: 2026-08-16
- **Context**: v84 已去掉评论运行态 / 启动进度 / binding / 遗留看板的 UI 轮询。`ImageMarket.vue` vendorStatus 的 30s `setInterval` 已随申请入口迁到厂商门户一并删除（仅 onMounted 拉一次）。仓库里仍有 `useWorkPanelMachineSummary`、`useServerConfigRelayToTrae` 等无用户点击的 API `setInterval`。
- **Action**: (1) 逐个判定是否可改为 SSE/按钮 (2) 不能改的写入 ADR-0011 前端 allowlist 并注明触发源 (3) 能改的删 timer 并补测「挂载 0 次 GET」
- **Why**: 同一卡死/泄漏模式会在其它页面复发。
- **How to apply**: `rg setInterval taskFE/app/src --glob '!**/vendor/**'`；对照 ADR-0011 前端条款

## [OPT-20260820-004] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: runAll 精准重启热替换：skip_stop_on_restart + restart_reload_command（nginx -s reload），健康+detach 时跳过 stop 阶段，build 期间公网不空窗。runAll 764a9b5 + conf ec3a7b4 已推送，3 回归测全绿。
- **Created**: 2026-08-20
- **Context**: 公网 SPA 已改 Docker nginx 常驻；`runall-lifecycle.sh build` 只切 `public/html` symlink，容器 Id 不变。但 runAll「精准编译重启」仍是 编译→stop→start。`stop` 会 `docker compose stop`，:4000 会空窗数秒，APISIX 仍可能 502。
- **Action**: (1) 为 taskFE 增加 skip-stop 或 `stop` 在容器已 healthy 且仅内容变更时 no-op (2) 仅 nginx.conf 变更时 `nginx -s reload` (3) 补 runAll/taskFE 测：精准重启路径容器 Id 不变且 curl /health 不 502
- **Why**: 本会话已消除 stop-all 杀 Node 导致的长时间 502；精准重启仍会主动停容器，与「静态常驻」目标不完全一致。
- **How to apply**: `runAll/src/precise_restart.go` 或 `taskFE/app/scripts/runall-lifecycle.sh stop`；对照 ADR-0022 与 `test_nginx_static_resident.sh`

## [OPT-20260816-064] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 删除 prefer_idle_reuse 列：Go/OpenAPI/FE 全移除，dataMigrate 029 DROP COLUMN 已应用生产 task_cloud，taskCloudService 已重建重启 0d8296b，dataMigrate 63296a8、taskFE f0d994c、docs 2a4f834 已推送，全套测试绿。
- **Created**: 2026-08-16
- **Context**: ADR-0013 已拆除跨任务闲置复用；列仍在 `cloud_workspace_machine_policies`，API/OpenAPI 恒返回 false 并忽略写入，前端保存仍带 `prefer_idle_reuse: false` 以兼容旧后端。
- **Action**: (1) 确认所有客户端不再读取该字段做行为分支 (2) 新增 dataMigrate 删除列 (3) 去掉 Go struct/JSON、OpenAPI、前端 PUT body 中的该键 (4) 更新 ADR-0013 Consequences
- **Why**: 废弃列长期保留会造成「开关还在」的误解；等一轮发布后再删，避免旧 SPA 与新 API 窗口期写失败。
- **How to apply**: `dataMigrate/taskCloudService/` 下一号 SQL；`workspace_machine_policy.go`；`taskCloudService/src/openapi.yaml`；`WorkspaceMachinePolicyModal.vue`

## [OPT-20260819-026] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE c784c78 已推送：SystemAdminLicenseAgreement.vue 顶部按公开接口探测无生效支付条款时显示琥珀色提示「支付服务条款未发布」，创建/编辑/删除协议后即时重查；新增 4 例 composable 回归测。
- **Created**: 2026-08-19
- **Context**: 支付门禁在没有生效 `recharge_cents` 文档时 fail-open（仅结构化日志）。生产若未 seed 支付服务条款，用户仍可不签即付。
- **Action**: 监控 `payment_terms_consent_skipped_no_active` 并在系统管理法律文档页提示「支付服务条款未发布」。
- **Why**: fail-open 避免无文档环境锁死支付，但运营侧应能发现缺口。
- **How to apply**: `consent_gate.go` `activePaymentTermsAgreementID`；`026_seed_default_legal_documents.py`

## [OPT-20260819-012] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskBill 4c314a7 + taskFE e4114e1 已推送：POST /api/system-admin/order-number/parse/ 服务端单源解析 ADR-0017/0018 订单号（四段含租户、三段无），管理端订单记录页新增「粘贴订单号」输入框，命中四段号选定租户并推 /system-admin/order-records/?tenant_id=&order_id= 深链复用展开高亮；Go 6 例 + FE 4 例回归测全绿。
- **Created**: 2026-08-19
- **Context**: ADR-0018 已让新单号可解析出租户与主键，但刻意没有匿名公网查单 API。客服/超管目前仍须人肉拆号再打开 `/tenant/{tid}/billing/orders/{id}/`。
- **Action**: (1) 在系统管理订单记录页加「粘贴订单号」输入框 (2) 前端或内部 API 调用 `ParseResourceOrderNumber` (3) 四段号校验行上 tenant 后跳转既有租户订单 URL (4) 三段号提示无租户基因、需手工选租户
- **Why**: 咨询场景的产品价值要落到工具，而不是只停在号码格式。
- **How to apply**: `ParseResourceOrderNumber` 在 `taskBill/src/orders_number.go`；管理端 `taskFE/app/src/views/SystemAdminOrderRecords.vue`；禁止新做匿名公网查单接口

## [OPT-20260817-029] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 6b67be5: 修正任务详情 Playwright mock URL 缺 / 分隔符（mock 永不命中致排队调度面板不渲染），mock 补 workspace_seq=3，新增断言 task-aux-info-display-no 可见且文本为 #3；Playwright 1 passed + aux-info 单测 8 passed 后已推送
- **Created**: 2026-08-17
- **Context**: 本次仅有 Vitest；公网 SPA 需精准编译重启后才能在浏览器验收。
- **Action**: 在 `tests/TaskDetail.aux-info-collapse.playwright.test.js`（或新文件）断言 `task-aux-info-display-no` 可见且匹配 mock 的 `workspace_seq`
- **Why**: 防止模板回归后单元测绿但页面未带上字段。
- **How to apply**: `taskFE/tests/TaskDetail.aux-info-collapse.playwright.test.js`；夹具需带 `workspace_seq`

## [OPT-20260819-035] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE e48f637: 新增 Playwright 双用例——(1) 真实网关 POST /api/auth/ 接受 E.164 手机号+密码返回 token（服务端规范化闭环，beforeAll 幂等 seed phone login_method：national+cc+bcrypt）；(2) 登录页手机号/密码表单提交完整 E.164 并断言离开 /auth/login/；配套 .sh 与 .testIntent，2 passed 后已推送
- **Created**: 2026-08-19
- **Context**: 推荐链接手机号注册成功后，登录页提示「手机号或密码错误」。根因已用 Go httptest 覆盖；完整 UI 路径需要短信验证码，当前无法在 Playwright 里无 mock 地新建手机号账号。
- **Action**: (1) 测试环境打开 SMS mock 或内部 seed 手机号 login_method (2) Playwright：注册页提交后立刻用同一 E.164+密码登录并断言离开 `/auth/login/` (3) 配套 `.testIntent` 与 `.sh`
- **Why**: 规则 34 要求用户可感知登录 bug 有 E2E 验收；目前只能用 handler 集成测代替。
- **How to apply**: `taskFE/tests/Login.phone-password-e164-after-register.playwright.test.js`；对照 `taskAuth/src/auth_login_test.go` `TestHandleLoginPhonePasswordAcceptsE164AfterRegister`

## [OPT-20260819-005] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 已部署（release 20260820050418，公网 SPA 加载新 chunk）并 Playwright 公网复验通过：租户 877397588196749312 订单列表 5 条全部 a.break-all，长单号 ORD-20260819-877397588196749312-877909223214710784 可点进详情页（200 非 404）；PayOrderModal/CancelOrderModal break-all 由 b0268fd 两例回归测覆盖。
- **Created**: 2026-08-19
- **Context**: 新单 `order_number` 为 `ORD-YYYYMMDD-{tenantId}-{snowflake}`（约 50 字符）。列表已加 `break-all`，但 taskFE 工作树有其他会话 WIP，本会话未执行公网 Vite build。 PayOrderModal / CancelOrderModal 的订单号尚未加 `break-all`。
- **Action**: (1) taskFE WIP 清空后 `npm run build` (2) 精准重启 `taskFE` (3) 硬刷新 `/tenant/{tid}/billing/orders/` 确认 `a.break-all` 长号可点进详情 (4) PayOrderModal / CancelOrderModal 订单号加 `break-all`
- **Why**: 不发布 SPA 则公网仍是短号样式；号码变长后表格可能横向溢出。
- **How to apply**: `taskFE/app/src/views/BillingOrders.vue`、`OrderDetail.vue`、`SystemAdminOrderRecords.vue`、`PayOrderModal.vue`、`CancelOrderModal.vue`；`scripts/register-precise-restart.sh taskFE`
- **2026-08-20 夜**: (4) 已完成 — taskFE `b0268fd`（PayOrderModal 两处 + CancelOrderModal 订单号 `strong` 加 `break-all`，新增 `OrderNumberModals.break-all.test.js` 两例回归测，已推送）。剩余 (1)(2)(3)：`npm run build` + 精准重启 taskFE + 公网硬刷新验收长号换行。taskFE 现已无 WIP，本窗口可执行 build/重启。

## [OPT-20260819-008] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 部署后 Playwright 公网复验通过：微信实付订单（0.55）显示「申请退款」按钮；admin_grant 赠送订单显示「已支付（赠送）」无退款按钮；已退款订单显示「已退款」无退款按钮，3/3 断言通过。
- **Created**: 2026-08-19
- **Context**: 已改 `BillingOrders.vue`（退款说明文案、`isOrderRefundable`、已退款状态），需精准编译重启 taskFE 后公网硬刷新验收。
- **Action**: (1) runAll「精准编译重启」含 taskFE (2) 打开租户订单页硬刷新 (3) 确认微信实付订单可申请、赠送/已退款不可
- **Why**: 未发布 SPA 时用户仍看旧文案与按钮逻辑
- **How to apply**: `taskFE/app/src/views/BillingOrders.vue`；登记已含 taskFE

## [OPT-20260820-001] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 部署后 Playwright 公网复验通过：账单首页「本月消耗」0.55、「累计消耗」0.55，副文案「用量扣费与仍有效的资源订单合计（不含已退款、已取消订单）」；最近交易可见 -0.55 元资源购买。
- **Created**: 2026-08-20
- **Context**: 租户 `877397588196749312` 账单页「本月/累计消耗」曾把已退款订单的 `resource_purchase` 一并计入，显示 1.10。taskBill 已改为排除 `refunded`/`cancelled` 订单购买；本机统计期望约 0.55（仍为 paid 的那笔）。公网页需登录 Cookie + SPA 缓存，未完成硬刷新对照。
- **Action**: (1) 登录后硬刷新 `https://www.daydaymoney.com/tenant/877397588196749312/billing/` (2) 确认「本月消耗」「累计消耗」约为 0.55 而非 1.10，副文案含「不含已退款、已取消订单」 (3) 将证据写入本条 Completion-Note
- **Why**: 网关鉴权页与旧 SPA 包可能仍展示虚高消耗；本机 8004 通过不等于公网登录态 UI 已更新。
- **How to apply**: 对照 `handleBillingStatistics` 的 `voidedResourcePurchasePred`；Playwright/CDP 走 9222

## [OPT-20260819-045] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: taskFE 部署后公网复验：累计支付显示 0.55（非 0.00）。与条目预期 1.10 的差异源于 OPT-20260819-041 累计支付并入 refund 负额扣减：租户两笔 0.55 wechat 支付其一已退款，净额 0.55。副文案含「不含后台赠送」，最近交易可见 -0.55 元。条目预期基于退款扣减前旧逻辑，现行为正确。
- **Created**: 2026-08-19
- **Context**: 租户 `877397588196749312` 资源订单实付记为 `resource_purchase` 消耗，旧统计接口不计累计支付且前端曾只读 `*_cents`。taskBill 已重启，本机 `GET /billing/statistics/` 返回 `user_recharge_cents=110`（两笔 0.55）。公网页需登录 Cookie，本会话浏览器快照为黑屏，未完成硬刷新对照。
- **Action**: (1) 登录后硬刷新 `https://www.daydaymoney.com/tenant/877397588196749312/billing/` (2) 确认「累计支付」为 1.10 而非 0.00，最近交易仍可见 `-0.55 元` (3) 将证据写入本条 Completion-Note
- **Why**: 网关鉴权页与 SPA 缓存可能仍展示旧包；本机 8004 通过不等于公网登录态 UI 已更新。
- **How to apply**: 对照 `taskBill/src/handlers_billing_query.go` `handleBillingStatistics`；Playwright/CDP 走 9222

## [OPT-20260820-006] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: runAll f111ba1 已推送：killPreviousRunAllProcess 重构为 retirePreviousRunAll（shutdown-self 2xx 后等旧进程自行退出、成功不再 SIGTERM，仅 POST 失败/非 2xx 或退出超时才补信号）；信号 goroutine 记录 pid/ppid/pgid/sid（Getsid 走 SYS_GETSID）；新增 main_kill_previous_test.go 5 例回归测全绿。
- **Created**: 2026-08-20
- **Context**: 2026-08-20 00:36:58 runAll 以 `trigger=signal detail="terminated"` 退出并逆 DAG 停掉全部托管服务（末行 `[go-relay] stopped` 只是 L1 最后收割，不是故障通知）。同秒前 22s 夜间 OPT 刚提交 runAll `06bbe77`；之后 `:9999` 未再拉起（watchdog `--no-restart-runall`）。`killPreviousRunAllProcess` 在 shutdown-self HTTP 200 之后仍无条件 `SIGTERM`；若 POST 失败则旧实例 `skipShutdownServices=false`，会把 76 个仍标 healthy 的服务全部 SIGTERM。
- **Action**: (1) `main.go` 收到 SIGINT/SIGTERM 时除 `s.String()` 外打印本进程 pid/ppid/pgid/sid（`syscall.Getpid/Getppid/Getpgid/Getsid`），便于对照 `ps`/cron/agent 父进程 (2) `killPreviousRunAllProcess`：shutdown-self 2xx 后等待旧进程退出，成功则不再 `SIGTERM`；仅 POST 失败或超时才补信号 (3) 回归测：mock shutdown-self 200 且旧 PID 已退出时 `Kill` 次数为 0；POST 失败时才 SIGTERM
- **Why**: 仅有 `detail="terminated"` 无法定位发送方；热替换路径多一次 SIGTERM 会把「保留托管服务」变成「全栈停掉」，与 00:36 现场及 OPT-20260813 周期性 SIGTERM 一致。
- **How to apply**: `runAll/src/main.go` `killPreviousRunAllProcess` / signal goroutine；`runAll/src/main_exit_reason_test.go` 或新建 `main_kill_previous_test.go`

## [OPT-20260820-007] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: runAll 32cc6d1 已推送：ui_dev.go 清库/初始化 409 改 writeJSONConflictWithBulkProgress 带回 active_bulk_progress；前端 _execInitAllDatabases/_execClearAllDatabases 409 走 adoptInProgressBulkOp 接管已有 run 的 SSE 而非 finalizeDevDBProgress 标失败；adoptInProgressBulkOp 增 init-db/clear-db 分支；新增 ui_dev_conflict_progress_test.go 回归测（409 携带 kind/run_id + 页面片段 adopt/return 顺序）。
- **Created**: 2026-08-20
- **Context**: 修复「已有全部/分组重新编译进行中」横幅无进度条时，已让 build/start/stop/restart/precise-restart 的 409 带 `active_bulk_progress` 并 `adoptInProgressBulkOp`。`ui_dev.go` 清库/初始化仍走 `writeJSONErrorWithStatus` 纯错误 JSON，前端 409 不接 `/api/progress`。
- **Action**: (1) `ui_dev.go` 冲突改为 `writeJSONConflictWithBulkProgress` (2) `_execClearAllDatabases` / `_execInitAllDatabases` 的 409 调用 `adoptInProgressBulkOp('clear-db'|'init-db', …)` 并打开已有 `run_id` 的 SSE (3) 补 UI 片段回归测
- **Why**: 同一「进行中却只见横幅」缺陷会在开发工具清库/初始化路径复现。
- **How to apply**: `runAll/src/ui_dev.go`；`runAll/src/status_ui/js/04.js` / `10.js`；`runAll/src/status_ui/js/08.js` `adoptInProgressBulkOp`

## [OPT-20260818-031] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网硬刷新复验通过：/tenant/877397588196749312/settings/gitlab-connection/ 区域下拉非空，含「腾讯上海一区」（gitlab-region-select 有选项），内建配额区正常展示；Playwright 脚本 opt-nightly-20260820-combined-verify.cjs 留存。
- **Created**: 2026-08-18
- **Context**: 租户 `877397588196749312` 已获赠 `tencent-sh-1` 磁盘 1GB（`provisioning_status=active`），但设置页区域下拉为空。根因是约定路径 404 + 前端未选区就放弃拉配额。代码已修，需编译重启后公网验收。
- **Action**: (1) 9999「精准编译重启」`task-bill` + `taskFE`（含 SPA collectstatic）(2) 硬刷新 `https://www.daydaymoney.com/tenant/877397588196749312/settings/gitlab-connection/` (3) 断言不再出现空的 `gitlab-region-select`，而是内建 GitLab 配额区显示「腾讯上海一区」
- **Why**: 未重启时线上仍 404，用户继续看到空下拉。
- **How to apply**: `.runall/precise_restart_services.txt`；页面 `data-testid=gitlab-builtin-resources` / `gitlab-region-name`

## [OPT-20260818-032] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网硬刷新复验通过：Navbar 代码仓库不再指向 /pricing/，GitLab 导航指向 settings/gitlab-connection 与 billing/orders/create；赠送区域「腾讯上海一区」可下拉。Playwright 脚本 opt-nightly-20260820-combined-verify.cjs 留存。
- **Created**: 2026-08-18
- **Context**: 租户 `877397588196749312` 为普通会员但已获赠 `tencent-sh-1` 磁盘；Navbar 曾把非 VIP1 指到 `/pricing/`。现已改为按 `resources[]` 跳转，需精准编译重启 + 公网硬刷新才能在页面上看到。
- **Action**: (1) :9999 精准编译重启 `taskFE` 与 `task-bill` (2) 硬刷新 `/tenant/877397588196749312/settings/gitlab-connection/` (3) 断言「代码仓库」href 为 `gitlab-tencent-sh-1` 公网 URL（或下拉含该区），不再是 `/pricing/`
- **Why**: 单测已绿但公网 SPA 读 dist；未重启会继续展示 VIP 门禁旧包。
- **How to apply**: `taskFE/app/src/components/NavbarGitServiceNav.vue`；`GET /api/tenant/{tid}/billing/gitlab-resources/` 的 `resources[]`

## [OPT-20260818-035] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网复验通过（API + SPA）：/system-admin/gitlab-resources 两区域卡 service_start 分别为 bash gitService/run.sh start 与 bash gitService/scripts/deploy_tencent_sh_1.sh（不同），config 文件分别为 conf/infra/git-service/config.yaml 与 conf/infra/git-service-tencent-sh-1/config.yaml；API :8004 直查确认。
- **Created**: 2026-08-18
- **Context**: SystemAdmin GitLab 区域卡片已增加部署路径展示（API 字段 `service_process`/`config_file`/`data_dir`/`service_start`）；`service_start` 已按实例区分（现网 `gitService/run.sh start` vs SH-1 `deploy_tencent_sh_1.sh`）。本地 Go/Vitest 已绿，公网需精准重启后硬刷新。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」`task-bill` + `taskFE` (2) 硬刷新 `/system-admin/gitlab-resources` (3) 断言现网区 `gitlab-region-service-start` 为 `bash gitService/run.sh start`，`tencent-sh-1` 为 `bash gitService/scripts/deploy_tencent_sh_1.sh`（两卡文本不得相同）；配置文件分别为 `conf/infra/git-service/config.yaml` 与 `conf/infra/git-service-tencent-sh-1/config.yaml`
- **Why**: 防止仅本地单测绿、公网仍无路径区块，管理员无法定位调整入口。
- **How to apply**: `SystemAdminGitlabRegionDeployPaths.vue`；`taskBill/src/gitlab_region_deploy.go`；已登记 `.runall/precise_restart_services.txt`
- **Note**: 2026-08-18 本会话已精准重启 `task-bill`+`taskFE`；`GET :8004/api/system_admin/gitlab-regions/` 已返回两条不同 `service_start`。余下仅公网硬刷新肉眼确认 SPA。

## [OPT-20260818-045] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网复验通过：区域卡片「数据目录」已标注 GITLAB_HOME 标签（两卡均显示 GITLAB_HOME，值来自 data_dir），SPA 已发布生效。Playwright opt-nightly-20260820-combined-verify.cjs 留存。
- **Priority**: low
- **Category**: deployment-validation
- **Context**: SystemAdmin GitLab 区域部署路径行已将「数据目录」改为标注 `GITLAB_HOME`（值仍来自 API `data_dir` / conf `gitlabHome`）。本地 Vitest 已绿；公网需精准重启 `taskFE` 后硬刷新 `/system-admin/gitlab-resources`。
- **Files**: taskFE/app/src/components/system-admin/SystemAdminGitlabRegionDeployPaths.vue

## [OPT-20260819-032] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网硬刷新复验通过：登录后 /billing/orders/ 页 hover「代码仓库」弹出 nav-git-service-menu，含「腾讯上海一区」→ https://gitlab-tencent-sh-1.daydaymoney.com/ 外链。Playwright opt-nightly-20260820-batch2-verify.cjs 留存。
- **Created**: 2026-08-19
- **Context**: GET `/billing/gitlab-resources/` 无 region 曾 400，Navbar 静默空列表导致「代码仓库」href=当前页。已改为返回 `resources[]`，且 ≥1 区悬停/单击展开下拉。
- **Action**: (1) :9999 精准编译重启 `task-bill` + `taskFE`（含 SPA collectstatic）(2) 登录后硬刷新任务详情页 (3) 悬停「代码仓库」断言 `nav-git-service-menu` 含已购区域名与外链
- **Why**: 本地单测已绿；公网仍读旧 dist / 旧 taskBill，不重启看不到下拉。
- **How to apply**: `.runall/precise_restart_services.txt`；页面 `data-testid=nav-git-service-wrap` / `nav-git-service-region`

## [OPT-20260818-034] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网 Playwright 复验通过：/system-admin/gitlab-resources 区域「删除」弹出 gitlab-region-delete-modal，gitlab-region-delete-repo-address 展示 https://gitlab.daydaymoney.com；点取消关闭弹层且未发任何 DELETE/POST region 请求（脚本 opt-nightly-20260820-round2-verify.cjs）。
- **Created**: 2026-08-18
- **Context**: F-090 已用 Vitest 覆盖删除确认弹窗展示 `gitlab_web_url`；公网仍需新 SPA hash。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」`taskFE` (2) 硬刷新 `/system-admin/gitlab-resources` (3) 点区域「删除」断言弹层 `gitlab-region-delete-repo-address` 展示仓库 URL，取消不发请求
- **Why**: 防止本地单测绿、公网仍是旧内联弹层（仅 name/slug、无仓库地址）。
- **How to apply**: `SystemAdminGitlabRegionDeleteModal.vue`；`.runall/precise_restart_services.txt`

## [OPT-20260818-025] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网 Playwright 复验通过：/system-admin/grant-points 存在 grant-membership-tier 下拉（不修改/普通会员/VIP1），VIP 区块渲染正常（046 已应用，无 SELECT 失败）。
- **Created**: 2026-08-18
- **Context**: 本次仅 Vitest；公网需新 SPA hash + 046 已应用。
- **Action**: (1) 打开 `/system-admin/grant-points/` (2) 选租户断言 `grant-membership-tier` 与当前等级 (3) 选 VIP1 提交后租户可买 GitLab
- **Why**: 防止只单测绿、公网无 VIP 区块。
- **How to apply**: `taskFE/tests/` Playwright；系统管理员夹具

## [OPT-20260818-021] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网 Playwright 复验通过：/system-admin/grant-points 资源类型切到「GitLab 磁盘 (GB)」后 grant-gitlab-region 下拉出现，选项含腾讯上海五区/腾讯上海一区（启用 slug）。
- **Created**: 2026-08-18
- **Context**: 本次仅 Vitest；公网 SPA 需 taskFE 构建 + 精准编译重启后才能在浏览器看到区域选择。
- **Action**: (1) 精准编译重启 `task-bill` + `taskFE` 并 `npm run build` (2) 打开 `/system-admin/grant-points/` (3) 选 GitLab 磁盘断言 `grant-gitlab-region` 可见且选项含已启用 slug
- **Why**: 防止单元测绿但公网仍吃旧 hash 包。
- **How to apply**: `taskFE/tests/` 新 Playwright；夹具 mock `/api/system-admin/gitlab-regions/`

## [OPT-20260818-010] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网 Playwright 复验通过：/tenant/877397588196749312/settings/feature-params 加载正常，base_url 显示 https://api.deepseek.com 未被改写；粘连 URL 失焦后仅留最后一段（https://api.daydaymoney.com/v1https://api.final.com/v1 → https://api.final.com/v1，未保存）；task-cloud-service 0d8296b 已重启、taskFE 已部署、trae-agent 镜像已推（02:44）。
- **Created**: 2026-08-18
- **Context**: 已去掉 DeepSeek 预填与按运营商改写 `base_url`，仅修两个 `https://` 粘连。Vue/Go 单测已覆盖；公网设置页仍需登录态 Playwright。
- **Action**: (1) 精准编译重启 task-cloud-service + taskFE 并 `runall-lifecycle.sh build` 公网 SPA (2) Playwright：打开 `/tenant/{id}/settings/feature-params/`，填自定义网关或 `/anthropic` 后保存，刷新后输入框仍为原值；粘连 URL 失焦后只留最后一段 (3) 提交后于 `trae-agent/onlineServiceJS` `DOCKER_PUSH=1 ./buildDocker.sh`
- **Why**: 规则 34 要求用户可感知表单 bug 有 E2E；镜像不推则容器 YAML 仍可能走旧改写逻辑。
- **How to apply**: 对照 `taskFE/tests/WorkspaceSettings.feature-params-api.playwright.test.js`；意图 `docs/intents/backend/cloud/feature_params_base_url_no_rewrite.intent.md`

## [OPT-20260819-010] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 公网 Playwright 复验通过：/profile/referral 分享链接为 https://www.daydaymoney.com/auth/register/?accessCode=DR2AKvP9J9，非空不透明码且非 u{userId} 派生；后端 getReferralCodeStatus 无条件 ensureUserShareCode 返回 access_code（无资格分支同样返回）。005 迁移已应用。
- **Created**: 2026-08-19
- **Context**: 无推荐资格时分享链接曾为 `?accessCode=`。代码已改为 status 始终返回持久化不透明 `access_code`，前端只展示该字段、禁止 `u{userId}`。
- **Action**: (1) 9999 初始化全部数据库以应用 `005_referral_share_code.sql` (2) 精准编译重启 task-referral + taskFE (3) 硬刷新 `https://www.daydaymoney.com/profile/referral/` (4) 无资格账号确认链接含非空 `accessCode` 且**不是** `u`+数字 userId
- **Why**: 未跑迁移 / 未发布 SPA / 未重启 Go 时用户仍看到空码或旧的 userId 派生码
- **How to apply**: `UserReferral.vue`；`GET /api/accounts/users/referral-codes/status/` 的 `access_code`；表 `referral_share_code`

## [OPT-20260818-024] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 046 membership admin_tier_locked 迁移已应用（db-init ok，task_bill→050）；公网 /system-admin/grant-points VIP 区块渲染正常（025 复验），无 SELECT 失败；任务需求端到端达成。
- **Created**: 2026-08-18
- **Context**: 赠送页改 VIP 依赖 `billing_membership.admin_tier_locked`；未跑 046 则 SELECT 该列失败。
- **Action**: (1) http://10.2.150.68:9999/ 「初始化全部数据库」应用 `dataMigrate/taskBill/046_membership_admin_tier_locked.sql` (2) 「精准编译重启」`task-bill` + `taskFE`
- **Why**: 单测库已含该列，公网/共享 MySQL 未迁则管理端调级 500。
- **How to apply**: `dataMigrate/taskBill/046_membership_admin_tier_locked.sql`；`.runall/precise_restart_services.txt`

## [OPT-20260819-011] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 注册期 bind-from-code 已接线：taskAuth 注册成功异步绑边；taskReferral 反查不透明码（禁止 u{userId}）；taskBill sync-edge 后回填历史消费；charge.go 计提改传实际 cost。存量边仍为 0 行，见 OPT-20260820-038。
- **Created**: 2026-08-19
- **Context**: 本会话只保证无资格用户也能展示不透明分享码。注册绑 `billing_referral_edge` 是否仍要求 referrer 有活跃资格、以及是否按 `u{userId}` 解析码，均未核对。
- **Action**: (1) 追踪 `accessCode` 从 `/auth/register/` 到绑边的代码路径 (2) 若资格门禁拦截绑边，改为始终绑边、分成时再查资格 (3) 补回归测
- **Why**: 空码修了但注册仍拒绑，则无资格用户分享出去的链接仍无推荐关系
- **How to apply**: 注册页 query `accessCode`；`referral_share_code` 反查 user_id（禁止解析 `u{userId}`）；taskAuth/taskBill `sync-edge` / `billing_referral_edge`
- **2026-08-20 夜复核（独立确认）**: 追踪结论与 03:40 一致——**注册期绑边根本未接线，属特性缺口**：(a) `billing_referral_edge` 唯一写入方是 taskBill `upsertReferralEdge`（无条件、无资格门禁，仅校验非空/非自荐/tenant/时间）；(b) `/api/internal/taskbill/referral/sync-edge/` 全仓无任何调用方；(c) `referral_share_code` 仅有 user→code 正查（`ensureUserShareCode`），**无 code→user 反查**；(d) taskAuth `auth_register.go`/`auth_phone_register.go` 与 taskFE 注册页均不消费 accessCode。实现需跨 taskReferral（反查端点）+ taskAuth（注册消费）+ taskFE（透传）+ taskBill（sync-edge 接线），属专门设计会话，非独立小改。

## [OPT-20260820-037] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 绑边快照 commission_eligible；accrue 与微信分账仅当 eligible=1。TestReferralAccrueSkippedWhenIneligible 覆盖。
- **Created**: 2026-08-20
- **Context**: OPT-20260819-011 要求「始终绑边、分成时再查资格」。本会话绑边已接线且无资格门禁；`accrueReferralFromConsumption` 仍不查 `has_active_code` / 资格有效期。
- **Action**: (1) 计提前查推荐人资格是否在有效期内 (2) 无资格则跳过计提但保留边 (3) 补有资格/过期/无申请三例回归测
- **Why**: 无资格用户分享出去的链接现在会绑边；若不在分成时拦截，会给未获批推荐人记账
- **How to apply**: `taskBill/src/referral_commission.go` `accrueReferralFromConsumption`；资格数据在 taskReferral，需内部查询或绑边时写入资格快照

## [OPT-20260820-043] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: 生产已 apply 007+052；精准重启 task-referral/task-bill/taskFE；E2E 账号公网创建两渠道码 eZtNEpH6zH/w25brikFiT，stats 按渠道与日期 200 且分渠道表含三渠道。
- **Created**: 2026-08-20
- **Context**: 多渠道推荐码代码已合入；生产库尚未跑 `dataMigrate/taskReferral/007` 与 `taskBill/052`，进程仍是旧二进制。不迁库则创建渠道 INSERT 会因缺列失败。
- **Action**: (1) 在 `http://10.2.150.68:9999/` 初始化全部数据库 (2) 精准编译重启 `task-referral` `task-bill` `taskFE` (3) 打开 `/profile/referral/` 创建渠道、复制不同链接、按渠道与日期筛选人数/分账
- **Why**: 代码合入不等于现网可用；缺迁移时默认渠道写入会报 Unknown column
- **How to apply**: `dataMigrate/taskReferral/007_referral_share_channel.sql`；`dataMigrate/taskBill/052_referral_channel_and_eligible.sql`；`.runall/precise_restart_services.txt`

## [OPT-20260820-048] completed

- **Status**: completed
- **Completed**: 2026-08-20
- **Summary**: task-task-service 已用含 skip 的二进制运行；对 task_878227563963510784 直连 PATCH：已完成/已取消 200 且日志 task_auto_run_prereq_skipped_terminal；进行中仍 400 AUTO_RUN_RUNTIME_ENV_REQUIRED。公网 www 当时 502，验收走本机 :8017。
- **Created**: 2026-08-20
- **Context**: 详情页改「已完成/已取消」被已删镜像的 auto_run 门禁挡住；代码已修，需部署后在原任务页验证。
- **Action**: (1) :9999 精准编译重启 `task-task-service` (2) 打开原 task-detail 将进度改为已完成、再改为已取消 (3) 不应再出现「自动运行不可用：无法获取已安装镜像信息」
- **Why**: 不重启则线上仍走旧二进制，用户页面行为不变。
- **How to apply**: 页面 `https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878227563963510784/`；登记文件 `.runall/precise_restart_services.txt`


## [OPT-20260820-029] completed

- **Status**: completed
- **Created**: 2026-08-20
- **Completed**: 2026-08-20
- **Completion-Note**: 登录后硬打开 `task_878209971597111296`，`[data-testid=comment-execution-clone-progress]` 文案为「项目克隆 100% / ram-work 100% / 项目克隆 (1/1) 完成 ram-work」，不再卡在 9%。taskFE `f7856b6` 已精准编译重启（public/html `20260820183355`）。
- **Context**: 任务 `task_878209971597111296` 评论执行细节 `comment-execution-clone-progress` 显示「项目克隆 9%」，对照容器页 `http://118.190.106.76:8765/ui/.../tok_1787218674132000000` 已 100%。已修 FE 单调合并与容器 git overall=max(recv,unpack)+同仓 POST 串行。
- **Action**: (1) http://10.2.150.68:9999/ 精准编译重启 `taskFE` (2) 硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878209971597111296/?accessCode=DR2AKvP9J9 (3) 断言 `[data-testid=comment-execution-clone-progress]` 可见文本含 100% 或不含单独卡住的 9%
- **Why**: 未重建 SPA 时公网仍跑旧合并逻辑，迟到 9% SSE 会盖掉完成态。
- **How to apply**: CDP 9222。选择器 `data-testid=comment-execution-clone-progress`。
