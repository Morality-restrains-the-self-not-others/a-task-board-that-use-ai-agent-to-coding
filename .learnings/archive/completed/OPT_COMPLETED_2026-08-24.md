# Completed OPT Archive — 2026-08-24

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 51 条。
> 归档执行时间：2026-08-25T22:13:52+08:00

## [OPT-20260824-054] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE `32ea30e` 已推送：删除 `CONSENT_CONTEXT_LABELS` 两处 `login_phone_code: '验证码登录'`（SystemAdminPrivacyPolicyConsentQuery.vue + useSystemAdminLicenseAgreement.js）。线上 `task_bill.billing_user_privacy_policy_consents` 实测 **0 行**（无任何历史 consent 记录，更无 login_phone_code context），删除安全。相关 composable 测试 8 例全绿（paymentTerms + traceId），pre-commit 全绿。
- **Title**: 移除 consent 标签映射中的 login_phone_code（历史数据清理后）
- **Created**: 2026-08-24
- **Context**: 手机号+验证码登录移除后，管理后台两处 `CONSENT_CONTEXT_LABELS` 仍含 `login_phone_code: '验证码登录'`（SystemAdminPrivacyPolicyConsentQuery.vue:80 / useSystemAdminLicenseAgreement.js:90）。为只读历史数据展示映射（旧 consent 记录 context code），无执行路径，当前保留合理
- **Action**: 确认线上 auth_consent 记录已无 login_phone_code context（或历史数据清理完成后），删除这两处映射条目，避免新开发者误以为验证码登录仍存在
- **Why**: 展示映射残留会误导后续维护者对登录方式现状的判断
- **How to apply**: 先查 `SELECT DISTINCT context FROM auth_consent`（或等价表）确认无 login_phone_code；再删两处条目

## [OPT-20260824-055] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE `cb99234` 已推送：新增 `app/src/components/auth/LoginMethodSelector.test.js` 6 例全绿 —— allowPhoneLogin=false 只渲染 邮箱/密码+访问令牌（无 手机号/密码 Tab）、allowPhoneLogin=true 三 Tab、默认高亮 emailPassword、点击访问令牌/手机号密码均发 update:modelValue+method-changed 双事件、modelValue prop watch 同步高亮。vitest 运行 6 例 40ms 绿 + node --test guard 通过（pre-commit 全绿）。
- **Title**: LoginMethodSelector 组件级单测缺失
- **Created**: 2026-08-24
- **Context**: LoginMethodSelector.vue（登录方式 Tab 选择器）无组件级 vitest 覆盖，codegraph 标记 ⚠️ no covering tests；当前仅 E2E（PhoneAuthLifecycle 测试 3 + policy-toggle 测试）间接覆盖「无手机号/验证码 Tab」断言
- **Action**: 为 LoginMethodSelector 补组件级测试：allowPhoneLogin 开/关下 Tab 渲染集合 + switchMethod emit 行为
- **Why**: 组件是登录页核心交互，单测可在无环境依赖下快速回归 Tab 集合（防止验证码 Tab 复活）
- **How to apply**: 参照 useLoginSubmit.test.js 的 vi.mock + @vue/test-utils 模式，新增 LoginMethodSelector.test.js

## [OPT-20260823-034] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill b6740dc 已推送：支付/分账渠道 err.Error() 出口净化，handlePayOrder 预下单与推荐人手动分账失败走 paymentActionClientError 剥离 WeChat dump，非退款动作措辞改「微信支付失败」；新增 handler 级 dump 不泄漏回归测 + paymentActionClientError 单测 3 例；taskBill 全包 66s 绿。FE 确认 showRequestError/formatApiError/humanizeRequestErrorMessage 均已走 humanizePaymentProviderError。
- **Created**: 2026-08-23
- **Context**: 超管退款审批已把微信 `APIError.Error()` dump（含 `Wechatpay-Signature`）从批准/拒绝 JSON 和弹窗剥离，走 `wechatPayClientError` + `humanizePaymentProviderError`。下单、关单、查单、回调失败仍可能把 `err.Error()` 原样 `writeErrorJSON` 给浏览器。
- **Action**: (1) 在 `taskBill/src` 搜索 `writeErrorJSON`/`err.Error()` 的支付渠道路径 (2) 统一改走 `wechatPayClientError`/`humanizeProviderErrorText` (3) 补测：dump 不得出现在 JSON (4) 前端其它支付失败 UI 确认已走 `humanizePaymentProviderError`
- **Why**: 签名头进浏览器是密钥材料泄露；用户也看不懂 HTTP dump。退款审批只修了一条路径。
- **How to apply**: `taskBill/src/wechat_pay.go`、预下单/关单 handler、`taskFE/app/src/utils/humanizePaymentProviderError.js`

## [OPT-20260823-050] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill bedb39d 已推送：processPendingProfitSharings 前置只读告警扫描，paid 且超 30 天窗口、分账非 finished 订单计数 + OVER_WINDOW_UNSHARED warn（明细仅 order_id/status/paid_at，无 openid），供 Loki 告警；新增 4 例回归测；taskBill 全包 67s 绿。Prometheus gauge 因 taskBill 无 metrics 端点暂以 Loki 告警路径替代，计数函数可复用。
- **Created**: 2026-08-23
- **Context**: 25 天兜底只处理 `paid_at` 落在 (now-30d, now-25d] 的订单。超过微信约 30 天冻结窗口后 CreateOrder 会失败，当前扫描直接跳过，运营无法从指标发现漏分账。
- **Action**: (1) 在 `processPendingProfitSharings` 或独立只读扫描统计 `paid` 且窗口外、分账非 finished 的订单数 (2) 打 warn 日志（只含 order_id/status，不含 openid）并可导出 Prometheus gauge (3) 单测：31 天 failed 计入，26 天 pending 不计入窗口外
- **Why**: 窗口外无法补分，漏单只能靠人工从微信商户平台对账；没有计数则 25 天兜底失败后会静默丢佣金。
- **How to apply**: `taskBill/src/wechat_profit_sharing_fallback.go`；现有 `billing_profit_sharing_scan` timer 同一入口即可，不必新 ticker

## [OPT-20260823-035] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill a59039f 已推送：queryWechatOutTradeNoByTxnID 加单进程限流(默认2/s)+连续失败熔断(阈值5/冷却60s)，短路期不发真实客户端；失败日志沿用 query_len 不打单号；新增可注入接缝；回归测 2 例；taskBill 全包 68s 绿。
- **Created**: 2026-08-23
- **Context**: 超管用微信支付单号查本地 0 条时会调用 Native `QueryOrderById`（8s 超时、可注入 stub）。无 QPS 上限；客服连点或脚本刷 28 位数字会打到微信查单配额。
- **Action**: (1) 给 `queryWechatOutTradeNoByTxnID` 加单进程限流（例如每秒 2 次）与熔断 (2) 失败只打 `query_len` 不打单号 (3) 补测：超限不发真实客户端
- **Why**: 管理端兜底是为历史单补 `wechat_transaction_id`，不应成为微信 API 的放大入口。
- **How to apply**: `taskBill/src/wechat_pay_vouchers.go`、`orders_list_trade_no.go`

## [OPT-20260823-042] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill 24fa803 已推送：billing_profit_sharing_scan 增加无佣金行微信单解冻补偿扫描（paid 微信单、无 ps 行、paid_at ∈ [now-30d, now-10min] 重试 UF{order_id}，依赖微信同号幂等；tracelog 可观测）。回归测 3 例；taskBill 全包 67s 绿。
- **Created**: 2026-08-23
- **Context**: ADR-0040 在支付成功 goroutine 里对无 `billing_profit_sharing` 行的微信单调用 Unfreeze。为避免与落佣金行竞态，already-paid 回调重放不再解冻。Unfreeze 失败时货款会冻到微信 30 天自动解冻。
- **Action**: (1) 在既有 `billing_profit_sharing_scan` timer 增加：已支付微信单、无佣金行、支付超过 N 分钟仍未解冻则重试 `UF{order_id}` (2) 需可观测本地「已解冻」标记或依赖微信同号幂等 (3) 补单测：有佣金行不重试、无行且 ledger 有 transaction_id 则调用 Unfreeze
- **Why**: 解冻瞬时失败时商户可用余额最长可冻 30 天。
- **How to apply**: `taskBill/src/wechat_profit_sharing_unfreeze_remainder.go`；`handleInternalProcessPendingProfitSharings`；taskEvents `billing_profit_sharing_scan`

## [OPT-20260823-041] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: meta fe4d700f 已推送：pre-commit gitlink MISSING 判定抽到 scripts/lib/gitlink_dirty_check.sh（仅已跟踪未提交变更才算 dirty，纯 ?? 不阻断），selftest 5 态通过，约束 32 补文档；已顺带推送 taskFE/taskAuth/db/docs 既有未推送提交以过 pre-push 门禁。
- **Created**: 2026-08-23
- **Context**: 本会话已 push docs `c8b4bca`（已合并 MR noop 设计补丁），但 docs 工作区有他会话 `?? intents/...` 未跟踪文件。meta pre-commit 用 `git status --porcelain` 判定 dirty，阻断把 gitlink 指到已推送 HEAD。结果 meta 只能同步 taskFE，docs 指针仍停在 `1e9fb78`。
- **Action**: (1) 改 `.githooks/pre-commit`：对暂存 gitlink 对应子仓，仅当存在已跟踪未提交变更（` M`/`M `/`A `/`D ` 等）才列入 MISSING；纯 `??` 不阻断 (2) 补 `scripts/lib` 或 hook 自测：工作区仅 untracked 时 `git commit` 指针可通过 (3) 文档写明：他会话未跟踪文件不得被本会话 `git add`
- **Why**: 并行会话的未跟踪意图文档会卡住无关 gitlink 同步，meta 长期落后 origin。
- **How to apply**: `.githooks/pre-commit` MISSING 循环；`python3 db/scripts/ci/` 或 `scripts/lib/auto-commit_selftest.sh` 同类 fixture；跑自测 exit 0

## [OPT-20260823-027] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskCloudService baca6cf 已推送：idle_recycle 清库路径清库前取 comment_id，recycle 后 publishTaskSSE 带「触发：工作区机器空闲回收」调度行写入评论 binding 日志（确认只清库不调云 API）；新增回归测；taskCloudService 全包 181s 绿。
- **Created**: 2026-08-23
- **Context**: `workspace_machine_recycle.go` 的 `idle_recycle` 目前只清库、不调云 API，也不走 `releaseMachineForTerminal` / `CLOUD_SERVER_STOPPED`，因此不会在评论启动日志留下「触发：工作区机器空闲回收」。
- **Action**: (1) 确认该路径是否仍会停真实 ECS (2) 若会，改为发布带 `stop_reason=idle_recycle` 的停服事件 (3) 若只清库，至少 `publishTaskSSE` 一条带触发说明的调度行写入 binding logs
- **Why**: 用户看到「正在调用aliyunAPI停止服务器」时需要知道是空闲回收还是任务取消；工作区回收是第三条源头，现在仍是盲区。
- **How to apply**: `taskCloudService/src/workspace_machine_recycle.go`；复用 `annotateStopServerMessage` + `publishTaskSSE`

## [OPT-20260823-025] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: runAll d9c18d8 已推送：toast 系统抽到 17.js（04.js 498 行 ≤500），manifest 登记；showToast 同 message+type 去重 + 超过 5 条立即 removeChild 硬上限；node --check + runAll 全包 112s 绿。
- **Created**: 2026-08-23
- **Context**: `runAll/src/status_ui/js/04.js` 已 558 行（超过 500 行门禁）。`showToast` 在超过 5 条时只给最旧 toast 加 `toast-removing`（280ms 后才从 DOM 删除），若调用频率高于移除速度，DOM 会堆积上千个 toast（本次 start-all 空计划 bug 曾到 toastCounter=35000）。本轮已从根因堵住重放，但 toast 层仍缺少硬上限。
- **Action**: (1) 把 `showToast`/`dismissToast`/`getToastContainer` 抽到新的 `status_ui/js/17.js` 并写入 `manifest.json`，使 `04.js` 回落到 ≤500 行 (2) `showToast` 对相同 message+type 的可见 toast 去重 (3) 超过上限时立即 `removeChild` 而不是只加 `toast-removing`
- **Why**: 行数门禁已触发；toast 层硬上限可防止未来任何重放路径再次打爆页面。
- **How to apply**: `runAll/src/status_ui/js/04.js`、`runAll/src/status_ui/manifest.json`；`node --check` 新文件；`go test ./src -run TestUIHomePage`

## [OPT-20260823-053] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill 3d97552 + taskFE ef0ef72 已推送：approveInvoiceApplication 支持可选 fapiaoNumber（VARCHAR(32) 对齐、超长 400、空置空）落库 wechat_fapiao_number 并在订单发票 JSON 露出；面板「已开具」前加可选输入框并随请求携带 fapiao_number。后端回归测 3 例 + 前端 vitest 2 例。
- **Created**: 2026-08-23
- **Context**: 开票改为手动开具后，「已开具」登记只落 `issued`，`billing_invoice.wechat_fapiao_number` 恒为空；租户订单发票列表只显示状态不显示票号，对账/客服需人工到商户平台查。
- **Action**: (1) 面板 pending 行「已开具」动作前增加可选输入框填写微信发票号码 (2) approve API 请求体携带 `fapiao_number`（后端校验长度，空则置空）(3) 单测：带号登记后订单发票 JSON 含该号码
- **Why**: 手动开具后票号是唯一跨系统凭证，登记可避免二次查证；也为后续自动冲红（`fapiao_apply_id` 需人工回填）留审计位。
- **How to apply**: `SystemAdminInvoicePanel.vue` act() 与列表行；`taskBill/src/invoice_approve.go` approveInvoiceApplication 入参扩展；`invoice_approve_test.go`

## [OPT-20260823-044] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskChromePlugin 8e2493d 已推送：webRequest requestId 精确对齐并发同 URL 的 POST body——onBeforeRequest 缓存带 requestId，HAR 条目透传 _requestId，lookup 优先按 requestId 精确命中，未知 requestId 回退时间窗；回归测 6 例（cache 并发双 POST/未知回退/SW 命中第二条/HAR 提取/devtools 透传）npm test 353 例全绿
- **Created**: 2026-08-23
- **Context**: 点选请求写入请求体已用 method+url+tabId+时间窗匹配（F-073）。同一标签页短时间内对同一 URL 并发两条 POST 时，时间窗可能把 body 配错。
- **Action**: (1) onBeforeRequest 缓存带上 `requestId` (2) DevTools HAR/`onRequestFinished` 若能拿到同一 requestId 则优先精确匹配 (3) 补单测：两条同 URL 不同 body 在 100ms 内先后发出，点选各自写入对应 body。
- **Why**: 时间窗匹配在并发下会串 body，任务描述可能贴上另一条请求的载荷。
- **How to apply**: `taskChromePlugin/lib/request-body-cache.js`、`background/service-worker.js`、`devtools/devtools.js`；测例扩 `test/request-body-cache.test.js`

## [OPT-20260824-001] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 管理员登录入口分离 + 种子管理员随机密码/邮箱重置：(a) 客户登录/验证码/访问令牌入口拒绝管理员账号，仅专用 POST /api/auth/admin-login/；(b) 种子密码改 256-bit 随机哈希（明文销毁），010_verify_super_admin.py 检测已知弱哈希自动轮换并输出邮箱重置指引；(c) 邮箱/验证码重置成功后清除 must_change_password；(d) E2E seed_e2e_account.py --superadmin 预置测试密码
- **Created**: 2026-08-24
- **Context**: (a) 客户登录入口可接受管理员账号，混合入口扩大账号接管面；(b) 种子管理员密码为已知明文（admin123），初始化脚本还打印密码提示，任何人可登录超管。
- **Action**: (1) taskAuth auth_login.go / auth_access_token.go 入口分离（含 auth_entry_separation_test.go）(2) dataMigrate/taskAuth/022_seed_bootstrap_admin.sql 随机哈希 + 010_verify_super_admin.py 轮换与指引 + test_010_verify_super_admin.py (3) taskAuth db.go clearMustChangePassword + auth_password_reset.go 两处重置后调用 + auth_password_reset_must_change_test.go (4) taskFE seed_e2e_account.py --superadmin / SystemAdmin 测试前置预置 (5) 4 个 e2e-tests 调试脚本移除 admin123 硬编码
- **Why**: 已知管理员密码与混合登录入口是账号接管风险面；随机密码 + 邮箱重置为唯一入口，即使开发者也无法用已知凭据登录。
- **How to apply**: `scripts/init_system.py`（初始化自动轮换）；`taskFE/tests/helpers/seed_e2e_account.py --superadmin`（测试环境回填）；runAll 页面「精准编译重启」编译重启 task-auth 生效；新轮换哈希须同步 010 与 022 两处。

## [OPT-20260823-058] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill 4b90e51 + dataMigrate ead8755 已推送（067 已应用生产 task_bill）：admin_grant 幂等键落库操作者 X-User-Id/目标租户/op_type，新增 GET /api/system-admin/idempotency-records/（平台员工可查，op_type/tenant_id/operator_user_id/key 过滤）；回归测 4 例 + OpenAPI 契约，整包 go test 75s 全绿
- **Created**: 2026-08-23
- **Context**: admin_grant 幂等键（body/header 双通道）仅用于去重校验，未落库；事后无法追踪「哪次操作产生了哪张赠送订单」。
- **Action**: billing_idempotency_key 行记录 admin_grant 操作时附带操作者 X-User-Id 与目标租户，并在管理端查询幂等记录（仅超管）
- **Why**: 赠送/退款类资金操作需要完整审计链；本次误送排查依赖订单时间+管理员身份手工比对，落库后可直接追溯。
- **How to apply**: `taskBill/src/handlers_admin_grant.go` adminGrantIdempotencyKey 调用处；幂等表 schema（db 子仓）

## [OPT-20260823-062] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill 162624f + runAll 2ae62ba 已推送：(1) taskBill evaluateGitlabTrafficGate 每次判定按 code（UNMAPPED/TRAFFIC_NOT_PURCHASED/EXCEEDED/CI_SKIP/INTRANET_SKIP/OK）累加计数，/metrics 聚合 tracelog HTTP RED + 闸门计数器，8004 隧道中断计数归零可按 rate 掉零告警；(2) ensure-edge-tunnels 巡检隧道缺失追加 alerts.log WARN（Promtail/Loki 检索）。回归测 2 例 + runAll pre-commit 全绿；(3) GitLab 侧 GATE_UNREACHABLE 日志级别提升为可选后续
- **Created**: 2026-08-23
- **Context**: 闸门在 taskBill 不可达时 fail-open（设计决策），但当前仅 initializer 侧 `warn_log`，taskBill 侧无指标。本次发现远程→taskBill 需 8004 隧道（ensure-edge-tunnels），隧道中断会导致闸门静默失效。
- **Action**: (1) taskBill 侧统计 gate 请求按 code（TRAFFIC_NOT_PURCHASED/EXCEEDED/INTRANET_SKIP/CI_SKIP/UNMAPPED/GATE_UNREACHABLE 等）计数并暴露指标 (2) 隧道健康检查：ensure-edge-tunnels 巡检失败通知（8004 隧道断连告警） (3) GitLab 侧可选：GATE_UNREACHABLE 连续 N 次在 Rails 日志级别提升
- **Why**: fail-open 是可用性权衡，必须有告警兜底，否则配额管控在故障期形同虚设且无人知晓。
- **How to apply**: `taskBill/src/gitlab_traffic_gate.go`（指标）；`runAll/scripts/ensure-edge-tunnels.sh`（健康巡检）；AiMonitor 告警规则

## [OPT-20260824-005] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: /profile/referral/ 页面 URL 溯源 accessCode 归一化为自身推荐码：(a) 根因确认：URL ?accessCode= 是分享链接溯源参数（他人/已删除码），与页面展示的用户自身推荐码（后端 status API）语义不同，本页面路由不改写导致地址栏残留外来码；(b) UserReferral.vue normalizeUrlAccessCode：挂载后若 URL accessCode/access_code 与自身码不一致 → router.replace 归一化为自身码（无自身码则移除），仅改已存在参数不新增，保留其他 query；(c) 5 个新单测（改写/保留其他参数/相等不动作/无参数不动作/无自身码移除）+ 5 个测试文件 router mock 补 useRouter；(d) 生产实测：playwright 登录生产，/profile/referral/?accessCode=w25brikFiT（已删除 E2E 渠道码）加载后 URL 归一化为自身默认码，验证通过
- **Created**: 2026-08-24
- **Context**: 用户反馈「地址栏推荐码 w25brikFiT 与页面自身推荐码 9aaHjbryhL 不一致」。实证：w25brikFiT 系 E2E 测试账号 2026-08-20 创建、后经渠道删除 API 删除（生产库 referral_share_code / billing_referral_edge 均已无记录）；9aaHjbryhL 为 bootstrap-admin 自身默认码。二者本非同一概念，但地址栏残留他人码造成误解，且复制地址栏分享会归属错误。
- **Action**: (1) UserReferral.vue 挂载后归一化 URL 溯源参数（replace 不留历史）(2) UserReferral.accessCode.test.js 新增 5 例（含 w25brikFiT 场景）(3) 4 个相关测试文件补 useRouter mock（router.replace 返回 Promise）(4) vite 构建通过 + 生产已生效（index hash 一致 + chunk 含归一化逻辑）(5) playwright 生产实测 URL 归一化
- **Why**: 溯源参数与自身码语义不同但共存于地址栏，误导用户且复制地址栏链接分享时归属错误；归一化后地址栏与页面一致。
- **How to apply**: 已上线（public/html 指向新 release）；若后续回归，检查 UserReferral.vue normalizeUrlAccessCode 与 UserReferral.accessCode.test.js「URL accessCode normalization」describe。
- **扩展（2026-08-24，taskFE `968f177`）**: 从「推荐页局部归一化」扩展为「登录后全站归一化」——main.js 全局 router.afterEach（登录态门控）调用共享工具 `normalizeUrlAccessCodeToOwn`（history.replaceState，ownCode 参数 > 会话缓存 > 状态接口；同步旧机制 localStorage `referralCode`，防 referralUtils 点击处理器重新附加他人码；API 不可达保持 URL 原样）；UserReferral.vue 原 router.replace 实现删除改复用共享工具。新增单测 14 例 `referralAccessCodeUtils.normalize.test.js`；UserReferral.accessCode 归一化用例改断言 window.location 重写。taskFE 已登记精准重启，待点击 runAll「精准编译重启」生效。

## [OPT-20260823-026] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: shareLib 5abd6f2 + taskCloudService a8a2a63 + taskEvents cb6b90b + db 3beefd5 已推送：新增 shareLib/stopreason 单源包（TriggerLabel/AnnotateMessage/Codes），taskCloudService/taskEvents 停服触发文案改薄封装委托，消除两处 Go 码表漂移；db 新增 check_stop_reason_labels.py 断言前端 serverStartHistoryDisplay.js STOP_REASON_LABELS 覆盖 Codes()，接入 db pre-commit [4/4]，自测 4 例。taskCloudService 整包 205s + taskEvents 整包全绿。注：FE 从同一生成物读取未做（taskFE 有他会话活跃 WIP），以 CI 断言 key 集合覆盖收口
- **Created**: 2026-08-23
- **Context**: 停服启动日志要注明触发源头时，在 `taskCloudService/src/stop_reason_display.go`、`taskEvents/internal/handlers/cloudcommon/stop_reason.go`、`taskFE/app/src/utils/serverStartHistoryDisplay.js` 各维护一份 `stop_reason` 中文标签，后续加码容易漏改一侧。
- **Action**: (1) 在 shareLib 增加 `stopreason` 包（Go 码表 + 标签）(2) taskCloudService / taskEvents 改为引用该包 (3) 前端从同一 JSON/生成物读取，或至少用 CI 断言三份 key 集合一致
- **Why**: 三份码表漂移会导致「启动日志触发文案」与「服务器启动历史关闭原因」对同一码显示不同中文。
- **How to apply**: 新目录 `shareLib/stopreason/`；对照 `taskCloudService/src/stop_reason_display.go` 与 `taskEvents/internal/handlers/cloudcommon/stop_reason.go`

## [OPT-20260824-014] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 55998f6 已推送：ProjectRunTemplatePanel 补导入 extractTraceId，保存失败错误块恢复 data-traceId；回归测 ProjectRunTemplatePanel.traceId.test.js 3 例全绿
- **Created**: 2026-08-24
- **Context**: 全量 vitest 跑批发现 `ProjectRunTemplatePanel.traceId.test.js` 2 例失败 + 5 unhandled rejection（`extractTraceId is not defined`）。`app/src/components/ProjectRunTemplatePanel.vue:275/281/305/313` 调用 `extractTraceId` 但文件无 import；`git log -S` 定位为 commit `002001d`（换镜像按选中已安装镜像推导架构）引入使用处，当时无全量测试拦截。该函数在 `utils/apiUtils.js`（或 traceId 工具）存在但未导入，属于既有回归（OPT-20260807-057 traceIdFromHeaders 之后）。
- **Action**: (1) 核对 `extractTraceId` 的真实定义位置（`apiUtils.js` / `traceIdFromHeaders`） (2) ProjectRunTemplatePanel.vue 补 import 或改用既有 `traceIdFromHeaders`（与全站归一化口径一致） (3) 重跑 `ProjectRunTemplatePanel.traceId.test.js` 2 例全绿 + 全量无 unhandled
- **Why**: 保存失败时错误块 traceId 属性缺失，前端排障丢失链路钥匙；且 unhandled rejection 会污染 vitest 全量跑批结果。
- **How to apply**: `taskFE/app/src/components/ProjectRunTemplatePanel.vue`；测试 `src/components/ProjectRunTemplatePanel.traceId.test.js`

## [OPT-20260824-015] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 510a7c6 已推送：useLinkedProjectsRepoOAuth watch/clearOauthOkQuery 对 route 判空；回归测 TaskCardCommentsSection.test.js 2 例全绿
- **Created**: 2026-08-24
- **Context**: 全量 vitest 跑批发现 `TaskCardCommentsSection.test.js` 2 例失败：`composables/taskDetail/useLinkedProjectsRepoOAuth.js:344` 的 `watch(() => [route.query?.gitlab, route.query?.github], ...)` 中 `route` 为 undefined（composable 内未接收/注入 route，挂载时 watch 立即求值抛 TypeError）。commit `62eb077`（OPT-20260816-009 看板评论 @镜像 身份）引入。
- **Action**: 组件侧确保将 `useRoute()` 实例传入 composable（或 composable 内部 `useRoute()`，watch 内对 route 判空）；TaskCardCommentsSection.test.js 2 例全绿。
- **Why**: watch 求值抛 TypeError 导致评论卡 repo-identity 渲染路径在测试与边界场景崩溃；修复后全量 vitest 干净。
- **How to apply**: `taskFE/app/src/composables/taskDetail/useLinkedProjectsRepoOAuth.js`；测试 `src/components/TaskCardCommentsSection.test.js`

## [OPT-20260824-016] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 95d331a 已推送：vite.config.js test.exclude 排除 7 个 node:test 静态文件；node --test 13 例仍全绿
- **Created**: 2026-08-24
- **Context**: `vitest run`（无 include 配置）收集到 7 个 `node:test` 源码级静态测试（`useTaskDetail.bootstrapCloneLog` / `bootstrapFailureTrace` / `zlog-released`、`Register.accessCode`、`taskDetailSectionBindings.bootstrapFailureTrace`、`TaskDetailCommentsSection.bootstrapCloneLog`、`TaskDetailCommentsSection.test.js`），报「No test suite found」。这些文件用 `node --test` 运行（验证 6 pass/0 fail），与 vitest 套件并存但无排除配置。
- **Action**: `vite.config.js` test 块加 `exclude`（排除纯 `node:test` 文件）或将这些文件统一改名为 `*.node.test.js` 并纳入 node --test 脚本；确认全量 vitest 0 失败。
- **Why**: 全量 vitest 跑批被这 7 个假红干扰（每轮必报），掩盖真实失败信号。
- **How to apply**: `taskFE/app/vite.config.js`（test.exclude）；`taskFE/app/package.json`（node --test 脚本）

## [OPT-20260823-063] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 闭环：014/015/016 三项修复后 taskFE 全量 vitest 624 文件 3310 例清零（此前 10 文件失败 + 5 文件 0-test 误报）；useServerConfigRuntimeFetch 隔离重跑 14 例全绿
- **Created**: 2026-08-23
- **Context**: 2026-08-23 全量 `vitest run`（627 文件）10 文件失败：`useServerConfigRuntimeFetch.test.js`（POST body comment_id 1 例）、`TaskCardCommentsSection.test.js`（repo-identity 2 例）、`ProjectRunTemplatePanel.traceId.test.js`（data-traceId 2 例），另有 5 文件 0 test 报错（`Register.accessCode`、`useTaskDetail.bootstrapFailureTrace/zlog-released/bootstrapCloneLog`、`TaskDetailCommentsSection.bootstrapCloneLog`、`TaskDetailCommentsSection.test.js` 等，疑似并行导入/共享 mock 干扰）。stash 本次改动后基线重跑结果完全一致，确认与推荐分账面板无关。
- **Action**: (1) 逐个文件单独重跑定位根因（0-test 文件多半是 hoisted mock 跨文件污染或依赖缺失）(2) 修复或按既有约定标注 skip (3) 修复后全量复跑清零。
- **Why**: 全量套件带红会让 pre-commit 随机门禁随机踩雷、掩盖真实回归。
- **How to apply**: `taskFE/app` 下 `./node_modules/.bin/vitest run <file>` 单文件重放；对比 `git stash` 前后基线。

## [OPT-20260823-049] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 590a3fb 已推送：useTaskDetail 经 serverConfigRef 运行时面板构造 runtimeStatusFor(cid) 传入 zlog factory，taskDetailZTreeExecLogState 透传到 refresh deps；binding 仍 running 但运行时 Released 时 shouldSkipZTreeExecLogFetch 命中跳过 clone-log；回归测 taskDetailExecLog.test.js 新增 2 例全绿
- **Created**: 2026-08-23
- **Context**: 任务关联面板的 `containerReleased` 用 binding + 该评论 runtime 快照；zlog `shouldSkipZTreeExecLogFetch` 主要靠 `serverRuntimeNotServing` 与 `bindingStatusFor`。binding 仍为 running 且心跳尚未标 notServing 时，第一次仍会打 clone-log 拿 409。
- **Action**: (1) `useTaskDetail` 给 zlog 传入 `runtimeStatusFor: (cid) => runtimeStatusFromCommentPanel(serverRuntimeStatusPanel, cid)` (2) `refreshZTreeExecutionLog` 内层 deps 透传 `runtimeStatusFor` (3) 单测：runtime Released + binding running 时不请求 clone-log
- **Why**: 与徽章同一判定，避免释放后首屏短暂出现「容器正在注册业务地址」。
- **How to apply**: `useTaskDetail.js` 的 zlog 工厂参数；`taskDetailZTreeExecLogState.js` refresh 透传；`taskDetailExecLog.test.js`

## [OPT-20260824-017] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskEvents 1b40d56 + conf d9711b7：TASK_AUTH_INTERNAL_URL 兜底/env 端口 8001→8003（taskAuth 实际监听 8003，runAll stop_command lsof -ti:8003 佐证）；新增 runner_test.go 3 例回归测全绿
- **Created**: 2026-08-24
- **Context**: `conf/runAll.yaml` 中 `task-events-user-account-deletion-execute-scan-1-execute-due` env 注入 `TASK_AUTH_INTERNAL_URL: http://${INFRA_HOST}:8001`；`taskEvents/internal/handlers/useraccountdeletionexecute/runner.go:35` 的兜底默认同样为 `http://127.0.0.1:8001`；而 taskAuth 实际仅监听 **:8003**（`conf/auth/task-auth/config.yaml` port: 8003、runAll stop_command `lsof -ti:8003`、当前 `ss` 均一致；:8001 无任何监听）。该意图执行账户删除扫描时连错端口（connection refused），功能静默失效。
- **Action**: 确认 taskAuth 是否应提供内部账户删除端点：若该端点应在 8003 提供，则将 runner.go 兜底与 runAll.yaml env 改为 `:8003`；若意图已废弃（如删除流程已改走其他服务），则移除该意图或标注 deprecated。修改后跑 taskEvents 相关单测 + 意图实启冒烟。
- **Why**: 意图配置与真实监听端口漂移，导致账户删除扫描静默不可用；`docs/runbooks/manual-full-deployment.md` §15.1 已记录此已知问题。
- **How to apply**: `taskEvents/internal/handlers/useraccountdeletionexecute/runner.go`（兜底 URL）；`conf/runAll.yaml`（env 覆盖）；核对 `taskAuth` 是否有对应内部端点。

## [OPT-20260824-018] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: db 584c692 + meta 891d9e5f：新增 check_deployment_doc_paths.py 解析 manual-full-deployment.md 代码块 bash/./ 路径按 cd 上下文 stat 存在性；14 例自测 + pre-commit 注册全绿
- **Created**: 2026-08-24
- **Context**: `docs/runbooks/manual-full-deployment.md` 引用约 50 个脚本/配置文件路径与命令接口（build.sh / run.sh / health.sh / migrate.sh 等），已一次性人工核对通过；但无自动校验，后续脚本改名/移动会产生文档漂移。
- **Action**: 增加脚本对部署文档引用的路径做存在性校验（如 `docs/runbooks/check-deployment-doc-paths.sh`：从文档提取 `bash xxx.sh`/`./xxx` 引用 → stat 存在性 → CI/pre-commit 接入；或复用现有文档检查类脚本模式）。
- **Why**: 部署文档是事故恢复与人工部署的唯一依据，路径漂移会直接导致手动部署失败。
- **How to apply**: 新建 `docs/runbooks/` 下校验脚本（或在 docs 既有 CI 检查中追加），解析 `manual-full-deployment.md` 中命令路径并 stat。

## [OPT-20260824-002] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 已由 OPT-015 修复闭环复核：useLinkedProjectsRepoOAuth 对 route 判空后，TaskCardCommentsSection / ProjectRunTemplatePanel.traceId / useServerConfigRuntimeFetch 三文件 19 例 vitest 全绿（2026-08-24 复核，无遗留）。
- **Title**: taskFE 单测既有失败：useLinkedProjectsRepoOAuth route stub 缺失
- **发现**：2026-08-24，全量 vitest 中 TaskCardCommentsSection / ProjectRunTemplatePanel.traceId / useServerConfigRuntimeFetch 共 4-5 例持续失败（暂存本人改动后仍失败，非本次 Home 页脚改动引入）
- **根因**：`taskFE/app/src/composables/taskDetail/useLinkedProjectsRepoOAuth.js:344` watch `route.query`，相关测试未 stub route → `Cannot read properties of undefined (reading 'query')`；连带 Vue 渲染 Unhandled Rejection（patch null vnode）
- **建议**：相关测试为组件提供最小 route stub（或 composable 内 route 兜底 `route || {}`），并复跑全量套件确认清零

## [OPT-20260824-003] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 9e798e3：HomeFeatureCard 根节点补 bg-white（shadow-lg 投影锚点）+ 新增 HomeFeatureCard.test.js 断言 bg-white+shadow-lg；Home.feature-cards + 新测全绿，已推送。
- **Title**: 首页核心功能卡补 bg-white 增强阴影可视性
- **发现**：2026-08-24，HomeFeatureCard 补 p-8 内边距时确认卡片无背景色（section 白底），shadow-lg 在白底上几乎不可见，卡片与背景界限模糊
- **建议**：卡片根节点 class 追加 `bg-white`（shadow-lg 才有投影锚点），三卡在 grid gap-10 上呈现清晰的浮起卡片感
- **How to apply**: `taskFE/app/src/components/HomeFeatureCard.vue` 根 div class；改后跑 `Home.feature-cards` 单测 + runAll 精准编译重启 taskFE

## [OPT-20260824-004] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 0b34495：nginx.conf 加 location /api/ 转发 docker host 网关 172.17.0.1:18081（APISIX 按路径路由，401 已证路由可达）；容器内 nginx -t 通过；live reload 待 runAll 恢复后触发。
- **Title**: taskFE 本地 4000 nginx 缺 /api/referral/ 代理
- **发现**：2026-08-24，验证渠道删除功能时发现本地 4000（taskFE 静态 nginx）把 `/api/referral/channels/` 当作 SPA 路由返回 index.html（200 HTML），须经 18081 taskGateway 才能正确代理 API；开发直连 4000 时 referral API 全部失败
- **建议**：taskFE nginx.conf 增加 `/api/` 转发到网关（或 docs 注明本地访问须经 18081）；改后 `nginx -t` + reload
- **How to apply**: `taskFE/nginx/nginx.conf` location /api/ proxy_pass；或更新 taskFE README 访问说明

## [OPT-20260824-006] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskReferral 564dac3 + taskFE 6c374b6：listReferralApplications 批量水合 referral_share_code 默认码（share_code 字段，is_default=1 active）；面板加「推荐码」列+复制按钮；Go 回归测 2 例 + FE 1 例全绿。
- **Title**: 推荐码申请列表每行展示该用户默认推荐码
- **发现**：2026-08-24，实现推荐码反查后注意到 applications 表格无各用户的推荐码列，管理员若想同时看「申请者 ↔ 其推荐码」需另行反查
- **建议**：listReferralApplications SQL 经 referral_share_code 关联 user_id 取默认码（is_default=1 且 active），加「推荐码」列；或前端逐行调用反查接口（N+1，不推荐）
- **How to apply**: `taskReferral/src/referral_code_store.go` JOIN referral_share_code；面板表格加列 + 复制按钮

## [OPT-20260824-007] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 82d07f6：反查命中用户在当前申请列表时行高亮 ring-2 + scrollIntoView 居中定位；不在列表不误伤；FE 回归测命中/未命中 2 例全绿。
- **Title**: 推荐码反查结果与申请列表联动跳转
- **发现**：2026-08-24，实现推荐码反查后：反查结果卡片与下方申请列表无联动，管理员找到用户后若该用户有申请需手动翻找
- **建议**：反查结果命中时，若该 user_id 存在于当前列表，滚动高亮对应行；或结果卡片加「查看申请」按钮（带 user_id 过滤参数扩展 applications 接口）
- **How to apply**: `taskFE/app/src/components/SystemAdminReferralApplicationsPanel.vue` 高亮/滚动逻辑；后端 applications 接口可加 user_id 过滤参数

## [OPT-20260824-010] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 复核确认已处理无需改动：Login.vue 经 captureReferralAccessCodeFromSearch 把 ?accessCode= 写入 sessionStorage，wechatLoginFlow.js 读取并透传 taskAuth 绑定推荐关系（OPT-20260820-036 链路）。登录页不展示分享信息属设计使然。
- **Title**: Login.vue 是否同步处理 accessCode 场景确认
- **Created**: 2026-08-24
- **Context**: 分享链接 ?accessCode= 进入登录页时，登录页未展示任何推荐信息；注册页已处理豁免，登录页语义需确认（accessCode 是否应保留到注册/绑定场景，或登录页提示分享者信息）
- **Action**: 检查 Login.vue 是否消费 accessCode（referralAccessCodeUtils 已在注册页使用）；确认「登录时 accessCode 无绑定语义」或补提示
- **Why**: 分享链接同时被用于登录与注册，行为不一致可能造成用户困惑
- **How to apply**: `taskFE/app/src/views/Login.vue` + `referralAccessCodeUtils.js` 调用链审查；如无必要则文档注明

## [OPT-20260824-013] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: db 90368a2 + meta 54e150b8：新增 check_copyright_ssot.py 扫描第一方 README 版权行强制等于 Copyright (c) 2025～2026 author@example.com（排除 Archi/third_party/trae-agent/gitService/sdk）；13 例自测 + pre-commit 注册全绿。
- **Title**: 版权声明 SSOT 防漂移检查（pre-commit 钩子）
- **Created**: 2026-08-24
- **Context**: 20 个第一方 README.md 的版权行（`Copyright (c) <年份> <所有人>`）为手工维护，本次批量修改时发现各仓格式已统一但无校验；年份/所有人漂移只能靠人工扫描发现
- **Action**: 在 `.pre-commit-config.yaml` 增加一次性检查钩子：扫描第一方 README（排除 `docs/architecture/Archi/**`、`**/third_party/**`、`trae-agent/**`、`gitService/**`、`sdk/**`）中版权行，强制等于 `Copyright (c) 2025～2026 author@example.com`，不匹配即失败
- **Why**: 版权声明是法律面文本，漂移难被发现；SSOT 化后任何人改动即被门禁拦截，避免再次全仓排查
- **How to apply**: 仿照 `check-wechatpay-go-sdk` 写 `db/scripts/ci/check_copyright_ssot.py`（只读检查），注册进 `.pre-commit-config.yaml`；注意排除第三方目录清单要与本次扫描一致

### OPT-20260824-052 — OIDC refresh token 环境级联验证（迁移 SQL + 服务重启 + 插件 E2E）

- **Status**: pending
- **Created**: 2026-08-24
- **Context**: taskAuth 已实现 offline_access/refresh_token grant（040 迁移 + 代码，单测全绿）；插件侧已实现静默续期（单飞锁 + invalid_grant 清空）。但未在真实环境联调：runAll「精准编译重启」task-auth 后迁移 SQL 由 dataMigrate 执行，需验证 auth_oidc_refresh_token 表实际创建；插件需经 oauth-callback.html 真实登录流验证获取到 refresh_token。
- **Action**: ① runAll 精准编译重启 task-auth（已登记）→ 确认迁移执行 + `SHOW CREATE TABLE auth_oidc_refresh_token`；② 浏览器真实插件登录 → storage 确认 refreshToken 落盘；③ 手动将 tokenExpiresAt 拨到过去 → 触发 getAuthStatus → 确认静默刷新成功且无重新登录；④ 观察 30 天 TTL 内多次轮换链路。
- **Why**: 单测覆盖逻辑正确性，但迁移执行与真实扩展存储链路需环境验证才闭环；插件 manifest/网关代理对 /api/oidc/token POST 的兼容性未在真实流量下确认。
- **How to apply**: `dataMigrate/taskAuth/040_oidc_refresh_token.sql`（已提交）；runAll 页面「精准编译重启」；插件 `test/sw-token-refresh.test.js` 已在 VM 沙箱覆盖 SW 行为，环境验证为最后一道闭环。

## [OPT-20260824-053] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: db `1e1e094`：dbload `cloneMySQLSchema` 钉单 `*sql.Conn` 执行全部克隆语句（`SET SESSION FOREIGN_KEY_CHECKS=0` 与 `USE dst` 为会话级，pool 把 CREATE/INSERT 分到其它连接时含外键表按字母序复制会偶发 errno 150——taskAuth「首轮 FAIL 后全绿」的可疑根因）；收尾重置 FK=1 + 默认库=information_schema 防会话污染回池。新增含外键 schema 克隆回归测 `TestPrepareClonedTestDBClonesForeignKeys`；dbload 整模块测试 + taskAuth 全量 `go test -count=1 ./src/...`（120s）均全绿。
- **Title**: taskAuth 全量测试套件偶发失败需定位（首轮 FAIL 后两次全绿）
- **Created**: 2026-08-24
- **Context**: 2026-08-24 02:04 首次 `go test ./...` 出现 FAIL（输出被截断未捕获用例名），随后两次 `-count=1` 全量运行均全绿（132-142s）。疑似 dbload MySQL 测试库克隆/清理偶发竞争或环境抖动；OIDC 相关用例独立运行多轮无失败。
- **Action**: 下次全量回归若再 FAIL，先捕获失败用例名；排查 dbload.OpenTestMySQLClonedFromDir 并发克隆是否偶发超时；必要时在 CI 中对该辅助函数加重试。
- **Why**: flaky 测试会掩盖真实回归，影响门禁可信度。
- **How to apply**: `go test -count=1 ./...` 捕获 `--- FAIL` 输出；`db/dbload` 克隆逻辑。

## [OPT-20260824-012] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 489ba78：Navbar 品牌区加 favicon.png 小尺寸 logo + 产品名「云端开发」，替换纯文本 AI项目推进平台；JS 字符串避免 transformAssetUrls 解析；Navbar.ui.test.js 新增品牌区 1 例全绿。
- **Title**: 其余品牌面 logo 应用（Navbar 品牌区等）
- **Created**: 2026-08-24
- **Context**: 公司 logo 已应用于 favicon + 登录页 + 管理员登录页 + Chrome 插件图标；产品内导航栏（Navbar.ui.vue）目前无品牌 logo/名称展示，登录后的全局品牌感仍缺失
- **Action**: 在 Navbar 左侧品牌区加入 logo（/favicon.png 小尺寸）+ 产品名「云端开发」，保持与登录页一致的视觉
- **Why**: 登录页到工作台的品牌连续性；客户/内部演示时全局可见品牌标识
- **How to apply**: `taskFE/app/src/components/Navbar.ui.vue` 品牌区 + `taskFE/app/src/views/WorkPanel.vue`（如含品牌头）；参考登录页 `w-16 h-16 rounded-2xl` 样式

## [OPT-20260823-038] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE 已提交 6e38217 推送：容器已释放时短路写路径（指令发送/文件树/job 动作）但保留 SaaS job 日志（COS step_full）。exec-log released 仍拉 job 日志不拉 clone-log；vitest 586 例全绿。精准重启已登记 taskFE。
- **Created**: 2026-08-23
- **Context**: 徽章「服务器已释放」时已隐藏指令/文件树/打开容器。`shouldSkipZTreeExecLogFetch` 现在只跳过打容器的 clone-log 与层变动 prefetch，SaaS `container-job-execution-log`（COS step_full）仍要拉。发送指令、文件树刷新等其它容器 HTTP 仍可能被 watcher 触发。
- **Action**: (1) 盘点任务详情里所有打评论容器的写/转发 fetch（指令发送、文件树、打开容器 URL、业务地址探测） (2) `containerReleased` / `isCommentExecutionReleased` 时跳过这些容器路径，但不得跳过 job-execution-log (3) 补测：released 时对应容器 `apiFetch` 不被调用，job-execution-log 仍被调用
- **Why**: 写路径 409 会误导用户；历史步骤必须仍能复查。
- **How to apply**: `taskFE/app/src/composables/taskDetail/` 下指令与文件树 composable；对齐 `taskDetailExecLog.js` 的 clone-log skip，不要复用「整段 refresh 直接 return」

## [OPT-20260823-043] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill aefcb59 + taskFE e08ae7e 已推送：GET /api/billing/profit-sharing/referrer-pending/ 统计本人微信 receivers.result=PENDING 笔数与最晚截止（支付+30 天）；ReferralProfitSharingConfirmNotice 拉取并展示「待确认 N 笔」。Go+vitest 全绿；精准重启已登记 taskFE/task-bill。
- **Created**: 2026-08-23
- **Context**: `/profile/referral/` 已注明须在微信服务通知有效期内点击确认。平台侧 `billing_profit_sharing` 若长期 `PENDING`，用户仍不知道是哪几笔、还剩多久。
- **Action**: (1) 在 taskBill 查询该用户 `PENDING` 分账（微信 `receivers.result=PENDING`）及冻结截止（支付成功+30 天）(2) 在推荐资格卡下展示「待确认 N 笔，最晚 YYYY-MM-DD 前打开微信确认」(3) 补单测：无 PENDING 不展示；有则展示截止日期。
- **Why**: 纯文案无法对应到具体订单，用户仍可能漏点服务通知导致无法补分。
- **How to apply**: `taskBill` 分账查询 + `UserReferral.vue`；测例挂 `UserReferral.profitSharingConfirmNotice.test.js`

## [OPT-20260823-059] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 复核闭环：taskFE 5682638 已实现（ReferralProfitSharingOrdersPanel.vue）——推荐人页面新增订单分账面板（冻结/可分账/已过期三态 display_status 徽标 + shareable 按钮态），点「分账」confirm 后 POST /share/（createClickGuard Idempotency-Key 防重），409/404/502 错误文案本地化 + data-traceId；8 例 vitest 全绿，UserReferral.vue 已挂载面板
- **Created**: 2026-08-23
- **Context**: 推荐人手动分账 API（列表 + POST share）已实现并测试，但前端「推荐人分账中心」页面尚未开发；15–30 天窗口、frozen/processing/shared 展示状态仅存在于 API 响应。
- **Action**: (1) taskFE 推荐人页面新增分账列表（display_status 徽标 + shareable 按钮态） (2) 点击「分账」→ 确认弹窗 → POST share（带 Idempotency-Key 防重） (3) 409/404/502 错误文案本地化
- **Why**: API 无 UI 则功能不可达；窗口状态可视化可减少「为什么不能分账」的客服咨询。
- **How to apply**: `taskFE/app/src/views/` 推荐人相关视图；`apiFetch` + `createClickGuard` 复用既有模式

## [OPT-20260824-056] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskFE `b25c0fc` 已推送：新增 `app/src/build/vite-plugin-build-time-meta.js`（仅生产构建向 index.html 注入 `<meta name="build-time" content="<ISO-8601 UTC>">`，纯函数可测）+ `app/vite.config.js` 注册 + `tests/helpers/prodBuildFreshness.js`（parseBuildTimeFromHtml / buildTimeToMs / latestSourceMtime / compareFreshness / checkProdBuildFreshness，支持 Playwright request）+ `tests/ProdBuild.freshness.playwright.test.js`（E2E 固化「先校新鲜度」，漂移默认仅告警、`REQUIRE_FRESH=1` 时升级失败）+ `.sh` 包装器。plugin 5 例 vitest + helper 8 例 node:test 全绿；taskFE 全量 vitest 627 文件 3333 例清零。另修 pre-existing `Navbar.collapsed.test.js` 品牌文案断言（d874a74，旧「AI项目推进平台」→「云端开发」）一并推送。部署复验（:4000 实际 meta）待 runAll 恢复。
- **Title**: E2E UI 断言前校验 :4000 产物服务与源码一致性
- **Created**: 2026-08-24
- **Context**: 本次 E2E 调试根因：:4000 产物服务承载旧构建（public/html → 旧 release，含「手机号/验证码」Tab），UI 断言失败并触发 Playwright worker 重启级联（新 worker 新手机号未注册 → not_found），浪费多轮诊断。构建产物与源码漂移无检测机制
- **Action**: 在 E2E 测试 beforeAll 或 CI 前检查产物新鲜度：对比 `public/html` 构建时间戳与源码最近修改时间，或产物内嵌 build-time 标记；过期则警告或自动触发 atomic-vite-build.sh
- **Why**: 产物/源码漂移是本类 E2E 失败最隐蔽的根因来源，自动化检测可把诊断时间从小时级降到秒级
- **How to apply**: 产物构建时写入 `__BUILD_TIME__` 常量（Login chunk 或 index.html meta），E2E 读取并与源码 mtime 比较；或 runAll 面板构建后校验

## [OPT-20260823-017] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: db cdb833a + dataMigrate 51cd3bb：新增 dataMigrate/taskAuth/041_impersonation_and_inbox_archive.sql（CREATE TABLE LIKE 复制热表结构 + 去唯一键 + archived_at，040 一并补应用）与 db/scripts/archive_impersonation_sessions.py（会话 ended/expires 超期 + 关联收信箱成对归档 + 无会话孤儿收信箱，每批≤1000 单事务，--dry-run/--self-test，4 例 pytest 全绿；生产 dry-run sessions=0 inbox=0）；已应用生产 task_auth（040/041 data_migrate_log），cron 周日上午 4:10 已注册（logs/archive-impersonation-cron.log）
- **Created**: 2026-08-23
- **Context**: `auth_impersonation_session` 与 `auth_user_inbox_message` 为时间累积型审计表，本期只建热表。年增量预估 <10 万，仍应按冷热分离补归档。
- **Action**: (1) 增加定时任务把 `ended_at` 或 `expires_at` 超过 90 天的行迁到归档表或删除（合规需保留则归档）(2) 分批 ≤1000 行 (3) 指标监控热表行数
- **Why**: 热表无限膨胀后索引与审计查询变慢。
- **How to apply**: `dataMigrate/taskAuth/` 新归档表或 TTL 清理；对照 `db/scripts/ensure_partitions.sh` 模式

## [OPT-20260823-066] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: AiMonitor `f4db363` 已推送：新增 scripts/disk_capacity_reporter.py（宿主 cron 写 textfile：repo_tmpfs_{size,used,avail}_bytes 覆盖 /tmp/ram-work 仓库 tmpfs + ramsync 持久盘，repo_data_dir_size_bytes 覆盖 data/taskbill_invoice_files 票面目录）；docker-compose.yaml node-exporter 开启 textfile collector + 挂载 /home/ljy/ramwork-recovery/node-exporter-textfiles:ro；prometheus 注册 disk-capacity-alerts.yml（RepoTmpfsUsageHigh 85%/Critical 95%、InvoiceFilesDirSizeHigh 2GB、DiskCapacityReporterDown absent）；8 例自测全绿，pre-commit 全绿已推。生产应用：node-exporter/prometheus 已重建、配置生效，prometheus rules 6 组含 repo-disk-capacity 全 ok 且 4 规则 inactive（tmpfs 66% < 阈值）；cron */5 注册（logs/disk-capacity-reporter-cron.log）。备份确认：data/ 已 gitignore 不入 git，但不在 ramsync-daemon EXCLUDE（仅 dockerInfra/mysql/data/），RAM→disk 双向同步覆盖发票目录；taskBill 宿主进程以 ljy 写盘 rsync 可读。(3) 按发票 id 归档为可选未做。
- **Created**: 2026-08-23
- **Context**: 手动开具发票文件落 `<repoRoot>/data/taskbill_invoice_files/`（data/ 已 gitignore），为租户票面文件，属业务数据；无备份/容量告警，磁盘满会导致上传失败。
- **Action**: (1) 确认 ramsync/备份清单包含该目录（参考 mysql/data 防护：确保不被 EXCLUDE 误伤） (2) 磁盘容量告警覆盖该目录增长 (3) 可选：按发票 id 归档过期文件
- **Why**: 票面文件丢失影响税务合规留存；容量告警缺失时上传静默失败。
- **How to apply**: `scripts/`（备份清单）；AiMonitor 磁盘告警规则

## OPT-20260824-008 — [pending] 生产 task-referral 配置 TASK_REFERRAL_INTERNAL_SECRET
- **Created**: 2026-08-24
- **Context**: 新增内部端点 `POST /api/internal/referral/share-code/validate/`（requireInternalSecret 保护）；生产 task-referral 未配置 `TASK_REFERRAL_INTERNAL_SECRET`，未配置时 requireInternalSecret 放行 → 内部端点无密钥保护裸奔
- **Action**: 生产环境 task-referral 启动配置注入 `TASK_REFERRAL_INTERNAL_SECRET`（强随机值），并在 taskAuth 侧配置同名密钥以携带 `X-TaskReferral-Internal-Secret` 头（referral_bind_client.go 已支持）
- **Why**: 该端点按 share_code 反查 owner，无密钥保护可被外部枚举有效分享码并探测 owner 用户
- **How to apply**: runAll 托管环境变量 / .env；配置后重启 task-referral + task-auth，验证 validate 端点 401（无密钥时拒绝）

## OPT-20260824-009 — [pending] 注册页 access_code 完整链路端到端验证
- **Created**: 2026-08-24
- **Context**: 实现 access_code 豁免 invite_code 后，线上仅验证了「有 accessCode 隐藏邀请码字段 + 无 accessCode 字段照常」；未走通「真实手机验证码 → 提交注册 → 绑定推荐关系」的完整链路（需真实短信验证码，手动操作）
- **Action**: 用真实手机号走完整注册：有效 access_code 注册成功且无邀请码要求；无效 access_code 提示「邀请链接无效」且用户不创建；成功后 /profile/referral/ 可查绑定关系
- **Why**: validate 与 redeem 豁免逻辑虽单测覆盖，端到端可确认「注册成功 + 不强制邀请码 + 推荐绑定不丢」三者同时成立
- **How to apply**: 浏览器 http://127.0.0.1:18081/auth/register/?accessCode=<有效码> 完整注册；查 referral_share_code 绑定记录

## OPT-20260824-011 — [pending] 公司 logo 落地后补同步 docs/taskFE 指针并线上验证
- **Created**: 2026-08-24
- **Context**: 公司 logo（docs/logo/icon128.png）应用到 docs、taskChromePlugin、taskFE 并各自提交；但 docs/taskFE 子仓存在他会话遗留的 README.md 版权行改动（未提交）→ 按规则 32 只同步了 clean 的 taskChromePlugin 指针，docs/taskFE 指针待脏 WIP 解决后补同步
- **Action**: ① README.md 版权变更提交后，在 meta 仓库补同步 docs/taskFE 指针；② 在 runAll 页面（http://10.2.150.68:9999/）执行「精准编译重启」后，浏览器验证 taskFE favicon、登录页/管理员登录页 logo 实际展示
- **Why**: 指针不同步则 meta 仓库无法追踪子仓新提交；线上验证确认 favicon 与登录页品牌实际生效（静态资源走 nginx 缓存，需确认 cache-bust）
- **How to apply**: `git add docs taskFE && git commit`（子仓 clean 后）；重启后访问 http://10.2.150.68:18081/ 检查 favicon 与登录页

## [OPT-20260822-002] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: runAll+conf 已推送：8 个直连 task-cloud-service 的 task-events 消费者（release-servers / cloud-server-started/stopped/start-auto / relay-lifecycle-1..4）补 depends_on task-cloud-service。runAll 新增两条回归测：TestProductionConfig_ReleaseConsumerDependsOnCloud（生产配置断言）+ TestServiceCascadeOrchestrationService_ReleaseConsumerOrderedAfterCloud（DAG 顺序：start-all 消费者晚于 cloud、stop-all 逆序先于 cloud）。整包 go test 绿。全部重启空窗消费者不再裸奔消费打满 DLT；durable backoff（OPT-20260822-004）另行部署。
- **Created**: 2026-08-22
- **Context**: 2026-08-22 10:32 精准重启后紧接两次「全部重启」，`TASK_STATUS_CHANGED` 释放消费者在 `:8018` connection refused 时进 DLT。共享 runner 已按 `_retry_attempt` 拉长退避（约 181s），仍短于 76 个服务的 restart-all。
- **Action**: (1) runAll restart-all 在停依赖服务前先停 `task-events-*` 消费者，启动完成后再拉起；(2) 或给 restart-all 加「drain consumers」阶段；(3) 回归：模拟 cloud 停 3 分钟时 release 消费者不在 20s 内 DLT。
- **Why**: 退避只能覆盖单服务闪断；全量重启仍会把进行中的终态释放打进 DLT，看板 SSE 已成功会造成「任务已取消但机器未释放」的错觉。
- **How to apply**: `runAll/src/` restart-all 编排顺序；`conf/runAll.yaml` `task-events-*` depends_on；对照 `.ai/09_failure_experience/02_runtime_errors/110_task_status_changed_dlt_cloud_refused_retry_backoff_reset.md`

## [OPT-20260823-013] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskBill c14c16a + conf 6c3d6b0 已推送：evaluateGitlabTrafficGate 未映射 tenant-{id} 仓新增 conf 开关 trafficGateUnmappedReject（默认 false 保持 fail-open，置 true 返回 UNMAPPED_PROJECT 拒绝 + slog.Warn）；env TASKBILL_TRAFFIC_GATE_UNMAPPED_REJECT 覆盖；回归测 UnmappedProjectFailOpen/Reject 双态全绿，taskBill 整包 67s 绿；盘点+迁组完成后可置 true
- **Created**: 2026-08-23
- **Context**: ADR-0036 闸门对 `project_path` 不含 `tenant-{id}` 的仓库 fail-open，避免误伤个人命名空间。存量未迁组的租户仓公网 clone 仍可能绕过流量配额。
- **Action**: (1) 盘点各区域 GitLab 上非 `tenant-{id}/` 路径但属于租户的项目 (2) 补迁组或在闸门按 GitLab namespace 自定义属性解析 tenant_id (3) 对仍无法映射的项目改为 conf 开关默认拒绝并打 warn
- **Why**: fail-open 会让未按约定建组的仓继续消耗出站流量，与「未预购则阻断」产品语义不一致。
- **How to apply**: `gitService/initializers/zzz_trae_gitlab_traffic_quota.rb`；`taskBill/src/gitlab_traffic_gate.go` `parseTenantIDFromGitlabProjectPath` / `trafficGateUnmapped`

## [OPT-20260821-023] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: AiMonitor c52909c 已推送：新增 loki/rules-files/cloud-orphan-reconcile-alerts.yml —— OrphanReconcileDeleteWithoutKeepInstance（keep_instance_id 空删除，critical）/ OrphanReconcileDeleteFailedOnInitializing（Initializing|LastTokenProcessing 删除失败，warning）/ OrphanReconcileSkipUnknownAgeBurst（skip_unknown_age 突发，warning）；按 tracelog JSON msg 字段 regex 过滤；Loki 采集管道恢复后即生效
- **Created**: 2026-08-21
- **Context**: 任务 `task_878583341551480832` 新评论 CSC 尚未回填 `instance_id` 时，`reconcileOrphanInstancesByName` 把刚 `RunInstances` 的 ECS 当孤儿删除（`keep_instance_id=`、`LastTokenProcessing`/`Initializing`）。代码已跳过年龄未知/分钟级 CreationTime，但仍需可观测。
- **Action**: (1) 在 Grafana/Loki 对 `orphan_reconcile_delete` 且 `keep_instance_id=` 为空、或 `orphan_reconcile_delete_failed` 含 `Initializing`/`LastTokenProcessing` 建告警 (2) 同时盯 `orphan_reconcile_skip_unknown_age` 突发量 (3) 用历史 trace `911ae4e16c0990808a1c5e41` 校验查询
- **Why**: 宽限期或 CreationTime 解析再漏时会真实删掉正在启动的机器，仅靠代码守卫不够。
- **How to apply**: `AiMonitor/` Loki 规则；检索 `task-cloud-service` `event=orphan_reconcile_delete`

## [OPT-20260823-048] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: taskReferral 72c748a + taskFE c4ec470 已推送：新增 POST /api/referral/legal-name/（校验 legal_name → 更新 referral_code.legal_name → 重新 ensure 微信分账接收方，taskBill ensure 端点已支持 legal_name 入参）；FE has_active_code 且 legal_name 为空时展示补填表单（实名警示文案 + createClickGuard + Idempotency-Key），成功后回显姓名 + 接收方状态。Go 4 例 + FE 2 例回归全绿，pre-commit 全绿。已登记精准重启 task-referral/taskFE。
- **Created**: 2026-08-23
- **Context**: 申请推荐资格现已必填 `legal_name` 并传给微信分账接收方。存量已批准用户 `referral_code.legal_name` 为空，添加接收方仍省略 `name`，实名校验未做。
- **Action**: (1) 资格页对 `legal_name` 为空的已获资格用户展示补填表单与同样警示 (2) 校验通过后更新 `referral_code.legal_name` 并再次 ensure 微信接收方 (3) 补单测：空名称不写微信 name；补填后 payload 含 Name。
- **Why**: 填错无法分账的约束只覆盖新申请；存量推荐人仍可能因缺姓名被微信拒绝。
- **How to apply**: `taskReferral/src/referral_legal_name.go`、`taskBill/src/wechat_profit_sharing_receiver.go`；前端 `ReferralQualificationApplyForm.vue`

## [OPT-20260821-002] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: Loki 采集已恢复：aimonitor-promtail 容器缺失（docker-compose restart:no，06:54 监控栈重启后未随起）→ 经 runall-local-promtail.sh up 拉起（tail /tmp/ram-work/logs → aimonitor-loki:3100）。Loki /labels 现返回 job/level/service/trace_id；{job="task-cloud-service"} | trace_id="<id>" 与跨服务 {job=~".+"} |= "<id>" 实测命中。历史 trace（98b2b77e/6abd23eb）已被整点 truncate 清空，以现网新 trace 复验。AiMonitor 无 WIP，仅队列迁移。
- **Created**: 2026-08-21
- **Context**: 本页 runtime-status `data-traceid=98b2b77e-1c33-4761-80e4-e120a2ddd610` 与启动 TraceId `6abd23eb9234860f8c5f26a7` 在 Loki `http://10.2.150.68:3100` ready 但无 labels/无日志，只能回落到 `logs/task-cloud-service.log`。
- **Action**: (1) 检查 Promtail/Alloy 是否在采 `logs/*.log` (2) 确认 job label 与 `tracelog` JSON 字段 (3) 用上述 trace_id 在 Grafana 复验能拉到 start-vm 400
- **Why**: 有 data-traceId 却查不到 Loki，排障退化成本高，且违反 trace-first 门禁的可操作性。
- **How to apply**: `AiMonitor/` Promtail/Alloy 配置；对照 `.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`。

## [OPT-20260820-017] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 同 21-002：根因=aimonitor-promtail 容器未运行（docker-compose restart:no，06:54 监控栈重启后未拉起）。经 runall-local-promtail.sh up 拉起后 tail /tmp/ram-work/logs/*.log；Loki /labels 现含 job/level/service/trace_id，{job=~".+"} |= "<trace_id>" 实测返回 task-task-service 行。历史 trace a2d081e7-7d92-4222-8c9c-ef31c98ddcac 已截断，以现网 trace 复验。AiMonitor 无 WIP，仅队列迁移。
- **Created**: 2026-08-20
- **Context**: 排障 `a2d081e7-7d92-4222-8c9c-ef31c98ddcac` 时 Loki `:3100/ready` 为 ready，但 `/loki/api/v1/labels` 无 data、`{job=~".+"}` 近 30 天 0 条。同 trace 在 `logs/task-container-gateway.log` / `task-cloud-service.log` 可完整重建。与 OPT-20260820-009 相关但范围是整条采集管道而非 Status UI。
- **Action**: (1) 检查 promtail-local / runall-local-promtail 是否在跑、scrape `logs/*.log` (2) 确认 Loki 多租户/保留期未丢掉 chunks (3) 用该 traceId 复验 `{job=~".+"} |= "<id>"` 有结果
- **Why**: 前端已带 data-traceId，Loki 空则只能 grep 磁盘日志，跨服务排障退化。
- **How to apply**: `AiMonitor` promtail-local；`runAll/scripts/runall-local-promtail.sh`；`conf/runAll.yaml` observability.local_promtail

## [OPT-20260823-037] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: (1) 根因修复：aimonitor-promtail 容器缺失（restart:no 未随监控栈重启）经 runall-local-promtail.sh up 拉起，Loki 采集恢复（07:10 窗）。(2) 短窗归档落地：runAll cb9eda5 已推送——scripts/lib/log-archive.sh（archive_short_window 截断前保留每文件尾窗默认 8MiB gzip 落 logs/archive/<batch>/；prune_archive_batches 保留最近 7 批；失败只告警不阻断）+ truncate-ram-work-logs.sh 四个截断目录前接入，与 promtail 采集双保险；lib 功能自测 + grep 断言扩展全绿。(3) taskGitOauth merge 路径 Tempo span 评估为低增量价值——merge 各阶段已有结构化 stage 日志带 trace_id + elapsed_ms 经 Loki 可检索，OTEL export 未激活，不再补 Tempo span。
- **Created**: 2026-08-23
- **Context**: 任务详情一键合并超时（traceId `ab3e126c-2618-47c4-ba4f-675948d61ad9`，17:57）时 Loki `{job=~".+"}` 无任何日志、`loki_ingester_memory_chunks=0`。18:00 cron `truncate-ram-work-logs.sh --all` 清空 `logs/task-git-oauth.log`，只剩 APISIX access 与 MySQL 审计可还原。
- **Action**: (1) 查清 otel-collector/promtail 为何不向 Loki ingest (2) truncate 前确认 Loki/独立 sink 已收到 JSON，或对 `http_request`/`merge_request_*` 事件做短窗口归档 (3) taskGitOauth merge 路径补 Tempo span（至少 status/merge 两段）
- **Why**: 有 data-traceId 却查不到 Loki，迫使每次事故翻网关 JSON；整点截断让 3 分钟内的 stdout 永久丢失。
- **How to apply**: `AiMonitor` 采集配置、`runAll/scripts/truncate-ram-work-logs.sh`、`taskGitOauth/src/merge_request_git_client.go`
- **2026-08-24 夜**: **(1) 根因已定位并修复** — aimonitor-promtail 容器缺失（docker-compose `restart: no`，06:54 监控栈重启后未随起，otel-collector/tempo/loki 均已在跑但 promtail 不在），经 `runall-local-promtail.sh up` 拉起后 tail `/tmp/ram-work/logs/*.log`，Loki `/labels` 现含 job/level/service/trace_id，`{job=~".+"} |= "<trace_id>"` 实测命中（同窗 21-002/20-017 迁出）。（2）（3）仍 pending：整点 truncate 抹 stdout 的短窗口归档/事前 sink 确认设计未落地；taskGitOauth merge 路径 Tempo span（status/merge 两段）待评估 OTEL export 是否激活后补。

## [OPT-20260824-059] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 已执行：filter-branch 全历史清除 + untrack + .gitignore 防护 + force push origin（github）历史净化，历史 0 pem；gitlab 实例残留见 OPT-20260824-061
- **Created**: 2026-08-24
- **Context**: `taskChromePlugin/extension-dev.pem`（PKCS#8 私钥，与 manifest `key` 公钥 SHA256 配对一致，2026-07-04 首次提交）被 git 跟踪且历史永久存在。拥有该私钥者可对同一扩展 ID 签名伪装版本。
- **Action**: (1) 用 git filter-repo / BFG 从全部历史清除该文件 (2) `.gitignore` 增加 `*.pem` 防再犯 (3) 评估是否轮换密钥——**注意轮换后扩展 ID（派生自公钥）会变化，已安装用户需重新加载/重装**，需在安全性与兼容性间决策
- **Why**: 签名私钥泄漏 = 供应链伪装风险；即使仓库私有，也不应留存可签名资产。
- **How to apply**: `taskChromePlugin/`（filter-repo 需全仓协作，改动历史前先备份与沟通）；`taskChromePlugin/.gitignore`

## [OPT-20260824-060] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 已执行：taskChromePlugin/taskFE/taskReferral .gitignore 补 *.pem/*.key/*.p12/*.jks/.env*，全量子仓巡检完成
- **Created**: 2026-08-24
- **Context**: `taskChromePlugin/.gitignore` 仅覆盖 node_modules/.DS_Store/*.crx/*.zip/test-results/bin，无 `*.pem`/`*.key`/`.env*` 通配，敏感资产（如 extension-dev.pem）可被轻易提交。
- **Action**: (1) 在 `taskChromePlugin/.gitignore` 追加 `*.pem`、`*.key`、`*.p12`、`.env*` (2) 检查其余子仓 .gitignore 是否同样缺失（grep 全量子仓） (3) 已跟踪文件不受 gitignore 影响，需结合 OPT-059 一并处理
- **Why**: 防御性 gitignore 是防止密钥类资产入库的第一道闸门。
- **How to apply**: `taskChromePlugin/.gitignore`；`scripts/` 巡检

## [OPT-20260824-067] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 排队调度入口移至工作面板头部更名自动调度安排，E2E 7/7 + 单测 5/5 全绿
- **Created**: 2026-08-24
- **Context**: 页面元素调整目标（工作面板页）：原侧栏第 3 项「工作空间的排队调度」入口移至头部搜索框与工作空间选择器之间，命名为「自动调度安排」，路由 `/tenant/:tenant/queue-schedule/` 不变。
- **Action**: (1) WorkPanelHeader 在 WorkPanelTaskSearch 后插入 router-link（监听 workspace-switched 取 workspaceId 构造链接，无 workspace 时不渲染）；(2) Sidebar 删除硬编码 queue-schedule 导航项；(3) tenantConsoleNav 移除 nav.queue_schedule 条目；(4) WorkspaceQueueSchedule 页标题与 TaskDetailQueuedScheduleToggle 入口文案统一为「自动调度安排」；(5) 新增 WorkPanel.auto-schedule-link E2E + 更新 4 处既有 E2E 文案断言 + 1 处单测。
- **Why**: 页面元素调整需求：入口位置与命名按产品期望收敛。
- **How to apply**: 已实现并全量验证（E2E 7/7、单测 5/5、构建已切换 public/html）。

## [OPT-20260824-079] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: GitLab 公网流量闸门绕过修复：个人命名空间/非 tenant-{id} 前缀仓按 gitlab 用户名归租户计费（凭据绑定→成员→区域资源三重过滤），Rails 传认证用户名 + 缓存键含用户名；GitLab 19.x private 方法（#project/#user）修复。线上验证：enforce! 抛 ForbiddenError（未预购流量），CI/intranet 跳过保持，Loki 全链路日志闭环
- **Created**: 2026-08-24
- **Context**: 设置页显示「公网 Git 克隆/拉取已阻断」但容器 task_879620507262021632_cmt_879620511582154752（CSC csc_879620523418480640，TraceId f59a2d8521174023e7e20e21）在阿里云成功克隆上海 GitLab 个人命名空间仓 example-user/somanyad。根因：闸门按 project_path 正则 `^tenant-(\d+)` 归租户，个人命名空间匹配失败 → UNMAPPED → fail-open（07:07:28Z 闸门日志=克隆瞬间，prepaid=0 仍放行）。第二层根因：GitLab 19.2.4 的 GitAccess#project/#user 是 private 方法，initializer 的 `respond_to?` 拿不到 → 真实请求 project_path/username 恒空 → 闸门对一切请求 fail-open。
- **Action**: (1) taskBill gitlab_traffic_gate.go 接入 gitlab_username 字段 + evaluate 增加用户名归集分支；gitlab_traffic_gate_username.go 实现解析链（git_oauth_appusercredential provider=gitlab:{region} + gitlab:% 前缀回退 → task2app_user_id → tenant_company_member JOIN billing_tenant_gitlab_resource 恰一租户，失败保持 UNMAPPED fail-open）；(2) Rails initializer 传 gitlab_username_for(access)（private 方法用 respond_to?(sym,true)+send）、缓存键含用户名；(3) BDD 测试 7 新增 + 2 既有适配（含 CI/intranet 不解析断言），taskBill 全量 73.3s 通过，initializer 语法+skip 规则+private 方法断言通过；(4) 部署：taskBill 精准编译重启（10.2.150.68:9999），initializer scp 到 sh（/opt/daydaymoney/gitservice-tencent-sh-1/initializers/）并 docker restart 容器（bind mount 变更不触发 compose 重建，必须显式 restart）；(5) 线上验证：gate 端点带/不带用户名均 TRAFFIC_NOT_PURCHASED + tenant_id=877397588196749312；Rails 进程 enforce! 抛 Gitlab::GitAccess::ForbiddenError；CI 构造请求跳过；Loki gitlab_traffic_gate_username_resolved + allowed=false 日志闭环。
- **Why**: 租户预购流量闸门对个人命名空间仓失效导致流量未预购仍可公网克隆；19.x private 方法使闸门整体失效（连带组仓）。
- **How to apply**: taskBill/src/gitlab_traffic_gate.go（request 解析 + evaluate 归集）、gitlab_traffic_gate_username.go（解析链，测试换桩 resolveTenantIDFromGitlabUsername）、gitlab_traffic_gate_test.go（7 新增 BDD）、gitService/initializers/zzz_trae_gitlab_traffic_quota.rb（gitlab_username 透传 + private 方法修复）、gitService/scripts/test_zzz_traffic_quota_skip.rb（private 方法断言）。已部署生效；真实 git HTTP 客户端级验证与 disk enforce 同根因修复见 OPT-20260824-080/081/082。

## [OPT-20260824-083] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 首页页脚 FAQ 链接 + /faq/ 页面落地（Faq.vue + 路由 + public/faq/*.md 文档 + 4+1 测试全绿 + vite build 通过）
- **Created**: 2026-08-24
- **Context**: 云端开发首页（Home.vue）页脚联系区（#footer-contact）仅邮箱 author@example.com，无 FAQ 入口；product/faq 目录此前为空。
- **Action**: (1) 新增 Faq.vue 常见问题页（import.meta.glob('/public/faq/*.md', ?raw) 构建期打包 + MarkdownContent.vue 渲染，自动发现新 md，标题取一级标题/文件名兜底，码点排序）；(2) publicRoutes.js 注册 /faq/ 路由（Navbar 布局）；(3) Home.vue 联系区新增「常见问题」router-link（data-testid=home-footer-faq-link）；(4) 文档 account/usage/billing + 外部补充 知识产权处理方法.md；(5) 测试：Faq.test.js 4 例 + Home.footer-links.test.js 常见问题断言翻转。
- **Why**: 用户需从首页直达 FAQ；md 文档托管于 public/faq 可免改代码添加（仅需重建）。
- **How to apply**: taskFE 精准编译重启后访问 /faq/；后续新增 FAQ 直接在 app/src/public/faq/ 放 .md 即可自动收录。

## [OPT-20260824-085] completed

- **Status**: completed
- **Completed**: 2026-08-24
- **Summary**: 公网 SPA UserReferral chunk 含渠道分账/share-channel 且无 ORD-；本机 taskBill 8004 GET 对真实推荐人返回仅 channels（row_keys=channel_code/period_*/金额/shareable），JSON 无 order_number/order_id/ORD-/openid。旧进程曾返回 orders[]，已用 21:44 二进制替换。
- **Created**: 2026-08-24
- **Context**: 推荐人 `/profile/referral/` 原「订单分账」表直接渲染受推荐人 `order_number`。已改为按渠道聚合（渠道号、订单时段、订单/冻结/可分账金额）+ `POST share-channel`；需部署后在公网确认旧订单号单元格消失。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点击「精准编译重启」消费 `task-bill` `taskFE`；(2) 硬刷新 https://www.daydaymoney.com/profile/referral/?accessCode=DR2AKvP9J9 ，断言表格无 `ORD-`、列为渠道号/时段/金额，可分账渠道仍有「分账」按钮。
- **Why**: 隐私修复仅在新二进制 + 新 SPA 上线后对用户生效；旧 SPA 仍会读空 `orders`。
- **How to apply**: `.runall/precise_restart_services.txt` 已登记；核验选择器不再匹配 `td.font-mono` 内的 `ORD-` 订单号。

