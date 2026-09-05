# Blocked TODOs — BROWSER

> 本文件存放因 **浏览器验收（公网硬刷新 / 登录 Cookie / CDP；优先 Playwright 闭环）** 阻塞而从开放清单分流的 OPT 条目。
> 阻塞解除后移回 [OPTIMIZATION_TODOS.md](./OPTIMIZATION_TODOS.md) 执行，或完成后迁入 [OPTIMIZATION_TODOS_COMPLETED.md](./OPTIMIZATION_TODOS_COMPLETED.md)。
> 分流规则见 [OPTIMIZATION_TODOS.ai.md](./OPTIMIZATION_TODOS.ai.md)「阻塞项分流」。
> **解除方式**：优先用 Playwright（CDP 9222 / 登录 helper）验收，见 [OPTIMIZATION_TODOS.ai.md](./OPTIMIZATION_TODOS.ai.md)「BROWSER 阻塞项的 Playwright 解除」。

- **Category**: `BROWSER`
- **Count**: 64

---

### OPT-20260902-007 — 精准重启后 Playwright 验收创建项目 OAuth 详情页为已授权

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-09-02
- **Context**: 创建项目时 pending `grant_ticket` 未写成 `project_git_oauth_grant`，详情「云端开发」Git 行显示「需要授权」。已修创建 POST 消费 ticket。存量项目 `proj_882824768007467008` 无 L2，须在详情点一次 OAuth，或新建项目走完整授权回流再打开详情。
- **Action**: (1) http://10.2.150.68:9999/ 对 `taskFE`、`task-project` 精准编译重启；(2) Playwright CDP 9222 登录后创建含 GitLab 仓的项目并完成 OAuth；(3) 打开项目详情断言 `[data-testid=git-repo-oauth-status-0]` 为「已授权」，且不再出现「OAuth 授权」按钮。
- **Why**: Vitest 不能覆盖 APISIX Cookie、真实 GitLab OAuth 回流与公网 SPA hash。
- **How to apply**: 对照 `CreateProject.submitForm.test.js` T2 与 `git_oauth_grant_ticket_test.go`；公网 URL 形态 `/tenant/{tid}/projects/{pid}/`。

### OPT-20260901-010 — 精准重启后 Playwright 复验 profile 邮箱验证码真实投递

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-09-01
- **Context**: Loki 证实 `send_verification_code` HTTP 200 但 email-sent 消费者 QQ SMTP 535。已修 conf-local overlay + OTP 同步 SMTP。须重启 `task-auth` 与 `task-events-email-sent-1-send-email` 后用登录态走绑定面板，断言成功文案仅在 SMTP 成功时出现，失败带 `data-traceId`。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」已登记服务；(2) Playwright CDP 9222 打开 `/profile/?sso_error=email_required#rg=profile.email_binding` 发验证码；(3) 断言：成功则 `p.text-sm.text-text-light` 含「验证码已发送」且 Loki `{job="task-events-email-sent-1-send-email"} |= "delivered"` 或 task-auth `OTP delivered via SMTP` 同 trace；失败则 `p.text-danger` 含「邮件发送失败」且 `data-traceId` 非空、不得出现「验证码已发送」。
- **Why**: 单测不能覆盖 live QQ SMTP 与公网 Cookie。
- **How to apply**: Playwright 登录 helper + Loki query `{job="task-auth"} |= "send_verification_code"` 最近 10 分钟；taskFE `npx vitest run src/components/UserProfileEmailBindingPanel.test.js` 已绿。

### OPT-20260830-006 — 二次 SPA 构建后以 extTest 验收关联项目面板（unwrap GET）

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-30
- **Context**: 首轮 merge 保护已上线仍空态。DB/`taskTaskService:8017` GET 仍返回 helloworld `projects`。剩余假设是详情 GET 被当成工作区列表赋给 `localTask`（数组无 `.projects`）或 `taskProjectsWithDetails` 未传到 ViewMode。已 unwrap 列表/信封、面板从 `task.projects` 回退、忽略 `props.task === null` 的 merge 清空。本会话浏览器是「软刀」，打开目标 URL 会 403/重定向。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `taskFE` 点「精准编译重启」；(2) 硬刷新后确认 `index.html` `build-time` ≥ `2026-08-30T05:21:26Z`；(3) 以 `881000534834704384`（extTest）打开 https://www.daydaymoney.com/tenant/881024523581812736/workspace/ws_881024527847419904/task-detail/task_881388002226499584/ ；(4) 断言 `[data-testid=task-linked-projects-panel]` 不含「暂无关联项目」，含 `helloworld` 或 `github.com` URL，且有 `[data-testid=task-repo-address-mismatch-badge]`。
- **Why**: Vitest 不能覆盖 APISIX 前缀与生产 cookie；软刀会话无法打开该租户任务页。
- **How to apply**: Playwright 登录 helper；本机已绿：`npx vitest run src/utils/unwrapTaskDetailPayload.test.js src/composables/taskDetail/taskDetailFetchFns.unwrap.test.js src/components/task-detail/TaskDetailLinkedProjectsViewMode.test.js`（taskFE/app）。

### OPT-20260830-005 — 精准重启后以任务所有者会话验收关联项目不再空态

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-30
- **Context**: `task_881388002226499584` 在 `task_projects` 已有 helloworld（`proj_881195029417193472`），内部 GET 返回 `projects` 且 `repo_address_mismatch=true`。插件快照却显示「暂无关联项目」。根因是详情页 `mergeTaskDetailUpdate` 被 `projects:[]` 部分更新擦掉。taskFE 已修并登记精准编译重启。本会话浏览器登录为「软刀」会被重定向到其它租户，无法打开该 URL。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `taskFE` 点「精准编译重启」；(2) 以租户 `881024523581812736` 成员（如 extTest / user `881000534834704384`）打开 https://www.daydaymoney.com/tenant/881024523581812736/workspace/ws_881024527847419904/task-detail/task_881388002226499584/ ；(3) 断言 `[data-testid=task-linked-projects-panel]` 不含「暂无关联项目」，含项目名或仓库 URL，且可见 `[data-testid=task-repo-address-mismatch-badge]`。
- **Why**: 公网 SPA 未重建前用户仍会看到空态；本机 Vitest 不能覆盖 APISIX + 生产 hash。
- **How to apply**: Playwright 登录态；本机已绿：`npx vitest run src/composables/taskDetail/mergeTaskDetailUpdate.test.js src/utils/taskProjectsWithDetails.test.js src/components/task-detail/TaskDetailLinkedProjectsViewMode.test.js`（taskFE/app）。

### OPT-20260825-034 — 精准重启后公网验收系统管理用户页分账不再展示 PARAM_ERROR dump

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-25
- **Context**: trace `4d2d9751-00cd-4ee1-9122-2f85de66044a` 显示超管对 profit_sharing_id=3 发起分账时微信拒空 `receivers[0].account`。代码已 fail-fast 并映射中文，但公网仍跑旧二进制/旧 SPA。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `task-bill` `taskFE` 点「精准编译重启」(2) 硬刷新 https://www.daydaymoney.com/system-admin/users/ 打开对应用户微信分账 Tab (3) 对未绑定微信的行点分账，断言 `[data-testid="referral-ps-error"]` 文案含「未绑定微信收款账号」且不含 `PARAM_ERROR`/`HTTP 400`
- **Why**: 未重启前用户仍会看到微信 PARAM_ERROR JSON dump。
- **How to apply**: Playwright 登录态；Loki `{job="task-bill"} |= "profit_sharing_openid_missing"`

### OPT-20260822-052 — 公网硬刷新确认订单详情支付成功横幅有退款按钮

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: 已支付订单详情页已加「申请退款」与「已消耗的资源无法退回」说明。需 SPA 新 release + 精准编译重启 + 硬刷新后才能在公网看到。 【2026-08-29 夜间二次探针】登录态打开订单 878981177317294080（tenant 878619773850644480）返回「订单不存在」，订单已清库，无法验收。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `taskFE` `task-bill` 点「精准编译重启」(2) 硬刷新 https://www.daydaymoney.com/tenant/878619773850644480/billing/orders/878981177317294080/ (3) 断言 `[data-testid="billing-refund-apply-btn"]` 可见且文案含「已消耗的资源无法退回」
- **Why**: 未切 public/html 时用户仍只看到旧的支付成功横幅。
- **How to apply**: Playwright 登录态；vitest `OrderDetail.refund.test.js`

### OPT-20260822-022 — 公网复验 @trae-agent 评论在 auto_run=否 时仍自动执行

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: 案例 116：任务 `task_878865840278106112` 评论 `@trae-agent 删除 用 java 写的 hello world` 只完成 bootstrap（TraceId `be794568-25eb-4646-9b53-1d912f50b2db`）。task-task-service 已默认 AIComment 8019 且先 notify；须重启后再发**新**评论验收。存量该条评论不会自动补跑。
- **Action**: (1) http://10.2.150.68:9999/ 对已登记 `task-task-service` 点「精准编译重启」(2) 在 https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878865840278106112/?accessCode=DR2AKvP9J9 发新评论 `@trae-agent` + 一句可观察指令（3) 断言执行细节出现 Agent job（不仅 bootstrap），Loki 有 `at_mention_notify_agent_ok` 且无单独的 `AUTO_RUN_FIRST_SKIP reason=auto_run_false`
- **Why**: 代码已合入；公网进程未重启前用户仍会遇到「评论已发、容器已起、指令不跑」。
- **How to apply**: Playwright 登录态；Loki `{app="task-task-service"} |= "at_mention_notify_agent"`；案例 `.ai/09_failure_experience/02_runtime_errors/116_at_mention_skip_empty_aicomment_url.md`

### OPT-20260822-011 — 公网硬刷新确认 ztree 命令失败不再显示裸 HTTP 403

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: 任务详情执行细节命令面板曾显示红色「HTTP 403」（案例 113）。前端已解析 `message/error/detail` 并映射友好文案、挂 `data-traceId`。公网需 SPA 构建 + 精准重启 + 硬刷新后才能验收。 【2026-08-29 夜间探针】登录态打开 task_878583341551480832 加载正常，页内无 .text-red-600 错误节点、无「HTTP 403」；4 条评论均为「服务器已释放」，无失败中的 ztree 命令可展开，失败路径未实际触发（弱验收）。 【2026-08-29 夜间三次探针】CDP 登录态再次硬刷新：加载正常、无裸「HTTP 403」、0 个 .text-red-600 错误节点，仍无失败命令可展开（弱验收）。
- **Action**: (1) 对已登记 `taskFE` `task-container-gateway` 点「精准编译重启」(2) 硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878583341551480832/?accessCode=DR2AKvP9J9 (3) 展开第 4 条评论执行细节，命令失败时断言不再出现恰好「HTTP 403」的红字，且错误 `<p class="text-red-600">` 带 `data-traceId`
- **Why**: 未切 public/html symlink 时用户仍看到旧 SPA 的裸 HTTP 403。
- **How to apply**: Playwright 登录态；选择器 `#comments-container p.mt-2.text-xs.text-red-600`；案例 `.ai/09_failure_experience/02_runtime_errors/113_ztree_layer_command_http_403.md`

### OPT-20260821-029 — 公网复验「提交并创建 PR」不再因 internal_gateway 哨兵报 Git 身份不存在

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-21
- **Context**: 任务 `task_878583341551480832` 点「提交并创建 PR」曾 404「Git 身份不存在或不属于当前租户」（trace `39e84a49-8629-4418-9ea4-a79827d7654e`）。Gateway 哨兵 user_id 已修（案例 105）。随后 prepare 曾 409，因 Cloud 打错 gitOauth summary 路径而跳过换票（案例 106）。2026-08-21 本地 prepare 已 200（`use_oauth_access_push=true`）。公网完整「提交并创建 PR」仍需容器在线。
- **Action**: (1) 确认任务页容器非「已释放」（必要时点「启动」等业务端点就绪）(2) 硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878583341551480832/?accessCode=DR2AKvP9J9 (3) 点「提交并创建 PR」，断言不再出现「Git 身份不存在或不属于当前租户」；成功则出现 PR 链接或推送成功提示
- **Why**: 身份+OAuth prepare 已在本机 8018 验证；公网点按钮还依赖运行中的任务容器执行 git push。
- **How to apply**: Playwright 登录态；选择器 `p.taskplugin-el-highlight`；Loki `{job=~".+"}` 按新 trace 查 `cloud_prepare_git_push` / `auth_internal_bypass`

### OPT-20260817-004 — 公网硬刷新确认 ztree 失焦后指令面板不消失

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-17
- **Context**: 选中任务关联 ztree 节点后，点评论区其它位置曾会 `emit(null)` 关掉指令面板/文件树/日志。已改为点外不取消选中；taskFE dist 已构建且已登记精准编译重启。公网可能仍是旧 SPA。 【2026-08-29 夜间二次探针】登录态打开 task_876895044743753728（tenant 875588283562749952）详情页重定向工作面板，任务已清库，无 ztree 节点可验。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 `taskFE` 点「精准编译重启」；(2) 硬刷新 https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876895044743753728/；(3) 展开评论「任务关联」，点击 `.layer-graph-ztree` 内一层/任务节点使 `[data-testid=comment-layer-ztree-command-panel]` 出现；(4) 点击 `#comments-container` 内面板外空白，断言指令面板仍存在且 ztree 选中高亮仍在。
- **Why**: 未重启 preview / 未硬刷新时用户仍会看到失焦即隐藏。
- **How to apply**: CDP 9222。选择器 `.layer-graph-ztree`、`[data-testid=comment-layer-ztree-command-panel]`、`#layer-graph-command-input`。

### OPT-20260816-055 — 公网硬刷新确认「容器 等待前序」可查看前序及状态

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-16
- **Context**: 任务详情「容器 等待前序」badge 原先只有文案。已在执行细节内增加 `comment-execution-predecessor-list`（摘要 + binding 状态），单击 badge 展开；taskFE dist 已构建并已登记精准编译重启。公网仍可能缓存旧 SPA。 【2026-08-29 夜间二次探针】登录态打开 task_876757038493888512 详情页重定向工作面板，任务已清库，无等待前序评论可验。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 `taskFE` 点「精准编译重启」；(2) 硬刷新 https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876757038493888512/；(3) 找到 `data-testid=comment-execution-binding-status` 文案含「等待前序」的评论，展开执行细节，断言 Tab 栏有 `comment-execution-tab-predecessors` 且默认**看不到**前序列表；单击该 Tab 或单击徽章后出现 `comment-execution-predecessor-list`，每行含 summary 与 status。
- **Why**: 未重启 preview / 未硬刷新时用户仍只能看到徽章，看不到前序列表。
- **How to apply**: CDP 9222。选择器 `data-testid=comment-execution-tab-predecessors` / `comment-execution-predecessor-list` / `comment-execution-predecessor-row`。

### OPT-20260816-043 — 公网硬刷新确认同任务第二条评论也有启动 TraceId

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-16
- **Context**: 任务 `task_876757038493888512` 第一条评论有启动 TraceId `dfc792b3b5a337f47f08adf0`，第二条 `cmt_876757333349265408` 有容器名/CSC 但无 TraceId。根因是 bootstrap 找不到任务级 start 载荷、heal 绑实例不写列。已修 sibling 回退 + list/heal 补写；须精准编译重启 `task-cloud-service` 后硬刷新。 【2026-08-29 夜间二次探针】登录态打开 task_876757038493888512 详情页重定向工作面板，任务已清库，无评论可验。
- **Action**: (1) http://10.2.150.68:9999/ 精准编译重启已登记的 `task-cloud-service`；(2) 硬刷新 https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876757038493888512/；(3) 展开两条评论执行细节，断言各有 `[data-testid=comment-execution-start-trace-id]`，值非空且 ≠ `task_876757038493888512`，且互不相同。
- **Why**: list 补写发生在重启后的进程内；未重启公网仍跑旧 Cloud，第二条评论继续缺 TraceId 行。
- **How to apply**: CDP 9222。选择器 `data-testid=comment-execution-container-meta` / `comment-execution-start-trace-id` / `comment-execution-start-trace-id-value`。

### OPT-20260815-029 — 公网硬刷新确认执行日志不再显示裸 404

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-15
- **Context**: 评论执行细节里克隆日志/任务日志曾显示裸「404」，根因是网关 mux 用 `Contains("/cloud/compute/container-")` 拒掉 kv-last 路径。已修 mux、JSON 404 与 FE 文案，taskFE dist 已构建，已登记 `task-container-gateway` 与 `taskFE` 精准编译重启。公网页需登录且可能缓存旧 SPA。 【2026-08-29 夜间二次探针】登录态打开 task_876469535748681728 详情页重定向工作面板，任务已清库。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记服务点「精准编译重启」；(2) 硬刷新 `https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876469535748681728/`；(3) 展开评论执行细节，断言克隆日志/任务日志红字不再是单独的「404」，若仍失败应含「HTTP 404」或 JSON `detail`。
- **Why**: 未重启网关/preview 或未硬刷新时用户仍会看到裸 404。
- **How to apply**: CDP 9222；选择器 `p.text-xs.text-red-600`、`[data-testid=comment-execution-start-trace-id]`。

### OPT-20260815-025 — 公网硬刷新确认评论头像紧贴昵称

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-15
- **Context**: 评论气泡头像曾因全局 `.flex.gap-3 { align-items:center }` 被垂直居中到执行细节中部。已把头像移入 `[data-testid=comment-author-row]` 并降低该 CSS 特异性；taskFE dist 已构建并登记精准编译重启。公网仍可能缓存旧 SPA。 【2026-08-29 夜间二次探针】登录态打开 task_876458313389207552 详情页重定向工作面板，任务已清库。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 `taskFE` 点「精准编译重启」；(2) 硬刷新 `https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876458313389207552/`；(3) 断言 `[data-testid=comment-author-avatar]` 与 `[data-testid=comment-author-name]` 为相邻兄弟，且气泡根节点无直接子 `img`。
- **Why**: 未重启 preview / 未硬刷新时用户仍会看到头像悬在卡片中部。
- **How to apply**: CDP 9222；选择器 `[data-testid=comment-author-row]`、`[data-testid=comment-author-avatar]`、`[data-testid=comment-author-name]`。

### OPT-20260815-018 — 公网硬刷新确认任务详情评论区顶端无镜像运行 header

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-15
- **Context**: 已从 `TaskDetailCommentsPanel` 移除 `data-testid=image-runtime-entry` 全局卡，管道 `buildImageRuntimeEntries` 已删，`taskFE` dist 已构建并登记精准编译重启。公网仍可能缓存旧 SPA。 【2026-08-29 夜间二次探针】登录态打开 task_876416048071471104 详情页重定向工作面板，任务已清库。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 `taskFE` 点「精准编译重启」；(2) 硬刷新 `https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876416048071471104/`；(3) 断言 DOM 无 `[data-testid=image-runtime-entry]`，文案无「镜像运行状态」；(4) 展开对应评论「执行细节」仍可见启动日志 / SSE / 服务器运行状态 Tab。
- **Why**: 未重启 preview / 未硬刷新时用户仍会看到旧全局 header。
- **How to apply**: CDP 9222；选择器 `div[data-testid=image-runtime-entry]`；评论执行细节 `data-testid=comment-execution-details`。

### OPT-20260814-024 — Playwright 断言同一工作空间连续建帖为 #1 #2 #3

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-14
- **Context**: workspace_seq 后端与 SPA 已上线（`index-BqrMVE7h.js` / `taskIdDisplay-BIYlzgNV.js` 公网 200）。存量任务帖已按 010 清空。需登录后在同一工作空间连续建 3 帖，卡片文案为 `#1` `#2` `#3`。 【2026-08-29 夜间探针】CDP 登录态 work-panel 已存在 #1–#72 任务卡片（workspace_seq 已用到 ≥72，010 清空后又被后续测试重建）；连续建 3 帖会得到 #73 #74 #75 而非 #1 #2 #3，按原断言验收需先清空目标工作空间任务帖（OPS）。
- **Action**: (1) Playwright 登录公网看板 (2) 同一 workspace 连续创建 3 个任务帖 (3) 断言 `TaskCardIdBadge` / `[data-testid=task-card-id]` 文本为 `#1` `#2` `#3` (4) Loki `{job=~"task-task-service.+"} |= "workspace_seq_allocated"` 对应 3 条
- **Why**: 未走浏览器则无法证明人读序号对用户可见。
- **How to apply**: CDP 9222；`taskFE/app/src/components/TaskCardIdBadge.vue`；https://www.daydaymoney.com/

### OPT-20260814-016 — 公网验收 Workbench 对真实 ECS 不再报平台 mock

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-14
- **Context**: 任务 `task_15805564155504060866` 评论 CSC 库内已是 `aliyun` + `cn-qingdao` + `cpa_-2740812859862113751` + `i-m5edps4oejpnwahrjpoa`（020 已应用，task-cloud-service 已于 2026-08-14 16:13 编译重启）。公网硬刷新后点按钮仍须浏览器验收。
- **Action**: (1) 硬刷新该任务详情并展开评论执行细节 (2) 状态条不得再出现「未找到云平台授权信息」或 Mock 文案 (3) 点「Workbench 访问实例」应新开 `ecs-workbench.aliyun.com` 且 URL 含 `instanceId=i-m5edps4oejpnwahrjpoa`，不得再出现「暂不支持平台 mock」
- **Why**: 旧 Cloud 进程不会自愈；未跑 020 则库内评论行仍是 mock，attach/ingress 仍可能跳过。
- **How to apply**: 机器验收：CDP 9222。https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_15805564155504060866/ ；按钮文案「Workbench 访问实例」。

### OPT-20260814-008 — 公网硬刷新验收启动成功日志不再带「服务器启动失败」红条

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-14
- **Context**: 任务 `task_15794155595830266865` 评论执行细节同时出现 `server-startup-error-banner`「服务器启动失败」与日志「aliyun服务器启动成功！」。根因是 failed binding 不随评论级 start-vm 成功回写；代码已修（恢复 starting + 前端按成功日志去红条），单测已绿。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」含 taskFE、task-cloud-service (2) 硬刷新该任务详情 (3) 展开对应评论执行细节 (4) 有「aliyun服务器启动成功」时不得再出现 `data-testid=server-startup-error-banner`，生命周期应为「启动中」或「已启动」。
- **Why**: 公网仍跑旧 SPA/旧 Cloud 进程时红条会继续误导。
- **How to apply**: 机器验收：CDP 9222。https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_15794155595830266865/

### OPT-20260813-021 — 公网验收同任务两评论启动 TraceId 互不相同且不等于 task_id

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-13
- **Context**: 【2026-08-14 日间 Playwright】登录后打开 task_15742467311115154867，`comment-execution-start-trace-id-value`=`225f6bf5919a59d16c2549f4`（非空且≠task_id）；同任务仅 1 条评论，第二条 @镜像会新起真实 VM，未执行。  `bindStartVmTraceContext` 曾把 run trace 设成 task_id，刷新后行消失。现已改为每评论独立 TraceId 写入 `start_trace_id`；单测已绿；须 9999 跑 018 并精准重启后硬刷新。**【2026-08-14 夜间】** 单评论 trace-id 已在生产验证：`225f6bf5919a59d16c2549f4` 非空、≠task_id、硬刷新持久；两评论互异性待第二条 `@镜像` 评论启动后验（首条 agent 仍 running，未并发启动第二条）。**【2026-08-14 夜间再验】** fork 链两条独立评论已确认各 start_trace_id 非空且互异：task_15742467311115154867→`225f6bf5919a59d16c2549f4`、task_15747409651224945866→`bb1157e1b936e3d8eb7e7b05`，均 ≠ 各自 task_id 且硬刷新持久；同任务两评论互异性仍未验（需提交第二条 @镜像 评论，会新起真实 VM，夜间未执行）。
- **Action**: (1) 完成 OPT-20260813-020（9999 init + 重启）；(2) 硬刷新任务详情；(3) 对两条 `@镜像` 评论分别展开执行细节；(4) 断言各卡 `comment-execution-start-trace-id-value` 非空、互不相同、且都不等于任务 ID；(5) 硬刷新后再打开，值仍在。
- **Why**: 未迁移/未重启则公网仍无列或仍显示 task_id。
- **How to apply**: 机器验收：CDP 9222。https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_15742467311115154867/ ；`data-testid=comment-execution-start-trace-id-value`。

### OPT-20260813-011 — 公网硬刷新验收评论区 Workbench 访问不再报缺少实例ID或地域

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-13
- **Context**: 【2026-08-14 日间 Playwright】目标租户 875561774391259136 URL 被改写为当前租户 875588283562749952 后任务 404。  点击「Workbench 访问实例」曾 400「服务器配置缺少实例ID或地域」，因 workbench-link 只读任务级 CSC。现已改为：评论卡按钮带 `comment_id`，后端 `resolveScopedCloudServerConfig` 只读该评论 CSC；无 comment_id 时 ForRuntime + region 回填。task-cloud-service / taskFE 已登记精准编译重启。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」（含 task-cloud-service、taskFE）；(2) 硬刷新任务详情；(3) 打开**一条有独立实例的评论**执行细节，点击 Workbench / 刷新状态 / 停止服务器；(4) DevTools Network 中 `workbench-link`、`server-runtime-status`、`stop-vm` 须带该评论 `comment_id`；(5) Workbench 应新开 `ecs-workbench.aliyun.com` 且不再出现「服务器配置缺少实例ID或地域」；(6) 若仍失败，`data-testid=server-runtime-status-message` 须带非空 `data-traceId`。
- **Why**: 单测已绿，公网进程与 SPA 产物需重启/硬刷新后才能证明评论级实例可打开 Workbench。
- **How to apply**: 机器验收：CDP 9222。https://www.daydaymoney.com/tenant/875561774391259136/workspace/ws_-2747179960968540749/task-detail/task_15700034959237981866/ ；按钮文案「Workbench 访问实例」。

### OPT-20260812-055 — 公网验收：任务详情启动失败面板展示 startTraceId 且可重新启动

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-12
- **Context**: 【2026-08-14 日间 Playwright】目标租户 875362439758114816 URL 被改写为本租户后 404。  任务 `task_15652393783064603866` 曾因镜像关联地域 `cn-hongkong` 与运行模板 `cn-qingdao` 漂移导致 start-vm 400；面板 `starttraceid=""` 且仅显示通用「容器启动失败」。代码已修（CSI 地域优先、traceId 冷打开还原、mock CSC 升级），并已精准重启 taskFE/cloud/ai-provider，需公网硬刷新验收。**【2026-08-14 夜间】** 当前账号（仅 tenant 875588283562749952 成员）打开该任务 URL 被路由改写为本租户后 404「获取任务详情失败」，需 tenant 875362439758114816 下账号验收。
- **Action**: (1) 打开 https://www.daydaymoney.com/tenant/875362439758114816/workspace/ws_-2794705041295509749/task-detail/task_15652393783064603866/ 硬刷新；(2) 触发自动/手动启动；(3) 确认 `server-start-status-panel` 带非空 `data-traceId`，执行细节出现「启动 TraceId」；(4) 启动不再报「未找到匹配地域的运行环境」。
- **Why**: 仅靠本地单测无法证明公网 SSE/冷打开链路已带上 traceId，且真实云启动仍依赖现网关联数据与编译产物。
- **How to apply**: 机器验收：CDP 9222。浏览器硬刷新任务详情；失败时用面板 `data-traceId` 查 Loki `{job=~".+"} |= "<traceId>"`。

### OPT-20260811-004 — 验证陈旧 tenant 书签跳转 onboarding

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-11
- **Context**: 【2026-08-14 日间 Playwright】`/tenant/875999999999999999/settings/company/` 重定向到真实租户公司设置（非 /onboarding/）。无公司账号分支仍缺。  公司设置页 `companies/current` 404 与 `lastActiveTenantId` 清理已合入；夜间 SPA 已发布。需无公司会话验证书签 URL。**【2026-08-14 夜间】** 当前会话绑定有效公司（软刀→tenant 875588283562749952），打开 `/tenant/875999999999999999/settings/company/` 被正确重定向到真实租户公司设置（无红字 404），但「无公司→/onboarding/」分支需无公司账号登录，单账号夜间无法验证。
- **Action**: (1) 无公司会话打开 `/tenant/<不存在的id>/settings/company/`；(2) 应跳转 `/onboarding/` 并清除 `localStorage.lastActiveTenantId`。
- **Why**: 清库后历史 URL 仍可能落到红字 404。
- **How to apply**: 机器验收：CDP 9222。`TenantCompanySettings.vue`；`staleTenantRecovery.js`；路由守卫（OPT-20260811-005 已合入）。
- **Related**: OPT-20260811-003

### OPT-20260811-024 — 公网验收目标分支/基准分支 Suggest 时间序与下方定位

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-11
- **Context**: 【2026-08-14 日间 Playwright】首次打开创建任务可见 `#task-base-branch-0-0` 且文案为「正在拉取各仓库分支…」（OAuth 未报过期）；复测弹窗未稳定打开，时间序/下拉定位未断言。  创建任务「目标分支」等已从原生 datalist 改为 `BranchSuggestInput`（Teleport fixed 下方）；后端 GitLab/Bitbucket 按 tip 更新时间降序。另已修 Teleport 下拉被 `.app-modal-overlay`(z=9999) 挡住（inline `zIndex:11000`）。需在生产 SPA 编译发布后浏览器验收。**【2026-08-14 夜间】** 创建任务弹窗确认基准/目标分支为 BranchSuggestInput（input#task-base-branch-0-0 autocomplete=list），但 GitHub access_token 已过期（「无法获取 GitHub 分支：access_token 无效或已过期，请解除并重新绑定 GitHub」），分支列表无法加载 → 时间序与下拉定位无法验证；需先重新 OAuth 绑定 GitHub 后复验。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」含 taskFE + task-project-service (2) 打开 work-panel 创建任务，焦点「基准分支/目标分支」确认下拉可见、在输入框下方且新分支靠前 (3) 可选跑 `WorkPanel.create-task-branch-suggest-no-reopen.playwright.test.js`
- **Why**: 单元测试无法覆盖真实 modal stacking 与远端分支时间戳。
- **How to apply**: 机器验收：CDP 9222。`BranchSuggestInput.vue`；`CreateTaskProjectBranchSection.vue`；`taskProjectService/src/git_branches.go`。
- **Related**: OPT-20260811-051 已合并入本条（2026-08-14 用户确认：一次验收覆盖时间序+z-index）。

### OPT-20260811-021 — 公网验收人员管理「访问管理」（v72 Region 树）

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-11
- **Context**: 【2026-08-14 日间 Playwright】`/people/access/` 20s 仍「加载中…」；无 region 第二账号未验。  早期「菜单↔粗码」FE 仅为桥接，已被 ADR-0003 / v72（A1/B2）取代。须先落地 page→region 树 + PDP `region:*`/`page:*` + `RequireRegion`，再公网验收。**【2026-08-14 夜间】** PeopleAccess.vue 已含 Region 树代码（commit 4ad19c6），但完整验收需「无 region 用户」第二账号 + region 勾选保存改动，单账号夜间无法闭环，仍 pending。
- **Action**: (1) 实现 v72 P1 后精准编译重启 taskAuth/taskFE 等相关服务 (2) 管理员打开访问管理，按 region 勾选保存 (3) 无 region 用户侧栏隐藏且对应 API 403
- **Why**: 仅前端菜单隐藏不算完成；须前后端同源 Enforce。
- **How to apply**: 机器验收：CDP 9222。`docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md`；`PeopleAccess.vue` 重做为 Region 树。
- **Related**: OPT-20260811-040 已并入本条；OPT-20260811-043 仍单独验收 tenant_admin 全量勾选。

### OPT-20260811-028 — 公网验收空评论不挂任务级运行态

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-11
- **Context**: 【2026-08-14 日间 Playwright】有评论任务 task_15747409651224945866 执行细节/启动 TraceId 正常；空评论负例需新建任务，未做。  已移除 `execution-details-fallback`；空评论仅「暂无评论」。taskFE 已登记精准编译重启。**【2026-08-14 夜间】** 有评论任务执行细节面板正常（task_15747409651224945866 展示容器/SSE/启动日志）；但无法创建无评论任务（GitHub repo `ram-work` 不可访问，access_token 过期，「创建」静默失败），空评论负例未能实测。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」含 taskFE (2) 打开无评论任务详情（如 `auto_run=false` 新建/Fork）(3) 确认评论区无「执行细节/服务器启动失败/SSE」冒充气泡，仅见「暂无评论」+ 添加评论 (4) 有评论任务仍可在气泡内展开执行细节
- **Why**: 本地 vitest 已绿；需公网 SPA hash 发布后目视确认不再误导。
- **How to apply**: 机器验收：CDP 9222。`TaskDetailCommentsPanel.vue`；`docs/superpowers/specs/2026-08-11-empty-comments-no-runtime-fallback-design.md`。

### OPT-20260812-018 — 公网验收：创建项目选默认模版后实例筛选自动对齐

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-12
- **Context**: 【2026-08-14 日间 Playwright】原租户 875216530801979392 URL 被改写为本租户 create-project；未选中工作空间，`#hardware-config-section` 未出现。  已修 `cloud_server_config_defaults` 硬件字段落库 + `defaultConfigToRunTemplate` 映射到 `filter_options`/`selected_instance`；全部重新编译与重启已完成（taskFE/task-cloud-service healthy）。存量默认配置需在「工作空间管理 → 机器节点」重新保存后才带有 cpu/内存/实例类型。
- **Action**: (1) 硬刷新打开 `https://www.daydaymoney.com/tenant/875216530801979392/create-project/`；(2) 选工作空间后，在「快速应用默认模版」选一条已重新保存过硬件的默认配置；(3) 断言 `#hardware-config-section` 内 CPU 核心数/内存/系统盘/数据盘与模版一致，且可用实例列表按筛选刷新；(4) 若有默认实例类型，确认列表选中对应规格。
- **Why**: 本地单测不覆盖公网 cookie、可用实例远端 API 与异步恢复选中实例。
- **How to apply**: 机器验收：CDP 9222。`ProjectRunTemplatePanel.vue` → `applyRunTemplate`；`projectRunTemplateUtils.js`；`taskCloudService` server-config-default。

### OPT-20260811-062 — 公网验收邀请页预授「访问管理」

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-11
- **Context**: 【2026-08-14 日间 Playwright】邀请页可见「授予访问管理」勾选与页面/区域预授；发邀请+受邀账号 join 未做。  已实现邀请 `pending_grants` + join 落权 + FE「授予访问管理」快捷项；需 9999 执行 `009_invitation_pending_grants.sql` 并精准重启后公网验收。
- **Action**: (1) http://10.2.150.68:9999/ 初始化库含 taskTenant 009；(2) 精准编译重启 task-tenant-service / task-auth / taskFE；(3) 打开邀请页勾选「授予访问管理」发邀请；(4) 受邀账号 join 后侧栏可见「访问管理」。
- **Why**: 单元测不覆盖跨服务 internal secret 与生产 SPA 缓存。
- **How to apply**: 机器验收：CDP 9222。`PeopleInvite.vue`；`invite_handlers.go`；`rbac_apply_member_grants.go`。

### OPT-20260811-083 — 公网验收：订单评论租户↔超管互通

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-11
- **Context**: 【2026-08-14 日间 Playwright】本租户订单 ORD-20260813-001 可展开行项，DOM 无评论线程；超管互评未做。  本会话已实现 billing_order_comment、租户/超管 API、OrderCommentThread，并已本地 migrate 038 + 单测绿。需精准编译重启后在公网订单页互发留言。
- **Action**: (1) http://10.2.150.68:9999/「精准编译重启」含 task-bill + taskFE；(2) 租户打开 `/tenant/{tid}/billing/orders/` 展开订单发评论；(3) 超管 `/system-admin/order-records/` 同订单回复；(4) 双方刷新可见对方留言与 author_side 标签。
- **Why**: 未发布则公网仍无评论线程。
- **How to apply**: 机器验收：CDP 9222。`BillingOrders.vue` / `SystemAdminOrderRecords.vue` / `OrderCommentThread.vue`；`taskBill` comments handlers。

### OPT-20260813-013 — 公网验收：成员加入自动创建 system-auto Git 身份

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-12
- **Context**: 【2026-08-14 日间 Playwright】未邀请新成员；`/people/manage/` 加载中，未见到 system-auto。  v77 已实现 MEMBER_JOINED → ensure-default；需精准编译重启后邀请加入或新建成员，确认 `task_git_identities` 出现 label=`system-auto` 且邮箱为 hash 格式。
- **Action**: (1) http://10.2.150.68:9999/ 精准编译重启含 task-tenant-service、task-task-service、task-events-member-joined-1-create-default-git-identity、taskFE；(2) 邀请并接受加入公司；(3) 查库或 PeopleManage「Git 身份」确认系统默认身份。
- **Why**: 未重启则消费者未上线，自动建身份不生效。
- **How to apply**: 机器验收：CDP 9222。`member_joined/1_create_default_git_identity` port 18060；`BuildSystemGitEmail`；PeopleManage。
- **Related**: 原误用已完成编号 OPT-20260812-027（工作面板静默回弹），2026-08-13 重编号。

### OPT-20260814-005 — 公网验收：停机后运行态不得再显示启动中

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-14
- **Context**: 【2026-08-14 日间 Playwright+API】task_15742467311115154867 生命周期文案「已启动」，但 `server-runtime-status` 返回 runtime_status=Starting / instance_id=null / message=云实例创建中，等待分配（csc_-2704251625846902750，comment cmt_15742473712028559869）；`comment-runtime-stop-server-btn` 不可见，未能点停机。  任务 `task_15742467311115154867` 在 10:45:07 已 `stop_vm_success`，10:45:32 的 `server-runtime-status`（trace `dbee1e70-9165-4359-a09d-1218f7393f65`）仍展示「启动中」+「云实例创建中」。本会话已修：停机清绑定写 `Released`；无 instance 时 Released 优先于评论 CSC=Starting；停机 SSE processing 不再把生命周期打成启动中。须发布后验收。
- **Action**: (1) http://10.2.150.68:9999/ 精准编译重启含 task-cloud-service、taskFE；(2) 硬刷新任务详情评论执行细节；(3) 对仍有实例的评论点「停止服务器」；(4) 断言 `server-runtime-status-message` 不为「云实例创建中，等待分配」，`serverRuntimeStatusDisplayText` 为「已释放」或「已停止」，生命周期不是「启动中」。
- **Why**: 未重启则公网仍跑旧逻辑，停机后会继续误报启动中。
- **How to apply**: 机器验收：CDP 9222。页面 `.../task-detail/task_15742467311115154867/`；`data-testid=server-runtime-status-message` / `server-lifecycle-status`。

### OPT-20260815-013 — Playwright 验收启动失败文案与 TraceId 复制按钮

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-15
- **Context**: 任务 `task_876332280065323008` 评论 `cmt_876332298901942272` 启机 persist 0 行后，runtime-status 曾显示空闲态「未找到服务器配置记录」，启动 TraceId 与 ID 在 innerText 中粘连。代码已改为失败提示 + `启动 TraceId：` + 复制按钮。须精准重启后用 Playwright 闭环。 【2026-08-29 夜间二次探针】登录态打开 task_876332280065323008 详情页重定向工作面板，任务已清库。
- **Action**: (1) http://10.2.150.68:9999/ 精准编译重启 `task-cloud-service`、`taskFE`；(2) 硬刷新 https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876332280065323008/；(3) 断言 `[data-testid=server-runtime-status-message]` 含「启动失败」且不等于「未找到服务器配置记录」；(4) 断言 `[data-testid=comment-execution-start-trace-id]` 文案匹配 `启动 TraceId[：:]`，且存在 `[data-testid=comment-execution-start-trace-id-copy]`。
- **Why**: 未重启则公网仍跑旧 SPA/旧 Cloud；本页是用户点选的原问题现场。
- **How to apply**: 机器验收：CDP 9222。选择器 `data-testid=server-runtime-status-message` / `comment-execution-start-trace-id` / `comment-execution-start-trace-id-copy`。

### OPT-20260816-033 — 公网硬刷新确认评论身份行有「自动克隆子仓库」开关

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-16
- **Context**: 评论级身份迁移把关联项目改成只读，误卸 `TaskDetailNestedReposCloneStatus`，任务详情找不到「是否克隆子仓库」开关。已把开关挂到 `@镜像` 后的 `comment-composer-repo-identity-row`，仍 PUT 项目级 `auto_clone_nested_repos`。taskFE dist 已构建并登记精准编译重启。 【2026-08-29 夜间二次探针】登录态打开 task_876722807445155840 详情页重定向工作面板，任务已清库。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 `taskFE` 点「精准编译重启」；(2) 硬刷新 https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876722807445155840/；(3) `@镜像` 后断言 `[data-testid=comment-composer-repo-identity-row]` 内存在 `[data-testid=task-nested-repos-auto-clone-toggle]` 且文案含「自动克隆子仓库」；(4) 关联项目只读区无该开关。
- **Why**: 未重启 preview / 未硬刷新时用户仍会看到旧 SPA，身份行没有开关。
- **How to apply**: CDP 9222。选择器 `comment-composer-repo-identity-row`、`task-nested-repos-auto-clone-toggle`。

### OPT-20260818-046 — 公网硬刷新确认关闭自动克隆后可启用自动运行

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-18
- **Context**: 项目 `proj_-2304947540687519745` 父仓已授权且「自动克隆子仓库」关闭时，自动运行仍显示「存在 Git 仓库授权异常」。代码已改为忽略子仓 token_error / nestedError；taskFE dist 已构建并登记精准编译重启。本机 Playwright 登录未跳出 `/auth/login`。 【2026-08-29 夜间探针】/tenant/877397588196749312/projects/proj_-2304947540687519745/ 显示「获取项目详情失败」。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 `taskFE` 点「精准编译重启」(2) 硬刷新 https://www.daydaymoney.com/tenant/877397588196749312/projects/proj_-2304947540687519745/ (3) 父仓行「已授权」、自动克隆未勾选时，断言 `[data-testid=project-auto-run-git-gate-hint]` 不存在，`[data-testid=project-default-auto-run-label]` 不是「无法启动」
- **Why**: 未重启 preview / 未硬刷新时公网仍跑旧 SPA，用户继续无法启用自动运行。
- **How to apply**: CDP 9222。选择器 `project-auto-run-git-gate-hint`、`project-default-auto-run-label`、`project-detail-auto-clone-nested-repos`。

### OPT-20260825-031 — 公网硬刷新验收已释放任务详情仍展示 COS step_full 步骤卡片

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-25
- **Context**: 任务 `task_880070939797123072` 在容器已释放且 ztree 显示「push 失败」时，已选节点只剩「步骤来自归档」横条、看不到 step_full。Loki 已证实 COS 归档成功（`step_full_archive_ok` bytes=158603，object_key 含 layer_20260825_131433）。本会话已改 watcher+面板，但本机浏览器无登录态无法硬刷新公网验收。 【2026-08-29 夜间探针】登录态打开 task_880070939797123072 显示「未找到任务数据，请刷新页面或返回工作面板重试」，任务不可达。
- **Action**: (1) 精准编译重启 taskFE 并确认 `npm run build` 已切 public/html (2) 登录后打开该任务详情 accessCode 页，选中 completed 节点 (3) 确认 `comment-layer-ztree-exec-log-panel` 可见且步骤卡片有 LLM/工具全文，不再只有单行横条
- **Why**: 归档在 COS 已存在，缺陷是前端隐藏；部署前公网仍是旧 SPA。
- **How to apply**: `scripts/register-precise-restart.sh taskFE`；页面 `https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_880070939797123072/`

### OPT-20260825-027 — 精准重启后硬刷新验收用户页「租户」Tab

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-25
- **Context**: 租户 Tab 与 `GET /api/system-admin/accounts/admin/tenants/` 已落地，公网 SPA 须 build + 精准重启 taskFE/task-tenant-service/task-gateway 后才能在 https://www.daydaymoney.com/system-admin/users/ 看到。 【2026-08-29 夜间探针】账号 contact@daydaymoney.com is_superuser=false、platform_roles=[]，/api/system-admin/accounts/admin/tenants/ 返回 403 superuser or staff required；租户 Tab 渲染但「共 0 个租户」，需真超管账号验收。
- **Action**: (1) 确认 `.runall/precise_restart_services.txt` 含 taskFE、task-tenant-service、task-bill、task-project-service、task-gateway 并在 :9999 执行精准编译重启 (2) 硬刷新用户页租户 Tab，点击公司名进入详情，三块数据可见
- **Why**: 未重启时线上仍是旧包，用户感知不到本增量。
- **How to apply**: `scripts/register-precise-restart.sh taskFE task-tenant-service task-bill task-project-service task-gateway`；页面 `/system-admin/users/?tab=tenants` 与 `/system-admin/tenants/:id/`。

### OPT-20260825-020 — Playwright 派生任务须按新 task-detail URL 认领标签页

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-25
- **Context**: 公网 Fork 用 `window.open(..., 'noopener,noreferrer')`。CDP `waitForEvent('page')` 可能拿到空 URL 页，脚本会误停在源任务上，把 0% 克隆条消失当成结束。
- **Action**: (1) 监听 fork POST 响应里的新 `task_id` 或 URL 含 `task-detail/task_` 且 id ≠ 源任务 (2) `context.pages()` 按该 id 认领 (3) 仅当 overall 含 100% 或「项目克隆…完成」才结束等待。
- **Why**: 认错页会在流量尚未入账时误报失败，也等不到真实自动运行结束。
- **How to apply**: `taskFE/app/src/composables/taskDetail/taskDetailEditing.js` `openTaskDetailInNewTab`；验收脚本用 CDP 9222。

### OPT-20260825-008 — 精准重启后公网验收资料页手机绑定与占用转移


- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-25
- **Context**: 资料页 `POST /profile/bind-phone/` 曾被 profile upsert 吞掉，且 `18959264502` 已绑在另一活跃账号。代码已修路由与 reclaim，需编译上线后才能让 `contact@daydaymoney.com` 真正绑上。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点「精准编译重启」（已登记 task-auth、taskFE）；(2) 公网硬刷新 `/profile/#rg=profile.phone_binding`；(3) 用 18959264502 完成短信验证，若 409 则点「确认将号码转移到本账号」；(4) 确认资料页显示已绑定且不再被手机门禁打回。
- **Why**: 未重启则线上仍走假成功路径，用户会继续绑定循环。
- **How to apply**: `.runall/precise_restart_services.txt`；验收账号 `contact@daydaymoney.com`。

### OPT-20260824-088 — 精准重启后核验工作台启机失败不再假报「创建中」

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-24
- **Context**: 工作台评论卡片在阿里云 `InvalidAccountStatus.NotEnoughBalance` 后仍显示「云实例创建中，等待分配」，CSC 对照区「尚未分配」。taskCloudService 已让 runtime-status 识别 failed binding/CSC、人话化余额不足，并保留 `csc_id`。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点击「精准编译重启」消费已登记的 `task-cloud-service`；(2) 硬刷新原工作台页，断言 runtime-status 为启动失败/余额不足而非「创建中」，且 CSC 行不再因空 `csc_id` 显示尚未分配。
- **Why**: 账户仍可能余额不足，无法凭空创建实例；验收是 UI 停止假「创建中」，并露出可操作原因。
- **How to apply**: `.runall/precise_restart_services.txt` 已含 `task-cloud-service`；对照 TraceId `80fd8a40f5f9a3da0eeadd55` / `40fd4ef8-68f1-460c-939c-b86c4bd4625b`。

### OPT-20260823-064 — 公网硬刷新验收推荐页「订单分账」面板（需登录态，runAll 精准重启后）

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-23
- **Context**: 推荐页订单分账面板（taskFE 5682638）已提交并登记精准重启，但公网页面渲染与真实分账按钮流程未在浏览器验收（需登录态；本机无登录 Cookie 会话）。 【2026-08-29 夜间探针】/profile/referral/ 渠道分账面板渲染（冻结金额/失败金额/可分账金额列，DR2AKvP9J9 行 冻结¥0.00 失败¥0.06 可分账¥0.00），所有行可分账金额为 ¥0.00，无「分账」按钮可点；缺 15-30 天窗口内 pending 分账记录。
- **Action**: runAll 页面点击「精准编译重启」后，用 Playwright/CDP 带登录态访问 https://www.daydaymoney.com/profile/referral/ ，核验：(1) 订单分账面板渲染（冻结/可分账/已过期三态文案）(2) 可分账订单点「分账」→ confirm → 成功刷新列表且状态变已分账 (3) 已过期订单显示「已过期无法分账」无按钮。
- **Why**: 单测只覆盖组件交互，网关路由/真实后端数据/渲染布局需浏览器闭环。
- **How to apply**: 仓库 e2e 惯例（BLOCK_TODO_BROWSER 分流）；推荐人账号需有 15-30 天窗口内的 pending 分账记录。

### OPT-20260823-021 — Playwright 固化编辑用户「超级用户」仅系统管理员可改

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-23
- **Context**: 本会话已在编辑/添加表单与 taskAuth create/patch 加上 `super_admin`/`platform:manage` 闸门。Vitest 覆盖 disabled 与 payload 省略；公网员工账号硬刷新未做。
- **Action**: (1) 用系统管理员打开 `/system-admin/users/` 编辑用户，断言 `#edit-is-superuser` 可勾选 (2) 用无 `super_admin` 的平台员工（若可进页）断言 disabled 且保存请求体无 `is_superuser`
- **Why**: 权限控件容易被后续表单重构冲掉，E2E 才能拦住。
- **How to apply**: `taskFE/tests/`；账号见 `task2app/测试.ai.md`

### OPT-20260822-063 — 公网释放态任务详情验收层图节点已隐藏

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: 本会话已让「服务器已释放」徽章与任务关联空态对齐（缓存 zNodes 不再挡住）。公网 `task_878979977859592192` 当前徽章仍为「容器 运行中」，需等该评论实例释放后再对照。 【2026-08-29 夜间二次探针】登录态打开 task_878979977859592192 徽章已为「服务器已释放」，但 DOM 仍有 `comment-layer-ztree-panel`（=1）且无 `comment-layer-ztree-released`——taskFE 待精准重启，修复未生效，暂不可验收。 【2026-08-29 夜间复探】CDP 登录态硬刷新 task_878979977859592192：徽章「服务器已释放」；`comment-layer-ztree-panel`（=1）仍在但 `layer-files-tablist`/`comment-layer-ztree-command-panel` 已隐藏，且出现 `comment-layer-ztree-released-selected-node`。公网 SPA（build-time 2026-08-28T16:36Z）已含 `comment-layer-ztree-released` 代码，但 `shouldShowCommentLayerZtreeReleased` 在 `layerGraphNodeCount>0` 时短路返回 false（缓存 zNodes 存在），故渲染折叠面板而非 released 空态。原断言「无 comment-layer-ztree-panel + 有 comment-layer-ztree-released」与 6dc0b90 的精炼设计（释放时保留面板但折叠交互）不符，需更新断言或找 zNodes 为空的已释放任务验空态路径。 【2026-08-29 夜间三次探针】CDP 登录态再次打开该任务详情，页面加载后 body 为空（innerText 空）、无任何 ztree/释放徽章元素，任务详情已不可稳定渲染，无法验收。
- **Action**: (1) 对 taskFE 精准编译重启并硬刷新该任务详情 (2) 待徽章变为「服务器已释放」后展开执行细节任务关联 (3) 断言无 `comment-layer-ztree-panel` / `layer-files-tablist` / `comment-layer-ztree-command-panel`，有 `comment-layer-ztree-released`
- **Why**: 单测覆盖门控；公网仍须确认运行态 SSE 与 binding 滞后场景下 DOM 真的切走。
- **How to apply**: `data-testid=comment-execution-binding-status`；`TaskDetailCommentLayerAssociationBody.released-hides-nodes.test.js`；页面 `https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878979977859592192/?accessCode=eZtNEpH6zH`

### OPT-20260822-061 — 在确有层变动载荷的任务上验收「文件变动 · N」徽章与列表

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: 公网 `task_878979977859592192` 已确认 Tab 切换：默认「项目文件树」，点「文件变动」见空态「暂无文件变动数据」。该页 `displayCount=0`，未覆盖计数徽章与真实变动列表。
- **Action**: (1) 找一条选中可写层后 `TaskDetailExecLayerChanges` 有条目的任务详情 (2) 硬刷新断言 Tab 文案为「文件变动 · N」且 N 等于列表条数 (3) 点条目仍能拉文件预览
- **Why**: 空态验收不能代替有载荷路径；徽章与点击拉文件是既有 Playwright 覆盖的行为，公网仍须对照一次。
- **How to apply**: `data-testid=layer-files-tab-changes`；`TaskDetailExecLayerChanges`；Playwright `openLayerFilesChangesTab.js`

### OPT-20260822-026 — 经 9999 应用分账/推荐比例列并精准重启验收申请表

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: 推荐码管理已可配置 `profit_sharing_ratio_percent` / `referral_rate_percent`，申请列表会展示两列。`057_referral_ratio_config.sql` 已于 2026-08-22 经 `apply_datamigrate.sh` 应用到生产 `task_bill`（data_migrate_log 显示已应用）。task-bill / task-referral 二进制已于 14:47 精准编译重启；taskFE SPA 已发布 `releases/20260822144823-1558766` 并 nginx reload。 【2026-08-29 夜间探针】/system-admin/users/?tab=referral-apps 推荐码申请列表加载失败（referral-applications 502），行内百分比列无法核验。
- **Action**: (1) ✅ 057 已应用 (2) ✅ Go 服务已重启、SPA 已发布 (3) 超管硬刷新 `/system-admin/referral-management` 保存分账/推荐比例 (4) 硬刷新 `/system-admin/users/?tab=referral-apps` 确认行内百分比
- **Why**: 未硬刷新时浏览器可能仍用旧 chunk，表头看不到两列。
- **How to apply**: 超管 Cookie 硬刷新上述两页；`data-testid=referral-applications-table`

### OPT-20260822-025 — 精准重启后验收推荐申请身份绑定授权勾选

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: 申请推荐资格已增加微信支付身份绑定说明与必选勾选；`009_identity_bind_consent.sql` 已用 `apply_datamigrate.sh` 应用到 `task_referral`。公网 SPA 与 taskReferral 进程仍须精准编译重启后硬刷新才能看到新文案。
- **Action**: (1) 在 http://10.2.150.68:9999/ 精准编译重启 task-referral + taskFE (2) 打开 https://www.daydaymoney.com/profile/referral/ 未获资格账号：未见勾选时申请按钮 disabled；勾选后可提交 (3) 确认文案含「账号标识」「用户身份信息」「微信支付」
- **Why**: 未重启则旧二进制仍接受无 `identity_bind_consent` 的申请，公网 SPA 也可能仍是旧包。
- **How to apply**: `scripts/register-precise-restart.sh task-referral taskFE`；页面 `data-testid="referral-identity-bind-consent"`

### OPT-20260822-023 — 经 9999 应用 billing_profit_sharing BIGINT 迁移

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: `056_profit_sharing_order_id_bigint.sql` 已于 2026-08-22 经 `apply_datamigrate.sh` 应用到 `task_bill`，`order_id`/`tenant_id` 已是 BIGINT。taskBill 工作树另有其他会话 `referral_config.go` WIP，此刻不能精准编译重启，以免把未提交代码打进二进制。
- **Action**: (1) 确认其他会话提交或还原 `taskBill/src/referral_config*.go`；(2) 在 `http://10.2.150.68:9999/` 精准编译重启已登记的 `task-bill` + `taskFE`；(3) 硬刷新 `/system-admin/order-records` 展开一笔有推荐关系的已支付订单，应出现「分账」表。
- **Why**: 不应用迁移则真实订单分账插入在 STRICT 下失败，管理端永远「无分账记录」。
- **How to apply**: `dataMigrate/taskBill/056_profit_sharing_order_id_bigint.sql`；入口见约束 34/40。

### OPT-20260822-009 — 经 9999 应用 task_comments git_pr 列迁移并验收 PR 回复

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-22
- **Context**: PR 回复/一键合并已合入 main（dataMigrate `6bbdb52` / taskTaskService `8ebb799` / taskGitOauth `0040dbb` / taskFE `79fc209` / docs ADR-0028）。`013_comment_git_pr.sql` 已于 2026-08-22 03:36:53 应用到本机 `task_task`（`parent_comment_id`/`git_pr_html_url`/`git_pr_json` 列存在）。业务进程仍是旧二进制；runAll :9999（127.0.0.1 与 10.2.150.68）均 down，精准编译重启与公网硬刷新仍阻塞。
- **Action**: (1) 在 `http://10.2.150.68:9999/` 执行「初始化全部数据库」或对 task_task 跑 `apply_datamigrate.sh` 使 013 入库 (2) 精准编译重启 `taskFE` `task-task-service` `task-git-oauth` (3) 硬刷新任务详情，推送出 PR 后确认嵌套回复、合并状态徽章与一键合并
- **Why**: 未应用 DDL 时评论 INSERT 新列会失败，前端会 warn 且会话看不到 PR 回复。
- **How to apply**: `dataMigrate/taskTaskService/013_comment_git_pr.sql`；`.runall/precise_restart_services.txt`；页面 `task-detail/task_878583341551480832/` `data-testid="comment-git-pr-reply"`

### OPT-20260821-024 — Playwright 验收启动面板不吸入他评论停机 SSE

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-21
- **Context**: 同任务旧评论 `stop-vm` 的「正在准备停止aliyun服务器」无 comment 标签，曾被 `soleActiveBinding` + `aliyun` 正则吸入新评论启动日志。前端已排除停机行，需公网硬刷新后 E2E 锁住。 【2026-08-29 夜间探针】登录态打开 task_878583341551480832，4 条评论均「服务器已释放」、无「正在准备停止aliyun服务器」文案，但无正在启动的容器面板，停机 SSE 吸入场景未实际触发（弱验收）。 【2026-08-29 夜间三次探针】CDP 登录态再验：仍无「正在准备停止aliyun服务器」、无启动中容器面板（弱验收）。
- **Action**: (1) 精准编译重启 `taskFE` (2) 打开任务详情展开两评论启动面板 (3) 断言新评论容器成功行附近不得出现「正在准备停止aliyun服务器」
- **Why**: 单测盖不住 SSE 水合 + `buildPerBindingServerStatusProps` 的任务级 `statusLogs` 合并路径。
- **How to apply**: 页面 `task-detail/task_878583341551480832/`；`#comments-container` 内 `bg-gray-50` 启动日志；`isCloudServerStopLogLine`

### OPT-20260819-025 — 公网硬刷新验收管理端「支付与签署」admin_grant 文案

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-19
- **Context**: 已修复抽屉将系统赠送误标为「无关联签署记录」，并补齐付费路径支付条款门禁。task-bill / taskFE 已登记精准编译重启。 【2026-08-29 夜间探针】账号非 superuser，/system-admin/users/ 默认用户列表 403；需真超管账号。
- **Action**: (1) 9999 精准编译重启 task-bill + taskFE (2) 应用 dataMigrate/taskBill/048_order_consent_id.sql (3) 硬刷新 `/system-admin/users/` 打开「支付与签署」，确认金额 0 的赠送流水显示「系统赠送，无需支付签署」
- **Why**: 公网 SPA/Go 进程在重启前仍是旧行为。
- **How to apply**: `SystemAdminUserRechargeDrawer.vue`；`taskBill consent_gate.go` + `048_order_consent_id.sql`

### OPT-20260819-018 — 公网硬刷新验收退款审批「关联订单」深链定位

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-19
- **Context**: 已实现列表 API `order_id` 分页对齐 + BillingOrders 展开滚动高亮；taskFE/task-bill 已登记精准重启，但公网 SPA 仍可能是旧 chunk。
- **Action**: (1) 在 9999 执行精准编译重启 taskFE (2) 硬刷新 `/system-admin/order-records/?tab=refund` (3) 点击「关联订单」，确认进入 `/system-admin/order-records/?tenant_id=&order_id=` 订单 Tab，目标行展开并带 `rg-deep-link-highlight`（不再进入租户 `/billing/orders/`）
- **Why**: 元素调整的最终验收在公网页面，本地单测无法代替缓存/发布。
- **How to apply**: `SystemAdminRefundPanel.vue` 链接；`BillingOrders.vue` + `useBillingOrderIdDeepLink.js`；`taskBill handleListOrders` 的 `order_id`/`focus_order_id`

### OPT-20260819-002 — 公网硬刷新验收任务详情克隆失败提示与创建任务 OAuth 门禁

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-19
- **Context**: 已实现 auto_run 条件 OAuth 门禁 + Fork 弹窗 OAuth 拦截（`7606f16`）；须 build 发布后在公网验收。 【2026-08-29 夜间探针】CDP 登录态 work-panel 创建任务弹窗勾选自动运行后 `create-task-repo-oauth-section` 出现，但默认仓 `https://github.com/task2money/ram-work` 显示「已绑定 Git OAuth」（无 `create-task-repo-oauth-bind` 链接）；提交被「请先选择智能体资源配置后再创建」拦截。需未绑定仓（如 tencent-sh-1 或未授权 GitHub 仓）才能触发 gate 负例。
- **Action**: (1) `taskFE/app` 执行 `npm run build` 并精准重启 taskFE (2) 硬刷新任务详情 URL（须已登录）(3) 创建任务勾选自动运行且未绑 OAuth 时提交 disabled 且出现 `create-task-repo-oauth-bind` (4) Fork 弹窗未绑 OAuth 时 `fork-confirm-auto-run` disabled 且出现 `fork-auto-run-oauth-bind`
- **Why**: 元素调整的最终验收在公网页面，本地单测无法代替缓存/发布。
- **How to apply**: `https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_877835528962076672/`；`CreateTaskModal.vue`；`ForkAutoRunConfirmModal.vue`

### OPT-20260818-042 — 公网硬刷新后对 tencent-sh-1 仓重新 OAuth 并验收分支列表

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-18
- **Context**: 项目页 `proj_-2304947540687519745` 对 `gitlab-tencent-sh-1.daydaymoney.com` 仓列分支曾 401（trace `2a45093b`）：默认实例 token 被借给区域 CE。代码与 Doorkeeper app 已修，需用户在该区域重新授权后公网验收。
- **Action**: (1) 硬刷新 `https://www.daydaymoney.com/tenant/877397588196749312/projects/proj_-2304947540687519745/` (2) 对该仓完成 GitLab OAuth（须跳到 `gitlab-tencent-sh-1.daydaymoney.com` 而非 `gitlab.daydaymoney.com`）(3) 断言红色 `p.mt-1.text-xs.text-red-600` 401 文案消失且分支下拉有数据
- **Why**: 旧绑定是默认实例凭证；不重新授权则 access-for-user 对 `gitlab:tencent-sh-1` 仍 404，页面会变成「未检测到可用授权」。
- **How to apply**: catalog `GET /api/git-oauth/providers/` 含 `gitlab:tencent-sh-1`；OAuth start `service_provider=tencent-sh-1`

### OPT-20260816-061 — 对齐存量串行误启实例与可并行卡 starting 的 Lisp 评论

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-16
- **Context**: 任务 `task_876810593758113792` 上 Java 串行评论 `cmt_876810842430009344` 被 @镜像旁路建成 Running 实例 `i-m5egq9vqz5hhe8qygvsv`，binding 仍为 `waiting_previous`；Lisp 可并行评论 `cmt_876810929306628096` binding=`starting` 但 CSC 无 instance。代码门禁已修，存量机器不会自动消失。
- **Action**: (1) 精准编译重启后硬刷新该任务详情，确认 Java 卡片显示「等待前序」且不再出现启停按钮 (2) 确认 Lisp 经 advance/bootstrap 出现 start 事件与实例；若仍 starting，用同任务已有 start 载荷再触发评论级 start-vm (3) 前序完成后若 Java 误启实例仍在，经 Workbench/stop-vm 释放 `i-m5egq9vqz5hhe8qygvsv`
- **Why**: 门禁只拦截新的冷启动；页面上的错位 CSC 会让验收误判修复未生效。
- **How to apply**: 任务 `https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876810593758113792/`；CSC/binding 表 `comment_container_bindings` / `cloud_server_configs` / `cloud_server_events`

### OPT-20260817-006 — 精准编译重启 taskFE 并发布公网 SPA

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-17
- **Context**: 评论执行细节「手动重试」已补 comment_id（path kv + body）。未编译发布前公网仍会 400「缺少评论ID」。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对 taskFE 执行「精准编译重启」（含 collectstatic）(2) 硬刷新任务详情 (3) 对失败仓点「手动重试」，确认不再出现「缺少评论ID」
- **Why**: 前端改动未进公网静态资源则用户仍无法重试克隆。
- **How to apply**: 服务名 `taskFE`；登记 `.runall/precise_restart_services.txt`；验收页同上任务详情评论执行细节克隆进度

### OPT-20260817-019 — 应用 012 迁移并精准重启后公网验收 auto_run 跳过横幅

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-17
- **Context**: 本会话已实现 skip reason 落库、Fork 弹窗、详情横幅与强制重试；已登记 `task-task-service` / `taskFE`。现网任务 `task_877108648822730752` 仍显示「未启动」无原因，需迁移 + 编译重启后才能冷打开见横幅（存量零评论任务有启发式兜底）。
- **Action**: (1) 在 http://10.2.150.68:9999/「初始化全部数据库」应用 `dataMigrate/taskTaskService/012_auto_run_start_skip_reason.sql` (2) 「精准编译重启」`task-task-service` + `taskFE` (3) 打开该任务详情确认琥珀色横幅；(4) 点「强制重新启动」：Git 仍超时则再弹 skip；网络/授权恢复后应出现【自动运行】评论并启服
- **Why**: 代码未部署时公网仍是旧行为；迁移未跑则 GET 无法读 skip 列（测试用 SQLite 已有列，生产 MySQL 缺列会扫表失败）。
- **How to apply**: 登记文件 `.runall/precise_restart_services.txt`；失败经验 `98_autorun_skip_no_ui_reason.md`；验收 URL 同 goal 任务详情页

### OPT-20260817-025 — 精准编译重启后验收「等待前序·终止」端到端

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-17
- **Context**: 已实现 waiting_previous → cancelled API 与执行细节摘要「终止」按钮；代码尚未部署到运行中的 task-cloud-service / taskFE。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」确认含 task-cloud-service、taskFE (2) 打开串行等待评论的任务详情 (3) 点摘要「终止」→ 确认弹窗 → badge 变为「已终止」且不再自动启机
- **Why**: 未重启则公网仍无终止入口；需验证网关路径 `.../commentId/cancel/` 与 SPA 产物。
- **How to apply**: 验收页示例 `…/task-detail/task_877131046670331904/`；API 契约见 `docs/intents/backend/cloud/comment_container_bindings.intent.md` T10

### OPT-20260818-007 — 精准编译重启 task-cloud-service 并硬刷新验收假「服务可用」修复

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-18
- **Context**: TraceId `dc780ea661e49d92fb84df21`：公网 IP 落库曾写入推测性 server_url，binding 误升 running。代码已改（不写 server_url），已登记精准编译重启。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点「精准编译重启」含 task-cloud-service；(2) 硬刷新任务详情；(3) 新启评论容器：公网 IP 出现后日志应停在「等待服务就绪」直至 register-reachability；(4) Loki 确认无 premature `comment_csc_bootstrap_cloud_ready` 仅因 public_ip。
- **Why**: 不重启则现网仍用旧二进制，假「服务可用」会复发。
- **How to apply**: `.runall/precise_restart_services.txt`；经验 `.ai/09_failure_experience/02_runtime_errors/57_binding_running_but_port_8080_refused.md`

### OPT-20260820-013 — 经 9999 应用 taskReferral 006 并精准重启后验收申请表

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-20
- **Context**: 推荐资格申请已改为必填个人介绍。DDL `006_referral_personal_intro.sql` 已用 `apply_datamigrate.sh` 应用到 `task_referral`（applied=6）。公网仍需精准重启 taskReferral + taskFE（SPA 已 `npm run build` release `20260820125539`）。
- **Action**: (1) 在 http://10.2.150.68:9999/ 精准编译重启 task-referral + taskFE (2) 打开 /profile/referral/ 确认申请表有个人介绍且未填不能提交
- **Why**: 未重启则旧二进制仍接受空 body，SPA 也可能仍是旧包。
- **How to apply**: `dataMigrate/taskReferral/006_referral_personal_intro.sql`；`scripts/register-precise-restart.sh task-referral taskFE`；公网 `/profile/referral/`

### OPT-20260823-047 — 精准重启后公网验收推荐绩效「微信分账」Tab

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-23
- **Context**: 本会话已实现超管用户列表推荐绩效抽屉的「微信分账」Tab（GET JOIN + POST QueryOrder），单测全绿。公网 SPA 与 taskBill 需精准编译重启后，用平台员工账号打开 `/system-admin/users/` 抽屉才能确认真实数据与微信侧核单。 【2026-08-29 夜间探针】账号非 superuser，/api/system-admin/users/ 403 forbidden、/api/system-admin/referral-applications/ 502 服务暂不可用；需真超管账号。
- **Action**: (1) 在 :9999 对 `task-bill` `taskFE` 精准编译重启 (2) 硬刷新 `/system-admin/users/` (3) 打开有被推荐人的用户抽屉，切「微信分账」，确认默认不提前请求、列表无 openid、同步按钮失败带 data-traceId
- **Why**: 只读运营核对依赖网关+微信 live QueryOrder，单测替不了商户平台限频与真实台账。
- **How to apply**: `.runall/precise_restart_services.txt`；`taskFE/app/src/components/ReferralWechatProfitSharingTab.vue`；`taskBill/src/handlers_admin_refresh_profit_sharing_wechat.go`

### OPT-20260817-009 — 重启 runAll 后验收全部重新编译清空登记徽章

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-17
- **Context**: `BuildAll` 收尾已恢复清空 `.runall/precise_restart_services.txt` + consumed-at；runAll 是最外层编排进程，不在精准编译重启名单里。现网 :9999 仍是旧二进制时，页头「已登记 N 个服务」不会按新逻辑消失。
- **Action**: (1) `cd runAll && ./build.sh` 后按现网方式重启编排进程（勿把 runAll 登记进精准重启以免递归）(2) 确认登记文件非空 (3) 在 http://10.2.150.68:9999/ 点页头「全部重新编译」并等完成 (4) 确认 `#precise-restart-reg-label` 为空且登记文件 0 行
- **Why**: 单元测试覆盖引擎，不覆盖正在运行的 :9999 二进制与 SSE→refresh 徽章链路。
- **How to apply**: `Runner.BuildAll` / `clearPreciseRestartRegistrationsAfterFullRebuild`；`status_ui/js/01.js` refresh→refreshPreciseRestartRegistrations；`status_ui/js/02.js` updateProgress done→refresh

### OPT-20260817-023 — 精准编译重启并验收 DeepSeek base_url 404 修复

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-17
- **Context**: TraceId `9257e326bcadc9405df4b150`：官方 DeepSeek + OpenAI 客户端配 `/anthropic` 会 404。**2026-08-18**：运营商端点不限制写法，代码不再把 `/anthropic` 改写成 `/v1`；仅修粘连 URL。验收改为「用户填什么就下发什么」。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」task-cloud-service + taskFE (2) 提交后于 `trae-agent/onlineServiceJS` 执行 `DOCKER_PUSH=1 ./buildDocker.sh` (3) 设置页保存自定义/anthropic 端点后读回原样；粘连 URL 失焦后只留最后一段 (4) 重建任务容器后 YAML `base_url` 与填写一致
- **Why**: 旧「强制改写 /v1」与运营商多样端点冲突；部署后才能在公网设置页验证不覆写。
- **How to apply**: 设置页 `…/tenant/877397588196749312/settings/feature-params/`；经验库 `101_feature_params_base_url_concat_overwrite.md`

### OPT-20260821-014 — 精准重启后强制重试 task_878541740905099264 自动启服

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-21
- **Context**: 该任务因父仓 `validate-git-repos` 24s 才 200、task 侧 15s 超时被软跳过。源码已改为 token-only + 40s 探测客户端；存量 skip_reason 仍落库，需重启进程后点「强制重新启动」。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点击「精准编译重启」`task-task-service` (2) 打开任务详情点「强制重新启动」 (3) 确认横幅消失且出现【自动运行】启服，而非再次「探测失败」
- **Why**: 仅合入代码不会清掉已落库的 skip_reason，也不会让旧二进制使用新超时。
- **How to apply**: `scripts/register-precise-restart.sh task-task-service`；页面 `data-testid="auto-run-start-skipped-banner"`；`force_auto_run`

### OPT-20260821-017 — Playwright 验收项目内联编辑镜像架构两侧文案

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-21
- **Context**: 项目详情内联编辑 400 已改为同时展示「镜像要求」与「实例系统支持」；`private_x86_64-latest` 读路径已推断 x86_64。公网保存成功/真不匹配红字尚未用登录态 Playwright 点一次。 【2026-08-29 夜间二次探针】登录态打开 proj_878549993831559168 返回「获取项目详情失败」，项目不可达。
- **Action**: (1) 打开 `https://www.daydaymoney.com/tenant/877397588196749312/projects/proj_878549993831559168/` (2) 保存现有 x86 镜像+`ecs.c6.large` 应成功 (3) 选无法识别 ISA 的镜像时 `project-inline-edit-error` / `project-image-arch-save-hint` 须含「镜像要求」和「实例系统支持」，且带 `data-traceId`
- **Why**: 仅 lookup/单测绿不能证明公网 SPA 与网关鉴权路径把新 400 原文渲染到原节点。
- **How to apply**: `taskFE/app/src/components/ProjectDetailImageField.vue`；E2E 账号见 `task2app/测试.ai.md`；禁止只断言旧笼统句「无法解析已安装镜像的 CPU 架构，拒绝保存以免规格不匹配」

### OPT-20260824-078 — 镜像/技能 mention ID 化三阶段实施：taskCloudService 派生 ID + taskTaskService 快照契约 + taskFE 保存/回填

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-24
- **Context**: 前端显示 `$镜像名 /技能名`、后端按 ID 存储（镜像名↔ID、技能名↔ID 映射持久化）三阶段实施完成：Phase 1 taskCloudService 在提取/安装 imageSkills.yaml 时生成稳定 `sk_<sha1(name+镜像seed)>[:12]`（seed = external_image_id 回退 installed id，041 迁移回填存量）；Phase 2 taskTaskService `task_tasks` 新增 `image_skill_id` 列 + `container_image_snapshot` JSON（{image_id, image_name, skill_id, skill_name}），API 契约 D4：请求可选 `container_image_skill_id` fail-closed 校验（skill_not_in_image / skill_id_requires_image / image_not_found / image_lookup_failed），update 宽容路径（未显式提供 image 时目录失效不阻断、保留快照），解绑语义 `container_image_id:""` 全清空；Phase 3 taskFE 保存携带 `container_image_skill_id`（目录反查技能名→稳定 ID）+ 编辑回填（详情对象 → 快照 → 扁平列回退链）+ taskChromePlugin buildCreateTaskPayload 透传。测试：taskTaskService 全量 src 绿、taskFE 3418 例绿、plugin 381 例绿。
- **Action**: 在 http://10.2.150.68:9999/ 点击「精准编译重启」消费登记（taskFE/task-task-service，taskCloudService 已消费）；重启后在 work-panel 创建/编辑任务验证：保存后详情返回 `container_image.skill = {id, name}`、快照落库、编辑弹窗技能绑定不丢失；ImageMarket 新安装镜像确认 image_skills_json 含 `id`。
- **Why**: 部署验证闭环 —— 契约与 ID 派生仅在真实服务链路可用后可见。
- **How to apply**: 登记文件 `.runall/precise_restart_services.txt` 已含 taskFE/task-task-service；重启后浏览器核验创建任务（$trae-agent /general-coding）→ 详情接口 `container_image.skill.id` 为 `sk_<hash>` 且与 imageSkillsFromImage 透传一致；数据库查 `task_tasks.container_image_snapshot` 含 skill_id。

### OPT-20260826-019 — 精准重启 task-bill 后用新幂等键重试 profit_sharing_id=3

- **Status**: pending
- **Blocked-By**: BROWSER
- **Created**: 2026-08-26
- **Context**: 本会话已把 55 分订单出站金额封顶为 2 分、微信业务拒单改为 409，并登记精准编译重启 `task-bill`。现网进程仍是旧二进制，id=3 在重启前重试仍会按 3 分打微信。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点「精准编译重启」 (2) 超管用**新** Idempotency-Key 对 `/api/system-admin/profit-sharing/3/share/` 再点一次分账 (3) 确认 200 且微信分账单号回写，或 409 带可读 message（不再 502）。
- **Why**: 代码未部署则生产单仍失败。
- **How to apply**: runAll 精准重启；系统管理待分账 Tab id=3。
