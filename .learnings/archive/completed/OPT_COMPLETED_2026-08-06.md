# Completed OPT Archive — 2026-08-06

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 8 条（含 2026-08-07 追加的产品决策归档 036/034）。
> 归档执行时间：2026-08-07T02:29:33+08:00

## [OPT-20260806-054] completed
- **Status**: completed
- **Completed**: 2026-08-06
- **Summary**: OPT-036 单账号语义落地：① login-finalize 契约测试固化 popup 登录不建槽（仅写 apiConfig+credentials）；② multi-account.js 新增 pruneSavedAccounts（批量删槽+活跃回退+全删清标记，7 例单测）；③ SW checkAllAccountsForExpiry 广播 accountExpired 后自动清理 401/403 失效槽位并广播 authStateChanged 刷新网页端 Navbar。node --test 224/224 全绿。
- **日期**: 2026-08-06
- **描述**: 产品决策（OPT-20260806-036，决策归档见本文件）确认 Popup 为单账号入口后的插件侧落地：① 固化「Popup 登录（loginWithAccessToken）不创建账号槽位」语义——登录收尾（persistLoginCredentials）仅写 apiConfig + credentials，契约测试防回归；② 插件侧槽位清理兜底——lib/multi-account.js 新增 pruneSavedAccounts（批量删除槽位、活跃槽被删时自动回退剩余账号、全删清活跃标记），SW checkAllAccountsForExpiry 在广播 accountExpired 后调用清理 401/403 失效槽位并广播 authStateChanged 触发网页端 Navbar 刷新。
- **验收**: node --test 全绿（multi-account 新增 7 例 prune 用例 + login-finalize 新增单账号语义契约用例）；SW 语法检查通过；既有 224 用例无回归。

## [OPT-20260806-055] completed
- **Status**: completed
- **Completed**: 2026-08-06
- **Summary**: OPT-034 决策落地：不提供「公司名=微信昵称」回填脚本——公司名为租户对外展示名，自动重命名有误伤风险；替代路径已验证存在（TenantCompanySettings.vue 公司名称字段，管理员可随时修改）；个人昵称自愈不受影响。无代码变更。
- **日期**: 2026-08-06
- **描述**: 产品决策（OPT-20260806-034（部分），决策归档见本文件）确认不提供一次性回填脚本：公司名是租户对外展示名称，等于微信昵称可能是用户有意设置的业务命名，自动重命名存在误伤风险；替代路径已确认存在——公司设置页（TenantCompanySettings.vue「公司名称」字段）可由公司管理员随时手动修改。个人昵称自愈（ensureWechatProfileNickname）不受影响。无代码变更。
- **验收**: 决策已记录于 PRODUCT_DECISIONS.md（Status: decided + 决策结论）；替代路径（公司设置页手动重命名）验证存在。


### OPT-20260806-057 — conf/core/django 目录退役：配置数据迁至语义化键（4 切片）
- **Status**: completed
- **日期**: 2026-08-06
- **Completed**: 2026-08-06
- **Summary**: Django 目录退役 4 切片全完成：① Root 锚点 6 处（confload/conf_lib/taskSSE/taskEvents/taskAIEndPoint/check_conf_sync）从 conf/core/django/config.yaml 切到 conf/base.yaml；② 真源化 conf/core/{sms,email,sso}/config.yaml（含 email2 保留），taskAuth/domain-events sync manifest 改指新源（片段更名 email.yaml/sms.yaml），conf_loader ssoJwtSecret/ssh_login_allowed_addresses 改读 core/sso，fanyi_agent 迁至 taskProjectService config.yaml；③ 消费者解除：taskBill paypal 删 django fallback（billing/paypal 唯一 SSOT）、BillingRecharge 测试路径改 core/sms/config.test.yaml、删 3 个死 django.yaml sync 片段；④ 目录退役：conf/core/django 删除、CI 脚本（check_git_oauth_provider_keys/consistency）改对比 auth vs task-credential 副本、migrate 脚本标记 DEPRECATED、测试 fixture 更新。
- **Verification**: 全局 grep core/django 残留 0（仅 migrate 脚本历史引用 + 注释）；20 服务健康；conf-read snapshot ssoJwtSecret/ssh_login 正常；DjangoMigration.config-reads 3/3；confload/runAll config 测试通过；taskAuth SMS/taskEvents email/taskBill paypal 配置加载与迁移前一致。

### OPT-20260806-052 — 容器镜像管理 API 增加系统管理员角色校验（当前任意登录用户可读写）
- **Status**: completed
- **日期**: 2026-08-06
- **Completed**: 2026-08-06
- **Summary**: taskCloudService handleSystemAdminContainerImagesRoutes 增加 authz.IsPlatformStaff 校验（网关 forward-auth 注入 X-User-Roles，super_admin/employee 放行），非平台角色返回 403「需要系统管理员权限」。新增单测 TestSystemAdminContainerImagesForbiddenWithoutStaffRole；既有 CRUD/Validation 测试补 X-User-Roles 头；taskFE 回归测试适配（普通用户 403 + 系统管理员 CRUD）。
- **Verification**: 单测 3/3；浏览器回归 4/4（普通用户 403、带 super_admin 角色 CRUD 201→列表→204）；直连服务无角色 403/有角色 200。

### OPT-20260806-056 — taskFE 网页端补充槽位清理桥接 + accountExpired 事件消费（OPT-036 后续）
- **Status**: completed
- **日期**: 2026-08-06
- **Completed**: 2026-08-06
- **Summary**: ① taskChromePlugin SW 新增 pruneSavedAccounts 消息分支（复用 MultiAccount.pruneSavedAccounts，网页端可主动清理过期/指定槽位）；② taskFE plugin_account_bridge.js 新增 pruneSavedAccounts 桥接（userId 字符串化、空列表短路）与 onAccountExpired 监听（与 onAccountStateChanged 同模式，消费 page-bridge 转发的 accountExpired postMessage），window 调试暴露更新；③ Navbar.logic.vue onMounted 消费 accountExpired：清理过期槽位 + 刷新账号列表（console 记录清理数）。
- **Verification**: plugin_account_bridge.prune.test.js 4/4（消息 payload 字符串化/空短路/事件回调/忽略其他 action）；Navbar.logic.platformStaff 3/3（mock 补齐）；taskChromePlugin 全量 224/224；e2e/popup-token-login 2/2。

## [OPT-20260806-065] completed
- **Status**: completed
- **Completed**: 2026-08-06
- **Summary**: 决策确定：厂商门户需绑定邮箱账号后方可申请/使用。实现：taskAuth bridge 拒绝无邮箱用户（重定向 /profile/?sso_error=email_required，7cfbc64 后新提交）；taskAiProvider ExchangeBridge 纵深防御拒绝 @sso.invalid 合成邮箱 + ssoOnlyBody 文案同步；taskFE UserProfile 引导提示。
- **问题**: 修复后微信扫码用户点击「厂商门户 SSO」即自动创建 ai_provider_vendor 档案（is_active=1、无人工审批），与 ssoOnlyBody 提示文案「新厂商档案由运营维护」存在语义张力（邮箱用户此前已是自动建号，微信用户只是行为对齐）
- **建议**: 产品确认：a) 维持自动建号（推荐，与邮箱用户行为一致）；b) 新档案改为 is_active=0 + 运营审批流（需 provider 增加审批接口与状态流转）。若取 b 需同步调整 UpsertVendorFromBridge 与 ExchangeBridge 的 IsActive 检查
- **决策**（2026-08-07 确认，自 PRODUCT_DECISIONS.md 归档）: **维持现状——申请+审核流，不恢复自动建号**。实际落地形态为选项 b 的完整版（已随 OPT-065/066 审核流实现并验证）：① 合成邮箱（`@sso.invalid`）bridge 在 provider 侧拒绝兑换并提示绑定邮箱（store_auth.go ExchangeBridge 纵深防御，TestSSOExchangeRejectsSyntheticEmailVendorBridge 固化）；② UpsertVendorFromBridge **不再自动建号**，无档案时报「未找到厂商档案，请先在镜像市场申请成为厂商」（store_auth.go:86）；③ 建档唯一路径 = 镜像市场提交厂商申请（ApplyVendorApplication，is_active=0 待审核）→ 运营 AdminPortal 审核通过（is_active=1）/驳回（review_note 回填）→ 用户侧按钮随 vendor-status 四态流转（none/pending/rejected/qualified）。该决策项无新增落地代码（流程已实现），关闭。

## [OPT-20260806-036] completed — Popup 多账号语义：单账号入口 vs 槽位入口（产品决策归档）
- **Status**: completed
- **Completed**: 2026-08-06
- **日期**: 2026-08-06
- **描述**: Popup 多账号 UI 移除后，历史账号槽位无插件侧管理入口。共享基础设施（lib/multi-account.js + SW 桥接 + checkAllAccountsForExpiry）保留供 taskFE 网页端消费。遗留观察：(a) SW loginWithAccessToken 仍会 upsert 账号槽位，历史登录积累的旧账号在插件侧不可见、不可管理（过期广播 accountExpired 仍会对所有槽位触发）；(b) 若产品定位 Popup 为单账号入口，可评估 loginWithAccessToken 不再 upsert 槽位（改由 taskFE 网页端负责槽位维护），或提供插件侧槽位清理兜底。
- **决策**（2026-08-06 确认）: **Popup 定位单账号入口**。① Popup 登录（loginWithAccessToken）只维护单一活跃凭据，不创建账号槽位——契约已由 test/login-finalize.test.js 固化（防回归）；② 多账号槽位由 taskFE 网页端（Navbar 账号切换器）作为唯一管理方经 SW 桥接（setActiveAccount 等）维护；③ 插件侧提供槽位清理兜底：SW checkAllAccountsForExpiry 在广播 accountExpired 后调用 MultiAccount.pruneSavedAccounts 自动清理 401/403 失效槽位（活跃槽被清理时自动回退剩余账号），并广播 authStateChanged 触发网页端 Navbar 刷新。落地项见 OPT-20260806-054。
- **Summary**: 产品决策闭环（自 PRODUCT_DECISIONS.md 归档）：落地项 OPT-054（插件侧单账号语义契约 + pruneSavedAccounts + accountExpired 自动清理，node --test 224/224）与 OPT-056（taskFE 网页端槽位清理桥接 + Navbar accountExpired 消费，plugin_account_bridge 4/4）均已 completed 归档，决策全部落地。

## [OPT-20260806-034（部分）] completed — 存量用户「公司名=微信昵称」历史数据回填策略（产品决策归档）
- **Status**: completed
- **Completed**: 2026-08-06
- **日期**: 2026-08-06
- **描述**: 微信昵称修复部署后，新扫码用户个人昵称=微信昵称、公司名称输入框不再预填昵称；存量用户二次登录后个人昵称自动补齐（ensureWechatProfileNickname 空则写，无需脚本）。但**公司名=微信昵称**的历史数据不会被自动改（需用户在公司设置页手动重命名，或评估是否提供一次性回填脚本将「公司名==该用户微信昵称」的公司重命名为中性名）。
- **决策**（2026-08-06 确认）: **不提供一次性回填脚本**。理由：① 公司名是租户对外展示名称，等于微信昵称可能是用户有意设置的业务命名，自动重命名为中性名存在误伤风险，且变更影响对外链接/分享展示；② 已有低成本用户可控路径：公司设置页（TenantCompanySettings.vue「公司名称」字段）可由公司管理员随时手动修改；③ 个人昵称自愈（ensureWechatProfileNickname）与公司名重命名解耦，互不影响。该决策项无落地代码，关闭。
- **Summary**: 产品决策闭环（自 PRODUCT_DECISIONS.md 归档）：决策为不提供回填脚本，无落地代码；落地项 OPT-055（决策落地记录，无代码变更）已 completed 归档。

