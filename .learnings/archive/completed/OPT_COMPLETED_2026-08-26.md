# Completed OPT Archive — 2026-08-26

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 21 条。
> 归档执行时间：2026-08-28T02:10:47+08:00

## [OPT-20260825-015] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskAuth 9072f94 + taskFE 4b14d5c 已推送：ValidateImpersonationReason 拒绝逐字等于弹窗说明/placeholder 的 reason（ErrReasonIsPromptText→400），FE 提交前同校验；Go domain+handler 与 composable/modal 回归测全绿。
- **Created**: 2026-08-25
- **Context**: 收信箱模拟登录通知的 `reason` 字段出现过与 `ImpersonateReasonModal` 说明段完全相同的文案（「请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。」）。前端只校验 ≥8 字，这段说明正好达标，会被当成真实理由落库。
- **Action**: (1) 在 `ValidateImpersonationReason` 拒绝与弹窗说明/placeholder 完全一致的字符串；(2) 前端确认前做同样校验并展示错误；(3) 补 Go domain 测与 `useImpersonateUser`/`ImpersonateReasonModal` 测。
- **Why**: 审计和被模拟用户收信箱会看到无信息量的说明文案，看起来像系统自填。
- **How to apply**: `taskAuth/domain/impersonation.go`、`taskFE/app/src/composables/useImpersonateUser.js`、`taskFE/app/src/components/ImpersonateReasonModal.vue`；验收 `go test ./domain -run ValidateImpersonationReason` 与 vitest 相关文件 exit 0。

## [OPT-20260825-013] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskFE bd66360 已推送：SystemAdminReferralManagement savePolicy/saveSettleConfig 接入 createClickGuard+mergeIdempotencyHeaders，补 policy click-guard 接线回归测 2 例。
- **Created**: 2026-08-25
- **Context**: 本会话去掉比例保存按钮时，同页「保存策略」与「保存」入账天数仍只靠 `disabled`，没有 `createClickGuard` + `Idempotency-Key`。
- **Action**: 为 `savePolicy` / `saveSettleConfig` 接入 `createClickGuard().run` 与 `mergeIdempotencyHeaders`，并补组件测。
- **Why**: 元规则 52 要求写操作同步门闩；比例卡已只读，这两处仍是副作用按钮。
- **How to apply**: `taskFE/app/src/views/SystemAdminReferralManagement.vue`；测例可放同目录 `*.policy.test.js`。

## [OPT-20260825-019] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskFE 67a33b8 已推送：UserInbox/UserReferral/SystemAdminReferralManagement/ReferralChannelPanel 错误路径改走 messageFromFailedResponse，网关 502 HTML 展示「服务暂时不可用」+ data-traceId；补 3 处组件回归测。
- **Created**: 2026-08-25
- **Context**: 系统管理支付抽屉与推荐绩效抽屉已改用 `messageFromFailedResponse`。同模式仍在 `UserInbox.vue`、`ReferralChannelPanel.vue`、`UserReferral.vue`、`SystemAdminReferralManagement.vue` 等：HTML 502 时落到通用「加载/申请失败」。
- **Action**: (1) 将上述调用改为 `messageFromFailedResponse(response, fallback)` 且错误路径不要先 `json().catch({})` (2) 补 502 HTML 回归测（data-traceId + 重试文案）。
- **Why**: 同类网关 502 会在其它页面重复「无信息红字」。
- **How to apply**: `taskFE/app/src/views/UserInbox.vue`、`UserReferral.vue`、`SystemAdminReferralManagement.vue`、`taskFE/app/src/components/ReferralChannelPanel.vue`；共享 `taskFE/app/src/utils/httpError.js`。

## [OPT-20260825-016] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskBill 6964aba + taskFE b466ff4 已推送：listProfitSharingQueue 对空 wechat_profit_sharing_id 行写入 wechat_state=尚未提交微信（复用常量），首屏免点同步；后端 handler + FE tab 回归测全绿。
- **Created**: 2026-08-25
- **Context**: 用户列表推荐业绩「微信分账」Tab 的微信状态列默认是「—」，只有点「同步微信状态」后才显示「尚未提交微信」。冻结期内多数行从未 POST 到微信，管理员仍会去点同步。
- **Action**: (1) `handleSystemAdminListProfitSharing` 对 `wechat_profit_sharing_id` 为空的行写入 `wechat_state=尚未提交微信`（仍禁止回传 openid / 微信分账单号）；(2) 前端首屏用该字段渲染，无需先点同步；(3) 补 list handler 与 `ReferralWechatProfitSharingTab` 回归测。
- **Why**: 少一次微信 QueryOrder 往返，也避免管理员把空状态理解成故障。
- **How to apply**: `taskBill/src/handlers_admin_list_profit_sharing.go`、`taskBill/src/profit_sharing_admin.go`、`taskFE/app/src/components/ReferralWechatProfitSharingTab.vue`；与 `wechatProfitSharingStateNotSubmitted` 共用文案常量。

## [OPT-20260825-037] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskFE 8eb5904 已推送：test_nginx_static_resident.sh 增加 live HTTP 探测（默认本地 :4000，MP_VERIFY_PUBLIC_BASE 打开公网），断言 200/text-plain/正文一致 + 缺文件 404；本地与公网实测通过。
- **Created**: 2026-08-25
- **Context**: 已将 `taskFE/app/static/MP_verify_TmQKnSMuqxrR7atQ.txt` 经 Docker nginx 挂到站点根路径，公网 `https://www.daydaymoney.com/MP_verify_TmQKnSMuqxrR7atQ.txt` 当前 200 且正文为校验码。现有门禁只静态检查 nginx.conf / compose 挂载，nginx 重构或漏挂卷时公网可能静默回成 SPA HTML。
- **Action**: (1) 在 `taskFE/app/scripts/test_nginx_static_resident.sh` 或独立探测脚本增加可选 live 检查：`curl -fsS` 本地 `:4000` 与 `publicBaseUrl`，断言 HTTP 200、`Content-Type` 含 `text/plain`、正文去空白后等于文件内容 (2) 缺文件时须 404 而非 200 HTML (3) 默认仅本地 :4000，公网探测用环境变量打开以免 CI 无网失败
- **Why**: 微信公众平台域名校验只认精确正文；SPA `try_files` 回 index.html 会显示 200 却校验失败，静态 conf 检查发现不了运行时挂载丢失。
- **How to apply**: `taskFE/nginx/nginx.conf` 的 `location ~ ^/MP_verify_`；`docker-compose.yml` 卷 `./app/static:/srv/taskfe-static`；`conf/frontend/vue/config.yaml` 的 `publicBaseUrl`。

## [OPT-20260825-033] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskBill 6d2249b 已推送：ensureProfitSharingReceiver 登记成功（含幂等分支）后 persistReferralEdgeOpenid 回填空值边 referrer_openid（不覆盖已有值）；补空边回填+已有值不覆盖回归测。
- **Created**: 2026-08-25
- **Context**: `upsertReferralEdge` 从不写入 `referrer_openid`，支付时刻快照常为空。本会话已在发起分账时现查 receiver/taskAuth；边表仍空会导致后续新订单继续空快照。
- **Action**: (1) 在 `ensureProfitSharingReceiver` 登记成功后 UPDATE 该推荐人全部 `billing_referral_edge.referrer_openid`（仅空值）(2) 补回归测：登记后边表非空、空 openid 不覆盖已有值
- **Why**: 少一次分账时的跨服务 identity 查询，并让 25 天扫描路径更早命中快照。
- **How to apply**: `taskBill/src/wechat_profit_sharing_receiver.go` `ensureProfitSharingReceiverWithLegalName` 成功分支；`referral_commission.go` `upsertReferralEdge`

## [OPT-20260825-009] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: 回退卡补 usagePercent 进度条 + vitest 断言 aria-valuenow；taskFE 64ffa40 已推送
- **Created**: 2026-08-25
- **Context**: 账单页已按区域展示磁盘/流量已用/配额。仅当 `gitlab_resources` 为空时仍走聚合回退卡，回退卡有已用数字但没有进度条，与按区卡不一致。
- **Action**: (1) 给 `gitlab-aggregate-disk-card` / `gitlab-aggregate-traffic-card` 复用 `usagePercent` 进度条；(2) 补一条 vitest：无 `gitlab_resources` 时进度条 `aria-valuenow` 与 used/quota 一致。
- **Why**: 少见回退路径体验不一致；用户在数据未按区返回时会少一根用量可视化。
- **How to apply**: `taskFE/app/src/views/BillingDashboard.vue`；`formatUsedGb.js` 的 `usagePercent`；`BillingDashboard.gitlabRegions.test.js`。

## [OPT-20260825-026] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: 租户行赠送资源深链 + GrantPoints query 预选租户 + 单测；taskFE c56eeb0 已推送
- **Created**: 2026-08-25
- **Context**: 本会话在超管用户页加了只读「租户」目录。公司名已链到 `/system-admin/tenants/{id}/` 详情；运营仍可能要从行跳到赠送资源页或该公司工作台，目前只能复制 ID 再手工搜索。
- **Action**: (1) 在 `SystemAdminTenantsPanel` 行增加真实 `<a href>` 指向赠送资源或租户工作台（禁止 click+router.push） (2) 补单测
- **Why**: 目录价值在定位后的下一跳；没有入口则运营仍要在多页间复制 ID。
- **How to apply**: `taskFE/app/src/components/SystemAdminTenantsPanel.vue`；深链可参考 GrantPoints `tenant-options` 选中逻辑。

## [OPT-20260825-028] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: 租户详情工作空间表补翻页控件 + 单测；taskFE 4f69e77 已推送
- **Created**: 2026-08-25
- **Context**: 系统管理租户详情页工作空间块只拉 `limit=50` 首页；订单块有「查看全部」链到订单记录页，工作空间没有同等出口。大租户会看不到第 51 个以后的空间。
- **Action**: (1) 在 `SystemAdminTenantDetail.vue` 工作空间块加下一页或 `total>limit` 提示 (2) 或提供真实 `<a href>` 跳到已有工作空间管理面（须带 tenant_id）(3) 补空态/翻页单测
- **Why**: 详情页承诺「工作空间情况」，截断且无出口会让超管误判租户只有 50 个空间。
- **How to apply**: `taskFE/app/src/views/SystemAdminTenantDetail.vue`；API 已支持 `limit`/`offset`：`GET /api/system-admin/tenant-workspaces/tenant_id/{id}/`

## [OPT-20260825-023] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskBill 6b308ea + taskFE 15050a8 已推送：商户单号 out_trade_no 表头列过滤（Go 双列 LIKE + Vue HeaderTextFilter）；已登记精准重启 taskBill taskFE
- **Created**: 2026-08-25
- **Context**: 系统管理用户抽屉与待分账面板已展示微信支付 `out_trade_no`（商户单号）与 `out_profit_sharing_no`（商户分账单号）。表头过滤仍只有订单号 / 租户 ID / 接收方，运营无法按微信商户后台可见的 WX 单号筛选。
- **Action**: (1) `listProfitSharingQueue` 增加 `OutTradeNo` 等值或 LIKE 条件（命中 `o.out_trade_no` 与 `payment_ref` 去前缀）(2) `SystemAdminProfitSharingPanel` 商户单号列表头接入 `HeaderTextFilter` (3) handler 解析 `out_trade_no` query 并补回归测。
- **Why**: 运营对照微信支付商户后台排查失败分账时，复制的是商户单号而非内部 ORD- 订单号。
- **How to apply**: `taskBill/src/profit_sharing_admin.go` 的 `profitSharingQueueFilter`；`taskFE/app/src/components/system-admin/SystemAdminProfitSharingPanel.vue` 过滤行第二列。

## [OPT-20260825-017] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskBill b446535 已推送：支付流水按订单 consent_id 一对一挂载签署 + 回归测
- **Created**: 2026-08-25
- **Context**: 超管「支付与签署」抽屉把最新支付条款复制到每一笔用户支付上，注册服务协议与历史条款只能进「其他协议签署」。文案已改清楚，但审计仍无法回答「这一笔支付当时签的是哪一版」。
- **Action**: (1) `listUserRecharges` / `enrichRechargesWithConsent` 优先读 `billing_resource_order.consent_id`（048 已有列）(2) 仅无 consent_id 的支付行才回退到最新支付条款 (3) 补 Go 回归：两笔支付挂不同 consent_id 时互不覆盖
- **Why**: 现在多笔支付显示同一份最新条款，历史签署只能挤在「其他」区块，合规审计对不上单笔。
- **How to apply**: `taskBill/src/admin_user_recharges.go` `enrichRechargesWithConsent`；`taskBill/src/consent_gate.go` `attachConsentToOrder`

## [OPT-20260825-030] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskAuth b22bb49 + taskEvents b67cf24 + conf eb80a7a 已推送：auth_oidc_sso_idempotency 过期清理 timer 落地。taskAuth 新增 POST /api/internal/taskauth/oidc-sso-idempotency/cleanup/（max_age_days 默认 7d，LIMIT 分批循环，内部密钥门禁）；taskEvents 新增 oidc_sso_idempotency_cleanup/1_cleanup timer worker（port 18069，默认 24h tick，透传 X-TaskAuth-Internal-Secret）；intent_registry/run.sh/runAll.yaml/events config 全链路登记。taskAuth 3 例 + taskEvents 4 例回归测全绿，整包 test 绿。runAll :9999 down，部署留待后续窗口。
- **Created**: 2026-08-25
- **Context**: ADR-0043 签发/轮换/关闭 SSO 把 Idempotency-Key 写入 `auth_oidc_sso_idempotency`（主键 company_id+operation+key）。表会随操作次数增长，当前无 TTL/归档。
- **Action**: (1) 评估热窗口（建议 24h–7d）(2) 用 taskEvents timer + taskAuth 一次性 DELETE 分批清理过期行（禁止业务进程 ticker）(3) 补回归测
- **Why**: 时间累积型小表现在量不大，但无限保留会变成无用主键膨胀；清理须走 timer worker（元规则 51）。
- **How to apply**: `dataMigrate/taskAuth/043_oidc_client_tenant_sso.sql` 表；清理 API 放 taskAuth internal；timer 在 `taskEvents/internal/handlers/`

## [OPT-20260819-038] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: 17 处未接线文件本窗全部清零（7 批 17 文件 + 15 回归测 20 例）：taskFE 十九续批 2a9a70c 推送；--strict 全仓 0 WARN。createClickGuard + Idempotency-Key 铺满 taskFE 全部存量写按钮。
- **Created**: 2026-08-19
- **Context**: 元规则 52 / ADR-0020 已落地，参考实现仅接到 `OrderCreate.vue`。支付、退款、取消订单、管理端保存策略等写按钮仍可能只靠异步 `disabled`。
- **Action**: (1) 扫描 taskFE `@click` + `method: 'POST'|PUT|PATCH|DELETE` (2) 对资金/资源路径改用 `createClickGuard` + `Idempotency-Key` (3) 纯 UI 补 `Anti-Replay-OK` (4) `--strict --files` 验收
- **Why**: 前端锁未铺开时，连点与超时重试仍会双发写请求。
- **How to apply**: `taskFE/app/src/utils/clickGuard.js`；门禁 `db/scripts/ci/check_frontend_button_anti_replay.py --strict --files`
- **2026-08-25 夜**: 首批 2 文件已转换并推送——taskFE `040e5ab`：SystemAdminLoginPaymentPolicy saveFeaturePolicy（资金路径）与 SystemAdminGitlabTenantPanel provisionTenant（资源路径）包 createClickGuard.run + Idempotency-Key；新增 2 个组件级回归测断言 POST 携带 Idempotency-Key 头；`--strict --files` 对两文件通过。剩余 62 处未接线的 @click+写方法（含支付/退款/取消订单等）留专门会话，保持 pending。
- **2026-08-25 次窗**: taskFE `afc063d` 已推送——WorkspaceSettingsCloudPlatform（云平台授权/凭据，资源路径）6 个写操作（添加/编辑授权 POST/PUT、凭据校验 POST、删除授权/删除 OAuth DELETE、激活切换 POST）全部包 createClickGuard.run + Idempotency-Key；新增 4 例组件级回归测（断言 DELETE/POST 带 Idempotency-Key 头）全绿，`--strict --files` 通过。`2dbffb8` 已推送——TaskDetailLlmBudgetPanel（LLM 预算，资金路径）保存覆盖 PATCH + 临时上调 POST 接入同款 + 2 例回归测。`5003f71` 已推送——SystemAdminSubTokenProviders（sub-token 供应商，资源/安全路径）保存 POST/PUT + 删除 DELETE 接入同款 + 2 例回归测。剩余 59 处未接线，保持 pending。
- **2026-08-25 07:05+ 夜**: 续转 9 文件 4 批（WARN 59→50）——`6cb9796`：UserProfileEmailBindingPanel（发验证码/绑定邮箱 POST）+ UserProfileWechatBindingPanel（解绑 DELETE）+ ResetPasswordRequest（重置链接/手机验证码 POST）；`5abcef5`：PhoneVerificationGate（发短信/验证手机 POST）+ WorkspaceSwitcher（切换工作空间 POST）；`8a8f730`：CodeLangOptionsSettingsModal / TaskKindOptionsSettingsModal（保存 PUT）；`0dad1c8`：CreateTaskFieldSettingsModal（保存同批 3 PUT 共用同一幂等键）+ PendingInvitations（撤销/重发邀请 POST）。全部 `createClickGuard.run` + `mergeIdempotencyHeaders`，新增 8 个组件级回归测（13 例）断言写请求携带 Idempotency-Key 头，vitest 相关 38+ 例全绿，`--strict --files` 对每批通过，pre-commit 全绿，已推送。剩余 50 处未接线（含支付/退款/取消订单等资金路径），保持 pending。
- **2026-08-25 08:00+ 夜**: 续转 6 文件 4 批（WARN 50→44）——taskFE `a211061`：WorkspaceSettingsLlmBudgetModal（LLM 预算默认保存 PATCH，资金路径，onEnterKey 尊重 guard）+ PrivacyReconsentGate（隐私同意 POST，合规写路径）+ WorkspaceContainerImageAtModeToggle（容器镜像 @ 模式切换 PATCH，资源路径）；taskFE `a222b44`：WorkspaceAssociation（添加/移除关联两个 POST，资源写路径）；taskFE `73e01c1`：MemberGitIdentitiesModal（创建 POST + 设默认 PATCH + 删除 DELETE）；taskFE `1e3b29c`：AccessTokenManagementPanel（创建 POST + 吊销 DELETE，安全写路径）。全部 `createClickGuard.run` + `mergeIdempotencyHeaders`，新增 6 个组件级回归测（10 例）断言写请求携带 Idempotency-Key 头，vitest 相关 10 文件 18 例全绿，`--strict --files` 对每批通过，pre-commit 全绿，已推送。剩余 44 处未接线，保持 pending。
- **2026-08-26 夜**: 续转 2 文件 1 批（WARN 44→42）——taskFE `418bedf`：SystemAdminDefaultColumnManagement（保存进度体系 POST/PUT、设为默认 POST、删除 DELETE）+ WorkspaceSettingsTaskPanel（保存任务存档档位 PATCH、保存工作空间 POST/PUT、删除工作空间 DELETE）接入 createClickGuard + Idempotency-Key；新增 2 个组件级回归测（4 例）断言写请求携带 Idempotency-Key 头，vitest 全绿，`--strict --files` 对两文件通过，pre-commit 全绿，已推送。剩余 42 处未接线，保持 pending。
- **2026-08-26 次批**: 续转 2 文件 1 批（WARN 42→40）——taskFE `e980d7e`：DeliverableSystemSettings（保存交付物体系 POST）+ SystemAdminRecommendedLLMProviders（新增/编辑供应商 POST/PUT、删除 DELETE、上下移排序 PUT）接入 createClickGuard + Idempotency-Key；新增 2 个组件级回归测（3 例）断言写请求携带 Idempotency-Key 头，vitest 全绿，`--strict --files` 对两文件通过，pre-commit 全绿，已推送。剩余 40 处未接线，保持 pending。
- **2026-08-26 三批**: 续转 2 文件 1 批（WARN 40→38）——taskFE `f165671`：UserGitIdentities（新建身份 POST、设默认身份 PATCH）+ SystemAdminUserKycDrawer（触发自动评估 POST、人工覆盖 POST、AML 登记 POST）接入 createClickGuard + Idempotency-Key；新增 2 个组件级回归测（4 例）断言写请求携带 Idempotency-Key 头，vitest 全绿，`--strict --files` 对两文件通过，pre-commit 全绿，已推送。剩余 38 处未接线，保持 pending。
- **2026-08-26 四批**: 续转 2 文件 1 批（WARN 37→35，gate 实测口径）——taskFE `b35de89`：SystemAdminEmailInvitationsSection（发送邀请/模态重发/列表行重发/批量重发 4 个 POST）+ TaskDetailApplyPatchBar（提交 patch POST）接入 createClickGuard + Idempotency-Key；新增 2 个组件级回归测（4 例）断言写请求携带 Idempotency-Key 头，vitest 701 文件 3640 例全绿（仅预存 node:test 文件 `SecureInteractiveRichTextFrame.origin.test.js` 在 vitest 下报 No test suite——node --test 直跑 1 例绿，非本批引入），`--strict --files` 对两文件通过，pre-commit 全绿，已推送。剩余 35 处未接线，保持 pending。
- **2026-08-26 五批**: 续转 2 文件 1 批（WARN 35→33，gate 实测口径）——taskFE `819d143`：SystemAdminPrivacyPolicy（创建 POST / 编辑 PUT / 删除 DELETE）+ UserProfilePersonalDataExportPanel（请求生成导出文件 POST）接入 createClickGuard + Idempotency-Key；新增 2 个组件级回归测（3 例）断言写请求携带 Idempotency-Key 头，既有 `UserProfilePersonalDataExportPanel.test.js` 8 例仍绿，`--strict --files` 对两文件通过，pre-commit 全绿，已推送。剩余 33 处未接线，保持 pending。
- **2026-08-26 六批**: 续转 2 文件 1 批（WARN 33→31，gate 实测口径）——taskFE `e6b7c8b`：TenantCompanySettings（保存公司名称 PATCH）+ UserCompanySettings（保存公司设置 PATCH）接入 createClickGuard + Idempotency-Key；新增 2 个组件级回归测（2 例）断言写请求携带 Idempotency-Key 头，既有 `TenantCompanySettings.stale404.test.js` 3 例仍绿，`--strict --files` 对两文件通过，pre-commit 全绿，已推送。剩余 31 处未接线，保持 pending。
- **2026-08-26 七批**: 续转 2 文件 1 批（WARN 31→29，gate 实测口径）——taskFE `6257446`：SystemAdminRegistrationInvitePanel（保存注册邀请策略 PUT）+ Activation（激活账号 POST，Options API + window.apiFetch 全局契约）接入 createClickGuard + Idempotency-Key；新增 2 个组件级回归测（2 例）断言写请求携带 Idempotency-Key 头，`--strict --files` 对两文件通过，pre-commit 全绿，已推送。剩余 29 处未接线，保持 pending。
- **2026-08-26 八批**: 续转 2 文件 1 批（WARN 29→27，gate 实测口径）——taskFE `f3a9df9`：SystemAdminDeliverableSystem（创建 POST / 编辑 PUT / 删除 DELETE）+ MemberList（预算权限 PATCH / 更新角色 PATCH / 切换状态 PATCH / 移除 DELETE）接入 createClickGuard + Idempotency-Key；新增 2 个组件级回归测（4 例）断言写请求携带 Idempotency-Key 头，`--strict --files` 对两文件通过，pre-commit 全绿，已推送。剩余 27 处未接线，保持 pending。
- **2026-08-26 九批~十三批（本窗 02:00-02:15）**: 续转 10 文件 5 批（WARN 27→17，gate 实测口径）——taskFE `9acd85d`（TaskDetailQueuedScheduleToggle 加入/离开队列 PATCH + WorkspaceSettingsFeatureParams 保存环境变量 POST）、`2e218f5`（SystemAdminBrowserExtension 保存插件白名单 PUT + GitIdentityCreateModal 创建 Git 身份 POST）、`4b1c88a`（ColumnSystemSettings 切换进度体系 POST + ProgressSystemSettings 保存进度体系 POST）、`75beb96`（PersonalFeatureParamsConfigs 保存配置 POST/PUT + 删除 DELETE + CreateWorkspace 组件 创建工作空间 POST）、`489a949`（CreateWorkspace 视图 创建工作空间 POST + PeopleJoin 加入团队 POST）接入 createClickGuard + Idempotency-Key，按钮 disabled 补 guard.isBusy()；新增 10 个组件级回归测（13 例）断言写请求携带 Idempotency-Key 头，`--strict --files` 对每批通过，pre-commit 全绿，已推送。剩余 17 处未接线，保持 pending。

## [OPT-20260825-032] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskAuth 83cd629 推送：补 TestOidcJWKSTenantPathSharesGlobalKeys 回归测锁定租户路径 JWKS 与全局同 kid（防误拆钥致验签失败）。结论：当前保持共享密钥，按租户轮换拆钥/iss 拆分保持 deferred（出现需求再做）。
- **Created**: 2026-08-25
- **Context**: ADR-0044 已把租户 GitLab SSO 的 `jwks_uri` 改为 `/api/oidc/{tid}/jwks`，但 handler 仍返回与 `/api/oidc/jwks` 相同的签名密钥集。
- **Action**: (1) 若出现按租户轮换 IdP 签名钥的需求，再为 tenant path 提供独立 JWKS/kid (2) 同步 ID token `iss` 是否仍保持全局 (3) 补回归测禁止串钥
- **Why**: URL 已可分片，密钥材料未分片；过早拆钥会让已签发 token 验签失败。
- **How to apply**: `taskAuth/src/oidc_handlers_userinfo.go` `handleOidcJWKS`；`domain.TenantOidcProtocolBase`

---

## [OPT-20260825-035] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: taskAuth 7fd86f2（auth_login_history 加 outcome 列 + 6 失败分支留痕，fail-open/截断 UA/不存 identifier；列表默认成功、include_failures=1 含失败；PIPL 导出保守只含成功待产品确认）+ taskFE bc92def（LoginHistoryPanel 仅成功/含失败切换 + 结果列徽章）+ dataMigrate 23c558f（045 迁移已应用生产 task_auth）。回归：taskAuth 全量 150s 绿 + FE LoginHistoryPanel 5 例及关联视图 4 例绿，pre-commit 全绿，三仓已推送。
- **Created**: 2026-08-25
- **Context**: 本会话交付 `auth_login_history` 只在成功认证写入。失败登录（错密、未激活、管理员入口误入客户入口）仍只打 slog，用户无法在侧边栏看到「有人试过我的账号」。
- **Action**: (1) 评估是否单独表 `auth_login_attempt`（避免失败行污染成功历史分页）或给现表加 `outcome` 列 (2) 失败写入同样 fail-open、截断 UA、不存 identifier (3) 账号中心默认只展示成功，可选筛选失败 (4) 补回归：错密不出现在默认列表
- **Why**: 成功历史已可排查本人 IP；暴力尝试与撞库仍只能靠日志，对个人用户不可见。
- **How to apply**: `taskAuth/src/auth_login.go` 失败分支；`login_history_handlers.go` 增加 outcome 过滤；`LoginHistoryPanel.vue` 筛选；PIPL 导出是否包含失败行需产品确认。

## [OPT-20260825-018] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: APISIX 502/503/504 改 JSON 错误体：taskGateway 395c737（apisix/config.yaml http_server_configuration_snippet error_page→@gateway_error JSON {error,trace_id} + 保留 X-Trace-Id 响应头；新增 test_error_page_json_502.py 4 例）+ taskFE f74b2ec（safeResponseJson 契约测 4 例：HTML 502 不作 API 成功体）。live 已应用：apisix init + nginx -t 通过 + reload，实测 /api/nonexistent-* 返回 Content-Type: application/json 且 trace_id 正确，health/SPA/业务路由不受影响。已登记精准重启 task-gateway。
- **Created**: 2026-08-25
- **Context**: 超管「支付与签署」抽屉 trace `2fb383c5-baf3-41d3-b869-2de4761b044f`：taskBill 重启窗口 APISIX `connect() failed (111)` → `content-type: text/html` 502。前端已改为识别 502/503/504 并展示「服务暂时不可用」+ 重试，但其它仍 `response.json().catch(() => ({}))` 的调用方会继续落到无信息兜底文案。
- **Action**: (1) 为 API 路由（`/api/*`）配置 APISIX 错误页或 serverless 将 502/504 写成 `{"error":"服务暂时不可用，请稍后重试","trace_id":"..."}` 且 `Content-Type: application/json` (2) 保留 `X-Trace-Id` (3) 补网关/前端契约测：HTML 502 不再作为 API 成功解析体。
- **Why**: 网关 HTML 错误页会让 `detail`/`error` 提取失败；JSON 502 让所有客户端自动吃到可读文案，不依赖每个抽屉手改。
- **How to apply**: `taskGateway/routes/routes.yaml` 全局 error-page / response-rewrite；生成物 `taskGateway/apisix/apisix.yaml`；对照 `taskFE/app/src/utils/httpError.js` `gatewayUnavailableMessage`。

## [OPT-20260826-004] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: upsertWechatIdentity 在非 mp 别名写入且 unionId 非空时调用 claimMpSubscribePending；follow-status 增加 has_unionid；测例 TestWeChatLoginUpsertClaimsMpPending / TestWeChatMPFollowStatusNoUnionID 通过
- **Created**: 2026-08-26
- **Context**: 关注在先、扫码登录在后时，目前要用户再点「我已关注」才消费 pending。`upsertWechatIdentity` 已超行数阈值，本会话未在登录路径挂钩。
- **Action**: (1) 在 `auth_wechat_mp.go` 保持 claim 函数 (2) 从 `findOrCreateWeChatUser` / `bindWeChatIdentity` 成功路径各调一次（或抽薄 wrapper 避免撑破 `auth_wechat.go`）(3) 回归：pending + web 登录后 mp 别名自动出现
- **Why**: 少一次手动确认，减少「已关注但仍看不到表单」客服单。
- **How to apply**: `claimMpSubscribePending`；测例可仿 `TestWeChatMPFollowStatusClaimsPending`。

## [OPT-20260826-005] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: task-auth 已精准编译重启；follow-status 对 user 877397583960502272 返回 bound=true ticket_status=bound；票 880376598551883776 已 bound。
- **Created**: 2026-08-26
- **Context**: 推荐页已改为动态 scene 码（POST follow-qr + SCAN 匹配 temp_id）。SQL `dataMigrate/taskAuth/047_wechat_mp_follow_ticket.sql` 与 task-auth/taskFE/task-gateway 已登记精准重启，但 runAll :9999 可能 down，线上尚未建表/换二进制。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点「初始化全部数据库」或 `bash db/scripts/apply_datamigrate.sh task_auth dataMigrate/taskAuth` (2) 点「精准编译重启」覆盖已登记的 task-auth / taskFE / task-gateway (3) 硬刷新 `/profile/referral/` 确认二维码 URL 为 `showqrcode?ticket=` 且冲突文案可复现
- **Why**: 未建表时 POST follow-qr 会 503；未重启时公网仍是静态 jjf 码。
- **How to apply**: 登记文件 `.runall/precise_restart_services.txt`；表 `auth_wechat_mp_follow_ticket`。

## [OPT-20260826-010] cancelled

- **Status**: cancelled
- **Completed**: 2026-08-26
- **Summary**: 产品已确认公网 HTTP GitLab 可签发 SSO，Path A 保存时无需再提示无法签发。
- **Created**: 2026-08-26
- **Context**: 租户 `877397588196749312` 把 Path A `base_url` 存成 `http://115.29.110.74`（该实例仅 HTTP、无 TLS）。OAuth 连接允许 HTTP，但签发平台 OIDC SSO 必须 HTTPS。本次已在 SSO 区块预检拦截；保存连接时仍无提示，用户会先保存再在签发处才看到原因。
- **Action**: (1) `WorkspaceSettingsGitlabConnection` 保存成功后若 base_url 为公网 HTTP，展示与 SSO 区块同一套中文 HTTPS 指引（复用 `gitlabOidcSsoHttps.js`）(2) 不拦截 Path A 保存（OAuth 仍可用 HTTP）(3) 单测覆盖 HTTP/HTTPS/loopback。
- **Why**: 把失败前移到用户正在编辑 URL 的表单，减少「保存成功 → 签发失败」两步认知负担。
- **How to apply**: `taskFE/app/src/views/WorkspaceSettingsGitlabConnection.vue`；`taskFE/app/src/utils/gitlabOidcSsoHttps.js`。

## [OPT-20260826-016] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: executeProfitSharing 出站金额改为 min(台账佣金, floor((订单净额-已退款)×5%), 微信剩余待分)；55分台账3分→调微信2分；全额退款409且不调微信。测例 wechat_profit_sharing_amount_test.go。
- **Created**: 2026-08-26
- **Context**: 超管对 profit_sharing_id=3 发起分账（trace `f9ca173d-aabc-48ff-a647-4004f5c39156`），微信 APIv3 返回 HTTP 400 `INVALID_REQUEST`「分账金额超出最大分账比例，最大可分账金额需等比例扣除退款与补差回退等逆向交易金额」。fail_reason 已完整落库；本会话只修了单元格截断。金额仍按原订单分账额出站。
- **Action**: (1) 出站 CreateOrder 前按微信规则用订单已退款/补差回退金额等比例下调 `amount` (2) 单测覆盖「有退款则金额小于原佣金」与「无逆向交易保持原额」 (3) 对 id=3 用下调后金额重试。
- **Why**: 不改金额则运营看清错误后仍无法分账成功。
- **How to apply**: `taskBill` CreateOrder 金额计算；对照微信「最大可分账金额」文档；`sdk/wechatpay-go` profitsharing 下单。

## [OPT-20260826-017] completed

- **Status**: completed
- **Completed**: 2026-08-26
- **Summary**: 微信 INVALID_REQUEST/RULE_LIMIT/NOT_ENOUGH 映射 HTTP 409 + 微信 message；admin_profit_sharing_share_failed 补 trace_id。handler 测锁定 409。
- **Created**: 2026-08-26
- **Context**: 同上 trace，task-bill `POST /api/system-admin/profit-sharing/3/share/` 对微信 INVALID_REQUEST 返回 **502**，`admin_profit_sharing_share_failed` warn 也未带 `trace_id` 字段（仅随后 http_request 行有）。前端把 502 当上游故障，运营与 Loki 关联成本高。
- **Action**: (1) 微信业务拒单（4xx + code/message）映射为 400/422 + 稳定 error code (2) warn 日志带上 ctx `trace_id` (3) handler 单测锁定状态码。
- **Why**: 502 会与网关/上游宕机混淆；warn 缺 trace_id 使 `{job=~".+"} | json | trace_id=` 漏掉业务行。
- **How to apply**: `taskBill` share handler / wechat profit sharing create；`handlers_admin_*share*_test.go`。

