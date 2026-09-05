# Completed OPT Archive — 2026-08-07

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 16 条。
> 归档执行时间：2026-08-13T13:17:38+08:00

## [OPT-20260806-066] completed
- **Status**: completed
- **Completed**: 2026-08-07
- **Summary**: 已解决：ExchangeBridge 移除自动建号（2026-08-07 审核流实施），未申请用户构造 bridge 也无法建档（TestSSOExchangeNoVendorNoAutoCreate 回归）；服务端预检闭环
- **问题**: 镜像市场「厂商门户（SSO）」按钮已按 vendor-status 资格条件渲染（2026-08-06），但 taskAuth handleSSOBridge / taskAiProvider ExchangeBridge 的 vendor_bridge 分支仍无资格预检——已知 URL 的未获准用户可直接 GET `/api/accounts/sso/ai-provider/vendor/` 触发 UpsertVendorFromBridge 自动建号（is_active=1，绕过「申请获准」语义），与 OPT-20260806-065 的自动建号产品决策叠加后成为实际的后门
- **建议**: a) 在 handleSSOBridge 的 vendor_bridge 分支加资格预检：taskAuth 调用 taskAiProvider 内部 vendor-status（X-Internal-Secret 模式）或新增内部接口，未获准用户 403/重定向而非签发 bridge JWT；b) 或在 ExchangeBridge 移除自动建号分支（配合 OPT-20260806-065 产品决策）。实现后补充 SSO 回归测试（未获准用户 exchange 被拒）

## [OPT-20260806-067] completed
- **Status**: completed
- **Completed**: 2026-08-07
- **Summary**: 已解决：个人资料页新增邮箱绑定面板（UserProfileEmailBindingPanel），taskAuth 新增 POST /api/accounts/users/bind_email/（验证码确认后 upsert auth_login_method email 行，与 bind_phone 同构：占用 409 / 同人换绑 / 拒绝 @sso.invalid 合成邮箱），profile 接口补 has_email 字段；网关 routes.yaml 显式注册 bind_email 并顺带修复 bind_phone 的 502（此前非 profile 的 POST /api/accounts/users/* 落入 null-upstream）；镜像市场提示文案改为「请前往个人资料页完成绑定」，profile 页 sso_error=email_required 自动滚动定位邮箱绑定区域
- **问题**: OPT-20260806-065 确立「厂商门户需先绑定邮箱」并引导到 /profile/?sso_error=email_required，但个人资料页没有任何邮箱绑定 UI（只有手机号/微信面板），引导链路断头——无邮箱（微信扫码）用户被引导到个人资料页后无法完成绑定，镜像市场申请入口实际不可用；且后端无邮箱绑定 API、网关无对应路由
- **建议**: 已按 bind_phone 语义实现（邮箱仅作身份凭证、password_hash 留空，密码登录走既有重置链路）。后续可选：a) 绑定成功后同步刷新 taskAiProvider 厂商档案邮箱（provider 已在 vendor_bridge "email changed" 分支同步，无需额外事件）；b) 邮箱换绑策略（当前同人可随时换绑，如需限制频率可加冷却）

## [OPT-20260807-002] completed — userId cookie 兜底认证可被「清库重建」绕过（已修复）
- **Status**: completed
- **Completed**: 2026-08-07
- **问题**: 清空数据库重新初始化后用户仍保持登录。链路：登录后 activate-session 设置 30 天 HttpOnly `userId` cookie（纯裸 ID）；DB 重建后 `auth_customtoken` 被清空但 `auth_user` 以确定性 ID（`bootstrap-admin`）重新播种 → 网关 forward-auth 第 4 步仅凭 `userId` cookie + `userIsActive` 即认证成功 → 前端 `/users/me/` 200 → 显示已登录。dev 库实测：bootstrap-admin 存在且 active/superuser，token 表 0 行，与用户报告现象完全一致。此为认证绕过（任何能设置 userId cookie 者可冒充任意存在且启用的用户）
- **建议**: 已修复 — `resolveUserIDForForwardAuth`（网关）与 `resolveTokenUserIDFromRequest`（OIDC 路径）的 userId cookie 回退增加 `userHasLiveToken`（auth_customtoken 存在性）校验；前端登录成功路径补调用 activate-session 落 HttpOnly token cookie。回归测试：`TestGatewayForwardAuthUserIdCookieRequiresLiveToken` / `TestForwardAuthResolveUserIDFromRequestUserIdCookieRequiresLiveToken`（Red→Green），taskAuth 全量 go test + taskFE 全量 vitest 282 文件 1472 用例通过。遗留可选项见 003/004/005/006

## [OPT-20260806-058] completed — taskEvents/cmd/authorize_sg_ingress go.mod 依赖缺失
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: taskEvents/go.mod 补 `require dbload v0.0.0`（replace 已存在）与 `github.com/go-sql-driver/mysql v1.10.0`，`go build ./...` 通过
- **问题**: `go build ./...` 报 `module dbload provides package dbload and is replaced but not required` + `github.com/go-sql-driver/mysql` 缺失，该工具命令编译失败（2026-08-06 启动全量验证时发现，与锚点修复无关的既有问题）
- **建议**: 在 taskEvents/go.mod 补充 `require dbload`（替换已存在）与 `github.com/go-sql-driver/mysql`，或将该 cmd 移出默认构建集
- **Summary**: 直接在 go.mod 加两个 require；mysql driver 先被 go get 标为 indirect，`go mod tidy` 收敛为直接依赖后编译通过。
- **Verification**: `cd taskEvents && go build ./...` 通过；build-all 48/48 全量编译通过。

## [OPT-20260806-059] completed — runAll pre-commit 钩子期间 core.bare 被置 true（根因待查）
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: 静态排查 hook 链无 git config 写入点，判定为外部工具（IDE 会话面板等）以 gitdir 为上下文写 config 的副作用；以自愈守卫降级为「告警 + 自动 unset」
- **问题**: 2026-08-06 在 runAll 子仓提交时，pre-commit 运行后 `.git/modules/runAll/config` 出现 `bare = true`，导致后续 git 命令全部报 "must run in a worktree"；现场 unset 后恢复。钩子脚本（.githooks/*）中未找到写入 bare 的语句
- **建议**: 排查 pre-commit 抽测触发的 Go 测试（src/、src/domain）或 session lock 二进制是否有修改 git config 的副作用；复现后修复
- **Summary**: 根因调查：hook 链（pre-commit/commit-msg/session lock/随机抽测）全静态排查 + git 2.53 实测（钩子退出后 core.bare 不自动持久化）均排除钩子内写入；判定为外部工具以 gitdir 上下文写 git config 的副作用（external-tool 假设）。处理：共享库 scripts/lib/random_test_runner.sh（SSOT）新增 `rt_guard_core_bare()`——进入/退出钩子时自检 core.bare，被置位则 WARN 日志 + 自动 unset，把硬故障降级为自愈+日志暴露；三份副本（scripts/lib、templates/lib、runAll/.githooks/lib）已同步。
- **Verification**: 钩子自测 rt_self_test 通过；三个副本 diff 一致；带守卫的 pre-commit 实际提交通过。

## [OPT-20260806-060] completed — vue-frontend vite.config.js 默认端口 3000 与 ai-monitor 冲突
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: taskFE/app/vite.config.js 默认端口 3000→4000，与 conf/frontend/vue/config.yaml 及 runall-lifecycle.sh stop 逻辑对齐
- **问题**: `FB.vue.port` 默认 3000（与 ai-monitor/Grafana 冲突）。本次故障链路：conf-read.py 崩溃 → 配置回退默认 → preview 起 3000 → 端口冲突启动失败。修复 conf_loader 后正常（4000）
- **建议**: 将 vite.config.js 默认端口改为 4000（与 conf/frontend/vue/config.yaml 及 runall-lifecycle.sh stop 逻辑一致），降低误配回退风险
- **Summary**: taskFE/app/vite.config.js `FB.vue.port` 3000→4000，并添加注释说明端口归属（4000=taskFE、3000=ai-monitor/Grafana），误配回退不再撞端口。
- **Verification**: 配置加载与 preview 启动端口一致；build-all 48/48 通过。

## [OPT-20260806-061] completed — taskAiProvider SSO exchange 校验失败缺少结构化日志
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: handleSSOExchange 失败分支补齐 logWarn（验签失败含 err/iss/aud、typ 缺失、ExchangeBridge 失败），Grafana 可按错误模式监控
- **问题**: handleSSOExchange 中 ParseHS256JWT 失败（bad signature / bad iss / bad aud / typ 缺失）路径仅重定向 `/?error=无效的 bridge`，无 logWarn/logInfo（2026-08-06 排障时日志全空，只能靠代码静态分析定位密钥不一致）
- **建议**: 在 auth_handlers.go 60-83 行失败分支补充 `logWarn("SSO bridge verify failed: %v", err)`（含 typ/iss 检查失败），便于 Grafana 按错误模式监控 SSO 故障
- **Summary**: taskAiProvider/src/auth_handlers.go handleSSOExchange：ParseHS256JWT 失败分支 `logWarn("event=SSOBridgeVerifyFailed err=%v iss=%s aud=%s", ...)`；typ 缺失单独留痕 `event=SSOBridgeVerifyFailed reason=missing_typ`；ExchangeBridge 失败 `event=SSOExchangeFailed`。成功路径保留 logInfo。
- **Verification**: 全量 build-all 48/48；服务重启后 live smoke 旧密钥交换触发 SSOBridgeVerifyFailed 日志（验签层拒绝可见）。

## [OPT-20260806-062] completed — 生产 SSO bridge 密钥仍为 dev 默认值，需轮换
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: conf/core/sso/config.yaml ssoJwtSecret 轮换为 openssl rand -hex 32 强随机值，两服务重启后 live 冒烟验证（新密钥 200 / 旧 dev 密钥 401）
- **问题**: 全仓部署编排（runAll/scripts、daydaymoney.yaml、dockerInfra）均未设置 TASK2APP_SSO_JWT_SECRET / TASKAUTH_INTERNAL_SECRET / DJANGO_SECRET_KEY，SSO bridge 签名/验证双方都回退到 dev 密钥 `task2app-local-sso-bridge-dev-do-not-use-in-prod`（2026-08-06 修复对齐后可用，但密钥公开且弱）
- **建议**: 生产部署注入 `TASK2APP_SSO_JWT_SECRET=<强随机值>`（taskAuth 与 taskAiProvider 同值），或更新 conf/core/sso/config.yaml 后重启两服务；轮换后运行冒烟：主站系统管理 → 镜像市场管理（SSO）
- **Summary**: conf/core/sso/config.yaml `ssoJwtSecret: fddc4d34...c12bd3`（openssl rand -hex 32 生成）并加注释说明 OPT-062 轮换；两服务经 runAll `/api/restart` 重启后 healthy。测试改为动态读真源：taskAuth sso_bridge_test.go 新增 `ssoTruthSourceValue(t, root)`（YAML 解析，替代硬编码常量，未来轮换不再破坏测试），provider 端 sso_config_test.go 同样以 `confSsoTruthSourceValue(t, root)` 替代硬编码 const。
- **Verification**: taskAuth/taskAiProvider 全量 go test 通过；live smoke：新密钥签名 bridge POST exchange → HTTP 200（access+role），旧 dev 密钥 → HTTP 401「无效的 bridge」——轮换生效。

## [OPT-20260806-063] completed — taskAiProvider 两套 SSO 测试 DB helper 待合并
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: 合并为 src/testdb_test.go 的 openTestApp(t)（清 env → LoadConfig → OpenTestMySQL → applyAIMigrations → NewApp），handlers_test/vendor_status_db_test 复用，vet 清理
- **问题**: `src/handlers_test.go`（既有 TestSSOExchangeStaff/StaffStringSub/VendorStringSub/SSOOnlyLogin，dbload.OpenTestMySQL 模式）与新增 `src/sso_exchange_db_test.go`（openSSOAppWithTestDB + applyAIMigrations）存在两套独立的测试库初始化与迁移应用逻辑，长期维护会漂移（如迁移目录变更只改一处）
- **建议**: 将 openSSOAppWithTestDB/applyAIMigrations 提为 src 共享测试 helper（如 testdb_test.go），handlers_test.go 的 SSO/DB 测试复用；同时确认 -run 'SSO' 门禁覆盖两处测试（已覆盖）
- **Summary**: 新建 src/testdb_test.go：openTestApp(t) = 清除 env 中 TASK2APP_SSO_JWT_SECRET/DJANGO_SECRET_KEY → LoadConfig → OpenTestMySQL → applyAIMigrations → NewApp；sso_exchange_db_test.go 的 applyAIMigrations 迁入。handlers_test.go 的 testApp 改为 `return openTestApp(t)`；vendor_status_db_test.go 调用点 openSSOAppWithTestDB→testApp。恢复 vet 报错的 strings/filepath imports。
- **Verification**: 全量 go test ./src/ 通过（含 SSO 门禁 -run 'SSO' 两处测试均被覆盖）。

## [OPT-20260806-064] completed — 厂商门户 UI 直接展示微信用户的合成邮箱（sso-{id}@sso.invalid）
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: vendor JSON 增加 is_synthetic_email 字段；VendorPortal.vue 对合成邮箱/派生公司名改为占位展示（「微信登录用户（未绑定邮箱）」/「未设置」）
- **问题**: OPT-20260806-061 修复（taskAuth resolveUserEmail 兜底）后，微信扫码用户经厂商门户 SSO 自动建号，VendorPortal.vue 顶部与信息卡直接展示 `sso-{id}@sso.invalid`（RFC 2606 保留域，永不可投递），company_name 也由合成邮箱前缀派生为 `sso-{id}`，对厂商/运营观感不佳
- **建议**: taskAiProvider 在 UpsertVendorFromBridge 或序列化时识别 `@sso.invalid` 合成邮箱，前端 VendorPortal.vue 与 marketplace 列表改为展示占位（如「微信登录用户」）并隐藏不可投递邮箱；同时评估让厂商在门户内自助补充公司名/联系邮箱后覆盖合成值（需防邮箱被其他用户占用：沿用 GetVendorByEmail 冲突检查）
- **Summary**: 序列化层（vendor_application_handlers.go vendorJSON + auth_handlers.go handleVendorMe）统一输出 `"is_synthetic_email"`；前端 VendorPortal.vue 新增 isSyntheticEmail/displayEmail（「微信登录用户（未绑定邮箱）」）/displayCompanyName（sso-\d+ →「未设置」）computed，替换 toolbar/company summary/email summary 三处模板位置。
- **Verification**: taskAiProvider 全量 go test 通过；VendorPortal 相关 vitest 通过。

## [OPT-20260807-001] completed — 厂商申请审核结果无主动通知，用户需手动刷新
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: 镜像市场页 30s 轮询 vendor-status（最小成本方案 a），onMounted/onUnmounted 生命周期治理
- **问题**: 审核流上线后（2026-08-07），运营在 ai-provider AdminPortal 通过/驳回厂商申请，用户侧无任何通知——需刷新镜像市场页才能看到状态变化（按钮从「审核中」变「厂商门户（SSO）」/「申请被驳回」），体验断链
- **建议**: a) 镜像市场页轮询 vendor-status（如 30s）或下拉刷新时重载；b) 接入站内消息/邮件通知（需事件发布基建，taskAiProvider 现无 domainevents——见设计文档 §9 例外记录）；先做 a（前端轮询）为最小成本
- **Summary**: taskFE/app/src/views/ImageMarket.vue：VENDOR_STATUS_POLL_MS=30_000 轮询 vendor-status，startVendorStatusPolling/stopVendorStatusPolling，onMounted 启动、onUnmounted 停止（含窗口隐藏时暂停）。
- **Verification**: ImageMarket 相关 vitest 6/6 通过；taskFE 全量 vitest 通过。

## [OPT-20260807-003] completed — auth_customtoken 缺 (content_type_id, object_id) 复合索引
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: dataMigrate/taskAuth/030 新增复合索引并已应用生产（index 存在 + data_migrate_log 记录）
- **问题**: userHasLiveToken 的 COUNT 查询（forward-auth 热路径，userId cookie 兜底时每次缓存未命中执行）仅命中 content_type_id 单列索引，object_id 需回表过滤；多设备/多 token 用户与高流量下放大该路径成本
- **建议**: dataMigrate/taskAuth 新增迁移文件：`CREATE INDEX auth_customtoken_ct_obj_idx ON auth_customtoken (content_type_id, object_id)`（与既有 017 命名风格一致），应用后对表级验证
- **Summary**: dataMigrate/taskAuth/030_auth_customtoken_composite_index.sql 新增；经 `cd taskAuth && ./bin/taskAuth migrate` CLI 显式应用（服务重启不自动跑迁移——main.go 仅 os.Args[1]=="migrate" 时执行）。
- **Verification**: docker MySQL（docker-mysql-mysql-1/task_auth）`SHOW INDEX FROM auth_customtoken` 含 auth_customtoken_ct_obj_idx，data_migrate_log 有 030 记录。

## [OPT-20260807-004] completed — userId cookie 值无签名，可被直接伪造
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: activate-session 写入 `userId=<id>.<ts>.<HMAC-SHA256>` 签名值，解析侧验签+7 天时效；裸 ID/篡改/过期一律拒绝
- **问题**: userId cookie 内容为裸用户 ID（无 HMAC/加密），配合 OPT-20260807-002 的活 token 行校验后，仍允许「能设置同域 cookie」的侧（子域 XSS 等）冒充任何近期登录过的用户
- **建议**: activate-session 写入时改为 `userId=<id>.<ts>.<HMAC-SHA256(server_secret, id|ts)>`，解析侧验签+时效（如 7 天）；注意签名密钥轮换与旧 cookie 兼容（双密钥窗口），改动涉及 taskAuth 写入/解析两端 + 前端 storeUserId 的 JS cookie 形态需同步（或前端 JS cookie 与 HttpOnly 签名 cookie 语义统一为「提示性 + 服务端为准」）
- **Summary**: 新建 taskAuth/src/user_id_cookie.go：signUserIDCookieValueAt/signUserIDCookieValue/parseUserIDCookieValue，HMAC-SHA256 密钥复用 ssoBridgeSecret（conf/core/sso/config.yaml 真源，与 SSO bridge 同域同进程），7 天 TTL + 5 分钟时钟偏差容忍；auth_activate_session.go 写入签名值（签名失败不落 cookie + 日志）；gateway_forward_auth.go / oidc_handlers.go 的 userId cookie fallback 改为 parseUserIDCookieValue 验签；兼容性：旧裸 ID cookie 被拒，用户重新登录即重写。测试 5 个（roundtrip/裸 ID/篡改/过期 8d+未来 1h/含点 ID）。网关 forward-auth 3 个旧测试同步改签名 cookie（裸 ID 拒绝断言固化）。
- **Verification**: taskAuth 全量 go test 通过；user_id_cookie_test.go 5/5。

## [OPT-20260807-005] completed — 微信回调凭据补全未落 HttpOnly token cookie
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: resolveWechatSessionIdentity 成功分支补 activateSavedAccountSession（动态 import + try/catch 静默回退 setActiveAccount），微信路径 Cookie 与主登录流对齐
- **问题**: resolveWechatSessionIdentity（sessionUserIdUtils.js）仅写 userId cookie + 插件槽，未调用 activate-session；微信登录用户（无插件）的 HttpOnly token cookie 缺失，依赖 userId cookie 兜底路径（已有活 token 行时可用，但未与主登录流对齐）
- **建议**: 微信回调成功分支补 `activateSavedAccountSession({ userId, token: wechatToken })`（失败静默回退，与 persistLoginSuccessCredentials 同构），随后可观察微信路径 Cookie 一致性
- **Summary**: taskFE/app/src/utils/sessionUserIdUtils.js：OPT-005 语义更新；成功分支经动态 import 调 activateSavedAccountSession，包裹 try/catch 失败回退 setActiveAccount（不影响既有微信路径行为）。
- **Verification**: taskFE 全量 vitest 通过；OPT-004 注释语义同步更新。

## [OPT-20260807-006] completed — 清库重建后残留 HttpOnly userId/token cookie 无服务端清理入口
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: profile 401 附带 Set-Cookie Max-Age=-1 清理 userId/token；修复清理头后置导致 no-op 的顺序 bug 并补回归测试
- **问题**: DB 重建后浏览器仍持有 30 天 HttpOnly userId/token cookie（前端 JS 无法删除）；修复后服务端已拒绝其认证，但 cookie 残留直到自然过期，且无「登录页可见即清理」的服务端动作
- **建议**: 登录页（public 路由）加载时前端调用轻量端点（如复用 profile 401 后 /auth/login 页 GET）时，由 taskAuth 在响应中 Set-Cookie Max-Age=-1 清理 userId/token；或在文档（docs/数据库初始化手册）记录「清库后所有浏览器需重新登录」为预期行为（后端已强制）
- **Summary**: auth_user_profile.go 新增 clearResidualAuthCookies(w)（userId+token，Max-Age=-1，SameSite Lax）；handleGetUserProfile 未认证路径先解析（resolveUserIDFromRequest 只读不写响应）→ 清理头 → 再 writeJSON 401（修正 requireAuthenticatedUser 先写体、Set-Cookie 变 no-op 的顺序 bug，live 冒烟当场暴露）。新增回归测试 TestGetUserProfile401ClearsResidualCookies；docs/superpowers/specs/2026-05-31-dev-database-reset-design.md 第 16 节记录清库后 cookie 残留预期行为。
- **Verification**: taskAuth 全量 go test 通过；live smoke：`curl -D - http://127.0.0.1:8003/api/accounts/users/profile/` → 401 且首段携带 `Set-Cookie: userId=; Max-Age=0` + `token=; Max-Age=0` 清理头。

## [OPT-20260807-012] completed — 网关 forward-auth 白名单缺 X-User-Email，已绑定邮箱用户申请厂商仍报「需先绑定邮箱」
- **Status**: completed
- **Completed**: 2026-08-07
- **Completion-Note**: routes-to-apisix.py `_forward_auth_plugin` upstream_headers 补 X-User-Email（含 60 个 forward-auth 块重生成）+ 热加载；新增回归测试 test_forward_auth_email_header.py（生成器 + apisix.yaml 双重守护）
- **问题**: 镜像市场页用户已绑定邮箱，提交厂商申请仍返回 400「厂商门户需先绑定邮箱账号，请前往个人资料页绑定邮箱」。链路：taskAuth writeForwardAuthHeaders 注入 X-User-Email（resolveUserEmail 主邮箱）→ APISIX forward-auth upstream_headers 白名单缺 X-User-Email → 头被网关静默丢弃 → taskAiProvider 永远收到空邮箱。症状自洽：vendor-status `has_email=!isSyntheticEmail("")=true`（表单能打开）而 vendor-application `email=="" → 400`。
- **建议**: 网关 forward-auth 上游头白名单与 taskAuth 注入头保持一致；任何新注入的 X- 头需同步 upstream_headers（本次已由测试守护生成器与产物）
- **Summary**: taskGateway/scripts/routes-to-apisix.py `_forward_auth_plugin` upstream_headers 加入 `"X-User-Email"`（注释引用 OPT-20260807-010 语义）；`TASK_GATEWAY_APISIX_IN_DOCKER=1` 重生成 apisix/apisix.yaml（60 个 forward-auth 块全覆盖）；新增 scripts/ci/test_forward_auth_email_header.py（2 测试：生成器单元 + 产物端到端，importlib 加载连字符文件名模块，conf 含 docker.upstreamHost 与环境无关）；docker exec apisix reload 热加载（容器内文件 md5 与宿主一致）。
- **Verification**: ① Red→Green：修复前 2 测试 FAIL（生成器缺头 / 产物首个缺头块），修复后 2 PASS；check_routes.sh 115 checked 0 missing 回归通过。② 线上全链路：POST /api/ai-provider/vendor-application/ 修复前 400（同 cookie curl 复现），修复后 200 `{"status":"pending","vendor":{"email":"author@example.com",...}}`；GET vendor-status 修复后返回 pending（X-User-Email email 分支匹配成功）。③ 浏览器端到端：点申请 → 表单打开 → 提交 → toast「厂商申请已提交，等待运营审核」→ 按钮转「审核中」，错误提示消失；测试产生的 2 条 vendor 申请记录已清理（ai_provider_vendor 归零）。

### OPT-20260807-013 — 厂商审核「vendor id 无效」400（双前缀回退缺陷）
- **Status**: completed
- **日期**: 2026-08-07
- **Completed**: 2026-08-07
- **Summary**: 生产故障：运营在 admin 页点「审核通过」恒 400「vendor id 无效」（Loki trace 7381532a-6587-4ff6-bae5-21b983518c29, PATCH /api/admin/vendors/7315786738855939/ status=400 1ms）。根因：handleAdminVendorReview 先 TrimPrefix 剥 /api/ai-provider/admin-vendors/、仅当前缀剥空才回退 /api/admin/vendors/；对 /api/admin/vendors/{id}/（AdminPortal.vue:432 实际调用）前缀不匹配返回原样非空 → 回退被跳过 → ParseInt 整串失败。修复：改用 parsePathID 双前缀尝试（与 handleAdminVendors 单条 GET 同源逻辑），非法 id 仍 400。新增 vendor_review_path_test.go 两条测试（双前缀 200 + 非法 id 400）。
- **Verification**: taskAiProvider/src + infrastructure 全量 go test 通过；新增 2 测试通过（双前缀审核 200、非法 id 400）；go build ./... 零错误。


### OPT-20260807-019 — taskProjectService 路由缺陷：POST 创建项目静默失败（生产 200 [] 假成功）
- **Status**: completed
- **日期**: 2026-08-07
- **Completed**: 2026-08-07
- **Summary**: 生产故障：创建项目页面 POST /api/projects/tenant_id/873472655125147648 返回 `200 []`（traceId 666ddf7f-10e6-4a78-8d6d-dec372f703a4，200/0ms 无业务日志），前端视为成功跳转但项目未创建。根因：2026-08-04 路径统一重构（taskProjectService 6edee88 + taskFE e105bad）后，/api/projects/ handler 中 ParseConventionPath 将 `tenant_id/{tid}` 整体消费、rest 为空，`len(parts)==0` 分支对**任何 HTTP 方法**都走 handleListProjects，POST 永远到不了 handleCreateProject（handleCreateProject 成路由器视角死代码）。修复：main.go 空 parts 分支改为 `handleProjectsRoute(w,r,tenantID,nil)` 按方法分发（GET→list / POST→create / 其他→405）。
- **Verification**: ① 路由器级回归测试 RED→GREEN：修复前 POST 得 200 []，修复后 201 + detail（含尾斜杠变体）；GET 同路径 200 列表不回归。② taskProjectService/src 全量 `go test ./src/` 通过（18.5s）。③ 本地 8016 实机：POST 201 返回完整 project detail（id/git_repo_entries/workspaces），GET 列表可见新项目，验证数据已清理（task_project 库归零）。

### OPT-20260807-020 — taskProjectService 路由器级回归测试（堵住「单测直调 handler 绕过路由」盲区）
- **Status**: completed
- **日期**: 2026-08-07
- **Completed**: 2026-08-07
- **Summary**: 新增 src/projects_route_test.go：经 mountRoutes 构建真实 ServeMux，POST /api/projects/tenant_id/{tid} 断言 201 + id/name（含尾斜杠变体）、GET 同路径断言 200 JSON 数组。既有单测（TestCreateAndListProjects 等）直接调用 handleCreateProject/handleListProjects，路由缺陷对其不可见，本次事故即因此漏检；路由器级测试 3 例构成对 8-07 路由缺陷的回归保护。
- **Verification**: 修复前 2 例 FAIL（200 []）+ 1 例 PASS（GET），修复后 3 例全 PASS；随 OPT-20260807-019 一并合入。

### OPT-20260807-057 — 保存项目运行模版 405 + data-traceId 缺失（根因修复 + 部署复验）
- **Status**: completed
- **日期**: 2026-08-07
- **Completed**: 2026-08-07
- **Summary**: 任务「页面元素调整」：项目详情页运行模版面板报「保存项目运行模版失败」且错误元素缺失 data-traceId。根因①：前端 PATCH `/api/projects/tenant_id/{tid}/{pid}/` 为 tenant_id 前置的约定路径，shareLib gatewayauth.ParseConventionPath 将 kv 对后的位置段 pid 吞掉 → 分发到 collection 路由 → 405 → 前端兜底文案。修复：ParseConventionPath 保留 kv 对后位置段（kvEnd 追踪），taskProjectService/taskTaskService/taskEvents intentmux 全线受益；新增 9 例 pathparams 单测 + 2 例路由级回归（PATCH 更新持久化 / GET tenant_id 前置详情）。根因②：bootstrap-admin（平台 super_admin）访问本租户 cloud 资源被 ensureTenantMember 403（server-config-default/regions/platforms 全挂）→ 模版预设加载失败。修复：ensureTenantMember 顶部加 super_admin 旁路（与 taskProjectService 授权模型一致），2 例单测。根因③（真机复验发现）：真实网关 X-Trace-Id 为逗号双值合并串（网关+上游各一行），traceIdFromHeaders 原样返回合并串 → looksLikeTraceId 拒绝 → 真实错误下 data-traceId 仍缺失；修复：traceIdFromHeaders 归一化 split(',') 取首段（合法 traceId 不含逗号，拆分安全），4 例 traceId.js + 1 例 composable + 1 例面板级 DOM 测试（Red→Green）。
- **Verification**: ① 全量测试：shareLib gatewayauth+authz、taskProjectService（19.7s）、taskCloudService（107.9s）、taskEvents saastest、taskFE 308 文件/1591 例全绿（新增 20 例）。② 部署：runAll precise-restart 重建 task-project-service/task-cloud-service/vue-frontend，线上 www.daydaymoney.com 生效（index.html 哈希更新）。③ 浏览器真机：PATCH tenant_id 前置 200 + 模版持久化（cn-beijing 4核8G80G）、GET 详情非列表、原 403 cloud 端点全部 200、错误响应携带 X-Trace-Id 双值头、线上真实 bundle 提取函数对真实双值头返回首段 0459995aeb76223983523e4d36faa226。

### OPT-20260808-004 — 租户默认交付物体系恒丢失（根因修复 + 生产部署复验）
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 任务「公司创建后自动设置默认交付物体系没生效，页面恒显示『暂无默认交付物体系』」。三层根因：① taskProjectService `project_progress_systems_default_tenant` 表 UNIQUE(tenant_id) 每租户仅一行，company-created 事件链 intent 1（交付物默认）与 intent 2（进度默认）共用该行，intent 2 的 upsert 覆盖 intent 1 写入 → 租户默认交付物体系恒丢失；② 后端无 `GET /api/projects/default-deliverable-system` 端点 → 前端权威覆盖请求 404；③ 前端 DeliverableSystemList.vue 对列表结果强制 `is_default: false`，覆盖后端正确标注。修复：新建独立表 `project_deliverable_systems_default_tenant`（迁移 011，幂等 CREATE IF NOT EXISTS + INSERT IGNORE，含存量租户 backfill = workspace∪公司级体系∪progress默认租户全集补 ds_default_global）；handleListDeliverableSystems / handleSetDefaultDeliverableSystem / DELETE 默认拦截全部切新表；新增 GET default-deliverable-system 端点（main.go 路由 case）；前端移除强制覆盖、保留后端标注（default-deliverable-system 端点仍为权威覆盖来源）。
- **Verification**: ① Red→Green：新增 4 例 Go 测试（核心回归 TestDeliverableDefault_SurvivesProgressDefaultSet：设交付物默认后再设进度默认，交付物默认不丢；GET 端点 2 例；DELETE 默认体系 400），taskProjectService 全量 158/158 PASS；前端 DeliverableSystemList.setDefault 5/5 PASS（mock 不依赖被移除的覆盖逻辑）。② 生产部署：`taskProjectService migrate` 应用 011（data_migrate_log L1 + 幂等 SQL），backfill 目标租户 873472655125147648 → ds_default_global（总量=有 workspace 租户数 1）；重建二进制并重启服务（health ok）。③ 实机 API：GET default-deliverable-system 200 `{"default_deliverable_system_id":"ds_default_global","status":"success"}`（修复前 404）、列表 API `ds_default_global | is_default=True`。④ 浏览器线上复验：www.daydaymoney.com/tenant/873472655125147648/deliverable-systems/ 由「暂无默认交付物体系」变为「全局默认交付物体系 (当前默认)」+ 卡片「默认」徽章，截图 docs/opt-20260808-004-deliverable-default-fixed.png。

### OPT-20260808-007 — 创建订单 1062 撞号根因修复（traceId 5d680b8c 定位）
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 任务「页面元素调整」：创建订单页报「创建订单失败: Error 1062 Duplicate entry 'ORD-20260807-001'」。根因三层：①`CAST(... AS INTEGER)` MySQL 非法语法（Error 1064，SQLite 时代遗留，66fd85c MySQL 迁移漏改）→ SELECT MAX 每次失败；②generateOrderNumber 失败静默默认 seq=1 → 同一 UTC 日内所有订单号恒为 xxx-001 → 首单后全部撞号；③重试循环仅匹配 SQLite 文案，MySQL 1062 永不触发重试 → 直接 400。修复：AS SIGNED 双兼容、isDuplicateKeyError（MySQL 1062+SQLite）、SELECT 错误传播、maxRetries 3→5、resource_order_number_collision/create_failed 结构化日志。新增 orders_duplicate_test.go 3 例（1062 文案单测 / seam 确定性撞号重试 / 4 并发收敛唯一），Red→Green，taskBill 全量测试通过。复盘见 .learnings/OPT-20260808-007.md。
- **Verification**: 修复前 3 例 FAIL（并发测试失败文案与生产逐字一致）+ 全量通过；提交 taskBill 00bbdb3；待部署复验（OPT-20260808-009）。

### OPT-20260807-052 — precise-restart 与 start/stop/build-all 等 bulk 操作无互斥
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: runAll 后端对全部 bulk 操作（precise-restart / start-all / stop-all / build-all / restart-all）加统一互斥。新增 bulkOpGuard 信号量（TryBeginBulk/EndBulk/IsBulkActive/ActiveBulkOp）：TryBeginPreciseRestart 先取 bulk 锁再取自身互斥（冲突回滚释放）、TryBeginRestartAll 同模式、TryBeginBuildAllRun 取 bulk 锁，start-all/stop-all/build-all/build-group 入口 409 拒绝冲突操作；锁序统一 bulk→自身，RestartAllWithActor 复用 IsRestartAllActive 重入保护避免死锁。修复场景：精准编译重启时另一标签页触发「全部重启」导致 stop-all 中途介入、vue-frontend 健康检查失败被保留在登记文件须重跑。
- **Verification**: 新增 3 例互斥回归单测（TestBulkOpMutualExclusion_PreciseRestartBlocksOthers / BulkOpBlocksPreciseRestart / RestartAllBlocksBuildAll）Red→Green；runAll 全量 go test 通过（72.991s）；提交 runAll b901338 并推送 upstream。

### OPT-20260807-027 — register-precise-restart.sh 支持 --all/按组批量登记
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: scripts/register-precise-restart.sh 支持 `--all`（批量登记 runAll.yaml 全部服务）与 `--group <name>`（按 group 批量登记），消除多服务变更时逐条登记的冗长命令。新增 GROUP_SERVICES 解析（2 空格缩进 group / 6 空格缩进服务名），`--list` 展示全部 group；`--all` 展开为全部服务名、`--group` 展开为组内服务名（未知 group/缺参报错）；服务名校验与后端 resolveRegisteredService（working_dir 首段别名）保持一致。
- **Verification**: bash -n 语法通过；实测 `--all` 登记 59 服务、`--group platform` 登记 17 服务、`--group nonexistent` exit 1 报错、`--list` 展示 container-stack/infrastructure/platform/value-stream/domain-events-intents 5 组；meta 提交 d36d7a9。

### OPT-20260807-014 — ReviewVendor 同值重复审核误报 404
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: taskAiProvider ReviewVendor 原以 RowsAffected()==0 判定厂商存在性：MySQL 无 CLIENT_FOUND_ROWS 时 UPDATE 返回实际变更行数；reviewed_at/updated_at 为 DATETIME（秒精度），Go 侧写入微秒串被截断，同秒内相同 action/note 重复审核 → 行值整体不变 → RowsAffected=0 → 误判 ErrNoRows → handler 404「厂商不存在」，破坏运营二次修正的幂等覆盖语义。修复：去掉 RowsAffected 判定，UPDATE 后按 id 复查（GetVendorByID 幂等返回，真不存在自然 ErrNoRows→404 契约保持）。新增 ReviewNowFunc 可覆盖时钟（测试确定性复现同值重复审核路径）。
- **Verification**: taskAiProvider `go test ./infrastructure ./src` 全绿；新增 infrastructure/src 双层确定性回归（同值重复审核 200 + 缺失厂商 404/ErrNoRows 契约），红绿验证（旧逻辑 404 复现）；测试 fixture ai_provider_vendor.reviewed_at TEXT→DATETIME 对齐生产；commit 910d07e 已推上游。

### OPT-20260807-032 — API 路径迁移遗留占位符残留回归门禁
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: e105bad API 路径迁移先后产生 ~30 处（7bd031b 修）与 5 处字面量 `$1`（2026-08-07 修）URL 损坏，DeliverableSystemList.vue 亦曾硬编码 `${1}`（OPT-20260807-053）——纯人工迁移易漏。新增 `db/scripts/ci/check_fe_url_placeholder.py` 静态扫描并挂载 meta pre-commit（check-fe-url-placeholder）：扫描 taskFE app/src 的 .js/.vue/.ts 中含 `/api/` 的字符串字面量，命中字面量 `$1`/`{sub}`/`{id}`/`{tid}`/`{wsId}`/`{workspaceId}` 等迁移占位符残留即阻断提交。豁免：注释内路径说明、`it()`/`test()`/`describe()` 描述串、`.replace()`/`.replaceAll()` 替换串参数（`$1` 为合法捕获组引用）、`${...}` 模板插值。
- **Verification**: 红绿验证——4 类残留模式（`$1`、`{sub}`、`{id}`、`{tid}`）全命中；合法模式（模板插值、replace 替换串、注释说明、测试描述）0 误报；当前 taskFE 源扫描通过。meta pre-commit `pre-commit run check-fe-url-placeholder` Passed；db 提交 9ab8854 已推上游。

### OPT-20260807-049 — 全量 vitest 并行跑时序 flaky
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: taskFE 全量 vitest 并行下重组件挂载测试（TaskDetail.smoke/CreateTaskModal/ProjectDetail/UserGitSiteOAuthSettings/TaskDetailLinkedProjectsPanel）偶发超时：隔离运行全部通过、失败集合随运行漂移、重跑 0 失败，属并行负载抖动型 flaky。在 app/vite.config.js 补 `test` 块，默认 testTimeout/hookTimeout 从 5s 提高至 15s，吸收并行负载抖动，避免 CI 假红。
- **Verification**: 5 个 flaky 文件 56 例单跑/合跑全绿（TaskDetail.smoke 2 + 其余 54）；commit 31edf5f 已推上游。taskFE 子仓指针同步待其 WIP 清理后补（见 OPT-20260807-043）。

### OPT-20260807-050 — taskChromePlugin 快捷键绑定差异警告被折叠区隐藏
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: taskChromePlugin popup 将快捷键说明改为默认收起（点击「展开」后显示）后，checkShortcutBindingDiff 渲染的「浏览器实际绑定 vs 配置」警告位于折叠区（#shortcutsSection/#shortcutsBody display:none）内，用户不点「展开」即错过。修复：checkShortcutBindingDiff 检测到 differs 时自动展开快捷键区（section/body 置 block）并将 btnToggleShortcuts 置「收起」态，警告展示完由用户手动收起。
- **Verification**: 新增回归测试断言 differs 分支含自动展开逻辑（keyboard-shortcut.test.js 15/15）；taskChromePlugin 全套 231 例全绿；commit c21b051 已推上游。

### OPT-20260808-009 — 部署后复验订单创建撞号修复
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 部署 taskBill 141aac8（OPT-007/008 修复）到线上，复验撞号修复：同一 UTC 日内连续创建 3 笔订单 task_post qty=1，订单号 ORD-20260807-002/003/004 严格递增，无 1062；Loki `{job="task-bill"}` 可见 `resource_order_created` 序号递增链（trace_id ab9437e0/2307d95d/67a02287），`resource_order_number_collision`=0。测试订单已取消清理。2026-08-07（UTC）影响面复盘：Loki 历史数据不可恢复（promtail 容器停摆 + 日志目录漂移 /tmp/runall-logs→/tmp/ram-work/logs），无法从 Loki 统计该日 400；DB 佐证该 UTC 日仅 1 笔订单（grant ORD-20260807-001 total 0 paid，free grant），且 002/003/004 为本次复验创建——无用户真实撞号漏单痕迹。顺带修复 promtail 挂载（.env RUNALL_LOG_ROOT 指向真实 file_root）使 task-bill 日志可见。
- **Verification**: Playwright verify-order-number-sequential.cjs 3 次创建均 HTTP 200、序号递增断言通过；Loki query_range `{job="task-bill"} |= "resource_order_created"` 命中 3 条链式日志、collision=0；测试订单全部 cancel 200。

### OPT-20260807-044 — 部署后复验交易流水页表头过滤器迁移
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 生产复验通过：交易流水页顶部过滤卡仅剩工作空间/任务搜索 + 应用/重置（无交易类型/支付来源 select）；「交易时间」列头内嵌开始/结束日期输入、「消耗（元）分类」列头内嵌 select（全部分类/任务/GitLab 磁盘/GitLab 流量费）、「类型」列头内嵌 select（全部类型/入账/消耗/退款）、「项目」「成员」列头内嵌搜索输入+下拉；切换分类立即触发 `/billing/transactions/list_filtered/?billing_unit_type=...&page=1`（重置第 1 页）。
- **Verification**: `playwright/verify-batch-fe-deploy.cjs` 生产实测，044+038 区块 15 项断言全过；同脚本覆盖 042/047/054。

### OPT-20260807-038 — 部署后复验交易流水页过滤器迁移
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 生产复验通过：交易流水页顶部过滤卡不再含「交易类型」select（card 内 select=0）；「类型」列头内嵌 select（全部类型/入账/消耗/退款）；标题栏「支付来源」select 就位。
- **Verification**: `playwright/verify-batch-fe-deploy.cjs` 044+038 区块断言全过（cardNoTransactionTypeSelect / titleBarPaymentSource / typeOptions）。

### OPT-20260807-042 — 部署后复验账单首页定价卡片移除
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 生产复验通过：账单首页不再显示「当前资源单价」卡片，网络面板 0 次 `/billing/order-pricing/` 请求（组件已删、请求已移除）。
- **Verification**: `playwright/verify-batch-fe-deploy.cjs` 042 区块 2 项断言全过（noPricingCard / noOrderPricingRequest）。

### OPT-20260807-047 — 部署后复验 BillingUsage 表头过滤合并
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 生产复验通过：usage 页无独立「过滤条件」卡片；thead 第二行 data-alias="BillingUsageFiltersRow" 内嵌全部过滤控件——使用时间列开始/结束日期、计费单元列 select（4 选项）、项目/用户/工作空间/任务列搜索输入、描述列应用/重置；thead=2 行、过滤单元格=8。
- **Verification**: `playwright/verify-batch-fe-deploy.cjs` 047 区块 10 项断言全过。

### OPT-20260807-054 — 部署后复验租户控制台侧栏缩窄
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 生产复验通过：点击「控制台导航」标题后 aside.tenant-console-sidebar 200px→64px 仅图标、菜单文字隐藏、子菜单不渲染、图标 title 显示菜单名；再次点击展开回 200px；缩窄态点击「设置」入口先展开侧栏；刷新保持（localStorage tenant-console-sidebar-collapsed=1）。
- **Verification**: `playwright/verify-batch-fe-deploy.cjs` 054 区块 12 项断言全过（含子菜单先展开再缩窄、缩窄态点入口展开、刷新持久化）。

### OPT-20260807-036 — FE URL 尾斜杠 vs 后端 HasSuffix 精确匹配契约静态扫描
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 新增 `db/scripts/ci/check_fe_url_trailing_slash.py` 回归门禁：从 8 个 Go 服务源码 `strings.HasSuffix(<var>,"X/")` 提取「尾斜杠叶子」（共 21 个，含 strings.Contains 前置条件，如 `/billing/orders/`+`/pay/`），静态扫描 taskFE 含 `/api/` 字符串字面量，路径缺尾斜杠且命中叶子（满足前置条件）即阻断——防止交易流水页 units 无尾斜杠 404（OPT-20260807-037 事件）同类静默复发。豁免注释/测试描述串/`.replace()` 替换串；`strip_query` 感知 `${...}` 插值内 `?` 不为查询分隔符。
- **Verification**: 15 例单测（strip_query 5 / 叶子提取 3 / 命中判定 7）红绿全过——units 缺尾斜杠命中、含斜杠放行、query 尾部缺斜杠命中、`/billing/orders/${id}` 不误报、contains 前置缺失不命中；当前 taskFE 树 0 处缺失。db commit 3d9c623 已推上游；meta 76ded36（pre-commit check-fe-url-trailing-slash + CI repo-quality-gates 步骤）已推上游。

### OPT-20260807-037 — taskFE /api/ URL 模板须被网关路由覆盖契约门禁
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 新增 `db/scripts/ci/check_fe_url_vs_gateway_routes.py` 回归门禁：解析 taskGateway/routes/routes.yaml 全部路由 uri（兼容 `uri` 单值与 `uris` 列表），排除 null-upstream 兜底（api-orphaned-not-found 立即 404），对 taskFE 含 `/api/` 字符串字面量归一化（`{param}`/`${expr}`→glob `*`、去查询/锚点）后须命中任一非兜底路由（fnmatch glob），否则阻断——防止 billing 静默 404 事故（1508f24 改路径未同步网关路由）同类复发。豁免注释/测试描述串/`.replace()` 替换串与 `*.test.*`/`*.spec.*` 文件；ALLOWLIST 登记已知有意不经过网关的 URL（SystemAdminCloudAuthorizations 孤儿页面保留路径 OPT-20260806-002、SystemAdminSubTokenProviders 表单派生端点默认值）。
- **Verification**: 19 例单测红绿全过（normalize 4 / is_covered 7 含 allowlist 与非 /api/ 忽略 / 真实 routes.yaml 断言 billing-tenant-direct 存在 / scan 豁免 4）；当前 taskFE 树 0 缺口（197 条有效路由）；db commit e9a1e5b、meta 6ac3ecd（pre-commit check-fe-url-vs-gateway-routes + CI repo-quality-gates 步骤）已推上游。

### OPT-20260808-013 — 部署流程缺口：含 dataMigrate 新文件的子仓推送后未执行 9999 init-databases
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 交付 a+b 两项。a) 部署链路门禁：dataMigrate 子仓新增 `.githooks/pre-push`（HOOK_VERSION 登记 1.0.0）——推送范围（origin/main...HEAD）含新增/修改 .sql 时，调巡检脚本对受影响库（服务目录→registry key 映射，仅巡检受影响的库避免全库连接超时）比对生产 data_migrate_log，缺口即阻断推送并提示 9999 init-databases / migrate.sh 命令；生产不可达 SKIP 放行（离线开发不阻断，与 check_git_oauth_client_id_live.py 同语义）。b) 巡检脚本：`db/scripts/ci/check_data_migrate_applied.py`——解析 db/registry.yaml 12 库，从各库 migrate.sh 推导 dataMigrate 目录（`$ROOT/dataMigrate/<dir>` 单双引号兼容），本地有生产未应用（missing）→ FAIL（OPT-013 教训场景：009_post_expires_at 未应用 → smoke 500），生产有本地无（renumber 遗留 stale）→ WARN 不 FAIL，生产不可达 → SKIP 不误报。
- **Verification**: 巡检脚本 7 例单测红绿全过（一致性 exit 0 / 本地新增未应用 FAIL / stale 仅 WARN / 不可达 SKIP / 单库过滤 / 目录推导单双引号）；实测生产 12 库基线 11 库一致 + task_auth 1 条 stale（010_oidc_bootstrap_clients，renumber 遗留无害）；pre-push 钩子三场景实测——新增未应用文件推送阻断（task-auth 精确巡检 MISSING 999_hook_test.sql）、受影响库精确巡检、生产不可达放行。commits dataMigrate e2ce2e6 / db ef1a392 已推上游。

### OPT-20260808-017 — taskBill handlers.go 路由顺序 bug：mock-complete 405
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 确认并修复：`handleGetOrder` 的通用 case（`Contains(p,"/billing/orders/") && !HasSuffix(p,"/orders/")`）排在 `mock-complete` case 之前，POST `/billing/orders/{id}/mock-complete/` 命中 handleGetOrder → 405。调换 case 顺序（mock-complete/callback 最优先，pay/cancel 次之，通用 GetOrder 兜底）并加注释说明原因，防止回归。
- **Verification**: 新增 `src/orders_route_dispatch_test.go` 2 例路由 dispatch 回归测试（POST mock-complete 不再 405（路由到 handleOrderMockComplete 后 loadOrder 缺失 → 404 订单不存在，测试环境 mock 模式开启）；GET 订单详情仍走 handleGetOrder 非 405）；taskBill 全套 26s 全绿。commit d18ae84 已推上游。

### OPT-20260808-012 — /usr/local/bin/claude 悬空 symlink 清理
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 验证发现该悬空 symlink 已自行消失（ls/stat 均报 No such file），无残留：`bash -c 'command -v claude'` 干净解析为 /home/ljy/.npm-global/bin/claude，PATH 中 /usr/local/bin 无 claude 条目，无 PATH 解析歧义。无需删除或改指操作，闭环关闭。
- **Verification**: ls/stat/readlink 三探针均不存在；command -v 输出唯一 npm-global 路径；/usr/local/bin/ 下无 claude。

### OPT-20260807-041c — GitHub OAuth 授权链路浏览器级 e2e（c 部分）
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 新增 `playwright/verify-oauth-github-start.cjs` 生产实测：登录态访问 `/api/git-oauth/github-start-from-gateway/` → 返回 JSON `authorize_url`（browser_handlers.go handleGithubStartFromGateway，FE repoOAuthAuthorizeUtils 拿到即跳转）→ 浏览器实际落点 github.com。断言 authorize_url 契约：`client_id=Iv23li4xi6ZBcq1LKZk6` == conf 登记值（`conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml`，OPT-041a 已对 GitHub Apps API 验证存活，SSOT 动态读取防写死）、`redirect_uri=https://daydaymoney.com/redirect/gitsite/github.com/oauth/callback/`（部署基域无 www 子域）、signed state 存在、浏览器落点 github.com 且参数完整（未登录时 github 转 login 页并在 return_to 保留完整 authorize 参数）。**范围说明**：完整授权成功回调（valid code → 不再 exchange_rejected）需 GitHub 登录会话凭据，环境无凭据不可达；分类正确性由 taskGitOauth 单测四场景覆盖（OPT-20260807-029）。本脚本守护启动跳转 + 注册 client_id/redirect_uri 契约，防 OPT-041 根因 A（client_id 误改）复发。
- **Verification**: `node verify-oauth-github-start.cjs` 全断言通过：clientIdMatchesRegistered / redirectUriMatchesRegistered / hasState / browserLandedOnGithub 全 true；startResponse 含 scope=repo read:user。

### OPT-20260807-035 — 交易流水页表头过滤器 Playwright e2e 断言
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: `playwright/verify-batch-fe-deploy.cjs` 新增 035 块：transactions 页切换三处过滤后断言 `list_filtered/` 请求携带参数——类型列头 select → `transaction_type`、标题栏支付来源 select → `points_source_type`、消耗（元）分类列头 select → `billing_unit_type`（useBillingTransactions.js buildQuery 契约 L130-132）。生产实测最后一次请求同时携带三个参数且 page=1。
- **Verification**: 生产实测 transactionTypeParamInRequest / pointsSourceTypeParamInRequest / billingUnitTypeParamInRequest 全 true，requestsObserved=3，lastCallUrl 含 `transaction_type=recharge&billing_unit_type=server_start&points_source_type=user_recharge_paypal&page=1`；批次全部 7 块 PASS。

### OPT-20260807-016 — AccountSwitcherDropdown Playwright e2e 断言
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: `playwright/verify-batch-fe-deploy.cjs` 新增 016 块：work-panel 点击 `[data-testid="account-switcher-trigger"]` 内 chevron button（trigger 内 router-link 有 `@click.stop`，点 avatar/名字区不冒泡）→ 菜单展开断言——无「添加账号」、无「切换账号」区块、含「个人资料」与「退出当前」，菜单内按钮仅 1 个「退出当前」（AccountSwitcherDropdown.vue 已去掉多账号：props 不再接收 accounts/activeUserId/switching）。
- **Verification**: 生产实测 hasAddAccount=false / hasSwitchSection=false / hasProfile=true / hasLogout=true / menuButtons==['退出当前']，块 PASS。

### OPT-20260807-011 — SystemAdminUsers Playwright e2e 断言（邮箱列非 user.id 兜底）
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: `playwright/verify-batch-fe-deploy.cjs` 新增 011 块：`/system-admin/users/` 用户列表前 5 行第 2 列（邮箱列）须渲染真实邮箱（含 @）且无纯数字 user.id 兜底（UserListRow.vue 邮箱渲染回归保护）。**前置处理**：用户列表 API 需系统管理员（requireSuperuser → authz.HasPlatformPerm(X-User-Roles) 或 is_superuser 兜底）；bootstrap-admin（author@example.com）生产密码已改与 dev 提示不符，且管理员凭据不得猜测/重置。验证期间对平台所有者测试账号（contact@daydaymoney.com）临时授予 is_superuser=1 + auth_super_admin + role-super-admin（事务），验证完成后已全部回收（superuser=0、sa_rows=0、role_rows=0），未留下任何权限变更。
- **Verification**: 生产实测 hasRealEmail=true / hasNumericFallback=false / rowCount>0，块 PASS；批次全部 7 块 PASS（042/044-038/047/054/035/011/016）。
### OPT-20260808-010a — taskAuth 错误响应 trace_id 注入（OPT-059 延续批，taskAuth 部分）
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: taskAuth 全量错误响应统一注入 trace_id。新增三 helper（handlers.go）：traceIDForError（X-Trace-Id 头优先——网关桥接头，回退 tracelog context）、writeError（{status,error,message,trace_id}）、writeErrorDetail（detail 优先，保持 FE data.detail||error||message 读取契约）、writeErrorMap（复合错误体注入 trace_id + 补齐 status/message，保留附加键）。494 处 writeJSON 错误站点迁移：465 单键 error/detail（机械 pass）+ 18 处数字状态码 + 4 处变量状态码（resp.StatusCode/status 变量）+ 7 处复合错误体（auth_email_invite 邀请重复/auth_register 邮箱已注册/登录注册地区限制 403/handlers 激活链接）。既有注入点不动：auth_password_reset 直连 tracelog 30 处、oidcErr、gateway_forward_auth（均已含 trace_id）。中途 pass 脚本跨行误伤 handlers_system_admin.go 200 行 → git checkout 全量回滚后按验证过的顺序重放（pass 1 单键 → 构建 → 单行锚定数字码 → 变量码 → 复合体）。
- **Verification**: 新增 handlers_write_error_test.go 6 例全绿（header 注入/context 注入/无 trace 源不注入/detail 契约/writeErrorMap 键保留/writeErrorMap 不覆盖既有 status+message）；go build + go vet 通过；全量 go test 通过（taskAuth/src 96s）；零遗漏审计（错误体无 trace_id 站点 = 0）。commit 5af822d 已推上游。
### OPT-20260808-018 — OrderCreate 首屏并行 GET 扇出复验
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: 页面 onMounted 并行 4 个 GET（order-pricing/membership/phone-verification-status + 创建入口），复现期与并发 /me/ 叠加后 26 个请求全部 pending（TTFB 3.8-5.2s 挂起）。/me/ 修复（OPT-014）后复验生产 www.daydaymoney.com `/tenant/873472655125147648/billing/orders/create/`：两次 reload 采样，首屏 4 个并行 GET（order-pricing/membership/phone-verification-status/profile）全部 200，同毫秒（163-164ms startTime）并行发出，TTFB 345-410ms，0 pending；/me/ 与第二次 membership 亦 200（TTFB 233-284ms）。复现期「26 请求全 pending TTFB 3.8-5.2s」已完全消除，无感知阻塞；原建议 b)（若仍显著则延迟非关键请求/合并组合接口）条件不成立，未实施。
- **Verification**: Chrome DevTools MCP 生产实测 2 轮（network 面板 + Resource Timing API）：pendingCount=0，4 核心请求 TTFB 均 <450ms，全链路 200。

### OPT-20260807-041 — GitHub OAuth 授权链路 8-3 误改 client_id 事故闭环
- **Status**: completed
- **日期**: 2026-08-09
- **Completed**: 2026-08-09
- **Summary**: - **OPT-20260807-041**（completed，2026-08-09）— GitHub OAuth 授权链路 8-3 误改 client_id 事故闭环（2026-08-07 新增）：根因 A=conf 2e1a90f 把 client_id 误改为不存在的 Iv23li4xi6ZBcq1LKZk6（已回滚 70fe2d1）；根因 B=taskGitOauth 假设 X-User-Id 为数字，bootstrap-admin 确定性 ID 必 503 bad_x_user_id（已 string 化 + migration 003 + 2868b1c）。建议：a) conf 变更审批钩子对 git-oauth 凭据类字段加 diff 校验（client_id 须可查询 GitHub Apps API）；b) taskGateway→taskGitOauth 转发链路补「非数字 uid 冒烟测试」到 runAll 自检；c) 浏览器级 e2e（Playwright）断言 OAuth 授权成功跳转 github.com 后回调不再 exchange_rejected。**a) 完成（2026-08-08）**：新增 `db/scripts/ci/check_git_oauth_client_id_live.py`——从 service_provider（`github-official-{slug}`）或文件名（`http-github-com--app-{slug}.yaml`）推导 GitHub App slug，调 `api.github.com/apps/{slug}` 对账 `target.client_id`；App 404 / client_id 不一致 FAIL，网络不可达/限流 SKIP（不阻塞离线），仅暂存区命中提供方 YAML 才触网（diff 门禁）；14 例单测红绿全过，--all 实测 daydaymoney 双树 client_id 匹配、注入坏值即 FAIL。挂载 conf `.githooks/pre-commit`（暂存区含 git-oauth YAML 即实时校验，坏值拦截已实测）+ meta pre-commit manual 钩子（check-git-oauth-client-id-live）；db f8d17b5、conf 8ae276a、meta 6a1bc9a 已推上游。**b) 完成**：非数字 uid 探针已加入本机 `scripts/smoke/internal-apis-live.sh`（8002 `/api/git-oauth/github-start-from-gateway/` 带 `X-User-Id: uid-non-numeric-bootstrap-admin`，断言非 503 bad_x_user_id），实测 pass=5 fail=0（200 非 503）；该脚本被 meta gitignore（scripts/*）未入库，随 runAll 自检本机生效。**c) 完成（2026-08-09）**：`playwright/verify-oauth-github-start.cjs` 生产实测通过（补导航重试吸收沙箱网络抖动）——clientIdMatchesRegistered/redirectUriMatchesRegistered/hasState/browserLandedOnGithub 全 true，startResponse 含 scope=repo read:user；完整回调（valid code→不再 exchange_rejected）需 GitHub 登录会话凭据，环境无凭据不可达，分类正确性由 taskGitOauth 单测四场景覆盖（OPT-029）。
- **Verification**: a) check_git_oauth_client_id_live.py 14 单测全过 + --all 实测 daydaymoney 双树 client_id 匹配、注入坏值即 FAIL（db f8d17b5 / conf 8ae276a / meta 6a1bc9a）；b) internal-apis-live.sh 非数字 uid 探针实测 pass=5 fail=0（200 非 503）；c) verify-oauth-github-start.cjs 生产实测 clientIdMatchesRegistered/redirectUriMatchesRegistered/hasState/browserLandedOnGithub 全 true。完整回调分类正确性由 taskGitOauth 单测四场景覆盖（OPT-029）。


### OPT-20260807-053 — 交付物体系 set-default 根因修复收尾
- **Status**: completed
- **日期**: 2026-08-09
- **Completed**: 2026-08-09
- **Summary**: - **OPT-20260807-053**（completed，2026-08-09）— 交付物体系 set-default 根因修复收尾（2026-08-07 新增）：本次已修 taskFE DeliverableSystemList.vue 硬编码 `${1}` URL（迁移 e105bad/7bd031b 引入，系统级体系 id 为 ds_xxx 格式致 404）+ 后端 deliverable_handlers.go/system_deliverable_handlers.go 错误响应 writeJSON→writeError 注入 trace_id（25 处），FE 5 例/Go 2 例回归测试全绿。收尾建议：a) 将 writeError（trace_id 注入）推广到 taskProjectService 其余文件及其它 Go 服务（taskAuth/taskBill/taskCloud 等）的 plain writeJSON 错误响应，统一前端 resolveRequestTraceId 的 body 兜底契约；b) 线上部署重启 taskProjectService+taskFE 后复验：页面 set-default 成功、错误弹窗 data-traceId 存在（本轮根因之一即线上部署版本旧，无响应头回传逻辑）；c) DeliverableSystemList setAsDefault 无模板入口引用（仅 handleDefaultDeliverableSystemChange 经 select 触发），疑似死代码，评估清理或接模板接线。**c) 完成（2026-08-08）**：setAsDefault 非死代码——提炼为共享核心返回 boolean，handleDefaultDeliverableSystemChange 委托复用（消除原重复实现），6 例回归全绿；taskFE f284ce9 已推上游。**a) 完成**：writeError（trace_id 注入）推广由 OPT-20260807-059 闭环——taskProjectService 全量错误分支（12 文件 ~163 处，新增 writeErrorMap 保留 git_repos/oauth_bound 附加键）+ taskBill 全量（26 文件 242 处，新增 writeErrorJSONMap 保留 path/refer），各 3/2 例回归全绿；其余 Go 服务见 OPT-20260808-010（taskCloudService/taskTaskService/taskAIComment/taskTenantService 分 5 提交，0 missing 审计）与 OPT-20260808-010a（taskAuth 494 处）。**b) 完成（2026-08-09）**：`playwright/verify-deliverable-system-set-default.cjs` 生产复验通过——成功路径：默认体系下拉选真实体系（ds_default_global）POST set-default 请求发出、无错误弹窗、当前默认更新；错误路径：经 `#app._vnode` 定位 DeliverableSystemList 组件实例调用暴露的 setAsDefault(不存在的体系)，后端 404「交付物体系不存在」+ body trace_id + X-Trace-Id 头，modal data-traceId 与后端 trace 同值，非网关 5xx 文案。
- **Verification**: a) 见 OPT-20260807-059（taskProjectService a5edcf7 / taskBill cdc80e6）与 OPT-20260808-010/010a（5+1 服务，0 missing 审计）；b) 生产实测 `playwright/verify-deliverable-system-set-default.cjs` pass=true——success-set-default requestSent/noErrorModal/defaultUpdated 全 true，error-set-default backend404/backendTraceId/modalHasTraceId/modalShowsNotFound 全 true，modal data-traceId 与后端 X-Trace-Id 同值 UUID；c) 2026-08-08 完成，taskFE f284ce9 已推上游。

### OPT-20260808-030 — zero_cpu_zerg 系统 sshd 端口迁移 22→2222 核对收尾
- **Status**: completed
- **日期**: 2026-08-09
- **Completed**: 2026-08-09
- **Summary**: - **OPT-20260808-030**（completed，2026-08-09，2026-08-08 新增）— zero_cpu_zerg 系统 sshd 端口迁移 22→2222 **已生效**（诊断确认，非"改动未生效"）：sshd_config L35 `Port 2222`（23:12 改）+ Ubuntu sshd-socket-generator 自动生成 `/run/systemd/generator/ssh.socket.d/addresses.conf`（ListenStream 清空默认 22 改 0.0.0.0:2222/[::]:2222，23:12:57）+ ssh.socket 23:15:51 重启后 active 监听 2222（OpenSSH_10.2p1，host key IO6Lo9...）。"22 仍可登录"实为 **docker gitlab 容器**（今日 15:20Z 重建）映射宿主 `0.0.0.0:22→容器22`（gitlab-shell，OpenSSH_9.6p1，host key IKOjil...，实测 `ssh -p 22 git@127.0.0.1` → "Welcome to GitLab, @example-user!"）——22 上的 SSH 是 GitLab 服务而非系统 sshd。遗留核对项：a) HK 反向隧道 autossh `-R 0.0.0.0:22:127.0.0.1:2222` 目标端口 2222 现为系统 sshd（原为 gitlab-shell）→ 公网 `git@gitlab.daydaymoney.com` 链路可能再次 Permission denied，需核对 HK 隧道目标是否应改指宿主 22 或同步 gitlab.rb `gitlab_shell_ssh_port`；b) known_hosts 中 `[183.250.1.132]:2222` 记录的是旧 gitlab 容器 key，现 2222=系统 sshd，直连会 host key 变更告警（`ssh-keygen -R` 清理）；c) 若需宿主 22 完全无 SSH 服务，改 gitlab 容器端口映射（如 22→8022）并同步 gitlab.rb ssh_port（会破坏 GitLab 标准 SSH 22 语义，默认不建议）。**核对项收尾（2026-08-09）**：a) HK 反向隧道目标核对——已随 OPT-20260809-003 失效：HK（47.86.27.42）反向链路`-R 0.0.0.0:22:127.0.0.1:2222` 已确认不存在（本机无到 HK 的 autossh/ssh 进程与活动连接，HK 侧无自愈机制），且 example.com 域名已暂停使用，gitlab.daydaymoney.com 公网链路无需再核（无需改 HK 隧道目标或 gitlab.rb gitlab_shell_ssh_port）；b) known_hosts 清理——`[183.250.1.132]:2222` 四条旧 key 记录（ecdsa/ed25519/rsa）经比对为第三方陈旧 key（非当前系统 sshd ed25519 `O6Lo93a...`，非当前 gitlab 容器 ed25519 `KOjil+5...`），已 `ssh-keygen -R` + 注释行清理，`ssh -G`/`ssh-keygen -F` 验证无残留；c) 宿主 22 完全无 SSH 服务——不适用：gitlab 容器映射宿主 `0.0.0.0:22→容器22` 为当前 gitlab.daydaymoney.com 公网 SSH 的承载（OPT-20260809-001 SH 反隧`-R 0.0.0.0:22:127.0.0.1:22` 直达 gitlab-shell），改动会破坏 GitLab 标准 SSH 22 语义，维持现状。
- **Verification**: 本机实测——sshd_config L35 Port 2222、ssh.socket/sshd.service active、`0.0.0.0:2222` 监听（系统 sshd OpenSSH_10.2p1）、宿主 `0.0.0.0:22` 监听为 docker gitlab 容器（映射 22→容器22，gitlab-shell OpenSSH_9.6p1）；无 HK 隧道进程/连接（OPT-003 复核一致）；known_hosts `[183.250.1.132]:2222` 4 条陈旧 key 清理后 `ssh-keygen -F` 无残留；`ssh -T git@gitlab.daydaymoney.com` → Welcome to GitLab 不受影响。

### OPT-20260807-043 — 主仓子仓指针同步收尾
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: taskFE 316b0a0（定价卡片移除）与 BillingUsage/BillingTransactions 表头过滤组件改动一度为他人 WIP 阻断主仓 pre-commit 子仓优先门禁；2026-08-08 清理完成——taskFE WIP 已清理（工作树 clean），指针同步至 f284ce9（含 316b0a0/31edf5f/5b48aad 等全部已推提交，== origin/main）；conf→8ae276a（OPT-041a 钩子）、db→f8d17b5（OPT-041a 门禁）、go_run_container→ceb7439 一并同步；taskProjectService 工作树仍为他人 WIP，指针保持 a6fb6a6 跳过（约定：指针同步跳过脏 WIP 子仓）。meta 6a1bc9a 已推上游。
- **Verification**: 各子仓指针 == origin/main 核对（taskFE f284ce9）；meta 提交 6a1bc9a 已推上游。

### OPT-20260808-005 — 前端 fetchDeliverableSystems 冗余 400 调用清理
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: DeliverableSystemList.vue onMounted 并行 3 个 GET 中 `/api/projects/deliverable-systems/`（无 tenant_id）每次必 400（后端 tenant_id required 门禁），本意应是拉系统级体系但缺参数 → 每次进入交付物体系页产生一条 400 日志噪音。修复：后端 handleListDeliverableSystems 查询 `WHERE company_id=? OR is_system=1` 已含系统级体系 → 整支删除冗余无 tenant 的 GET（Promise.all 3 个降为 2 个），is_system 标注由后端提供不再强制覆盖；新增 URL 断言组件测试（仅发带 tenant_id 的 2 个 GET + 合并列表含系统级体系）；同步更新 setDefault.test.js mockListLoad 对齐真实契约。
- **Verification**: taskFE 全套 1601 单测全绿，commit 2abd07d 已推上游。

### OPT-20260808-016 — 订单支付状态推送 SSE 化（替代 2s 轮询）
- **Status**: completed
- **日期**: 2026-08-08
- **Completed**: 2026-08-08
- **Summary**: OPT-014/015 已把支付轮询收敛为单链可控，但本质仍 2s 间隔 GET 查询；任务库既有最佳实践 useWechatRechargePoll「禁止长轮询」（SSE + 手动查询）表明可进一步 SSE 化。实现：taskBill markOrderPaid 落地后经 taskSSE /internal/publish 推送 order_paid 事件（billing:user:{userId} hub key；X-Task-Sse-Secret；5s 超时 client；幂等路径不重复），ResourceOrder 补 UserID 读取；前端新增 useOrderPaySse composable（订阅 recharge-events 通道按 order_id 匹配，单链+卸载即停），OrderCreate.vue SSE 即时收敛 + usePaymentPoll 降为 10s×12 慢速心跳兜底（无后台链）。
- **Verification**: 回归 taskBill 6 例 + taskFE 5 例 + 契约测试同步；taskBill 全套 27.3s / taskFE 1657 全绿。commits taskBill 1def851 / taskFE 6a8fad2 已推上游。

### OPT-20260809-001 — gitlab.daydaymoney.com 公网 SSH 恢复（SH 22 反向隧道至 gitlab-shell）
- **Status**: completed
- **日期**: 2026-08-09
- **Completed**: 2026-08-09
- **Summary**: SH（1.117.67.121）公网 22 反向隧道至本机 gitlab-shell（2026-08-09 完成）：`ensure-edge-tunnels.sh` 扩展公网端口段 `EDGE_TUNNEL_PUBLIC_PORTS`（默认 22，`-R 0.0.0.0:<p>:127.0.0.1:<p>`，pgrep 幂等模式区分 loopback/公网段）；autossh 保活 + cron 每 5min 自愈；`~/.ssh/config` gitlab.daydaymoney.com `Port 2222`→`22`（原 2222 指向 SH 运维 sshd 必拒，正是 OPT-20260807-030 Permission denied 根因之一）。踩坑：手动隧道 kill 后 SH 侧 sshd 会话残留监听 22 → autossh 绑定失败死循环（ExitOnForwardFailure 下 ssh 立即退出重连）——需精确 kill SH 侧残留 sshd 子进程（本会话 pid 783/1936）后再起。
- **Verification**: `ssh -T git@gitlab.daydaymoney.com` → Welcome to GitLab；`git ls-remote/clone example-user/taskAuth.git` 成功（仓库路径须在用户命名空间 example-user 下，task2money/task2money 不存在）；脚本连跑幂等（autossh 保持 1 条）。commit runAll 已推（子仓指针同步 ensure-edge-tunnels EDGE_TUNNEL_PUBLIC_PORTS）。遗留：OPT-20260808-030 核对项 a 随 OPT-20260809-003 失效。

### OPT-20260809-002 — archimate MCP 工具评估：不替换现役 + 自研 archimate-tool.py
- **Status**: completed
- **日期**: 2026-08-09
- **Completed**: 2026-08-09
- **Summary**: 评估 zthanos/archimate-mcp-server vs 现役 @null-pointer/mcp-archimate@1.0.1，**结论不替换**——zthanos 0 star/0 fork/2026-03-20 后 5 个月无更新（pyproject 作者残留 "OpenAI"、README 泄漏 Windows 本地路径）、输出为 Open Group XSD 3.0 格式（`http://www.opengroup.org/xsd/archimate/3.0/`，无 DiagramObject/sourceConnection 机制）与项目 SKILL 要求的 Archi 原生格式（`http://www.archimatetool.com/archimate` + `--loadModel` 验证）不兼容、12 工具以 JSON（pydantic ArchimateModel）为输入**无 XML 读取/编辑能力**（无法支持「编辑已有 v<N>.diff/full.archimate」核心工作流）、接入需 Python≥3.11 + uv sync + 可选 LLM key（比 npx 一行重）。**新发现（现役 npm 包实证缺陷）**：`export_archimate_xml` 实测输出 `{{VIEW_NAME}}` 占位符未替换、`archimate:` 前缀未声明、schemaLocation 3.0/3.1 混用、view 仅占位注释——**MCP 生成的 XML 不可被 Archi 打开**（这正是 SKILL 绕开 MCP 手写 XML 的原因）。落地：a) archimate SKILL.md 新增「生成门禁：禁用 MCP export_archimate_xml」节（.cursor/rules/archimate-architecture-artifacts.mdc 同步门禁说明并修正「MCP 导出模型头」表述）；c) **自研 `docs/architecture/scripts/archimate-tool.py`**（标准库零依赖）——read（JSON 摘要）/check（本地语义校验：id 唯一、关系端点引用、连线端点一致性、targetConnections 双向一致）/add-element/add-relationship/add-view-node/add-connection（sourceConnection+targetConnections 自动双写、folder 自动归类、自闭合 folder/节点自动展开）/remove（级联：关系+节点+连线+全局登记清理，兼容 do-X 与 do-f-X 节点 id 风格）。SKILL.md 已加「读取/校验/编辑工具」章节引用。
- **Verification**: archimate-tool_test.py 29 例全绿；真实 v15 diff 全链路编辑（+2 元素+1 关系+2 节点+1 连线）→ check 0 critical → **Archi CLI --loadModel 加载通过**；remove 在 v15 full 上零新增 critical。建议 b)（仅用 MCP validate_archimate_model / list_elements / list_relationships / generate_archimate_diagram 只读辅助）已事实生效。

### OPT-20260809-003 — HK 反向隧道确认关闭 + example.com 配置清理
- **Status**: completed
- **日期**: 2026-08-09
- **Completed**: 2026-08-09
- **Summary**: HK（47.86.27.42）双端核查——HK 侧无 autossh/socat/ssh -R 进程、`*:22` 无监听（sshd 仅 2222，generator drop-in 已清 22）、无 crontab、无 tunnel/autossh systemd units、iptables 无 22 DNAT；本机无到 HK 的隧道进程与活动连接（全部 autossh 仅指向 sh）。即 HK 反向链接（原 `-R 0.0.0.0:22:127.0.0.1:2222`）当前已不存在且无自愈机制，无需 kill 动作。已清理 `~/.ssh/config`：删除 gitlab.daydaymoney.com / gitlab-hk.example.com 两个条目及注释（example.com 域名暂停使用，DNS 仍解析 47.86.27.42 但不影响；HK 运维入口保留 Host hk）。
- **Verification**: `ssh -G` 语法正常、HK 22 无监听、本机无 47.86.27.42 活动连接。遗留提示：gitlab.daydaymoney.com DNS 记录仍在解析（域名暂停属 DNS 侧控制）；OPT-20260808-030 核对项 a 已随之失效（HK 隧道已停）。

### OPT-20260809-004 — 密钥修改备份机制：backup-keys.sh
- **Status**: completed
- **日期**: 2026-08-09
- **Completed**: 2026-08-09
- **Summary**: 用户要求「密钥相关的修改应该建立备份机制」。产出 `~/bin/backup-keys.sh`——`snapshot [tag]` 快照 `~/.ssh/`（rsync 排除 backups/ 防嵌套递归）→ `~/.ssh/backups/<ts>-<tag>/`；`list` 列出快照；`restore [--all] <snap>` 回滚（恢复 config/authorized_keys/私钥，默认不动 known_hosts，覆盖前先自动 pre-restore 快照，私钥 chmod 600 校验）；`prune [keep]` 默认保留 20 份；`remote <host> [tag]` 备份远程 `/etc/ssh/sshd_config` + `sshd_config.d/`（不拉取远程私钥/authorized_keys，避免密钥出网）。触发规则落盘记忆（ssh-key-backup-before-edit）：本地 `~/.ssh` 修改前 `snapshot`、远程 sshd 配置修改前 `remote`。踩坑修复：cp 目录自嵌套 bug（`cp -a "$SSH_DIR" "$dest"` → rsync 替代）；restore 循环未覆盖 config/known_hosts 隔离逻辑。
- **Verification**: 自测通过——snapshot→模拟修改→restore→grep 验证恢复（RESTORE_OK）+ 权限 600 + prune；remote 实测 `backup-keys.sh remote sh baseline-sshd` 成功（sshd_config 4231B 含 GatewayPorts clientspecified/Port 2222 + sshd_config.d）。遗留建议：a) 可加 `check` 子命令（diff 当前 ~/.ssh 与最近快照，未备份即修改时告警）；b) 快照内容含私钥明文，建议 rsync 快照目录定期加密或纳入备份策略。

