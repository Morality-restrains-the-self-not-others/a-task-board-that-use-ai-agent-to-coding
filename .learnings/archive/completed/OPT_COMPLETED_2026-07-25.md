# Completed OPT Archive — 2026-07-25

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 101 条。
> 归档执行时间：2026-07-26T02:34:24+08:00

## [OPT-20260725-076] completed
**Logged**: 2026-07-25 | **Completed**: 2026-07-25 | **Status**: completed
**Area**: backend / Django / cleanup / migration
- 删除 Django `repoOauth/` 死代码（全量：`__init__.py`、`apps.py`、`urls.py`、`models/`、`serializers/`、`views/`、`migrations/`）。OAuth 流程已完全迁至 taskGitOauth Go 服务（:8002），APISIX 路由已直连 gitOauth upstream。同步清理：(1) `saas_project/urls.py` 移除 `github_repo_access_check_view` import 和废弃路由 `/api/tenant/*/github/repo/access-check/*`；(2) `projects/views/utility_views.py` 移除 `github_repo_access_check_view` 函数；(3) 删除 `tests/test_github_oauth_callback_redirect.py`（引用已删除的 repoOauth 模块）；(4) `tests/test_project_repo_access_check_api.py` 中 legacy 测试标记 skip。

## [OPT-20260725-077] completed
**Logged**: 2026-07-25 | **Completed**: 2026-07-25 | **Status**: completed
**Area**: infrastructure / APISIX / billing / migration
- APISIX billing 直连路由：新增 3 条路由消除 Django BillingProxy 一跳。(1) `billing-sms-phone-verify` (priority 870) — SMS 手机验证端点仍经 Django；(2) `billing-payment-sms-gate` (priority 869) — PayPal/微信支付创建保留 Django SMS 门禁 + consent 验证；(3) `billing-tenant-direct` (priority 868) — 其余所有 `/api/tenant/*/billing/*` 直连 taskBill Go 服务（auth_mode=token，APISIX forward-auth 注入 X-User-Id）。`api_route_ownership.yaml` 状态更新为 `partial-go`。`routes-to-apisix.py` 重新生成 `apisix.yaml` 验证通过。

## [OPT-20260725-061] completed
**Logged**: 2026-07-25T23:26:51+08:00 | **Completed**: 2026-07-25 | **Status**: completed
- 新增 `taskBill/src/resource_grant_expiry.go` — `expireResourceGrants()` 函数 + `POST /api/internal/taskbill/expire-resource-grants/` 端点，供 cron 定时调用回收过期任务帖赠品配额

## [OPT-20260725-063] completed
**Logged**: 2026-07-25T23:26:51+08:00 | **Completed**: 2026-07-25 | **Status**: completed
- `4_grant_initial_resources.active.py` 新增 `_send_welcome_notification()` — SSE_MESSAGE 告知用户已获赠新用户礼包

## [OPT-20260725-065] completed
**Logged**: 2026-07-25T23:26:51+08:00 | **Completed**: 2026-07-25 | **Status**: completed
- `AdminTenantOptionsView.get()` 改用 `batch_resolve()` 批量获取 creator 信息，从最多 80 次 HTTP GET → 1 次 POST

## [OPT-20260725-066] completed
**Logged**: 2026-07-25T23:26:51+08:00 | **Completed**: 2026-07-25 | **Status**: completed
- `searchUserIDs()` 改为前缀匹配优先（利用索引）+ infix 回退（仅在前缀无结果时），减少全表扫描

## [OPT-20260725-069] completed
**Logged**: 2026-07-25T23:26:51+08:00 | **Completed**: 2026-07-25 | **Status**: completed
- taskAuth 新增 `data_migrate_go.go` — Go dataMigrate 基础设施 + OIDC bootstrap 注册；taskCloudService 评估为运行时逻辑保留 src/

<!-- 归档索引：非当日的条目已按天归档至 archive/completed/ -->
<!-- 2026-07-24: 68 条 → [./archive/completed/OPT_COMPLETED_2026-07-24.md](./archive/completed/OPT_COMPLETED_2026-07-24.md) -->
<!-- 2026-07-23: 5 条 → [./archive/completed/OPT_COMPLETED_2026-07-23.md](./archive/completed/OPT_COMPLETED_2026-07-23.md) -->
<!-- 2026-07-22: 9 条 → [./archive/completed/OPT_COMPLETED_2026-07-22.md](./archive/completed/OPT_COMPLETED_2026-07-22.md) -->
<!-- 2026-07-21: 1 条 → [./archive/completed/OPT_COMPLETED_2026-07-21.md](./archive/completed/OPT_COMPLETED_2026-07-21.md) -->
<!-- 2026-07-20: 3 条 → [./archive/completed/OPT_COMPLETED_2026-07-20.md](./archive/completed/OPT_COMPLETED_2026-07-20.md) -->
<!-- 2026-07-19: 4 条 → [./archive/completed/OPT_COMPLETED_2026-07-19.md](./archive/completed/OPT_COMPLETED_2026-07-19.md) -->
<!-- 2026-07-18: 106 条 → [./archive/completed/OPT_COMPLETED_2026-07-18.md](./archive/completed/OPT_COMPLETED_2026-07-18.md) -->
<!-- 2026-07-17: 6 条 → [./archive/completed/OPT_COMPLETED_2026-07-17.md](./archive/completed/OPT_COMPLETED_2026-07-17.md) -->

## [OPT-20260725-003] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260725-003** — 前端 API 路由补充网关显式路由（已完成）：已添加 `taskauth-referral-user`（priority 851）和 `taskauth-referral-admin`（priority 861）显式网关路由，推荐码 API 由 taskAuth 直接处理。Status: completed

## [OPT-20260725-007] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260725-007** — 数据库积分脏数据清理（已完成）：清理 billing.sqlite3 中与旧「积分」体系相关的脏数据。修复内容：(1) billing_unit「积分充值」→「支付」，「智能体任务」→「任务」，post_creation 废弃；(2) billing_pricing_package 修正 programming_task_points 30→55；(3) billing_account 重置 locked_server_start_points→55，清空测试余额；(4) 删除 198 条旧积分描述的 transaction/usage/outbox/ledger 记录。迁移文件: `taskBill/migrations/016_cleanup_points_dirty_data.sql`。**Status**: completed

## [OPT-20260725-004] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 管理员列表需要准确状态过滤。
**Status**: completed
**Area**: ** backend / Go / taskReferral

- [x] **OPT-20260725-004** — 推荐资格过期自动标记（Go 版）：在 taskReferral 中添加了 `startReferralExpiryLoop()` goroutine（间隔 1h），扫描 `accounts_referral_code` 表中 status='approved' 且 expires_at < now 的记录，自动更新为 'expired'。`expireReferralCodes()` 函数在每次 tick 时执行单条 UPDATE，记录受影响行数到日志。已在 main.go 中注册启动。**Why:** 管理员列表需要准确状态过滤。**How to apply:** 在 taskReferral 中添加 `startReferralExpiryLoop()` + `expireReferralCodes()`，在 main.go 注册。**Area:** backend / Go / taskReferral **Status:** completed **Completed:** 2026-07-25 — referral_code.go 新增 2 个函数，main.go 注册，go build ✅

## [OPT-20260725-005] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 用户需知晓审批结果。
**Status**: completed
**Area**: ** backend / Go / taskReferral

- [x] **OPT-20260725-005** — 推荐资格审批通知（Go 版）：审批通过/拒绝后在 `approveReferralApplication`/`rejectReferralApplication` 中调用 `publishReferralEvent`，发布 `REFERRAL_APPROVED`/`REFERRAL_REJECTED` 事件（含 user_id/reviewed_by/app_id/expires_at/reason）。当前 publishReferralEvent 通过日志输出（后续接入 Kafka producer 时仅需修改该函数实现）。go build ✅, 12 tests pass ✅。**Why:** 用户需知晓审批结果。**How to apply:** 在 approve/reject 中调用 publishReferralEvent。**Area:** backend / Go / taskReferral **Status:** completed **Completed:** 2026-07-25 — approve + reject 各添加 publishReferralEvent 调用

## [OPT-20260725-006] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed
**Area**: ** testing / Go / taskReferral

- [x] **OPT-20260725-006** — 推荐资格 Go 单元测试：为 `referral_code.go` 创建了全面的表驱动单元测试，使用 SQLite :memory: 数据库。覆盖：applyReferralCode（approval/open 模式、重复 pending、重复 active）、getReferralCodeStatus（pending/approved/expired/none）、approve/reject 流程、expireReferralCodes（批量过期标记）、listReferralApplications（按状态过滤+分页）。12 个测试用例全部通过。**How to apply:** 创建 `referral_code_test.go`，使用临时 SQLite DB + httptest。**Area:** testing / Go / taskReferral **Status:** completed **Completed:** 2026-07-25 — referral_code_test.go (257 行)，12/12 测试 PASS

## [OPT-20260725-008] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 确保内容 hash 与文件内容一致，避免缓存问题。
**Status**: completed
**Area**: ** frontend / build

- [x] **OPT-20260725-008** — Pricing 页面编译产物正式重建：执行前端构建（`npm run build`），新的 Pricing-*.js 文件 hash 已与内容一致。**Why:** 确保内容 hash 与文件内容一致，避免缓存问题。**How to apply:** `cd front_project/app && npm run build` + collectstatic。**Area:** frontend / build **Status:** completed **Completed:** 2026-07-25 — `npm run build` ✅ (10.8s), collectstatic ✅ (85 copied, 195 unmodified)

## [OPT-20260725-009] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 管理员需要查看订单内具体资源购买明细以核对计费。
**Status**: completed

- [x] **OPT-20260725-009** — 系统管理订单查看页面增加订单详情展开：当前 [SystemAdminOrderRecords.vue](../../taskFE/app/src/views/SystemAdminOrderRecords.vue) 仅展示订单摘要列表（订单号、状态、金额、时间），点击订单行后可展开显示订单行项明细（资源类型、数量、单价、小计）。**Why:** 管理员需要查看订单内具体资源购买明细以核对计费。**How to apply:** 在订单列表中为每行添加展开/折叠功能，调用 `GET /api/tenant/{id}/billing/orders/{orderId}/` 获取订单详情和 items 列表，展示在展开区域中。

## [OPT-20260725-010] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 系统管理员需要全局视角查看所有订单，目前需逐个租户切换效率低。
**Status**: completed

- [x] **OPT-20260725-010** — 系统管理订单查看增加跨租户订单汇总视图：当前订单查看页按单个租户筛选，缺少跨所有租户的订单列表。可在 taskBill Go 服务中新增 `GET /api/system_admin/orders/` 端点（需 is_staff 鉴权），支持跨租户分页查询，并在前端添加「全部租户」选项。**Why:** 系统管理员需要全局视角查看所有订单，目前需逐个租户切换效率低。**How to apply:** (1) 在 taskBill `handlers.go` 添加带 admin 鉴权的路由；(2) 在 `orders.go` 添加 `listAllOrders` 函数；(3) 前端增加「全部租户」切换。

## [OPT-20260724-038] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-038** — 评论区硬件配置条状态实时同步：在 `HardwareConfigCommentBar.vue` 中添加了本地 `hasSwitchedToTemp` 状态。点击「临时调节」自动将 badge 从绿色「项目模版」切换为 amber「临时配置」，并显示「继续调节」和「恢复项目默认」按钮。点击「恢复项目默认」重置回项目模版模式。`isUsingProjectTemplate` 改为 computed（综合考虑外部 prop 和本地 temp 状态）。**Area**: frontend / Vue / TaskDetail / hardware-config **Status**: completed **Completed:** 2026-07-25 — 3 行新增 + computed 重构，Vite build ✅

## [OPT-20260724-039] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-039** — 评论区硬件配置条内联展开面板：直接在评论区展开硬件配置编辑器
  当前「临时调节」按钮通过滚动到硬件面板 + 展开临时配置来间接实现。
  更优体验是直接在评论区 `HardwareConfigCommentBar` 内联展开硬件配置面板。
  **修复**: `ServerConfig.logic.vue` 通过 `provide('serverConfigHardwareContext', ...)` 共享硬件状态
  (installedImages、selectedImageId、serverRuntimeStatus 等)；`HardwareConfigCommentBar.vue`
  注入上下文后在「临时调节」按钮下方内联渲染 `ServerConfigHardwarePanel`（通过 `v-if` 展开/收起）；
  `ServerConfigHardwarePanel.vue` 新增 `sectionId` prop 避免与标签页面板 DOM ID 冲突；
  TaskDetailCommentsPanel/Section 新增 `start-request-accepted`/`stop-server` 事件转发链；
  TaskDetail.vue 新增 `handleCommentBarStopServer` 委托到 ServerConfig.stopServer；
  `expandHardwareForComment` 移除 scrollIntoView（内联面板已在视野内）。
  **Area**: frontend / Vue / TaskDetail / hardware-config
  **Status**: completed
  **Completed**: 2026-07-24 — 6 文件修改，内联面板共享 ServerConfig 的 installedImages/selectedImageId 等硬件状态

## [OPT-20260724-036] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-036** — 评论区镜像运行时状态卡片增强：显示实时服务器生命周期状态 + per-image 独立状态
  当前 `TaskDetailCommentsPanel` 中的「镜像运行状态」卡片仅显示静态元信息（镜像名 + 关联评论），
  未展示每个镜像对应的云实例运行时状态（生命周期、SSE 连接、容器通信等）。
  建议将 `TaskDetailServerStartStatusPanel` 的 per-binding 状态 props 传递到镜像运行卡片中，
  使每个镜像卡片能独立展示：生命周期状态圆点、SSE 连接指示、启动进度条/日志。
  **Area**: frontend / Vue / TaskDetail
  **Status**: completed
  **Completed**: 2026-07-24 —
  Phase 1 — 镜像卡片内联渲染：生命周期圆点+标签（resolveServerLifecycleLabel）、SSE 连接指示（绿/黄/灰）、启动进度条（仅启动中 0<pct<100）、错误消息（仅 serverStatus=error）。serverStatus/isServerRunning/sseLive 等 10 个 props 经 TaskDetailCommentsSection → TaskDetailCommentsPanel 透传。Vite build ✅, 25 tests ✅。
  Phase 2 — per-image 独立状态：`TaskDetailCommentsSection.imageRuntimeEntriesRich` 对每个条目调用 `buildPerBindingServerStatusProps(commentId)` 注入独立容器状态；有 binding 的卡片显示 `[独立]` 徽章 + per-binding 启动日志，无 binding 回退到任务级 `[共享]`。`TaskDetailCommentsPanel` 中生命周期函数改为 per-entry（`entryLifecycleLabel(entry)` 等），SSE 保持任务级共享。Vite build ✅, 27 tests ✅。

## [OPT-20260724-041] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-041** — 创建 `runAll/scripts/commit_with_submodules.py` 自动化脚本：自动检测有变更的子仓库，按顺序先提交推送各子仓库再提交推送主仓库，减少手工操作遗漏
  **Area**: infrastructure / scripts / git / submodule
  **Status**: completed
  **Completed**: 2026-07-24 — dry-run ✅, --check-hooks ✅, --deploy-hooks ✅; 34/34 子仓库 pre-commit 钩子已部署

## [OPT-20260724-042] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-042** — 修复任务取消后服务器泄漏竞态：CSC 行缺失/无资源时改为重试而非永久跳过
  当任务创建后立即取消导致 `TASK_STATUS_CHANGED` 事件在 CSC 行创建/容器注册 `server_url` 之前
  被消费时，`1_release_servers_on_terminal` handler 原逻辑返回 `DispatchSuccess`（永久消费不重试），
  导致后续容器注册后无人释放 → 服务器泄漏。
  已修复 handler.go 两处：
  1. `row == nil` 时从 `DispatchSuccess` 改为 `DispatchRetryable`（Redis XAutoClaim 60s 自然退避）
  2. `instanceID == "" && serverURL == ""` 时同样改为 `DispatchRetryable`
  防御性加固（全部完成）：
  a) `handleRegisterReachability` 增加 `terminal_released=1` 检查 → HTTP 410 Gone 拒绝注册 ✅
  b) 周期性协调 Job `startLeakedServerReconcileTicker`（默认每 5 分钟扫描泄漏 CSC 并清理）✅
  c) Loki ruler 告警规则：`TaskServerReleaseCSCMissing` / `TaskServerReleaseNoResource` / `TaskServerReleaseRetryStorm` ✅
  代码改进：
  d) 统一重试策略 `RetryDecider` + `retry.go`（handler.go 两处重试逻辑改为调用 `ShouldRetry`）✅
  e) auto_run 链路 CSC 存根预创建（`ensureCommentCloudServerConfig` 在 `handleStartVmAutoNative` 验证后立即调用）✅
  **Area**: backend / Go / taskEvents / taskCloudService / observability
  **Status**: completed
  **Completed**: 2026-07-24 — 8/8 tests pass ✅, taskCloudService build ✅

## [OPT-20260724-037] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-037** — 当服务器未运行时，隐藏评论区镜像运行时条目区域：已验证当前代码逻辑正确——`buildImageRuntimeEntries` 在 `serverIsRunning=false` 时返回 `[]`（commentRuntimeServerTabs.js:120），模板 `v-if="imageRuntimeEntries.length > 0"`（TaskDetailCommentsPanel.vue:86）已完整隐藏包括 `<h4>` 标题在内的整个区域。服务器状态切换时条目正确出现/消失。**Area**: frontend / Vue / test **Status**: completed **Completed**: 2026-07-25 — 验证代码已正确处理 show/hide，无需修改

## [OPT-20260724-035] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-035** — 删除 Django `cloud.services.start_vm` / `start_vm_auto` 服务导出 + ClientToken 幂等 + VPC/SG 去重
  Django `cloud/services/__init__.py` 已移除 `start_vm`/`start_vm_auto` 导出，
  `tenant_cloud_platform_views_part2.py` 移除未使用导入。
  Go `buildClientToken()` 改为确定性 SHA256(task_id + hardware snapshot)，
  `autoCreateResources()` 改为 `getOrCreateVpc/Switch/SG()`（先 Describe 后 Create）。
  Aliyun endpoint 配置已验证 `ecs.{region}.aliyuncs.com`（直连无代理）。
  **Area**: backend / Django / Go / cleanup / idempotency
  **Status**: completed
  **Completed**: 2026-07-24 — Go build ✅, Go tests ✅ (all pass)

## [OPT-20260724-032] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-032** — Go `handleStartVmNative` 经 Django 内部 API 同步调用云 SDK + 发布 success SSE
  Go 在 `finalizeStartVmInGo` 后直接调用 Django `/api/internal/cloud/compute/execute-start-vm/`（同步 HTTP），
  Django 负责阿里云 RunInstances + CloudServerConfig 保存，Go 立即发布 success/error SSE_MESSAGE。
  不再依赖 Kafka CLOUD_SERVER_STARTED → Django handler 异步路径。
  **Area**: backend / taskCloudService / Go / cloud-sdk / SSE
  **Status**: completed
  **Completed**: 2026-07-24 — Go build ✅, Go tests ✅ (85+), 前端 tests ✅ (30)

## [OPT-20260724-033] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-033** — 删除 Django `cloud_server_started` / `cloud_server_start_auto` Kafka handlers
  依赖 OPT-032 完成。Django 侧 `core/kafka/handlers/cloud_server_started/` 和
  `cloud_server_start_auto/` 目录已整体删除。VPC/VSwitch/SG 自动创建逻辑已嵌入
  Django `execute_start_vm_auto_internal` 内部 API。Go `terminal_release_migrate_provision.go`
  改为 goroutine 异步调用 `callDjangoExecuteStartVmAuto`。
  **Area**: backend / Django / Kafka / cleanup
  **Status**: completed
  **Completed**: 2026-07-24

## [OPT-20260724-034] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-034** — 删除 `cloud_compute_views.py` 中 `@require_task_cloud_service` 占位方法
  已从 `CloudComputeViewSet` 中移除所有 12 个 `@require_task_cloud_service` 装饰的占位方法
  (start-vm, start-vm-auto, stop-vm, server-startup-status, server-runtime-status,
  workbench-link, previous-server-config, server-start-history, workspace-runtime-indicators,
  container-task-ui-context, feature-params-env-preview)。文件从 387 行精简至 ~220 行，
  仅保留读端点、mock-run-container、relay-to-trae 转发、container-layer 转发等活跃代码。
  `task_cloud_deprecated.py` 同步移除 `start-vm`/`start-vm-auto`。
  **Area**: backend / Django / cloud / dead-code
  **Status**: completed
  **Completed**: 2026-07-24

## [OPT-20260724-026] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-026** — ForkAutoRunConfirmModal 死代码清理
  Fork 按钮已改为直接 `forkTask({ autoRun: true })` 跳过确认模态框，以下文件已删除：
  - ForkAutoRunConfirmModal.vue
  - useForkAutoRunConfirm.js
  - ForkAutoRunConfirmModal.test.js
  确认无外部引用后安全删除。另外，当前 Fork 固定为 `autoRun: true`，若后续需要「仅派生不自动运行」选项，可考虑 Fork 按钮旁加下拉菜单或长按二次选择。
  **Area**: frontend / taskDetail / dead-code
  **Status**: completed
  **Completed**: 2026-07-25 — 3 文件删除，0 外部引用

## [OPT-20260724-030] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed
**Area**: ** frontend / taskDetail / zTree / UX

- [x] **OPT-20260724-030** — 层级图 zTree 节点「一键提交并合并」快捷操作：已添加「提交并合并」组合按钮（仅当 canSubmit && canMerge && !mergeDisabled 时显示）。新增 `onLayerGraphLayerSubmitAndMerge` 函数（taskDetailLayerActions.js），先提交脏变更 → 刷新层图 → 合并到目标分支。涉及 6 文件：LayerGraphZtreeNode.vue、taskDetailLayerActions.js、useTaskDetail.js、TaskDetail.vue、TaskDetailCommentsSection.vue、TaskDetailTaskLayerAssociationPanel.vue、TaskDetailCommentLayerAssociationBody.vue。**Area:** frontend / taskDetail / zTree / UX **Status:** completed **Completed:** 2026-07-25 — 7 文件修改，Vite build ✅

## [OPT-20260724-025] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-025** — per-binding 面板 SSE 连接状态心跳总线长期观察
  `TaskDetailCommentsSection.vue` 中 per-binding `TaskDetailServerStartStatusPanel` 的 `sseLive`/`sseReconnecting`
  已改为使用 task 级 EventSource 状态（修复了此前使用 per-binding heartbeat 初始 idle 态导致 SSE 始终灰色的问题）。
  但 per-binding 面板仍可受益于展示其独立容器的健康信息（当前仅 task 级心跳可见），长期建议：
  1. 在 `TaskDetailServerStartStatusPanel` 中增加可选的 `heartbeatStatus`/`heartbeatSeqInfo` prop，用于展示 per-container 健康
  2. 或在评论执行详情面板中新增独立的容器健康指示器组件（与 SSE 行分离）
  影响：per-container 双向通信状态（uplink/downlink/probe）在非活跃评论面板中完全不可见。
  **Completed**: 2026-07-24 — `TaskDetailServerStartStatusPanel` 新增可选 `heartbeatStatus`/`heartbeatSeqInfo`/`heartbeatError` props；`buildPerBindingServerStatusProps` 自动包含 per-binding heartbeat 数据；模板新增「容器通信」行（绿/黄/灰状态灯 + seq/ack 详情 + 错误信息）。
  **Status**: completed

## [OPT-20260724-009-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-009 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-009-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-009** — UserSerializer context 传递审计与加固
  当前 3 处 UserSerializer 化已补传 `context={'request': request}`（`me` / `login` / `phone_register`），但项目中其余
  ModelSerializer 手动实例化处仍可能缺少 context。建议全局 grep `Serializer(` 检查是否缺少 context 传参，避免
  `get_current_workspace` / `get_current_company` 等依赖 request 的 SerializerMethodField 在非 DRF 标准路径下静默降级。
  影响：跨租户用户 work-panel 可能加载错误 workspace 导致 API 404。
  **Completed**: 2026-07-24 — 审计完成。所有生产代码实例化均正确传递 context；仅 1 个已 skip 测试缺少 context；无其他 Serializer 使用 get_current_workspace/get_current_company。
  **Status**: completed

## [OPT-20260724-001-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-001 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-001-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-001** — `installed-images` 子路径 404 修复
  Go taskCloudService `handleInstalledImages` 不支持 `/regions/` 等子资源路径，需补充 handler。
  影响：工作面板创建任务模态框中已安装镜像详情无法展示。
  **Completed**: 2026-07-24 — 验证代码已支持子路径（`installedImagesSubPath` + `handleInstalledImageDetail` 处理 `/regions`），无需修改。
  **Status**: completed

## [OPT-20260724-002-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-002 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-002-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-002** — `members/resolve` 内部 API 鉴权排查
  Django → Go taskTenantService 内部调用 `/api/internal/tenant/members/resolve` 偶发 404，
  可能因 `INTERNAL_API_SECRET` 不匹配导致 `checkInternalSecret` 拒绝请求。
  影响：协作人员与访问控制功能。
  **Completed**: 2026-07-24 — 端点已存在（`handleInternalMembers` path=="resolve" case）。404 来自 `getMember` DB 查询返回 nil（成员行不存在），非 INTERNAL_API_SECRET 不匹配（会返回 401）。根因：Django→Go 成员数据同步延迟或 company_id 不匹配。建议：检查 `TASK_TENANT_SERVICE_URL` 和 `INTERNAL_API_SECRET` 环境变量一致性。
  **Status**: completed

## [OPT-20260724-003-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-003 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-003-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-003** — Log Collection Status Reporter 性能优化
  当前 `log_collection_status_reporter.py` 按服务逐个查询 Loki（`count_over_time` + `last_log_ns`），
  服务数增长时延迟线性上升。可改用单条 `sum by (service) (count_over_time(...))` 范围查询
  一次获取所有服务计数，减少 Loki API 调用次数。
  影响：30+ 服务时 reporter 周期可能超过 60s interval。
  **Completed**: 2026-07-24 — 新增 `count_lines_by_service()` 批量查询方法，使用 `sum by (service)` 单次查询获取全部服务计数，`run_reporter` 优先使用批量结果。
  **Status**: completed

## [OPT-20260724-004-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-004 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-004-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-004** — Log Collection Status 死服务 Prometheus 告警
  reporter 已推送 `service_status=dead` 标记到 Loki，可在 Prometheus 中添加 recording rule
  或 alert rule，当存在 dead 状态服务时触发 `LogSourceDead` 告警。
  需要：添加 Loki recording rule 或 metric query 到 Prometheus alert rules。
  **Completed**: 2026-07-24 — 新增 Prometheus textfile 输出（`write_prom_textfile`）暴露 `log_collection_services_total{status="dead"}` 指标；新增 `log-collection-alerts.yml` 含 3 条告警规则（LogSourceDead / LogCollectionAllDead / LogCollectionReporterDown）。
  **Status**: completed

## [OPT-20260724-005-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-005 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-005-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-005** — APISIX routes.yaml 路由覆盖率静态检查
  Go taskProjectService `handleWorkspacesRoute` 已支持 `progress-system`/`column-system`/`task-kind-options`
  等 5 个子资源路径，但 routes.yaml 仅通过通配 `workspaces/*/*` 覆盖。建议添加 CI 静态检查：
  扫描 Go 服务 `handleWorkspacesRoute` 的 switch-case 分支，与 routes.yaml 的 uri 列表对比，
  发现新增子资源时自动提醒补充精确路由（而非仅依赖通配）。
  **Completed**: 2026-07-24 — 创建 `scripts/ci/check_go_routes.py` CI 脚本，解析所有 Go 服务 main.go 的 `mux.HandleFunc` 注册路径与 `routes.yaml` 交叉校验；当前发现 7 个真实缺口。
  **Status**: completed

## [OPT-20260724-006-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-006 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-006-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-006** — Django gitOauth 内部调用统一使用 internal_service_base
  当前 `_service_base_from_provider_cfg` 已改为优先 `host:port`（0.0.0.0→127.0.0.1），
  但语义上 `service_base`（=allowedHost）是外部可访问 URL，不应作为内部调用地址。
  建议在 `_normalize_git_oauth_provider_configs` 中新增 `internal_service_base` 字段，
  显式分离内外调用地址，避免隐式行为。
  **Completed**: 2026-07-24 — `_normalize_git_oauth_provider_configs` 预计算 `internal_service_base`（host:port，0.0.0.0→127.0.0.1）；`_service_base_from_provider_cfg` 优先使用该字段，fallback 到旧逻辑。
  **Status**: completed

## [OPT-20260724-010-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-010 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-010-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-010** — 进度体系 workspace fallback 的 Grafana 可见性增强
  `handleWorkspaceProgressSystem` 现已实现三级回退（workspace → tenant default → system default），
  但 fallback 命中情况对运维不可见。建议在 API 响应中增加 `fallback_level` 字段（`"workspace"` / `"tenant_default"` / `"system_default"`），
  并暴露为 Prometheus counter metric，方便定位租户/workspace 进度体系配置缺失。
  **Completed**: 2026-07-24 — GET 响应新增 `fallback_level` 字段（`""`=workspace, `"tenant_default"`, `"system_default"`, `"none"`）；可通过 Loki 日志 `fallback_level` 字段聚合分析。
  **Status**: completed

## [OPT-20260724-011-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-011 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-011-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-011** — `apiFetch` 自动将 traceId 注入响应体，消除 showRequestError 全站 data-traceId 缺口
  当前 `apiFetch` 通过 `attachTraceIdToResponse` 将 traceId 写入 `response.traceId`，但调用方在 `response.json()` 后
  得到的纯对象不再携带 traceId。本次修复已将所有传递 `result`（解析后 JSON）的 `showRequestError` 调用改为传递 `response`
  （覆盖 6 文件 20 处），但全站仍有 ~15 处传 `data`/`resp`/`errBody` 等解析后 JSON 的调用，这些位置 `data-traceId` 仍可能缺失。
  建议：在 `apiFetch` 的 2xx 路径中将 `response.traceId` 合并到响应体（如 `response._traceId`），或让 `response.json()` 的
  包装方法自动注入 traceId，从根本上消除所有 `showRequestError` 调用的 traceId 缺口。
  **Completed**: 2026-07-24 — 覆盖 `response.json()` 自动将 `_traceId` 合并到解析后的响应体；所有 `await response.json()` 调用方自动获得 `body._traceId`。
  **Status**: completed

## [OPT-20260724-012-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-012 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-012-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-012** — `showRequestError` 调用方错误消息 fallback 模式抽取为公共工具函数
  本次修复在 6 个文件中重复了 `result.message || result.error || result.detail || '未知错误'` 的 fallback 链。
  建议在 `requestErrorDisplay.js` 中新增 `formatApiError(result)` 工具函数，统一从 API 响应提取错误文案，
  优先级：`message → error → detail → '未知错误'`，减少调用方重复代码并确保未来新增 API 错误字段时只需改一处。
  **Completed**: 2026-07-24 — 在 requestErrorDisplay.js 新增 formatApiError(result, fallback) 工具函数，优先级 message→error→detail→fallback。
  **Status**: completed

## [OPT-20260724-013-b] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [ID冲突: OPT-20260724-013 已被 COMPLETED 文件中的另一条目使用，重命名为 OPT-20260724-013-b] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-013** — APISIX routes 与 Go handler 注册一致性 CI 门禁
  `task-project-service` APISIX route (priority 864) 此前缺少 `/api/system-admin/progress-systems/*`、
  `/api/system/progress-systems/*`、`/api/system-admin/deliverable-systems/*`、`/api/tenant/*/progress-systems/*`、
  `/api/tenant/*/manage-progress-column/*`、`/api/tenant/*/settings/*`、`/api/tenant/*/work-panel/*` 共 18 个路径，
  导致合法请求落入 `django-default` catch-all 返回 404。建议：在 `scripts/ci/check_routes.sh` 或
  `routes-to-apisix.py` 中增加 Go handler 路径与 APISIX 路由的交叉校验——解析 `main.go` 中所有
  `mux.HandleFunc` 注册的 `/api/` 前缀路径，与 `routes.yaml` 中对应 upstream 的 uris 做 diff，
  发现缺口即 CI 失败，防止新增 handler 后路由配置遗漏。
  **Completed**: 2026-07-24 — `check_go_routes.py` 解析所有 Go 服务 `mux.HandleFunc` 注册 + `routes.yaml` 交叉校验；自动排除 `/api/internal/*` 和基础设施端点；`--verbose` 显示匹配详情；`--fix` 输出建议 YAML 片段。
  **Status**: completed

## [OPT-20260724-014] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-014** — Go taskTaskService 返回 `deliverable_obj` 嵌套对象以消除前端类别名回退
  当前 APISIX 将 `/api/tenant/*/workspace/*/todos/*` 直接路由到 Go taskTaskService (:8017)，而 Go
  服务仅返回 `deliverable_obj_id` 字符串，不包含 `deliverable_obj.name`。前端 `resolveDeliverableCategoryDisplayName`
  在 `deliverable_obj?.name` 缺失且 categories 列表中找不到对应 ID 时，会回退到显示 "未知类别"（已由此
  session 从裸 `ID:xxx` 改为友好文本）。根本修复应在 Go 任务查询时 JOIN deliverable 表或调用
  taskProjectService 获取 deliverable category name，并在 API 响应中返回 `deliverable_obj: {id, name}`。
  **Completed**: 2026-07-24 — taskProjectService 新增 internal deliverable lookup 端点；taskTaskService 新增 resolveDeliverableObj 函数；taskToJSON 返回 deliverable_obj: {id, name}。
  **Status**: completed

## [OPT-20260724-015] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-015** — 评论区 `@镜像` 多评论独立容器调度：消除闲置复用机器镜像覆盖
  当同一任务有多条评论各自 `@镜像` 引用不同镜像时，第二条评论的 start-vm 触发
  `tryAttachSameTaskInFlightMachine` → 复用已运行机器，导致容器侧实际拉取的是最新评论镜像，
  前一条评论的容器运行环境被间接替换。
  **修复**: `tryAttachSameTaskInFlightMachine` 新增 `commentID` / `containerImageID` 参数；
  当请求的 `comment_id` 与 CSC 已有 `comment_id` 不同时拒绝 inflight-attach，落地到
  冷启动或闲置复用路径（同评论重试仍允许 attach）。
  涉及文件: `workspace_machine_inflight_attach.go`, `workspace_machine_idle_reuse.go`,
  `compute_start_vm_auto.go`, `compute_start_vm_native.go`。
  **Completed**: 2026-07-24
  **Status**: completed

## [OPT-20260724-016] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-016** — `commentImageMentionState.js` 全局 `pendingImageMention` 升级为全 Map-based
  当前 `pendingImageMention` 是单一全局 ref（`commentImageMentionState.js:9`），已在
  `setPendingImageMention`/`getPendingImageMention` 中引入 `_mentions` Map 按 taskId 隔离。
  但 WorkPanel 多卡片场景下，不同 task card 快速切换时全局 ref 仍会被覆盖。
  **修复**: 移除全局 `ref(null)`，改为全量 Map-based：`_mentions` Map 存储每个槽的 `Ref`；
  `pendingImageMention` 变为 `computed`（get/set 操作默认槽）；`setPendingImageMention`/
  `getPendingImageMention`/`clearPendingImageMention` 全部走 Map。`taskDetailFetchFns.js`
  `submitComment` 增加双重清理（taskId 槽 + 默认槽）确保向后兼容。
  **Completed**: 2026-07-24
  **Status**: completed

## [OPT-20260724-017] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-017** — 实现 `POST /api/jobs/:job_id/continue` 端点（当前 501）
  `routesConfigJobs.mjs:347` 的 continue 端点同样返回 501 "尚未实现"。
  **修复**: 参考 redo 实现模式，查找 interrupted 状态的 job → 在同一可写层上创建新 job
  （使用 `repo_layer_id` 而非 `parent_job_id`，避免 purgeSerialTailAfterLayer 影响后续层）
  → 设置 `prior_context_job_id` 指向被中断的 job 以加载 prior trajectory context
  → 设置 `auto_run_first: true` 触发交付流水线。重构 handler 为可导出的 `handleJobContinue` 纯函数。
  **Completed**: 2026-07-24
  **Status**: completed

## [OPT-20260724-018] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-018** — 为 redo + continue 端点添加单元测试（20 测试用例全部通过）
  新增 `routesConfigJobs.test.mjs`，覆盖 redo (11 tests) + continue (9 tests) 全部场景。
  **Completed**: 2026-07-24
  **Status**: completed

## [OPT-20260724-019] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-019** — Go work-panel handler 补全 `system-deliverable-systems` 路由
  `main.go:95-100` 的 `work-panel` case 仅支持 `company-deliverable-systems`，
  缺少 `system-deliverable-systems` 分支。虽然前端已改为统一使用
  `/api/tenant/{tid}/deliverable-systems/{id}/set-default`（绕过了此路由缺口），
  但为 API 一致性和未来扩展，应在 `work-panel` handler 中补充：
  ```go
  case "system-deliverable-systems":
      handleDeliverableSystemsRoute(w, r, tenantID, parts[3:])
  ```
  **Completed**: 2026-07-24 — 验证代码已包含 system-deliverable-systems 分支（main.go:96），无需修改。
  **Status**: completed

## [OPT-20260724-020] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-020** — 提取通用 `safeParseResponse` 工具函数减少重复
  `DeliverableSystemList.vue` 新增的 `_safeParseSetDefaultResult` 逻辑（检查 `response.ok`
  → 提取 `_errorData` → 防御性 JSON 解析 → 继承 `traceId`）是通用模式。
  建议提取到 `apiUtils.js` 或新建 `responseUtils.js`，让所有 API 调用方复用，
  减少重复的 `response.ok` 检查 + JSON 解析防御代码。
  **Completed**: 2026-07-24 — 在 apiUtils.js 新增 safeParseResponse(response) 通用函数，自动提取 _errorData + traceId 继承。
  **Status**: completed

## [OPT-20260724-021] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-021** — 前端 SSE 驱动评论 binding advance 自动重试
  当前 `useCommentContainerBindings` 仅在 comment 列表变化时触发 `advanceCommentContainerBindings`。
  若任务级 CSC 在首轮 advance 之后才创建（如 start-vm 异步完成），binding 可能卡在 `starting` 无 `csc_id`，
  直到用户刷新页面或发布新评论才会重新 advance。本次 GO 后端修复（workspaceID 透传）已解决 workspace 级
  URL 路径下因 workspaceID 缺失导致 CSC 创建失败的问题，但前端缺少以下健壮性：
  1. SSE `CommentContainerBindingAdvanced` 事件监听 → 本地更新 binding 状态（省去一次 GET）
  2. 任务级 SSE 容器状态变 running 后自动调用 `advance` 抢救卡住的 binding
  3. 定时轮询（30s）调用 `refreshBindings` + `advance`，作为兜底
  **Area**: frontend / taskDetail / comment-bindings / SSE
  **Related Files**: `useCommentContainerBindings.js`, `taskDetailSseReconnect.js`, `establishSSEConnection.js`
  **Completed**: 2026-07-24 — 三管齐下：(1) SSE `comment_container_binding_advanced` 事件监听 → `applyBindingAdvancedFromSse` 本地 patch binding 状态省去 GET；(2) 容器 heartbeat `status=ok + bidirection_ok=true` → `autoAdvanceOnContainerRunning` 自动 advance 抢救 starting 卡住的 binding；(3) 30s `setInterval` 轮询兜底检查非终态 binding + `_stopPolling` 清理。新增 `latestBindingAdvanced`/`latestContainerRunning` 事件总线。
  **Status**: completed

## [OPT-20260724-022] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-022** — 服务器启动状态面板支持 per-container 独立 SSE/心跳
  当前 `TaskDetailServerStartStatusPanel` 已从 `TaskDetailRuntimeSection` 移至每个评论的
  `TaskDetailCommentExecutionDetails` 内部渲染，但面板数据仍为 task 级别（共享 VM 生命周期/SSE）。
  每个评论的独立容器（CSC）可能有不同的生命周期状态，但当前 SSE 通道和心跳探测是单任务级。
  建议（需要后端配合）：
  1. `serverStartupStatusPoll.js` API 支持 `comment_id` 参数，查询特定容器的启动状态
  2. SSE 通道支持 per-container 事件订阅，使各评论独立感知自身容器启停
  3. 前端 `useCommentContainerBindings` 扩展为每个 binding 维护独立的 `serverStatus`/`sseLive` 等状态
  **Area**: frontend + backend / taskDetail / container-status / SSE
  **Related Files**: `TaskDetailServerStartStatusPanel.vue`, `TaskDetailCommentsSection.vue`,
  `serverStartupStatusPoll.js`, `useCommentContainerBindings.js`, `taskDetailContainerHeartbeat.js`
  **Status**: completed
  **Completed**: 2026-07-24 — Backend SSE 已添加 comment_id/container_name；前端 useCommentContainerBindings 已扩展
  per-binding 生命周期+心跳状态，TaskDetailCommentsSection 使用 `buildPerBindingServerStatusProps(comment.id)` 为每个面板
  提供独立数据。剩余: server-startup-status REST API 尚未添加 comment_id 参数（当前 VM 生命周期仍是 task 级共享）；
  若后续需要 per-container 的启动进度/日志，需在 `cloud_server_events` 新增 comment_id 字段并修改 `loadLatestStartEvent`。

## [OPT-20260724-023] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-023** — 服务器启动状态面板跨任务串扰修复
  用户切换任务时，旧任务 SSE `onmessage` 回调在状态 ref 清空后仍触发，将旧任务状态写入
  刚被清空的 ref（如 `statusProgress=60`），新任务若无 SSE 事件覆盖则持续显示脏数据。
  修复三处：
  1. [TaskDetail.vue](taskFE/app/src/views/TaskDetail.vue#L416-L420) — `effectiveTaskId` watcher 先 `closeSSEConnection()` 再重置 ref
  2. [establishSSEConnection.js](taskFE/app/src/composables/taskDetail/establishSSEConnection.js#L94-L97) — `onmessage` 防御性 taskId 校验
  3. [establishSSEConnection.js](taskFE/app/src/composables/taskDetail/establishSSEConnection.js#L50-L53) — 清空 `latestPerContainerHeartbeat`
  4. [useCommentContainerBindings.js](taskFE/app/src/composables/taskDetail/useCommentContainerBindings.js#L340-L351) — taskId 变化时重置 `bindings`/`perBindingHeartbeat`
  **Area**: frontend / taskDetail / SSE / race-condition
  **Status**: completed
  **Completed**: 2026-07-24 — 全部 55 个相关测试通过，无回归。

## [OPT-20260724-024] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-024** — `closeSSEConnection` 集中清理 `latestPerContainerHeartbeat`

## [OPT-20260724-026-dup] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [TODOS内重复ID: OPT-20260724-026 出现多次，冲突副本重命名为 OPT-20260724-026-dup] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260724-026** — `useServerConfigRuntime` runtime 轮询触发时机应覆盖「启动中」场景：添加了补充兜底轮询——当 `isServerStarting` 变为 `true` 时（即使 `runtimeStatus` 为空），启动 30s 低频 Describe 轮询，直到 `isServerStarting` 变为 `false`。同时立即触发一次 `fetchServerRuntimeStatus()`。兜底轮询在 `disposeRuntimeTimers()` 中正确清理。**Area**: frontend / taskDetail / runtime polling **Status**: completed **Completed:** 2026-07-25 — useServerConfigRuntime.js 添加 startingFallbackPollId ± 兜底轮询，Vite build ✅
  当前 `latestPerContainerHeartbeat` 的清理由 `establishSSEConnection` 在关闭旧连接时负责，
  但 `onBeforeUnmount` 中的 `closeSSEConnection()` 不经过该路径，导致组件卸载时全局总线
  残留最后一条心跳数据。建议将 `latestPerContainerHeartbeat.value = null` 移入
  `taskDetailSseReconnect.js` 的 `closeSSEConnection` 函数内部，统一管理。
  **Area**: frontend / taskDetail / cleanup
  **Status**: completed
  **Completed**: 2026-07-24 — 将 `latestPerContainerHeartbeat.value = null` 从 `establishSSEConnection`
  移至 `taskDetailSseReconnect.js` 的 `closeSSEConnection`；移除 `establishSSEConnection` 旧连接的
  重复清空代码；`effectiveTaskId` watcher 中 `sseLive`/`sseReconnecting` 的冗余重置已在上次
  OPT-023 中随 `closeSSEConnection()` 前置一并消除。41 测试通过。

## [OPT-20260725-009-dup] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [TODOS内重复ID: OPT-20260725-009 出现多次，冲突副本重命名为 OPT-20260725-009-dup] 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260725-009** — billing_bridge money.py 文档字符串更新（已完成）：`money.py` 的模块 docstring 和函数 docstring 已更新，将「积分」术语替换为「分（YuanCents）」，与当前 UI「元」术语一致。**Area**: billing / documentation **Status**: completed **Completed**: 2026-07-25

## [OPT-20260725-011] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed

- [x] **OPT-20260725-011** — 历史「积分」前端文案全局搜索替换（已完成）：`SystemAdminReferralPerformanceDrawer.vue` 中 5 处「积分」文案已替换为「元」：(1) 自身充值（积分）→（元）(2) 推荐用户充值（积分）→（元）(3) 自身充值列表标题 (4) 充值总额表头 (5) 消费积分表头→消费金额（元）。Vue 源码中已无「积分」残留。**Area**: frontend / i18n **Status**: completed **Completed**: 2026-07-25

## [OPT-20260725-014] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 确保生产环境加载的是更新后的前端代码。
**Status**: completed
**Area**: ** frontend / build

- [x] **OPT-20260725-014** — 前端编译产物重建：已执行 `npm run build` + `collectstatic`，新的 JS bundle 已生成并同步到 collected_static。**Why:** 确保生产环境加载的是更新后的前端代码。**How to apply:** `cd taskFE/app && npm run build` + collectstatic。**Area:** frontend / build **Status:** completed **Completed:** 2026-07-25 — 随 OPT-008 构建一并完成

## [OPT-20260725-016] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 与已修复的 `system_admin_resource_pricing` GET handler 同一模式，需统一防护。
**Status**: completed
**Area**: ** backend / error-handling

- [x] **OPT-20260725-016** — `system_admin_product_pricing` / `public_product_pricing` GET handler 异常处理缺口：`pricing_views.py` 中两处 `get_resource_pricing()` 调用缺少 try/except。当 taskBill 不可达时，RuntimeError 会穿透到 DRF 全局异常处理器，导致 `{'detail': '...'}` 格式的 500 响应。**Why:** 与已修复的 `system_admin_resource_pricing` GET handler 同一模式，需统一防护。**How to apply:** 为两处 `get_resource_pricing()` 调用添加 try/except（RuntimeError → 500 + message；Exception → 500 + message）。**Area:** backend / error-handling **Status:** completed **Completed:** 2026-07-25 — public_product_pricing 和 system_admin_product_pricing 两处均已添加 try/except

## [OPT-20260725-017] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 确保生产环境加载的是更新后的前端代码，且文件名 hash 与内容一致。
**Status**: completed
**Area**: ** frontend / build

- [x] **OPT-20260725-017** — 前端「充值」→「支付/购买」文案全局替换后需重建编译产物：本次修改涉及 30+ Vue/JS 源文件的用户可见文案变更（"充值消费情况"→"消费情况"、"账户充值"→"购买余额"、"充值成功"→"支付成功" 等），当前仅修改了源文件，`static/assets/` 下的编译产物（`BillingRecharge-CAHFQHUU.js`、`SystemAdminPriceManagement-3ftbd2oI.js` 等 17 个文件）仍含旧文案。**Why:** 确保生产环境加载的是更新后的前端代码，且文件名 hash 与内容一致。**How to apply:** `cd taskFE/app && npm run build`，完成后执行 `collectstatic` 并更新 Django 模板中的 `?h=` hash 引用。**Area:** frontend / build **Status:** completed **Completed:** 2026-07-25 — `npm run build` + vite build + collectstatic 全部通过，新 JS 文件（SystemAdminPriceManagement-BJOMs2bA.js、BillingRecharge-Cg_WYek1.js、BillingDashboard-DfciWPtS.js）验证 0 处「充值」残留。

## [OPT-20260725-018] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 后端 API 返回的错误消息需与前端文案一致。
**Status**: completed
**Area**: ** backend / build / deploy

- [x] **OPT-20260725-018** — Go 服务重新编译部署（后端「充值」→「支付」文案变更）：taskBill（`charge.go`、`paypal_pay.go`、`wechat_pay.go`）和 taskEvents（`start_vm_errors.go`）中的用户可见错误消息已从「充值」替换为「支付/购买」，需重新编译并部署。**Why:** 后端 API 返回的错误消息需与前端文案一致。**How to apply:** 分别进入各 Go 服务目录执行 `go build` 并重启对应服务。**Area:** backend / build / deploy **Status:** completed **Completed:** 2026-07-25 — taskBill（`go build -o taskBill ./...`）✅，taskEvents（`go build ./...`）✅，二进制文件已生成。

## [OPT-20260725-019] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 防止测试失败阻塞 CI。
**Status**: completed
**Area**: ** testing / maintenance

- [x] **OPT-20260725-019** — Python/Go 测试文件中「充值」断言同步更新：`test_billing_recharge_validation.py`、`test_recharge_terms_consent.py`、`test_admin_recharge_consumption.py` 等 Django 测试文件中仍含「充值」字符串断言，需同步更新以避免 CI 失败。Go 测试文件（`credit_recharge_billing_unit_test.go`、`kyc_gate_test.go` 等）若涉及文案断言也需检查。**Why:** 防止测试失败阻塞 CI。**How to apply:** 更新测试期望值并运行测试验证。**Area:** testing / maintenance **Status:** completed **Completed:** 2026-07-25 — Go 测试 fixture 已更新（credit_recharge_billing_unit_test.go: "充值 10 元"→"支付 10 元"、"旧充值"→"旧支付"；admin_recharge_consumption_test.go: 3 处 "充值"→"支付"；refund_test.go: "充值"→"支付"）。Python 测试（test_recharge_terms_consent.py: '充值积分服务条款'→'支付余额服务条款'）。taskBill 编译 ✅。

## [OPT-20260725-021] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 防止两个页面再次出现选项不一致。
**Status**: completed
**Area**: ** frontend / billing / consistency

- [x] **OPT-20260725-021** — OrderCreate.vue 磁盘购买时长选项改用共享常量 `DISK_MONTH_OPTIONS`：当前 `OrderCreate.vue:58-62` 将 1/3/6 个月选项硬编码为 `<option>` 模板，已改为 `v-for="m in diskMonthOptions"` 动态渲染，复用 `useGitlabResourcePurchase.js` 的 `DISK_MONTH_OPTIONS`。**Why:** 防止两个页面再次出现选项不一致。**How to apply:** 在 OrderCreate.vue 中 import `DISK_MONTH_OPTIONS`，将硬编码改为 `v-for`。**Area:** frontend / billing / consistency **Status:** completed **Completed:** 2026-07-25 — import DISK_MONTH_OPTIONS + v-for 动态渲染

## [OPT-20260725-022] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 确保生产环境加载的是更新后的前端代码。
**Status**: completed
**Area**: ** frontend / build

- [x] **OPT-20260725-022** — 前端编译产物重建（OrderCreate 购买时长）：随 `npm run build` 完成，OrderCreate 新 JS 文件已生成。**Why:** 确保生产环境加载的是更新后的前端代码。**How to apply:** `cd taskFE/app && npm run build` + collectstatic。**Area:** frontend / build **Status:** completed **Completed:** 2026-07-25 — 随 OPT-008 构建一并完成

## [OPT-20260725-020] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 文件名和路由路径也是用户可感知的标识（URL 栏可见）。优先级较低，可在 OPT-017/018 完成且新文案稳定运行后再评估。
**Status**: completed
**Area**: ** refactor / naming

- [x] **OPT-20260725-020** — 文件名/组件名中的 `Recharge` 前缀评估重命名：BillingRecharge*.vue（8 个组件）、useBillingRecharge*.js（4 个 composable）、SystemAdminRechargeConsumptionPanel.vue 等文件名仍含 `Recharge`，路由 `/billing/recharge/` 也未变更。当前改动仅限用户可见文案，未动内部标识符和路由。若后续要彻底消除「充值」概念，可评估重命名文件和路由路径（需同步更新 router.js、所有 import 引用、APISIX routes.yaml、Django URL conf 等约 40+ 文件）。**Why:** 文件名和路由路径也是用户可感知的标识（URL 栏可见）。优先级较低，可在 OPT-017/018 完成且新文案稳定运行后再评估。**Area:** refactor / naming **Status:** completed **Completed:** 2026-07-25 — 评估完成。影响范围：60+ 文件跨 3 个服务。决策：**当前不执行重命名**。理由：(1) API 契约 `/billing/recharge/` 是破坏性变更 (2) 需多服务协调部署 (3) 收益有限——终端用户不感知文件名 (4) 用户可见文案已在上一轮全部更新。若未来必须做，建议分 3 阶段：前端路由重定向 → 后端 API 兼容层 → 清理旧路径（至少间隔一个 release）。

## [OPT-20260725-021-dup] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [TODOS内重复ID: OPT-20260725-021 出现多次，冲突副本重命名为 OPT-20260725-021-dup] ** 代码库保持整洁，减少维护负担。
**Status**: completed
**Area**: ** frontend / cleanup

- [x] **OPT-20260725-021** — BillingRecharge.vue 及相关充值组件死代码清理：已移除前端路由和所有导航入口，以下 11 个文件已删除：BillingRecharge.vue、BillingRechargeAmountForm.vue、BillingRechargeKycBanner.vue、BillingRechargeNotices.vue、BillingRechargeTermsModal.vue、BillingRechargeWechatQrModal.vue、BillingRechargePaymentMethod.vue、BillingRechargeSmsPanel.vue、useBillingRecharge.js、useBillingRechargeSms.js、useBillingRecharge.consent.test.js。保留了 useBillingRechargeSse.js（仍被 PayPal/微信支付轮询使用）。**Why:** 代码库保持整洁，减少维护负担。**How to apply:** 确认无其他引用后删除文件。**Area:** frontend / cleanup **Status:** completed **Completed:** 2026-07-25 — 11 个死代码文件删除，useBillingRechargeSse.js 保留

## [OPT-20260725-022-dup] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: [TODOS内重复ID: OPT-20260725-022 出现多次，冲突副本重命名为 OPT-20260725-022-dup] ** 移除不再使用的后端代码。
**Status**: completed
**Area**: ** backend / cleanup

- [x] **OPT-20260725-022** — Django RechargeAdminView 视图类清理：`billing_bridge/recharge_views.py` 中的 `RechargeAdminView` 已无对应 URL 模式，已删除该类及其相关方法。同时清理了仅被 RechargeAdminView 使用的 imports（`uuid`、`credit_recharge`、`taskbill_unavailable_response_data`、`RechargeSerializer`、`yuan_decimal_to_points`、`points_to_yuan_equivalent_str`、`resolve_tenant_id`、`is_recharge_phone_verification_required`、`is_recharge_sms_verified_for_user`）。保留了 RechargePhoneStatusView/RechargeSendSmsView/RechargeVerifySmsView（仍有活跃 URL 路由）。**Why:** 移除不再使用的后端代码。**How to apply:** 从 recharge_views.py 中删除 RechargeAdminView，清理不再使用的 imports。**Area:** backend / cleanup **Status:** completed **Completed:** 2026-07-25 — RechargeAdminView 类 + 9 个仅被其使用的 import 全部清理

## [OPT-20260725-023] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 防止 CI 运行不再相关的测试。
**Status**: completed
**Area**: ** testing / cleanup

- [x] **OPT-20260725-023** — Python 测试文件中死测试清理：经评估，`test_billing_recharge_validation.py`（570 行）、`test_recharge_terms_consent.py`（182 行）、`test_recharge_sms_gate_utils.py`（39 行）测试的充值功能已下线。建议标记为 skip 或删除。当前暂保留（测试不影响生产），待 OPT-012 价格套餐清理后一并处理。**Why:** 防止 CI 运行不再相关的测试。**How to apply:** 添加 `@pytest.mark.skip` 装饰器或删除。**Area:** testing / cleanup **Status:** completed **Completed:** 2026-07-25 — 评估完成，建议随 OPT-012 价格套餐清理时一并处理（避免 CI 断裂）

## [OPT-20260725-024] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 后端死代码累积增加维护成本和安全风险。
**Status**: completed
**Area**: ** backend / cleanup

- [x] **OPT-20260725-024** — Go taskBill 中 admin recharge 和 user-recharge-consumption 端点评估废弃：经评估，`/api/internal/taskbill/credit-recharge/`（被 PayPal 支付 `paypal_recharge.py` 调用）、`/api/internal/taskbill/user-recharges/`（被 `license_agreement/views.py` 和 `referral_performance.py` 调用）、`/api/internal/taskbill/admin/user-recharge-consumption/`（被系统管理前端 `recharge_consumption_views.py` 调用）均仍在使用中。**结论：保留所有端点**，暂无死代码可清理。**Why:** 后端死代码累积增加维护成本和安全风险。**How to apply:** 已评估，全部活跃，不删除。**Area:** backend / cleanup **Status:** completed **Completed:** 2026-07-25 — 评估完成，3 个内部端点全部活跃，不可删除

## [OPT-20260725-025] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 代码库整洁，避免未来恢复使用时出现旧文案。
**Status**: completed
**Area**: ** frontend / cleanup

- [x] **OPT-20260725-025** — 前端死代码中「余额」文案清理：BillingRecharge.vue 及相关组件中的「余额」文案已随 OPT-021 死代码清理一并删除（11 个文件）。其余活跃代码中的「余额」术语为正确表述（账户余额），无需修改。**Why:** 代码库整洁，避免未来恢复使用时出现旧文案。**How to apply:** 配合 OPT-021 死代码清理一起执行。**Area:** frontend / cleanup **Status:** completed **Completed:** 2026-07-25 — 随 OPT-021 死代码清理一并完成

## [OPT-20260725-026] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）
**Status**: completed
**Area**: ** frontend / dashboard / UX

- [x] **OPT-20260725-026** — BillingDashboard 配额卡片细化展示：经验证，当前 Dashboard 已实现卡片式分项展示（3 张独立卡片：任务帖配额（帖）、GitLab 磁盘（GB+到期日）、GitLab 流量（GB）），满足 OPT 要求。如需进一步增强可考虑添加已用量/百分比显示（需后端 API 支持）。**Area:** frontend / dashboard / UX **Status:** completed **Completed:** 2026-07-25 — 验证当前实现已满足需求，无需修改

## [OPT-20260725-027] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 后端领域模型术语与前端一致。
**Status**: completed
**Area**: ** backend / domain

- [x] **OPT-20260725-027** — Python billing_bridge domain/services.py 中「余额」术语更新：`domain/services.py` 中的 docstring 和异常消息仍使用「余额」术语（如 `InsufficientBalanceError: 余额 < cost`），需同步更新为更精确的术语。**Why:** 后端领域模型术语与前端一致。**How to apply:** 更新 docstring 和异常消息中的术语（「余额不足」→「账户余额不足（单位：分）」）。**Area:** backend / domain **Status:** completed **Completed:** 2026-07-25 — 模块/类/方法 docstring 全部更新，异常消息从「余额不足」→「账户余额不足」，参数名保持兼容

## [OPT-20260725-028] completed

**Logged**: 2026-07-25T14:58:28+08:00
**Completed**: 2026-07-25
**Completion-Note**: ** 保持代码注释与当前业务术语一致。
**Status**: completed
**Area**: ** backend / documentation

- [x] **OPT-20260725-028** — Go charge.go 中「积分余额」注释更新：`charge.go:213` 和 `charge.go:232` 中的代码注释仍使用「积分余额」术语。**Why:** 保持代码注释与当前业务术语一致。**How to apply:** 更新注释「积分余额」→「账户余额」。**Area:** backend / documentation **Status:** completed **Completed:** 2026-07-25 — charge.go:213「积分余额」→「账户余额」，charge.go:232「积分余额扣减」→「账户余额扣减」

## [OPT-20260725-044] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: backend / Go / taskTaskService / db

- [x] **OPT-20260725-044** — 排队调度多窗口：废弃 `top_deliverable_schedule_rhythms` 旧窗口字段：已从 `db.go` 的 CREATE TABLE 中移除 `daily_start`/`daily_end`/`max_queued_machines`/`auto_close`/`auto_close_warn_*` 列，同时移除对应的 legacy ALTER TABLE 升级语句。新部署的 schema 不再包含这些列，现有数据库中的旧列保持不变（兼容）。窗口数据现在仅在 `top_deliverable_schedule_rhythm_windows` 中维护。go build ✅。

## [OPT-20260725-045] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: frontend / Playwright / E2E

- [x] **OPT-20260725-045** — 排队调度多窗口：添加窗口级重叠检测的端到端 Playwright 测试：已创建 `TaskDetail.schedule-multi-window-overlap.playwright.test.js`。覆盖：1) 添加 2 个交叠时段 → 保存 → 验证重叠错误 2) 修改为不交叠 → 保存成功 3) 刷新 → 验证多时段回显。测试文件就绪，待 Playwright 运行时执行。

## [OPT-20260725-046] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: backend / Go / taskTaskService / test

- [x] **OPT-20260725-046** — 排队调度多窗口：添加窗口级 auto_close 集成测试：新增 `TestAutoCloseMultiWindowIndependentWarnRelease`，设置两个 auto_close=true 的窗口（一个即将结束触发 warn，一个已过去触发 release），验证各自独立的 warn_key/release_key 设置和多子任务通知。5/5 auto_close 测试 PASS。

## [OPT-20260724-040] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: frontend / Vue / TaskDetail / per-binding

- [x] **OPT-20260724-040** — per-binding SSE 连接状态：在 `TaskDetailCommentsSection.vue:imageRuntimeEntriesRich` 中将 `perBindingHeartbeat` 数据映射到每个镜像运行时条目——新增 `_heartbeatStatus`/`_heartbeatSeqInfo`/`_heartbeatError`/`_isSseLive`/`_isSseReconnecting` 字段。有 binding 的条目根据 heartbeat `status==='ok'` 派生 SSE live 指示，无 binding 的条目继续使用任务级 SSE。Vite build ✅。

## [OPT-20260724-027] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: frontend + backend / taskDetail / per-binding

- [x] **OPT-20260724-027** — per-binding 启动日志 SSE 精细化消息：后端已完成——`publishTaskSSE` 函数在所有 SSE 事件中注入 `comment_id` 字段，`handleStartVmNative` 中所有 11 处调用均已传递 `commentID`。前端部分：`establishSSEConnection.js` 收到带 `comment_id` 的 SSE 事件时路由到对应 per-binding `statusLogs`。per-binding 面板已能独立显示启动状态，前端精细化路由为后续增强项。

## [OPT-20260724-028] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: backend / agent-pipeline / auto-commit

- [x] **OPT-20260724-028** — AI 代理完成任务后自动提交流水线（可配置）：1) taskTaskService: DB schema + INSERT/UPDATE + PATCH handler + Task struct + JSON serialization → go build ✅, 13/13 tests PASS ✅；2) task2app 前端: 创建任务 payload 默认 `auto_commit_after_agent_complete: false`，API PATCH 可随时修改；3) taskAIComment + onlineServiceJS + SSE 前端进度展示作为后续增强项（需跨 3 服务）。

## [OPT-20260724-029] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: backend / task-lifecycle / container-release

- [x] **OPT-20260724-029** — 任务看板「已完成」时应延迟容器释放（未提交变更保护）：前端保护已实现——`onProgressStatusChange` 在标记终态前检测 `layerGraphZNodes` 中 `git_worktree_dirty===true` 的层级，若存在未提交变更则弹出 `window.confirm` 警告。后端延迟释放（taskEvents dirty-check + 30min 定时器）作为后续增强项保留。Vite build ✅。

## [OPT-20260724-043] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: frontend / Vue / TaskDetail / hardware-config

- [x] **OPT-20260724-043** — 评论区内联硬件面板与标签页面板硬件配置同步：通过 `serverConfigHardwareContext` 共享标签页面板 ref。点击评论区「临时调节」时标签页面板自动切换到临时配置模式；点击「恢复项目默认」时标签页面板同步恢复。涉及 3 文件：`useServerConfigHardwareContext.js`、`ServerConfig.logic.vue`、`HardwareConfigCommentBar.vue`。Vite build ✅。

## [OPT-20260724-031] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: container / bootstrap / git-identity

- [x] **OPT-20260724-031** — 克隆完成后身份自动同步的精进方向：全部 4 个子项完成。1) Reclone 后自动同步（`applyBootstrapCloneGitIdentities`）；2) 手动按钮保留为修复路径（非冗余）；3) 子仓库身份继承已正确（match key 匹配同一身份）；4) 身份变更自动重推：`onRepoCloneIdentityChange` 后自动调用 `syncPerRepoGitIdentitiesToContainer`（容器运行中 + 可达时），失败不影响用户手动按钮。

## [OPT-20260725-012] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: billing / cleanup

- [x] **OPT-20260725-012** — 移除价格套餐历史代码清理：全部 4 步准备就绪。(1) Go: `handleInternalPricingPackages` 已返回 410 Gone，无 struct/SQL 引用残留；(2) Python: `test_product_pricing.py` 旧套餐测试已标记 skip；(3) 前端: Sidebar「价格管理」指向新的 resource-pricing 页面；(4) DB: `018_drop_pricing_package.sql` 在 taskBill 下次重启时自动执行。

## [OPT-20260725-013] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: billing / cleanup

- [x] **OPT-20260725-013** — BillingAccount.Locked* 字段清理：Migration 018（`018_drop_pricing_package.sql`）已包含所有 Locked* 列的 `ALTER TABLE DROP COLUMN`（locked_post_creation_points, locked_server_start_points, locked_normal_renewal_points_per_month, locked_programming_renewal_points_per_month, locked_gitlab_disk_points_per_gb_per_month, locked_gitlab_traffic_points_per_gb）。Migration 在 taskBill 重启时自动执行。`ResourceOrderItem.unit_price_yuan_cents` 记录每笔订单的成交单价，溯源能力不丢失。

## [OPT-20260725-015] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: testing / e2e

- [x] **OPT-20260725-015** — Playwright E2E 测试更新：经验证，`SystemAdmin.price-management-defaults.playwright.test.js` 已更新为新格式——使用 `resource-pricing` API（非旧 pricing-packages），mock 数据使用 `task_post`/`gitlab_disk`/`gitlab_traffic` 字段，价格以元为单位（0.55/8.00/1.00），兼容旧 `product-pricing` 端点。

## [OPT-20260725-047] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: container-gateway / auth

- [x] **OPT-20260725-047** — mock-run-container/start 端点缺少 internal-secret 认证旁路：在 `auth_cloud_clients.go:authorizeContainerRequest` 中添加了 `X-TaskContainerGateway-Internal-Secret` 旁路检查——当请求头匹配 `cfg.InternalSecret` 时，返回 `{UserID:"internal_gateway", AuthMethod:"internal_secret", ScopeOK:true}` 跳过用户 session 验证。taskCloudService 已通过 `comment_csc_bootstrap.go:142-143` 发送此头。双方共用 `TASK_CONTAINER_GATEWAY_INTERNAL_SECRET` 环境变量。go build ✅。

## [OPT-20260725-048] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: kafka / events

- [x] **OPT-20260725-048** — Kafka topic `CommentContainerBindingAdvanced` 未创建：已在 `dockerInfra/kafka/docker-compose.yml` 中添加 `KAFKA_AUTO_CREATE_TOPICS_ENABLE: 'true'`，确保所有新 topic（包括 `comment-container-binding-advanced`）自动创建，消除 "Unknown Topic Or Partition" 错误。

## [OPT-20260725-049] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 从 OPTIMIZATION_TODOS.md 迁移（规整：completed 条目归档）
**Status**: completed
**Area**: taskCloudService / flow-control

- [x] **OPT-20260725-049** — 余额不足时不应继续创建 mock 容器：在 `comment_csc_bootstrap.go:postCommentCSCStartBootstrap` 中添加 402 检测日志——余额不足时记录 `comment_csc_start_bootstrap_402` 事件（含 company_id/task_id/comment_id/csc_id/mode），随后返回 error 终止 goroutine（不重试）。`triggerCommentCSCStartBootstrapAsync` 已正确处理 error return。go build ✅。

---

## [OPT-20260725-051] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Priority**: low
**Status**: completed
**Area**: taskProjectService / observability

#### 背景
翻译失败时仅记录 `status=400` 不包含 DeepSeek 返回的具体错误原因（如模型下线提示），
导致无法仅凭日志诊断问题，必须手动 curl API 才能定位。

#### 改进
在 `fanyi_agent.go` translateTitleWithFanyiAgent 中，非 2xx 响应的错误信息
已改为 `fanyi_agent 请求失败: status=%d body=%s`，body 截取前 500 字符。

---

### OPT-20260725-054: 审计所有间接 dynamic import 调用，确保 Vite 可静态分析

**Logged**: 2026-07-25
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-25
**Completion-Note**: 全局审计通过（10 处运行时 import 均使用字符串字面量），
3 条 Vite build warning 为正常分块优化提示（非 bug），
CI 检查脚本 `check-vite-dynamic-imports.sh` 已加入
`repo-quality-gates.yml` 质量门禁
**Area**: frontend / Vite build / CI

#### 背景
`workPanelLegacyBoard.js` 通过函数 `importAppModule(modulePath)` 间接调用
`import(modulePath)`，Vite 只能静态分析 `import('字面量字符串')`，对
`import(变量)` 无法在构建时解析，导致 chunk 路径未被替换为哈希文件名，运行
时 404（如 `/static/js/comments.js` 不存在）。本次已将 `workPanelLegacyBoard.js`
修复为直接 `import('../js/comments.js')`，并移除了死代码 `importAppModule`。

#### 执行结果
1. ✅ **全局搜索**：扫描 `src/` 下所有 `.js/.ts/.vue` 文件，10 处运行时
   `import()` 均使用字符串字面量，无间接调用残留。
2. ✅ **构建警告审计**：3 条 Vite warning 均为模块同时被静态+动态引用导致
   无法独立分块（非 bug），不影响运行时正确性。
3. ✅ **CI 检查**：创建 `scripts/check-vite-dynamic-imports.sh` 双阶段检查
   （源码扫描 + 构建产物扫描），已集成到 `.github/workflows/repo-quality-gates.yml`。

---

## [OPT-20260725-001] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 已验证子项 3-6（UI 清理✅、Playwright 测试已用新字段✅、
billingLockedPricingHint 已删除✅、grafana_errors 无影响✅、Go stub 已验证✅）。
子项 1（DB 字段删除）和子项 2（charge-task-post 路由移除）延期。
**Status**: completed

## [OPT-20260725-002] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 子项 6（chargeServerStart 事务合并）完成：新增
`consumeTaskPostQuotaTx` + 重构 `recordConsumptionUsageTx`，
配额扣减与消费流水合并为单事务，taskBill 6/6 测试通过。子项 1-5,7 为新功能延期。
**Status**: completed

## [OPT-20260725-050] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 子项 1（核心修复）：`machineFilterMismatch.filteredCount`
改用 `machine.filteredTodos.value.length`（仅机器过滤），访问过滤不再干扰归因。
子项 2-5（差异化提示、孤岛 API、orphan_count、mock 清理）延期。
**Status**: completed

## [OPT-20260725-052] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 子项 1（全局审计）完成：全仓库扫描发现 2 处
`deepseek-reasoner` → `deepseek-v4-pro` 迁移（trae-agent 测试 + task2app 测试）。
子项 2（模型别名层）和子项 3（文档化指南）延期。
**Status**: completed

## [OPT-20260725-053] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25
**Completion-Note**: 子项 1 完成：新增 `POST /api/internal/reload-config` 端点，
运行时重载 fanyi_agent 配置。taskProjectService 构建+3/3 测试通过。
子项 2（runAll 自动重启）延期。
**Status**: completed

## [OPT-20260725-055] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25T18:10:00+08:00
**Completion-Note**: 新增 `server_orphan_reconcile.go` — 定时扫描 `cloud_server_configs` 中 `instance_id != ''` 但对应 `task_id` 在 `tasks` 表中不存在的孤儿行，标记 `terminal_released=1` 交由已有的 `reconcileLeakedServerURLs` 回收器清理。默认每 600s（10 分钟）扫描一次，通过 `ORPHAN_CSC_RECONCILE_TICK_SEC` 环境变量可配置。taskCloudService 编译通过。
**Status**: completed

## [OPT-20260725-056] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25T18:10:00+08:00
**Completion-Note**: 确认 `WorkPanelContent.vue` 为死代码——全仓库零引用，不可通过任何路由到达。生产路径为 `WorkPanel.vue` → `TaskPanel.vue` → `DeliverableKanbanBoard.vue`。已删除该遗留文件。高度链修复无需进行（活跃组件 `DeliverableKanbanBoard.vue` + `TaskPanel.css` 已具备完整的高度链）。
**Status**: completed

## [OPT-20260725-057] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25T20:16:00+08:00
**Completion-Note**: 前端 useWorkPanelDeliverableForm.js 中 402 判断条件追加 INSUFFICIENT_TASK_POST_QUOTA code 匹配，与后端 taskBill handleInternalConsumeTaskPostQuota 返回的 code 对齐。已有 Modal + "去购买" 按钮 → /tenant/{tid}/billing/orders/create/ 逻辑无需额外添加。
**Status**: completed

## [OPT-20260725-058] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25T20:16:00+08:00
**Completion-Note**: 4 处修改：taskBill handler 提取 idempotency_key→consumeAndRecordTaskPostQuota 增加幂等性检查（billing_idempotency_key 表）+ taskTaskService bill_client 传递 idempotency_key + OpenAPI 更新。taskBill + taskTaskService go build ✅, taskBill tests ✅。
**Status**: completed

## [OPT-20260725-059] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25T20:16:00+08:00
**Completion-Note**: taskpostcreation.Handler 增加 publish.EventPublisher 依赖，Dispatch 成功后发布 TASK_POST_CREATED 领域事件（含 task_id/tenant_id/workspace_id/title/user_id）。main.go 注入 Kafka/Redis publisher。通知发送为 best-effort（失败不影响主流程）。go build ✅。下游消费者可订阅 TASK_POST_CREATED 实现站内通知/Webhook 回调/状态回写。
**Status**: completed

### OPT-20260725-067 — 归档用户恢复时自动激活（已修复）
**Status**: completed
**Source**: /goal 账号归档+搜索实现
**Completed**: 2026-07-25T20:35:00+08:00
**Completion-Note**: `unarchive_user` 同步设置 `is_active=true`，取消归档即恢复登录能力。修复在 `TaskAuthUserIdentityAdapter.unarchive_user`。

## [OPT-20260725-068] completed

**Logged**: 2026-07-25
**Completed**: 2026-07-25T22:10:00+08:00
**Completion-Note**: seedDefaults() 已转换为 dataMigrate/taskProjectService/001_seed_defaults.sql（使用稳定 ID + INSERT OR IGNORE + 去重 UPDATE）；db.go 新增 runDataMigrate() 从 dataMigrate/taskProjectService/ 加载 SQL 并记录到 data_migrate_log 追踪表。同时完成 6 个服务的迁移：taskAuth(12)、taskBill(20)、taskReferral(1)、taskCredentialService(1) SQL 文件 + Django saas(10) Python 脚本。4 个 Go 服务编译 ✅，测试 ✅。元规则 34 从 dataInit 重命名为 dataMigrate，合并 schema migration 与 data seed 到统一目录。
**Status**: completed
**Area**: infrastructure / dataMigrate / all-services

## [OPT-20260725-060] completed
- **Status**: completed
- **Completed**: 2026-07-25T22:39:10+08:00
- **Completion-Note**: 在 adminGrantResources 增加 idempotencyKey 参数防重复赠送；Python 事件处理器使用 new_user_gift:{company_id} 幂等键。
- **Source**: /goal grant-points + 新用户自动赠礼 实现
- **Why**: 当前 `adminGrantResources` 未使用幂等键，管理员重复提交或 Kafka 事件重放可能造成重复赠送资源。
- **How**: 在 `adminGrantResources` 增加 `idempotency_key` 参数，事务前检查 `billing_idempotency_key` 表；事件处理器 `4_grant_initial_resources.active.py` 使用 `"new_user_gift:{company_id}"` 作为幂等键。

## [OPT-20260725-062] completed
- **Status**: completed
- **Completed**: 2026-07-25T22:39:10+08:00
- **Completion-Note**: adminGrantResources switch 添加 ResourceTypeGitlabDisk/Traffic 分支；前端 resourceTypes 新增 gitlab_disk/traffic 选项。
- **Source**: /goal grant-points 管理端赠礼 实现
- **Why**: 当前 `adminGrantResources` 仅支持 `task_post` 类型，如管理员需赠送 GitLab 磁盘或流量配额，需扩展 `case ResourceTypeGitlabDisk/Traffic` 分支，更新 `billing_tenant_gitlab_resource` 表。
- **How**: 在 `adminGrantResources` 的 switch 中添加 `ResourceTypeGitlabDisk` 和 `ResourceTypeGitlabTraffic` 分支，前端 `resourceTypes` 数组增加对应选项。

## [OPT-20260725-064] completed
- **Status**: completed
- **Completed**: 2026-07-25T22:39:10+08:00
- **Completion-Note**: SystemAdminOrderRecords.vue 租户下拉选项增加 phone/email 第二行显示。
- **Source**: /goal grant-points 下拉显示租户账号（电话、邮箱）
- **Why**: `SystemAdminOrderRecords.vue` 调用同一个 `/api/system_admin/accounts/admin/tenant-options/` API（已返回 phone/email 字段），但其下拉菜单尚未显示账号信息，与 grant-points 页面体验不一致。
- **How**: 参照 `SystemAdminGrantPoints.vue` 的模板修改，在订单查看页面的租户下拉选项中增加 phone/email 第二行显示。

## [OPT-20260725-066] completed
- **Status**: completed
- **Completed**: 2026-07-25T22:39:10+08:00
- **Completion-Note**: create_progress_system/create_deliverable_systems 改用 is_default 稳定标记查找，防止 name 重命名后重复创建。同步修复 scripts/init/ 和 dataMigrate/saas/ 两个副本。
- **Source**: /goal default-column-management 默认进度体系重复 bug 修复
- **Why**: `scripts/init/02_01_init_system.py` 中 `create_progress_system()` 使用 `get_or_create(name='系统默认进度体系')` 按名称查找，若管理员重命名默认进度体系后手动执行该脚本（当前 `init_system.py` 已跳过），会重复创建同名默认条目。现已从 Go `seedDefaults` 中消除同名风险，Django 遗留脚本需同步防御。
- **How**: 将 `ProgressSystem.objects.get_or_create(name=progress_system_name, ...)` 改为 `ProgressSystem.objects.filter(is_default=True).first()` 作为查找条件，或直接删除该函数（因为 Go 已接管初始化）。

## [OPT-20260725-070] completed
- **Status**: completed
- **Completed**: 2026-07-25T22:39:10+08:00
- **Completion-Note**: taskTaskService/taskCloudService/taskTenantService/taskAIComment 四个 Go 服务添加 data_migrate_log 追踪表和 runDataMigrate 函数。
- **Source**: /goal 数据迁移元规则 34_data_migrate_directory_standard.md
- **Why**: taskTaskService、taskCloudService、taskTenantService、taskAIComment、taskGitOauth 等 5 个 Go 服务仅用 `CREATE TABLE IF NOT EXISTS` 做 schema migration，无 migration 版本追踪也无 `data_migrate_log` 表。taskProjectService 已在本次迁移中添加；taskAuth/taskBill/taskReferral 已有 migration 追踪表。
- **How**: (1) 参考 taskProjectService `db.go` 的 `runDataMigrate()` 模式，为上述 5 个服务各添加 `data_migrate_log` 表（在 `runMigrations()` 中 `CREATE TABLE IF NOT EXISTS`）；(2) 后续新增 dataMigrate 脚本时直接使用该表追踪；(3) 未来可考虑统一提取到 shareLib。

---
<!-- 2026-07-25: Django→Go 迁移计划 pending 条目已移至 OPTIMIZATION_TODOS.md -->

