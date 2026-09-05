# Completed OPT Archive — 2026-07-26

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 31 条。
> 归档执行时间：2026-07-27T00:13:03+08:00

## [OPT-20260726-028] completed

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: frontend / security / access-control

**Summary**: 前端路由守卫接入租户成员资格校验。账号 `author@example.com` 可访问不属于它的租户 `850256677331562496` 的 work-panel —— 根因是 `router.js` 的 `beforeEach` 守卫仅检查认证状态（`requiresAuth`），从未校验已登录用户是否属于 URL 中的 `:tenant` 参数。已存在完整校验函数 `resolveUnauthorizedTenantRedirect`（`workPanelTenantAccess.js`）但从未被导入调用，是死代码。

**修复内容**:
1. `router.js` L8: 导入 `resolveUnauthorizedTenantRedirect` from `workPanelTenantAccess.js`
2. `router.js` L873-897: 新增 `fetchUserProfileCached()` — 带缓存的 profile API 调用，获取用户 companies 列表供租户校验
3. `router.js` L938-951: 在 `beforeEach` 认证通过后，对 `to.params.tenant` 的路由调用 `resolveUnauthorizedTenantRedirect`，非成员重定向到 `/user/{id}/profile/`

**Completion-Note**: 4 个已有单元测试全部通过，Vite 生产构建成功。OPT-029 + OPT-030 追踪后续后端防线加固。

<!-- 归档索引：非当日的条目已按天归档至 archive/completed/ -->
<!-- 2026-07-25: 101 条 → [./archive/completed/OPT_COMPLETED_2026-07-25.md](./archive/completed/OPT_COMPLETED_2026-07-25.md) -->
<!-- 2026-07-24: 68 条 → [./archive/completed/OPT_COMPLETED_2026-07-24.md](./archive/completed/OPT_COMPLETED_2026-07-24.md) -->
<!-- 2026-07-23: 74 条 → [./archive/completed/OPT_COMPLETED_2026-07-23.md](./archive/completed/OPT_COMPLETED_2026-07-23.md) -->
<!-- 2026-07-22: 32 条 → [./archive/completed/OPT_COMPLETED_2026-07-22.md](./archive/completed/OPT_COMPLETED_2026-07-22.md) -->
<!-- 2026-07-21: 10 条 → [./archive/completed/OPT_COMPLETED_2026-07-21.md](./archive/completed/OPT_COMPLETED_2026-07-21.md) -->
<!-- 2026-07-20: 22 条 → [./archive/completed/OPT_COMPLETED_2026-07-20.md](./archive/completed/OPT_COMPLETED_2026-07-20.md) -->
<!-- 2026-07-19: 26 条 → [./archive/completed/OPT_COMPLETED_2026-07-19.md](./archive/completed/OPT_COMPLETED_2026-07-19.md) -->
<!-- 2026-07-18: 106 条 → [./archive/completed/OPT_COMPLETED_2026-07-18.md](./archive/completed/OPT_COMPLETED_2026-07-18.md) -->
<!-- 2026-07-17: 6 条 → [./archive/completed/OPT_COMPLETED_2026-07-17.md](./archive/completed/OPT_COMPLETED_2026-07-17.md) -->

<!-- 2026-07-26 (本日完成 9 条) -->

## [OPT-20260726-010] completed

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: data-migration / django / infrastructure

**Summary**: Django 迁移依赖清理 + 防护体系建设。三管齐下完成：
1. **Migration 完整性检查脚本**: 创建 `dataMigrate/check_migration_integrity.py` — 解析 Django `INSTALLED_APPS` + 扫描所有 165 个 migration 文件中的 `dependencies` 引用，检测引用已删除 app 的过期依赖，支持 `--json` 机器可读输出。当前扫描结果：✅ 全部通过。
2. **启动流程修复**: `runall-saas-backend.sh` 改为 `00_manage_init.py run migrate` → `00_manage_init.py run-group project-bootstrap --continue-on-error || true`（幂等 seed: install-deps/create-admin/legal-docs），防止 seed 缺失导致功能异常。
3. **init 脚本统一到 dataMigrate/saas/**: `manage_init.py` 已同步到 `00_manage_init.py`（引用 `dataMigrate/saas/` 路径），`init_project.sh` 同步更新引用。

**Completion-Note**: 三项建议全部落地：(1) `dataMigrate/check_migration_integrity.py` — 可集成到 CI；(2) `runall-saas-backend.sh` 启动时自动执行 migrate + seed；(3) `manage_init.py` 统一指向 `dataMigrate/saas/` 标准路径。

## [OPT-20260726-008] completed
**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: backend / Go / taskBill / profit-sharing / refund
- 分账退款保护：(1) `wechat_profit_sharing.go` 新增 `cancelProfitSharingForOrder` — 按分账状态分级处理：`pending`→直接作废，`finished`→调用微信 `POST /v3/profitsharing/return-orders` 回退，`failed`→直接取消；(2) 新增 `cancelPendingProfitSharingsForTenant` — 退款审批后按 FIFO 取消租户待分账记录；(3) `refund_approve.go` 的 `approveRefundApplication` 在提交后异步调用回退逻辑，失败不阻塞退款流程。

<!-- 2026-07-26 (本日完成 7 条) -->

## [OPT-20260726-007] completed
**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: backend / Go / taskBill / referral / wechat-profit-sharing
- 微信分账推荐佣金基础设施：(1) DB: `dataMigrate/taskBill/021_profit_sharing.sql` — `billing_profit_sharing` 表 + `referral_edge` 扩展 openid 字段；(2) Go: `wechat_profit_sharing.go` — 微信分账 API 客户端（添加/删除接收方、创建分账订单、查询结果、解冻资金）+ 本地 `markOrderForProfitSharing` + `runProfitSharingDaemon` 定时任务 + `handleProfitSharingNotify` 回调；(3) `orders.go`: `markOrderPaid` 完成后异步标记分账；(4) `main.go`: 启动分账 daemon；(5) `handlers.go`: 注册分账回调路由；(6) APISIX: `billing-profitsharing-notify` 路由。

<!-- 2026-07-26 (本日完成 6 条) -->

## [OPT-20260726-006] completed
**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: backend / Go + Django + APISIX / billing / migration
- 移除余额充值路径：(1) taskBill: 删除 `credit.go` + `consent_gate.go`，移除 10 个 recharge handler 函数（`handleRechargeWechatCreatePublic/StatusPublic`, `handleRechargePaypalCreate/Status`, `handlePaymentChannels`, `handleInternalCreditRecharge`, `handleRechargeDelegate`, `handleInternalWechatPrepay/MockComplete/Status`），移除 handlers.go 对应路由；(2) `wechatCreditFromPending` 非订单分支改为返回错误；(3) PayPal webhook 注销 creditRecharge 调用；(4) `referral_commission.go` 改用 `adminGrantResources` 发放任务帖配额；(5) Django: 删除 `recharge_views.py` + `wechat_recharge.py`，简化 `proxy_views.py` 为纯转发，清理 `urls.py` + `internal_views.py` consent 端点；(6) APISIX: 移除 `billing-sms-phone-verify` 路由；(7) KYC/SMS gate 函数重命名 `Recharge→Payment`。

<!-- 2026-07-26 (本日完成 5 条) -->

## [OPT-20260726-005] completed
**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: backend / Go / taskBill / audit + fix
- 计费模型审计 + 修正：(1) 审计确认当前为「资源订单 + 余额充值」双路径模型，非单一充值模型；(2) `handlePayOrder` 新增 KYC 门禁 + SMS 门禁（此前仅有旧 recharge handler 有这些检查）；(3) 回退对 `handleRechargeWechatCreatePublic` / `handleRechargePaypalCreate` 的误 deprecation 标记（双路径均活跃）；(4) 更新 `plans/django-to-go-migration.md` 计费模型说明 + OPT-073 描述。

<!-- 2026-07-26 (本日完成 4 条) -->

## [OPT-20260726-004] completed
**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: backend / Django / cleanup / migration
- 删除 Django `subscriptions/` app（全量：`models/` Plan+Subscription、`views.py` ViewSets、`serializers/`、`urls.py`、`migrations/`）。调研确认 vestigial：前端零引用，违反项目 ForeignKey 规则，Plan/Subscription 功能已由 taskBill 资源订单+定价体系取代。同步清理 `settings.py` INSTALLED_APPS 和 `saas_project/urls.py` 路由。

<!-- 2026-07-26 (本日完成 3 条) -->

## [OPT-20260726-001] completed
**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: backend / Go / taskBill / sms
- taskBill 新增 `src/sms_gate.go` — SMS 手机验证门禁。`checkSmsRechargeGate(ctx, userID)` 调用 taskAuth `/api/internal/recharge-sms-gate/` 验证用户是否已完成手机号短信验证。支持 mock/off/live 三种模式（`TASKBILL_SMS_GATE` 环境变量），默认 live。`fetchSmsFeaturePolicy` 作为 fallback 从 Django 读取 system feature policy。遵循 kyc_gate.go 的 pattern：`smsGateError` 错误类型 + `writeSmsGateDenied` HTTP 403 响应 + `smsGateUserMessage` 中文错误文案。

## [OPT-20260726-002] completed
**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: backend / Go / taskBill / consent
- taskBill 新增 `src/consent_gate.go` — 充值条款同意验证。`validateRechargeConsent(ctx, userID, tenantID, consentID)` 调用 Django 新内部 API `/api/internal/taskbill/validate-recharge-consent/` 验证同意记录（归属用户/租户/状态/版本）。`attachConsentProviderRef` 在支付订单创建后绑定 provider_ref。`bindConsentOnPaymentCompleted` 在支付完成时绑定 transaction。对应 Django 侧在 `billing_bridge/internal_views.py` 新增 3 个内部端点。

## [OPT-20260726-003] completed
**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: infrastructure / APISIX / billing / migration
- APISIX billing 路由完成 Go 迁移：移除 `billing-payment-sms-gate` (priority 869)，所有 `/api/tenant/*/billing/*` 直连 taskBill (priority 868)。SMS 门禁 + consent 验证已由 taskBill (`sms_gate.go` + `consent_gate.go`) 内置处理，不再依赖 Django BillingProxyView。仅保留 `billing-sms-phone-verify` (priority 870 → Django) 用于 send/verify/status 视图。`api_route_ownership.yaml` 状态 `partial-go` → `go`。Django `BillingProxyView` 标记 deprecated（仅作 django-default 降级回退）。

<!-- 2026-07-25 (本日完成 7 条) -->

## [OPT-20260726-009] completed
- **Status**: completed
- **Completed**: 2026-07-26
- **Completion-Note**: `urls_task_events_internal.py` deleted (dead code — not included from any URL config); `BillingProxyView` retained as fallback (APISIX priority 868 covers main path); `api_route_ownership.yaml` stale route entries cleaned up
- **Source**: /goal Django→Go 迁移 Phase 2 收尾
- **Why**: billing_bridge task_events dispatch was no longer routed through Django
- **How**: Deleted urls_task_events_internal.py; cleaned api_route_ownership.yaml obsolete billing route entries

<!-- ═══════════════════════════════════════════════════════════════════════ -->
<!-- 2026-07-26 批量完成 (15 条，/goal + /loop 三轮执行) -->
<!-- ═══════════════════════════════════════════════════════════════════════ -->

## [OPT-20260726-011] completed — Internal API 端点废弃防回归机制

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: django / infrastructure
**Source**: /goal 修复 runAll smoke 两个 404 失败

**Summary**: Django `urls_feature_params_internal.py` 添加 catch-all `re_path(r'^.*$', ...)` 兜底返回 410 Gone。防止删除已迁移的内部 API 路径时遗忘 410 stub 导致 smoke 期望 410 却收到 404。

**Completion-Note**: 在 urlpatterns 末尾添加 `re_path` catch-all → `feature_params_tenant_gone` view，任何未匹配的内部 API 路径均返回 410 而非 Django 默认 404。

---

## [OPT-20260726-012] completed — legal_client 重复输出消除

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: python / django / legal
**Source**: /goal admin-login NameError 修复 (traceId: fcfe15a7)

**Summary**: `pending_post_login_privacy_for_user()` 将重复的字段提取逻辑委托给 `policy_to_pending_dict()`，消除两处独立维护相同 dict 格式的漂移风险。

**Completion-Note**: 函数末尾 `return {...}` 替换为 `return policy_to_pending_dict(current)`，与 OPT-016 缓存改动联动验证。

---

## [OPT-20260726-013] completed — 网关 legal 路由 CI 自检脚本

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: ci / gateway / go
**Source**: /goal privacy-policy/consent/ 404 修复 (traceId: 2ae28a9b)

**Summary**: 创建 `taskGateway/scripts/ci/check_routes.sh` — 扫描 taskBill `handlers.go:mountRoutes()` 的 handler 注册并与 `routes.yaml` 中 `upstream: taskBill` 的 URI 交叉比对，发现缺失路由时 CI 报错（`--ci` 模式 exit 1）。跳过 `/api/internal/*`、health/schema/swagger、webhook 回调等内部路径。当前验证 6/6 条公开路由全覆盖。

**Completion-Note**: 脚本内联 PyYAML 解析 + bash fallback，可在 CI pipeline 中作为 `bash scripts/ci/check_routes.sh --ci` 门禁使用。

---

## [OPT-20260726-014] completed — OrderCreate 手机验证状态预检 + 加载态优化

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: vue / frontend / billing
**Source**: /goal 订单创建页手机验证门禁前置

**Summary**: `OrderCreate.vue` 页面加载时即调用 `checkPhoneRequired()` 预检验证状态，结果通过 `phoneStatus` prop 传入 `PhoneVerificationGate.vue`。Gate 移除内部 `fetchStatus()` 独立请求，消除重复 API 调用（两次 `/billing/phone-verification-status/` → 一次）。

**Completion-Note**: OrderCreate 新增 `phoneVerificationStatus` ref + `onMounted` 预检；PhoneVerificationGate 新增 `phoneStatus` prop + `initPhoneInfo()` 替代 `fetchStatus()`。

---

## [OPT-20260726-014b] completed — test_legal_client 单元测试

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: python / testing / legal
**Source**: /goal 页面元素调整 work-panel 隐私政策阅读器弹窗卡加载

**Summary**: 新建 `accounts/tests/test_legal_client.py`（9 个 pytest 用例），覆盖 `policy_to_pending_dict` 输出完整性（5 字段验证）、类型正确性、缺字段默认值、`pending_post_login_privacy_for_user` 委托逻辑、无策略时返回 None、已同意时返回 None、taskBill 不可达安全降级。

**Completion-Note**: 使用 `mocker.patch` mock `get_current_privacy_policy` 和 `_user_has_consented_to_privacy_policy`，无需真实 HTTP 调用。

---

## [OPT-20260726-015] completed — taskBill KYC/SMS 门禁 feature_policy 第二层查询

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: go / billing / security
**Source**: /goal 订单创建页 KYC_TIER_BLOCKED 阻塞修复

**Summary**: `isSmsPhoneVerificationRequired()` 新增 Django `system_feature_policy.enable_recharge_phone_verification` 第二层查询（复用已有 `fetchSmsFeaturePolicy()` + 60s 缓存）；`checkKycPaymentGate` 同样添加 feature policy 回退。config `off` → skip；config `live` → 再查 Django，若 Django 返回 `false` 则也 skip。

**Completion-Note**: 添加 `fetchSmsFeaturePolicyCached()` 带 `sync.Mutex` + 60s TTL 缓存，避免每次支付校验额外 HTTP 调用。KYC gate 复用同一缓存（共享 endpoint）。

---

## [OPT-20260726-016] completed — legal_client HTTP 缓存

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: python / performance / legal
**Source**: /goal PrivacyReconsentGate "同意并继续" 弹窗重复弹出修复

**Summary**: `_user_has_consented_to_privacy_policy()` 添加进程内 dict 缓存（60s TTL，key=`(user_id, policy_id)`），每次访问时惰性清理过期条目。消除高频页面路由切换导致的重复 taskBill HTTP 查询。

**Completion-Note**: 使用 `time.monotonic()` 计时 + 每次查询后 prune 过期条目防止内存泄漏。缓存不可用时 fallback 到直接 HTTP 查询（不阻塞业务）。

---

## [OPT-20260726-017] completed — writeErrorJSON 推广到 taskBill 全部 handler

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: go / observability / billing
**Source**: /goal 修复订单创建 UNIQUE 约束冲突 + traceId 缺失

**Summary**: 6 个 handler 文件（`handlers_orders.go`、`handlers_billing_query.go`、`handlers_internal_charge.go`、`handlers_tenant.go`、`handlers_resource_pricing.go`、`handlers_admin_grant.go`）中所有 `writeJSON(w, status, map[string]string{"error": ...})` 替换为 `writeErrorJSON(w, status, ..., tracelog.TraceIDFromContext(r.Context()))`，前端可通过 `data-traceId` 展示 trace_id。

**Completion-Note**: sed 批量替换约 60 处 + 6 个文件添加 `"tracelog"` import。`handlers_orders.go` 已有 import 仅替换调用。

---

## [OPT-20260726-018] completed — Snowflake 共享 Go 包抽取

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: go / infrastructure / shared-lib
**Source**: /goal 数据库 ID 雪花算法元规则落地

**Summary**: 创建 `shareLib/snowflake/`（`snowflake.GenerateID() int64` + `snowflake.GenerateIDString() string`），统一替换 `taskEvents/internal/snowflake/`、`taskTenantService/src/snowflake.go`、`taskCloudService/src/snowflake.go` 三处独立实现。更新 4 份文档（`35_snowflake_id_generation.md`、`CLAUDE.md`、`project_rules.md`、`00_project_constraints.md`）。

**Completion-Note**: 三服务 go.mod 各添加 `replace snowflake => ../shareLib/snowflake`；旧 snowflake.go 均已删除；文档引用全部指向 `shareLib/snowflake`。

---

## [OPT-20260726-019] completed — BillingOrders 支付弹窗 QR 码本地渲染

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: vue / frontend / billing
**Source**: /goal billing/orders 页面添加支付/取消按钮

**Summary**: `PayOrderModal.vue` 和 `OrderCreate.vue` 的 QR 码改为本地 Canvas 渲染（`qrcode` npm 包），替代外部 `api.qrserver.com`。含 fallback：npm 包未加载时回退到 qrserver.com 图片，网络不可用时显示 "QR 加载失败" 文本。

**Completion-Note**: `package.json` 添加 `"qrcode": "^1.5.4"` 依赖；动态 `import('qrcode')` 按需加载；`PayOrderModal.vue` 新增 `renderQR()` + `watch`/`onMounted` 触发。

---

## [OPT-20260726-020] completed — SystemAdminResourcePricingPanel requires_unit_type 编辑

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: vue / frontend / admin
**Source**: /goal 价格管理页 GitLab 磁盘/流量购买约束

**Summary**: `SystemAdminResourcePricingPanel.vue` 的 GitLab 流量费区域添加 `<select>` 下拉框（无依赖 / GitLab 磁盘），管理员可通过 UI 修改 `requires_unit_type` 前置依赖，表单提交时发送 `gitlab_traffic_requires_unit_type` 字段。

**Completion-Note**: form 新增 `gitlab_traffic_requires_unit_type` 字段；submit 逻辑包含该字段；form reset 同步清空。

---

## [OPT-20260726-021] completed — taskTaskService hasWorkspaceAccess group_id 检查

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: go / acl / task-service
**Source**: /goal 任务列表为空问题排查（Phase 4 Django→Go 迁移遗留缺陷）

**Summary**: `hasWorkspaceAccess` 新增 `checkUserInGroup()` → 调用 taskTenantService `GET /api/internal/tenant/groups/user-in-group?user_id=X&group_id=Y` 验证组成员身份。config 新增 `TaskTenantServiceURL`；API 不可达时 fallback 到仅 `user_id` 检查（宽松策略）。

**Completion-Note**: `config.go` 添加 `TaskTenantServiceURL` + `baseConf.Services.TaskTenantService` 解析；`config.yaml` 添加 `taskTenantService: host: 127.0.0.1 port: 8020`；`project_client.go` 新增 `checkUserInGroup()` 5s 超时。

---

## [OPT-20260726-022] completed — taskTaskService 评论数据迁移 SQL

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: data-migration / sql / task-service
**Source**: /goal 任务列表为空问题排查

**Summary**: 创建 `dataMigrate/taskTaskService/002_migrate_comments_from_django.sql` — ATTACH DATABASE 方式将 Django `projects_comment` 迁移到 taskTaskService `comments` 表（映射 `todo_id→task_id`、`created_by_id`、`content`、`created_at`），新字段 `mentions_json`/`execution_mode`/`depends_on_comment_ids` 设默认值。幂等（`INSERT OR IGNORE`），自动被 `runDataMigrate()` 扫描执行。

**Completion-Note**: 仅迁移对应任务已存在的评论（`WHERE EXISTS (SELECT 1 FROM tasks WHERE id = ...)`），孤立评论跳过。

---

## [OPT-20260726-023] completed — 访问令牌独立页面 Profile 重定向 + 移动端适配

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: vue / frontend / ux
**Source**: /goal 访问令牌独立菜单页

**Summary**: `UserProfile.vue` 添加 info banner 引导到独立访问令牌页面；`UserAccessTokens.vue` flex 布局添加 `flex-col lg:flex-row` 响应式 class，侧边栏 `w-full lg:w-auto`。

**Completion-Note**: 两处模板级改动，无需 script 逻辑变更。

---

## [OPT-20260726-024] completed — SMS 验证状态始终返回真实值

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: go / billing / api
**Source**: /goal 订单创建页未绑定手机号时显示绑定入口

**Summary**: `phone_verification.go:handlePhoneVerificationStatus` 中 `fetchSmsVerifiedStatus` 从 `if required` 条件内移出，始终查询 taskAuth 获取真实 SMS 验证状态，API 返回值不再受策略开关影响。

**Completion-Note**: 单行逻辑变更 — `smsVerified := false; if required { smsVerified = fetchSmsVerifiedStatus(...) }` → `smsVerified := fetchSmsVerifiedStatus(...)`。

## [OPT-20260726-034] completed — BillingOrders 支付弹窗集成手机号验证门禁

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: vue-frontend / billing / payment

**Summary**: 管理员在 `SystemAdminLoginPaymentPolicy` 开启"支付前手机号验证"后，`BillingOrders.vue` 支付弹窗（`PayOrderModal.vue`）直接调支付 API 展示二维码，未检查手机验证状态。修复：1) `PayOrderModal.vue` 集成 `PhoneVerificationGate` 组件，两阶段显示（验证门禁 → QR码）；2) `useBillingOrderActions.js` 新增 `checkPhoneVerification()` 预检 + `initiatePayment()` 延迟支付，`openPayModal()` 先检查验证状态再决定是否调支付 API；3) `BillingOrders.vue` 传递新 props 并监听 `@phone-verified` 事件。统一了 `OrderCreate.vue` 和 `BillingOrders.vue` 两条支付路径的安全策略执行。

**Completion-Note**: 三文件修改，总计 ~70 行新增代码。PayOrderModal.vue 新增 `phone-verified` emit 和 `tenantId`/`phoneGateActive`/`phoneVerificationStatus` props；useBillingOrderActions 新增 3 个函数 + 2 个 ref；BillingOrders.vue 传递 3 个 props + 1 个事件处理。

## [OPT-20260726-036] completed — Profile 页面布局左对齐：侧边栏贴近左边缘

**Logged**: 2026-07-26 | **Completed**: 2026-07-26 | **Status**: completed
**Area**: vue-frontend / layout / profile

**Summary**: Profile 设置页面（7 个视图）原先使用 `max-w-6xl mx-auto px-6` 居中布局。在宽屏幕 (1920px) 上侧边栏距离左边缘 ~408px，内容区仅 ~860px 宽，无法容纳较宽的内容（如访问令牌管理面板）。用户反馈"把菜单往左移，以便内容可以放得下"。

**修复内容**:
1. 容器 `max-w-6xl` → `max-w-7xl`（1152px→1280px, +128px 总宽）
2. 移除 `mx-auto` 居中 → 左对齐（侧边栏从 ~408px → ~32px 贴近左边缘）
3. 响应式内边距 `px-6` → `px-4 sm:px-6 lg:px-8`
4. 间距 `gap-6` → `gap-5`（24px→20px）
5. Sidebar 增加 `shrink-0` 类防止被内容区压缩
6. `UserAccessTokens.vue` 新增 `lg:px-8` 大屏适配

**修改文件**: UserAccessTokens.vue, UserProfile.vue, UserReferral.vue, UserGitIdentities.vue, UserGitSiteOAuthSettings.vue, PersonalFeatureParamsConfigs.vue, UserCompanySettings.vue（共 7 个文件）

**Completion-Note**: Vite build 验证通过，11s 零错误。内容区可用宽度从 ~860px 增至 ~976px (+116px)，侧边栏从动态居中位置移至固定左边缘 32px 处。


## [OPT-20260726-031] completed

- **Status**: pending
- **Created**: 2026-07-26
- **Context**: 已在计费订单创建页（PhoneVerificationGate）、登录页（Login.vue）、用户资料页（UserProfilePhoneBindingPanel）、注册页（PhoneRegister）中将国家代码下拉框数据源从硬编码 `COUNTRY_DIAL_OPTIONS` 改为从 `/api/public/system-feature-policy/` 动态获取 `allowed_phone_country_codes` 过滤。API 错误/网络故障时自动回退到完整列表以保证可用性。当前实现正确但不展示过滤加载状态。
- **How**: 在 `useAllowedCountryCodes` composable 返回的 `loading`/`error` 状态基础上，为下拉框组件增加：1) 加载中 → select 右侧显示小 spinner；2) API 错误 → 不阻塞用户但在下拉旁显示 tooltip 提示"区域限制暂不可用，显示全部地区"。**Why**: 用户可能在策略已限制但 API 暂时不可达时看到完整列表，透明告知状态可避免困惑。
- **Related**: [[pricing-restructure-remove-normal-task]]

## [OPT-20260726-027] completed

- **Status**: pending
- **Created**: 2026-07-26
- **Context**: 将"登录与支付策略"从 SystemAdmin 仪表板提取为独立的 `/system-admin/login-payment-policy/` 页面。基本功能已迁移完毕，构建通过。后续可增强：添加面包屑导航、在仪表板快捷操作区添加入口卡片、评估其他嵌入式区块是否也值得独立提取。
- **How**: 1) 面包屑: `系统管理 > 安全策略 > 登录与支付策略`；2) 快捷入口: SystemAdmin.vue 快捷操作区添加策略卡片链接；3) 审计模板中其他内联区块。**Why**: 用户反馈该功能应独立成菜单页，当前已完成核心迁移，上述为可选 UX 增强。
- **Related**:


## [OPT-20260726-026] completed

- **Status**: pending (previously blocked by OPT-20260727-023 — unblocked 2026-07-27)
- **Created**: 2026-07-26
- **Context**: KYC 接口已从 Django 代理迁移至 Go taskAuth 公开端点（`/api/kyc/admin/users/*`, `/api/kyc/me/`）。当前 Go 侧仅有内部 KYC 端点测试（`kyc_test.go`），缺少新公开端点的 HTTP handler 测试。测试模式已在 `auth_users_test.go` 中验证（`setupTestAuthDB` + `getOrCreateToken` + `Authorization: Token`），handler 代码已审计完毕可编写测试，但 Go 编译因 MySQL 驱动缺失被阻塞。
- **How**: 在 `kyc_test.go` 或新测试文件中添加：staff/superuser 权限校验测试（非管理员 403）、管理员获取 KYC 聚合数据、管理员 override/AML/evaluate、普通用户 me/kyc 摘要。**Why**: 公开端点直接暴露给前端，需保证 auth gating 和响应格式正确。
- **Related**: [[resource-order-system]], OPT-20260727-023

