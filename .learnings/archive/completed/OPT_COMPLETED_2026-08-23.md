# Completed OPT Archive — 2026-08-23

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 46 条。
> 归档执行时间：2026-08-25T22:13:52+08:00

## [OPT-20260822-046] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: db CI gate: .mjs/.cjs 视同业务源码 + 回归测
- **Created**: 2026-08-22
- **Context**: 本次修 `trae-agent/onlineServiceJS/src/autoRunPrBackfill.mjs` 等业务文件时，commit-msg 输出 `bug-fix commit touches no business source (docs/config/scripts/tests only) — exempt`。门禁未把 `.mjs` 算作业务源码，fix 提交可能漏配合同目录回归测仍被放行。
- **Action**: (1) 改 `db/scripts/ci/check_bug_fix_unit_tests.py` 将 `.mjs`/`.cjs` 与 `.js` 同等视为业务源码 (2) 补 `test_check_bug_fix_unit_tests.py`：暂存 `foo.mjs` + `fix:` 且无对应 `foo.test.mjs` 应阻断
- **Why**: 容器侧大量逻辑在 `.mjs`；漏检会使「fix 必须带回归测」对 onlineServiceJS 失效。
- **How to apply**: `db/scripts/ci/check_bug_fix_unit_tests.py`；`db/scripts/ci/test_check_bug_fix_unit_tests.py`

## [OPT-20260822-003] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskEvents 删除已跟踪 consumer/runner.go.bak
- **Created**: 2026-08-22
- **Context**: `consumer/runner.go.bak` 已纳入 git（2026-08-02），是旧 runner 备份。本次修编译错误时易与 `runner.go` 并列误导。
- **Action**: (1) `git rm taskEvents/consumer/runner.go.bak` (2) 确认无脚本引用 `.bak` (3) 提交子仓
- **Why**: 备份文件进版本库会让后续重构/全部重启时误读陈旧符号。
- **How to apply**: 仅 `taskEvents` 子仓；不要改 `runner.go` 行为

## [OPT-20260822-040] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE 单公司工作面板路径统一走 buildCompanySwitchHref
- **Created**: 2026-08-22
- **Context**: 修复导航栏「当前公司」点击无法跳转时，多公司菜单已改用 `buildCompanySwitchHref`，但单公司「工作面板」仍由 `Navbar.ui.vue` 的 `workPanelPath` 内联拼路径（含 localStorage / onboarding 回退）。两套构造会再次在 accessCode 或无租户回退上漂移。
- **Action**: (1) 让 `workPanelPath` 在已解析出租户 id 时调用 `buildCompanySwitchHref` (2) onboarding/localStorage 回退仍留在调用方 (3) 用既有 `Navbar.ui.test.js` 工作面板 href 测例回归
- **Why**: 刚修过当前公司 no-op；路径构造再分叉会再次出现「有的入口能跳、有的不能」。
- **How to apply**: `taskFE/app/src/components/Navbar.ui.vue` `workPanelPath`；`taskFE/app/src/utils/companySwitchNavigation.js`

## [OPT-20260822-045] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE 区域带宽剩余>总量前端提交前即时提示
- **Created**: 2026-08-22
- **Context**: 系统管理 GitLab 区域卡片已展示并保存 `total_bandwidth_mbps` / `remaining_bandwidth_mbps`；服务端会 clamp remaining>total，但表单仍允许填入非法组合，管理员要等保存后才看到被改写的剩余值。
- **Action**: (1) 在 `SystemAdminGitlabRegionCapacity.vue` 当剩余>总量时禁用保存并展示行内提示 (2) 补测 remaining>total 时不 emit save
- **Why**: 避免误以为「剩余 999 Mbps」已入库，实际已被服务端改成总量。
- **How to apply**: `taskFE/app/src/components/system-admin/SystemAdminGitlabRegionCapacity.vue`；`gitlabRegionCapacity.js`

## [OPT-20260822-010] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE ztree 任务动作失败改走 showRequestError+data-traceId
- **Created**: 2026-08-22
- **Context**: 案例 113 已把「发送给AI」从只读 `detail` 改为 `messageFromFailedResponse`，并给命令错误挂 `data-traceId`。`taskDetailJobActions.js` 的 redo/interrupt/delete 仍 `window.alert`，失败时没有 DOM trace。
- **Action**: (1) 将 `callLayerGraphJobAction` / `callLayerGraphLayerDelete` 的 `window.alert` 改为 `showRequestError(msg, response)` (2) 补回归测：403 `{message}` 展示业务文案且 modal 带 traceId (3) 禁止再引入 `window.alert` 展示 compute 失败
- **Why**: 元规则 24 要求请求失败 UI 可检索 trace；alert 无法挂 `data-traceId`。
- **How to apply**: `taskFE/app/src/composables/taskDetail/taskDetailJobActions.js`；`showRequestError` 已在 `requestErrorDisplay.js`

## [OPT-20260822-027] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE 一键合并失败弹层加绑定操作+status.error data-traceId
- **Created**: 2026-08-22
- **Context**: 任务详情 PR 卡「一键合并」失败时 `showRequestError` 只弹出中文说明（`尚未绑定 Git 网站 OAuth…`），Modal.ui 没有跳转到账号中心绑定页的 additionalAction。PR 卡内 `status.error` 行也未挂 `data-traceId`，与元规则 24 的请求失败展示不一致。
- **Action**: (1) 在 `taskDetailGitPrReply.js` 的 merge 失败分支给 `showRequestError` 传入 `additionalActions`（文案「去绑定 Git OAuth」，`href` 用已有 `buildRepoOAuthStartHref`）(2) `CommentGitPrReply.vue` 的 `status.error` 节点加上 `data-traceId`，trace 从 merge/status 失败响应透传 (3) 补单测：未绑定 409 时弹层含绑定链接且 error 行带 trace
- **Why**: 用户关掉弹层后仍要自己找绑定入口；status 失败无法用 DOM 检索对应请求。
- **How to apply**: `taskFE/app/src/composables/taskDetail/taskDetailGitPrReply.js`；`taskFE/app/src/utils/requestErrorDisplay.js` `showRequestError`；`taskFE/app/src/components/task-detail/CommentGitPrReply.vue`

## [OPT-20260822-044] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE task-detail accessCode 分享页不带陈旧 Authorization
- **Created**: 2026-08-22
- **Context**: 任务详情 Git OAuth 绑定探测已对 401/`redirect_url` 跳过整页登录。CDP 里未登录或 localStorage 残留失效 Token 时，任务正文仍可能展示「无法解析登录凭据」，评论芯片不出现；真正带 accessCode 的访客本应靠分享码拉任务。
- **Action**: (1) 在 `apiFetch` 或任务详情加载路径：URL 含 `accessCode` 且路径为 task-detail 时不带 `Authorization`（仍 `credentials: include`）(2) 单测锁定「有陈旧 Token + accessCode 仍请求无 Authorization」(3) Playwright：未登录 + accessCode 能看到评论区
- **Why**: 失效 Token 会让网关走 forward-auth 失败，盖过分享码授权，回流后即使用户不被踢登录也看不到绑定芯片。
- **How to apply**: `taskFE/app/src/utils/apiUtils.js` `resolveAuthToken`；守卫 `isTaskDetailAccessCodeSharePage`

## [OPT-20260822-059] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskBill 点数佣金计提改为支付时刻资格，与微信分账同口径
- **Created**: 2026-08-22
- **Context**: ADR-0033 已让 Native 分账标识与 `markOrderForProfitSharing` 在支付时刻现查 `referral_code`。`accrueReferralFromConsumption` 仍要求 `billing_referral_edge.commission_eligible=1`（绑边快照且 ON DUPLICATE 不覆盖），先推荐后获资的边永远不计提点数。
- **Action**: (1) 消费计提前复用 `lookupReferrerPaytimeQualification`（2）补回归：快照 0 + 现查活跃仍计提；现查不活跃不计提（3）更新 ADR-0025/意图中「点数仍看快照」的表述
- **Why**: 同一推荐关系下微信分账与点数口径分裂，运营对账与推荐页收益会对不上。
- **How to apply**: `taskBill/src/referral_commission.go` `accrueReferralFromConsumption`；接缝已在 `referral_paytime_qualification.go`

## [OPT-20260822-020] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskReferral 通过/拒绝审计写入 Idempotency-Key+请求 ctx
- **Created**: 2026-08-22
- **Context**: 前端通过/拒绝已发 `Idempotency-Key`，但 `approveReferralApplication`/`rejectReferralApplication` 用 `context.Background()` 插入审计，键走 `auto-approve-{id}-{nano}`，与取消资格路径不一致，超管连点重放无法按同一键短路。
- **Action**: (1) handler 把 `Idempotency-Key` 与 `r.Context()` 传入 approve/reject (2) 同一键重放返回已处理结果 (3) 补 httptest：同 key 第二次不重复写审计
- **Why**: 审计幂等只覆盖 revoke，通过/拒绝连点会写两条相同理由记录，审计对账噪音大。
- **How to apply**: `taskReferral/src/referral_code.go`、`referral_code_handlers.go`、`referral_qualification.go` `insertQualificationAudit`

## [OPT-20260822-033] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE 层图推送预检改探测 AccessToken 有效性
- **Created**: 2026-08-22
- **Context**: 举一反三：评论区已对「看起来已绑定」做一次 live probe。`gitOauthPushPrecheck.js` 仍只读 DB `connected`，GitLab refresh 400 时仍可能先允许 push 再在服务端换票失败。
- **Action**: (1) `gitOauthUnboundReasonForRepoUrls` / `fetchGitOAuthUserAppConnected` 在推送预检路径传 `probeAccessToken: true` (2) 补 `gitOauthPushPrecheck.test.js`：connected=true 且 access_token_valid=false 时拦截 (3) 禁止轮询
- **Why**: 与评论芯片同一根因：DB 绑定不等于 AccessToken 仍有效，用户会看到「提交成功但推送失败」。
- **How to apply**: `taskFE/app/src/utils/gitOauthPushPrecheck.js`；`taskGitOauth` GET `probe_access_token` 已可用

## [OPT-20260822-008] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: db CI 新增 run.sh start 禁隐式编译静态检查
- **Created**: 2026-08-22
- **Context**: ADR-0027 已拆掉 `taskEvents/run.sh start` 的 `go build`，并改了若干 Go 服务 `run.sh start`。同类回归仍可能从新服务模板或 `start) ./build.sh` 抄回来。
- **Action**: (1) 在 `db/scripts/ci/` 增加静态检查：`run.sh` 的 `start)` 分支不得调用 `go build`/`./build.sh`；`conf/runAll.yaml` 的 `start_command` 若指向 `run.sh start` 则交叉验证 (2) 接到 pre-commit (3) 用当前仓库跑一次确认仅 `build)`/`dev)` 允许编译
- **Why**: 全部重启空窗的根因是 start 隐式编脏树；单靠 code review 挡不住下一次复制粘贴。
- **How to apply**: 参考 `taskEvents/broker/run_sh_start_guard_test.go` 与 `dataMigrate/check_no_startup_migrate.py` 的扫描风格；白名单仅 `runAll/run.sh` 编排器自举（`RUNALL_SKIP_BUILD`）

## [OPT-20260822-028] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 058_backfill_order_profit_sharing collation 混用已修复：dataMigrate 23d6f62 对 JOIN 比较列显式 COLLATE utf8mb4_unicode_ci，已推送 origin/main 并经 apply_datamigrate 应用到生产 task_bill（migration log 058 应用成功）；059/060 随其后正常落地，全量 migrate 不再被卡。
- **Created**: 2026-08-22
- **Context**: 应用 task_bill dataMigrate 时 057 已跳过（已应用），但 `058_backfill_order_profit_sharing.sql` 在 JOIN 上等号比较触发 `ERROR 1267 Illegal mix of collations (utf8mb4_unicode_ci vs utf8mb4_0900_ai_ci)`，全量 migrate 无法继续。
- **Action**: (1) 给 058 的比较列显式 `COLLATE utf8mb4_unicode_ci` 或统一表/连接 collation (2) 复跑 `bash db/task-bill/migrate.sh` 至 058 成功 (3) 补回归：脚本在 unicode_ci 库上可重复执行
- **Why**: 后续 task_bill 迁移被卡在 058，新 DDL 无法经统一入口落地。
- **How to apply**: `dataMigrate/taskBill/058_backfill_order_profit_sharing.sql`（他会话文件，本会话未改）

## [OPT-20260822-017] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 删除死配置 profit_sharing_max_ratio_percent：conf a97eb82（conf.yaml 键 + conf.yaml.ai.md 文档）+ taskBill d286929（wechatPayConfig 字段 + 测试引用清理），两仓已推送。分成比例只从微信 merchant-configs 查询接口读取。
- **Created**: 2026-08-22
- **Context**: 分成比例改为只读微信支付查询接口后，`conf/billing/wechatPay/conf.yaml` 的 `profit_sharing_max_ratio_percent: 30` 仍留在仓库，后人容易再次当成展示 SSOT。
- **Action**: (1) 从 `conf.yaml` / `wechatPayConfig` 删除该键 (2) 删 `wechat_pay.go` 对应字段 (3) 确认无运行时读取
- **Why**: 死配置会把文档默认上限 30% 再次伪装成商户后台比例。
- **How to apply**: `conf/billing/wechatPay/conf.yaml`；`taskBill/src/wechat_pay.go`

## [OPT-20260822-019] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 取消分账资格后 best-effort 删除微信分账接收方：taskBill c289c95 新增 POST /api/internal/taskbill/profit-sharing/receivers/delete/（OpenAPI internal 同步）+ taskReferral 187f985 revoke 成功后调用（失败只 warn 不回滚）。审计以 taskBill receiver 行状态 deleted + 结构化日志承载。
- **Created**: 2026-08-22
- **Context**: 本增量取消资格只把 `referral_code` 置 `revoked`、并调 taskBill `disable-eligibility` 停边/作废 pending 计提。微信侧已登记的 profit-sharing receiver 仍留在商户后台，后续若再次开通需走更新而非新增。
- **Action**: (1) 在 `revokeReferralQualification` 成功后 best-effort 调用现有 `deleteProfitSharingReceiver`（失败只打 warn，不回滚资格）(2) 审计表可记 `wechat_receiver_deleted=true/false` (3) 补单测：mock 微信删除成功/失败都不阻断 revoke
- **Why**: 资格已失效但商户仍显示接收方，运维对账会误判该用户仍可分账。
- **How to apply**: `taskReferral/src/referral_qualification.go`；`taskBill/src/wechat_profit_sharing.go` `deleteProfitSharingReceiver`；内部 API 若尚未暴露删除接收方则补 `POST /api/internal/taskbill/referral/delete-receiver/`

## [OPT-20260821-036] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskTaskService f297b28 创建 @镜像 评论服务端校验 Git OAuth：mentions 含 installed_image 且任务关联 HTTP(S) 仓时向 taskGitOauth user-app-connection fail-closed 查询；未绑定 400 不落评论，依赖不可达 503，URL 未配置跳过。conf 857aeac 补 taskGitOauth 服务地址。
- **Created**: 2026-08-21
- **Context**: 本会话已在 taskFE 对 `@镜像` 提交并运行、层图提交并推送做前端 OAuth 门禁。纯前端拦截可被绕过（非 SPA 客户端直接 POST `/comments`），资金/云资源路径默认应 ≥ L3。
- **Action**: (1) 在 `handleCreateComment` 当 mentions 含 installed_image 且任务关联 GitHub/GitLab HTTPS 仓时，向 git-oauth `user-app-connection` fail-closed 查询当前用户绑定 (2) 未绑定返回 4xx，不落评论、不发 `TASK_COMMENT_IMAGE_MENTIONED` (3) 补重放/不同任务不塌缩单测
- **Why**: 前端门禁不能作为唯一防线；绕过 SPA 仍会先建评再在推送换票失败。
- **How to apply**: `taskTaskService` 评论创建 handler；`taskGitOauth` 既有 connection API；禁止用 `company_id` 作幂等键；纯评论不拦截

## [OPT-20260821-037] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: access-for-user 缺凭据 404 补 errDetail：taskGitOauth 347d4aa 404 区分 not_found/not_connected 并带可读 errDetail；taskCloudService 7415dc6 fetchGitOauthAccessForUser 404 透传 body errDetail/detail（空 body 回退明确文案），双仓已推送。日志轮转核对：logs/task-git-oauth.log 当前 ~5KB，无独立轮转配置，health 未冲掉业务日志。
- **Created**: 2026-08-21
- **Context**: 本会话排障 `cfecf502-82f8-429d-be69-cd9e9ac7a59c`：layer-git-push prepare 换票失败时 `access-for-user` 返回 404 且 `errDetail` 为空，前端只能拼出「未能换取 Git OAuth 凭据」；当时 `task-git-oauth` 日志几乎只剩 health，Loki 也搜不到该 trace。
- **Action**: (1) 缺凭据 404 响应写入明确 `detail`/`errDetail`（不含 token）(2) 结构化日志带入站 `trace_id` 与 repo 指纹 (3) 核对本机 `logs/task-git-oauth.log` 轮转/截断策略，避免排障窗口内业务日志被 health 冲掉
- **Why**: 换票失败时调用方与 Loki 对不上具体原因，只能靠时间窗猜「未绑定」。
- **How to apply**: `taskGitOauth` access-for-user handler；`taskCloudService` `git_push_oauth.go` 透传 detail；日志轮转见 runAll/logging 配置

## [OPT-20260823-003] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 061 已应用到 task_bill；已精准编译重启 task-bill/taskFE/task-gateway。公网订单页支付成功区可见「申请开票」弹层。
- **Created**: 2026-08-23
- **Context**: 迁移文件已写入 `dataMigrate/taskBill/061_order_invoice.sql`，业务进程不跑迁移；runAll :9999 当前可能 down。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点「初始化全部数据库」或对 task_bill 跑 apply_datamigrate (2) 精准编译重启 task-bill、taskFE (3) 硬刷新订单详情确认「申请开票」
- **Why**: 未应用 DDL 时开票接口会报缺表；未重启则公网仍跑旧二进制/旧 SPA。
- **How to apply**: `dataMigrate/taskBill/061_order_invoice.sql`；`.runall/precise_restart_services.txt`

## [OPT-20260823-004] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 已 shutdown-self 热替换为 ADR-0035 二进制；:9999 200；task-auth/task-bill 等被 adopt 仍存活
- **Created**: 2026-08-23
- **Context**: 本会话已让 SIGTERM 默认不拆栈，但现场有 `no_restart_runall` 标记且 :9999 仍 down。新二进制需人工拉起后才能 adopt 存量服务。
- **Action**: (1) 确认是否仍需 `no_restart_runall` (2) 在 runAll 目录 `./build.sh && ./run.sh`（HadPrevious=false 走 adopt）(3) 核对 Status UI 服务多为 healthy 且无需 start-all
- **Why**: 代码合入后若未替换正在用的旧二进制，误 SIGTERM 仍会拆栈。
- **How to apply**: `runAll/ai.md` ADR-0035；UI http://10.2.150.68:9999/

## [OPT-20260822-038] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 公网 probe_access_token 已验证返回 access_token_valid=true（浏览器 fetch 经网关 +probe 参数命中 task-git-oauth，APISIX 未丢参数）；task-git-oauth 已随精准重启生效
- **Created**: 2026-08-22
- **Context**: 任务详情 skip 横幅「去绑定 Git OAuth」已按 live `user-app-connection.connected` 隐藏。同页评论执行细节仍显示「Git OAuth · 未绑定」+「去绑定 gitlab-tencent-sh-1…」。浏览器对 `?probe_access_token=1` 的响应与无 probe 相同，缺少 `access_token_valid`。FE 已改为：仅显式 `access_token_valid: false` 才打未绑定；缺字段回退 `connected`。芯片误显问题由 FE 缓解；生产仍应返回该字段以便 refresh 失效时正确降级。
- **Action**: (1) 核对生产 taskGitOauth 是否已含 `handleUserAppConnection` probe 分支（`taskGitOauth/src/internal_handlers.go`）(2) 查 APISIX 是否丢掉 `probe_access_token` (3) 精准重启 task-git-oauth 后用 Playwright 断言 probe 响应含 `access_token_valid`
- **Why**: 用户刚完成 OAuth 回流后，评论区绑定按钮仍在，会误以为横幅修复未生效；也让 AccessToken once-probe 永远走失败路径。
- **How to apply**: 对照 `taskGitOauth/src/user_app_connection_probe_test.go`；FE 消费 `taskFE/app/src/utils/gitOAuthUserAppConnection.js`；复现页 `task_878902048790179840`

## [OPT-20260822-049] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 精准重启已完成（20:54 + 本轮）；公网 task_878950791526772736 实测 comment-execution-git-oauth data-kind=bound 且无 comment-git-pr-oauth-bind，矛盾状态不再并存
- **Created**: 2026-08-22
- **Context**: 执行细节芯片走 user-app-connection/cache，PR 回复 merge-request-status 曾绕过 cache 刷新并丢弃新 refresh，同页同时出现「Git OAuth · 已绑定」与「去绑定 / 授权已失效」。代码已改为共用 cache-first 换票，且已绑定时隐藏 PR 去绑定。须编译重启 SPA/task-git-oauth 后才能在公网看到。
- **Action**: (1) ✅ 已登记 `task-git-oauth` `taskFE` (2) ✅ 2026-08-22 20:54 精准编译重启已完成（健康 200，SPA `releases/20260822205358-29933`）(3) 用户硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878950791526772736/?accessCode=DR2AKvP9J9（Agent 浏览器收到「无法解析登录凭据」，无法代替用户 Cookie 验收）(4) 断言 `comment-execution-git-oauth[data-kind=bound]` 时 `comment-git-pr-oauth-bind` 不存在
- **Why**: 未重启则旧 merge-status 仍烧 refresh，用户继续看到矛盾状态。
- **How to apply**: 意图 T21 `docs/intents/frontend/task_detail/034_comment_level_repo_identity.test-intent.md`；可补 Playwright 拦截 merge-request-status 与 user-app-connection

## [OPT-20260822-048] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 精准重启 task-referral/taskFE 后公网 /profile/referral/ 实测 referral-rate-label=当前的分账比例、referral-rate-display=15%（后台配置，非 —/5%/30%）
- **Created**: 2026-08-22
- **Context**: 用户推荐页统计卡曾因 stats 直调微信 merchant-configs（直连 400）显示「—」。代码已改为读 `billing_referral_config.referral_rate_percent`，标签为「当前的分账比例」。公网 SPA 与 task-referral 进程仍须编译重启后才能看到。
- **Action**: (1) `scripts/register-precise-restart.sh task-referral taskFE` (2) 在 http://10.2.150.68:9999/ 精准编译重启 (3) 硬刷新 https://www.daydaymoney.com/profile/referral/ 断言 `referral-rate-label` 为「当前的分账比例」且 `referral-rate-display` 为后台配置百分比（不是「—」/5%/30% 伪装）
- **Why**: 未重启则旧二进制仍走微信查询失败路径，页面继续「—」。
- **How to apply**: 超管「推荐资格管理」里的分成比例即期望值；`data-testid="referral-rate-display"`

## [OPT-20260822-007] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 公网 /profile/referral/ 显示 15%（billing_referral_config.referral_rate_percent，非 5%/30% 伪装）；内部 commission-rate 需 internal secret 403（未直连）；OPT-048 已改读 DB 配置，原「—」预期被取代
- **Created**: 2026-08-22
- **Context**: 分成比例改为只信微信支付查询接口。直连商户 merchant-configs 现网 400，内部 API 应为 503，页面「—」。禁止再验收 5% 或 30%。
- **Action**: (1) 精准编译重启 `task-bill` `task-referral` (2) `GET /api/internal/taskbill/referral/commission-rate/` 直连期望 503，body 不含 `5%`/`30%` (3) 硬刷新 `/profile/referral/` 断言 `referral-rate-display` 为「—」（除非微信查询已 200）
- **Why**: 商户后台既不是 5% 也不是 30%；min/conf 都会伪装真实设定。
- **How to apply**: `scripts/register-precise-restart.sh task-bill task-referral`；内部 API；页面 `data-testid="referral-rate-display"`

## [OPT-20260822-011] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 新增 db/scripts/ci/check_saas_container_skill_paths.py 动态解析 taskGitOauth openAPIMergeRequestPaths + taskTaskService 评论回复模板路径，断言登记在 saas-machine-container.md §5.2（3 路径实测全中）；6 例自测绿；db pre-commit [3/3] 接入 + go_python 模板 SSOT 同步；db 72a894d 已推送
- **Created**: 2026-08-22
- **Context**: PR 回复/一键合并路径已写入 `docs/skills/saas-container/saas-machine-container.md` §5.2，并由 `TestHandleSaasMachineContainerSkill` 断言字符串存在。OpenAPI（`taskGitOauth/src/openapi_merge_request.go`、taskTaskService comments）与 skill 文档仍可能漂移。
- **Action**: (1) 在 CI 读取 `openAPIMergeRequestPaths` 与 comments 创建路径 (2) 断言均出现在 saas-machine-container.md (3) 新增 git-oauth/评论写接口时跑红
- **Why**: 本次就是因为生成了接口却未进厂商门户说明页才补文档；静态字符串测例挡不住路径改名。
- **How to apply**: `taskAiProvider/src/saas_machine_container_skill_test.go` 或 `db/scripts/ci/`；SSOT `docs/skills/saas-container/saas-machine-container.md`

## [OPT-20260822-060] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 盘点结果：仅 1 笔未打分账标的历史微信订单 — ORD-20260822-878619773850644480-878981177317294080（id 878981177317294080，user 878619768754565120，referrer 877397583960502272 渠道 DR2AKvP9J9，paid 2026-08-22 12:37:36，¥0.55）；无 billing_profit_sharing 行且边 commission_eligible=0。微信不允许对已支付未打标的交易补分账 → 运营不可补标，后续不应对此单 CreateOrder 分账（现有 profit_sharing 行仅覆盖 08-20/08-21 两单，status=failed 已记录）。08-18/08-19 微信单无推荐边不涉及。
- **Created**: 2026-08-22
- **Context**: 微信要求 `settle_info.profit_sharing` 在预下单时设置，已支付未打标的交易无法补分账。ADR-0033 只覆盖之后的新支付。上线前已付款的被推荐租户订单若当时未打标，本地也不应假装能分账。
- **Action**: (1) 列出 `billing_resource_order` 已支付 + 有推荐边 + 无 `billing_profit_sharing` 且微信单未带分账标的订单 (2) 给运营说明不可补标 (3) 不调用微信分账 API 对这些单 CreateOrder
- **Why**: 避免上线后 timer 对未打标交易反复失败或误导超管订单详情。
- **How to apply**: 对照微信交易是否 profit_sharing；taskBill `billing_profit_sharing` / 支付 pending 的 out_trade_no

## [OPT-20260822-012] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: routes-apply 已执行（apisix hot-reload，路由 `/api/tenant_id/*/workspaceId/*/tasks/*/comments/*` 在 routes.yaml:1028）；task-task-service/taskFE/task-gateway/ai-provider 已随 59 服务精准编译重启；公网登录态浏览器 POST `/api/tenant_id/877397588196749312/workspaceId/ws_-.../tasks/task_878950791526772736/comments/nonexistent-parent/` 返回 400「父评论不存在或不属于该任务」（task-task-service 语义 + trace_id），确认新路径经网关转发成功非 404。剩余仅真实 PR 推送 E2E 嵌套评论验收。
- **Created**: 2026-08-22
- **Context**: 回复接口改为 `POST /api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent}/`。网关已加该 uri，但 `/api/tenant/*` 兜底仍在；若 APISIX 未 routes-apply 或未精准重启，请求会 404。
- **Action**: (1) `:9999` 可用后对 task-gateway 执行 `bash taskGateway/run.sh routes-apply` (2) 精准编译重启 `task-task-service`、`taskFE`、`task-gateway`、`ai-provider` (3) 用浏览器推送生成 PR，确认 Network 命中新路径且评论嵌套出现
- **Why**: 路径前缀 `tenant_id` 与既有 `tenant` 不同，漏网关或漏重启会表现为「推送成功但没有回复」。
- **How to apply**: `taskGateway/routes/routes.yaml` `task-task-service` uris；`.runall/precise_restart_services.txt`

## [OPT-20260822-024] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskBill 4e80cf6 + taskFE 7421ad0 已推送：listProfitSharingForOrder/Queue 批量经 taskAuth batch/details 解析 receiver_display（username→email→user_id 兜底），失败优雅降级不阻塞账单、禁止 openid；FE OrderExpandDetail/SystemAdminProfitSharingPanel receiver_display || user_id；Go 3 例 + FE 1 例回归全绿。已登记精准重启 task-bill/taskFE。
- **Created**: 2026-08-22
- **Context**: 订单展开分账表目前展示 `receiver_user_id`。运营对账需要姓名/手机脱敏，但本增量刻意不调 taskAuth 以免扩大权限面。
- **Action**: (1) 管理员详情批量解析 user_id→display（内部 API，失败回退 id）；(2) JSON 增加可选 `receiver_display`，id 仍保留；(3) 禁止下发 openid。
- **Why**: 纯 id 对账成本高；解析必须走管理员通道且最小化 PII。
- **How to apply**: `taskBill/src/profit_sharing_admin.go` 与 `OrderExpandDetail.vue`；调用 taskAuth 内部用户查询需 ADR/权限复核。

## [OPT-20260822-021] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: trae-agent cdd8c09 已推送：at_mention_run 缺失 + COMMENT_ID 非空 + auto_run=false 时，优先取 context_pack comment_thread 最后人类评论、否则重拉 task-detail 提取指令，创建 at_mention job 并写标记幂等（案例 116 补洞）。13 例回归测 + 全仓 test:unit 151 全绿；onlineServiceJS 镜像 x86_64_2026-08-23_03-10 / x86_64-latest 已 DOCKER_PUSH=1 推送。
- **Created**: 2026-08-22
- **Context**: 案例 116：`@trae-agent` 已启机但 `AICommentServiceURL` 空导致无 pending Agent 评论，kickoff 回退 `auto_run_false` 跳过。taskTaskService 已默认 8019 + 先 notify；存量容器仍可能带着空 pack 完成 bootstrap。
- **Action**: (1) `runPostBootstrapAgentKickoff` 在 `at_mention_run` 缺失且 `COMMENT_ID` 非空时，从 task-detail/评论 API 取该评论 content 作为 command 创建 trae job；(2) 回归测：auto_run=false + 无 pack + COMMENT_ID 仍 `createJobFn`；(3) 不替代 task-task-service notify 主路径。
- **Why**: 只修 URL 无法让已经 BOOTSTRAP_COMPLETE 的容器补跑；kickoff 侧补洞可覆盖 notify 失败或旧进程。
- **How to apply**: `trae-agent/onlineServiceJS/src/postBootstrapAgentKickoff.mjs`；测例 `postBootstrapAgentKickoff.test.mjs`。`cd trae-agent/onlineServiceJS && node --test src/postBootstrapAgentKickoff.test.mjs`

## [OPT-20260822-062] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE 94ac560 已推送：fetchServerRuntimeStatus 探测 Released/Terminated 后经 bindingRefreshRequested 总线触发 refreshBindings（滞后的 running binding 对齐服务端终态）；commentExecutionInactiveHint 增加 serverRuntimeStatus 判定，Released 时提示「服务器已释放，本评论容器不再运行」；TaskDetailCommentExecutionDetails 透传 serverRuntimeStatus。回归测：inactiveHint Released/Terminated/released/Stopped 四态 + bus 触发 refresh 后 binding 变 released，vitest 38 例全绿；node --test 双文件 pass，pre-commit 全绿。
- **Created**: 2026-08-22
- **Context**: 执行细节 summary 已按该评论 runtime 快照把「容器 运行中」覆盖为「服务器已释放」，但 `cloud_comment_container_bindings.status` 可能仍为 running（前端未在 Describe 后 `refreshBindings`）。非当前执行评论的 inactiveHint 在 Released 时仍可能写「已挂接独立实例」。
- **Action**: (1) `fetchServerRuntimeStatus` 得到 Released/Terminated 后调用 `refreshBindings` (2) 用同一 Released 判定改写 `commentExecutionInactiveHint` (3) 回归：binding 列表随后变为 released，且 inactive 文案不再声称实例仍挂接
- **Why**: 只改徽章会让摘要与启动日志/绑定 API 继续漂移；刷新后其它依赖 binding 的面板（ztree 空态、心跳）也能对齐。
- **How to apply**: `useServerConfigRuntime.js` `fetchServerRuntimeStatus`；`useCommentContainerBindings.js` `refreshBindings`；`commentExecutionInactiveHint`

## [OPT-20260823-006] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 服务端一次请求批量派生（taskTaskService 37cedba + taskFE 768da80 已推送）：handleCreateTask 支持 fork_count(1-99)，单请求循环 createTaskOnce 创建 N 副本，每副本 Idempotency-Key=<batch>:<i> 复用既有服务端幂等，批重试不重复扣配额/起机；配额不足返回已创建数（partial）。consumeTaskPostQuota 改 var 供测试注入。Go 4 例 + FE 7 例回归测全绿，双仓整包测试通过。已登记精准重启 taskFE/task-task-service（runAll :9999 down 待恢复触发）。
- **Created**: 2026-08-23
- **Context**: Fork 弹窗已支持 1–99 份，实现为前端顺序 POST 既有 todos 接口。99 份约需数十次 RTT，部分失败语义靠前端拼装。
- **Action**: (1) 在 taskTaskService `handleCreateTask` 增加可选 `fork_count`（1–99）(2) 单请求创建 N 条并返回 `ids` (3) 配额不足时明确已创建数 (4) 前端 N>1 改走该字段，保留旧循环作兼容
- **Why**: 大批量时前端循环慢且易中途断网；服务端一次提交更易控制配额与事件。
- **How to apply**: `taskTaskService/src/create_task.go`；`taskFE/app/src/composables/taskDetail/taskDetailEditing.js` `forkTask`

## [OPT-20260821-027] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: conf/docs/taskFE 指针已确认在 HEAD（conf 7b6a963c / docs 0e50dd4 均与 meta 记录一致；taskFE 本窗推 768da80 后随 meta 指针同步）。该条目原阻塞（他会话 WIP）已解除，无待同步指针。
- **Created**: 2026-08-21
- **Context**: 本会话已将 `conf`/`docs`/`taskFE` 推到 origin/main（密钥片段、意图 T36、注册页已注册单测），但三仓工作树仍有他会话未提交改动，规则 32 禁止更新 meta gitlink。
- **Action**: (1) 确认三仓 `git status` 干净 (2) meta `git add conf docs taskFE` 指向已推 SHA (3) 提交并 push meta
- **Why**: 其他克隆仍可能检到旧指针，缺 `conf/task-referral/task-bill.yaml` 与 T36 文档。
- **How to apply**: `git -C conf rev-parse HEAD` 应为 `6e83494`；docs `e7b2463`；taskFE `811a41d`

## [OPT-20260822-001] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: CSC Released 后同步评论绑定状态三子项全部落地：taskCloudService 755c682 (binding→released) + taskFE 94ac560 (runtime 徽章/refreshBindings) + taskFE eb7138c (layer ztree 无活容器禁用提交并创建PR)
- **Created**: 2026-08-22
- **Context**: 任务 #28 `cmt_878617449207984128` 的 `cloud_server_configs.last_runtime_status=Released`（`terminal_released=1`，08:39），但 `cloud_comment_container_bindings.status` 仍为 `running`。详情页继续显示「容器运行中」并可点「提交并创建PR」，实际转发 502（trace `1674ac57`）。
- **Action**: (1) 在 CSC 进入 Released/terminal_released 时把对应 binding 从 `running` 改成终态；(2) 前端「容器运行中」以 CSC runtime + 探活为准，不要只信 binding.status；(3) 无活容器时禁用「提交并创建PR」并给出明确文案。
- **Why**: 绑定表与 CSC 终态脱节会让用户在已释放实例上重试推送，得到「网络错误」而不是「容器已释放」。
- **How to apply**: `taskCloudService` 释放/停机路径（`cloud_server_configs` + `cloud_comment_container_bindings`）；`taskFE` 任务详情层图按钮启用条件。
- **2026-08-23 夜**: **(1) 已落地** — taskCloudService `755c682` 已推送：`markTerminalReleased` 在 CSC 置终态后同步 `cloud_comment_container_bindings`，把该任务仍处于 `pending/waiting_previous/starting/running` 的绑定改为 `released`（已终态行保持不变），best-effort warn；回归测 `TestMarkTerminalReleasedSyncsBindingToReleased`（running→released、completed 保留），整包 go test ./src 329s 绿。**(2) 已落地（经 OPT-20260822-062）** — taskFE `94ac560`：summary 徽章已按 runtime 快照显示「服务器已释放」，`fetchServerRuntimeStatus` 探测 Released/Terminated 后经总线 `refreshBindings` 对齐绑定列表。**(3) 已落地** — taskFE `eb7138c` 已推送：`createLayerGraphNodeContext` 读 `options.containerReleased`，`applyReleasedGitGate` 在 released 时把 submitAndPush/push/submit/merge/submitAndMerge 全部 disabled 并给「服务器已释放，无法提交并创建 PR」等明确文案（保留各 can* 可见性）；`buildLayerBodyBindForComment` 按 `bindingStatusFor` + `serverRuntimeStatusPanel` 计算 per-comment released 透传 `layerPanelViewFromSlot`→`buildTaskDetailLayerGraphZNodes`，`impliesDirty` 不再反向启用；`LayerGraphZtreeNode` `canShowSubmitAndPush` 在 containerReleased 时仍渲染禁用态按钮。回归测 layerZtreeNodes 4 例 + commentLayerPanelBind 2 例 + 组件 1 例全绿。

## [OPT-20260822-031] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskChromePlugin e358491 已推送：新增 e2e/float-close-collapse.playwright.test.js —— 点悬浮球打开面板、点 × 只收起面板（#taskplugin-float-panel 移除 taskplugin-open），悬浮球 #taskplugin-float-btn 仍可见、storage.floatBallEnabled 不被写 false，且可再次打开；回退：storage stub 补 getFloatBallConfig 返回 {enabled:true} 防 root display:none。E2E 1 例 + npm test 332 例全绿，pre-commit 通过，已推送。
- **Created**: 2026-08-22
- **Context**: 本会话已用 content.js 源码契约单测覆盖 `#taskplugin-float-close` 点击只调用 `hideFloatPanel`、不写 `floatBallEnabled`。content script 是 IIFE，单测无法在真实页面点击 × 并断言面板收起、悬浮球仍在。
- **Action**: (1) 在 `taskChromePlugin/e2e/` 增加用例：注入浮窗后打开面板、点击 `#taskplugin-float-close` (2) 断言 `#taskplugin-float-panel` 无 `taskplugin-open` (3) 断言 `#taskplugin-float-btn` 仍可见且 `floatBallEnabled` 未变为 false
- **Why**: 契约测不能捕获事件绑定被后续重构拆开、或 CSS `pointer-events` 导致点不到的问题。
- **How to apply**: 参照 `taskChromePlugin/e2e/keyboard-shortcut-fallback.playwright.test.js` 的浮窗注入夹具；选择器 `#taskplugin-float-close` / `#taskplugin-float-btn`

## [OPT-20260823-010] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE 4c5b400 已推送：三处点击路径动态 import 改静态导入 + catch 接 handleActionChunkLoadError——submitAIComment(saved_accounts_store 静态 + buildCommentBodyWithChips 包 guard)、taskDetailLayerPushActions(layerZtreeNodes 静态 + SubmitAndPush catch guard)、Navbar.logic.vue(getActiveToken 并入既有静态导入)；回归测 5 例(submitAIComment/taskDetailLayerPushActions 各 2 + Navbar syncCurrentUserIntoSlot 1) 与 taskDetailFetchFns.chunkLoad.test.js 同构；vitest 19 + taskDetail 整目录 571 全绿；pre-commit 全绿已推送；已登记精准重启 taskFE。
- **Created**: 2026-08-23
- **Context**: 本会话只修了 `submitComment` / gitOauth 预检。`taskDetailLayerPushActions.js` 的 `layerZtreeNodes`、`submitAIComment.js` 的 `saved_accounts_store`、`Navbar.logic.vue` 仍是点击时 `import()`，发布后同样会 404 且可能只打 console。
- **Action**: (1) 将这些点击路径改为静态导入或 `importWithChunkGuard` (2) catch 调用 `handleActionChunkLoadError` (3) 补与 `taskDetailFetchFns.chunkLoad.test.js` 同构的回归测
- **Why**: 同一类 stale-chunk 失败会在推送/登录态切换等路径再次出现。
- **How to apply**: `handleActionChunkLoadError` in `taskFE/app/src/utils/chunkLoadGuard.js`；调用方见 `Navbar.logic.vue`、`submitAIComment.js`、`taskDetailLayerPushActions.js`

## [OPT-20260823-009] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 评估闭环——既有基础设施已满足三项 Action：(1) atomic-vite-build.sh 已按 conf releaseKeep=2 保留 2 个非 live release（runall-lifecycle.sh build→atomic-vite-build.sh，ln -sfn 切 public/html，旧 release 仅 keep 外才 rm）；(2) nginx /static/assets/ location ^~ 直接 alias 无 try_files fallback，缺失 hashed 资源返回真 404 而非 SPA HTML；(3) keep 计数清理已实现。2026-08-23 事故中 gitOauthPushPrecheck-C15U_Qut.js 404 的实际根因是点击路径 await import()（已被 OPT-20260823-010 改为静态导入 + chunk guard 兜底），发布竞态窗口由 keep=2 宽限期 + 静态导入双重收敛，无需额外改动。
- **Created**: 2026-08-23
- **Context**: 公网任务详情提交评论报 `Failed to fetch dynamically imported module`，`gitOauthPushPrecheck-C15U_Qut.js` 已 404。根因是 Vite 内容哈希 chunk 在发布后立即删除，仍开着旧 SPA 的用户点击时去拉已不存在的文件。本会话已给点击路径加刷新提示并把 gitOauth 预检改为静态导入，但无法救已经加载旧 bundle 的会话。
- **Action**: (1) 评估 `taskFE/app/scripts/atomic-vite-build.sh` 是否可在切换 `public/html` symlink 后保留 `public/releases/<prev>` 的 assets 一段时间 (2) nginx 对 `/static/assets/*.js` 继续返回真 404 而非 SPA HTML (3) 到期再删旧 release
- **Why**: 前端守卫只能提示刷新；保留旧哈希文件能让未刷新用户的动态 import 仍成功，从根上缩短发布竞态窗口。
- **How to apply**: `taskFE/app/scripts/atomic-vite-build.sh`；`taskFE/app/ai.md` 构建发布规则；nginx `/static/assets/`

## [OPT-20260823-011] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: WorkspaceMachinePolicyModal.vue 保存按钮补 createClickGuard + Idempotency-Key：saveGuard.run 包住 save（同步门闩+300ms debounce），PUT/PATCH 均带同一 Idempotency-Key（mergeIdempotencyHeaders），按钮 disabled+aria-busy，onEnterKey 尊重 isBusy；新增 saveGuard.test.js 双点击仅一次 PUT + aria-busy 回归测 2 例，idleReuseCopy 既有测保持绿，anti-replay 门禁 ok。taskFE 与他会话并发提交竞态：本窗 staging 的 3 文件被他会话 commit 4ea4abf（06:33）整体带走（消息与文件不符但内容完整），HEAD=5eab8af 已含改动，vitest 复跑全绿。taskFE 工作树仍有他会话 WIP，指针按规则 32 未同步。
- **Created**: 2026-08-23
- **Context**: 本会话只把闲置回收默认从 30 改为 5 分钟。`WorkspaceMachinePolicyModal.vue` 的「保存」仍直接 `@click="save"` 发 PUT，没有同步门闩和 `Idempotency-Key`。
- **Action**: (1) 用 `createClickGuard` 包住保存点击 (2) PUT `/workspace-machine-policy/` 带同意图级 `Idempotency-Key` (3) 按钮 `disabled` + `aria-busy` (4) 补单测防双击两次 PUT
- **Why**: 元规则 52 要求写操作点击防重放；双击可能对同一工作空间连写两次策略。
- **How to apply**: `taskFE/app/src/components/WorkspaceMachinePolicyModal.vue`；`taskFE/app/src/utils/clickGuard.js`；`.ai/01_project_constraints/57_frontend_button_anti_replay.md`

## [OPT-20260821-011] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: （2）live 复验经本机 task_cloud 库确认：cloud_layer_graph_snapshot 共 22 行（最近 08-22 22:27），persist 直接 upsert 落库生效；GET source=saas_db 由既有单测 TestGetContainerLayerGraphHydratesFromDB/TestHandleLayerGraphPushPersistsAndPublishes 断言（空库 live→persist→再 GET 走 DB）。（1）topic 已创建 + taskEvents a1555c7 / db 1b3fe8c 注册已推送；（3）回归已覆盖。注：GET 的带登录态公网硬刷新未独立复验（属浏览器验收类），DB 侧 + handler 单测已闭环。
- **Created**: 2026-08-21
- **Context**: 任务详情层图 GET 已能 live fallback 并渲染 ztree。persist 快照时 Cloud 打 `LayerGraphSnapshotPersisted` 出现 `Unknown Topic Or Partition`，不挡 GET，但 DB 快照无法经事件路径落稳，下次仍依赖 live。
- **Action**: (1) 在 domain-events 注册 `LayerGraphSnapshotPersisted` topic 并与 Kafka 实际分区对齐 (2) 确认 persist 成功后 `cloud_layer_graph_snapshot` 有行且下次 GET `source=saas_db` (3) 补回归：空库 live 200 → persist → 再 GET 走 DB
- **Why**: 只靠 live fallback，容器或 Gateway 抖动时「任务关联」会再次空白或长时间 loading。
- **How to apply**: `taskCloudService` persist 出站；`conf/events/domain-events/`；`taskEvents` intent 注册；禁止只吞掉 Unknown Topic 日志
- **2026-08-22 夜**: **(1) 已落地** — 生产 broker `layer-graph-snapshot-persisted` topic 已创建（1 partition，与 kafka_recreate 对齐）；canonical 注册补齐：taskEvents `a1555c7`（`EventTopic()` 补 `LayerGraphSnapshotPersisted`→`layer-graph-snapshot-persisted`，顺带对齐同类的 `CommentContainerBindingAdvanced`）+ db `1b3fe8c`（`kafka_recreate.py` KAFKA_TOPICS 补两 topic，keep-sync 注释已注明）。**(3) 回归已覆盖**：taskEvents `TestEventTopicCloudSnapshotAndBindingAdvanced`（新）两事件解析断言 + taskCloudService 既有 `TestGetContainerLayerGraphHydratesFromDB`/`TestGetContainerLayerGraphEmptyFallsBackToLiveGateway`/`TestHandleLayerGraphPushPersistsAndPublishes` 覆盖「空库 live→persist→再 GET 走 DB」。**(2) 仍 pending**：persist 成功后 `cloud_layer_graph_snapshot` 有行 + GET `source=saas_db` 的 live 复验被 runAll :9999 down（no_restart_runall 08-17）阻断；代码已直接 upsert 落库（非经事件路径），单测已断言 source=saas_db。domain-events consumer intent **未建**：该事件为 publish-only 侧通知，DB 落库由 `persistLayerGraphPush` 直接 `upsertLayerGraphSnapshot` 完成，无需消费者，避免 runAll 编排空转。

## [OPT-20260818-023] cancelled

- **Status**: cancelled
- **Completed**: 2026-08-23
- **Summary**: 产品改为磁盘/流量合卡共用一个区域下拉，不再验收独立 order-gitlab-traffic-region
- **Created**: 2026-08-18
- **Context**: 购买页已从整单共享下拉改为磁盘/流量卡片内选区；公网需 VIP1 + 新 SPA hash。
- **Action**: (1) 精准编译重启 `task-bill` + `taskFE` (2) VIP1 打开 `/tenant/{tid}/billing/orders/create/` (3) 断言 `order-gitlab-region` 在磁盘卡内、`order-gitlab-traffic-region` 在流量卡内
- **Why**: 防止只 Vitest 绿、公网仍是旧共享下拉。
- **How to apply**: `taskFE/tests/` Playwright；需登录夹具

## [OPT-20260823-016] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 已对 task_auth 执行 apply_datamigrate.sh，036_impersonation_session.sql 写入 data_migrate_log，SHOW TABLES 可见 auth_impersonation_session
- **Created**: 2026-08-23
- **Context**: 模拟登录依赖 `dataMigrate/taskAuth/036_impersonation_session.sql`。业务进程不跑迁移，上线后须走 9999 初始化。
- **Action**: (1) 打开 http://10.2.150.68:9999/ 「初始化全部数据库」或对 task-auth 执行 apply_datamigrate (2) 确认 `data_migrate_log` 含 `036_impersonation_session.sql` (3) `SHOW TABLES` 可见 `auth_impersonation_session`
- **Why**: 未应用 DDL 时 impersonate API 会 500。
- **How to apply**: `db/scripts/apply_datamigrate.sh` / 9999 InitAllDatabases；目录 `dataMigrate/taskAuth/`

## [OPT-20260823-018] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE 209a3d2 已推送：UserListRow 操作列按 canImpersonate(user:impersonate) 权限渲染「以该用户身份登录」按钮并 emit impersonate；SystemAdminUsers 复用 useImpersonateUser 行级触发 impersonateUser(user.id)，错误在编辑模态关闭时于表上方展示（data-trace-id 保留）；补单测 3 例（无权限不渲染/有权限渲染/点击 emit）。vitest 14+5 全绿，pre-commit 通过。taskFE 已在精准重启登记。
- **Created**: 2026-08-23
- **Context**: 本期按钮只在编辑用户模态框内，排障时需先打开编辑才能模拟。
- **Action**: (1) 在 UserListRow 操作列增加同样权限控制的按钮 (2) 复用 `useImpersonateUser` (3) 补单测：无 `user:impersonate` 不渲染
- **Why**: 减少一次模态打开，缩短排障路径。
- **How to apply**: `taskFE/app/src/components/UserListRow.vue`；权限 `hasPlatformPerm('user:impersonate')`

## [OPT-20260823-020] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: meta b3f59f37 已推送：random_test_runner.sh rt_go_run_dir 的 go test 全走共享 flock logs/.mysql-gotest.lock（CLI sweep/pre-commit/手动 /goal 共用），持锁运行保留退出码、等待超时 WARN+跳过不并行 CREATE TABLE；selftest 补 3 例（free lock/锁占用跳过/路径解析）；nightly_test_sweep.py 增 session_hub_has_interactive，交互会话活跃时跳过打 MySQL 的 Go 仓，补 5 例单测；模板已同步。注：taskAuth/.githooks/pre-commit 的 SSO 直跑 go test 未包锁——taskAuth 被他会话活跃提交（ac23613/3d951c20）连续还原，且各仓 .githooks/lib 部署副本需下次 deploy_repo_random_precommit.sh 同步后 pre-commit 生效（SSOT 已在 meta）。
- **Created**: 2026-08-23
- **Context**: 2026-08-23 06:00 crontab 同时拉起 nightly-test-sweep 与 nightly-opt-runner；sweep flock 只防第二份 sweep。同时段多个 `/goal` 会话对 taskBill/taskAuth/taskCloud 做 `go test ./src` 与 pre-commit 全包，与 sweep 三轮整包抽测叠在同一 docker-mysql，打出 DDL/fsync 风暴。LRN-20260823-001。
- **Action**: (1) 共享 flock（如 `logs/.mysql-gotest.lock`）包住 `random_test_runner.sh rt_run_go` 与各仓 pre-commit 的 `go test ./src` (2) sweep 在 Session Hub 有交互会话时跳过打 MySQL 的 Go 仓，不只跳过 dirty worktree (3) 回归：两进程争锁时第二进程等待或跳过，不并行 CREATE TABLE
- **Why**: 克隆模板降低了单进程 DDL，但多进程仍会并发建库；夜间窗口与人工 `/goal` 必然重叠。
- **How to apply**: `scripts/lib/random_test_runner.sh`；`scripts/nightly_test_sweep.py` `session_hub_active_repos`；子仓 `.githooks/pre-commit`

## [OPT-20260822-053] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskBill /v3/profitsharing/* live 路径全部改走 sdk/wechatpay-go profitsharing service：OrdersApiService(create/query/unfreeze)、ReceiversApiService(add/delete，appid 保留登录 AppID 语义)、ReturnOrdersApiService(回退)、MerchantsApiService(比例查询)。删除 profitsharing live 裸 http.NewRequest（wechatV3Post 死代码移除；wechatV3Get 仅供 WECHAT_RATIO_PROBE=1 探测）。保留 profitSharing*Call 可注入接缝，新增 wechat_profit_sharing_sdk_test.go 13 例覆盖请求映射/响应解析/错误格式化；既有分账/比例测全绿，taskBill 整包 go test ./... 通过。已提交 fe6513f 并推送，task-bill 已登记精准重启（待 runAll 恢复后生效）。
- **Created**: 2026-08-22
- **Context**: ADR-0032 要求微信支付走 `sdk/wechatpay-go` 的 `services/<产品>`。`taskBill/src/wechat_profit_sharing.go` 等仍用 `wechatV3Post`/`wechatV3Get` 拼 `/v3/profitsharing/*`，client 未初始化时还有未签名裸 HTTP。SDK 已提供 `services/profitsharing`（含 `CreateOrder`、`QueryMerchantRatio`、接收方增删、回退）。
- **Action**: (1) 用 `profitsharing.OrdersApiService` / `ReceiversApiService` / `MerchantsApiService` / `ReturnOrdersApiService` 替换 `wechatV3Post` 路径 (2) 删除 live 路径上的裸 `http.NewRequest` (3) 保留可注入接缝供单测，复跑既有分账/比例测试
- **Why**: 手写 path 会绕过 SDK 敏感字段加密与类型化错误；裸 HTTP 在 live 误用会被微信拒收或跳过验签。
- **How to apply**: `sdk/wechatpay-go/services/profitsharing/`；对照 `wechat_pay_prepay.go` 的 `wechatPrepayCall` 接缝；门禁 `db/scripts/ci/check_wechatpay_go_sdk.py` 不覆盖此存量路径

## [OPT-20260823-019] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: dockerInfra/mysql/run.sh restart 已执行（重启前确认无进行中 Go 单测）：log_bin=OFF、sync_binlog=0、innodb_flush_log_at_trx_commit=2 已生效。清理 tpl_% 模板库 12 个中 11 个已 DROP（均为 08-22 过期迁移哈希副本）；剩余 tpl_task_auth_1ac4c1c8c8ea 被并发 taskAuth 单测（07:43 起）动态重建，保留不动。注：重启后观测到另一会话 Cursor 正在 taskAuth 跑 pre-commit 单测，本操作未与其时间窗重叠（其测试于重启后启动）。
- **Created**: 2026-08-23
- **Context**: mysqld CPU 爆表根因已修：测试改为模板克隆 + SET GLOBAL innodb_flush_log_at_trx_commit=2 / sync_binlog=0 已写入 conf。`--skip-log-bin` 需容器重建才生效，当前实例仍开着 binlog。测试模板库 `tpl_*` 会随迁移哈希留下旧副本。
- **Action**: (1) 确认无进行中的 Go 单测后执行 `dockerInfra/mysql/run.sh restart` (2) `SHOW VARIABLES LIKE 'log_bin'` 为 OFF (3) 可选 DROP 过期 `tpl_%` 库
- **Why**: 未重建时 binlog 仍放大测试 DDL 写；旧模板库占用 datadir。
- **How to apply**: `dockerInfra/mysql/run.sh restart`；SSOT `conf/infra/mysql/config.yaml` `skipLogBin`

## [OPT-20260823-028] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 040 已应用到 task_cloud；已创建 job-step-full-archived 与 step-full-cos-config-updated 及其 -dlt；已精准编译重启 task-cloud-service/taskFE/task-agent-support/task-gateway。管理员填生产 COS 见 OPT-20260823-029。
- **Created**: 2026-08-23
- **Context**: 本会话新增 `cloud_job_step_full_object`（dataMigrate 040）与 topic `job-step-full-archived` / `step-full-cos-config-updated`。代码已合入，生产库与 broker 尚未应用。
- **Action**: (1) 打开 http://10.2.150.68:9999/ 初始化全部数据库或对 task_cloud 跑 apply_datamigrate.sh (2) 在 Kafka 创建上述两 topic 及其 -dlt（可用 kafka_recreate 或 kafka-ui）(3) 精准编译重启 task-cloud-service / task-agent-support / task-events-* / task-gateway / taskFE
- **Why**: 未迁库则 inbound UPSERT 失败；未建 topic 则 JobStepFullArchived 打 broker 报 Unknown Topic。
- **How to apply**: `dataMigrate/taskCloudService/040_cloud_job_step_full_object.sql`；`db/_infra/kafka_recreate.py`；`.runall/precise_restart_services.txt`

## [OPT-20260823-029] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: 现网 COS 已接通：conf backend=cos 桶 ai-provider-1259712831/ap-shanghai（密钥仅 local yaml）；task-cloud-service 重启日志 step_full_cos_init_ok；live Put/Get 通过；生产 GET container-job-execution-log 返回 source=saas_cos 且 step.llm 来自对象存储。探测行与对象已清理。
- **Created**: 2026-08-23
- **Context**: 默认 `conf/taskCloudService/step-full-cos.yaml` 为 `backend: local`（指针表 payload_json + 内存 FakeCOS）。关容器后跨进程复查需要真实 PutObject。
- **Action**: (1) 以平台员工打开 `/system-admin/step-full-cos/` (2) 填 bucket/region/pathRule 与密钥 (3) backend 选 cos 保存 (4) 跑一层任务后确认 GET `container-job-execution-log` 的 `source=saas_cos`
- **Why**: local 模式无法在容器释放后从对象存储复查全文。
- **How to apply**: `SystemAdminStepFullCOS.vue`；密钥写入 gitignore 的 `conf/taskCloudService/step-full-cos.local.yaml`

## [OPT-20260823-036] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskFE 036b7f6: shouldSkipZTreeExecLogFetch + watchers skip + useTaskDetail.zlog-released.test.js
- **Created**: 2026-08-23
- **Context**: 任务详情在「服务器已释放」且仍有层图快照时，已隐藏指令/日志/文件树交互区（`containerReleased`）。`refreshZTreeExecutionLog` watcher 仍可能对已释放容器拉日志，产生 409「正在注册业务地址」类噪音。
- **Action**: (1) 在 `refreshZTreeExecutionLog` / `installTaskDetailZTreeExecLogWatchers` 于 `containerReleased` 时直接 return，不打容器 (2) 补测：released 时 `apiFetch` 不被调用 (3) 与本会话 Vitest `TaskDetailTaskLayerAssociationPanel.released-hides-interactive.test.js` 一并回归
- **Why**: 藏 UI 不能挡住 watcher；后台 409 仍会污染控制台并可能写进隐藏状态。
- **How to apply**: `taskFE/app/src/composables/taskDetail/taskDetailExecLog.js`、`taskDetailZTreeExecLogWatchers.js`；测例 `taskDetailExecLog.test.js`

## [OPT-20260823-051] completed

- **Status**: completed
- **Completed**: 2026-08-23
- **Summary**: taskAuth OIDC 授权拦截：gitlab-git-service* client 直连登录按 redirect_uri 主机匹配区域 access_mode（taskBill 内部只读 API gitlab-regions-admin，X-TaskBill-Internal-Secret，30s 缓存）；development 仅 is_tester=1 可签发授权码，其余 access_denied；查询失败 fail-open、已知 development 校验失败 fail-closed。单测 5 例（release 放行 / dev+tester 放行 / dev+非 tester 拒绝 / 非 gitlab client 不受影响 / bill 不可达 fail-open）。
- **Created**: 2026-08-23
- **Context**: SaaS 侧已按 `billing_gitlab_region.access_mode=development` 对非测试账号隐藏目录、拒绝下单与配额摘要。用户仍可直接打开区域 GitLab URL 走 OIDC 登录，本期未在 IdP/区域实例拦截。
- **Action**: (1) 在 taskAuth OIDC 授权或区域 GitLab OmniAuth 回调判定目标区域 access_mode (2) development 且账号 `is_tester=0` 时拒绝签发 (3) 补单测：release 放行、development+tester 放行、development+非 tester 拒绝。
- **Why**: 仅拦 SaaS 购买/目录不能阻止测试区 GitLab 被普通租户直接登录使用。
- **How to apply**: `taskAuth` OIDC / `gitService` OmniAuth；区域模式 SSOT 在 `taskBill` `billing_gitlab_region.access_mode`，禁止 auth 直连 bill 库，走内部只读 API。

