# Completed OPT Archive — 2026-08-25

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 34 条。
> 归档执行时间：2026-08-28T02:10:47+08:00

## [OPT-20260824-057] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskGateway 1acd92d 已推送：3 条 lint 警告路由各补 /* 通配（taskauth-users-bind-login-methods / taskauth-tenant-memberships / api-system-admin-order-number-parse），routes-apply 热重载后 --lint 0 警告；顺带修复 checker 检出的真实缺口 idempotency-records（OPT-20260824-058 新增端点缺网关路由 → 502，补双挂载路由后实测 403）。
- **Created**: 2026-08-24
- **Context**: `python3 taskGateway/scripts/routes-to-apisix.py --lint` 报 3 条通配符覆盖率警告：`taskauth-users-bind-login-methods`（priority 848）、`taskauth-tenant-memberships`（priority 860）、`api-system-admin-order-number-parse`（priority 856）——均只有精确 URI（含结尾 `/`）无 `/*` 通配，若上游存在子路径（如 /resend/、/{id}/）将回退到低优先级通配路由（spa-catch-all → 502 HTML）。存量问题，非本次导出功能引入（新增路由均含 `/*`）。
- **Action**: 在 routes.yaml 为上述 3 条路由的 uris 各追加一条 `/*` 通配条目，重新生成 apisix.yaml 并跑 `routes-apply` 热重载；跑 `--lint` 确认 0 警告。
- **Why**: 精确 URI 缺通配时子路径请求会落入 spa-catch-all 返回 502 HTML，破坏前端体验；lint 门禁已存在，消除告警即消除隐患。
- **How to apply**: `taskGateway/routes/routes.yaml` → `TASK_GATEWAY_APISIX_IN_DOCKER=1 bash run.sh routes-apply`（内容式新鲜度 + 热重载）。

## [OPT-20260824-072] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskGateway 5488010 已推送：check_go_routes_vs_apisix 新增 _extract_dispatch_routes 提取内联 closure 内 parts[N]=="literal" 分发路由（合成 /api/tenant/*/workspace/*/<lit> 类路径），DISPATCH_ROUTE_EXCLUSIONS 排除 retired/约定路径 legacy；补 5 例回归测；TTS 从 4 checked 增至 9 checked，全服务 0 missing。
- **Created**: 2026-08-24
- **Context**: 排队调度页 404（OPT-20260824-005 后端合入但网关漏登记）根因修复时确认：`taskGateway/scripts/ci/check_go_routes_vs_apisix.py`（OPT-20260728-014 专防此类缺陷）只提取 `mux.HandleFunc("...")` 字面路由，而 TTS 的 `/api/tenant/` 前缀 handler 内 `parts[N] == "queue-schedule"/"todos"` 分发式路由完全不可见（本次 queue-schedule 与并行会话的 todos 缺陷均未检出，双双靠线上 404 暴露）。
- **Action**: 扩展 checker 提取逻辑：解析 `/api/tenant/` 前缀 handler 内 `parts[N] == "<literal>"` 分发模式，合成 `/api/tenant/*/workspace/*/<literal>` 类路由参与比对；或对 TTS 引入显式路由清单（如 `queued_schedule_workspace.go` 头部注释表）供 checker 消费。
- **Why**: 分发式路由占比持续上升（queue-schedule/todos/translate-branch-title 等），checker 盲区 = 此类缺陷系统性回归通道。
- **How to apply**: 在 `extract_go_routes` 增加 parts 分发正则（`parts\[\d+\] == "([a-z0-9-]+)"` + 所在前缀 handler 上下文），归一化为 `/*` 路径后加入提取集合；补 `test_check_go_routes_vs_apisix.py` 用例（用 TTS main.go 的 queue-schedule/todos 段落）。

## [OPT-20260824-073] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskGateway 5488010 已推送：SKIPPABLE_EXACT 增加 /api/tenant_id（taskTaskService handleTenantIDPrefixedRoutes 裸子路由挂载点），消除持续误报噪音；补 is_skippable 断言。
- **Created**: 2026-08-24
- **Context**: checker 对 taskTaskService 报 `MISSING: /api/tenant_id`——`mux.HandleFunc("/api/tenant_id/", handleTenantIDPrefixedRoutes)` 是子路由裸挂载点，实际路由（`/api/tenant_id/*/workspaceId/*/tasks/*/comments/*`）已在 routes.yaml 登记（task-task-service 862）。与 SKIPPABLE_EXACT 中 `/api/tenant` 同型。
- **Action**: 将 `/api/tenant_id`（及检查其他服务的同类裸挂载点，如未来 `mux.HandleFunc("/api/user/")` 模式）加入 `SKIPPABLE_EXACT`。
- **Why**: 持续性误报噪音淹没真实 MISSING 信号（本次 summary 3 missing 中 1 条即此误报）。
- **How to apply**: 在 `check_go_routes_vs_apisix.py` SKIPPABLE_EXACT 增加 `/api/tenant_id`；重跑确认 taskTaskService 仅剩真实缺失。

## [OPT-20260824-066] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskAiProvider f0a0f8a 已推送：minttoken staff 分支 SQL 从旧表名 marketplace_platformstaff 改 ai_provider_platformstaff（与 infrastructure GetStaffByID 一致），抽 staffLookupSQL/staffInsertSQL 单点 + main_test.go 断言防回退；go test 绿。
- **Created**: 2026-08-24
- **Context**: `taskAiProvider/scripts/minttoken/main.go` 查询 `marketplace_platformstaff`（表前缀迁移前的旧名），实际表为 `ai_provider_platformstaff`，导致 `go run ./scripts/minttoken staff` 报错不可用（本次修复验证用临时脚本绕过）。
- **Action**: minttoken staff 分支查询/插入改用 `ai_provider_platformstaff`（与 infrastructure.GetStaffByID 一致）。
- **Why**: 内部调试/测试需要稳定的 staff token 铸造入口，脚本损坏会拖慢后续联调。
- **How to apply**: `taskAiProvider/scripts/minttoken/main.go`（改表名即可，加一条 staff 铸造冒烟验证）。

## [OPT-20260824-084] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE 6adb80b 已推送：抽 parsePaymentProviderError 共享 dump 清洗层（退款/分账共用），profitSharingFailReasonLabel 对 SDK dump 输出「分账失败：<Message 摘要>」不泄漏签名头；机器码映射优先级不变；补 4 例回归测 + 相关组件 10 例绿。
- **Created**: 2026-08-24
- **Context**: 待分账队列失败原因已把 `qualification_revoked` 等机器码映射为中文。`processPendingProfitSharings` 失败时仍把 `err.Error()`（含 `error http response` / `Wechatpay-Signature`）写入 `fail_reason`，前端未知码会原样展示。
- **Action**: (1) 在 `profitSharingFailReasonLabel` 中对 SDK dump 调用既有 `humanizePaymentProviderError`（或抽一层与退款共用的 dump 清洗）；(2) 补测例断言签名头不出现在 UI。
- **Why**: 管理员失败原因列不应泄露微信支付签名头；与退款面板已有的 dump 清洗口径应一致。
- **How to apply**: `taskFE/app/src/utils/profitSharingFailReasonLabel.js` + `humanizePaymentProviderError.js`；回归 `profitSharingFailReasonLabel.test.js`。

## [OPT-20260824-091] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE 4b22362 已推送：新增 activeTabHasArchivedResults computed（active Tab + users 含 is_archived），SystemAdminUsers.vue 展示提示条（data-testid=archived-in-active-results-hint，提示标识已回收可切已归档 Tab）；补 3 例组件测试，既有 20 例全绿。
- **Created**: 2026-08-24
- **Context**: 后端已在 `q`/`phone`/`email` 检索时不再套用 is_archived=false，活跃页能搜到归档占用者。行上虽有「已归档」徽标，搜索后仍停在活跃 Tab 可能让运营以为筛选坏了。
- **Action**: (1) 活跃 Tab 搜索结果含归档用户时展示提示或自动带上归档标记；(2) 补 FE 单测。
- **Why**: 降低「搜得到但状态列是已归档」的认知成本。
- **How to apply**: `taskFE/app/src/composables/useSystemAdminUsers.js` 与 `UserListRow.vue`；已归档行已有样式可复用。

## [OPT-20260824-089] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskAuth 1f6b4d5 已推送：新增 findActiveIdentifierConflicts + describeIdentifierConflict，解档 handler 前置冲突检测（手机号按 cc+identifier、邮箱/用户名按 LOWER(identifier)，仅比对活跃用户），冲突 409 说明需先解绑或换号；3 例回归测绿。
- **Created**: 2026-08-24
- **Context**: 本会话将注册占用改为仅计活跃用户，并在新注册成功时作废归档/孤儿 login_method。解档旧用户时，其手机号可能已被新人占用，解档后若恢复绑定会冲突。
- **Action**: (1) 解档路径检测该用户仍持有的未作废标识是否与活跃用户冲突；(2) 冲突则 409 并说明需先解绑或换号；(3) 补回归测。
- **Why**: 避免解档把已回收手机号重新激活，造成双活占用或登录歧义。
- **How to apply**: `taskAuth/src/handlers_system_admin.go` 解档分支；断言与 `createUserWithPhoneLogin` 冲突用例。

## [OPT-20260824-090] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskAuth a7d4c07 已推送：新增 voidOrphanLoginMethods（双层子查询规避 MySQL 多表 UPDATE LIMIT 限制，只作废用户行缺失绑定）+ 内部端点 POST /api/internal/taskauth/login-methods/void-orphans/?limit=N（requireInternalSecret 保护，供 cron/运维定时清理）；2 例回归测绿。
- **Created**: 2026-08-24
- **Context**: 注册占用已忽略「用户行缺失」的 phone/email 绑定，并在新注册时作废。库中仍可能残留孤儿行，干扰人工查库与审计。
- **Action**: (1) 用 LEFT JOIN auth_user 找出 binding_voided_at IS NULL 且用户不存在的 login_method；(2) 运维脚本或一次性 API 批量作废；(3) 补测禁止误伤活跃用户。
- **Why**: 查询语义已修复，存量脏数据仍会让排障时误判「号码被占用」。
- **How to apply**: `taskAuth` 管理脚本或 dataMigrate 一次性 UPDATE；WHERE 必须含用户缺失条件。

## [OPT-20260824-075] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE 850e41e 已推送：ImageMarket 已安装卡片「安装时间」改「更新时间」（复用 formatInstalledImageUpdateTime，优先 updated_at 回退 installed_at），加 data-testid；helper 16 例绿。
- **Created**: 2026-08-24
- **Context**: 创建任务弹层镜像选项现展示「更新时间」= 目录快照 updated_at（缺失回退 installed_at，迁移 038 回填存量）。ImageMarket「已安装镜像」卡片仍只显示「安装时间 installed_at」，与弹层口径不一致，且新字段 updated_at 尚未在已安装区展示。
- **Action**: ImageMarket 已安装镜像卡片将「安装时间」行改为「更新时间 updated_at（缺失回退 installed_at）」或同时展示两者；复用 utils/installedImageLabel.js 新增的 formatInstalledImageUpdateTime。
- **Why**: 同一字段在弹层与镜像市场展示口径不一致会让用户困惑「更新时间」含义。
- **How to apply**: taskFE/app/src/views/ImageMarket.vue 已安装列表行（L219-221）；卡片内可加 data-testid="installed-image-updated-at" 供断言。

## [OPT-20260824-086] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskBill 7739862 + taskFE 0523447 已推送：渠道聚合新增 failed_amount_yuan_cents（display==referrerPSFailed 计入），推荐页订单分账表加「失败金额」列（红色 + data-testid）；Go 回归测断言 failed-in-freeze 计入失败非冻结，FE 9 例绿。
- **Created**: 2026-08-24
- **Context**: 管理端「失败」的分账在 15 天窗口内曾被映射为推荐人「冻结」。已改为 display=`failed` 且冻结金额不再计入该佣金；失败金额目前只含在订单金额中，推荐页无单独「失败」列。
- **Action**: 渠道聚合增加 `failed_amount_yuan_cents`；推荐页表格增加「失败金额」列；回归测 failed-in-freeze 计入失败而非冻结。
- **Why**: 用户对照管理端「失败」时，推荐页目前只能看到冻结金额变少，缺少正向「失败」展示。
- **How to apply**: `profit_sharing_referrer_channels.go` 聚合 + `ReferralProfitSharingOrdersPanel.vue` 新列；补 channels_test / panel.test.js。

## [OPT-20260824-064] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE ab8caa5 已推送：resolveAuthenticatedUserId 命中本地缓存时后台经 profile 校验（TTL 60s 限频、sessionStorage impersonatorAccountBackup 判定模拟会话跳过），不一致回填 storeUserId + console.warn；补 6 例回归测，调用方测试全绿。
- **Created**: 2026-08-24
- **Context**: 模拟会话 git-identities「获取身份列表失败」(403) 根因是 localStorage currentUserId 残留模拟者 ID；本次已在 activate-session 成功分支补 storeUserId 同步 + 模拟状态 refresh 自愈覆盖，但 getStoredUserId 对本地陈旧值（任何历史来源）仍无校验，未来其他写入路径可能再次引入不一致。
- **Action**: 在 resolveAuthenticatedUserId（或 Sidebar 已有的 profile 同步点）对本地 currentUserId 增加服务端校验：命中时用 profile API 的权威 user_id 比对，不一致则回填并告警日志；仅在模拟会话外比对（模拟会话由 status 端点自愈接管）。
- **Why**: 以「服务端为准」闭环所有 userId 本地缓存的一致性问题，防患同类 403。
- **How to apply**: `taskFE/app/src/utils/sessionUserIdUtils.js`（resolveAuthenticatedUserId）+ 复用 profile 缓存/短路避免多余请求

## [OPT-20260824-077] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE 6941651 已推送：onInput 在镜像下拉未开且正文含完整 $镜像、DOM 无 chip 时重建 chip（guard !menuOpen 避免打断类型选择），粘贴/纯文本路径 / 技能菜单可开；补 2 例回归测，相关 composer 21 例绿。
- **Created**: 2026-08-24
- **Context**: 修复 OPT 根因（评论区 $镜像 后 / 技能下拉）时发现：`onInput` 只做 `pushModel + syncMentionState + refreshMenuFromCaret`，**不调用 `renderPlainWithMention`**；chip 渲染只发生在 watch(model)/onMounted/selectImage/selectSkill 四条路径。用户直接粘贴 `$trae-agent /general-coding`（onPaste 走 execCommand insertText 仅触发 onInput）或手输完整文本后刷新时，正文无 chip（纯文本 `$trae-agent`），`extractMentionFromDom` 返回 null → mentionedSkills 恒空 → / 技能下拉永不开启（纯文本恢复仅靠 watch 的 model 同步兜底，但 input 序列化模型与 DOM 相等时 watch 短路）。
- **Action**: 在 `onInput` 末尾（或 onPaste 内 renderPlainWithMention）增加轻量一致性修复：当 `serializeComposerDom` 中出现已知 `$镜像名` 且 DOM 无对应 chip 时，调用 `renderPlainWithMention(plain, mention)` 重建 chip（注意 caret 保持与 undo 栈影响评估，Vue 无受控序列化历史，代价低）。
- **Why**: 评论区核心交互（$镜像 + /技能）在粘贴路径下仍不可用，属于本次根因修复的同类未覆盖分支；create-task 描述框已有同类逻辑（TaskDescriptionSkillField），口径应一致。
- **How to apply**: CommentImageMentionEditor.vue `onInput`（约 L273-278）在 `syncMentionState(plain)` 后追加：`const m = extractMentionFromPlainText(plain, props.installedImages); if (m && !extractMentionFromDom(editorEl.value)) renderPlainWithMention(plain, m)`；补单测：粘贴 `$trae-agent /` 文本后断言 chip 存在且 / 技能下拉可开。

## [OPT-20260824-070] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE 969bc22: git rm CreateTaskImageMentionField.vue/test.js + renderImageMentionText 清理，34 例 vitest 绿
- **Created**: 2026-08-24
- **Context**: 创建任务弹窗「已安装镜像」独立字段（#task-container-image）下线后（镜像/技能选择收敛到任务描述 @ 弹层 TaskDescriptionSkillField），`taskFE/app/src/components/CreateTaskImageMentionField.vue` 与 `CreateTaskImageMentionField.test.js` 已无任何引用方（代码库 grep 确认），成为休眠死代码。本次因文件删除权限门禁未删除，保留在仓库。
- **Action**: 确认无复用计划后删除 `CreateTaskImageMentionField.vue` 及其 `CreateTaskImageMentionField.test.js`（git rm），并同步移除 `createTaskImageMention.js` 中仅被该组件使用的 `renderImageMentionText`（当前因组件保留而保留）。
- **Why**: 休眠组件增加维护面与认知负担，且其保留导致 utils 中一个无生产调用方的导出继续存活。
- **How to apply**: `git rm taskFE/app/src/components/CreateTaskImageMentionField.vue taskFE/app/src/components/CreateTaskImageMentionField.test.js`；删除 `renderImageMentionText` 及其测试块；跑 taskFE 全量 vitest 确认全绿。

## [OPT-20260824-071] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE f6b3df2 + taskProjectService c26e9a8: 字段设置开关移除 container_image，前后端对齐契约，FE 40 例 + Go 绿
- **Created**: 2026-08-24
- **Context**: 「已安装镜像」字段从创建任务弹窗移除后，`taskFE/app/src/utils/createTaskFieldSettings.js` 的 `CREATE_TASK_FIELD_SETTING_KEYS` 仍含 `container_image`（注释声明与 taskProjectService `knownCreateTaskFieldKeys` 对齐），工作区「字段设置」弹窗（CreateTaskFieldSettingsModal）仍展示该开关，但切换已不影响任何渲染。
- **Action**: 与 taskProjectService 协调：后端 `knownCreateTaskFieldKeys` 移除 `container_image`，前端同步从 `CREATE_TASK_FIELD_SETTING_KEYS`/`CREATE_TASK_FIELD_SETTING_LABELS` 移除，保持对齐契约。
- **Why**: 保留一个无效开关会误导工作区管理员；前端单独移除会破坏「与后端对齐」契约注释。
- **How to apply**: 后端 `taskProjectService`（knownCreateTaskFieldKeys）与前端 `createTaskFieldSettings.js` 同步移除 `container_image`，并删除字段设置相关测试中的该键断言。

## [OPT-20260824-068] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE 38a7339: work-panel 纯 mock E2E catch-all 最先注册 + user-permissions mock，10 例全绿
- **Created**: 2026-08-24
- **Context**: `WorkPanel.auto-schedule-link` E2E 首跑被 tenantRouteGuard 重定向 /onboarding/：诊断发现 `/api/accounts/users/me/` mock 返回 `{}`——Playwright `page.route` 为 LIFO（后注册先匹配），最后注册的 catch-all `**/api/**` 抢占了 /me/、workspaces 等全部具体 mock。实测验证（两种注册序对比）。task-detail 系测试因 catch-all 兜底模式先行（其断言不依赖具体 mock 响应体）未暴露。
- **Action**: work-panel 系纯 mock E2E（optimization-regression 等）按「catch-all 最先注册、具体 mock 在其后」重排；顺带修其缺 catch-all 兜底导致的真实后端 401 跳登录页/Onboarding 问题（既有环境问题，本任务未修）。
- **Why**: mock 顺序错误导致守卫判定错误、测试结果失真。
- **How to apply**: 参见 `tests/WorkPanel.auto-schedule-link.playwright.test.js` setupMocks 的注释与顺序约定。

## [OPT-20260824-069] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE 38a7339: optimization-regression 补 catch-all/权限码/todos mock 显式 advance，4 例全绿
- **Created**: 2026-08-24
- **Context**: `WorkPanel.optimization-regression.playwright.test.js` 在本次任务验证中同样失败（deliverable-section-other 不可见 + 未 mock API 打到真实后端 401 → 跳登录页）。与 068 同批修复：catch-all 200 兜底 + mock 顺序重排 + user-permissions mock（缺 tenant_perms 时 Sidebar 主导航三项隐藏）。
- **Action**: 修复 optimization-regression 测试（catch-all 最先注册 + user-permissions mock + EventSource mock），回归全绿后补 `t.Cleanup` 类防护核验。
- **Why**: 纯 mock E2E 不能依赖真实后端；既有失败掩盖了后续回归信号。
- **How to apply**: 复用 `WorkPanel.auto-schedule-link.playwright.test.js` 的 setupMocks 模式。

## [OPT-20260824-080] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskBill 213d013: disk enforce 个人命名空间仓校验归属租户（复用 resolveTenantIDFromGitlabUsername），BDD 7 例 + 整包 76s 绿
- **Created**: 2026-08-24
- **Context**: 排查流量闸门绕过时发现同类问题：taskBill `gitlab_disk_enforce.go`（sync-gitlab-disk-quotas 路径）对个人命名空间项目（example-user/somanyad）报 `gitlab_project_limit_failed ... project limit`（Loki 17:15:01 有 warn，tenant 877397588196749312）。与流量闸门同根因 —— 项目归租户逻辑只认 `tenant-{id}` 前缀，个人命名空间无法映射租户。
- **Action**: 将流量闸门同款「gitlab 用户名 → 凭据绑定 → 成员 → 区域资源」解析（gitlab_traffic_gate_username.go 的 resolveTenantByUsername 链）复用到 disk enforce 的租户解析；保持 fail-open/fail-safe 语义与既有开关一致。
- **Why**: 个人命名空间仓在磁盘配额上同样绕过租户级限制，修复流量闸门后此处成为剩余缺口。
- **How to apply**: taskBill/src/gitlab_disk_enforce.go 定位租户解析处，复用 resolveTenantIDFromGitlabUsername hook（含测试桩模式）；补 BDD 用例（个人命名空间命中租户 disk 配额）。

## [OPT-20260824-092] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: trae-agent 45700a6: POST /repos/clone 成功且 .git 出现后以 manual_clone 补跑 resumeAgentKickoffAfterCloneReady（buildManualCloneKickoff），cloneQueue 增 onCloneSuccess 钩子；6 例单测 + 全量 186 例绿；onlineServiceJS x86_64-latest=4015dabaf 已推送
- **Created**: 2026-08-24
- **Context**: 修复引导克隆失败后手动 reclone 不恢复 kickoff 时，刻意把空工作区（任务详情无关联仓库、`createInitialWorkspaceLayer`）的 `POST /api/repos/clone` 新建层成功路径排除在外。该路径同样会在无 git 时推迟/失败 kickoff，用户稍后手动 clone 成功也不会自动唤起 auto_run。
- **Action**: (1) 在 `POST /api/repos/clone` 队列完成且层内出现 `.git` 后调用 `resumeAgentKickoffAfterCloneReady`（reason=`manual_clone`）(2) 补单测：无关联仓库 bootstrap 推迟或不建任务；clone 成功后 kickoff 一次。
- **Why**: 与 reclone 恢复是同一「克隆能力恢复 → 继续被中断自动任务」语义，漏掉会在无预置仓库的任务上复现。
- **How to apply**: `trae-agent/onlineServiceJS/src/cloneQueue.mjs` 完成回调或 `routesReposClone.mjs` clone 成功路径；复用 `resumeAgentKickoffAfterCloneReady`。

## [OPT-20260824-058] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskAuth 0308388 已推送：internalPersonalDataGet 对每次上游 personal-data 调用套 context.WithTimeout（personalDataExportUpstreamTimeout=20s，测试可覆盖）；慢上游不再拖死整次同步导出，按既有 sections_unavailable 机制降级为 partial。回归测 TestPersonalDataExportUpstreamTimeoutDegradesToPartial（慢 httptest + 30ms 阈值 → bill unavailable + status=partial），taskAuth 全包 go test 127s 绿。
- **Created**: 2026-08-24
- **Context**: 导出 request/ 同步聚合 taskAuth+taskBill+taskTenantService+taskCloudService 四服务数据（每 section LIMIT 200 防超量）。上游任一服务慢（如 taskBill 全量回归 75s 量级的复杂查询）时，前端请求可能超时导致生成失败；当前无重试/异步兜底。
- **Action**: 将生成改为异步任务模式（request/ 立即返回 202 + task_id，后台聚合后落库，status/ 轮询或通知）；或在生成前对上游内部调用加超时与部分成功降级（现有 sections_unavailable 已覆盖失败降级，缺的是超时阈值）。
- **Why**: PIPL 导出权应保证任何数据量的用户都能在期限内成功导出；同步阻塞在多服务聚合上是可预见的瓶颈。
- **How to apply**: `taskAuth/src/personal_data_export.go`；复用既有 service 间内部 API 模式；dataMigrate 表结构已含 status 字段可扩展 pending 态。

## [OPT-20260824-062] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskAiProvider e7ad042 已推送：编辑镜像版本弹窗 openEdit 时若 image_skills_extract_status 与 auto_run_steps_extract_status 均为 ok，经 persistedResolveInfo 直接展示已持久化技能/自动运行说明并跳过 resolve-target-architectures 层下载；仅字段缺失/非 ok 退回解析端点。新增 hasPersistedResolveInfo/persistedResolveInfo 纯函数 + 5 例单测，前端全量 117 例绿。
- **Created**: 2026-08-24
- **Context**: 本次实现中编辑弹窗打开时走 resolve-target-architectures 重新向仓库同步提取技能与自动运行说明（与保存后异步提取同源同成本）；已保存版本在 DB 已有 image_skills_json / auto_run_steps_md（列表接口已返回），重复走层下载浪费带宽与时间。
- **Action**: openEdit 时若镜像记录已含 image_skills_json / auto_run_steps_md 且提取状态为 ok，直接 parseResolvePayload 展示，仅当字段为空/状态 pending 时才调用解析端点。
- **Why**: 编辑场景大部分是改大小/接口版本，不涉及镜像内容变更，重复全层扫描无收益。
- **How to apply**: `taskAiProvider/frontend/src/components/VendorContainerImageVersionModals.vue`（openEdit + resolveEditArch）

## [OPT-20260824-065] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskAiProvider 7cc2296 已推送：镜像组创建/编辑 POST/PUT 拒绝空白 name/description（镜像组名称必填/镜像组描述必填，trim 落库），前端 saveGroup 保存前本地校验。回归测 TestVendorImageGroupCreateRequiresNameAndDescription（空描述/空名称 400 + 正常创建 201 trim 生效），Go 全包 + 前端 117 例绿。存量空值补录：存量空描述镜像组在下次编辑时被要求补录。
- **Created**: 2026-08-24
- **Context**: admin 镜像列表字段补齐后（OPT 同批修复），仍空的两列是数据本身缺失：vendor.company_name（厂商注册未填）与 ai_provider_containerimagegroup.description（创建镜像组未填）在库中即空串。代码层无法补造真实数据。
- **Action**: (1) 厂商申请表单（VendorApplyPanel.vue）与镜像组创建表单加 company_name / 镜像组描述必填校验（前端 + 后端双重）；(2) 对存量 vendor 提供资料补录入口（复用 KYC 证照上传流程或厂商资料编辑）。
- **Why**: 厂商/描述是镜像市场的关键决策信息，空值降低买家信任；源头校验优于事后补录。
- **How to apply**: `taskAiProvider/frontend/src/components/VendorApplyPanel.vue`、镜像组创建流程；后端 `handleVendorApplication`/`CreateImageGroup` 加空值拒绝。

## [OPT-20260825-002] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskFE 8e4b4f2 已推送：登录后未绑手机引导「去验证」时经 savePhoneVerifyRedirect 把原 next/工作面板落点暂存 sessionStorage（与 POST_LOGIN_REDIRECT_STORAGE_KEY 区分，PostLoginReturnUrl.normalize 拒外链）；UserProfilePhoneBindingPanel 绑定/更换成功经 maybeNavigateToPhoneVerifyRedirect 跳回原落点并清除，无暂存则正常发成功文案。回归测：phoneBindingDeepLink 9 例 + prompt service 7 例 + 面板接线 2 例全绿，关联 96 例全绿，pre-commit 全绿。
- **Created**: 2026-08-25
- **Context**: 登录后未绑手机弹窗选「去验证」会到 `/profile/#rg=profile.phone_binding`，原 `next` / 工作面板落点被丢掉；用户绑完停在资料页。
- **Action**: (1) 「去验证」前把原 redirect 写入 sessionStorage；(2) `UserProfilePhoneBindingPanel` 绑定成功后若存在该键则 `location.href` 回去并删除键。
- **Why**: OIDC/充值 next 在引导绑手机后应能继续原意图，否则用户要再找入口。
- **How to apply**: `post_login_phone_verify_prompt_service.js`、`UserProfilePhoneBindingPanel.vue`；键名与 `POST_LOGIN_REDIRECT_STORAGE_KEY` 区分。

## [OPT-20260824-063] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskAiProvider 935dbfd 已推送：镜像列表行「技能/运行说明」摘要——imageResolveInfo.js 新增 summarizeImageResolveInfo 纯函数（ok 返回技能名列表 + 自动运行说明首行摘要，markdownFirstMeaningfulLine 跳过标题/空行/去列表标记；未 ok 复用 skillsStatusHint/autoRunStatusHint，缺省「未提取」）；VendorPortal.vue 版本行新增 version-resolve 摘要区（computed 一次映射所有版本，data-testid=version-resolve-summary，空状态提示未提取）；VendorPortal.css 补样式。回归测 6 例 + 前端全量 122 例绿，vite build 通过。视觉验收待浏览器/精准重启（属部署复验类）。
- **Created**: 2026-08-24
- **Context**: 保存后异步提取已落库（image_skills_json / auto_run_steps_md），但厂商门户镜像列表行与详情仅展示架构/大小，技能与运行说明对后续发布审核、买家选型可见性低。
- **Action**: VendorPortal 镜像行/详情处展示技能名列表与运行说明摘要（可复用 ImageResolveInfoPanel），空状态提示未提取。
- **Why**: 让厂商无需重开弹窗即可核对已保存版本的技能/运行说明；与租户侧 installed image 展示对齐。
- **How to apply**: `taskAiProvider/frontend/src/views/VendorPortal.vue`

## [OPT-20260825-001] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskCloudService 2e75a0d 已推送：无指令闲置回收先探测 clone_done（probeCloneDone 复用 resolveContainerTarget，GET /api/requirements/task-gate），clone_done=false 或探测失败跳过，防长克隆超 idle 窗口被误回收。回归测 3 新例 + 既有 4 例改指向 probe server；整包 200s 绿。
- **Created**: 2026-08-25
- **Context**: 本会话给「克隆完成但从未下发指令」补了 L2：用 `userdata_run_verified + idle_recycle_minutes` 作为就绪时钟。本机克隆曾耗时约 40 分钟，工作区策略仅 5 分钟；若在克隆中途部署该 L2，可能把尚未 BOOTSTRAP_COMPLETE 的机器回收。
- **Action**: (1) recycle 无 `instruction_idle_since` 的候选时，若 `server_url` 可达则 GET `/api/requirements/task-gate` 读 `clone_done`；(2) `clone_done=false` 或探测失败则跳过；(3) 补测：userdata 已过期但 clone_done=false 不释放。
- **Why**: 仅靠 userdata 时间会把长克隆当成闲置；BOOTSTRAP_COMPLETE 打标只覆盖新路径，旧镜像在克隆期仍暴露。
- **How to apply**: `taskCloudService/src/instruction_idle.go` `instructionIdleAnchorTime`；容器 `onlineServiceJS/src/routesConfigJobs.mjs` GET `/requirements/task-gate`；需要容器 token 时走已有 CSC inbound 凭证。

## [OPT-20260818-041] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: gitService a9d258d 已推送：新增 scripts/push_gitlab_image_to_sh.sh（SSOT imageTag → docker save|gzip|ssh sh gunzip|docker load；--check 查远端镜像存在性、--image/GITLAB_IMAGE_OVERRIDE 分步升级、输入白名单防注入）+ 8 例自测全绿；ai.md 升级 checklist 引用脚本。
- **Created**: 2026-08-18
- **Context**: GitLab CE 从 19.0.0 升到 19.2.4 时，Host sh `docker pull registry-1.docker.io` 超时；改由 INFRA `docker save | gzip | ssh sh docker load` 才装上镜像。下次升级仍会卡在拉镜像。
- **Action**: (1) 给 Host sh Docker daemon 配可达的 registry mirror，或 (2) 在 `gitService/ai.md` 的 save/load 步骤写成 `scripts/push_gitlab_image_to_sh.sh` 并在升级 checklist 引用
- **Why**: 3GB+ Omnibus 镜像每次手工管道传输耗时长且易中断。
- **How to apply**: Host sh `/etc/docker/daemon.json`；或 `gitService/scripts/` 新脚本消费 `conf/infra/git-service*/config.yaml` 的 `imageTag`

## [OPT-20260824-087] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: meta b6adde42 已推送：新增 scripts/sync_commercial_email.py（根仓 README/COMMERCIAL 商务邮箱 SSOT，按行模式同步各子仓副本；--check 门禁/--dry-run 预览/--repo 定向）+ 10 例自测全绿；COMMERCIAL.md 增加维护说明并注明 SMTP 发件账号与商务联系相互独立（换邮箱不强制换 SMTP，反之亦然）。
- **Created**: 2026-08-24
- **Context**: 将 README/COMMERCIAL 商务联系从 `riguangyu88@foxmail.com` 改为 `author@example.com` 时，共改 51 个文档副本（根仓 + 各子仓）。`conf/auth/task-auth/email.yaml`、`conf/events/domain-events/email.yaml`、`conf/core/email/config.yaml` 的 `host_user` 仍是 foxmail，那是 SMTP 发件账号，不是商务联系。文档多副本无单源，下次改邮箱仍要扫全仓。
- **Action**: (1) 明确 SMTP `host_user` 是否随商务邮箱一并更换，若否，在 COMMERCIAL 注明发件地址与商务联系不同；(2) 增加根仓 `COMMERCIAL.md`/`README.md` 商务邮箱为 SSOT，子仓由脚本同步，避免 50+ 处手改。
- **Why**: 多副本手改易漏；SMTP 与商务联系混用同一旧邮箱时，对外联系已换、系统邮件仍从 foxmail 发出，合作方会对不上。
- **How to apply**: 先问版权方 SMTP 是否改 189；同步脚本可放 `scripts/` 扫描各子仓 README 第 11 行与 COMMERCIAL「邮件联系」行。

## [OPT-20260824-081] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: 主 GitLab（19.2.4，容器 gitlab 2026-08-24T22:27+08 启动）经 gitlab-rails runner 实测：GitAccess#project 为 private 实例方法（private_instance_methods(true) 含 :project），initializer 已加载（TraeGitlabTrafficQuota 定义），project_path_for/gitlab_username_for 经 respond_to?(sym,true)+send 从私有方法正确提取（mock 实测 tenant-123/demo / example-user）。gitService 10a613431 修复已在运行进程生效，无需再部署。
- **Created**: 2026-08-24
- **Context**: SH GitLab（19.2.4）的 `Gitlab::GitAccess#project/#user` 是 private 方法（git_access.rb:140 起 private 区段），initializer 的 `respond_to?(:project)`/`respond_to?(:user)` 返回 false → 闸门拿不到路径/用户名 → 对一切请求 UNMAPPED fail-open（线上症状：租户组仓与个人仓一律放行）。已修复为 `respond_to?(sym, true)` + `send`（OPT-20260824-079）。主 GitLab（daydaymoney-gitlab 区域，版本可能不同）使用同一 initializer 文件，需确认其 GitAccess 方法可见性 —— 若为 public 则无需改；若同为 private 则主实例此前同样 fail-open。
- **Action**: 在主 GitLab 实例上运行 `gitlab-rails runner "puts Gitlab::GitAccess.private_instance_methods(false).grep(/project|user/).inspect"` 或直接验证 project_path_for/gitlab_username_for 返回值；确认后决定是否推送同款修复。
- **Why**: 同一 initializer 文件跨实例部署，版本差异可能导致闸门静默失效（fail-open 无告警）。
- **How to apply**: 主实例 GitLab 容器（gitService 部署）验证 `GitAccess.private_instance_methods`；若含 :project/:user 则同步 `respond_to?(sym, true)` 修复（gitService/initializers/zzz_trae_gitlab_traffic_quota.rb 已含，仅需部署）。

## [OPT-20260823-033] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskBill 31ead3d 已推送：main.go 挂载 tracelog.MetricsMiddleware（此前仅 taskAuth 有），admin_order_wechat_account_query 等全部路由路径进入 <prefix>_http_request_duration_seconds 直方图；metrics_test.go 增回归测（经中间件请求 /api/health/ 后 /metrics 出现 bucket/count），整包 77s 绿。AiMonitor abee177 已推送：新增 admin-wechat-account-latency 看板（p95 + QPS，job/path 维度，按路径 regex 匹配 taskAuth wechat-linked-account 与 taskBill 微信查单）。nickname 模糊 LIKE 放行与否为产品决策，已迁 PRODUCT_DECISIONS.md（OPT-20260823-033）。
- **Created**: 2026-08-23
- **Context**: 超管「微信关联账号」查单已按等值匹配落地（`wechat_identity.nickname/openid/unionid` + 已绑定登录标识），并加了 nickname 等值索引。若运营后续需要「粘贴一段昵称也能命中」，等值索引不够，且缺 Grafana 面板观察 `admin_order_wechat_account_query` 的 p95。
- **Action**: (1) 评估 nickname 前缀/全文索引与写放大 (2) 为 `wechat_linked_account_lookup` / `admin_order_wechat_account_query` 补 latency 直方图与 Grafana 行 (3) 仅在产品确认允许模糊后才放开 LIKE
- **Why**: 现在严格等值可避免误伤；若查询变慢或运营要模糊，需要可观测性与明确的索引方案，而不是临时改 SQL。
- **How to apply**: `dataMigrate/taskAuth/038_wechat_identity_nickname_index.sql`、`taskAuth/src/auth_wechat_linked_account.go`、`AiMonitor` Grafana 订单管理看板

## [OPT-20260825-007] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: meta 54be4b86 已推送：sonar-project.properties 从 sonar.sources 移除 e2e-tests、sonar.exclusions 加 runAll/src/status_ui/**。复跑 `sonarqube.sh` EXECUTION SUCCESS（2202 源文件），开放 issue 6714 → 5802。QG 仍 ERROR 系 `new_coverage=0%`/`new_security_hotspots_reviewed=0%`（OPT-20260825-004/006）与 2 条新 reliability 问题（taskAiProvider imageResolveInfo.js / taskFE toastService.js，非本改动引入）。
- **Created**: 2026-08-25
- **Context**: 开放 issue 约 6714 条，绝大多数是 CODE_SMELL；`e2e-tests/` 与 `runAll/src/status_ui/` 贡献了大量与产品路径无关的噪音，淹没 Quality Gate 相关的新代码问题。
- **Action**: (1) 在 `sonar-project.properties` 用 `sonar.exclusions` 或独立项目拆出上述目录；(2) 复跑 `sonarqube.sh` 对照 open issue 数量与 QG；(3) 若拆独立项目，给 status_ui 单独 Quality Gate。
- **Why**: 主项目应盯产品代码的漏洞/可靠性，而不是测试夹具与运维面板的 smell。
- **How to apply**: 仓库根 `sonar-project.properties`；扫描入口 `sonarqube.sh`。

## [OPT-20260825-005] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskAuth 6a0d1ef 已推送：未验证手机业务写路径 forward-auth 403 门禁。APISIX forward-auth 子请求固定 GET 但注入 X-Forwarded-Method/Uri（forward-auth.lua 3.11.0 实测），据此对客户业务写路径(POST/PUT/PATCH/DELETE)判定：无已验证 phone login_method 且非员工/超管/模拟登录 → 403 code=phone_verification_required；豁免 members/join|invite|validate-invite（SPA people/join|invite 对未验证豁免）；/api/accounts/ 认证路径天然豁免。响应注入 X-User-Phone-Verified。回归测 6 集成 + 13 单元，taskAuth 全量 127s 绿。
- **Created**: 2026-08-25
- **Context**: 登录后未验证手机已改为前端硬门禁；直打 `/api/tenant/...` 仍可能绕过 SPA。
- **Action**: (1) 在 taskAuth `/me/` 或网关对客户业务写路径增加「已验证 phone login_method」判定；(2) 未验证返回 403 + 明确 error code；(3) 员工/模拟登录/分享 accessCode 豁免；(4) 补回归测。
- **Why**: 仅前端门禁挡不住脚本和旧客户端。
- **How to apply**: `taskAuth` 登录方法查询与 `taskGateway` forward-auth；与 `userHasVerifiedPhone` 谓词对齐。

## [OPT-20260823-045] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: taskBill 7c2dfe8 + taskFE 0bd3d72 已推送：待分账队列表头列过滤——listProfitSharingQueue 收 profitSharingQueueFilter{OrderNumber/TenantID/ReceiverUserID}（order_number LIKE / tenant_id 等值 / receiver_user_id 等值），handler 解析 query + invalid tenant_id 400，profitSharingQueueFromSQL 改返回 fromArg 消除 referrer+status 组合占位符错位；FE SystemAdminProfitSharingPanel 表头第二行 HeaderTextFilter 订单号/租户/接收方，debounce 400ms，重置清空重拉，空态改 tbody 空行使过滤行过滤后仍可见；Go 8 组 DB 回归测 + FE column-filters 5 例 + 系统管理目录 104 例全绿，taskBill 整包 71s 绿。至此 users/refund/invoice/profit-sharing 四表均已复用表头列过滤；剩余 SystemAdminOrderListPanel 已有搜索条+状态 tab 过滤，列过滤低价值留待统一设计（条目内已注记）。
- **Created**: 2026-08-23
- **Context**: `/system-admin/users/` 已按账单表模式在每列标题下加过滤器。订单记录、退款面板等系统管理表格仍只有表头没有列过滤。
- **Action**: (1) 盘点 `SystemAdminOrderRecords.vue`、`SystemAdminRefundPanel.vue` 等管理表格列 (2) 复用 `HeaderTextFilter` / `HeaderSelectFilter` / `HeaderDateFilter` (3) 后端列表 API 补对应 query，补 vitest。
- **Why**: 超管大表分页后无法按列缩小，和用户表体验不一致。
- **How to apply**: `taskFE/app/src/components/HeaderTextFilter.vue`、`SystemAdminUsersFilters.vue` 为参考；对应 taskBill/taskAuth 列表 handler。
- **2026-08-24 夜**: **退款审批面板已落地（部分）** — taskBill `8c8bc0c`（listRefundApplications 改收 refundListFilter{TenantID/OrderID/Status}，system-admin handler 解析 query，3 调用点对齐，新增 6 组 DB 过滤回归测）+ taskFE `20e95c1`（SystemAdminRefundPanel 表头第二行渲染 HeaderTextFilter 租户ID/关联订单，debounce 400ms 自动带 query，重置按钮清空并重拉，onUnmounted 清理 timer，新增 column-filters.test.js 4 例；9 个 refund 测试文件 20 例全绿）。**剩余**：SystemAdminOrderListPanel 已有搜索条 + 状态 tab 过滤（tradeNo/wechatAccount/status），列过滤增量价值低，留待统一设计；其余管理表格（ProfitSharing/Invoice）待后续。
- **2026-08-25 夜**: **开票申请面板已落地（Invoice）** — taskBill `aa43e4d`（listInvoiceApplications 改收 invoiceListFilter{ID/TenantID/OrderID/BuyerName/Status}，handler 解析 id/tenant_id/order_id/buyer_name query，等值 + 抬头 LIKE，OpenAPI 两处补参数，新增 8 组 DB 过滤回归测，整包 75s 绿）+ taskFE `1347102`（SystemAdminInvoicePanel 表头第二行渲染 HeaderTextFilter 申请ID/租户/订单/抬头，debounce 400ms 自动带 query，重置清空重拉，onUnmounted 清理 timer，新增 column-filters.test.js 6 例；全量 vitest 3491 例绿）。**剩余**：SystemAdminOrderListPanel（已有搜索条+状态 tab，列过滤低价值留待统一设计）、ProfitSharing 管理表（待后续）。

## [OPT-20260823-061] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: gitService c65382407 已推送：新增 scripts/deploy_tencent_sh_1_from_infra.sh（SSOT 渲染 conf/infra/git-service-tencent-sh-1/ → .env → scp → docker compose up -d 幂等）+ 18 例单测全绿。dry-run/diff 实测：远端 docker-compose.yml 与 SSOT 一致，.env 存在手工维护漂移（diff 可见）。docker compose config 校验渲染 env 合法（external_url=https://gitlab-tencent-sh-1.daydaymoney.com，ports 8014/2223）。实际首次应用（生产 GitLab 重启）留专门运维窗口。
- **Created**: 2026-08-23
- **Context**: 现网 tencent-sh-1 实例 compose 为手动复制维护（非 git checkout）；本次已将带 traffic gate 的 compose 基线入库 `conf/infra/git-service-tencent-sh-1/docker-compose.yml` 作为 SSOT，但部署仍是手工 scp + compose up。
- **Action**: (1) 编写 deploy 脚本：从 SSOT 渲染（含 .env 区域变量）→ scp → `docker compose up -d` 幂等应用 (2) 记录到 runAll 或 gitService/scripts (3) 后续区域配置变更只改 SSOT
- **Why**: 手动复制必然漂移（本次实例与本地实例差异即如此），闸门/安全补丁会漏投。
- **How to apply**: `conf/infra/git-service-tencent-sh-1/docker-compose.yml`（SSOT）；`gitService/scripts/`（deploy 脚本）

## [OPT-20260822-015] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: ADR-0042：各区域 GitLab workhorse sidecar 解析 git-upload-pack written_bytes，POST charge-gitlab-traffic（project_path+region，禁止 defaultGitlabRegion）。容器 received_bytes 生产路径已删。Live：SH 注入 1MiB 后 tenant 877397588196749312 tencent-sh-1 traffic_used_gb 0.031494→0.032471；同 correlation_id 重放不双计。
- **Created**: 2026-08-22
- **Context**: 租户 `877397588196749312` 容器 clone 路径已于 2026-08-25 落地：`received_bytes` → taskCloudService → `charge-gitlab-traffic` 按实际字节抬高 `traffic_used_gb`（live：`0.010498 GB / 1 GB`）。GitLab workhorse/nginx `git-upload-pack` 全量采集仍未做，笔记本/非容器 clone 仍可能漏计。
- **Action**: (1) 在各区域 GitLab workhorse/nginx 日志解析 `git-upload-pack` 的 `written_bytes`，按租户+region 调用 `reportGitlabTrafficUsageFloor` 或 `POST /api/internal/taskbill/charge-gitlab-traffic/` (2) 扣费请求与 `addGitlabTrafficUsedGBForRegion` 必须带 `region`，禁止写入 `defaultGitlabRegion` (3) 补重放测：同租户同 region 字节只抬高水位；不同 region 不塌缩
- **Why**: 无精确采集时超额后水位不再随 clone 增长（闸门会阻断），但已购流量的租户用量会偏低，也无法按公网出站计费。
- **How to apply**: `taskBill/src/handlers_internal_charge.go`、`chargeGitlabTraffic`、`gitlab_traffic_usage.go`；采集侧建议 `taskEvents` timer 或 GitLab log shipper，禁止业务进程内 ticker

## [OPT-20260825-029] completed

- **Status**: completed
- **Completed**: 2026-08-25
- **Summary**: 公网 gitlab-connection 页 Playwright 连本机 Chrome CDP :9222 验收：snippet 含 /api/oidc/877397588196749312/{authorize,token,userinfo,jwks} 与 discovery: false；无全局 /api/oidc/authorize。直连 taskAuth :18081 租户/全局 authorize 均为 400 redirect_uri required，jwks 均为 200。
- **Created**: 2026-08-25
- **Context**: 租户自建 GitLab 平台 OIDC SSO（ADR-0043 / v110）已合入：taskAuth CRUD + 成员闸门、APISIX 868、taskFE `WorkspaceSettingsGitlabOidcSso`。线上仍须经 9999 精准编译重启并发布新 SPA 后才可见。
- **Action**: (1) 确认 `.runall/precise_restart_services.txt` 含 task-auth、taskFE、task-gateway 并在 http://10.2.150.68:9999/ 点「精准编译重启」 (2) 经 9999 执行 taskAuth `043_oidc_client_tenant_sso.sql`（若尚未应用） (3) 硬刷新 `/tenant/{tid}/settings/gitlab-connection/`，确认 SSO 区块、签发/轮换带 Idempotency-Key、GET 不回明文 secret，**snippet 的 authorization/token/userinfo/jwks 含 `/api/oidc/{tid}/` 且 `discovery: false`**
- **Why**: 未重启时网关仍把 `/api/tenant/*/gitlab-oidc-sso/` 打到 task-tenant-service 863 兜底，签发 404。
- **How to apply**: `scripts/register-precise-restart.sh task-auth taskFE task-gateway`；页面 `WorkspaceSettingsGitlabOidcSso` `data-testid=gitlab-oidc-sso`

