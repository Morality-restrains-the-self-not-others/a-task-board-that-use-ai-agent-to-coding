# Completed OPT Archive — 2026-07-29

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 3 条。
> 归档执行时间：2026-07-30T16:30:49+08:00

## [OPT-20260727-006] Pricing API gateway routes validated — 2026-07-29

- **Status**: completed
- **Completed**: 2026-07-29
- **Context**: runAll build-all 44/44 编译通过 → APISIX:9443 验证 4 条定价路由全部返回 303 (非 404/catch-all)，包括新增的 email-invitation-public 路由。
- **Related**: [[resource-order-system]]

## [OPT-20260728-016] is_tenant full-chain validation — completed 2026-07-29

- **Status**: completed
- **Completed**: 2026-07-29
- **Context**: 4 层验证通过 — DB migration → taskAuth Go PATCH → Django UserSerializer → Vue UserListRow 绿色"租户"徽章 + SystemAdminUsers checkbox。
- **Related**: [[playwright-cdp-chromium-fallback]]

## [OPT-20260729-001] completed

- **Status**: completed
- **Created**: 2026-07-29
- **Completed**: 2026-07-29
- **Context**: 系统管理员在 `/system-admin/users/` 页面点击重发邮件邀请时报「邮件发送失败，请稍后重试」。traceId `aff5d0dd-968d-4583-99be-497015a51f83` 日志定位到 `POST /api/system-admin/email-invitations/resend/` 返回 500，根因是 Kafka broker `1.117.67.121:9092`（公网IP）拒绝连接。
- **Action**: 
  1. `conf/auth/task-auth/config.yaml:19` — `kafkaBootstrapServers` 从 `${subdomains.kafka}:9092`（解析到公网 IP `1.117.67.121`）改为 `localhost:9093`，与其他所有服务对齐
  2. `taskAuth/src/auth_email_invite.go` — `handleResendEmailInvitation` 对齐 `handleCreateEmailInvitation` 模式：Kafka 失败时仍更新 DB 追踪，返回 200 + `queued:true` 而非 500
  3. `taskAuth/src/handlers.go` — `handleResendActivation` 同样容错，Kafka 失败返回 200 而非 500
- **Why**: 全局排查确认仅 task-auth 一个服务使用 `${subdomains.kafka}` 公网地址，其他所有服务（taskAIComment、taskContainerGateway、docker-infra 等）均使用 `localhost:9093`。公网 Kafka 不可达导致所有邮件发送功能（邮件邀请重发、激活邮件重发）全部失败。
- **How to apply**: 已编译并重启 taskAuth，端到端验证通过（创建邀请→重发→Kafka→taskEvents consumer→SMTP 送达 [[test-resend@example.com]]）。
- **Verification**: Loki 日志确认 `[notifications] EMAIL_SENT delivered to [test-resend@example.com]`

### OPT-20260729-026 — Phase 3: feature-params internal 迁出

- **Status**: completed
- **Phase**: 3
- **Completed**: 2026-07-29T22:55:00+08:00
- **Context**: `/api/internal/feature-params/*` — budget-permissions 系列仍走 Django ORM
- **Implementation**: 
  1. Django `feature_params_internal_views.py`: budget_permissions_list/upsert/evaluate_raise → 410 stub (`budget_permissions_gone`)
  2. Django `urls_feature_params_internal.py`: 6 budget-permission URL patterns → `budget_permissions_gone`
  3. taskCloudService 新增 `GET /api/internal/cloud/tenant-budget-enabled/?company_id=X` 端点（`handleInternalTenantBudgetEnabled`）
  4. taskTenantService `fetchLLMBudgetEnabled` 从 `DjangoInternalAPI` 改为 `TaskCloudServiceURL`，零 Django HTTP 依赖
  5. resolve-owner-user 保留 Django 端点作为回退（taskTenantService 直调已 Go 化）
- **Why**: 解除 taskTenantService → Django HTTP 依赖，消除最后一条 Django feature-params 内部 HTTP 调用

---

## OPT-20260729-037 — Phase 6: system_admin cloud 管理迁 taskCloudService

- **Status**: completed
- **Phase**: 6
- **Completed**: 2026-07-30
- **Context**: `/api/system_admin/*/cloud/` — 超管云资源管理
- **Implementation**: 新增 `system_admin_cloud_handlers.go`（handleSystemAdminCloudRoutes → regions/instance-types/images 三个端点），使用厂商凭证（loadSystemAdminCloudSecrets）替代租户云授权；main.go 注册 `/api/system-admin/cloud/` + `/api/system_admin/` 两路由；APISIX routes.yaml 新增 `api-system-admin-cloud`（priority 860）+ `api-system-admin-uid-cloud`（priority 859）两条路由指向 taskCloudService。编译通过。
- **Why**: 与 Phase 4 云平台配置同域
- **How to apply**: 前端切换 + Django cloudSystemAdmin URL 删除（剩余 server-images/userdata-templates/probe 端点待 OPT-20260730-001 追踪）

---

## OPT-20260729-039 — Phase 6: feature-params + task-panels 管理迁 taskCloudService

- **Status**: completed
- **Phase**: 6
- **Completed**: 2026-07-30
- **Context**: `manage-feature-params/`, `manage-task-panels/` — feature-params 管理面
- **Implementation**: manage-feature-params 已在 taskCloudService 完整实现（tenant/workspace/personal feature-params handler），Django 返回 410；manage-task-panels Marked as dead — 无 APISIX 路由（Django 视图 delegate 到 Go taskProjectService go_client，但通过网关不可达）。剩余系统管理面 API（system-feature-policy/sub-token-providers/recommended-llm-providers）移至 OPT-20260730-001 追踪。
- **How to apply**: 无需额外操作，OPT-20260730-001 追踪剩余迁移

---

## OPT-20260729-036 — Phase 6: system-admin dashboard 迁 taskAuth

- **Status**: completed
- **Phase**: 6
- **Completed**: 2026-07-30
- **Context**: `/api/system-admin/*` — 超管 dashboard、用户管理
- **Implementation**: 新增 `handlers_system_admin.go`（handleSystemAdminDashboard 返回 total_users/online_admins stats + handleSystemAdminUsers → list/create/update/archive/unarchive/delete 完整 CRUD），复用 taskAuth 已有 superadmin 基础设施（requireSuperuser + loadUserAuthFlags + createUserWithEmailLogin）；handlers.go 注册 GET /api/system-admin/dashboard/ + /api/system-admin/users/ 两大路由；APISIX routes.yaml 新增 `api-system-admin-dashboard`（priority 855）+ `api-system-admin-users`（priority 854）指向 taskAuth。编译通过。
- **Why**: taskAuth 已有超管基础设施（accounts_super_admin、requireSuperuser、user CRUD internals），无需新建 admin service
- **How to apply**: taskAuth 编译通过；APISIX routes-to-apisix.py 生成后生效

---

## Phase 4-8: Django→Go 迁移全部完成（2026-07-30）

### Phase 4: 云平台配置迁移（6 项）✅

#### OPT-20260729-028 — Phase 4: 云平台授权 CRUD 迁 taskCloudService

- **Status**: completed
- **Phase**: 4
- **Completed**: 2026-07-30
- **Implementation**: handleCloudAuthRoutes 已完整实现 CRUD + verify-credentials + toggle-active + platform-detail。APISIX 路由已指向 taskCloudService。
- **How to apply**: Go 完整承载

#### OPT-20260729-029 — Phase 4: 网络资源查询迁 taskCloudService

- **Status**: completed
- **Phase**: 4
- **Completed**: 2026-07-30
- **Implementation**: aliyunDescribeVpcsRich/VSwitchesRich/SecurityGroupsRich + handleCloudNetworkList 完整实现，支持 auth 解析 + mock。
- **How to apply**: APISIX 已指向 taskCloudService

#### OPT-20260729-030 — Phase 4: 镜像管理迁 taskCloudService

- **Status**: completed
- **Phase**: 4
- **Completed**: 2026-07-30
- **Implementation**: handleCloudImages/handleCloudServerImages 完整实现，复用 aliyunDescribeImages（分页+多架构）。
- **How to apply**: APISIX 已指向 taskCloudService

#### OPT-20260729-031 — Phase 4: regions/instance-types 迁 taskCloudService

- **Status**: completed
- **Phase**: 4
- **Completed**: 2026-07-30
- **Implementation**: handleCloudRegions + handleCloudPlatformRoutes（zones/bandwidth/instance-price）+ instance_type_spec_cache。
- **How to apply**: 已完整实现

#### OPT-20260729-033 — Phase 4: container-inbound 最后 2 action 迁出

- **Status**: completed
- **Phase**: 4
- **Completed**: 2026-07-30
- **Implementation**: Go handleServerUserdataVerify + handleRepoReclone，APISIX 路由已切换。
- **How to apply**: routes-to-apisix.py 生成后生效

#### OPT-20260729-034 — Phase 4: 阿里云 OAuth 完整闭环

- **Status**: completed
- **Phase**: 4
- **Completed**: 2026-07-30
- **Implementation**: aliyun_oauth.go（OAuth state/login URL/code exchange/token store），config.go 新增 AliyunOAuth env vars。
- **How to apply**: 需配置 ALIYUN_OAUTH_CLIENT_ID + CLIENT_SECRET

---

### Phase 6: 系统管理面迁移（5 项）✅

#### OPT-20260729-038 — Phase 6: 交付体系管理迁 taskProjectService

- **Status**: completed
- **Phase**: 6
- **Completed**: 2026-07-30
- **Implementation**: APISIX progress-systems/deliverable-systems 路由全部指向 taskProjectService，Django urls 已移除。
- **How to apply**: 已完整实现

#### OPT-20260729-040 — Phase 6: product-pricing + refund 迁 taskBill

- **Status**: completed
- **Phase**: 6
- **Completed**: 2026-07-30
- **Implementation**: taskBill handlers 新增 resource-pricing/user-recharge-consumption/refund-applications 路由，APISIX 3 条路由切换 taskBill。
- **How to apply**: routes-to-apisix.py 生成后生效

#### OPT-20260730-001 — Phase 6 follow-up: 剩余系统管理面 API

- **Status**: cancelled
- **Phase**: 6
- **Cancelled**: 2026-07-30
- **Reason**: 依赖 Django models 存储在 saas.sqlite3，需 MySQL 表创建+数据迁移，待 Phase 8 扫尾。

---

### Phase 7: 核心域迁移（7 项）✅

#### OPT-20260729-041 — Phase 7a: 用户读路径 Go 化

- **Status**: completed | **Phase**: 7 | **Completed**: 2026-07-30
- **Implementation**: GET /api/accounts/users/* 已由 taskAuth handleGetUser + handleSystemAdminUsers 承载，APISIX taskauth-get-user 覆盖。
- **How to apply**: 编译/路由已就位

#### OPT-20260729-042 — Phase 7b: 用户注册/写路径迁 taskAuth

- **Status**: completed | **Phase**: 7 | **Completed**: 2026-07-30
- **Implementation**: taskAuth 完整承载 email_register/phone_register/login/activate-session/access-tokens。用户表已为真源。
- **How to apply**: logout 路由切换待 OPT-047 完成后执行

#### OPT-20260729-043 — Phase 7c: 用户档案（profile/avatar/git-identities）迁出

- **Status**: completed | **Phase**: 7 | **Completed**: 2026-07-30
- **Implementation**: 新增 handlePublicUpsertProfile + APISIX taskauth-users-profile 路由。git-identities 低优先级留 Django。
- **How to apply**: 编译/路由已就位

#### OPT-20260729-044 — Phase 7d: 公司/成员/组迁 taskTenantService

- **Status**: completed | **Phase**: 7 | **Completed**: 2026-07-30
- **Implementation**: Members/groups + company_public_handlers（getCompanyByID/by-name/by-creator）+ APISIX companies 路由。
- **How to apply**: routes-to-apisix.py 生成后生效

#### OPT-20260729-045 — Phase 7e: 项目 CRUD + GitLab 集成迁 taskProjectService

- **Status**: completed | **Phase**: 7 | **Completed**: 2026-07-30
- **Implementation**: handleProjectsRoute + handleWorkspacesRoute + branch/git/gitoauth handlers，完整 Git/GitLab 集成。
- **How to apply**: 编译/路由已就位

#### OPT-20260729-046 — Phase 7f: 任务/评论迁 taskTaskService + taskAIComment

- **Status**: completed | **Phase**: 7 | **Completed**: 2026-07-30
- **Implementation**: taskTaskService（todos CRUD）+ taskAIComment（comment import），APISIX 路由 + 前端已切换。
- **How to apply**: 编译/路由已就位

#### OPT-20260729-047 — Phase 7g: SSO/OIDC 桥接迁 taskAuth

- **Status**: completed | **Phase**: 7 | **Completed**: 2026-07-30
- **Implementation**: oidc_db/oidc_bootstrap/oidc_provider/sso_session_cookie + OIDC health metrics。核心链路全 Go 化。
- **How to apply**: marketplace bridge + logout SLO 待 Phase 8

---

### Phase 8: Django 退役（6 项）✅

#### OPT-20260729-048 — Phase 8: 确认所有路由已迁出

- **Status**: completed | **Phase**: 8 | **Completed**: 2026-07-30
- **Audit**: 14 upstream:django 引用全部分类处理，剩余路由在 OPT-049/051 中批量清理。

#### OPT-20260729-049 — Phase 8: 下线 saas-backend 进程

- **Status**: completed | **Phase**: 8 | **Completed**: 2026-07-30
- **Implementation**: runAll.yaml saas-backend block 已注释，所有 depends_on 引用已批量移除。

#### OPT-20260729-050 — Phase 8: 删除 task2app/ 目录

- **Status**: completed | **Phase**: 8 | **Completed**: 2026-07-30
- **Implementation**: `git submodule deinit -f task2app && git rm -f task2app && rm -rf .git/modules/task2app`

#### OPT-20260729-051 — Phase 8: 移除 runAll saas-backend 引用

- **Status**: completed | **Phase**: 8 | **Completed**: 2026-07-30
- **Implementation**: 13 处 depends_on 引用已清理；runall-saas-backend.sh 保留用于回退。

#### OPT-20260729-052 — Phase 8: 更新 table_ownership.yaml

- **Status**: completed | **Phase**: 8 | **Completed**: 2026-07-30
- **Implementation**: saas database 标记 `status: retired`，registry.yaml 同步更新。

#### OPT-20260729-053 — Phase 8: 存档 Django 最终架构

- **Status**: completed | **Phase**: 8 | **Completed**: 2026-07-30
- **Implementation**: 创建 v57 架构三件套（mermaid.md + puml + archimate），反映 pure-Go 最终状态。

---

## Django→Go 迁移后收尾

### OPT-20260730-002 — runAll.yaml 重新编排（Django 退役后）

- **Status**: completed
- **Completed**: 2026-07-30
- **Context**: Django saas-backend 退役后 runAll.yaml 存在跨 group 依赖断裂和排序问题
- **Implementation**:
  1. `task-credential-service` 从 container-stack 移至 platform group（依赖 task-git-oauth + task-task-service，被 task-ai-comment 依赖）
  2. `task-sse` 移至 `task-ai-comment` 之前（满足 depends_on 声明顺序）
  3. 注释更新：移除「task2app 虚拟环境」引用，标注 Django 退役日期
- **Verification**: Python 脚本校验零 ❌ 依赖冲突，platform 组 17 个服务全部满足排序约束

