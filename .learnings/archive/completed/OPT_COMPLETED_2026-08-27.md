# Completed OPT Archive — 2026-08-27

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 29 条。
> 归档执行时间：2026-08-28T02:10:47+08:00

## [OPT-20260826-018] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskAuth 29786fc 已推送：未登录回跳 next 改相对路径 r.URL.RequestURI()（/api/oidc[/{tid}]/authorize?...），登录页 sanitizeOidcResumeNext 已接受并拼回 gateway；更新既有测试断言 + 补租户路径未登录 302 测例；go test ./... 全绿。
- **Created**: 2026-08-26
- **Context**: GitLab SSO 未登录时 `handleOidcAuthorize` 把 `next` 拼成 `www` 绝对 URL；taskFE 烘焙的 gateway base 是 apex，精确 hostname 匹配会丢掉回跳。本会话已在前端把 www/api/apex 视为同站。后端仍写死绝对 www。
- **Action**: (1) `resumeTarget` 改为 `r.URL.RequestURI()`（`/api/oidc/{tid}/authorize?...`）(2) 更新 `TestOidcAuthorizeUnauthenticatedPrefersWwwLoginBase` (3) 补租户路径未登录 302 测例
- **Why**: 相对 next 不依赖 www vs apex vs api，登录页 `sanitizeOidcResumeNext` 已接受该形态并拼回 gateway。
- **How to apply**: `taskAuth/src/oidc_handlers.go` 约 195–198 行；`oidc_provider_test.go`。

## [OPT-20260826-013] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskAuth a53306f + taskFE f1050da 已推送：OmniAuthSnippet identifier 下一行输出 secret: '<PASTE_CLIENT_SECRET>' 占位（绝不回填真实密钥），Go 测例断言含占位且仅一条 secret 行；前端 insertOmniAuthSecret 改为替换占位行（保留旧片段后插兜底），vitest 6 例绿。已登记 task-auth/taskFE 精准重启。
- **Created**: 2026-08-26
- **Context**: 租户 GitLab SSO 卡片已告诉小白把密钥写到 `secret:`，但 GET 返回的 `omniauth_snippet`（`taskAuth/domain/tenant_gitlab_oidc_sso.go` `OmniAuthSnippet`）不含 `secret` 行；仅签发当次由前端 `insertOmniAuthSecret` 注入。刷新后片段里看不到该键。
- **Action**: (1) 在 `OmniAuthSnippet` 的 `identifier` 下一行加 `secret: '<PASTE_CLIENT_SECRET>'`（或等价占位，禁止回填真实密钥）(2) 补 Go 测例断言 GET 片段含占位、不含真实 secret (3) 前端 `insertOmniAuthSecret` 改为替换占位而非仅插入
- **Why**: 小白按灰色代码块整段粘贴时，刷新后容易漏掉 `secret` 行导致 OmniAuth 换票失败。
- **How to apply**: `taskAuth/domain/tenant_gitlab_oidc_sso.go`；`taskFE/app/src/utils/gitlabOidcSsoSnippet.js`；对应 `*_test.go` / `gitlabOidcSsoSnippet.test.js`。

## [OPT-20260826-014] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskFE c97385e 已推送：gitlab.rb 片段与 client_secret 各加「复制」按钮（只读剪贴板 writeText，标注 Anti-Replay-OK 无写 API），复制反馈短暂提示；补 2 例 vitest：点击调 clipboard.writeText 且不发写 API。taskFE 已登记精准重启。
- **Created**: 2026-08-26
- **Context**: gitlab-connection 的 OIDC SSO 卡片已展示 `gitlab.rb` 片段和一次性 `client_secret`，用户需手动划选复制。
- **Action**: (1) 在 `WorkspaceSettingsGitlabOidcSso.vue` 为 snippet 与 secret 增加「复制」按钮 (2) 使用同步 `createClickGuard` 或标注 `Anti-Replay-OK: 只读剪贴板` (3) 补 vitest：点击后 `navigator.clipboard.writeText` 被调用且不发写 API
- **Why**: 密钥仅显示一次且字符串很长，划选易漏；复制按钮降低贴进 gitlab.rb 的出错率。
- **How to apply**: 对照同页 Redirect URI 的复制按钮实现；测例放 `WorkspaceSettingsGitlabOidcSso.test.js`。

## [OPT-20260826-009] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskBill 853b380 + taskFE 29df3e9 已推送：wechat_state 未提交哨兵由中文「尚未提交微信」改机器码 not_submitted；前端兼容识别新旧值；Go/前端单测全绿。已登记 task-bill/taskFE 精准重启。
- **Created**: 2026-08-26
- **Context**: 列表/同步接口把未 POST `/v3/profitsharing/orders` 的行写成中文 `wechat_state=尚未提交微信`。前端已映射为「未向微信发起分账」，但 API 仍把展示文案当契约，新客户端或 i18n 会绑死这句中文。
- **Action**: (1) `wechatProfitSharingStateNotSubmitted` 改为机器码 `not_submitted`（或并列返回 `wechat_state_code`）(2) 前端 `profitSharingWechatStateLabel` 同时识别新旧值 (3) 更新 list/refresh handler 单测。
- **Why**: 展示文案应只在 UI 层；API 用中文哨兵会导致契约漂移、难做多语言。
- **How to apply**: `taskBill/src/profit_sharing_refresh_wechat.go` 常量；`profit_sharing_admin.go` 列表回填；`taskFE/app/src/utils/profitSharingWechatStateLabel.js`。

## [OPT-20260826-012] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskBill 1eef183 + taskFE 69edaa2 已推送：订单详情 profit_sharing[] JOIN receiver 下发 app_id/openid（快照回退），原始 referrer_openid 键不暴露；订单展开分账表新增 AppID/OpenID 列。Go+前端测例全绿。已登记 task-bill/taskFE 精准重启。
- **Created**: 2026-08-26
- **Context**: 管理端待分账队列与推荐绩效微信分账 Tab 已展示 AppID/OpenID。订单详情 `GET /api/system-admin/orders/{id}/` 的 `profit_sharing[]` 仍不含这两字段，展开单笔订单时无法对照「appid 与 openid 不匹配」。
- **Action**: (1) `listProfitSharingForOrder` SELECT/JOIN 与队列一致的 `app_id`/`openid` (2) 更新 `handlers_admin_get_order_test.go` 正向断言 (3) 订单展开 UI 若有分账表则加列。
- **Why**: 同一对账场景第三条入口仍缺字段，运营从订单记录深链进来会漏看。
- **How to apply**: `taskBill/src/profit_sharing_admin.go` `listProfitSharingForOrder`；`taskFE` `OrderExpandDetail` 分账块。

## [OPT-20260826-008] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskBill e662a27 + taskFE 588bf2d 已推送：refresh-wechat 错误行带 wechat_error_trace_id；referral-ps-wechat-state 在 wechat_error 非空时挂 data-traceId（无错误不挂），同步合并保留字段。Go+前端测例全绿。已登记 task-bill/taskFE 精准重启。
- **Created**: 2026-08-26
- **Context**: 失败原因列已挂持久化 `fail_trace_id`。同表「微信状态」列在 `wechat_error` 非空时仍无 `data-traceId`，该文案来自 `POST refresh-wechat` 当次 QueryOrder，不是 CreateOrder 落库失败。
- **Action**: (1) refresh-wechat 响应每行带 `wechat_error_trace_id`（该次 QueryOrder 的 ctx trace）(2) `referral-ps-wechat-state` / 待分账队列同类单元格在 wechat_error 非空时挂 `data-traceId` (3) 单测覆盖有/无 error。
- **Why**: 运营点「同步微信状态」看到的微信侧错误无法从 DOM 一键查 Loki。
- **How to apply**: `taskBill` refresh-wechat handler；`ReferralWechatProfitSharingTab.vue` 微信分账单状态 td；`SystemAdminProfitSharingPanel.vue` 若展示 wechat_error。

## [OPT-20260826-006] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskFE bf38808 已推送：创建专有网络/交换机后提供「写入默认机器配置」按钮，把新建 vpc_id/vswitch_id POST 回已有默认配置行（不新建授权）；CreateVswitchModal created 事件补 vswitch_id 载荷；补成功/失败 data-traceId 两例。taskFE 已登记精准重启。
- **Created**: 2026-08-26
- **Context**: GitLab 设置页同 VPC 提示可创建专有网络/交换机，但创建成功不会写入 `cloud_server_config_defaults`。用户仍须去「工作空间管理 → 机器节点」手动保存，否则任务启动仍可能 auto_create_vpc。
- **Action**: (1) 在 `GitlabSelfHostedSameVpcHint` VPC/交换机 `created` 后询问是否 POST 回默认配置的 vpc_id/vswitch_id (2) 仅更新已有默认配置行，不新建授权 (3) 单测覆盖「创建后可选回写」。
- **Why**: 只建网不写入默认机器，任务节点与 GitLab 仍可能不在同一 VPC。
- **How to apply**: `taskFE/app/src/views/GitlabSelfHostedSameVpcHint.vue`；`POST /api/cloud/server-config-default/tenant_id/{tid}/` 既有写入契约。

## [OPT-20260826-001] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: shareLib 3ee896a + taskAuth 3876d42 + taskBill 4efc874 + taskCloudService 866a78d + taskTaskService d7bf12d 已推送：新增 shareLib/clientip 单源包（Resolve/CanonicalIPString/IsPublicIPAddr/SplitForwardedFor + 13 例单测），四服务删本地副本改委托薄包装，服务专属 body 优先语义保留。已登记 taskFE/task-auth/task-bill/task-cloud-service/task-task-service 精准重启。
- **Created**: 2026-08-26
- **Context**: 修复登录历史 Docker 网桥 IP 时，在 taskAuth / taskBill / taskCloudService / taskTaskService 各复制了一份「XFF 从右跳过 RFC1918」算法。后续口径漂移风险高。
- **Action**: (1) 新增 `shareLib/clientip`（Resolve + 单测）(2) 四服务 `replace` 后删本地副本 (3) go test 相关包全绿
- **Why**: 登录历史、自动 SG 白名单、协议同意 IP 必须同一套公网解析，复制会再次把 172.26.0.1 写进库。
- **How to apply**: 对照 `taskAuth/src/client_ip.go` 的 `rightmostPublicIP` / `isPublicIPAddr`；`daydaymoneymeta` 模块骨架可复用。

## [OPT-20260826-003] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskAuth b779fe6 + taskEvents 84542e9 + conf f97711e 已推送：taskAuth 新增 POST /api/internal/taskauth/wechat-mp-cleanup/（分批删除 auth_wechat_mp_subscribe_pending created_at>30d 与 auth_wechat_mp_follow_ticket expire_at>7d 行，pending_max_age_days/ticket_max_age_days/limit 参数，requireInternalSecret 门禁）；taskEvents 新增 wechat_mp_cleanup/1_cleanup timer worker（port 18070，默认 24h tick，透传 X-TaskAuth-Internal-Secret）；conf 登记 events config + runAll.yaml。回归测：taskAuth 4 例（空表/仅超龄/分批循环/端点 403+200）+ taskEvents 4 例，全量 go test 绿；已登记精准重启 task-events-wechat-mp-cleanup-1-cleanup（task-auth 已在既有登记）。
- **Created**: 2026-08-26
- **Context**: 服务号关注时 unionid 尚未对应平台用户会写入挂起行。动态 scene 票过期后仍可能以 pending/expired 留在 `auth_wechat_mp_follow_ticket`。用户若不登录微信或不点「我已关注」，行会永久留存。
- **Action**: (1) 在 taskEvents 增加一次性 cleanup API 调用方 (2) taskAuth internal DELETE 分批删除 created_at 超过 30 天的 pending，以及 expire_at 超过 7 天的 follow_ticket (3) 单测覆盖空表与超龄行
- **Why**: 挂起表与票据表无归档策略，长期会堆积无法认领的 unionid 与过期 scene。
- **How to apply**: 表 `auth_wechat_mp_subscribe_pending`、`auth_wechat_mp_follow_ticket`；对照 `oidc_sso_idempotency_cleanup` timer worker。禁止在 taskAuth 进程内 ticker。

## [OPT-20260827-009] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: runAll status UI adds 服务网址目录 button opening Grafana uid service-url-api-catalog with docs md fallback; ui_test asserts; runAll 1b4cb4d pushed
- **Created**: 2026-08-27
- **Context**: 网址目录的浏览面在 Grafana `:3000`，运维同时用 `:9999` 做编译重启。两边没有互链，找「某服务有哪些公网 URI」仍要记 UID。
- **Action**: (1) 在 `runAll/src/status_ui/index.html` 增加「服务网址目录」链接，指向 Grafana uid `service-url-api-catalog` 与 `docs/architecture/service-url-api-catalog.md` (2) 补 `ui_test.go` 断言
- **Why**: 目录已经生成，缺的是从日常运维入口一跳到达。
- **How to apply**: 只加链接，不在 runAll 进程内嵌生成器。

## [OPT-20260827-004] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: 5-nfr skill adds references/redis-cache.md (role split, decision tree, cache table, degradation, prohibitions); SKILL.md §4 + template optional table; meta 182122ca
- **Created**: 2026-08-27
- **Context**: 审计 `/5-nfr` 时技能 713 行仅 3 处提到 Redis/缓存，且都不是 cache-aside。存量约 279 份 NFR 里 7% 提 Redis、1.4% 同时写缓存。仓内 PDP 已用 `membership_rev` + 进程缓存 + Redis 宕机降级，技能未引用。
- **Action**: (1) 新增 `.claude/skills/5-nfr/references/redis-cache.md`，正文从 `docs/superpowers/specs/2026-08-27-5-nfr-redis-cache-audit.md` 文末附录拷入并补全决策树 (2) 在 `SKILL.md` §4 性能/可用性下增加「读路径 ≥ L2 或跨进程共享易失状态时引用该附录」；模板增加可选「缓存策略审视」表 (3) Hard Gate 与 CI **不** 把该表列为必填 (4) 写明禁止 Redis Streams 作新领域事件总线、禁止 Redis 作资金 SoT
- **Why**: 不补则 Agent 继续把 Redis 写成 MQ/启停，读路径默认 locmem，多 worker 缓存漂且 Redis 宕机语义不可验证。
- **How to apply**: 只改 `.claude/skills/5-nfr/`；同步 `.ai/01_project_constraints/48`/`53` 仅当需要交叉引用。验收：新增量若含多实例读缓存，NFR 出现角色列与 TTL/失效/降级。

## [OPT-20260827-005] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: architecture README Redis row now SSE pub/sub, read-aside/version-stamp, transient session; Kafka is domain event bus; docs 475184d pushed
- **Created**: 2026-08-27
- **Context**: `docs/architecture/README.md` 基础设施表仍写 Redis「消息队列 (Redis Streams)、缓存、SSE pub/sub」。领域事件已走 Kafka；Redis 现网是 SSE pub/sub、PDP rev、relay session、短缓存。
- **Action**: (1) 改该行用途列为「SSE pub/sub、读旁路/版本戳、短暂会话；不作新领域事件总线」 (2) 扫 `docs/architecture/` 其它仍把 Redis 当 MQ 的句子一并改 (3) 不改历史 NFR
- **Why**: 技能审计发现 Agent 易被过时架构摘要带去把 Redis 当总线；README 是常见第一读物。
- **How to apply**: `docs/architecture/README.md` §3.6；用 rg `'Redis Streams|消息队列'` 限定 docs/architecture。

## [OPT-20260827-008] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: catalog CORE_API_GLOBS adds billing/cloud/container/referral system-admin prefixes; taskBill core 2→62; db 5e07a32 + docs 2cbe192 + AiMonitor 6343356; --check clean
- **Created**: 2026-08-27
- **Context**: 生成全站 URL 目录时 taskBill 仅 2 条 declared core、70 条 supporting，因为 CORE_API_GLOBS 只覆盖 `/api/billing/*` 与 `/api/tenant/*/billing/*`，大量 `/api/system-admin/profit-sharing` 等未标核心。
- **Action**: (1) 对照 `conf/value-stream.yaml` 计费/云/容器步骤列出公网前缀 (2) 写入 `db/scripts/ci/build_service_url_api_catalog.py` 的 `CORE_API_GLOBS` (3) 补单测后 `build_service_url_api_catalog.py` 再生目录
- **Why**: 声明式核心过窄会让 Grafana 目录把充值/配额当成「非核心」，和价值流不一致。
- **How to apply**: 只改 globs + 测试；不要把 `/api/internal` 标成 core。

## [OPT-20260827-001] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: GET /api/auth/verify on taskauth-verify (prio 852 token) off login 0.5/s; regression test; apisix regenerated + hot-reload; taskGateway 8994b11
- **Created**: 2026-08-27
- **Context**: GET `/api/auth/user-roles/` 与 `/api/auth/user-permissions/` 曾落入 `taskauth-login`（`limit-req` 0.5/s），Navbar 并行请求被漏桶拖到 ~2s。已把这两条升到 `taskauth-rbac-admin`。仍有 `GET /api/auth/verify`、`/api/auth/wechat/login|callback|bind` 等走 `/api/auth/*` 登录限流。
- **Action**: (1) 对照 `taskAuth/src/handlers.go` 列出所有 `GET /api/auth/` (2) 与 `taskGateway/routes/routes.yaml` 中 priority>850 的显式 URI 做差集 (3) 已登录高频读接口按 activate-session/rbac-admin 模式拆出，登录/注册/验证码保留 0.5/s
- **Why**: 漏桶对超额请求延迟约 1/rate=2s，已登录会话 API 与防暴力破解限流叠在一起会再次出现 2s TTFB。
- **How to apply**: 仿 `taskGateway/scripts/ci/test_user_roles_not_on_login_ratelimit.py` 给差集补回归测；改 `routes.yaml` 后 `bash taskGateway/run.sh routes-apply`。

## [OPT-20260827-006] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskProjectService POST /api/internal/taskproject/tenant-gitlab-cache/invalidate + TTL 30s→5s; taskGitOauth PUT/DELETE calls it; both repos pushed (adae738/9ceb82a)
- **Created**: 2026-08-27
- **Context**: `taskProjectService` `defaultLookupTenantGitLabConn` 按 tenant 缓存 30s。管理员在设置页改 Path A `base_url` / 停用连接后，分支预览最多 30s 仍用旧 host/`provider_key`。
- **Action**: (1) tenant-connection PUT/DELETE 成功后打内部失效接口或缩短 TTL (2) 或让 lookup 带 `base_url` 指纹，不匹配则绕过缓存 (3) 补测：PUT 改 host 后立即 `matchRepoProvider` 用新 origin
- **Why**: 缓存正确性优先于少一次内部 GET；错 host 会再次出现「未检测到可用授权」或打错 GitLab。
- **How to apply**: `taskProjectService/src/tenant_gitlab_provider.go` 缓存表；`taskGitOauth` tenant-connection 写路径。

## [OPT-20260827-010] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: verified already implemented in taskFE: sanitizeStaleTenantPathSuffix strips proj_/task_ from stale tenant redirect; staleTenantRecovery.test + tenantRouteGuard.test 32 cases green
- **Created**: 2026-08-27
- **Context**: 打开 `/tenant/877397588196749312/projects/proj_880498883115905024/` 时，若当前账号不在该公司，`handleTenantRouteGuard` 会把 tenant 换成 `companies[0]` 但 **保留** `/projects/proj_…/`。本会话已让 GET 跨租户 404，页面会失败；此前还会用错 company 的 Git OAuth catalog，Git 行显示「未授权」且没有授权按钮。
- **Action**: (1) 在 `taskFE/app/src/utils/staleTenantRecovery.js` 的 `resolveMissingCompanyRedirectPath` 对项目详情/编辑等资源路径改落到 `/projects/` 列表，而不是原样拷贝 `proj_`/`task_` 段 (2) 补 `tenantRouteGuard` 单测：非成员打开他公司项目 URL → 目标不含该 project id (3) 鉴权失败仍须 modal，禁止静默 replace 回来源页（元规则 49）
- **Why**: 只 404 项目详情仍会留下「错误租户 + 他人项目 ID」的书签/地址栏；用户以为在看自己的项目。
- **How to apply**: `taskFE/app/src/utils/tenantRouteGuard.js`、`staleTenantRecovery.js` 及对应 `*.test.js`。

## [OPT-20260827-011] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: 042 applied; taskCloudService compile-then-swap pid 2838750 health 200 with ccb_startup_log_archive_ok; 启动日志 SPA unchanged so taskFE not rebuilt. :9999 still down (no_restart_runall).
- **Created**: 2026-08-27
- **Context**: 启动日志 COS 指针表在 `dataMigrate/taskCloudService/042_cloud_comment_startup_log_object.sql`。业务进程不再启动时自动 migrate。2026-08-27 已用 `apply_datamigrate.sh task_cloud` 应用 042（`check_data_migrate_applied.py --db task-cloud` → local=43 applied=43）。runAll :9999 当前不可达，task-cloud-service / taskFE 二进制尚未精准编译重启。
- **Action**: (1) ~~migrate~~ 已完成 (2) ~~表存在~~ 已确认 (3) 9999 恢复后精准编译重启 task-cloud-service 与 taskFE（登记文件已含这两项）
- **Why**: 没有指针表则 COS 归档无法落库索引，分片裁剪后无法还原启动日志。
- **How to apply**: `dataMigrate/taskCloudService/042_cloud_comment_startup_log_object.sql`；9999 InitAllDatabases

## [OPT-20260827-013] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskFE ec6b254 已推送：queued_auto_run 时已发出评论执行细节锁定串行（setMode 拒 independent + 可并行按钮 disabled + badge/hint 按逐条语义），文案与 composer 对齐，4 例回归测绿（含不得发出 independent PATCH）
- **Created**: 2026-08-27
- **Context**: 任务详情已把队列与发评依赖同卡，入队后**新评论草稿**强制 wait_previous。已发出评论仍可在「执行细节」切到 independent，与「一条一条执行」语义不一致。
- **Action**: (1) `TaskDetailCommentExecutionDetails` 在 `task.queued_auto_run` 时禁用切 independent (2) 补回归测：queued 任务展开执行细节不得发出 independent PATCH (3) 文案与 composer hint 对齐
- **Why**: 用户入队后仍能把单条历史评论改成并行，队列「逐条」会被打穿。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.vue` 的 `setMode`；task 从 CommentsSection 下传

## [OPT-20260827-002] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: 复验通过：aimonitor-promtail 运行中且抓 /tmp/ram-work/logs；Loki /labels 现含 job（task-auth/task-bill/task-cloud-service/promtail-local 等）；用现网 task-auth 日志真实 trace_id f84decf9...7a35 跑 {job=~".+"} |= "<id>" 命中 2 条 http_request，data-traceId 全链路可检索。
- **Created**: 2026-08-27
- **Context**: 排障 GET `/api/auth/user-roles/`（trace `07840710-be49-4d74-a999-49876bfaf16d`）时 Loki `/ready` 为 ready，但 `{job=~".+"}` 过去 7 天 0 条、`/labels` 无 job。根因定位靠 Tempo `GET /api/traces/<hex>`（task-auth span 1.52ms）。
- **Action**: (1) 确认 aimonitor-promtail 在跑且 scrape `/tmp/ram-work/logs` (2) Loki `/loki/api/v1/labels` 出现 job (3) 用任意现网 X-Trace-Id 跑 `{job=~".+"} |= "<id>"` 命中
- **Why**: 有 data-traceId 必须先查 Loki；管道空会迫使只靠 Tempo，网关漏桶等 APISIX 侧耗时看不到。
- **How to apply**: `AiMonitor/run.sh` / `runall-local-promtail.sh up`；对照 OPT-20260824 Loki 恢复记录。

## [OPT-20260827-007] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: AiMonitor 5db17d7 已推送 + 现网生效：partition_metrics_exporter 默认口令对齐 compose（root123456，env 可覆盖）；新增 crontab 每分钟写 /home/ljy/ramwork-recovery/node-exporter-textfiles/partition_metrics.prom；node-exporter :9100 已暴露 mysql_partition_count/mysql_hot_rows/mysql_partition_rows/mysql_partition_maxvalue_rows；补齐 aimonitor-prometheus 容器（compose 托管，本窗拉起）后 Grafana「数据库分区状态监控」面板查询非 0（分区表 7、有效分区 29、热数据 1338 行）。
- **Created**: 2026-08-27
- **Context**: `数据库分区状态监控` 已能打开，mysqld-exporter `mysql_up=1` 且 `mysql_info_schema_table_rows` 有 task_bill 行数。分区总览仍全 0，因为 `mysql_partition_count` / `mysql_hot_rows` 来自 `AiMonitor/scripts/partition_metrics_exporter.py`，现网未写入 node-exporter textfile。
- **Action**: (1) 按脚本 `--output` 写到 compose 中 `TEXTFILE_DIR` (2) 用 cron 每分钟跑 (3) Grafana 分区总览非 0
- **Why**: 冷热分离看板的分区裁剪/MAXVALUE 告警依赖这些自定义指标，只有表行数不够。
- **How to apply**: `AiMonitor/scripts/partition_metrics_exporter.py`；`docker-compose.yaml` node-exporter `--collector.textfile.directory`。

## [OPT-20260826-015] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: 已回填并推送：dataMigrate 073（1f543cd）幂等 JOIN receiver(status=registered,openid<>"") 回填 billing_profit_sharing.referrer_openid；dry-run 2 行（用户 877397583960502272 行 1/2 空值）→ 应用后抽查该用户 3 行全部与接收方 openid 一致，still_unmatched=0；已登记 data_migrate_log（applied=72），pre-push 放行。
- **Created**: 2026-08-26
- **Context**: 列表/出站已改为优先 `billing_profit_sharing_receiver` 成对身份。存量 `billing_profit_sharing.referrer_openid` 仍可能是网站应用 openid（如用户 877397583960502272 的行 3），下次分账会覆盖，但扫描/其他读快照路径在覆盖前仍可能读到旧值。
- **Action**: (1) 写一次性脚本：JOIN receiver，将 `ps.referrer_openid <> rcv.openid` 且 rcv 已登记的行更新为 rcv.openid（2) 先 dry-run 计数（3) 应用后抽查该用户行 3。
- **Why**: 展示已对齐 wechat_identity，但台账快照不修则扫描路径仍可能用错 openid。
- **How to apply**: `taskBill` 一次性 UPDATE 或 dataMigrate 幂等 SQL；条件 `rcv.status='registered' AND rcv.openid<>''`。

## [OPT-20260827-003] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskGateway a4ac71c + AiMonitor ed3d32f 已推送，现网生效：APISIX prometheus 插件全局启用（config.yaml plugins + global_rules，prefer_name），plugin_attr export_addr 0.0.0.0:9091 + compose 发布 19091:9091；Prometheus apisix-gateway job target up（119 条 histogram 系列）；Grafana http-api-latency 新增 Gateway p95 by route 与 p50/p95/p99 面板（ms→s），实测 p95 task-cloud-service=47ms/spa=1.8ms、整体 p50=1.9ms/p95=43.8ms/p99=48.8ms；routes-to-apisix --check 绿 + test_apisix_yaml 3 例绿。
- **Created**: 2026-08-27
- **Context**: Grafana `HTTP API Latency`（uid `http-api-latency`）已用各 Go 服务 `*_http_request_duration_seconds` 展示 p50/p95/p99。这是服务进程内耗时，不含 APISIX 路由/forward-auth/上游排队。用户感知延迟应以网关 hop 为准。
- **Action**: (1) 在 taskGateway 启用 APISIX prometheus 插件或等价 histogram (2) Prometheus scrape APISIX metrics (3) 在 `http-api-latency.json` 增加 Gateway 面板（按 route/upstream p95）
- **Why**: 只看下游服务会漏掉网关漏桶、502 JSON error_page、upstream 连接耗时。
- **How to apply**: `taskGateway/apisix/` prometheus plugin；`AiMonitor/prometheus/prometheus.yml` 新 job；看板 uid `http-api-latency`。

## [OPT-20260827-012] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: taskCloudService f802a08 已推送：新增 backfill-ccb-startup-logs 一次性运维 CLI（按 workspace 分片扫描 cloud_comment_container_binding_logs_{00..15} 中无 cloud_comment_startup_log_object 指针的 comment_id，组装 bundle PutObject + UPSERT 指针，COS 失败回退 payload_json 本地保底）；4 例回归测 + 整包 go test 269s 全绿，pre-commit 通过。实际回填执行需人工/运维窗口对生产库跑 go run ./src backfill-ccb-startup-logs（本窗不触碰生产数据）。
- **Created**: 2026-08-27
- **Context**: 双写只覆盖新 insert。已有 `cloud_comment_container_binding_logs_{00..15}` 行不会自动出现在 `startup_logs.json`。
- **Action**: (1) 按 workspace 分片扫描未出现在指针表的 comment_id (2) 组装 bundle PutObject (3) UPSERT 指针；一次性运维 CLI，禁止业务 ticker
- **Why**: 旧任务冷打开在分片裁剪后仍会丢启动日志。
- **How to apply**: 新 CLI 或 9999 运维按钮；复用 `persistCommentStartupLog`

### OPT-20260824-061 — taskChromePlugin pem 残留于 daydaymoney 私有 GitLab 实例历史

- **Status**: cancelled
- **Created**: 2026-08-24
- **Context**: OPT-059 执行中 github origin/main 已历史净化（0 pem），但 daydaymoney 私有 GitLab 实例（gitlab.daydaymoney.com 主站 SSH 流量未预购阻断、gitlab-tencent-sh-1 的 main 为 protected branch 拒绝 force push）的历史仍含 extension-dev.pem（2026-07-04 提交）。
- **Action**: (1) 主站流量恢复后确认 gitlab.daydaymoney.com main 是否含 pem 并同步新历史 (2) gitlab-tencent-sh-1 需 maintainer 在 UI/API 临时解除 main 分支保护 → force push 净化历史 → 重新启用保护（或用 `git push --mirror` 从净化后的 origin 全量镜像覆盖）
- **Why**: 私有实例同样留存签名私钥，任何获得该实例访问权者均可伪装扩展；净化目标应为全部远端。
- **How to apply**: `taskChromePlugin/`；GitLab 实例管理员操作（分支保护设置页 / API `POST /projects/:id/protected_branches`）

- **Cancelled**: 2026-08-27
- **Cancel-Note**: 违反约束 61（禁止 git filter-repo / force-push 清共享历史中的密钥）；pem 泄露按已泄露处置（废弃凭据 + 轮换），不执行历史重写。

### OPT-20260818-012 — 将私网 Registry 检测抽到 shareLib 并拆分 VendorPortal.vue

- **Status**: cancelled
- **Created**: 2026-08-18
- **Context**: 本次在 taskAiProvider 与 taskCloudService 各放了一份 `private_registry.go`，规则需与前端 `privateRegistryHint.js` 人工同步。`VendorPortal.vue` 已 1596 行，未为加提示去改模板以免触发整文件削减。
- **Action**: (1) 抽 `shareLib/registryhost` 供两服务引用 (2) 考虑用共享用例表驱动 JS/Go (3) 按模态框规范把添加/编辑镜像版本弹层拆出，使 `VendorPortal.vue` ≤ 500 行
- **Why**: 双份 Go 实现会漂移；超大 Vue 文件后续改错误展示成本高。
- **How to apply**: `taskAiProvider/infrastructure/private_registry.go`、`taskCloudService/src/private_registry.go`、`taskAiProvider/frontend/src/views/VendorPortal.vue`

- **Cancelled**: 2026-08-27
- **Cancel-Note**: 剩余工作与 OPT-20260817-016（拆分 VendorPortal.vue ≤500）重复；(1)(2) shareLib/registryhost 与用例表已落地。

## [OPT-20260827-016] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: login-credential.mdc 与 sso-login.mdc 已改为只读 PLAYWRIGHT_TEST_PASSWORD；rg .cursor/rules 无明文口令
- **Created**: 2026-08-27
- **Context**: `.cursor/rules/login-credential.mdc` 与 `sso-login.mdc` 把 `rgNodkdq8677!ci` 写进 always-on 规则。门禁不扫 `.mdc`，但规则文件仍进 Git。
- **Action**: (1) 两处规则改为只引用 `PLAYWRIGHT_TEST_PASSWORD` / `task2app/测试.ai.md` (2) 删除明文口令 (3) `rg -n 'rgNodkdq8677' .cursor/rules` 须无命中
- **Why**: 规则文件会被 clone；明文口令与第 57 条「不把密钥写进仓库」不一致。
- **How to apply**: 改完后 `rg -n 'rgNodkdq8677' .cursor/rules` 退出码 1；`python3 db/scripts/ci/check_no_hardcoded_secrets.py` 仍为 0。

## [OPT-20260827-033] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: 插件项目已改为单选；自动运行控件随 projectAllowsAutoRun 启用/禁用并默认勾选。npm test 457 通过。
- **Created**: 2026-08-27
- **Context**: 本增量只在项目名称旁标注 `default_auto_run`。taskFE 创建任务会在项目不允许时关掉 auto_run；插件仍可勾选，提交后由后端 `AUTO_RUN_PROJECT_NOT_ALLOWED` 拒绝。
- **Action**: (1) 根据已选项目的 `projectAllowsAutoRun` 同步禁用/提示浮窗与面板 auto_run 勾选 (2) 多选时定义「任一不允许则关」或「主项目」规则并写测 (3) 使用说明补一句
- **Why**: 标注之后用户仍可能勾选自动运行并在提交时报错。
- **How to apply**: `taskChromePlugin/content/content.js` autoRunInput；`panel` single/batch AutoRun；复用 `lib/project-auto-run-label.js`

## [OPT-20260827-025] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: Path A IP YAML miss 不再标 missing：gitsite:{host} 走 gitOauth access-for-user；T8b + clone_provider/build_credentials/layer_oauth/gitoauth_client 测绿
- **Created**: 2026-08-27
- **Context**: 任务 `task_880716211791360000` 引导克隆 `REPO_CLONE_CREDENTIALS_INCOMPLETE`，缺失 `http://115.29.110.74/example-user/somanyad.git`。任务关联被空层锚点挡住是 UX 问题（已修）；凭证不齐本身可能是 host-alias 未匹配或用户未绑该实例 OAuth。
- **Action**: (1) 对照 `ResolveProvider` / `user-app-connection?repo_url=` 对该 IP URL 的实例键 (2) 确认 website 插值是否覆盖 `115.29.110.74` (3) 单测：IP GitLab URL 不得落到 `gitlab:default` 误放行或误拒绝 (4) 若仅未绑定则文档化须在创建任务时绑该实例
- **Why**: overlay 只能提示「Git 授权未齐」；不修匹配则同 URL 的自动运行会持续 BOOTSTRAP_FAILED。
- **How to apply**: `taskCredentialService` `ResolveProvider`；`taskGitOauth` connection 检查；`cd taskCredentialService && go test ./infrastructure -run ResolveProvider`

## [OPT-20260827-041] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: 排队任务卡刷新改为 emit refresh → reload()，WorkspaceQueueSchedule.test.js 断言再发 GET
- **Created**: 2026-08-27
- **Context**: `WorkspaceQueueSchedule.vue` 排队任务卡「刷新」按钮 `@click="loadSnapshot"`，但 `setup()` 只 return 了 `reload`（内部才调 `loadSnapshot` 并 `fillForm`）。Vue 模板拿不到 `loadSnapshot`，点击刷新无请求。本次只改任务标题链接，未动刷新。
- **Action**: (1) 将 `@click="loadSnapshot"` 改为 `@click="reload"` (2) 在 `WorkspaceQueueSchedule.test.js` 断言点击 `refresh-members` 会再发 GET `/queue-schedule/`
- **Why**: 用户点刷新期望更新排队列表；当前按钮是空操作。
- **How to apply**: `taskFE/app/src/views/WorkspaceQueueSchedule.vue` 刷新按钮；对照已有 `reload()` 实现

## [OPT-20260827-042] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: enqueueQueuedAutoRun 在工作空间节奏 enabled=false 时返回 409，创建路径回滚任务行；Go 单测覆盖拒绝/允许/legacy/HTTP 409。
- **Created**: 2026-08-27
- **Context**: 任务详情「加入自动执行队列」已在前端 GET `queue-schedule` 且仅 `schedule_rhythm.enabled===true` 时 PATCH。`enqueueQueuedAutoRun` 仍允许未启用时写入 membership 并标 deferred。其它客户端或旧 SPA 可绕过按钮门闹。
- **Action**: (1) 工作空间已配置节奏且 `enabled=0` 时 `PATCH queued_auto_run=true` 返回 409 与明确文案 (2) 补 Go 单测：未启用入队失败、已启用入队成功 (3) 与前端 GET 失败/未启用弹窗文案对齐
- **Why**: 前端门闹可被直接打 API 绕过，队列里会继续出现「自动调度未启用」的假入队。
- **How to apply**: `taskTaskService/src/queued_schedule.go` `enqueueQueuedAutoRun`；对照 `loadEffectiveWorkspaceRhythm`；测例放 `queued_schedule_test.go` / `queued_schedule_workspace_test.go`

## [OPT-20260827-043] completed

- **Status**: completed
- **Completed**: 2026-08-27
- **Summary**: Chrome 插件浮窗与 DevTools 单请求/批量创建同步「加入自动调度队列」：GET queue-schedule、payload queued_auto_run、user-guide 1.8.19。
- **Created**: 2026-08-27
- **Context**: 工作面板创建任务在自动运行且工作空间已启用自动调度时可选入队。Chrome 插件浮窗仍只有 auto_run 开关，没有 queued_auto_run。
- **Action**: (1) 浮窗在项目允许自动运行时 GET queue-schedule (2) enabled 时展示入队勾选 (3) 创建 payload 带 `queued_auto_run` (4) 补 `project-auto-run-label` / create-task-payload 单测
- **Why**: 插件创建路径会继续立即启服，与工作面板语义分叉。
- **How to apply**: `taskChromePlugin/content/content.js`、`lib/project-auto-run-label.js`；对照 `taskFE` `createTaskQueuedAutoRun.js`

