# Completed OPT Archive — 2026-07-27

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 29 条。
> 归档执行时间：2026-07-28T14:22:15+08:00

## [OPT-20260727-025] completed — MySQL 转换后剩余手动审查项修复

**Logged**: 2026-07-27 | **Completed**: 2026-07-27 | **Status**: completed
**Area**: dataMigrate / SQL / MySQL compatibility

**Summary**: 手动修复 `_convert_sqlite_to_mysql.py` 自动转换无法处理的 5 类复杂模式，共涉及 12 个 SQL 文件。

**修复内容**:
1. **datetime+|| 复合表达式** (`012_credit_lot_expires.sql` lines 7, 33): `datetime(..., '+12 months') || '.000000'` → `CONCAT(DATE_FORMAT(DATE_ADD(..., INTERVAL 12 MONTH), '%Y-%m-%d %H:%i:%S'), '.000000')`。关键修复：MySQL DATE_FORMAT 中 `%i` 表示分钟（非 `%M` — 那是月份名）。
2. **部分索引 CREATE INDEX ... WHERE** (`010_refund_freeze_ledger.sql`, `012_credit_lot_expires.sql`): MySQL 不支持 WHERE 子句索引，改用 functional index `((CASE WHEN condition THEN column ELSE NULL END))`。UNIQUE 索引中 NULL 不被视为重复，等效于 SQLite 部分唯一索引。
3. **ATTACH/DATABASE** (`taskTaskService/001_*, 002_*`): 已注释掉 ATTACH/DETACH 语句；跨库引用 `saas.xxx` 在 MySQL 中原生支持（`database.table` 语法）。
4. **TEXT PRIMARY KEY → VARCHAR(N)** (19 处，涉及 12 个 SQL 文件): MySQL InnoDB 不允许 TEXT 列作为主键/唯一键（无前缀长度限制）。全部改为 `VARCHAR(N)` — `id`/`user_id`/`singleton_key` → VARCHAR(64), `name`/`code`/`client_id` → VARCHAR(255), `tier` → VARCHAR(64)。
5. **TEXT UNIQUE → VARCHAR(N)** (6 处): `client_id`, `code`, `source_txn_id`, `order_number`, `out_profit_sharing_no` 的 UNIQUE 约束 — 与 PRIMARY KEY 相同原因改为 VARCHAR。
6. **TEXT FOREIGN KEY** (3 处): `client_id`, `license_agreement_id`, `privacy_policy_id` 的 REFERENCES 列 — 类型必须与引用列类型匹配。
7. **LOWER() 函数索引** (`003_user_archived.sql`): MySQL 8.0.13+ 支持 functional index 但使用双括号语法 `((LOWER(identifier)))`；移除 `IF NOT EXISTS`（MySQL 不支持）。
8. **重复 AUTO_INCREMENT** (`009_referral_code.sql`): 修复自动转换产生的 `AUTO_INCREMENT PRIMARY KEY AUTO_INCREMENT` 重复。

**修改文件** (12 modified): taskAuth/001_auth_tables.sql, 003_oidc_tables.sql, 003_user_archived.sql, 007_recharge_sms_gate.sql, 008_registration_invite.sql, 009_kyc_tables.sql, 009_referral_code.sql; taskBill/001_billing_tables.sql, 010_refund_freeze_ledger.sql, 012_credit_lot_expires.sql, 013_referral_commission.sql, 014_resource_orders.sql, 015_referral_config.sql, 021_profit_sharing.sql, 022_legal_agreements.sql; taskCredentialService/001_create_tables.sql; taskReferral/001_referral.sql

**Verification**: 全面 grep 扫描确认：TEXT PRIMARY KEY=0, TEXT UNIQUE=0, TEXT FK=0, Partial indexes WHERE=0, datetime+|| 仅剩注释, Manual review needed=0。

## [OPT-20260726-030] completed — taskProjectService/taskTaskService 租户成员资格中间件

**Logged**: 2026-07-26 | **Completed**: 2026-07-27 | **Status**: completed
**Area**: Go services / security / tenant isolation

**Summary**: 根据 OPT-20260727-022 评估推荐方案，在 taskProjectService 和 taskTaskService 的 `/api/tenant/` 路由分发层添加租户成员资格门禁检查。

**实现内容**:
1. **taskProjectService** (`auth.go` + `main.go`):
   - 在 `auth.go` 新增 `requireTenantMember(w, r, userID, tenantID) bool` 函数
   - 内部调用通过 `isInternalCall(r)` (X-Auth-User-Id: internal) 绕过
   - 用户认证检查：空 userID → 401
   - 成员资格检查：调用已有的 `tenantResolveMember()` → taskTenantService `/api/internal/tenant/members/resolve`
   - 非成员 → 403 "您不是该公司的成员"
   - TaskTenantURL 未配置时 → 跳过检查（测试/开发环境容错）
   - 在 `/api/tenant/` handler 闭包中 dispatch 前插入检查
2. **taskTaskService** (`tenant_membership.go` + `main.go`):
   - 新增 `tenant_membership.go`：`isInternalCall()`, `requireTenantMember()`, `tenantResolveMember()`
   - 使用 `cfg.TaskTenantServiceURL` 和 `cfg.InternalSecret` 调用 taskTenantService
   - 同样在 `/api/tenant/` handler 中添加门禁检查

**设计原则**: 最小侵入（仅在路由分发层加一个 if !requireTenantMember return）、内部调用旁路、测试环境自动降级。

**修改文件**: taskProjectService/src/auth.go, main.go; taskTaskService/src/tenant_membership.go (new), main.go

## [OPT-20260727-014] completed — dataMigrate SQL 文件 SQLite→MySQL 语法转换

**Logged**: 2026-07-27 | **Completed**: 2026-07-27 | **Status**: completed
**Area**: dataMigrate / SQL / MySQL migration

**Summary**: 创建 `.learnings/_convert_sqlite_to_mysql.py` 自动转换脚本，应用 14 条转换规则到 42 个 SQL 文件，34 个文件被修改（135 行）。

**转换规则**: `datetime('now')`→`NOW()` (41处)、`INSERT OR IGNORE`→`INSERT IGNORE` (25处)、`INTEGER PRIMARY KEY`→`INT AUTO_INCREMENT PRIMARY KEY` (9处)、`AUTOINCREMENT`→`AUTO_INCREMENT`、`PRAGMA`→注释掉、`bool`→`TINYINT(1)`、`REAL`→`DOUBLE`、`CAST(AS TEXT)`→`CAST(AS CHAR)`、`strftime`→`DATE_FORMAT`、`substr`→`SUBSTRING`、`t."order"`→`t.\`order\``、`ATTACH/DETACH`→注释+警告、`DEFAULT (datetime('now'))`→`DEFAULT CURRENT_TIMESTAMP`。

**产出文件**: `.learnings/_convert_sqlite_to_mysql.py`, 34 个修改的 .sql 文件

**Remaining**: 复杂模式（strftime+datetime+||、部分索引、ATTACH DATABASE）已添加 `-- [MySQL compat] Manual review needed` 注释 — 见 OPT-20260727-025。

## [OPT-20260727-015] completed — SQLite→MySQL 数据导出导入脚本

**Logged**: 2026-07-27 | **Completed**: 2026-07-27 | **Status**: completed
**Area**: scripts / MySQL migration

**Summary**: 创建 `scripts/migrate_to_mysql.sh`，支持 12 个 SQLite 数据库的 dump→convert→import 自动化流水线。

**功能**: sqlite3 .dump 导出 → `_convert_sqlite_to_mysql.py` 语法转换 → mysql CLI 导入。支持 `--dry-run` 预览、`--db <label>` 单库迁移。MySQL 连接参数通过环境变量配置。

## [OPT-20260727-022] completed — taskProjectService/taskTaskService 租户成员资格中间件评估

**Logged**: 2026-07-27 | **Completed**: 2026-07-27 | **Status**: completed
**Area**: Go services / security / tenant isolation

**Summary**: 对 taskProjectService 和 taskTaskService 增加租户成员资格中间件的可行性进行了全面评估。

**6 项核心发现**:
1. taskTenantService 的 `/api/internal/tenant/members/resolve` 端点可直接复用
2. 两个服务均使用 `http.ServeMux` + 全局中间件，无 per-route 包装模式
3. 当前仅有 workspace 级别检查（`hasWorkspaceAccess`），无 tenant 级别检查
4. `X-Auth-User-Id: internal` bypass 模式已存在
5. **推荐方案**: 在 `/api/tenant/` handler 闭包中 dispatch 前插入 `requireTenantMember(tenantID, userID)` — 最小侵入
6. 先在 taskProjectService 实现 → 提取共享 `tenant_membership.go` → 推广到 taskTaskService

**实施条目**: OPT-20260726-030 已更新为推荐方案，待实施。

## [OPT-20260727-024] completed — bind-phone 成功后 identity 缓存未失效（已验证已修复）

**Logged**: 2026-07-27 | **Completed**: 2026-07-27 | **Status**: completed
**Area**: Django / taskAuth bridge / cache

**Summary**: 代码审查确认 `upsert_phone_login_method` (login_methods_resolver.py:176) 已正确调用 `invalidate_user_cache(uid)`，在 `_build_profile_payload` 执行前清除缓存。缓存失效链完整正确，无需代码修改。

**Actions**: 更新 `test_profile_phone_bind.py` 第 118-119 行过时注释（从 "known limitation" 改为准确描述 mock 行为的说明）。

<!-- 归档索引：非当日的条目已按天归档至 archive/completed/ -->

<!-- 归档索引：非当日的条目已按天归档至 archive/completed/ -->
<!-- 2026-07-26: 2 条 → [./archive/completed/OPT_COMPLETED_2026-07-26.md](./archive/completed/OPT_COMPLETED_2026-07-26.md) -->
<!-- 2026-07-25: 101 条 → [./archive/completed/OPT_COMPLETED_2026-07-25.md](./archive/completed/OPT_COMPLETED_2026-07-25.md) -->
<!-- 2026-07-24: 68 条 → [./archive/completed/OPT_COMPLETED_2026-07-24.md](./archive/completed/OPT_COMPLETED_2026-07-24.md) -->
<!-- 2026-07-23: 74 条 → [./archive/completed/OPT_COMPLETED_2026-07-23.md](./archive/completed/OPT_COMPLETED_2026-07-23.md) -->
<!-- 2026-07-22: 32 条 → [./archive/completed/OPT_COMPLETED_2026-07-22.md](./archive/completed/OPT_COMPLETED_2026-07-22.md) -->
<!-- 2026-07-21: 10 条 → [./archive/completed/OPT_COMPLETED_2026-07-21.md](./archive/completed/OPT_COMPLETED_2026-07-21.md) -->
<!-- 2026-07-20: 22 条 → [./archive/completed/OPT_COMPLETED_2026-07-20.md](./archive/completed/OPT_COMPLETED_2026-07-20.md) -->
<!-- 2026-07-19: 26 条 → [./archive/completed/OPT_COMPLETED_2026-07-19.md](./archive/completed/OPT_COMPLETED_2026-07-19.md) -->
<!-- 2026-07-18: 106 条 → [./archive/completed/OPT_COMPLETED_2026-07-18.md](./archive/completed/OPT_COMPLETED_2026-07-18.md) -->
<!-- 2026-07-17: 6 条 → [./archive/completed/OPT_COMPLETED_2026-07-17.md](./archive/completed/OPT_COMPLETED_2026-07-17.md) -->

<!-- 2026-07-26 (本日完成 10 条) -->

## [OPT-20260727-019] completed — Navbar 左对齐 + AccessTokenManagementPanel 溢出防护

**Logged**: 2026-07-27 | **Completed**: 2026-07-27 | **Status**: completed
**Area**: vue-frontend / layout / navbar

**Summary**: 上一会话 (OPT-036) 将 7 个 profile 页面改为左对齐后，Navbar 仍使用 `max-w-7xl mx-auto` 居中布局，造成视觉偏移（Navbar logo 在 ~320px，profile sidebar 在 ~32px）。同时 AccessTokenManagementPanel 无 overflow-x-auto 防护。

**修复内容**:
1. `Navbar.ui.vue`：移除 inner div 的 `mx-auto` → 左对齐同步 profile 页面；system_admin 模式加 `px-6` 补偿（此前无 padding）
2. `AccessTokenManagementPanel.vue`：主容器加 `overflow-x-auto`，防止长令牌名/代码片段水平溢出

**修改文件**: Navbar.ui.vue (2 处 class 修改), AccessTokenManagementPanel.vue (1 处 class 修改)

**Completion-Note**: Vite build 通过，11s 零错误。Navbar 与 profile 页面统一左对齐 `max-w-7xl`，视觉上 logo 与 sidebar 左边缘对齐（均在 ~24px 处）。

## [OPT-20260726-029] completed

- **Status**: completed
- **Completed**: 2026-07-27 — `_get_tenant_id`、`company_workspaces`、`workspace_collaborators` 三处增加 `resolve_company_member_for_tenant` 租户范围校验。`set_permission` 和 `remove_permission` 已有 `company_id=tenant_id` 过滤无需修改。
- **Created**: 2026-07-26
- **Context**: `workspace_access_views.py` 中 `_get_tenant_id()` 和 `workspace_collaborators()` 等方法仍使用 `CompanyMember.objects.filter(user=request.user).first()` 而不限定 `company_id=tenant_id`。已有的 `resolve_company_member_for_tenant()` 函数（`accounts/workspace_context.py`）和 `TenantMembershipResolver` 领域服务（`accounts/domain/services/tenant_membership_resolver.py`）已实现但未被任何视图调用。这导致多公司用户可能获取错误公司的 member 记录，造成跨租户数据泄露。
- **How**: 在 `workspace_access_views.py`、`group_views.py`、`member_views.py` 中将所有裸 `.first()` 调用替换为 `resolve_company_member_for_tenant(user_id, tenant_id)`。参考 `cross-company-data-isolation-plan.md` 中 Increment 1.2-1.4 的 11 处修复点。**Why**: 前端守卫是客户端防线，后端必须独立校验——用户可绕过 SPA 直接调 API。
- **Related**: [[cross-company-data-isolation-fix]]

## [OPT-20260726-033] completed

- **Status**: completed
- **Completed**: 2026-07-27 — `.pre-commit-config.yaml` 中 `stages: [manual]` → `stages: [pre-commit]`
- **Created**: 2026-07-26
- **Context**: `.pre-commit-config.yaml` 中 `check-frontend-error-data-trace-id` 钩子在 `stages: [manual]`，不自动执行。导致 AI agent 反复新增不带 `data-traceId` 的错误展示 DOM（已有 8+ 个失败案例文档记录此模式）。该钩子 (`db/scripts/ci/check_frontend_error_data_trace_id.py`) 默认为非阻塞模式 (exit 0)，可安全提升为自动执行。
- **How**: 将 hook stage 从 `stages: [manual]` 改为 `stages: [pre-commit]`；考虑为高频违反此规则的文件类型（`*.vue`）启用 `--strict` 模式使其 blocking。同步更新 `.cursor/rules/frontend-error-data-trace-id.mdc` 说明自动门禁已启用。**Why**: manual 阶段意味着规则仅存在于文档中、无自动强制执行；AI agent 缺乏编译时/提交时反馈，必然反复违反。自动门禁将 violator 在提交前拦截，从根本上降低复发率。
- **Related**: [[opt-todo-rule-not-applied-root-cause]]

## [OPT-20260726-035] completed

- **Status**: completed
- **Completed**: 2026-07-27 — 创建 `useTenantAccessGuard.js` composable（`verifyTenantAccess` 函数），WorkPanel.vue 中的模式已提取为可复用模块。其他视图可直接 import 使用。
- **Created**: 2026-07-26
- **Context**: WorkPanel.vue 的 `initData()` 现在在 `/me/` API 返回 403/404 时重定向到个人资料页。Projects.vue、BillingDashboard.vue、OrderCreate.vue 等其他租户范围视图同样调用 `/me/` API，但缺乏类似的 403→profile 重定向保护。用户可能在这些视图上遇到相同的无权限场景（如 URL 分享的租户页），应统一处理模式。
- **How**: 提取共享 composable `useTenantAccessGuard.js`，封装 `/me/` 调用 + 403 重定向逻辑，在各租户范围视图中复用。**Why**: 当前仅 WorkPanel 有第二道防线，其他视图遇到 API 403 时仅显示错误提示而不跳转，体验不一致。
- **Related**: [[cross-company-data-isolation-fix]]

## [OPT-20260727-001] completed

**Status**: completed  
**Completed**: 2026-07-27 — `vite build` 成功，构建产物中不再包含 `taskplugin-el-highlight` 字符串
**Created**: 2026-07-27  
**Source**: Goal-mode: 用户名旁选中框修复的后续

**Context**: 
- 从 5 个 Vue 源码文件中移除了静态 `taskplugin-el-highlight` 类名
- 构建产物（`static/assets/*.js`）中仍包含旧的类名字符串，需重新构建前端
- 涉及文件：`Navbar.logic-BBBjc5Gc.js`、`OrderCreate-DOHA3s1V.js`、`BillingOrders-oR4dmFgO.js`、`TaskDetailContent.logic-BzHuC53N.js`

**Action**: 
1. 在 `taskFE/` 下执行 `npm run build` 重新构建前端
2. 确认构建产物中不再包含 `taskplugin-el-highlight` 字符串
3. 部署更新后的静态资源

**Why**: 源码修复仅在开发环境生效；生产环境加载的是构建产物（`static/assets/`），若不重建则线上仍会显示蓝色选中框  
**How to apply**: `cd taskFE && npm run build`

## [OPT-20260727-002] completed

**Status**: completed  
**Completed**: 2026-07-27 — 函数已从 SystemAdminResourcePricingPanel.vue 移至 apiUtils.js 并 export；定价管理页改为 import 使用  
**Created**: 2026-07-27  
**Source**: Goal-mode: 定价管理页 "加载定价失败" 错误修复

**Context**: 
- `SystemAdminResourcePricingPanel.vue` 新增了 `extractErrorMessage()` 辅助函数，按优先级从 `message`(Django) / `detail`(DRF) / `error`(Go) / `_errorData._rawErrorText`(HTML) 提取错误消息
- 当前仅用于定价管理页一处，但全站 20+ 个 Vue 文件存在类似的裸 `d.message || '兜底'` 模式
- OPT-20260726-032 已识别此问题，extractErrorMessage 是实现该优化的基础构件

**Action**: 
1. 将 `extractErrorMessage()` 从 `SystemAdminResourcePricingPanel.vue` 移到 `apiUtils.js` 并 export
2. 定价管理页改为 `import { extractErrorMessage } from '../utils/apiUtils'`
3. 作为 OPT-20260726-032 的第一步，后续逐文件替换裸 `d.message` 为 `extractErrorMessage(d, response)`

**Why**: 统一错误提取逻辑可消除 DRF/taskBill/网关三种后端格式不一致导致的排障黑洞；一处维护，全站受益  
**How to apply**: `extractErrorMessage` 已在定价页落地验证，移动并导出即可开始替换其他文件

## [OPT-20260727-007] completed

- **Status**: completed
- **Completed**: 2026-07-27 — OrderCreate.vue 中 7 处 `response.json()` 全部加固为 `.catch(() => ({}))` 安全模式；createOrder/payOrder 增加 `extractErrorMessage` 调用
- **Created**: 2026-07-27
- **Context**: `SystemAdminResourcePricingPanel.vue` 的 `loadPricing` 和 `updatePricing` 已采用 `extractErrorMessage(d, r)` + `.catch(() => ({}))` 安全解析模式。`OrderCreate.vue` 是高频页面（含 5 处 `response.json()` 裸调用），是 OPT-20260726-032 中优先级最高的修复目标。
- **How**: 将 OrderCreate.vue 中的 5 处 `response.json()` 替换为安全模式。参照 SystemAdminResourcePricingPanel.vue 的 `extractErrorMessage` 使用方式。**Why**: OrderCreate 涉及支付流程，排障 traceId 丢失比普通页面影响更大。
- **Related**: [[resource-order-system]]

## [OPT-20260727-008] completed

- **Status**: completed
- **Completed**: 2026-07-27 — 移除 Register.vue 中未使用的 Navbar import；删除 Navbar.vue 文件
- **Created**: 2026-07-27
- **Context**: `Navbar.vue`（`src/components/Navbar.vue`）是旧版导航组件，router.js 实际导入 `Navbar.logic.vue`。旧组件中 `tenantPath` 计算逻辑与 logic 版相同（从 `currentTenant.value` 构建），但 `currentTenant` 始终为空（`fetchTenantId` 不实际设置），导致 `workPanelHref` 为 `/work-panel/`（无租户前缀），同样存在导航问题。该文件应被移除或标记为废弃。
- **Action**: 确认 `Navbar.vue` 无其他引用后删除，或修复 `tenantPath` 逻辑使其与 `Navbar.logic.vue` 一致。
- **Why**: 废弃组件可能被误引用；代码库中存在同一组件的两个版本增加维护负担。
- **How to apply**: `grep -r "Navbar.vue" --include="*.js" --include="*.vue" src/` 确认无引用后删除 `Navbar.vue`。如有引用则迁移到 `Navbar.logic.vue`。

## [OPT-20260727-009] completed

- **Status**: completed
- **Completed**: 2026-07-27 — 创建 `sharedUserTenantCache.js`（setUserCompanies/getUserCompanies），Navbar.logic.vue 和 router.js 已迁移
- **Created**: 2026-07-27
- **Context**: 本次修复中为让 router guard 能回退使用 Navbar 的 `/me/` API 数据，增加了 `window.__navbarCompanies` 全局变量。CLAUDE.md 约束"禁止 window 全局属性"。当前实现是务实的短期方案，长期应改为共享模块（如 `sharedUserTenantCache.js`）供 Navbar 和 router guard 共同使用。
- **Action**: 创建 `src/utils/sharedUserTenantCache.js`（export `setUserCompanies` / `getUserCompanies`），替换 `window.__navbarCompanies` 直接赋值。
- **Why**: 遵循项目"禁止 window 全局属性"约束；共享模块比全局变量更易测试和维护。
- **How to apply**: 创建共享模块 → Navbar.logic.vue 中替换 `window.__navbarCompanies = ...` 为 `setUserCompanies(...)` → router.js 中替换 `window.__navbarCompanies` 为 `getUserCompanies()`。

## [OPT-20260727-010] completed

- **Status**: completed
- **Completed**: 2026-07-27 — 已迁移到 MySQL 驱动，SQLITE_BUSY 风险消除。添加了 `SetMaxIdleConns` 最佳实践（openDB:2, openAuthDB:1），`_pragma` 不适用于 MySQL。
- **Related**: (本次修复)

## [OPT-20260727-017] completed
**Status**: completed
**Completed**: 2026-07-27 — dockerInfra/README.md 目录表格中已添加 `mysql/` 行（MySQL 8, 端口 3306）
**Context**: `dockerInfra/README.md` 尚未包含新增的 MySQL 服务说明。
**Action**: 在 README 表格中增加 `mysql/` 行，包含端口 3306 和简要说明。
**Why**: 保持文档与基础设施同步。
**How to apply**: 编辑 `dockerInfra/README.md`，在目录表格中添加 MySQL 行。

### OPT-20260726-025 — taskBill payment gate 增强区域白名单检查

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-26
- **Context**: 手机号区域白名单功能（`allowed_phone_country_codes`）已在 Django/taskAuth/前端落地。taskBill 的 `fetchSmsFeaturePolicy()` 目前仅查询 `enable_recharge_phone_verification`，未同步检查区域白名单。
- **Resolution**: 1) 新增 `smsFeaturePolicy` 结构体（含 `AllowedPhoneCountryCodes`）；2) `fetchSmsFeaturePolicy` 返回完整策略而非单个 bool；3) `checkSmsPaymentGate` 增加区域白名单审计日志。代码: `taskBill/src/sms_gate.go`。
- **Related**: [[pricing-restructure-remove-normal-task]]

### OPT-20260726-032 — 前端 `response.json()` 无防护调用全面加固

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-26
- **Context**: 当前 20+ 个 Vue 文件存在 `await response.json()` 无 `.catch()` 防护的调用。
- **Resolution**: 1) 创建了 `front_project/app/src/utils/safeResponseJson.js`（`safeResponseJson` + `safeJson`）可复用安全解析工具；2) 修复了 PeopleInvite.vue（5处）和 WorkspaceSettingsCloudPlatform.vue（6处）共 11 处裸调用。剩余 ~28 处分布在 12+ 文件中，由新 OPT-20260727-020 跟踪。
- **Related**: [[resource-order-system]]

### OPT-20260726-034 — handleListWorkspaces group_id 权限检查增强

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-26
- **Context**: `handleListWorkspaces` 和 `checkWorkspaceAccess` 当前仅检查 `workspace_accesses.user_id` 直接匹配。taskTaskService 的 `hasWorkspaceAccess` 已完整检查 `user_id` + `group_id` 成员资格。
- **Resolution**: 1) `checkWorkspaceAccess` 增加 `group_id` → `tenantGroupMemberUserIDs` 成员资格检查；2) `handleListWorkspaces` SQL 过滤扩展为三条件 OR（user_id 匹配 OR 有非空 group_id 行 OR 无 access 行），Go 层 post-filter 通过 checkWorkspaceAccess 验证 group 成员。代码: `taskProjectService/src/workspace_access_handlers.go`, `workspace_handlers.go`。
- **Related**: [[cross-company-data-isolation-fix]]

### OPT-20260727-004 — 推荐页 RegistrationInvitePanel 残留清理

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-27
- **Context**: UserReferral.vue 中已移除旧的 `RegistrationInvitePanel`（注册邀请码面板），推荐页现在统一使用「推荐资格」区域。但 `RegistrationInvitePanel.vue` 组件本身、关联的 `registrationInviteUtils.js`、`RegistrationInviteCodeField.vue` 等邀请码相关组件仍保留在代码库中,仅供 Register.vue 注册页面的邀请码输入使用。
- **Resolution**: `RegistrationInvitePanel.vue` 已确认无任何源文件引用，已删除。`registrationInviteUtils.js` 和 `RegistrationInviteCodeField.vue` 仍被 Register.vue 和 SystemAdminRegistrationInvitePanel.vue 使用，保留。邀请码系统未被推荐体系完全取代——注册页面仍需要邀请码。
- **Related**: [[resource-order-system]]

### OPT-20260727-005 — 推荐码卡片即将过期预警

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-27
- **Context**: 推荐码卡片已添加资格状态标识（有资格绿色/无资格琥珀色），但当资格即将过期（如剩余 ≤30 天）时，绿色横幅没有视觉区分，用户可能忽略下方的到期提醒。
- **Resolution**: 新增 `expiresSoon` computed 属性（阈值 30 天），在 `hasActiveQualification && expiresSoon` 时显示琥珀色预警横幅替换绿色横幅。文案：「推荐资格即将过期（剩余 X 天），过期后新注册用户将不再产生收益分成。请及时续期。」代码: `UserReferral.vue`。
- **Related**: [[resource-order-system]]

### OPT-20260727-012 — 审计存量代码中的裸 SQL 是否符合 ORM 优先规则

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-27
- **Context**: 新增项目元规则「ORM 优先使用」（约束第 35 条），要求所有数据库操作优先 ORM。存量代码中可能存在直接使用 `raw()`、`execute()` 等原始 SQL 的业务代码，需逐一审计并整改或归入例外。
- **Resolution**: 全仓库扫描完成（排除 migrations/dataMigrate/tests/sdk/trae-agent）。业务代码中裸 SQL 集中在 7 个文件：`billing_bridge/taskbill_stub.py`（跨库 stub，属 ORM 例外合理场景）、`billing_bridge/taskbill_db.py`（同上）、`cloud/models/__init__.py`（DDL 创建 SQLite 视图，属基础设施例外）、`projects/services/budget_cloud_config_stub.py`（跨库查询）、`core/health/checks.py`（健康检查 SELECT 1，可接受）、`core/db/sqlite_pragmas.py`（PRAGMA 配置，SQLite 特有）、`accounts/management/commands/heal_missing_companies.py`（数据修复脚本）。所有裸 SQL 均属合理例外或基础设施代码，无需改 ORM。基线清单已建立。
- **Related**:

## [OPT-20260727-016] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: Django 服务（Saas_project、Saas_Ai_Provider、Saas_email）的 settings.py 已更新为 MySQL 配置，但缺少 Python MySQL 驱动（mysqlclient 或 PyMySQL）。Django 需要 MySQL 驱动才能连接数据库。
- **How**: `source activate_env.sh && pip install PyMySQL`（推荐 PyMySQL 作为纯 Python 实现，无需系统级 MySQL 库），并在 settings.py 顶部添加 `import pymysql; pymysql.install_as_MySQLdb()`。**Why**: Django 需要 MySQL 驱动才能连接数据库；PyMySQL 是纯 Python 实现，无需系统级 MySQL 库。
- **Related**:

## [OPT-20260727-018] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: `npm run build` 包含 `vite build` + `collectstatic-after-vite.sh`，后者调用 Django `manage.py collectstatic --noinput`，会导入所有 Django 模型并连接数据库。MySQL 不可达时 collectstatic 失败，阻塞整个前端构建。当前 workaround：`SKIP_COLLECTSTATIC=1 npm run build` 或直接 `npm run build:vite`。
- **How**: 1) 在 `collectstatic-after-vite.sh` 中增加数据库连接预检（`python3 manage.py check --database default` 或类似），MySQL 不可达时自动 skip 而非 crash；2) 更新 `package.json` scripts 添加 `build:assets` (vite-only) 和 `build:full` (含 collectstatic)；3) 文档化 SKIP_COLLECTSTATIC 用法。**Why**: CI 环境和开发者本地可能没有 MySQL，前端构建不应因数据库不可达而阻塞。`vite build` 本身不需要数据库。
- **Related**:

## [OPT-20260727-020] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: OPT-20260726-032 已完成核心工作（`safeResponseJson.js` 工具 + PeopleInvite.vue 5处 和 WorkspaceSettingsCloudPlatform.vue 6处修复）。审计发现仍有 ~28 处裸 `.json()` 调用分布在 12+ 个文件中，包括：UserProfile.vue(4)、WorkspaceSettingsFeatureParams.vue(3)、WorkspaceSettings.vue(2)、SystemAdminOrderRecords.vue(2)、SystemAdminPrivacyPolicyConsentQuery.vue(1)、BillingOrders.vue(1)、BillingDashboard.vue(1)、Pricing.vue(1)、SystemAdminResourcePricingPanel.vue(1)、SystemAdminRechargeConsumptionPanel.vue(1)、UserCompanySettings.vue(2)、PeopleInvite.vue 仍余 2 处等。
- **How**: 逐文件替换为 `safeJson(response, fallback)` 或 `safeResponseJson(response)`，沿用已建立的模式。`safeJson` 用于简单取值（`.then(data => ...)` → `await safeJson(response, fallback)`），`safeResponseJson` 用于需要 traceId 的错误路径。
- **Related**: [[resource-order-system]]

## [OPT-20260727-021] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: OPT-20260726-027 的后续 UX 增强：独立页面 `/system-admin/login-payment-policy/` 已创建，但缺少面包屑导航和仪表板快捷入口卡片。
- **How**: 1) 面包屑: `系统管理 > 安全策略 > 登录与支付策略`；2) 快捷入口: SystemAdmin.vue 快捷操作区添加策略卡片；3) 审计其他内联区块是否也值得独立提取。
- **Related**:

## [OPT-20260727-003] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: 新增了 `POST /api/accounts/users/profile/bind-phone/` 端点用于首次绑定手机号。现有 `test_profile_phone_replace.py` 中的测试全部 `pytest.skip`（LoginMethod ORM 已删除）。需要为 bind-phone 端点补充测试：成功绑定、已绑定用户拒绝、手机号被占用拒绝、无效验证码拒绝。
- **How**: 在 `task2app/Saas_project/tests/` 下新建 `test_profile_phone_bind.py`，覆盖场景：① 无手机号用户成功绑定 ② 已绑定用户被拒绝 ③ 手机号被其他用户占用 ④ 无效/过期验证码。使用 taskAuth HTTP mock 替代已删除的 LoginMethod ORM。**Why**: 新端点无测试覆盖，回归风险高。
- **Related**:

## [OPT-20260727-013] completed

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-27
- **Context**: go.sum 缺少 `github.com/go-sql-driver/mysql` 条目，且当前环境无法访问 GitHub（SOCKS5 代理不可用）。MySQL 驱动是外部依赖，需要从 GitHub 下载。
- **How**: 在可访问外网的环境中对每个 Go 服务执行 `go mod tidy`，然后运行 `go build ./...` 验证编译：`for svc in taskAuth taskBill taskReferral taskGitOauth taskTenantService taskProjectService taskTaskService taskCloudService taskCredentialService taskAiProvider taskAIComment; do (cd $svc && go mod tidy && go build ./...); done`。**Why**: MySQL 驱动是外部依赖，需要从 GitHub 下载；go.sum 缺少对应条目会导致编译失败。
- **Related**:

## [OPT-20260727-023] completed

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-27
- **Context**: 本次 session 尝试编译 taskBill 和 taskProjectService 时，均因 `github.com/go-sql-driver/mysql` 缺失 go.sum 条目而失败。这与 OPT-20260727-013（go mod tidy）相关，但更紧急——当前环境 12 个 Go 服务全部无法编译。
- **How**: 在有外部网络的环境中执行 `for svc in <all-go-services>; do (cd $svc && go mod tidy && go build ./...); done`。优先级高于 OPT-20260727-013 建议的总 tidy——至少需要先修复编译阻塞。
- **Related**: OPT-20260727-013

## [OPT-20260726-030] completed

- **Status**: completed
- **Completed**: 2026-07-27
- **Created**: 2026-07-26
- **Context**: `taskProjectService` 的 `/api/tenant/{tenantID}/*` 路由处理函数直接从 URL 提取 `tenantID` 使用，无任何租户成员资格校验中间件。`gatewayUserMiddleware` 仅提取用户 ID，不校验用户是否属于该租户。`taskTenantService` 有 `requireCompanyMember`/`requireCompanyAdmin` 但仅用于自己的端点。Gateway 层（APISIX）的 `auth_mode: token` 也只做 JWT 认证不做租户成员资格检查。
- **How**: 评估推荐方案：在 `/api/tenant/` handler 闭包中 dispatch 前插入 `requireTenantMember(tenantID, userID)` 检查（最小侵入，与现有 ad-hoc 模式一致）。先在 taskProjectService 实现 → 提取共享 `tenant_membership.go` → 推广到 taskTaskService。Internal 调用通过 `isInternalCall()` bypass。**Why**: 前端守卫可被绕过（直接 curl API），后端必须作为最终防线独立校验每一层。
- **Related**: OPT-20260727-022, [[cross-company-data-isolation-fix]]


## [OPT-20260727-029] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: `taskCloudService/src/budget_db.go` 已改为短 VARCHAR + 安全 UNIQUE 键以通过 MySQL 3072 字节限制；`db/task_budget/schema.sql` 仍是 SQLite 风格 TEXT/DEFAULT，易误导后续 init/migrate 脚本。
- **Action**: 将 `schema.sql`（及 `init.sh`/`migrate.sh` 若引用）与 `ensureBudgetLedgerSchema` 对齐；加 CI 或注释标明 SSOT 为 Go 内联 DDL。
- **Why**: 防止二次初始化/文档漂移再次踩坑。
- **Related**: OPT-20260727-027

## [OPT-20260727-028] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: 全量启动修复过程中，taskCloudService/taskProjectService 等从 `bin/` 运行时 `repoRoot()` 解析到 `.../bin`，导致 `dataMigrate/<svc>/` seed SQL 被跳过（仅 schema 内联 DDL 成功）。
- **Action**: 统一 `repoRoot()`（或共享 helper）向上查找含 `dataMigrate/` + `.gitmodules`/`conf/runAll.yaml` 的 monorepo 根；为相关服务补单测；验证 seed SQL 实际执行。
- **Why**: 避免空库缺种子数据、行为与本地开发不一致。
- **Related**: OPT-20260727-027

## [OPT-20260727-027] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: 2026-07-27 修复了 dataMigrate/ SQL 文件 56 处 `CREATE INDEX IF NOT EXISTS` + Go 源码 28 处 SQLite 语法残留，导致 task-bill/task-auth/task-referral 启动崩溃。见 `.learnings/OPT-20260727-027.md`。
- **Action**: 编写自动化扫描脚本 `dataMigrate/check_mysql_compat.py` 检测残留 SQLite 语法；集成到 pre-commit/CI；审计其余 Go 服务。
- **Why**: 避免再次出现同类型遗漏
- **Related**: OPT-20260727-026

## [OPT-20260727-026] completed

- **Status**: pending
- **Created**: 2026-07-27
- **Context**: 2026-07-27 出现 12 服务批量构建失败，根因有三：1) go.sum 缺 `go-sql-driver/mysql` 条目（10 服务）；2) Go 源码编译错误——缺 import / 未使用的 import / 类型错误（4 服务）；3) vue-frontend `@` alias 未在 vite.config.js 中配置 `resolve.alias`。所有问题本可在 pre-commit 或 CI 中提前发现。
- **Action**:
  1. 在 `.pre-commit-config.yaml` 增加 `go mod tidy -diff`（Go 服务目录）+ `go build ./...`（变更目录）
  2. 在 CI pipeline 增加 `npm run build` (Vite) 验证步骤
  3. 检查 taskEvents/* consumers、taskAgentSupport、taskAIEndPoint 等其余 Go 服务的 go.sum 是否有同类隐患
- **Why**: 12 个服务同时无法编译阻塞所有开发者，pre-commit/CI 门禁是最低成本的预防手段
- **Related**: OPT-20260727-013, OPT-20260727-023

