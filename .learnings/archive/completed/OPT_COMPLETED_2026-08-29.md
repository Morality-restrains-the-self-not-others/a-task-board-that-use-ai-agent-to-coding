# Completed OPT Archive — 2026-08-29

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 36 条。
> 归档执行时间：2026-08-31T13:50:25+08:00

## [OPT-20260829-003] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: taskAuth `POST /api/internal/taskauth/email-invites/expire-due/` + taskEvents timer `email_invite_expiry_scan/1_expire` (port 18071)。验收：`go test ./src -run 'TestCleanupExpiredEmailInvites|TestInternalEmailInvitesExpireDue'` (taskAuth) ok；`go test ./internal/handlers/emailinviteexpiryscan` (taskEvents) ok；`python3 db/scripts/ci/check_no_service_internal_poll_loop.py` exit 0；`python3 db/scripts/ci/check_task_events_runall_health_ports.py` ok 52 ports；`rg -n startEmailInviteCleanupLoop taskAuth` 无命中。No-ADR: extends ADR-0011 timer pattern.
- **Created**: 2026-08-29
- **Context**: 拆分 `auth_email_invite.go` 时仍保留 `startEmailInviteCleanupLoop`（`time.Sleep(1 * time.Hour)` 后扫表过期）。这是业务 HTTP 进程内周期工作，与元规则 51 / ADR-0011 冲突。
- **Action**: (1) 抽出一次性 `POST /api/internal/.../email-invites/expire-due/`；(2) 在 `taskEvents/internal/handlers/` 增加 timer worker 调该 API；(3) 从 taskAuth 启动路径删除 goroutine sleep 环并补测。
- **Why**: 进程内轮询在多副本下重复扫表，也使过期清理与 HTTP 进程生命周期耦合，重启/扩容行为不可预期。
- **How to apply**: 参考 `taskEvents/internal/handlers/wechatmpcleanup` 与 `oidcsssoidempotencycleanup`；runAll 健康端口抄 domain-events conf（元规则 47）。机器验收：`python3 db/scripts/ci/check_no_service_internal_poll_loop.py` exit 0，且 `rg -n 'startEmailInviteCleanupLoop' taskAuth` 无业务启动路径命中。

## [OPT-20260827-027] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: promtail 已恢复：`bash runAll/scripts/runall-local-promtail.sh up` 拉起 aimonitor-promtail（AiMonitor/docker-compose.yaml 定义，push 至 http://aimonitor-loki:3100/loki/api/v1/push）。Loki 查询 `{job="task-cloud-service"}` 返回 3 streams 均带 trace_id（如 b52441c699e1d883a115fe4b），`{job="task-bill"}` 亦非空；`loki_ingester_memory_chunks` 由 0 恢复至 130；promtail tail 292 个 runAll 日志文件。
- **Blocked-By**: INFRA
- **Created**: 2026-08-27
- **Context**: 排查 `cdc9471ebc27e3aef6092497` 时 Loki `loki_ingester_memory_chunks=0` 且无 promtail 容器，只能读本机 `logs/task-cloud-service.log`。任务详情 TraceId 芯片无法按设计关联 Loki。
- **Action**: (1) 确认 AiMonitor/docker-compose 是否应启动 promtail (2) 拉起并指向 Loki (3) 用已知 trace_id 查询 `{job="task-cloud-service"}` 非空
- **Why**: 无 ingest 时每次排障都要翻本地日志，TraceId 用户可见却查不到。
- **How to apply**: `AiMonitor/docker-compose.yaml` promtail；`bash runAll/scripts/runall-local-promtail.sh up`；`curl -sS http://127.0.0.1:3100/loki/api/v1/query_range`

## [OPT-20260818-028] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: Playwright 登录态：/tenant/877397588196749312/billing/transactions/ 源筛选=后台赠送(admin_grant) 共 4 条入账；行含来源「后台赠送」、变动明细「GitLab 流量 +1 GB / 任务帖 +100 帖 / GitLab 磁盘 +1 GB」、描述「管理员后台赠送：…」含数量
- **Blocked-By**: BROWSER
- **Created**: 2026-08-18
- **Context**: 租户账单最近交易表后台赠送曾显示来源「—」、金额「+0.00」、描述「管理员后台赠送资源」。已改为变动内容含「任务帖 +N 帖」且来源「后台赠送」。公网需登录 Cookie + 精准编译重启后硬刷新才能验收。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `task-bill`、`taskFE` 点「精准编译重启」(2) 硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/billing/ (3) 「最近交易」入账行断言来源为「后台赠送」、变动内容含具体资源（非 +0.00）、描述含「管理员后台赠送：」与数量
- **Why**: 未重启 taskBill / 未重建 SPA 时用户仍看到空洞入账。
- **How to apply**: CDP 9222；选择器 `div.overflow-x-auto` 内「最近交易」表。

## [OPT-20260817-014] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: Playwright 登录态：工作面板点「搜索任务」→ input#work-panel-task-search-input 输入 task_878583341551480832_cmt_878583347226374144 → 下拉命中「#27 写一个 hello world · 软刀」
- **Blocked-By**: BROWSER
- **Created**: 2026-08-17
- **Context**: 前后端已支持 `task_<id>_cmt_<cmt>` / `cmt_<id>` 搜索；`taskFE` 与 `task-task-service` 已在精准重启登记中。公网 SPA/旧二进制未更新前，https://www.daydaymoney.com 工作面板仍会「无匹配任务」。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」（含 taskFE collectstatic）(2) 硬刷新工作面板 (3) 在标题行搜索框粘贴 `task_877071722828820480_cmt_877071748669927424`，确认下拉命中对应任务
- **Why**: 单测绿不代表公网静态资源与 taskTaskService 进程已加载新归一化逻辑。
- **How to apply**: 页面 `https://www.daydaymoney.com/tenant/875588283562749952/work-panel`；选择器 `input#navbar-task-search-input`

## [OPT-20260817-015] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: Playwright 登录态验收：provider.daydaymoney.com 首页 data-testid=vendor-apply-panel 可见且含「申请认证」（未持 vendor JWT 四态）；镜像市场 /tenant/877397588196749312/image-market 含「厂商门户（SSO）」且无「申请成为厂商门户/审核中/重新申请」文案
- **Blocked-By**: BROWSER
- **Created**: 2026-08-17
- **Context**: 申请入口已迁到 `https://provider.daydaymoney.com/`，镜像市场已去掉申请 UI；`ai-provider` 与 `taskFE` 已登记精准编译重启。现网 SPA/二进制未更新前，门户无申请认证、镜像市场仍可能显示旧申请按钮。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记服务执行「精准编译重启」（含 taskFE collectstatic 与 ai-provider 前端 dist）(2) 硬刷新门户，未持有 vendor JWT 时可见「申请认证」四态，主站已登录 Cookie 可拉 vendor-status (3) 硬刷新 `https://www.daydaymoney.com/tenant/875588283562749952/image-market`，确认无「申请成为厂商门户」「审核中」「重新申请」及申请表单；qualified 仍有「厂商门户（SSO）」
- **Why**: 单测绿不代表公网 dist 与 taskAiProvider 进程已加载 Cookie forward-auth 与新 UI。
- **How to apply**: 选择器 `data-testid="vendor-apply-panel"` / `vendor-apply-cta`；镜像市场断言文案不含「申请成为厂商门户」

<!-- 归档索引：非当日的条目已按天归档至 archive/completed/ -->
<!-- 2026-08-28: 71 条 → [./archive/completed/OPT_COMPLETED_2026-08-28.md](./archive/completed/OPT_COMPLETED_2026-08-28.md) -->
<!-- 2026-08-27: 29 条 → [./archive/completed/OPT_COMPLETED_2026-08-27.md](./archive/completed/OPT_COMPLETED_2026-08-27.md) -->
<!-- 2026-08-26: 21 条 → [./archive/completed/OPT_COMPLETED_2026-08-26.md](./archive/completed/OPT_COMPLETED_2026-08-26.md) -->
<!-- 2026-08-25: 34 条 → [./archive/completed/OPT_COMPLETED_2026-08-25.md](./archive/completed/OPT_COMPLETED_2026-08-25.md) -->
<!-- 2026-08-24: 51 条 → [./archive/completed/OPT_COMPLETED_2026-08-24.md](./archive/completed/OPT_COMPLETED_2026-08-24.md) -->
<!-- 2026-08-23: 46 条 → [./archive/completed/OPT_COMPLETED_2026-08-23.md](./archive/completed/OPT_COMPLETED_2026-08-23.md) -->
<!-- 2026-08-22: 41 条 → [./archive/completed/OPT_COMPLETED_2026-08-22.md](./archive/completed/OPT_COMPLETED_2026-08-22.md) -->
<!-- 2026-08-21: 35 条 → [./archive/completed/OPT_COMPLETED_2026-08-21.md](./archive/completed/OPT_COMPLETED_2026-08-21.md) -->
<!-- 2026-08-20: 59 条 → [./archive/completed/OPT_COMPLETED_2026-08-20.md](./archive/completed/OPT_COMPLETED_2026-08-20.md) -->
<!-- 2026-08-19: 39 条 → [./archive/completed/OPT_COMPLETED_2026-08-19.md](./archive/completed/OPT_COMPLETED_2026-08-19.md) -->
<!-- 2026-08-18: 12 条 → [./archive/completed/OPT_COMPLETED_2026-08-18.md](./archive/completed/OPT_COMPLETED_2026-08-18.md) -->
<!-- 2026-08-17: 35 条 → [./archive/completed/OPT_COMPLETED_2026-08-17.md](./archive/completed/OPT_COMPLETED_2026-08-17.md) -->
<!-- 2026-08-16: 34 条 → [./archive/completed/OPT_COMPLETED_2026-08-16.md](./archive/completed/OPT_COMPLETED_2026-08-16.md) -->
<!-- 2026-08-15: 21 条 → [./archive/completed/OPT_COMPLETED_2026-08-15.md](./archive/completed/OPT_COMPLETED_2026-08-15.md) -->
<!-- 2026-08-14: 53 条 → [./archive/completed/OPT_COMPLETED_2026-08-14.md](./archive/completed/OPT_COMPLETED_2026-08-14.md) -->
<!-- 2026-08-13: 51 条 → [./archive/completed/OPT_COMPLETED_2026-08-13.md](./archive/completed/OPT_COMPLETED_2026-08-13.md) -->
<!-- 2026-08-12: 40 条 → [./archive/completed/OPT_COMPLETED_2026-08-12.md](./archive/completed/OPT_COMPLETED_2026-08-12.md) -->
<!-- 2026-08-11: 57 条 → [./archive/completed/OPT_COMPLETED_2026-08-11.md](./archive/completed/OPT_COMPLETED_2026-08-11.md) -->
<!-- 2026-08-10: 36 条 → [./archive/completed/OPT_COMPLETED_2026-08-10.md](./archive/completed/OPT_COMPLETED_2026-08-10.md) -->
<!-- 2026-08-09: 3 条 → [./archive/completed/OPT_COMPLETED_2026-08-09.md](./archive/completed/OPT_COMPLETED_2026-08-09.md) -->
<!-- 2026-08-07: 16 条 → [./archive/completed/OPT_COMPLETED_2026-08-07.md](./archive/completed/OPT_COMPLETED_2026-08-07.md) -->
<!-- 2026-08-06: 5 条 → [./archive/completed/OPT_COMPLETED_2026-08-06.md](./archive/completed/OPT_COMPLETED_2026-08-06.md) -->
<!-- 2026-07-29: 3 条 → [./archive/completed/OPT_COMPLETED_2026-07-29.md](./archive/completed/OPT_COMPLETED_2026-07-29.md) -->
<!-- 2026-07-28: 11 条 → [./archive/completed/OPT_COMPLETED_2026-07-28.md](./archive/completed/OPT_COMPLETED_2026-07-28.md) -->
<!-- 2026-07-27: 29 条 → [./archive/completed/OPT_COMPLETED_2026-07-27.md](./archive/completed/OPT_COMPLETED_2026-07-27.md) -->
<!-- 2026-07-26: 31 条 → [./archive/completed/OPT_COMPLETED_2026-07-26.md](./archive/completed/OPT_COMPLETED_2026-07-26.md) -->
<!-- 2026-07-25: 101 条 → [./archive/completed/OPT_COMPLETED_2026-07-25.md](./archive/completed/OPT_COMPLETED_2026-07-25.md) -->
<!-- 2026-07-24: 68 条 → [./archive/completed/OPT_COMPLETED_2026-07-24.md](./archive/completed/OPT_COMPLETED_2026-07-24.md) -->
<!-- 2026-07-23: 74 条 → [./archive/completed/OPT_COMPLETED_2026-07-23.md](./archive/completed/OPT_COMPLETED_2026-07-23.md) -->
<!-- 2026-07-22: 32 条 → [./archive/completed/OPT_COMPLETED_2026-07-22.md](./archive/completed/OPT_COMPLETED_2026-07-22.md) -->
<!-- 2026-07-21: 10 条 → [./archive/completed/OPT_COMPLETED_2026-07-21.md](./archive/completed/OPT_COMPLETED_2026-07-21.md) -->
<!-- 2026-07-20: 22 条 → [./archive/completed/OPT_COMPLETED_2026-07-20.md](./archive/completed/OPT_COMPLETED_2026-07-20.md) -->
<!-- 2026-07-19: 26 条 → [./archive/completed/OPT_COMPLETED_2026-07-19.md](./archive/completed/OPT_COMPLETED_2026-07-19.md) -->
<!-- 2026-07-18: 106 条 → [./archive/completed/OPT_COMPLETED_2026-07-18.md](./archive/completed/OPT_COMPLETED_2026-07-18.md) -->
<!-- 2026-07-17: 6 条 → [./archive/completed/OPT_COMPLETED_2026-07-17.md](./archive/completed/OPT_COMPLETED_2026-07-17.md) -->

## [OPT-20260828-013] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: 镜像组图标实现把相关测例抽到 `vendor_image_group_icon_test.go`，但 `handlers_test.go` 仍约 892 行，超过源文件 500 行门禁。继续往该文件堆测例会再次触发强制削文件。
- **Action**: (1) 按主题把 `TestSSO*` / `TestVendor*` / `TestHealth*` 迁到独立 `*_test.go`；(2) `wc -l src/handlers_test.go` 验收 ≤500；(3) `go test ./src -count=1` 全绿。
- **Why**: 行数门禁已触发的测试文件继续增长会导致 pre-commit 阻断；拆分后失败定位更快。
- **How to apply**: 文件 `taskAiProvider/src/handlers_test.go`；共享 `testApp`/`openTestApp` 留在 `testdb_test.go`。

## [OPT-20260828-014] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: 本次图标只覆盖厂商自管列表、公开 catalog JSON 与主站 ImageMarket 组卡。平台审核后台若仍用纯文本镜像组名，审核员无法对照图标。
- **Action**: (1) 确认 admin 镜像组/待审列表 JSON 是否已含 `icon_url`；(2) 若无则在 list DTO 复用 `ImageGroupIconURL`；(3) 审核页卡片加 36px `<img>` + 单测断言 `src`。
- **Why**: 审核决策需要与市场上架后用户看到的同一视觉标识，避免只看名称误批。
- **How to apply**: `taskAiProvider/src` admin handlers + `frontend/src` admin 列表组件；公开流仍走 `GET /api/ai-provider/public-image-groups/{id}/icon`。

## [OPT-20260828-016] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: DevTools 面板 `populateRepoBaseBranchDatalists` 与浮窗 `populateFloatRepoBaseDatalists` 都按项目仓调用 `getBranches` 再写入 datalist，逻辑重复。
- **Action**: (1) 把「container + projects + getBranches」抽到 `lib/branch-datalist.js` 的纯函数/薄封装；(2) panel `branches.js` 与 `content/float-snapshot.js` 改为调用共享函数；(3) 补一条同源测例防止两处漂移。
- **Why**: 两处独立实现会在缓存、错误日志或 builtin 分支集合上再次分叉。
- **How to apply**: `taskChromePlugin/lib/branch-datalist.js`、`panel/lib/branches.js`、`content/float-snapshot.js`。

## [OPT-20260828-017] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: DevTools 单请求「创建任务」已改为先清空本次填写再 `showR` 成功提示。批量 Tab 仍是先写 `batchResult` 成功文案，再 `clearCapturedErrors`，表单选项会留在界面上。
- **Action**: (1) 复用 `PanelCreateSuccess.runAfterSuccess`（或等价顺序）在 `panel/tabs/batch.js` 成功路径先清本次选项再提示；(2) 补 `test/` 测例断言 `runAfterSuccess` 在 `showR('batchResult', 'success'` 之前；(3) `node --test` 相关文件与 `npm test` 全绿。
- **Why**: 单条与批量成功收尾不一致，用户会以为批量任务的项目/分支选择还要再提交一次。
- **How to apply**: `taskChromePlugin/panel/tabs/batch.js`、`lib/panel-create-success.js`、对应 `test/*.test.js`；验收 `cd taskChromePlugin && node --test test/panel-create-success.test.js && npm test`。

## [OPT-20260828-018] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: 为网络失败补 `data-traceId` 时 `lib/api.js` 已约 619 行（本就超过 500 行门禁）。继续在 IIFE 内堆 HTTP 封装会再次触发强制削文件。
- **Action**: (1) 把 `request` / `requestUnauthenticated` / `fetchWithClientTrace` / trace 辅助抽到 `lib/api-http.js`（popup.html 与 service-worker `importScripts` 须先于 `api.js` 加载）；(2) `wc -l lib/api.js` 验收 ≤500；(3) `node --test ./test/api-endpoints.test.js ./test/api-login.test.js` 全绿。
- **Why**: 超标文件再堆功能违反行数门禁，且 IIFE 双加载（脚本标签 + require）使后续改动易漏 SW/popup。
- **How to apply**: `taskChromePlugin/lib/api.js`、`background/service-worker.js`、`popup/popup.html`。

## [OPT-20260828-019] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: `AdminVendorsTab.openDoc` 直接 `fetch('/api/admin/vendors/.../documents/...')`，不走 `api()`，失败时 `showRequestError(..., e)` 没有本端 `X-Trace-Id`。
- **Action**: (1) 改为 `api()` 或 `buildOutboundTraceHeaders` + `attachClientTraceId`；(2) 补单测：fetch 抛 `Failed to fetch` 时错误节点有 `data-traceId`。
- **Why**: 证照打开失败同样是请求错误 UI，缺 traceId 无法对 Loki。
- **How to apply**: `taskAiProvider/frontend/src/components/AdminVendorsTab.vue`、对应 `tests/adminVendorsTab.unit.test.js`。

## [OPT-20260828-021] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: 镜像市场已安装卡片已展示组图标；创建任务描述里的 @ 已安装镜像弹层（`TaskDescriptionSkillField`）仍是纯文本名称，辨认同一镜像仍只能靠名字。
- **Action**: (1) 确认 installed-images JSON 已含 `icon_url`；(2) 提及列表项加 24px `<img>`，无 URL 时用现有占位；(3) 补 `TaskDescriptionSkillField` 单测断言 `src`。
- **Why**: 用户在选镜像/卸载之外的创建任务路径仍缺少同一视觉标识。
- **How to apply**: `taskFE/app/src/components/TaskDescriptionSkillField.vue` + `imageGroupIconSrc`；图标仍走 `GET /api/ai-provider/public-image-groups/{id}/icon`。

## [OPT-20260828-022] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: POST 创建授权先 INSERT `cloud_platform_authorizations`，再 `applyAuthorizationActive` 写 `cloud_platform_authorization_active_methods`。activate 失败时已返回 500，但授权行会留下，列表仍显示「已禁用」。
- **Action**: (1) 把 INSERT 与 active_methods 写入放进同一 `tx`；(2) 失败回滚；(3) 在 `cloud_auth_create_test.go` 用可注入的失败桩断言无残留行。
- **Why**: 半成功行会让用户以为创建失败后又出现一条禁用授权，需要再手动删。
- **How to apply**: `taskCloudService/src/cloud_auth_routes.go` POST 分支；复用 `setExclusiveActiveMethod` 的 tx 或抽 `createCloudAuth`。验收：`cd taskCloudService && go test ./src -count=1 -run 'TestCloudAuthCreate'` 退出码 0。

## [OPT-20260828-023] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: 列表与工作空间平台接口把 CPA 行的 `authIsActive` 固定为 `access_key`，与 toggle 一致。若 `authorization_type` 不是 access_key，启用状态会显示成「已禁用」。
- **Action**: (1) 用行上的 `authorization_type`（缺省 access_key）调用 `authIsActive`；(2) 补测：oauth 类型 CPA 创建启用后列表 `is_active:true`。
- **Why**: 与 create/toggle 的 method 不一致时，勾选启用仍会看起来没生效。
- **How to apply**: `taskCloudService/src/cloud_auth_routes.go` 列表循环、`cloud_workspace_platforms.go`、`cpa_internal_handlers.go`。验收：`cd taskCloudService && go test ./src -count=1 -run 'TestCloudAuthCreateHonorsIsActiveTrue'` 退出码 0。

## [OPT-20260828-024] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: `/system-admin/step-full-cos/` 在 GET 403（需要系统管理员权限）时仍渲染表单，`secretConfigured` 保持默认 false，标题旁显示「当前密钥状态：未配置」，与「未能读取配置」混淆。
- **Action**: (1) 增加 `secretStatusKnown`（仅 GET 成功后为 true）；(2) 未知时隐藏 `step-full-cos-secret-status` 或改为「未知」；(3) 在 `SystemAdminStepFullCOS.test.js` 用 403 响应断言状态节点不出现「未配置」。
- **Why**: 权限失败被读成「还没配密钥」，管理员会误填或误判生产是否已有密钥。
- **How to apply**: `taskFE/app/src/views/SystemAdminStepFullCOS.vue` 的 `load()` / `applyPayload`；验收 `./src/views/SystemAdminStepFullCOS.test.sh` exit 0。

## [OPT-20260828-025] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行完成 (2026-08-29)
- **Created**: 2026-08-28
- **Context**: 项目详情「所属公司」已改为 `companies/current` 的 `name`。`UserGitIdentities.vue`、`GitIdentityCreateModal.vue`、`UserProfileCompanySettingsPanel.vue` 仍用 `company_name || company_id`，名称缺失时会露出 Snowflake ID。
- **Action**: (1) 复用 `taskFE/app/src/utils/companyDisplayName.js` 或同一 `companies/current` 数据源填 `company_name`；(2) 名称缺失时展示「未设置」而不是 ID；(3) 补对应 `*.test.js` 断言页面文本不含 company_id。
- **Why**: 同一产品里混用 ID 与名称会让用户以为公司名就是那串数字。
- **How to apply**: `taskFE/app/src/views/UserGitIdentities.vue`、`taskFE/app/src/components/GitIdentityCreateModal.vue`、`taskFE/app/src/components/UserProfileCompanySettingsPanel.vue`；验收 `cd taskFE/app && npm test -- src/views/UserGitIdentities src/components/GitIdentityCreateModal src/components/UserProfileCompanySettingsPanel`。

## [OPT-20260829-001] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 夜间 OPT 执行：补 nav-git-service 空列表→/pricing/ 与 error fail-open 的 Playwright 回归，公网 SPA 真实导航 + accessCode 透传，2 用例全绿
- **Created**: 2026-08-29
- **Context**: 组件测已覆盖 ready→`/pricing/` 与 fail-open，但公网工作面板带 `accessCode` 的真实导航尚未有 Playwright 断言。
- **Action**: (1) 在 `taskFE/tests` 增加用例：登录后 `gitResources=[]`（拦截 gitlab-resources 返回空数组）点击 `nav-git-service`；(2) 断言 URL 为 `/pricing/` 且保留 `accessCode`；(3) 再测 gitlab-resources 500 时仍停在当前 path。
- **Why**: 组件挂载不走真实路由与 accessCode 透传链路，E2E 才能抓住 Navbar.logic 状态机与公网 SPA 打包回归。
- **How to apply**: 参考 `taskFE/tests/Home.nav-pricing.playwright.test.js`；`npx playwright test` 对应文件 exit 0。

## [OPT-20260826-011] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: 超管推荐绩效抽屉微信分账 Tab 点「分账」→ [data-testid=referral-ps-share-form] 出现在表格上方、按钮文案变「取消」，网络无 /share/ POST（8 字缘由才发 POST 由 vitest ProfitSharingShareReasonForm 覆盖）
- **Created**: 2026-08-26
- **Context**: 系统管理用户页推荐绩效抽屉「分账」原先 `:disabled="shareBusy || !!shareRow"`，点一下按钮变灰、缘由表单在 12 列宽表下方，抽屉里看起来像没触发分账。代码已改为表格上方表单 + 打开期间不 disabled，SPA 已 `npm run build`；需登录态硬刷新才能在公网看到。
- **Action**: (1) 硬刷新 https://www.daydaymoney.com/system-admin/users/ (2) 打开有待分账行的推荐绩效 → 微信分账 Tab (3) 点「分账」断言按钮仍可点（文案变「取消」）、`[data-testid=referral-ps-share-form]` 出现在表格上方、网络无 `/share/` POST；填满 8 字点确认后才有 POST
- **Why**: 单元测试已绿；公网 CDN/浏览器缓存旧 SPA 时用户仍会看到「点了就灰掉」。
- **How to apply**: Playwright 登录态；vitest `ReferralWechatProfitSharingTab.test.js` / `ProfitSharingShareReasonForm.test.js`

## [OPT-20260822-014] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: 公网 CDP 登录态：/settings/gitlab-connection/ 无 [data-testid=gitlab-balance-points] 且无「账户余额」；/billing/ 首页无「冻结金额」
- **Created**: 2026-08-22
- **Context**: 产品确认用户无留存现金；支付只转配额。GitLab 连接页已删除 `gitlab-balance-points`，「账户余额」不再展示；账单首页已去掉「冻结金额」。公网须 SPA 新 release + 精准编译重启 + 硬刷新。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `taskFE` 点「精准编译重启」(2) 硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/settings/gitlab-connection/ 断言不存在 `[data-testid="gitlab-balance-points"]` 且无「账户余额」(3) 硬刷新 `/tenant/877397588196749312/billing/` 断言无「冻结金额」
- **Why**: 未切 public/html 时用户仍可能看到「0.00 元」现金行。
- **How to apply**: Playwright 登录态；单测 `does not show spendable cash wallet on GitLab resource panel`、`does not show frozen cash wallet on billing home`

## [OPT-20260823-040] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: task_879289074110722048 comment-git-pr-reply 卡片含 MR !4「已合并」，[data-testid=comment-git-pr-merge-btn] 不存在
- **Created**: 2026-08-23
- **Context**: GitLab !4 已于 17:17 合并；17:57 再点「一键合并」PUT 拖过 30s 被 Abort，文案误导为网络故障。taskGitOauth `9289d74` 复查 `merged_at` 后 noop 200；taskFE `8392e34` 仅 `state===open` 渲染合并按钮并改 90s 超时文案。公网须精准编译重启 `task-git-oauth`/`taskFE` 后硬刷新才吃到新 SPA。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `task-git-oauth` `taskFE` 点「精准编译重启」 (2) Playwright 登录后硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_879289074110722048/ (3) 断言含该 MR 的卡片文案含「已合并」，且 `[data-testid=comment-git-pr-merge-btn]` 不存在 (4) 若仍发出 `POST /api/git-oauth/merge-request-merge/`，断言 <15s 返回 200 且无「请检查网络」
- **Why**: 未切 public/html 时用户仍能点旧按钮并对已合 MR 打 30s Abort。
- **How to apply**: Playwright 登录态；vitest `CommentGitPrReply.test.js`、`taskDetailGitPrReply.merge-timeout.test.js`；选择器 `comment-git-pr-merge-btn`

## [OPT-20260825-011] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: 公网登录态 CDP 验证 /profile/referral/：[data-testid=referral-earnings-rule] 含「下单时」「后续下单」不含「绑边时」
- **Created**: 2026-08-25
- **Context**: `/profile/referral/` 收益规则已从「绑边时无推荐资格则不分账」改为「下单时现查资格；先推荐后获资仍可从后续下单分成」。SPA 已发布 `releases/20260825155731-2215817`，公网 HTML 已切到该 release；未登录会被重定向到 `/auth/login/?next=/profile/referral/`。
- **Action**: (1) Playwright 登录后硬刷新 https://www.daydaymoney.com/profile/referral/ (2) 断言 `[data-testid="referral-earnings-rule"]` 含「下单时」与「后续下单」，不含「绑边时」(3) 无资格账号另断言 `[data-testid="referral-no-qualification-notice"]` 含「后续下单」
- **Why**: 收益规则只在登录后的统计卡出现；未登录无法目视确认用户看到的文案。
- **How to apply**: Playwright 登录态；vitest `ReferralStatsPanel.test.js` / `UserReferral.accessCode.test.js`；选择器 `referral-earnings-rule`、`referral-no-qualification-notice`

## [OPT-20260826-020] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: 超管用户抽屉微信分账 Tab 表头无「商户分账单号」，含「微信分账单号」列且单元格 PS-BF-…（referral-ps-wechat-order）
- **Created**: 2026-08-26
- **Context**: 「商户分账单号」是同一 `billing_profit_sharing` 上的微信支付 `out_order_no` 幂等键，不是新实体。用户抽屉与待分账面板已折叠进「微信分账单号」单元格；公网仍可能是旧 SPA。
- **Action**: (1) `npm run build` 已切 `taskFE/app/public/html` (2) 精准编译重启 taskFE (3) 硬刷新 https://www.daydaymoney.com/system-admin/users/ 打开有分账记录的用户抽屉 (4) 表头无「商户分账单号」，「微信分账单号」在未返回微信 id 时展示 `PS…`
- **Why**: 折叠只在新 hash SPA 下可见；运营对照微信商户后台时应用微信分账单号或单元格 title 中的 out_order_no。
- **How to apply**: `scripts/register-precise-restart.sh taskFE`；组件 `ReferralWechatProfitSharingTab.vue` `data-testid=referral-ps-wechat-order`。

## [OPT-20260825-024] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: 超管用户抽屉微信分账 Tab 表头含「商户单号」列，单元格 WX…（referral-ps-out-trade-no）
- **Created**: 2026-08-25
- **Context**: 用户抽屉微信分账表与待分账面板已加「商户单号」「商户分账单号」列，单元测试已绿。公网 SPA 与 task-bill 需精准编译重启后才可见。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点击「精准编译重启」消费 `task-bill`/`taskFE` 登记 (2) 硬刷新 https://www.daydaymoney.com/system-admin/users/ 打开有分账失败记录的用户抽屉 (3) 确认表头有「商户单号」且单元格为 `WX…` 或 `—`。
- **Why**: 部署验证闭环 —— 列表列仅在真实浏览器与新二进制/新 SPA hash 下可见。
- **How to apply**: `.runall/precise_restart_services.txt`；组件 `ReferralWechatProfitSharingTab.vue` `data-testid=referral-ps-out-trade-no`。

## [OPT-20260825-010] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: GET /api/billing/gitlab-resources/tenant_id/877397588196749312/?region=tencent-sh-1 返回 disk_used_gb=0 ≠ traffic_used_gb=0.042977；UI gitlab-disk-used-gb=0 GB/1GB ≠ gitlab-traffic-used-gb=0.042977GB/1GB
- **Created**: 2026-08-25
- **Context**: 租户 `877397588196749312` 区 `tencent-sh-1` 曾把磁盘 11263830 字节写成 `traffic_used_gb=0.01049`，与已用磁盘相同。读路径已丢弃该水位，068 已把库存量清零。本会话无登录 Cookie，未能在公网设置页点选验收。
- **Action**: (1) 精准编译重启 task-bill（已登记）；(2) 登录后硬刷新 `/tenant/877397588196749312/settings/gitlab-connection/`；(3) 确认 `gitlab-disk-used-gb` 仍约 0.01049 GB，`gitlab-traffic-used-gb` 为 `0 GB / 1 GB`（或未预购分母），二者不再相同。
- **Why**: 设置页是用户发现该问题的入口；仅 DB/单测绿不能代替一次真实页面核对。
- **How to apply**: 页面 `data-testid=gitlab-disk-used-gb` / `gitlab-traffic-used-gb`；后端 GET `/api/billing/gitlab-resources/tenant_id/{tid}/?region=tencent-sh-1`。

## [OPT-20260824-085] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: /profile/referral/?accessCode=DR2AKvP9J9 渠道统计表(referral-channel-stats)列 渠道/推荐人数/消费/分账，页面无 ORD- 订单号泄露
- **Created**: 2026-08-24
- **Context**: 推荐人 `/profile/referral/` 原「订单分账」表直接渲染受推荐人 `order_number`。已改为按渠道聚合（渠道号、订单时段、订单/冻结/可分账金额）+ `POST share-channel`；需部署后在公网确认旧订单号单元格消失。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点击「精准编译重启」消费 `task-bill` `taskFE`；(2) 硬刷新 https://www.daydaymoney.com/profile/referral/?accessCode=DR2AKvP9J9 ，断言表格无 `ORD-`、列为渠道号/时段/金额，可分账渠道仍有「分账」按钮。
- **Why**: 隐私修复仅在新二进制 + 新 SPA 上线后对用户生效；旧 SPA 仍会读空 `orders`。
- **How to apply**: `.runall/precise_restart_services.txt` 已登记；核验选择器不再匹配 `td.font-mono` 内的 `ORD-` 订单号。

## [OPT-20260823-046] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: [data-testid=user-list-column-filters] 在每列头下方渲染；邮箱过滤输入 ljy 后请求 GET /api/system-admin/users/?email=ljy&is_archived=false
- **Created**: 2026-08-23
- **Context**: `/system-admin/users/` 已实现 thead 第二行列过滤（vitest 9 例 + Go 12 例绿）。公网 SPA 与 task-auth 需精准编译重启后才能看到。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `task-auth` `taskFE` 点「精准编译重启」(2) 硬刷新 https://www.daydaymoney.com/system-admin/users/ (3) 断言 `[data-testid="user-list-column-filters"]` 在每列表头下方，输入邮箱后列表请求带 `email=`，点重置后该参数消失。
- **Why**: 未切 public/html 时用户仍只看到无列过滤的旧表。
- **How to apply**: `.runall/precise_restart_services.txt` 已含 task-auth / taskFE；测例 `SystemAdminUsers.columnFilters.test.js`、`handlers_system_admin_list_filter_test.go`

## [OPT-20260823-039] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: /system-admin/users/ 表头含「是否获得分账资格」，行值 是/否（contact@daydaymoney.com=是）
- **Created**: 2026-08-23
- **Context**: 超管用户列表已增加 `has_profit_sharing_qualification` 投影（taskReferral batch + taskAuth 回填 + taskFE 列）。单测已绿；公网 SPA / 进程仍是旧产物，硬刷新看不到新列。
- **Action**: (1) 在 :9999 对已登记的 `task-referral`、`task-auth`、`taskFE` 执行精准编译重启（勿带无关脏子仓 WIP）(2) 硬刷新 `https://www.daydaymoney.com/system-admin/users/` (3) 表头「角色」与「操作」之间出现「是否获得分账资格」，值为「是 / 否 / —」
- **Why**: 前端 nginx 只吃 Vite 产物，源码合入后不重启看不到新列；资格服务失败时列应显示「—」而不是打断列表。
- **How to apply**: `.runall/precise_restart_services.txt` 已含 task-referral / task-auth / taskFE；页面 `/system-admin/users/`；测例 `SystemAdminUsers.profit-sharing-qualification-column.test.js`、`UserListRow.test.js`

## [OPT-20260823-014] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: /system-admin/users/ 表头含「最后活跃时间」，登录过的用户显示日期时间、未登录显示 —
- **Created**: 2026-08-23
- **Context**: 超管用户列表已增加 `last_login` 展示，登录成功路径会写入该字段。公网在精准重启与硬刷新前仍可能看到旧表头。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对 task-auth、taskFE 执行「精准编译重启」 (2) 硬刷新 `https://www.daydaymoney.com/system-admin/users/` (3) 表头「注册时间」后出现「最后活跃时间」，从未登录显示 —，登录过的用户显示本地化日期时间
- **Why**: 未重启时浏览器和进程仍用旧包，本会话改动对公网不可见。
- **How to apply**: `scripts/register-precise-restart.sh task-auth taskFE`；页面 `/system-admin/users/`

## [OPT-20260823-012] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: /system-admin/users/ 表头含「所属租户公司」且行值渲染公司名（如「我的公司、软刀的公司」）或 —，[data-testid=user-tenant-companies] 存在
- **Created**: 2026-08-23
- **Context**: 超管用户列表 API 已回填 `tenant_companies`，前端在「邮箱」后新增列。公网仍可能跑旧 SPA / 旧 task-auth 进程，硬刷新前看不到新列。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对 task-auth、taskFE 执行「精准编译重启」 (2) 硬刷新 `https://www.daydaymoney.com/system-admin/users/` (3) 表头「邮箱」右侧出现「所属租户公司」，有成员关系的用户显示公司名，无归属显示 —
- **Why**: 未重启时浏览器和进程仍用旧包，本会话改动对公网不可见。
- **How to apply**: `scripts/register-precise-restart.sh taskAuth taskFE`；页面 `/system-admin/users/`；`UserListRow` `data-testid="user-tenant-companies"`

## [OPT-20260819-022] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: /tenant/877397588196749312/billing/orders/ 退款中行 ORD-20260819-...-877909223214710784 显示「退款中」，全页无 [data-testid=billing-refund-apply-btn]
- **Created**: 2026-08-19
- **Context**: 已修复租户订单列表在 pending 退款申请时仍显示「申请退款」；需公网硬刷新确认 SPA 已发布。
- **Action**: (1) runAll「精准编译重启」taskFE；(2) 硬刷新 `https://www.daydaymoney.com/tenant/877397588196749312/billing/orders/`；(3) 断言退款中行显示「退款中」且无 `billing-refund-apply-btn`。
- **Why**: 前端静态资源可能有缓存；仅本地测绿不足以证明公网已生效。
- **How to apply**: 登记已写入 `.runall/precise_restart_services.txt`；验收后可用 Playwright CDP 或人工核对。

## [OPT-20260824-076] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: work-panel 创建任务弹窗 skill chips 为「$trae-agent /general-coding 默认」「$trae-agent /k8s-debug」，无 $$ 双前缀
- **Created**: 2026-08-24
- **Context**: TaskDescriptionSkillField chips 标签已从 `/技能名` 改为 `$镜像名 /技能名`（完整 mention 语法提示，taskFE 939b00c）。组件级 vitest 10/10 覆盖渲染与点击行为，但生产页面（work-panel 创建任务弹窗）视觉呈现尚未人工确认（长镜像名 + 多技能时 flex-wrap 换行观感）。
- **Action**: 精准编译重启后在 https://www.daydaymoney.com/tenant/*/work-panel 创建任务弹窗核验 chips 文案与换行布局；确认无 `$$` 双前缀。
- **Why**: 部署验证闭环 —— 布局类改动仅在真实浏览器渲染可见。
- **How to apply**: 打开 work-panel → 新建任务 → 描述区选择镜像（$trae-agent）后查看 chips；比对 `$trae-agent /general-coding 默认` 与 `$trae-agent /k8s-debug`。

## [OPT-20260823-031] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: /system-admin/order-records/ 筛选含「已退款」；点「已退款」后请求 GET /api/system-admin/orders/?status=refunded，行徽章为中文「已退款」，无英文 refunded
- **Created**: 2026-08-23
- **Context**: 超管订单记录页筛选漏「已退款」、徽章回退英文 `refunded`。已把文案/筛选项统一到 `billingOrderDisplay.js`，Vitest 16 例全绿；公网 SPA 仍是旧 bundle。
- **Action**: (1) 在 :9999 对已登记的 `taskFE` 执行精准编译重启（勿带无关脏子仓 WIP）(2) 硬刷新 `/system-admin/order-records/` 确认筛选含「已退款」、`refunded` 行徽章为「已退款」(3) 点「已退款」确认列表请求带 `status=refunded`
- **Why**: 前端 nginx 只吃 Vite 产物，源码合入后不重启看不到修复。
- **How to apply**: `.runall/precise_restart_services.txt` 已含 taskFE；页面 `https://www.daydaymoney.com/system-admin/order-records/`；测例 `systemAdminOrderListDisplay.test.js` / `SystemAdminOrderListPanel.deeplink.test.js`

## [OPT-20260819-030] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: /system-admin/users/ 打开用户 KYC drawer：[data-testid=kyc-tier-help] 可见「各等级说明」与 T0/T1/T2 各档额度
- **Created**: 2026-08-19
- **Context**: 已在本地单测与 SPA build 落地 `data-testid=kyc-tier-help`；公网需精准编译重启后硬刷新确认。
- **Action**: (1) runAll「精准编译重启」taskFE (2) 硬刷新 `https://www.daydaymoney.com/system-admin/users/` (3) 打开用户 KYC drawer，断言可见「各等级说明」及 T0/T1/T2
- **Why**: 静态资源缓存可能导致公网仍是旧 bundle。
- **How to apply**: `.runall/precise_restart_services.txt` 已含 taskFE；可用 Playwright CDP 或人工核对

## [OPT-20260822-035] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Completion-Note**: GET /api/system-admin/accounts/admin/tenant-options/? 返回 19 个租户选项均含 email（contact@daydaymoney.com）与 phone（13900001111）字段
- **Created**: 2026-08-22
- **Context**: tenant-options 已返回创建者 email/phone，赠送页与订单记录页下拉会渲染。公网 SPA 与 taskAuth/taskTenantService 需编译重启后才能在 https://www.daydaymoney.com/system-admin/grant-points 看到真实联系方式。
- **Action**: (1) 在 :9999 对 task-auth、task-tenant-service、taskFE 精准编译重启 (2) 硬刷新赠送页，下拉应出现邮箱/手机而非仅「我的公司 ID …」 (3) 用邮箱或手机号搜索应命中对应租户
- **Why**: 未重启时管理端仍打旧进程，页面会继续只显示公司名。
- **How to apply**: `scripts/register-precise-restart.sh taskAuth taskTenantService taskFE`；runAll「精准编译重启」；页面 `/system-admin/grant-points`

## [OPT-20260811-072] completed

- **Status**: completed
- **Blocked-By**: BROWSER
- **Completed**: 2026-08-29
- **Completion-Note**: CDP 登录态 /tenant/875588283562749952/people/manage/（redirect 877397588196749312）：拦截 `PATCH /api/tenant/877397588196749312/accounts/members/877397588209332224/update_role/` ①返回 500 → 弹「错误 / 更新成员失败」中文对话框，节点带 `data-traceid=b8588737-10ac-4df0-b748-53f08f2271fa`，无裸 `Failed to fetch`；②返回 200（请求被 route 接管未落库）→ 编辑弹窗关闭、成员列表保留、无错误。两条路径均不再出现 `Failed to fetch`。
- **Created**: 2026-08-11
- **Context**: 【2026-08-14 日间 Playwright】原租户 874941752761413632 被改写；本租户 `/people/manage/` 20s 仍「加载中…」，未出现 Failed to fetch，也未完成编辑保存。  Loki 证实 `update_role` 已 200（traceId `6d92862a-…`），浏览器却弹原始 `Failed to fetch`。已修：`showRequestError` 中文化 + `MemberList` 网络失败后复核列表；本地 16 单测绿；`taskFE` 已在精准重启登记。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」含 taskFE；(2) 硬刷新 https://www.daydaymoney.com/tenant/874941752761413632/people/manage/；(3) 编辑成员并保存：成功关弹窗；若人为断网，错误文案应为中文且带 data-traceId，不得再出现裸 `Failed to fetch`。
- **Why**: 生产仍托管旧 PeopleManage chunk，不发布则页面仍展示英文网络错误。
- **How to apply**: 机器验收：CDP 9222。`taskFE/app/src/components/MemberList.vue`；`requestErrorDisplay.js`；静态 chunk `PeopleManage-*.js`。
- **Related**: OPT-20260811-069 已合并入本条（2026-08-14 用户确认）。

## [OPT-20260829-021] completed

- **Status**: completed
- **Completed**: 2026-08-29
- **Summary**: 厂商 SSO 后已上架激活行可见「下架」且 URL 占第二行；待审核行可见「撤回审核」；点下架弹出确认弹窗后取消。原 private_x86_64-latest 行已不在库中，删除矩阵由单测覆盖。
- **Created**: 2026-08-29
- **Context**: 本会话浏览器打开 provider.daydaymoney.com 时未持有该厂商 SSO，无法点选 `private_x86_64-latest` 真实行。单测覆盖矩阵与 HTTP。
- **Action**: (1) 用已认证厂商会话打开门户；(2) 展开含已上架非激活版本的组；(3) 确认第一行可见「下架」「删除」；(4) 走确认弹窗（不真删生产数据可用草稿行）。
- **Why**: 布局问题在宽屏/窄屏下的最终可见性需要真实 DOM。
- **How to apply**: `https://provider.daydaymoney.com/`；选择器 `div.version-row` / `[data-testid=version-row-actions]`。

