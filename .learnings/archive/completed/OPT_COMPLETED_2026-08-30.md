# Completed OPT Archive — 2026-08-30

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 19 条。
> 归档执行时间：2026-08-31T13:50:25+08:00

## [OPT-20260829-020] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: db gate scans *.example for high-confidence secrets; self-test 24/24 green
- **Created**: 2026-08-29
- **Context**: 扫描 `trae-agent` 时确认 `trae_config.yaml.example` / `trae_config.json.example` 只有占位 `api_key`。但 `db/scripts/ci/check_no_hardcoded_secrets.py` 的 `is_allowed_secret_store` 对 `*.example` 整文件跳过，PEM / `AKIA` / `ghp_` / `sk_live_` 也不会被扫到。
- **Action**: (1) 允许 `.example` 继续豁免 assignment 占位符；(2) 仍对 HIGH_CONFIDENCE 模式扫描 `.example`；(3) 自测：example 里写 `your_openai_api_key` 通过，写入 PEM 或 `sk_live_` 失败。
- **Why**: 示例配置是最容易被误贴真实密钥的落点；整文件豁免会让门禁对最常见泄漏路径失明。
- **How to apply**: `db/scripts/ci/check_no_hardcoded_secrets.py` 与 `test_check_no_hardcoded_secrets.py`；机器验收：`python3 db/scripts/ci/test_check_no_hardcoded_secrets.py` exit 0。

## [OPT-20260829-019] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: trae-agent run.sh redacts ACCESS_TOKEN in console URLs; self-test passes; online image pushed
- **Created**: 2026-08-29
- **Context**: 2026-08-29 扫描 `trae-agent` 确认版本库无真实密钥，且 `init.log` 已脱敏。但 `onlineServiceJS/run.sh` 在 Docker/本地启动时把完整 `ACCESS_TOKEN` 写进 stderr 控制台 URL（约 L181、L183、L220、L222），终端滚动、CI 日志或录屏会留下明文令牌。
- **Action**: (1) 控制台 URL 打印改为 `/ui/.../<redacted>` 或只打印不含 token 的 origin+path 提示；(2) 本地开发若需可点击链接，改为写到 gitignored 的 runtime 文件而非 stderr；(3) 补 shell/node 测或对 `run.sh` 的断言：输出不含实际 `ACCESS_TOKEN` 值。
- **Why**: 令牌一旦出现在终端日志，泄露面与 `init.log` 脱敏不在同一层；本地 `dev-local-token` 无害，容器注入的真实 token 有害。
- **How to apply**: `trae-agent/onlineServiceJS/run.sh`；对照 `envLogRedact.mjs` 的 redacted 文案。机器验收：`ACCESS_TOKEN=prod-like-token-xyz bash -n onlineServiceJS/run.sh` 且用提取 echo 段的单元/脚本断言 stderr 不含 `prod-like-token-xyz`。

## [OPT-20260829-012] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskTenantService MEMBER_JOINED contract test asserts invitation_id/link_kind/use_count
- **Created**: 2026-08-29
- **Context**: 开放邀请 join 已把 `invitation_id`、`link_kind`、`use_count`、`invitation_exhausted` 写入 `publishMemberJoined` extra，测试意图 T9 要求每次成功 join 发布 MEMBER_JOINED，但 `invite_open_test.go` 只断言 HTTP 201，没有捕获事件载荷。
- **Action**: (1) 在 `taskTenantService` 测试里注入/替换 `publishEvent` 记录器；(2) `TestOpenInviteTwoDistinctUsersJoin` 断言两次 MEMBER_JOINED 且 `invitation_id` 相同、`use_count` 为 1 然后 2；(3) 单次默认 join 仍发布且 `link_kind=single`。
- **Why**: 开放链的下游（git identity、通知）依赖 extra 字段；HTTP 绿不能证明事件契约。
- **How to apply**: `taskTenantService/src/invite_join.go` `publishMemberJoined`；机器验收：`go test ./src -count=1 -run 'OpenInviteTwo|ValidateAndJoin'` 且新测例断言事件名与 extra。

## [OPT-20260829-011] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskTenantService drops invitation_token from INVITATION_CREATED payload
- **Created**: 2026-08-29
- **Context**: `handleInvite` 的 Kafka 载荷仍含 `invitation_token` 明文。开放链接同一 token 供多人使用，日志/Kafka UI 泄露窗口更大；应用日志已改为 `token_fp`。
- **Action**: (1) 从 `INVITATION_CREATED` payload 去掉 `invitation_token`，保留 `invitation_url`（邮件模板用）；(2) 盘点 `taskEvents` 邀请邮件/短信消费者是否读 token 字段，改为从 URL 解析或只读 invitation_id；(3) 补单测：事件 map 无 `invitation_token` 键。
- **Why**: 开放链有效期内 token 等价于入司凭证，进入 Kafka 即扩大泄露面。
- **How to apply**: `taskTenantService/src/invite_handlers.go` `eventPayload`；`taskEvents` invitation-created 消费者；机器验收：`go test ./src -run Invite`（tenant）+ 对应 events 消费者测例无 token 键。

## [OPT-20260829-022] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskAiProvider submit-review guarded with clickGuard + Idempotency-Key; runtime test
- **Created**: 2026-08-29
- **Context**: 版本行删除/下架/激活已走确认弹窗 + clickGuard。`submitReview` 仍直接 POST，双击可能重复提交。
- **Action**: (1) 将 `submitReview` 纳入 `vendorImageActionConfirm.js` 的 guard；(2) 按钮 `disabled`/`aria-busy`；(3) 单测断言 Idempotency-Key。
- **Why**: 与元规则 52 一致，避免待审核状态被重复 submit。
- **How to apply**: `taskAiProvider/frontend/src/composables/useVendorPortalImageGroups.js` `submitReview`；`vendorVersionRowActions.unit.test.js`。

## [OPT-20260829-027] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskFE email/phone invite forms show shared expiry select incl 365
- **Created**: 2026-08-29
- **Context**: 有效期选项与默认 90 天只出现在「复制邀请链接」。邮箱/电话邀请共用 `formData.expiration_days`，现已静默默认 90，但表单上没有选择器。
- **Action**: (1) 把有效期 select 抽成共用控件或在 email/phone 表单复用同一 options；(2) 单测断言三种邀请方式都能选 365。
- **Why**: 少轮换诉求对邮件/短信邀请同样成立。
- **How to apply**: `PeopleInvite.vue` email/phone 表单；`peopleInviteExpiration.js` 已提供 options。

## [OPT-20260829-009] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskFE tenant order voucher labels aligned to 交易单号/商户单号
- **Created**: 2026-08-29
- **Context**: 订单详情摘要用「交易单号 / 商户单号」（与列表查询框一致），但租户订单列表展开复用管理端 `OrderExpandDetail`，文案是「微信支付单号 / 商户订单号」。同一租户两处对照凭证时会对不上号。
- **Action**: (1) 给 `OrderExpandDetail` 增加可选标签 props（默认保持管理端文案）；(2) 租户 `BillingOrders` 传入「交易单号」「商户单号」；(3) 扩展 `BillingOrders.tradeNos.test.js` 断言展开行含「交易单号」而非仅 testid。
- **Why**: 客服让用户「看交易单号」时，详情页和列表展开用词不一致会增加对账成本。
- **How to apply**: `taskFE/app/src/components/system-admin/OrderExpandDetail.vue` 与 `taskFE/app/src/views/BillingOrders.vue`；机器验收：`npx vitest run src/views/BillingOrders.tradeNos.test.js src/components/system-admin/OrderExpandDetail.profitSharing.test.js`。

## [OPT-20260829-007] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskFE invoice apply gated on wechat channel like backend
- **Created**: 2026-08-29
- **Context**: 后端 `applyInvoiceApplication` 只允许微信渠道开票；前端 `canApply` 仍只看 `status=paid` 与金额。正额但 `payment_method=admin_grant` 的订单仍会露出「申请开票」，提交后才 400。
- **Action**: (1) `OrderInvoiceSection` 在 `canApply` 增加与后端一致的微信渠道判断（`payment_method`/`payment_ref`）；(2) 非微信已支付显示只读说明；(3) 补 Vitest：admin_grant 正额无按钮。
- **Why**: 避免用户点开弹层后才发现不能开票，也减少无效 POST。
- **How to apply**: `taskFE/app/src/components/OrderInvoiceSection.vue` 与 `orderRefundChannel` 语义对齐（勿复制后端私有函数，按 JSON 已有字段判断）；机器验收：`npx vitest run src/components/OrderInvoiceSection.test.js`。

## [OPT-20260829-006] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskFE billing trade-no search persisted to URL query
- **Created**: 2026-08-29
- **Context**: 本轮在订单列表增加了交易单号/商户单号查询，但查询串只活在组件状态里，刷新或分享链接会丢掉筛选。管理端订单页已有 `order_id` / `order_number` 深链。
- **Action**: (1) 查询成功后把规范化后的值写入 `?order_number=`（或与现有 `order_id` 互斥）；(2) 进入页面时若 query 有该键则预填输入框并请求；(3) 清空时去掉该 query。
- **Why**: 客服/用户对账时经常要发「就是这一单」的链接；没有 URL 状态只能口头报单号。
- **How to apply**: `taskFE/app/src/views/BillingOrders.vue` 的 `onTradeNoSearch` / `onTradeNoClear` 与 `useBillingOrderIdDeepLink` 对齐；单测扩展 `BillingOrders.tradeNoSearch.test.js`。

## [OPT-20260829-018] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: promtail scrapes APISIX logs, x-trace-id as structured_metadata; Loki verified
- **Created**: 2026-08-29
- **Context**: WeChat callback `trace_id=cf7b5ea2918884602b81fd80eae1efb1` 的 502 由 APISIX `error_page` 本地生成，从未进入 taskAuth。Loki `{job=~".+"}` 0 条；证据只在 `taskGateway/logs/error.log` / `access.log`。OPT-20260829-005 只覆盖 promtail 容器常驻，不覆盖网关日志路径。
- **Action**: (1) 在 `AiMonitor/promtail/promtail-local.yaml`（或 generator）增加 scrape `taskGateway/logs/access.log` 与 `error.log`；(2) pipeline 提取 `x-trace-id` / `trace_id` 为 structured_metadata；(3) 用一笔故意 upstream down 的请求断言 Loki `{job=~".+"} |= "<trace_id>"` 命中 APISIX 行。
- **Why**: 登录/回调 502 发生在 taskAuth 进程之外时，TraceId 门禁在 Loki 全空只能猜微信配置。
- **How to apply**: `AiMonitor/promtail/`；`taskGateway/logs/`；机器验收：`python3 -c "import yaml; yaml.safe_load(open('AiMonitor/promtail/promtail-local.yaml'))"` 且 Loki `query_range` 对该 trace 非空。

## [OPT-20260829-025] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskFE sync triggers container reclone + 需重新克隆 badge
- **Created**: 2026-08-29
- **Context**: 任务 `task_881388002226499584` 记录仍是 `github.com/ruandao/helloworld`，项目已改为 `test-ruandao/helloworld.git`。已修复 `repo_address_mismatch` 检测与「同步仓库地址」徽章，但 PATCH 只改 `task_projects.repo_address`，运行中容器 `origin` 仍指向旧仓，推送会继续失败。
- **Action**: (1) 在 `syncStaleTaskRepoAddresses` 成功后，若容器已注册则对当前 git_repos 调用 `onRepoReclone`；(2) 同步按钮旁展示「需重新克隆」直到 origin 与项目 URL 规范化一致；(3) Vitest 断言 sync 成功后发出 reclone。
- **Why**: 用户点同步后徽章消失，误以为推送已改到新仓；实际 `prefer_container_remote` 仍推旧 origin。
- **How to apply**: `taskFE/app/src/composables/taskDetail/taskDetailProjectRepoState.js` 的 `syncStaleTaskRepoAddresses`；`taskDetailRepoReclone.js`。机器验收：对应 Vitest 断言 sync 后 `onRepoReclone` 被调用。

## [OPT-20260829-002] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: taskAuth POST /api/public/email-resubscribe/ (HMAC + Idempotency-Key, 幂等删 auth_email_unsubscription 行, 发 EMAIL_RESUBSCRIBED)；taskFE UnsubscribeConfirm.vue 重新接收邀请邮件按钮(clickGuard)；taskEvents topic 映射。验收: taskAuth go test ./src -run TestPublicEmailResubscribe ok；taskFE vitest UnsubscribeConfirm 5 通过。
- **Created**: 2026-08-29
- **Context**: 本轮交付了邀请邮件退订列表与跳过 SMTP，确认页只有「已退订」说明，没有把邮箱移出列表的入口。误点退订或后续想再收邀请邮件的用户只能走运维改库。
- **Action**: (1) 在 `/auth/unsubscribe/` 增加带 HMAC token 的「重新接收邀请邮件」按钮（写操作须 clickGuard + Idempotency-Key）；(2) 增加 `DELETE` 或 `POST /api/public/email-resubscribe/` 幂等删除 `auth_email_unsubscription` 行；(3) 发布 `EMAIL_RESUBSCRIBED` 并补单测。
- **Why**: 退订是单向集合写入，没有对等恢复路径会导致支持成本上升，也与常见 List-Unsubscribe 产品预期不一致。
- **How to apply**: `taskAuth/src/auth_email_unsubscription.go`、`taskFE/app/src/views/UnsubscribeConfirm.vue`、`dataMigrate` 无需改表（删行即可）；权限上保持公开 token 与退订同一 HMAC。机器验收：`go test ./src -run 'PublicUnsubscribe|Resubscribe'`（taskAuth）与 `npx vitest run src/views/UnsubscribeConfirm.test.js`（taskFE）均 exit 0。

## [OPT-20260830-008] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: Playwright extTest 硬刷新后 build-time 2026-08-30T06:33:25.833Z；无 TypeError；关联项目显示 helloworld + 仓库地址已变更，不再暂无关联项目。409 clone-log 为容器已释放的预期响应。
- **Created**: 2026-08-30
- **Context**: extTest 打开 `task_881388002226499584` 时 GET 已返回 helloworld `projects`，但 `runtimeSectionProps` 对未从 `useTaskDetail` 导出的 `staleRepoSyncNeedsReclone` 读 `.value` 抛 TypeError，关联项目面板停在「暂无关联项目」。源码已导出该 ref。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `taskFE` 点「精准编译重启」；(2) 硬刷新后 `build-time` 新于本修复；(3) Playwright 以 extTest 打开该 URL，断言无 `Cannot read properties of undefined (reading 'value')`，`[data-testid=task-linked-projects-panel]` 不含「暂无关联项目」。
- **Why**: 公网仍跑旧 SPA 时用户继续看到空态与控制台报错。
- **How to apply**: `scripts/register-precise-restart.sh taskFE` 已登记；CDP 9222 + `playwrightLoginWithLegalAccept`。

## [OPT-20260830-010] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: Playwright TaskDetail.server-start-history-empty-state: URL taskId 拉历史，断言无缺少任务ID、空态一条、history query 含 task_id；exit 0
- **Created**: 2026-08-30
- **Context**: 任务详情 compute 拉取曾只读 `props.task.id`，URL 已有 `taskId` 时面板仍显示「缺少任务ID」。单测已覆盖路由回退；公网硬刷新与 hashed 入口 JS 仍需 E2E。
- **Action**: (1) 在 `taskFE/tests/` 增加 Playwright：打开带 `task-detail/:taskId/` 的任务页；(2) 点击「历史服务器启动记录」；(3) 断言 `[data-testid=server-start-history-message]` 文案不是「缺少任务ID」，空成功则为「暂无历史服务器启动记录」或历史卡片。
- **Why**: 单测打不到生产 hashed bundle 与真实路由 params。
- **How to apply**: `taskFE/tests/TaskDetail.server-start-history-empty-state.playwright.test.js` 或新文件；`npx playwright test` 对应文件 exit 0。

## [OPT-20260830-013] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: Created private github.com/task2money/daydaymoney-deploy; seed excludes config.local.yaml and *Pwd.md.
- **Created**: 2026-08-30
- **Context**: 配置仓名与托管已锁定（`github.com/task2money/daydaymoney-deploy`，私有），本会话未 `gh repo create`（避免无凭据空转）。现网仍单仓 conf。
- **Action**: (1) 建私有仓；(2) 按 `envs/<env>/conf/` 拷贝当前 `conf/` 与 docker 配方骨架、`releases.yaml`；(3) 本机 runAll 可改 `--config` 指向 clone，源码仓 conf 暂保留双写。
- **Why**: P1 不落地则 P0 环境变量在生产没有独立配置源。
- **How to apply**: 平台 GitHub；密钥仍留主机 `config.local.yaml`，不进该仓。

## [OPT-20260830-014] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: File + GitHub Release ArtifactFetcher; runAll -command deploy-sync; DEPLOY_MODE strips build.
- **Created**: 2026-08-30
- **Context**: `SyncPinnedArtifacts` 已有端口与假 fetcher 测例，部署机还没有真实拉包实现，也没有 `-command deploy-sync` 入口。
- **Action**: (1) 用只读 Packages token 实现 `ArtifactFetcher`（禁止日志打 token）；(2) 增加 runAll 命令读 `$DEPLOY_ROOT/releases.yaml` 后 sync；(3) 拉失败保留 last-good。
- **Why**: 没有真实 fetcher 则 P2 升级路径无法在无源码主机上执行。
- **How to apply**: `runAll/src/deploy_pin.go` 旁新增 fetcher 文件 + `main` 命令分支；测例继续用 fake。

## [OPT-20260830-016] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: Source contract: conf.example + check_no_prod_conf_in_source.py + meta-rule 47. Live conf/ submodule kept as dual-write so this host keeps booting. Physical unmount is P4.
- **Created**: 2026-08-30
- **Context**: ADR-0052 P0 已让 `FindConfigRoot` 认 `CONF_ROOT`/`DEPLOY_ROOT`，但现网 runAll 仍读 monorepo `conf/`。P3 才把运行时 SSOT 换成 `daydaymoney-deploy`，源码仓只留 `conf.example/`。
- **Action**: (1) 双写验证后把生产拓扑从源码仓 `conf/` 去掉；(2) CI 拒绝再提交完整运行时 `conf/`；(3) 回写元规则 42/`conf/ai.md`。
- **Why**: 源码 clone 仍能看到部署拓扑，隔离目标未完成。
- **How to apply**: `conf/` 子仓、`conf.example/`、`.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md`；门禁脚本放 `db/scripts/ci/`。

## [OPT-20260830-018] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: ADR-0052 + conf.example/ai.md document host db/registry.yaml; /tmp/ram-deploy/db/registry.yaml present; task-auth cwd=/tmp/ram-deploy finds it (exe=/tmp/ram-deploy/bin/taskAuth).
- **Created**: 2026-08-30
- **Context**: 切到 `/tmp/ram-deploy` 后 task-auth 因 `CONF_ROOT` 把配置根指到部署目录，打开 `/tmp/ram-deploy/db/registry.yaml` 失败而退出；task-gateway 健康检查转发 :8003 随之 502。已在本机 `ln -s` 并改 `prepare-ram-deploy.sh`，文档/ADR 仍只写 conf + 二进制。
- **Action**: (1) ADR-0052 / `conf.example/ai.md` 写明部署根必须有 `db/registry.yaml`（可 symlink 或拷贝，不含业务库数据）；(2) 确认 `taskCredentialService` 等 walk `db/registry.yaml` 的路径在有 `CONF_ROOT` 时也能解析；(3) 无源码主机验收：缺 db 则启动失败信息指向该文件。
- **Why**: 只交付 conf 仓不够；DSN SSOT 在 `db/`，漏了会让认证网关整链挂掉。
- **How to apply**: `docs/adr/0052-*.md`、`conf.example/ai.md`、`runAll/scripts/prepare-ram-deploy.sh`（已含 `db` symlink）。

## [OPT-20260830-023] completed

- **Status**: completed
- **Completed**: 2026-08-30
- **Summary**: check_p4 prunes shareLib generated .go and dockerInfra data.corrupt.* symlinks; tests added; /tmp/ram-deploy check exits 0
- **Created**: 2026-08-30
- **Context**: P4 `check_p4_deploy_root.sh` 在复制 `trae-agent/onlineServiceJS/run.sh` 后仍因 `/tmp/ram-deploy/shareLib/gatewaycors/allow_headers_gen.go` 失败。该文件不是本次 run.sh 引入的。
- **Action**: (1) 确认 `shareLib/gatewaycors` 是否为 APISIX 路由生成所需；(2) 若需要则让 checker 排除该路径；(3) 若不需要则从 `$DEPLOY_ROOT/shareLib` 删除 `.go` 并禁止再同步。
- **Why**: 门禁与真实部署树不一致，后续 materialize 会被误判失败。
- **How to apply**: `bash runAll/scripts/check_p4_deploy_root.sh /tmp/ram-deploy` 退出码 0；`find /tmp/ram-deploy -name '*.go'` 只含豁免清单。

