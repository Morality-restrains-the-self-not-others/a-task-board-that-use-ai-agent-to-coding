# Completed OPT Archive — 2026-08-22

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 41 条。
> 归档执行时间：2026-08-25T22:13:52+08:00

## [OPT-20260821-032] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskBill 6ae55de 已推送：下单写 user_id + 付款从 payment_pending 回填买家，回归测见 orders_buyer_userid_test.go
- **Created**: 2026-08-21
- **Context**: 生产 `billing_transaction` 中 4 笔 `resource_purchase` 有 3 笔 `user_id` 为 NULL（`billingBuyerUserID` 把 0/空当成缺失）。推荐绩效已用同账户反查兜底，源头仍应写购买人。
- **Action**: (1) 查 `billing_resource_order.user_id` 为 0 的路径；(2) 付款完成时写入真实买家；(3) 回归测禁止 resource_purchase 空 user_id。
- **Why**: 无账户关联流水时反查失效，被推荐人支付会再次显示 0。
- **How to apply**: `taskBill/src/order_payment.go` `billingBuyerUserID`；下单创建 `billing_resource_order` 的 handler。

## [OPT-20260821-033] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskBill 5eb1282 已推送：打款比例 min(微信 max_ratio, 本地政策 5%)，展示仍用微信结果；回归测 max_ratio=3000 时展示 30% 打款 5%
- **Created**: 2026-08-21
- **Context**: 推荐页分成比例已改为查询微信 `GET /v3/profitsharing/merchant-configs/{mchid}` 的 `max_ratio`（万分比）。该值是商户平台「最大分账比例」上限（常见默认 30%），不等于推荐人实际佣金。当前 `getCommissionRate()` 也走同一 SSOT，微信返回 30% 时会按 30% 打款。
- **Action**: (1) 展示继续用微信查询结果；(2) `markOrderForProfitSharing` / 计提把打款比例改为 `min(wechat_max, 本地政策 5%)` 或拆成独立配置；(3) 补回归测：max_ratio=3000 时展示 30%、打款仍 5%。
- **Why**: 把上限当佣金会超发分账，资金路径须 ≥ L3。
- **How to apply**: `taskBill/src/wechat_profit_sharing.go` `getCommissionRate`；`taskBill/src/wechat_profit_sharing_ratio.go` `resolveCommissionRate`。

## [OPT-20260821-035] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskBill 47835f5 已推送：分账走签名 client，回调回写真实 transaction_id 到 ledger provider_capture_id，executeProfitSharing 删假值；回归测见 wechat_profit_sharing_txnid_test.go
- **Created**: 2026-08-21
- **Context**: `wechatProfitSharingHTTP` 注释称复用 wechatClient，实际是未签名的裸 HTTP；`executeProfitSharing` 仍用 `wechat:` + 订单号 + `TODO` 当 transaction_id。比例查询已走签名 Client，打款路径还是假值。
- **Action**: (1) 分账 POST 改走 `wechatClient` 签名；(2) 用真实微信支付单号；(3) 删除 TODO 占位。
- **Why**: 未签名请求会被微信拒绝，分账永远落不到商户平台。
- **How to apply**: `taskBill/src/wechat_profit_sharing.go` `wechatV3Post` / `executeProfitSharing`。

## [OPT-20260821-021] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskCloudService 6094c52 已推送：mark-comment-terminal-released 与 markTerminalReleased 同套清字段（instance/public_ip/server_url/business_api_endpoint/vscode_url/error_reason），测试断言回填字段被清空
- **Created**: 2026-08-21
- **Context**: `markTerminalReleased` 已按任务批量清空评论级 `instance_id`/`server_url` 并置 `Released`。`handleInternalMarkCommentTerminalReleased` 仍只改 `terminal_released`+`last_runtime_status`，开机中途终态时 instance 可能已回填，看板仍可能按 instance 计「已启动」。
- **Action**: (1) 让 `mark-comment-terminal-released` 与 `markTerminalReleased` 同一套清字段（instance/public_ip/server_url/status=Released）；(2) 扩展 `TestInternalMarkCommentTerminalReleased` 断言 instance_id 为空。
- **Why**: 两条终态打标路径字段不一致，Starting 晚到 VM 的补偿窗口会再出现「已取消但绿环」。
- **How to apply**: `taskCloudService/src/terminal_release_migrate.go` `handleInternalMarkCommentTerminalReleased`；测例 `terminal_release_migrate_test.go`。

## [OPT-20260821-019] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskCloudService aaa4d6f 已推送：终态查找按 task_id 短 TTL 缓存（终态 5 分钟、非终态/错误 8 秒），心跳/job-stream 热路径不再每次打 taskTaskService；回归测见 inbound_terminal_guard_test.go
- **Created**: 2026-08-21
- **Context**: 取消任务后机器仍跑的补偿在每次 heartbeat / job-stream 同步调用 `POST /api/internal/tasks/terminal-kinds/`。进行中任务高频心跳会给 taskTaskService 带来额外 QPS。
- **Action**: (1) 在 `taskCloudService/src/inbound_terminal_guard.go` 为 `lookupTaskTerminalKindsFn` 加进程内短 TTL 缓存（建议 5–15s，按 task_id）；(2) 终态（cancelled/completed）可缓存更久；(3) 单测：同 task 第二次查找不发 HTTP、TTL 过期后重查。
- **Why**: 守卫必须同步查终态才能 410，但每个心跳都打 task 服务会放大 task 抖动；短缓存把热路径成本降到一次往返/窗口。
- **How to apply**: `inbound_terminal_guard.go` 的 `lookupTaskTerminalKind`；注意 fail-open 错误不要把空结果当成「非终态」长期缓存。

## [OPT-20260821-038] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: dataMigrate be00b3a + taskBill 05dbfb6 已推送；055 已应用生产 task_bill（1 已应用 53 跳过）；EXPLAIN 确认 payment_ref 等值查询 type=ref 走 idx_billing_resource_order_payment_ref，rows=1
- **Created**: 2026-08-21
- **Context**: 管理端交易单号查询对 `payment_ref` 做精确等值匹配，当前无独立索引；订单量小时可接受。
- **Action**: (1) 在 `dataMigrate/taskBill/` 增加 `payment_ref` 索引（可过滤空串）(2) 9999 初始化应用 (3) 用 `EXPLAIN` 确认 `WHERE payment_ref = ?` 走索引
- **Why**: 订单累积后按微信交易单号查询会退化成全表扫描。
- **How to apply**: `dataMigrate/taskBill/` 下一序号 SQL；`listOrdersByTradeNo` 的 `payment_ref = ?` 条件

## [OPT-20260821-020] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskEvents c3df522 已推送：serviceBase 缺省走 INFRA_HOST（同节点），TASK_CLOUD_SERVICE_BASE_URL 环境变量仍优先；回归测非 loopback-only + env 覆盖
- **Created**: 2026-08-21
- **Context**: `TASK_STATUS_CHANGED` 释放机器时 `list-by-task` 打 `http://127.0.0.1:8018`。taskCloudService 短暂不可达即 11 次重试后 DLT，任务已取消但机器继续跑（本会话根因）。
- **Action**: (1) 将 `TASK_CLOUD_SERVICE_BASE_URL` 默认改为 `${INFRA_HOST}` / conf `subdomains` 或 `conf/events/` 同步片段；(2) 禁止提交无注释的 `127.0.0.1:8018` 作为跨进程连接；(3) 回归：配置缺省时解析结果不是 loopback-only（或有「同节点」注释）。
- **Why**: 写死 loopback 在 cloud 进程未监听或拆节点时必然 connection refused，终态释放只能靠入站/对账补偿。
- **How to apply**: `taskEvents` cloudconfig/client.go 与 `conf/events/domain-events/`；对照 `.ai/01_project_constraints/39_network_topology_aware_config.md`。

## [OPT-20260821-008] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskChromePlugin 29fb481 已推送：悬浮球开关只写 storage，content 监听 floatBallEnabled onChanged 自更新；测试在 popup-layout/content/no-cross-tab-broadcast
- **Created**: 2026-08-21
- **Context**: 本会话去掉了登录/快捷键/过期的跨 tab 扇出；Popup 改「显示悬浮球」仍 `tabs.query({})` + `setFloatBallEnabled`。
- **Action**: (1) 改为只写 storage (2) content 用已有或新增 onChanged 显隐悬浮球 (3) 单测断言 popup 无 `query({})` 扇出
- **Why**: 与本目标同类的跨 tab 广播残留。
- **How to apply**: `taskChromePlugin/popup/popup.js` 悬浮球开关；`content/content.js` `setFloatBallEnabled`

## [OPT-20260821-025] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskFE 61d2cfb 已推送：PhoneRegister/EmailRegister 新增 traceId prop 挂字段级 data-traceId，Register.vue 透传 submitErrorTraceId；组件测断言见 EmailRegister.test.js / PhoneRegister.test.js / Register.test.js
- **Created**: 2026-08-21
- **Context**: 注册提交 400 已用表单级横幅挂 `data-traceId`。`PhoneRegister` 的 `#code-error`/`#phone-error` 与 `EmailRegister` 的 `#email-error` 仍只有 `text-error` 文案，字段级 API 错误无法从 DOM 提取 trace。
- **Action**: (1) 从父组件把本次提交 `traceId` 传入字段错误节点 (2) 绑定 `:data-traceId` (3) 补组件测断言属性
- **Why**: 用户可能只盯验证码输入旁的红字；横幅之外的字段错误仍无法走 trace-first 排障。
- **How to apply**: `taskFE/app/src/components/auth/PhoneRegister.vue`、`EmailRegister.vue`；对照 `.ai/01_project_constraints/24_frontend_error_data_trace_id.md`

## [OPT-20260821-039] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskFE 7882fd1 已推送：交易单号查询写入 ?order_number= 可分享 URL，applyFromRoute 读取自动查询；测例在 orderNumberPaste/deeplink test
- **Created**: 2026-08-21
- **Context**: 订单记录页查询命中后列表已过滤，但刷新/分享不会带上 `order_number`。
- **Action**: (1) 查询成功后 `router.replace` 写入 `?order_number=` (2) `applyFromRoute` 读取并自动查询 (3) 补 vitest
- **Why**: 客服无法把「查到这一单」的链接发给同事。
- **How to apply**: `SystemAdminOrderListPanel.vue`；`useSystemAdminOrderListDeepLink.js`

## [OPT-20260821-026] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: conf 988bd5f + taskAuth 412900c 已推送：taskAuth 改读 auth/task-auth/task-bill.yaml 片段（规则 29），env 优先；回归测见 bill_internal_secret_fragment_test.go
- **Created**: 2026-08-21
- **Context**: 本次 taskReferral 已用 `sync.manifest.yaml` + `task-bill.yaml` 加载账单密钥。`taskAuth/src/config_downstream_urls.go` 仍 `ReadAppConfig(..., "billing/task-bill")`，违反规则 29。
- **Action**: (1) 在 `conf/auth/task-auth/sync.manifest.yaml` pick `internalSecret` (2) `loadConfig` 改 `ReadAppFragment` (3) 单测 env 优先、片段回退
- **Why**: 跨服务直读 conf 会在拆分/同步遗漏时静默丢密钥，与本次 sync-edge 403 同类。
- **How to apply**: 对照 `taskReferral/src/config.go` `loadBillInternalSecret` 与 `conf/task-referral/sync.manifest.yaml`

## [OPT-20260821-022] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskTaskService 9a68af7 已推送：auto_run 校验 git_identity_id 归属（user+tenant），他人身份/跨租户 400；回归测见 comment_repo_identities_test.go
- **Created**: 2026-08-21
- **Context**: 创建/更新任务勾选自动运行时已强制每仓 `git_identity_id` 并写入【自动运行】评论 JSON；与评论 POST 一样尚未校验该 ID 属于当前用户/租户，伪造 ID 仍可能落库。
- **Action**: (1) 在 `ValidateGitIdentitiesForCreateAutoRun` 或 create/update 路径按 `user_id`+可选 `company_id` lookup 身份 (2) 评论 POST 运行身份走同一校验 (3) 补单测：他人 identity id → 400
- **Why**: 只校验非空不能阻止把别人的署名写进自动运行克隆/提交。
- **How to apply**: `taskTaskService/src/comment_repo_identities.go` `resolveAutoRunRepoIdentities`；已有 internal lookup `/api/internal/git-identities/lookup/`

## [OPT-20260821-015] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskCloudService eff02c4 已推送：git identity lookup 出站 http.NewRequestWithContext + ApplyOutboundHeaders 透传 X-Trace-Id；回归测断言出站携带 trace
- **Created**: 2026-08-21
- **Context**: 推送失败 trace `539d4fdc-53bf-49bf-9ecc-2270f92285d1` 的 Loki 只有 Cloud/Gateway/Auth，没有 task-task-service，因为 `realFetchUserCompanyGitIdentity` 新建 `http.Request` 未复制入站 `X-Trace-Id`。根因（缺 `X-Auth-User-Id: internal`）已修，日志缺口仍在。
- **Action**: (1) 给 `taskCloudService/src/git_repo_identities_prepare.go` 的 lookup 请求注入当前 ctx/`X-Trace-Id`（与 OPT-20260821-012 `ApplyOutboundHeaders` 对齐）(2) 同步其它 Cloud→Task 出站（`server_orphan_reconcile` tasks/exists）(3) 用新一次推送 trace 验证 Loki 能命中 `task-task-service`
- **Why**: 服务间 403/502 若对不上同一把钥匙，排障只能靠时间窗模糊搜，无法证明是 lookup 还是网关拒绝。
- **How to apply**: `realFetchUserCompanyGitIdentity`；`shareLib/tracelog` 出站头；禁止只打 `http_request` 而不带上游 trace_id

## [OPT-20260821-027] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskFE 52f3b1e：OAuth 绑定行移到自动运行下方（CreateTaskRepoOAuthSection.vue 新组件 + DOM 顺序测）
- **Created**: 2026-08-21
- **Context**: 创建任务弹窗已把「Git 提交身份」从项目仓行挪到「是否自动运行」下方。同为 `auto_run` 才显示的「OAuth 绑定」行仍留在仓行内、位于自动运行勾选之前，用户仍会先看到绑定提示再看到自动运行。
- **Action**: (1) 将 `create-task-repo-oauth-row` 从 `CreateTaskProjectBranchSection` 抽到自动运行区块附近；(2) 保持每仓 URL 可辨认；(3) 用 DOM 顺序测锁定在 `task-auto-run-field` 之后。
- **Why**: 与 Git 身份同一交互依赖：未勾选自动运行时不需要看到 OAuth 门禁 UI。
- **How to apply**: `taskFE/app/src/components/CreateTaskProjectBranchSection.vue` 的 oauth 行；对照本次 `CreateTaskGitIdentitySection.vue` 的挂载位置。

## [OPT-20260821-028] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskFE be4f729：首评 repo_identities 空时回退任务级身份（三态单测+接线静态断言）
- **Created**: 2026-08-21
- **Context**: 执行细节 summary 已回显评论 `repo_identities` 的 Git 身份。本页自动运行评论 `cmt_878583347226374144` 的 `repo_identities` 为空，摘要不显示身份，尽管创建任务时可能已选过仓库身份。
- **Action**: (1) 评论 `repo_identities` 为空时，用任务级 `task_repo_identities` / `taskRepoRows` 生成同一徽章；(2) 单测锁定「评论有身份优先、空则回退、两边都空不展示」。
- **Why**: 自动运行首条评论没有 composer 选择，用户仍期望在执行细节看到实际克隆身份。
- **How to apply**: `commentExecutionGitIdentity.js` 增加 fallback 参数；`TaskDetailCommentsSection.vue` 传入 `taskRepoRows`。

## [OPT-20260821-009] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskFE d220037：页脚 AGPL 第 13 条 Source 源码入口（PageFooter + 单测）
- **Created**: 2026-08-21
- **Context**: 仓库已改为 GNU AGPL-3.0。若通过网络向用户提供本软件（SaaS/控制台），第 13 条要求向远程交互用户提供对应完整源代码的获取途径。
- **Action**: (1) 在 Web 控制台页脚或关于页增加「Source」链接，指向本仓库公开源码归档 (2) 确认链接对未登录用户也可达 (3) 文档注明对应版本 tag/commit
- **Why**: 缺少该入口则网络提供修改版本时可能无法满足 AGPL 第 13 条。
- **How to apply**: `taskFE` 页脚/关于页；`LICENSE` 第 13 条；不要把源码入口藏在仅管理员可见页

## [OPT-20260821-030] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskReferral 6a98300 + taskFE a577dcf：推荐绩效自身支付 own_payments 流水表（Go 回归测 + 抽屉单测）
- **Created**: 2026-08-21
- **Context**: 系统管理用户「推荐绩效」抽屉「自身支付」目前只有标题计数，没有流水表。本次已把计数口径改成实付，但明细表仍缺。
- **Action**: (1) 后端 `getReferralPerformance` 增加 `own_payments[]`（时间、金额、points_source_type）；(2) `SystemAdminReferralPerformanceDrawer.vue` 在自身支付标题下渲染表格；(3) 补抽屉单测。
- **Why**: 管理员看到「1 笔 / 0.55 元」却无法核对是哪一笔订单。
- **How to apply**: `taskReferral/src/referral_performance_handlers.go`；`taskFE/app/src/components/SystemAdminReferralPerformanceDrawer.vue`

## [OPT-20260821-005] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskFE 273528d 已推送：loadRegions 加代际 token + 请求镜像键，换镜像立即发起新请求、过期结果丢弃（同镜像仍单飞）；watch(selectedImageId) 换镜像强制 scheduleFetchAvailableInstances；3 例回归测全绿。（1）拆 ≤500 行未做——2580 行文件跨域高度耦合，留专门会话。
- **Created**: 2026-08-21
- **Context**: 硬拦截落地时故意未改 `useServerConfigHardwarePanel.js`（2580 行）。`loadRegions` 仍合并进行中 promise，快速切换镜像可能把旧地域列表套到新镜像。
- **Action**: (1) 拆文件到 ≤500 (2) `loadRegions` 用 generation，过期结果丢弃 (3) 换镜像强制 `scheduleFetchAvailableInstances`
- **Why**: 跨架构切换时旧 promise 返回会短暂显示错误地域/实例。
- **How to apply**: `taskFE/app/src/composables/hardwarePanel/useServerConfigHardwarePanel.js` ~713；对照设计 `2026-08-21-image-switch-hardware-binding-design.md`

## [OPT-20260821-007] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 共享架构推断用例表已落地：shareLib edbb5bd（archfixture/testdata/instance_architecture_cases.json）+ taskProjectService e9ed92b + taskCloudService a8c45f5 + taskFE 35cebfa 均已推送；三侧表驱动测试读同一 JSON（extractCPUArchitecturesFromText + inferInstanceArchitecture），任一实现漂移即红。
- **Created**: 2026-08-21
- **Context**: 启发式在 `taskCloudService`、`taskProjectService`、`taskFE/containerImageArchitecture.js` 各有一份。
- **Action**: (1) 抽共享 fixture JSON (2) 三侧测例读同一组 (3) CI 比对
- **Why**: 再加规格族时容易只改一处。
- **How to apply**: 可放 `shareLib` 或 `docs/fixtures/instance-architecture-cases.json`

## [OPT-20260821-018] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 厂商保存/目录回填与安装回填的 autoRunStep+imageSkills 合并单次 OCI walk。taskAiProvider 6a73b90（infrastructure.ExtractAutoRunAndSkillsFromImage + scheduleAutoRunAndSkillsExtract，POST/PUT 与 catalog 双缺改合并路径，双文件同层/分居两层/仅 auto/皆无 + App 级单次 walk 双字段双事件回归测）；taskCloudService 06893b5（extractAutoRunAndSkillsFromImage + ensureInstalledImageExtracts 单次 walk 回填，ensure 单文件抽 apply* 共享，安装 handler 改合并路径，回归测双缺/已存在/仅缺 auto）。两仓整包 go test 全绿后已推送；已登记精准编译重启 ai-provider/task-cloud-service。
- **Created**: 2026-08-21
- **Context**: 厂商保存镜像时 `scheduleAutoRunExtract` 与 `scheduleImageSkillsExtract` 各自走一遍 registry layer blob，同一 image_url 拉两次。镜像市场上传延迟与 registry 带宽翻倍。
- **Action**: (1) 在 `taskAiProvider/infrastructure` 抽一次 walk 同时匹配 `/app/autoRunStep.md` 与 `/app/imageSkills.yaml`；(2) `runImageSkillsExtract` / autoRun 调度改为共用结果；(3) Cloud `ensureInstalledImage*` 再抽路径同样合并；(4) 补双文件同层/分居两层的回归测。
- **Why**: 上传绑定技能列表后每次保存都双倍拉层；合并后厂商保存与 catalog 回填延迟接近原来的 autoRun 单次成本。
- **How to apply**: 入口 `taskAiProvider/src/vendor_container_auto_run.go`、`vendor_container_image_skills.go`；抽取 `extract_auto_run_steps.go` / `extract_image_skills.go`；Cloud 对偶 `extract_auto_run_steps.go` / `extract_image_skills.go`。

## [OPT-20260821-031] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 用户列表「推荐人」列数据补齐。taskReferral 9cc6bae（新增 POST /api/internal/referral/referrers/lookup/ 批量返回 billing_referral_edge 边，IN 查询，内部密钥门禁，回归测批量/未知/空）；taskAuth b67df5c（handleSystemAdminListUsers best-effort 批量水合 referrer_code，展示名取 profile username → email/phone → user_id，客户端空输入/不可达不阻塞 + displayName 优先级 + 列表水合/无边为空回归测；FE UserListRow 已渲染该列无需改动）。两仓整包 go test 全绿后已推送。
- **Created**: 2026-08-21
- **Context**: `UserListRow` 展示 `user.referrer_code`，但 `handleSystemAdminListUsers` 从未填充该字段，列恒为「—」。超管难以从列表识别谁是推荐人，容易点开被推荐人看到空的向下推荐明细。
- **Action**: (1) 用户列表批量查询 `billing_referral_edge` 或 taskReferral 内部接口；(2) 返回 `referrer_user_id` / 渠道码；(3) 行上「推荐绩效」对推荐人加标记。
- **Why**: 列表按注册时间倒序，最新行几乎都是被推荐人；没有推荐人列就会误点空抽屉。
- **How to apply**: `taskAuth/src/handlers_system_admin.go`；`taskFE/app/src/components/UserListRow.vue`；跨服务读边须走 taskReferral/taskBill API（禁止 taskAuth 直连 task_bill）。

## [OPT-20260821-016] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 数据迁移 036 已落地并应用生产 task_cloud (applied=36)：trae-agent x86_64-latest / private_x86_64-latest 两行 target_architectures 由 [] / ["x86_64","unknown"] 回填为 ["x86_64"]；dataMigrate 7b50966 已推送，pre-push 迁移一致性门禁通过。
- **Created**: 2026-08-21
- **Context**: 租户 `877397588196749312` 的 `trae-agent` / `private_x86_64-latest`（id `878236807722987520`）在 `cloud_tenant_installed_images.target_architectures` 为 `[]`，另一条带 `"unknown"`。本次已在 lookup/校验读路径从 version 推断并具体化 400 文案，但库内仍是空数组。
- **Action**: (1) 扫描 `task_cloud.cloud_tenant_installed_images` 中 JSON 空数组或含 `unknown` 的行 (2) 用 version/name/image_url 中的 x86_64|amd64|arm64|aarch64 回填规范 ISA (3) 无法推断的行保持空并出监控/清单
- **Why**: 读时推断不能覆盖不走 lookup JSON 的路径，也容易在下次只读 DB 的脚本里再次当成「无架构」。
- **How to apply**: `dataMigrate/taskCloudService/` 幂等 UPDATE 或一次性运维脚本；对照 `taskCloudService/src/cpu_architecture.go` 的 `extractCPUArchitecturesFromText`

## [OPT-20260820-026] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskCloudService 19e596b + dataMigrate c5f790d 已推送：cloud_tenant_installed_images 加 saas_inbound_skill_version 列（037 已应用生产 task_cloud 37/37），安装时从目录归一化拷贝（v1→1），列表/详情 JSON 暴露，start-vm eventData 补字段 + UserData 渲染注入 SAAS_INBOUND_SKILL_VERSION export+docker -e；回归测覆盖归一化/注入/落库/JSON 暴露，整包测试全绿
- **Created**: 2026-08-20
- **Context**: ADR-0024 一期只把契约版本记在 `ai_provider_vendorcontainerimage.saas_inbound_skill_version`，未写入 UserData / 容器环境。容器进程无法自证实现的 inbound 版本。
- **Action**: (1) 在 Cloud/Credential 启动注入路径读取公开目录或厂商镜像字段 (2) 写入 env `SAAS_INBOUND_SKILL_VERSION` (3) 补回归测：镜像声明 v1 则容器 env 为 `1`
- **Why**: 运行时与登记声明脱节时，排障只能查库；注入后 Loki 与容器自检可对齐。
- **How to apply**: `taskCloudService` start-vm / UserData 渲染；公开目录字段 `saas_inbound_skill_version`；禁止改镜像 `version` 标签语义

## [OPT-20260820-024] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: db/scripts/check_shard_row_counts.py：information_schema.tables.TABLE_ROWS 监控哈希分片表族行数（ccb logs / job_execution_event 各 16 片，partition_metrics_exporter 只覆盖 RANGE 分区不覆盖独立分片表）；单片超 THRESHOLD_ROWS 默认 500 万退出码 1（cron 告警）、分片表少于期望片数 WARN；6 例自测绿，本机 dev task_cloud 实测两族各 16 片 exit 0；db 9b2ba82 已推送。
- **Created**: 2026-08-20
- **Context**: ADR-0023 首日用 16 张哈希表、主键仅 `id`，未做 `PARTITION BY RANGE(TO_DAYS(created_at))`。冷热分离门禁对 `*log*` 表仅警告。单片热数据涨到百万级后，list 仍按 `(workspace_id, company_id, task_id, created_at)` 索引，但运维裁冷数据会锁整片。
- **Action**: (1) 用 `partition_metrics_exporter` / `information_schema.partitions` 监控 16 片行数；(2) 任一片 >500 万行时：要么 `PRIMARY KEY (id, created_at)` + 按月 RANGE，要么扩到 64 片并双写迁移；(3) 冷分区 `TRUNCATE PARTITION`。
- **Why**: 哈希分表解决租户膨胀，不解决单 workspace 时间累积；不规划分区会在热片上重演锁表迁移。
- **How to apply**: 阈值与脚本对齐 `db/scripts/ensure_partitions.sh`；改 DDL 时同步 Go 选片常量 `ccbLogShardCount`。

## [OPT-20260821-034] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 直连商户 merchant-configs 实测 400 INVALID_REQUEST；展示改为 conf profit_sharing_max_ratio_percent（默认 30%）镜像商户平台分账管理比例，不再调用该 API。打款仍 5% 封顶。
- **Created**: 2026-08-21
- **Context**: 官方文档把「查询最大分账比例」写在合作伙伴路径 `GET /v3/profitsharing/merchant-configs/{sub_mchid}`。本项目是直连商户（`conf/billing/wechatPay` 的 `mchid`）。查询失败会 warn 并回退展示 5%，页面看起来像没改。
- **Action**: (1) 用生产 mchid 打一次该接口看 200 还是 403；(2) 若 403，改查商户平台「产品中心-分账-分账管理比例」是否有直连查询接口，或把业务佣金写入 conf；(3) 失败路径保持 warn，不要静默当成功。
- **Why**: 直连商户调合作伙伴 API 可能永远失败，前端会一直显示回退 5%。
- **How to apply**: `taskBill/src/wechat_profit_sharing_ratio.go` `queryWechatMerchantMaxRatioLive`；对照 https://pay.weixin.qq.com/doc/v3/partner/4012466864

## [OPT-20260822-005] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: task-bill 精准重启后 GET /api/internal/taskbill/referral/commission-rate/ 返回 30%/wechat_merchant_config；已重启 task-referral 以清空 60s 缓存。
- **Created**: 2026-08-22
- **Context**: 直连商户误调 merchant-configs 导致页面回退 5%。代码已改为 `profit_sharing_max_ratio_percent: 30`（微信默认上限）。旧 task-bill 进程仍返回 `source=local_fallback`。
- **Action**: (1) 点 9999「精准编译重启」`task-bill`（2) `GET /api/internal/taskbill/referral/commission-rate/` 应为 `30%` / `wechat_merchant_config`（3) 打开 `/profile/referral/` 的 `referral-rate-display` 不为 5%（4) 若商户平台分账管理比例不是 30%，只改 conf 该键再重启
- **Why**: 不重启则公网仍显示 5%；conf 与微信后台漂移时页面会再次不一致。
- **How to apply**: `scripts/register-precise-restart.sh task-bill`；失败经验 `.ai/09_failure_experience/02_runtime_errors/112_referral_rate_wechat_partner_api_400.md`

## [OPT-20260822-006] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 页面不再把 conf 30% 当收益分成；展示改为打款口径 min(上限,5%)，见失败经验 114。conf 仅保留为 RULE_LIMIT 上限。
- **Created**: 2026-08-22
- **Context**: 直连商户无查询 API，展示默认按微信文档上限 30% 写入 `profit_sharing_max_ratio_percent`。若商户平台实际改过（例如申请到更高或调低），页面会与后台再次不一致。
- **Action**: (1) 打开微信支付商户平台「产品中心-分账-分账管理比例」记下数字 (2) 若不是 30，只改 `conf/billing/wechatPay/conf.yaml` 的 `profit_sharing_max_ratio_percent` (3) 精准重启 `task-bill`（及 `task-referral` 以清 60s 缓存）
- **Why**: conf 是唯一能与微信后台对齐的 SSOT；改后台不改 conf 会重现本 bug。
- **How to apply**: `conf/billing/wechatPay/conf.yaml`；失败经验 112

## [OPT-20260822-013] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: Playwright T6/T7 已绿：未选身份禁用自动运行；选择后 POST repo_identities 为当前用户 git_identity_id。公网 bundle 含 fork-auto-run-git-identity-section。
- **Created**: 2026-08-22
- **Context**: 派生自动运行已在确认弹窗选择当前用户 Git 身份并 POST `repo_identities`。现有 `TaskDetail.fork-popup.playwright.test.js` 的源任务 `projects: []`，E2E 盖不住身份门禁。
- **Action**: (1) mock 源任务带 GitLab 仓与 `/api/git-identities/user/...` (2) 断言未选身份时 `fork-confirm-auto-run` disabled (3) 预填/选择后 POST body 含当前用户 `git_identity_id`
- **Why**: 生产 400 `repo_identities_required`（trace `cdbb1723-a513-409e-b066-b5128baaafeb`）正是有关联仓时发生的；单元测过了仍缺浏览器路径。
- **How to apply**: `taskFE/tests/TaskDetail.fork-popup.playwright.test.js`；身份选择 `data-testid=create-task-repo-git-identity-select`

## [OPT-20260822-016] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 租户 GET gitlab-regions 在 encode 前清空 AdminPrivateToken；TestHandleGitlabRegionsList_StripsStoredAdminToken 覆盖非空 PAT
- **Created**: 2026-08-22
- **Context**: 租户设置 GitLab 页通过 `GET /api/billing/gitlab-regions/tenant_id/{tid}/` 拉区域下拉。`listGitlabRegions` 扫描并 JSON 序列化 `GitlabRegion.AdminPrivateToken`（`json:"admin_private_token,omitempty"`），token 非空时会下发到浏览器。
- **Action**: (1) 租户侧 list 响应在 encode 前清空 `AdminPrivateToken`（2) 系统管理端点若需 PAT 保持现有内部/管理员通道（3) 加单测：租户 list 的 JSON 不含 `admin_private_token`
- **Why**: 区域 GitLab Admin PAT 属于基础设施密钥，出现在租户前端即可被任意租户管理员取走。
- **How to apply**: `taskBill/src/gitlab_region.go` `handleGitlabRegionsList` / `GitlabRegion`；对照 `handleSystemAdminGitlabRegions`

## [OPT-20260822-014] completed

- **Status**: completed
- **Created**: 2026-08-22
- **Completed**: 2026-08-22
- **Completion-Note**: 已否定。比例只能从微信支付接口获取，不再把 5% 抽到 conf。
- **Context**: 收益分成展示已改为打款口径 `min(上限, defaultProfitSharingCommissionRate=5)`。佣金分子仍硬编码在 `taskBill/src/wechat_profit_sharing.go`，改政策必须发版。
- **Action**: (1) 在 `conf/billing/wechatPay/conf.yaml` 增加 `profit_sharing_commission_percent`（默认 5） (2) taskBill 读取该键，0 或越界时 503 不得静默回退 (3) 单测覆盖 conf=5 展示 5%、conf=3 且上限 30 展示 3%
- **Why**: 佣金政策与微信上限一样属于人工可调配置，应落在 conf SSOT（元规则 47）。
- **How to apply**: `conf/billing/wechatPay/conf.yaml`；`taskBill/src/wechat_pay.go` `wechatPayConfig`；`applyPayoutPolicy`

## [OPT-20260822-018] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskBill fd47c02: POST /purchase/ 409 USE_RESOURCE_ORDER，删除 purchaseGitlabResources；TestHandleGitlabResourcesPurchase_DoesNotDebitWallet / OmitsSpendableCash；go test ./src 全绿。taskFE 3aee60c 去掉 purchase()。docs 1ae2e54 意图/价值流已更新。
- **Created**: 2026-08-22
- **Context**: 产品确认支付只转资源配额、无用户留存现金。租户 UI 已去掉现金展示，但 `purchaseGitlabResources` / `gitlab_region_purchase.go` 仍 `UPDATE billing_account SET balance`。租户下单走资源订单发放配额，这条扣钱包路径与模型冲突。
- **Action**: (1) 确认租户购买只走 `markOrderPaid` 发配额 (2) 删除或切断 POST `/purchase/` 扣 `balance` 的用户路径，余额不足改为引导下单 (3) 补单测：购买成功不减少 `billing_account.balance`；`go test` 相关包全绿
- **Why**: 扣空钱包会让旧账户出现虚假「余额不足」，即使用户应按订单买配额。
- **How to apply**: `taskBill/src/gitlab_resources_purchase.go`、`gitlab_region_purchase.go`；对照 `order_payment.go` `markOrderPaid`

## [OPT-20260822-034] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 微信建号在 publish USER_CREATED 前写入 auth_user_profile；batch/details 在发事件时已能读到昵称。TestCreateWechatUserProfileAvailableBeforeUserCreated 绿。
- **Created**: 2026-08-22
- **Context**: 新用户 onboarding 默认公司名已改为「{user}的公司」。微信路径在 `createWechatUser` 里先异步发 `USER_CREATED`（username 空），回调里才 `ensureWechatProfileNickname`。Kafka 若抢先消费，自动建公司仍可能落成「我的公司」，要等引导页用 profile 预填才改过来。
- **Action**: (1) 新用户建号成功后先写入个人昵称，再 `publishUserCreatedAsync` (2) 补单测：profile 已有昵称时 `CreateCompanyForUser` 得到「{昵称}的公司」且不依赖引导页 (3) 禁止把手机号当展示名
- **Why**: 仅靠 Onboarding 预填时，用户若直接确认后端已创建的「我的公司」，会把中性占位保存下去。
- **How to apply**: `taskAuth/src/auth_wechat.go` `createWechatUser` / `handleWeChatCallback`；`taskEvents` `ResolvePersonalCompanyName`

## [OPT-20260822-037] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: taskCloudService 5094db9：生产 mintAliyunReleaseSTS 走 STS AssumeRole + 单实例 session policy；无 Role / minutes=0 省略字段；单测覆盖 CPA 密钥传入与 DurationSeconds
- **Created**: 2026-08-22
- **Context**: 指令闲置 L3 已实现 session policy 与可注入 `stsReleaseMinter`；生产 `stsReleaseMinter` 仍为 nil，有 Role 时 task-detail 也不会下发 `machine_release_sts`。L1/L2 已能释放，L3 仅在容器连不上平台时补位。
- **Action**: (1) 用阿里云 STS AssumeRole + `buildSTSReleaseSessionPolicy(instance_id)` 实现生产 minter（禁止 GetSessionToken）(2) 无 Role 仍省略字段 (3) 补单测：minter 被调用时 Resource 仅含该 instance
- **Why**: 平台不可达时容器无法靠 L1，L2 需等 timer；有 Role 却不下发 STS 会使 ADR-0031 的 L3 形同虚设。
- **How to apply**: `taskCloudService/src/sts_release_session_policy.go` `stsReleaseMinter`；CPA `sts_release_role_arn`；Credential 已透传 `machine_release_sts`

## [OPT-20260822-039] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 公网硬刷新后 index-CseyaJ4s.js；collapsed=1 时 nav[data-alias=cmp-navbar-main] 可见 sticky 73px，文案含 AI项目推进平台/价格/退出
- **Created**: 2026-08-22
- **Context**: 租户页点「控制台导航」会把 `tenant-console-sidebar-collapsed=1` 写入 localStorage，系统管理页没有恢复入口却整段不渲染 Navbar。源码已按路由豁免（`navbarCollapsedVisibility.js`），本地 `npm run build` 已切到 `releases/20260822174742-3070293`，待精准编译重启让公网 nginx 吃到新 html。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点「精准编译重启」消费已登记的 `taskFE` (2) Playwright/CDP 打开 `/system-admin/?accessCode=DR2AKvP9J9`，先 `localStorage.setItem('tenant-console-sidebar-collapsed','1')` 再刷新 (3) 断言 `nav[data-alias=cmp-navbar-main]` 存在且 sticky
- **Why**: 不验收的话，用户浏览器仍可能对着旧 hash，看起来像没修好。
- **How to apply**: 复现 URL `https://www.daydaymoney.com/system-admin/?accessCode=DR2AKvP9J9`；单测已在 `taskFE/app/src/components/Navbar.collapsed.test.js` 与 `navbarCollapsedVisibility.test.js`

## [OPT-20260822-043] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 精准编译重启已消费 taskFE/task-bill/task-referral；公网 SPA release 20260822183633-3517908 / index-DQkIvqg9.js；Playwright 超管页：referral-ratio-range=5%~30%、仅 1 个 number input、无 profit-sharing 输入、rate=5；申请表头仅「分成比例」、两行均为 5%。DB billing_referral_config 两列均为 5。临时 is_superuser 已回收。
- **Created**: 2026-08-22
- **Context**: 推荐资格管理已改为范围展示 5%~30% + 单一分成比例输入；`059_referral_single_ratio.sql` 将存量双列对齐。task-bill / task-referral / taskFE 已登记精准编译重启，SPA 已发布 `releases/20260822182523-3417993`。
- **Action**: (1) ✅ 059 已应用到 `task_bill` (2) 精准编译重启已登记的 task-bill / task-referral / taskFE (3) 超管硬刷新 `/system-admin/referral-management`：可见 `5%~30%`、仅一个数字输入 (4) 硬刷新申请列表确认只剩「分成比例」一列
- **Why**: 未应用 059 则库内分账列仍可能是 30、推荐列是 5；未硬刷新则旧 SPA 仍并排两个输入。
- **How to apply**: `scripts/register-precise-restart.sh task-bill task-referral taskFE`；`data-testid=referral-ratio-range` / `referral-rate-input` / `referral-app-ratio`

## [OPT-20260822-041] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: ztree 红色 push 失败芯片，来自 git_remote.last_push_error；与提交并创建PR同时可见
- **Created**: 2026-08-22
- **Context**: `task_878932440129761280` 自动运行第 4 步 push 400 失败后，ztree 仍显示可点的「提交并创建PR」，用户误以为步骤未调度。容器侧已把失败写回 Agent 评论，层图未同步失败态。
- **Action**: (1) 在 onlineServiceJS 层图快照为交付失败写 layer command / 错误节点 (2) FE ztree 渲染该失败态（禁用或标注失败）(3) 用层图快照单测锁定失败文案含 OAuth access_token
- **Why**: 评论回填可被折叠；ztree 是用户判断「第 4 步有没有跑」的主入口。
- **How to apply**: `trae-agent/onlineServiceJS/src/autoRunDeliveryHooks.mjs`；层图 snapshot 写入点；`taskFE` ztree 层命令渲染

## [OPT-20260822-007] cancelled

- **Status**: cancelled
- **Completed**: 2026-08-22
- **Summary**: 用户页当前分账比例改为读 billing_referral_config，不再以微信失败显示「—」为验收标准；部署验收见 OPT-20260822-048
- **Created**: 2026-08-22
- **Context**: 分成比例改为只信微信支付查询接口。直连商户 merchant-configs 现网 400，内部 API 应为 503，页面「—」。禁止再验收 5% 或 30%。
- **Action**: (1) 精准编译重启 `task-bill` `task-referral` (2) `GET /api/internal/taskbill/referral/commission-rate/` 直连期望 503，body 不含 `5%`/`30%` (3) 硬刷新 `/profile/referral/` 断言 `referral-rate-display` 为「—」（除非微信查询已 200）
- **Why**: 商户后台既不是 5% 也不是 30%；min/conf 都会伪装真实设定。
- **How to apply**: `scripts/register-precise-restart.sh task-bill task-referral`；内部 API；页面 `data-testid="referral-rate-display"`

## [OPT-20260822-050] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: GitLab 换票改为独立 25s 响应头超时；运维补发 Doorkeeper token 后 nested-git 成功，force_auto_run 已 start-vm（aliyun 启动成功）。旧评论原文不改写。
- **Created**: 2026-08-22
- **Context**: `task_878979977859592192` 自动运行评论曾泄漏 `gitlab refresh http 400`。2026-08-22 20:46 已精准编译重启 task-git-oauth / task-project-service：refresh 现带官方字段且行锁串行。复探测仍 `invalid_grant`（凭据 `gitlab:tencent-sh-1` 更新于 18:00 CST，随后 2.5h 旧进程无 redirect_uri 换票失败，refresh 已失效）。文案已改为「授权已失效，请重新绑定」。
- **Action**: (1) 用户在个人资料完成 GitLab 腾讯上海一区重新绑定 (2) 打开任务详情点「强制重新启动」(`force_auto_run`) (3) 断言 skip_reason / 新自动运行评论不含 `gitlab refresh http 400`，子仓可获取或仅在仍失效时显示重新绑定引导 (4) start-vm 在授权有效时发生
- **Why**: 过期/作废的 refresh_token 无法用代码救活；不重新绑定则自动运行会继续软跳过启服。
- **How to apply**: 页面 `https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878979977859592192/?accessCode=DR2AKvP9J9`；provider `gitlab:tencent-sh-1`

## [OPT-20260822-030] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 同批验收：skip_reason 已清空，task_878979977859592192 start-vm_ok。
- **Created**: 2026-08-22
- **Context**: 任务详情 `task_878902048790179840` / `task_878979977859592192` 软跳过原因曾是 `gitlab refresh http 400`。`RefreshGitLabToken` 已补官方字段+行锁。2026-08-22 20:46 已精准编译重启 `task-git-oauth` / `task-project-service`（健康 200）。复探测现为人性化「授权已失效，请重新绑定」，底层 `invalid_grant`（refresh 在旧进程窗口内过期）。存量 skip_reason 仍需 `force_auto_run`；启服还须先重新绑定，见 OPT-20260822-050。
- **Action**: (1) 在 :9999 对已登记的 `task-git-oauth` / `task-project-service` / `taskFE` 做精准编译重启 (2) 打开同一任务详情点「强制重新启动」 (3) 断言 `auto-run-start-skipped-reason` 不再含 `gitlab refresh http 400`，子仓列表可获取或仅在真正失效时显示重新绑定引导
- **Why**: 代码修复不自动清已落库的 skip_reason；未重启服务则 refresh 仍走缺字段请求。
- **How to apply**: `.runall/precise_restart_services.txt`；页面 `https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878902048790179840/?accessCode=DR2AKvP9J9`

## [OPT-20260822-057] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 公网任务详情已硬刷新：默认「项目文件树」Tab；点「文件变动」后面板显示「暂无文件变动数据」（本页无变动载荷，无 · N 徽章）
- **Created**: 2026-08-22
- **Context**: 任务关联区把项目文件树与层变动列表改成 Tab。公网 SPA 须 `npm run build` + 精准重启 taskFE 后才可见。
- **Action**: (1) 确认 taskFE 已 build 且 `public/html` 指向新 release (2) 硬刷新任务详情选中可写层 (3) 断言默认「项目文件树」Tab，点「文件变动」后见变动列表且计数徽章正确
- **Why**: 本地 Vitest 已覆盖，公网仍跑旧 bundle 直到 SPA 发布。
- **How to apply**: 页面 `/tenant/.../task-detail/.../`；`data-testid=layer-files-tablist`

## [OPT-20260822-056] completed

- **Status**: completed
- **Completed**: 2026-08-22
- **Summary**: 精准重启已执行；公网三 Tab 可见；staff GET 返回未完成分账行且不含 finished。写操作另开增量。
- **Created**: 2026-08-22
- **Context**: 本增量在 `/system-admin/order-records/` 增加「待分账」只读队列与 `GET /api/system-admin/profit-sharing/`。公网 SPA 与 task-bill / APISIX 须编译重启后才可见。立即向微信发起分账或失败重试不在本增量。
- **Action**: (1) 在 http://10.2.150.68:9999/ 精准编译重启已登记的 `task-bill` `taskFE` `task-gateway` (2) 硬刷新 https://www.daydaymoney.com/system-admin/order-records/ 断言第三 Tab「待分账」且默认列表不含已完成 (3) 若运营需要写操作，另开增量：staff-only 立即分账/失败重试 + Idempotency-Key + NFR L3
- **Why**: 只读列表交付后仍须部署验收；写路径会动资金，不能塞进展示增量。
- **How to apply**: `scripts/register-precise-restart.sh`；`SystemAdminOrderRecords.vue`；`handleSystemAdminListProfitSharing`

