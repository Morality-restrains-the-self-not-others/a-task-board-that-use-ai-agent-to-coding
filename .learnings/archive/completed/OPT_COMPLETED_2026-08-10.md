# Completed OPT Archive — 2026-08-10

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 36 条。
> 归档执行时间：2026-08-13T13:17:38+08:00

## [OPT-20260809-008] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: openapi 契约 GET/DELETE service_provider 改非必填 + 缺省语义注明；回归测试断言参数非必填。commit taskGitOauth 419e4be
- **Created**: 2026-08-09
- **Context**: openapi.go 中 `/api/git-oauth/user-app-connection/` GET 的 `service_provider` 标 `required: true`，但任务详情页检查只传 `repo_url`（sp 缺省 "default"），DELETE 亦然。契约文档与实际调用不一致。
- **Action**: 将 GET/DELETE 的 `service_provider` 改为非必填并在 summary 注明缺省语义，或明确契约要求调用方显式传 provider/service_provider。
- **Why**: 契约与实现不符会误导新调用方（本次缺陷即因调用方未传 sp 且实现不解析配置键导致）。
- **How to apply**: 修改 src/openapi.go 参数定义后跑 `go test ./src/ -run TestOpenAPI`（openapi_path_contract_test.go 会校验契约）。

## [OPT-20260809-009] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 状态检查 github 分支展开全部配置存储键 + 去重；单键接口保留回调写路径。回归测试覆盖多配置展开与去重。commit taskGitOauth cbd80ca
- **Created**: 2026-08-09
- **Context**: `githubStoredProviderKey()` 与 `userAppConnectionLookupKeys` 的 github 分支都只取第一个 github 配置行的存储键；若未来配置多个 github provider（不同 sp），凭据可能存于非 rows[0] 的键。
- **Action**: github 分支改为遍历所有 github 配置行（与 gitlab 分支一致），或明确多配置时 rows[0] 即权威存储键的约定并注释。
- **Why**: 与 gitlab 分支的遍历语义不一致，多配置部署下存在同类漏查风险。
- **How to apply**: 修改 src/internal_handlers.go `userAppConnectionLookupKeys` 的 github case 后跑连接测试回归。

## [OPT-20260809-014] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: taskTaskService 5xx 时 approve 503+detail、status 200 降级；collectTaskGithubRepos 返回 (repos,err)。回归测试覆盖。commit taskCloudService 06ef1dd
- **Created**: 2026-08-09
- **Context**: `collectTaskGithubRepos` 吞掉 collectTaskGitReposForAuthContext 的错误（返回空列表），approve 校验 1 在 taskTaskService 宕机/超时时会以 400 "该仓库不在任务关联项目中" 拒绝，错误消息误导排障（实为下游服务不可达，应 503/降级提示）。
- **Action**: 让 collectTaskGithubRepos 透出错误（或单独区分"收集失败"与"无匹配"），approve 对收集失败返回 503 + detail，status 保持空列表降级（展示层可接受）。
- **Why**: 误导性错误消息在真实故障时拖慢根因定位（TraceId 日志优先原则下应给 5xx 而非业务 400）。
- **How to apply**: 改 collectTaskGithubRepos 签名返回 (repos, err)；approve 分支处理 err → 503；status 分支忽略 err 保持现状；补单测覆盖 taskTaskService 5xx 场景。

## [OPT-20260809-015] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: gitOauth.bridgeSecret 启动比对 SSOT conf/auth/git-oauth/django.yaml ssoJwtSecret；占位符/空回退 SSOT、漂移 WARN。回归测试四路径。commit taskCloudService c720676
- **Created**: 2026-08-09
- **Context**: 本次 OPT-052 端到端被 401 阻断，根因是 conf/taskCloudService/config.yaml 的 gitOauth.bridgeSecret 仍是本地占位符，而 taskGitOauth 实际以 conf/auth/git-oauth/django.yaml 的 ssoJwtSecret 为准。两处手工同步密钥，OPT-20260806-062 轮换时若漏改一边即静默 401。
- **Action**: 启动时校验两侧密钥一致（taskCloudService 启动日志告警比对 GitOauthBridgeSecret 与预期值）或让 taskCloudService 直接从 django.yaml 派生（conf 里用 ${} 引用或加载 auth/git-oauth/django.yaml 的 ssoJwtSecret 作为回退优先级提升）。
- **Why**: 密钥轮换是既有操作（OPT-20260806-062），配置漂移是真实故障源，本次已实际发生。
- **How to apply**: config.go 加载 GitOauthBridgeSecret 时，若 conf 值等于已知本地占位符则 WARN；更优：优先读 auth/git-oauth/django.yaml ssoJwtSecret。

## [OPT-20260809-016] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 启动路径 openDB 后自动 runDataMigrate（幂等 step_key 去重）+ budget 同步；回归测试重复执行不重复应用。commit taskCloudService 442838b
- **Created**: 2026-08-09
- **Context**: runDataMigrate 只在 `./bin/taskCloudService migrate` 时执行，服务启动不自动迁移。本次 014 建表在服务重启后未应用，status 端点 500 才暴露；dataMigrate 子仓另有 pre-push 门禁（推送含未应用迁移阻断）。
- **Action**: 评估启动时自动执行 runDataMigrate（幂等：data_migrate_log step_key 已去重），或部署脚本统一在服务启动前调 migrate（与 taskGitOauth 等其他服务迁移惯例对齐）。
- **Why**: 启动时静默缺表比部署时显式失败更难发现，端到端验证依赖人工记得 migrate。
- **How to apply**: main.go 在 openDB 后无条件调用 runDataMigrate（当前仅在 migrate CLI 分支），全量测试回归验证幂等。

## [OPT-20260809-031] completed
- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: taskTenantService 47b76e0（发布端补 company_name）+ taskEvents 725eb57（消费端容错+永久失败回调 failed）均已提交并推送，tests 全绿。DLT 重放属部署后补偿，见部署项。
- **Created**: 2026-08-09
- **Context**: 线上租户 874176608758427648 邀请 ljy080829@gmail.com 邮件永远显示「投递中」。根因：taskTenantService 发布 INVITATION_CREATED（invite_handlers.go:107 创建 / :362 重发）只带 `company_id` 不带 `company_name`；taskEvents 消费端 handleInvitationCreated email 分支校验 `company_name` 非空 → DispatchPermanent → dead-letter 到 invitation-created-dlt（死信消息 error="invitation email missing fields"，retry_count=1，两条 10:37Z/11:34Z）→ 不触发 SMTP、不回调 delivery-callback → tenant_invitation.delivery_status 永远 queued。
- **Action**: ① 发布端两处补 `company_name`（从 tenant_company 表查 name）；② 消费端容错：company_name 为空时以 company_id 兜底渲染公司名，或降级为仅校验 invitation_url；③ delivery 永久失败路径也应回调 failed 状态（避免「永久卡 queued 假象」）；④ 存量补偿：修复后重放 invitation-created-dlt 中该 key 消息或标记 failed。
- **Why**: 字段契约（company_id vs company_name）不一致破坏异步闭环；permanent 错误无回调导致状态假死，前端无法区分「投递中」与「已死信」。
- **How to apply**: 修发布端两处 + 消费端容错 + 回调失败态 + 补 delivery_test 断言 company_name 缺失时行为。

## [OPT-20260809-028] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 决策落地：无 task 绑定实例排除出头部计数（与指示器同向），taskCloudService f2383a3，含单扫描单元测试（task_id=空 + mock 实例），全量测试通过，已推送。
- **Created**: 2026-08-09
- **Context**: computeWorkspaceMachineSnapshot 统一扫描后，instance_id 非空且 countsAsStarted 的行即使 task_id 为空（如容器绑定在 comment_id 上但任务已删除的残留）仍计入 started 集合 → 头部「已启动 N」可能大于卡片亮起数；当前生产数据无此类行，两端点一致性不受影响。
- **Action**: 评估快照层对无 task 绑定已启动实例是否应排除出头部计数（与指示器同向），或保持现状并在文档中固化该语义；如排除需补单扫描单元测试用例（task_id='' + mock instance）。
- **Why**: 让「头部计数 ⊆ 卡片可见」成立为结构性不变量（当前为防御性语义、未固化）。
- **How to apply**: workspace_machine_snapshot.go 决定 filter 位置（instance 集合或派生层），同步更新 workspace_machine_snapshot_test.go。

## [OPT-20260809-027] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 后端 taskTenantService 12059f6（pending-invitations SELECT/响应带出 email_sent_at，COALESCE 空串）+ 前端 taskFE 9541342（投递时间列展示，无则 -，含 2 断言）。两端测试全绿，已推送。
- **Created**: 2026-08-09
- **Context**: 投递状态闭环（delivery_status 008 迁移）已上线：pending-invitations 返回 delivery_status/delivery_error，任务侧回调已记录 email_sent_at；但 PendingInvitations.vue 未展示投递时间，管理员无法区分「早该送达」与「刚发送」的邀请。
- **Action**: handlePendingInvitations 响应带出 email_sent_at（COALESCE NULL→''），前端状态列旁或「过期时间」后新增投递时间展示（无则 '-'），补 vitest 断言。
- **Why**: 投递时间对排查「投递中」滞留邀请（如 Kafka 堆积、SMTP 队列挂起）有直接诊断价值，字段已存在零成本带出。
- **How to apply**: 后端 SELECT 加 email_sent_at 列 + 响应 map 带出；前端 formatDate 复用；跑 taskTenantService 与 taskFE 相关测试。

## [OPT-20260809-013] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: apiUtils TimeoutError traceId 单测改 vi.useFakeTimers 确定性推进；全量 331 文件 1762 例通过，taskFE 647288a 已推送
- **Created**: 2026-08-09
- **Context**: `utils/apiUtils.test.js` 的 `apiFetch TimeoutError carries request traceId` 在全量 1672 例并发下偶发失败（单跑 14/14 通过），疑似超时定时器在测试线程调度延迟下越界。
- **Action**: 为该用例放大 mock 超时窗口或改 vi.useFakeTimers 确定性推进，消除调度抖动。
- **Why**: 全量回归失败会误报为回归，淹没真实问题。
- **How to apply**: 单测文件内对该用例显式控制定时器后验证全量稳定通过。

## [OPT-20260809-019] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: feature-params 来源可用性标志：后端 company/workspace serialize 随 data 下发 env_var_sources_available（summary 脱敏保留）；前端 view=summary 消费标志 + 个人配置数计算可用态，fail-open 兜底。taskCloudService 1dde22b + taskFE 98db000，已推送；已登记精准编译重启
- **Created**: 2026-08-09
- **Context**: 可用性判定需要 extra_env_vars 的 key 结构，但 view=summary 会把 extra_env_vars 脱敏为 []，前端只得拉 full payload（含 provider API key）计算存在性；个人配置列表另发一请求。
- **Action**: taskCloudService workspace/company GET 响应增加 `env_var_sources_available: { company, workspace }` 标志（key 非空计数 > 0），前端直接消费标志，full payload 拉取可改回 view=summary。
- **Why**: 减少敏感字段下发面与请求体量；可用性口径集中在后端一处。
- **How to apply**: 修改 serializeWorkspaceFeatureParamsData/serializeCompanyFeatureParamsData 后跑 taskCloudService `go test ./src/` + 前端回归。

## [OPT-20260809-018] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: ServerConfig.logic 镜像区 :sources-available 接入，三类来源全空时选择器禁用；7+2 回归测试
- **Created**: 2026-08-09
- **Context**: 本次仅接线 create-task-modal（用户报告的页面）；任务详情镜像区（ServerConfig.logic.vue 经 useServerConfigFeatureParams 使用同一 ServerConfigFeatureParamsBlock）的 sourcesAvailable 仍为默认 true，三类来源全空时该处选择器仍可点，语义不一致。
- **Action**: useServerConfigFeatureParams 增加与 useCreateTaskFeatureParams 相同的可用性拉取（复用 hasAnyFeatureParamsEnvSource），ServerConfig.logic.vue 传入 :sources-available；注意任务详情场景缺 workspace id 时降级公司级 GET。
- **Why**: 同一组件两处入口行为应一致，避免用户在详情区创建了无意义的环境变量绑定。
- **How to apply**: 参照本提交 fetchFeatureParamsSourcesAvailability 模式；改后跑 taskFE 全量单测 + TaskDetail.feature-params-source-switching Playwright 用例。

## [OPT-20260809-022] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: WorkspaceFeatureParamsSettings 内容区改独立滚动容器 max-h-[calc(100dvh-74px)] overflow-y-auto；jsdom 回归测试
- **Created**: 2026-08-09
- **Context**: 公司级 feature-params 设置页（WorkspaceSettingsFeatureParams.vue）已修复「内容区无法滚动」（内容区改为 max-h-[calc(100dvh-74px)] overflow-y-auto 独立滚动容器）；工作空间级页面 WorkspaceFeatureParamsSettings.vue（/tenant/:tenant/settings/workspace/:workspace/feature-params/）根元素同为 `class="p-8"` 无滚动容器，存在同样问题（任务执行人直接滚动该区域无效）。
- **Action**: 对 WorkspaceFeatureParamsSettings.vue 根元素应用同样的滚动容器类；补 jsdom 回归测试（仿 WorkspaceSettingsFeatureParams.scroll.test.js）。
- **Why**: 同构页面保持一致的滚动契约，避免任务执行人在该页再次遇到「内容区域无法滚动」。
- **How to apply**: 改后跑 vitest 相关单测 + 构建验证。

## [OPT-20260809-026] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: WorkspaceFeatureParamsSettings 用户可见功能参数文案统一为智能体资源配置；wording 回归测试
- **Created**: 2026-08-09
- **Context**: 侧边栏菜单（Sidebar.vue:162）与人员管理页 LLM 预算提示链接（MemberList.vue:27）均已使用「智能体资源配置」指代 /settings/feature-params/ 页面；但目标页 WorkspaceFeatureParamsSettings.vue 正文仍残留「功能参数」措辞（:17「个人功能参数配置」、:66「公司尚未配置功能参数」），用户点击进入后文案称谓不一致。
- **Action**: 将 WorkspaceFeatureParamsSettings.vue 中用户可见的「功能参数」措辞改为「智能体资源配置」（保留代码/路由/接口命名 featureParams 不动，仅改展示文案），并核对相关 Playwright 断言。
- **Why**: 同一页面的入口称谓（侧边栏、快捷链接）已统一，目标页正文保持一致避免认知割裂。
- **How to apply**: 全局 grep 用户可见「功能参数」→ 逐一评估改文案；改后跑 vitest + 相关 Playwright。

## [OPT-20260809-012] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: header-summary-row data-traceId 绑定（机器摘要/运行态轮询失败 traceId 落 DOM）
- **Created**: 2026-08-09
- **Context**: 本次修复仅覆盖 header-title-row（工作空间列表加载失败 traceId）。header-summary-row 的机器摘要/运行态轮询（`workspace-machine-summary` / `workspace-runtime-indicators`，15s 轮询）失败时仅 `warnNetworkFailure` 控制台警告，页面降级显示「机器节点：—」无任何 traceId 可查。
- **Action**: useWorkPanelMachineSummary 捕获失败请求 traceId 并注入 headerBind，header-summary-row 绑定 `:data-traceId`（成功后清除），复用本次 workspace-loaded/workspace-load-error 模式。
- **Why**: 机器摘要轮询失败是全站高频静默失败路径之一，Grafana 排查无钥匙。
- **How to apply**: 参照本次 WorkPanelHeader.traceId 提交的 emit+绑定模式，单测覆盖轮询失败/恢复两条路径。

## [OPT-20260809-030] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: warnNetworkFailure 模块级去抖层，(context,错误摘要) 窗口 30s 只告警一次，异错误透传；5 例回归
- **Created**: 2026-08-09
- **Context**: 网络瞬时故障持续数分钟时，refreshMachineSummary 每 15s 调用两次 warnNetworkFailure，控制台/指标被同类告警刷屏；失败本身已由 pair-atomic 保留旧快照兜底，告警只需状态变化时触发。
- **Action**: warnNetworkFailure 包装为按 (endpoint, error-message 摘要) 去抖（如首次失败 + 恢复事件），或轮询侧仅记录 firstFailureAt 并在连续失败时降频。
- **Why**: 可观测性硬门禁要求关键路径埋点，但同信号高频重复产生噪音，掩盖真正状态变化。
- **How to apply**: workPanelApiUtils.js 增加简单去抖层（模块级 Map），补单测断言同错误去抖/异错误透传。

## [OPT-20260809-029] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 已随 taskFE 6894968 提交并推送；pair 测试 14 例 + header 测试 10 例全绿；head 摘要陈旧角标（staleSince 阈值 3 轮 / 恢复即清除）
- **Created**: 2026-08-09
- **Context**: pair-atomic 刷新修复后，任一侧持续失败时头部与卡片保留上一对快照（一致性正确），但 UI 无任何「数据可能过期」信号，用户无法区分实时与陈旧状态。
- **Action**: refreshMachineSummary 记录 lastSuccessAt/连续失败次数，超过阈值（如 3 轮 / 45s）时在 headerBind 处展示轻微陈旧标记（灰色角标或时间戳提示），恢复成功即清除。
- **Why**: 一致性修复的正确性收益需要可观测性配合，否则陈旧展示会被误认为实时。
- **How to apply**: useWorkPanelMachineSummary.js 增加 staleSince ref + 派生 computed，TaskPanel.vue 渲染条件标记；补 pair 测试断言阈值翻转。

## [OPT-20260809-011] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: ccb 启动阶段事件持久化完成：cloud_comment_container_binding_logs 表 + 各调度节点写阶段行 + list API logs 字段 + 前端 refreshBindings 合并去重
- **Created**: 2026-08-09
- **Context**: OPT-20260809-010 前端阶段日志为客户端派生，页面加载前的历史阶段（排队/启动/分配）无法重建——冷打开时已 running 的 binding 仅显示「正在启动容器实例 + 已分配」等当前可观察行，早期阶段缺失。
- **Action**: taskCloudService 为 comment container binding 增加启动阶段事件持久化（如 cloud_comment_container_binding_logs 表，ccbStartBinding/ensure/attach/promote 各节点写阶段行），list API 返回 logs 字段，前端 refreshBindings 优先合并后端日志（去重后与本地 SSE 派生行共存）。
- **Why**: 服务端日志为权威时间线，冷打开/换设备可见完整启动过程；本地派生仅覆盖页面打开后的窗口。
- **How to apply**: 参考 task 级 statusLogs 的 pushServerStatusLogLines 模式；改后跑 taskCloudService `go test ./src/` 回归 + taskFE useCommentContainerBindings.test.js。

## [OPT-20260809-025] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: PeopleInvite/PeopleManage/PeopleGroups 增加 member:manage/group:members:manage 权限探针，无权展示「无权限访问」空态；补 6 个 jsdom 单测。taskFE c109605
- **Created**: 2026-08-09
- **Context**: 侧边栏位置 2 菜单已按「公司租户」条件改为人员管理（Sidebar.vue，isCompanyTenant = member_is_active/admin/creator 任一），非管理员成员也能看到「人员管理」及子菜单（邀请人/管理人员/管理分组）；但 /people/ 各页面后端按 RBAC 权限码（member:manage / group:members:manage）强制，非管理员点击子页面会触发 403 错误态。
- **Action**: 对 PeopleInvite/PeopleManage/PeopleGroups 页面补充非管理员友好提示（复用 companies/current 的 member_is_admin 判定，页面加载时先探权限，无权时展示「无权限」空态而非原始 403）；或在侧边栏按 isCompanyAdmin 进一步控制子菜单项显示（顶层入口保留）。
- **Why**: 菜单可见性已按产品需求放宽到全体公司租户，但子页面权限仍是管理员向；避免普通成员看到菜单却落入错误页的割裂体验。
- **How to apply**: 选定方案后补 jsdom 单测（仿 TenantCompanySettings canEdit 模式）+ 浏览器验证两种角色视图。

## [OPT-20260809-020] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 新增 WorkPanel.create-task-feature-params-unavailable Playwright 用例（mock 驱动），断言 create-task-modal 环境变量选择器 disabled 与 unavailable hint 可见；taskFE 7090a70 已推送
- **Created**: 2026-08-09
- **Context**: 禁用态目前仅 jsdom 集成测试覆盖（mock apiFetch）；无真实浏览器用例验证「无任何环境变量的租户打开创建弹窗 → 选择器 disabled + 提示文案」。
- **Action**: 增加 WorkPanel.create-task-feature-params-unavailable Playwright 用例，前置条件为测试租户无公司/工作空间/个人环境变量（fixture 或测试后清理），断言 selector disabled 与 unavailable hint 可见。
- **Why**: 真实浏览器下 select disabled 属性 + 提示文案的端到端保障（含网关转发路径）。
- **How to apply**: 参照 WorkPanel.create-task-with-projects.playwright.test.js 结构；改后在有凭据环境跑通。

## [OPT-20260809-021] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 全站「功能参数」UI 文案统一为「智能体资源配置」。taskFE 57c73a1 已提交并推送。改动：PersonalFeatureParamsConfigs h2 / TaskFeatureParamsSelector / TaskFeatureParamsSnapshotPanel / UserCenterSidebar / LlmBudgetModal / 任务详情 composable 错误文案 / relayToTraeUtils 阶段标签；同步 FeatureParamsHierarchy e2e 断言 + relayToTraeUtils.test + 新增 PersonalFeatureParamsConfigs.wording.test 回归。全量单测 339 文件/1809 用例绿。遗留：ServerConfigFeatureParamsBlock.vue:26 旧措辞在他人 WIP 内未代改；MemberList.vue 他人 WIP 已改。
- **Created**: 2026-08-09
- **Context**: task-panel 设置页按钮文案已改（本任务，taskFE 6f781fa），但 feature-params 生态其余 UI 仍为「功能参数」：WorkspaceFeatureParamsSettings.vue 页标题「功能参数设置」、PersonalFeatureParamsConfigs.vue「我的功能参数配置」、MemberList.vue、TaskFeatureParamsSelector.vue、TaskFeatureParamsSnapshotPanel.vue 等；Sidebar.vue 与 ServerConfigFeatureParamsBlock.vue 已用「智能体资源配置」，同页两套术语并存。
- **Action**: 待确认改名范围后，将上述视图/组件文案及相关测试断言（FeatureParamsHierarchy.e2e.test.js 断言 h2「功能参数设置」、pageTitle「功能参数配置」等）统一为「智能体资源配置」。
- **Why**: 产品渐进式更名，避免「侧栏叫智能体资源配置、打开的页面叫功能参数设置」的术语割裂。
- **How to apply**: 逐文件替换后跑 vitest 相关单测 + 受影响 Playwright 断言同步更新（FeatureParamsHierarchy 等）。

## [OPT-20260809-023] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 导航栏 74px 魔法数字收敛为 --app-navbar-h CSS 变量（taskFE ad54111 已提交并推送）。styles.css :root 定义 --app-navbar-h:74px（注释注明同步源）；两处滚动容器改用 max-h-[calc(100dvh-var(--app-navbar-h))]；scroll 测试断言同步。全量单测 339/1809 绿；vite build 产物实测编译为 calc(100dvh - var(--app-navbar-h))。因 App.vue 有他会话未提交布局重构，改在全局 CSS :root 定义变量（同等的单点维护目标），移动端换行策略按 OPT 另行评估未纳入。
- **Created**: 2026-08-09
- **Context**: 滚动容器 max-h 硬编码 74px（Navbar.logic.vue 实测高度，py-4 + 内容行高）；375px 宽下导航栏换行为 347px，74px 假设失效（内容区会超出视口产生文档级滚动，功能可用但观感不佳）。侧栏高度（780px 视口下 564px > 内容区 363px）也会使文档出现少量滚动。
- **Action**: 在 App.vue 定义 CSS 变量 --app-navbar-h（实测 74px，注释注明同步源）并让滚动容器使用 calc(100dvh - var(--app-navbar-h))；移动端（<md）另行评估导航栏换行下的布局策略（sticky 顶部栏 + 内容区滚动）。
- **Why**: 消除魔法数字，导航栏改高时单点维护；移动端场景不被 74px 假设破坏。
- **How to apply**: App.vue 根元素 style 绑定变量 → 各页滚动容器引用；改后构建 + Playwright 多视口回归。

## [OPT-20260810-005] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: kill-first 重构完成：stopProcess 组消亡等待（SIGTERM→轮询组→SIGKILL）、restartService 改序（stop→build→start）、startAndCheck forceFreshStart 不跳过启动、terminateListenersByPort 排除自身 PID；7 单测新增/重写全绿（101.4s ok）；真机端到端 task-auth PID 3011828→3013039 顺序日志 SIGTERM→stopped→building→healthy 验证
- **Created**: 2026-08-10
- **Context**: 12:15:40 精准编译重启 task-events-email-sent-1-send-email（18022）：build succeeded 后 stopProcess 发 SIGTERM，bash 组长立即退出使 `cmd.Wait()` 快速返回并打 "stopped for restart"，5s 后 SIGKILL 升级被短路；实际服务进程 2410097 优雅关闭中仍监听且健康 → startAndCheck [runner.go:595](runAll/src/runner.go#L595) 判定「端口已监听且健康，跳过启动」，新二进制（mtime 12:15 已编译）未启动；20s 后旧进程退出 → 端口空 → liveness failed，auto-restart 因 no process handle 跳过 → 服务宕机（当前 UI failed，18022 HTTP 000）。另一场景：runAll 热替换收养进程（[runner_adopt.go](runAll/src/runner_adopt.go) 只回填 PID 不登记 cmd 句柄）时 stopProcess 直接 return false，若 stop 命令也失败则旧副本持续监听、新代码永不生效。
- **Action**: 1) stopProcess 改为等待进程组消亡/端口释放而非 `cmd.Wait()` 组长退出（SIGTERM → 轮询组内存活 → 超时 SIGKILL → 确认端口释放）；2) restartService 场景下 startAndCheck 跳过前校验：刚构建过新二进制且端口监听进程无匹配 cmd 句柄（或 PID 非新启动）时不得跳过，走 waitForPortFree 后强制启动；3) failed 状态清理死 PID（当前 store 仍挂 2410097）；4) failed 后 auto-restart 支持按已构建二进制重新拉起。
- **Why**: 「跳过启动」语义在 restart 场景下 = 停止未完成，新代码不生效；两种结局（宕机 or 旧副本残留）均不可接受。
- **How to apply**: 修改 [runner.go](runAll/src/runner.go) stopProcess/startAndCheck；回归：复现 12:15 时序（bash 包装 start_command + 优雅关闭服务）跑 runner 单测 + 真机精准重启验证。

## [OPT-20260810-001] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 实况验证：kafka_cleanup_junk_topics.py 只读扫描 junk=0（3071 个已清理，metadata 3131→61），list_topics/删除重建正常；治理双保险（random_test_runner 排除 third_party + KAFKA_SKIP_NETTEST=1）已随 OPT-20260810-003 完成提交
- **Created**: 2026-08-10
- **Context**: Kafka 集群积累 3071 个 `kafka-go-<16hex>` 前缀 topic（Go kafka 客户端事务管理 topic 历史遗留），总计 3131 个 topics。metadata 过大影响 list_topics 可靠性/速度，并放大 topic 删除-重建窗口的阻塞风险（清空数据库时实测 task-created 等 topic 删除被阻塞）。
- **Action**: 用 AdminClient.delete_topics 批量清理 `kafka-go-*` 前缀 topics（保留业务 topics + DLT + __consumer_offsets），可在维护窗口执行；执行前确认无活跃 Go 事务性生产者（否则删除被连接阻塞）。
- **Why**: 3131 topics 使 controller 元数据与删除管理压力增大，本次 Kafka 重建超时与 metadata 规模相关。
- **How to apply**: `python3 db/_infra/kafka_recreate.py` 旁路脚本或 kafka CLI 批量删除 `kafka-go-*`；清理后复查 list_topics 耗时。

## [OPT-20260810-002] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 核心由 OPT-20260810-005（kill-first 重构：stopProcess 进程组 SIGTERM→轮询→SIGKILL、ensureServiceNotReachable 端口监听者强杀、forceFreshStart 不跳过启动）闭环；本次收尾：kafka_recreate.py 删除超时新增 _report_kafka_connectors() 阻塞者诊断（ss/lsof 枚举持有 9092 连接进程），6 单测全绿（新增 4 个诊断用例）
- **Created**: 2026-08-10
- **Context**: 本次「清空全部数据库」Kafka 部分失败的根因：3 个历史遗留进程（task-events-user-created-2-sync-user-profile 3 天 18h、task-events-task-status-changed-2-fanout-work-panel-sse 12h46m、taskCloudService 7h31m，均 ppid=1 孤儿）在 runAll 状态显示 stopped 后仍存活，持有 Kafka 活跃 fetch/produce 连接 → Kafka 无限期推迟这些 topic 的真正删除 → 重建报 TOPIC_ALREADY_EXISTS 且 180s 无法创建。runAll 的 stop 流程未核验进程退出（或未处理 kill -9 失败的场景）。
- **Action**: 评估在 runAll stop_all / clear-databases 流程中增加「进程退出确认 + 超时 SIGKILL」；或 clear-databases 前对持有 Kafka/MySQL 连接的遗留进程做强杀。增加回归测试：清空数据库后断言无残留进程持有 db/kafka 端口连接。
- **Why**: 事件消费者/云服务僵尸进程会阻塞 Kafka topic 删除与数据清空，导致 clear-db status=partial 而非 ok。
- **How to apply**: 在 runAll src 的 stop 路径（runner.go StopAll 相关）加退出等待；ClearDatabases 前执行 lsof/ss 检查并告警或强杀。

## [OPT-20260810-004] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: nightly_test_sweep.py 报告阶段接入 scripts/lib/test_resource_cleanup.sh 只读扫描（约束 44 六类资源）：退出码 2 → 报告「⚠️ 发现测试残留」+ 残留清单 + 清理命令提示；0 → ✅ 无残留；超时/异常降级记录。实机验证 exit=0 路径（无残留）与退出码语义；夜间不自动 --fix（托管服务窗口内运行，防误杀）
- **Created**: 2026-08-10
- **Context**: 约束 44 与 `scripts/lib/test_resource_cleanup.sh`（六类资源扫描：Kafka junk topics/孤儿进程/SQLite 残留/大文件/MySQL/Redis/容器端口）已建立并实机验证（发现并清理了 tmp/oauth-verify-profile 7.8M + tmp/wp-before.trace.json 229M 两个 Playwright 测试残留）。当前扫描仅支持人工触发，会话结束时依赖开发者自觉运行；孤儿进程判定依赖 runAll.yaml 服务名白名单，runAll 配置变更（服务退役）后旧进程会正确被识别为残留，但不会自动告警。
- **Action**: 1) 将 `test_resource_cleanup.sh` 扫描纳入夜间巡检（nightly_test_sweep.py 或独立 cron 每早执行一次，残留非零即输出告警到巡检日志）；2) 评估在 runAll UI 增加「测试残留」只读面板（复用扫描输出）；3) 把「测试后必跑扫描」写入测试套件收尾（如 nightly sweep 结束时自动跑）。
- **Why**: 3071 个 kafka-go-* topics 是数月无监控积累的结果；定期自动扫描能在残留变多前暴露问题，避免「清理成本随残留量线性增长」。
- **How to apply**: nightly-test-sweep.sh 末尾追加 `bash scripts/lib/test_resource_cleanup.sh`（退出码 2 即记录残留清单）；runAll UI 增加只读端点复用同一脚本输出。

## [OPT-20260810-003] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 监控闭环：1) list_topics 基线——清理前 3131 topics（junk 3071 拖累），现 61 topics，junk 扫描秒级完成；2) kafka-go-* 计数监控已随 OPT-20260810-004 纳入夜间巡检（test_resource_cleanup.sh [1/6] junk 扫描，残留非零即告警到报告，退出码 2 记录清单），增量>0 时治理命令已内嵌报告；3) 一个月后（2026-09-10 前后）复查 topic 总数防回潮——标注人工项
- **Created**: 2026-08-10
- **Context**: 已删除 3071 个 `kafka-go-*` junk topics（metadata 3131→61），并根治了随机单测抽测运行 vendored kafka-go 集成测试的路径（collect-go 排除 third_party/ + KAFKA_SKIP_NETTEST=1 双保险 + 约束 43 固化）。仍需验证：1) 清理后 Kafka 性能/可靠性改善（list_topics 耗时、删除-重建窗口）；2) 未来是否还会出现 kafka-go-* 增量（如有人手动跑库测试或 CI 环境变量覆盖）。
- **Action**: 1) 对比清理前后 list_topics 耗时（应在日志/监控记录基线）；2) 在夜间巡检或 runAll 观察面板增加「kafka-go-* topic 计数」指标，增量 >0 即告警，触发 `python3 db/_infra/kafka_cleanup_junk_topics.py` 治理；3) 一个月后复查集群 topic 总数确认无回潮。
- **Why**: 3071 个残留是数月积累，只有持续监控才能确认根治生效；metadata 规模直接影响 Kafka 删除/创建可靠性（本会话实测的删除阻塞与 metadata 规模相关）。
- **How to apply**: 巡检脚本或 Grafana 面板加计数器；检查是否有其他路径（非 random_test_runner）直接运行 third_party 测试。

## [OPT-20260810-006] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 实机核对（2026-08-10 13:5x）：13:10 那 14 个无主进程已不存在（可能已被治理/daemon 重启覆盖）；当前 6 个 ppid=1 事件消费者（task-events-task-completed/status-changed×2/post-renewed/post-expired/post-expiry-scan）实为 daemon 通过 bash 包装启动的托管进程——env 含 RUNALL_LOG_ROOT/OTEL_SERVICE_NAME 注入特征、cwd→taskEvents、status API healthy 认领、Kafka 连接活跃、test_resource_cleanup.sh [2/6] 白名单扫描判定「无孤儿进程」。治理评估：端口被无主进程占用的场景已由 OPT-20260810-005 闭环（stopProcess 组强杀 + ensureServiceNotReachable 的 terminateListenersByPort 兜底 + forceFreshStart 不跳过启动）。观察项（非阻塞）：status API pid 字段为 cmd 句柄 PID（bash 包装已退出）与实际服务 PID 不一致，纯展示问题不影响健康判定
- **Created**: 2026-08-10
- **Context**: 13:08 重启 runAll daemon 后（13:09:44 start task-auth 触发 docker-mysql healthy），13:10:00-13:10:52 出现 14 个业务服务进程（taskBill/taskReferral/taskGitOauth/taskAiProvider/taskAgentSupport/taskAIEndPoint/taskContainerGateway/taskProjectService/taskTenantService/taskTaskService/taskCloudService/taskCredentialService/taskAIComment + vue-frontend vite preview），全部 ppid=1（孤儿）、env 无 runAll 注入特征（INFRA_HOST/OTEL_SERVICE_NAME/RUNALL_LOG_ROOT 均缺）、daemon 日志零记录（非当前 daemon 启动）、status API 显示 idle（不认领）。启动者疑似已退出的 runAll 系实例（stdout→logs/<svc>.log 与 cwd→子仓为 runAll 风格）。特征与 12:15 缺陷的「进程不受管理」面一致（无 cmd 句柄、stopProcess 无法定位、端口兜底/stop_command 是唯一治理手段）。
- **Action**: 1) 人工核对这些进程是否承载业务（ps/端口/lsof），确认无主后 `kill` 释放端口，使 runAll「全部启动」可正常接管；2) 排查 13:10 启动者的身份（共享机 10.2.150.68 其它会话/残留实例？）防止复发；3) 评估：runAll 对「端口被无主进程占用」场景的启动治理（当前 skip-start 收养 vs 强制接管）是否符合预期。
- **Why**: 无主进程占着 8004 等托管端口，daemon「全部启动」时端口冲突、状态混乱；且其不受 runAll 生命周期管理（无法优雅停止/健康监控）。
- **How to apply**: `ps -eo pid,ppid,cmd | grep -E 'bin/task|vite preview'` 核对；`lsof -i :8004` 等确认端口；确认无主后 kill；runAll UI 观察端口释放与重新启动。

## [OPT-20260809-017] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 生产核对完成：docker-mysql（UTC 时区）实测 task_auth.auth_system_feature_policy id=1 行 = enable_email_register=0 / enable_wechat_login=1 / allowed_phone_country_codes=["+86"]，与产品新默认完全一致，且 created_at=updated_at（2026-08-10 02:57:13 UTC = 本地 10:57，data_migrate_log 026_system_feature_policy.sql 同期 02:57:14 应用）——库在 10:57 被清空重建，026 种子以新默认创建该行，夜间复查记录的旧行（codes=[]）已不存在，无需人工保存/UPDATE。代码侧全部落库：taskAuth COALESCE（98b7291 已推送）+ dataMigrate 026 种子（a891f10）+ taskFE 首帧占位（47a4e62）
- **Created**: 2026-08-09
- **Context**: 产品调整「邮箱注册默认关闭、微信登录默认开启、手机号区域默认仅 +86」已落到 dataMigrate 026 种子 + taskAuth COALESCE 兜底 + taskFE 首帧占位。data_migrate_log 按文件名去重，026 只在每库首次运行执行，已存在策略行的线上库不会自动获得新默认（显式管理配置不被覆盖，语义正确）。
  - **Verified 2026-08-10**: taskAuth COALESCE 兜底已提交并推送（taskAuth `98b7291`，全量回归 113s 绿）；taskFE 首帧占位改动仍在 WIP（SystemAdminLoginPaymentPolicy.vue，未提交，属他会话脏 WIP 不代提）。生产 `task_auth.auth_system_feature_policy` id=1 实测：`enable_email_register=0`、`enable_wechat_login=1`（与默认一致），但 `allowed_phone_country_codes='[]'` ≠ 新默认 `["+86"]`；该行 `updated_at` 仅比 `created_at` 晚 5 分钟（06:25:15 vs 06:20:37），疑似管理员显式保存「允许所有区域」，**未擅自改库**，需人工确认后经管理页「保存策略」落库。
  - **Verified 2026-08-10（夜间复查）**: 补充核查 git 实际状态——dataMigrate `taskAuth/026_system_feature_policy.sql` 的默认值改动（email=0/wechat=1/codes='["+86"]'，含「2026-08-09 产品调整」注释）**仍为未提交 WIP**（working tree 修改，HEAD 63b6719 未含），与上方「已落到 dataMigrate 026 种子」的表述不符；该文件与 taskFE 首帧占位同属 OPT-20260809-017 进行中的他会话 WIP，**不代提**。即本条目代码侧现状：taskAuth COALESCE 已提交推送，dataMigrate 026 种子 + taskFE 首帧占位均在他人 WIP，仅剩人工生产核对一项。
  - **Verified 2026-08-10（13:5x）**: 代码侧已全部落库——dataMigrate 026 种子已提交（`a891f10`「026 默认策略种子对齐产品新默认（邮箱注册关/微信登录开/区域仅+86，OPT-20260809-017）」），taskFE 首帧占位已提交（`47a4e62`，SystemAdminLoginPaymentPolicy.vue 工作区干净）。本条目仅剩人工生产核对（Action）一项。
- **Action**: 上线后人工核对生产 auth_system_feature_policy 行（SELECT * FROM auth_system_feature_policy WHERE id=1）；若确认无显式配置需应用新默认，经系统管理页「保存策略」落库（优先）或执行等价 UPDATE。
- **Why**: 新默认值对既有部署是「不生效的期望」，不核对会误判发布失败。
- **How to apply**: 部署 checklist 增加一条：比对 id=1 行与默认值 (email=0, wechat=1, codes='["+86"]')，差异先经管理页保存后再验收页面三项显示。

## [OPT-20260810-029] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: Archi CLI 用 -application com.archimatetool.commandline.app + xvfb-run 成功 Loaded model: v70 diff/full
- **Created**: 2026-08-10
- **Context**: 本迭代已写 v70 `.diff.archimate` / `.full.archimate`，但 `Archi --loadModel` 在无显示环境失败/挂起，未拿到 `Loaded model:` 成功判据。
- **Action**: (1) 用可用显示或稳定 xvfb 配置重跑 Archi `--loadModel` 对 v70 双文件；(2) 若 XML 结构不合规则按 v66 样板修正；(3) 将验证结果记入 VERSION_HISTORY。
- **Why**: 架构制品门禁要求双 ArchiMate 文件通过 loadModel。
- **How to apply**: `docs/architecture/v70-application-integration-20260810-1955-claude.*.archimate`、`/home/ljy/Archi/Archi`。

## [OPT-20260810-047] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: 从 package.json build 移除 collectstatic；脚本改为无声 no-op；test/ai.md/.ai 前端规则已对齐 Django 退役
- **Created**: 2026-08-10
- **Context**: `package.json` 已恢复串联 `collectstatic-after-vite.sh`；但 `test_collectstatic_after_vite.sh` 仍要求存在 `task2app/`，而脚本本身在 Django 退役后会 skip，导致自测失败。
- **Action**: (1) 更新 `test_collectstatic_after_vite.sh` 接受「无 task2app → skip 成功」；(2) 同步 `taskFE/app/ai.md` 成功判据，去掉对 `collected_static/main-*.js` 的硬依赖。
- **Why**: 避免 Agent/CI 被过时 Django 路径误判为构建失败。
- **How to apply**: `taskFE/app/scripts/test_collectstatic_after_vite.sh`、`taskFE/app/ai.md`。

## [OPT-20260810-009] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: test_collectstatic_after_vite.sh 改为断言 build 不调用 collectstatic 且退役脚本静默 exit 0
- **Created**: 2026-08-10
- **Context**: `taskFE/app/scripts/test_collectstatic_after_vite.sh` 断言 `task2app_root` basename 为 task2app、activate_env.sh 存在、package.json build 必须引用 collectstatic-after-vite.sh —— 均为 Django 时代约束；task2app/ 已退役（2026-07-30，collectstatic-after-vite.sh 自身已内置 task2app 缺失短路径），测试持续 FAIL（本次会话验证前即失败，非本次改动引入）。package.json 当前 build 仅 `npm run build:vite`，collectstatic 为幂等空转（task2app 不存在时 exit 0）。
- **Action**: 更新该测试：删除 task2app_root/activate_env.sh 断言，package.json 断言改为「build 必须包含 build:vite 且不破坏 dist 清理」，保留 SKIP_COLLECTSTATIC 短路与 bash -n 检查。
- **Why**: 退役组件遗留断言使回归测试恒红，掩盖真实回归。
- **How to apply**: 修改后跑 `bash scripts/test_collectstatic_after_vite.sh` 确认 Green。

## [OPT-20260810-014] completed

- **Status**: completed
- **Completed**: 2026-08-10
- **Summary**: ai.md 与 00_frontend_development 已改为纯 Vite build；collectstatic 不再串联
- **Created**: 2026-08-10
- **Context**: `taskFE/app/ai.md` 写明 `npm run build` 已串联 Vite + `collectstatic-after-vite.sh`，但 `package.json` 的 `build` 仅调用 `build:vite`；`runall-lifecycle.sh build` 亦未调用 collectstatic。Django 已退役时脚本会 skip，文档仍易误导。
- **Action**: (1) 更新 `taskFE/app/ai.md` / 上级 companion 描述真实构建链；或 (2) 让 `build`/`runall-lifecycle.sh` 显式调用 `collectstatic-after-vite.sh` 并保持 skip 逻辑。
- **Why**: 避免 Agent/人工只跑 vite 以为已完成公网静态同步，或误以为已串联 collectstatic。
- **How to apply**: 改 `taskFE/app/package.json` 与 `taskFE/app/scripts/runall-lifecycle.sh`，或只改正文 companion；择一并跑 `scripts/test_build_clean_dist.sh`。

## [OPT-20260810-007] completed

- **Status**: completed（2026-08-10 修复并部署，回归测试已入库）
- **Created**: 2026-08-10
- **Context**: 线上 https://www.daydaymoney.com/auth/login/ 手机号/密码模式登录按钮恒禁用。根因：LoginPhonePasswordFields.vue 模板 4 处使用 emit(...)（@input/@change/@click），但 script setup 仅裸调用 defineEmits([...]) 未接收返回值 → 编译后 _ctx.emit 为 undefined → 每次交互抛 TypeError: i.emit is not a function（生产控制台 24 次）→ 父组件 phoneNationalPassword ref 永不更新 → isPhoneValidForPassword 恒 false → 按钮永远「请填写正确手机号」。
  - **修复**：`const emit = defineEmits([...])`（commit a0368c4）。
  - **回归测试**：app/src/tests/components/LoginPhonePasswordFields.test.js（4 用例：手机号输入/粘贴、区号切换、密码显示按钮事件透传），修复前 Red（_ctx.emit is not a function 与线上同错）、修复后 Green。
  - **部署验证**：vite build 后线上 4000 端口即时生效（index-iGyVDuB_.js）；浏览器实测手机号+密码+协议全流程，按钮启用并成功提交 POST /api/auth/（400 为测试账号未注册的预期后端响应），控制台无 TypeError。
- **Action**: 后续若新增 .vue 组件模板使用 emit(...)，必须 `const emit = defineEmits([...])` 接收返回值（或改用 $emit）；建议在 pre-commit 或 nightly sweep 加「模板裸 emit 无 const emit 声明」静态扫描（grep 模式：`@[a-z-]+="[^"]*emit\(` 且文件无 `const emit =`）。
- **Why**: 编译期不报错、运行时才爆，且仅有交互路径才暴露 —— 极易漏测。
- **How to apply**: 新增/重构 auth 类表单组件时对照本条目自查；同类扫描命令见会话记录（grep -rE '@[a-z-]+="[^"]*emit\(' + 检查 const emit）。

## [OPT-20260810-008] completed

- **Status**: completed（2026-08-10 已实现并验证，回归测试入库）
- **Created**: 2026-08-10
- **Context**: runAll 对 taskFE（taskFE）的 build_command 为 `bash scripts/runall-lifecycle.sh build`（working_dir: taskFE/app），产物为 `app/dist/`。虽 vite.config.js 已设 `emptyOutDir: true`，但构建中断/失败时 vite 清空逻辑可能未执行，旧 hash 文件会残留被 vite preview / nginx 误服务（与 OPT-20260807-056 chunk 404 同源隐患）。
- **Action**: 在两条编译入口显式先清 dist 再构建：
  - `taskFE/app/scripts/runall-lifecycle.sh` build 分支：`rm -rf dist` 先于 `npm run build`（runAll 精准编译重启/全量构建入口）。
  - `taskFE/app/package.json` build:vite / build:assets：`rm -rf dist && vite build`（直接 npm run build 入口，与 dev:fresh 的 rm -rf 风格一致）。
  - 回归测试：`taskFE/app/scripts/test_build_clean_dist.sh`（3 断言：runall-lifecycle build 分支 rm 先于 build、package.json build:vite 含 rm、实测 rm -rf dist 生效）。
- **Why**: 保证每次编译部署都是全新产物，杜绝旧 hash chunk 残留导致的白屏/404 与磁盘膨胀。
- **How to apply**: 后续 taskFE 部署走 runAll「精准编译重启」或 `npm run build` 均自动先清旧版本；改动构建脚本后跑 `bash scripts/test_build_clean_dist.sh`。

## [OPT-20260810-030] completed

- **Status**: completed（代码已部署，待充值+发码）
- **Created**: 2026-08-10
- **Context**: 短信失败原因已透出到 `detail`（含 `isv.AMOUNT_NOT_ENOUGH`）；taskAuth 已发布。发送成功仍依赖阿里云余额。
- **Action**: (1) 登录页发码，错误含「账户余额不足」/错误码/流水号且有 `data-traceId`；(2) 运维充值后再测发送成功。
- **Why**: 未充值则只能验详细错误文案，无法验完整发码链路。
- **How to apply**: `taskAuth/src/sms_user_error.go`、`sms_cloud.go`、`verification_code.go`。
- **Related**: 见 [B](#b-运维与破坏性操作) 充值依赖

---

## [OPT-20260810-049] completed

- **Status**: completed（已转出，待产品决策）
- **Created**: 2026-08-10
- **Context**: 多过滤栏时过滤区自滚 vs 看板贴底，UX 未拍板。
- **Action**: 决策后回迁执行。见 PRODUCT_DECISIONS.md § OPT-20260810-049。
- **Why**: 指针防重复登记。
- **How to apply**: PRODUCT_DECISIONS.md。

