# 系统测试意图目录 (System Test Intent Catalog)

> 生成日期：2026-08-01  
> 用途：记录系统所有"设计意图"对应的测试定义，用于变迁后回归验证  
> 运行方式：`bash scripts/run-all-intent-tests.sh` 或 `/intent-test` 技能

## 统计概览

| 维度 | 数量 |
|------|------|
| 意图定义文件（`.intent.md`） | ~97 |
| 有测试定义的意图（`.test-intent.md`） | ~89 |
| 缺测试定义的意图 | ~8 |
| 测试意图伴随文档（`.testIntent`） | ~40 |
| Go 单元测试文件 | ~666 |
| JS/TS 测试文件 | ~547 |
| 覆盖的服务 | 20 |

---

## 一、后端意图 (Backend Intents)

### 1.1 基础设施与部署

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-001 | 全量应用服务监听 0.0.0.0 | [all_services_bind_0.0.0.0](backend/all_services_bind_0.0.0.0.intent.md) | ✅ [test](backend/all_services_bind_0.0.0.0.test-intent.md) | `ss -tlnp` 端口检查 + runAll 健康检查 |
| B-002 | 边缘 Nginx loopback 绑定 | [edge_nginx_loopback_bind](backend/edge_nginx_loopback_bind.intent.md) | ✅ [test](backend/edge_nginx_loopback_bind.test-intent.md) | Nginx 配置审计 |
| B-003 | 应用启动禁用环境 Proxy | [.ai.md 一级规则](../../.ai.md) | — | runAll use_proxy=false 检查 |
| B-004 | 网络拓扑感知配置 | [.ai.md 一级规则](../../.ai.md) | — | conf/base.yaml 模板变量审计 |
| B-005 | 数据库初始化注册强制 | [.ai.md 一级规则](../../.ai.md) | — | db/registry.yaml 覆盖检查 |
| B-006 | 公网 taskFE nginx 静态常驻 | [taskfe_nginx_static_resident](backend/taskfe_nginx_static_resident.intent.md) | ✅ [test](backend/taskfe_nginx_static_resident.test-intent.md) | `ss` :4000=nginx；构建中 SPA 仍 200；hashed 缺失 404 |

### 1.2 认证与授权 (Auth & OAuth)

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-010 | gitOauth 迁 Go（taskGitOauth） | [gitoauth_go_migration](backend/gitoauth_go_migration.intent.md) | ✅ [test](backend/gitoauth_go_migration.test.intent.md) | `taskGitOauth/src/*_test.go` (14 files) |
| B-011 | GitLab SSO 跨子域登录 | [gitlab_taskauth_sso_cross_subdomain_login](backend/gitlab_taskauth_sso_cross_subdomain_login.intent.md) | ✅ [test](backend/gitlab_taskauth_sso_cross_subdomain_login.test-intent.md) | `taskAuth/*_test.go` (136 files) + OIDC Playwright 测试 |
| B-012 | SSO AI Provider admin 已登录不跳转登录页 | [sso_ai_provider_admin_logged_in_no_login_redirect](backend/sso_ai_provider_admin_logged_in_no_login_redirect.intent.md) | ✅ [test](backend/sso_ai_provider_admin_logged_in_no_login_redirect.test-intent.md) | `taskAiProvider/*_test.go` (7 files) |
| B-013 | Token 初始化路径作用域 | [token_init_path_scope](backend/token_init_path_scope.intent.md) | ✅ [test](backend/token_init_path_scope.test-intent.md) | `taskCredentialService/interfaces/token_init_path_test.go` |
| B-014 | Feature Params 访问控制 | [feature_params_access_control](backend/feature_params_access_control.intent.md) | ✅ [test](backend/feature_params_access_control.test-intent.md) | `taskTenantService/*_test.go` (7 files) |
| B-015 | Feature Params Django→Go 迁移 | [feature_params_django_to_go](backend/feature_params_django_to_go.intent.md) | ✅ [test](backend/feature_params_django_to_go.test-intent.md) | `taskTenantService/*_test.go` |
| B-016 | Feature Params Proxy Rewrite (Go) | [feature_params_proxy_rewrite_go](backend/feature_params_proxy_rewrite_go.intent.md) | ❌ 缺测试定义 | — |
| B-017 | People/Member/Group Go 迁移 | [people_member_group_go_migration](backend/people_member_group_go_migration.intent.md) | ✅ [test](backend/people_member_group_go_migration.test-intent.md) | `taskTenantService/*_test.go` |
| B-017b | 开放式邀请链接 | [open_invite_link](backend/open_invite_link.intent.md) | ✅ [test](backend/open_invite_link.test-intent.md) | `taskTenantService/src/invite_open_test.go` |
| B-017c | 租户邀请有效期最长 365 天 | [people_invite_expiration](backend/people_invite_expiration.intent.md) | ✅ [test](backend/people_invite_expiration.test-intent.md) | `invite_expiration_test.go` + `invite_handlers_test.go` |
| B-018 | 手机/邮箱占用仅计活跃用户 | [auth_phone_occupancy_excludes_archived](backend/auth_phone_occupancy_excludes_archived.intent.md) | ✅ [test](backend/auth_phone_occupancy_excludes_archived.test-intent.md) | `taskAuth/src/auth_phone_register_test.go`；`auth_login_method_lookup_test.go`；`handlers_system_admin_list_filter_test.go` |
| B-019 | 资料页手机绑定真实写入且占用可转移 | [auth_profile_phone_bind](backend/auth_profile_phone_bind.intent.md) | ✅ [test](backend/auth_profile_phone_bind.test-intent.md) | `taskAuth/src/auth_phone_bind_test.go`；`handlers_test.go`；`phoneBindingApi.test.js` |
| B-019e | 超级用户/员工/测试一号最多 5 绑定 | [auth_privileged_phone_multi_bind](backend/auth_privileged_phone_multi_bind.intent.md) | ✅ [test](backend/auth_privileged_phone_multi_bind.test-intent.md) | `taskAuth/domain/phone_share_test.go`；`taskAuth/src/auth_phone_bind_test.go`；`auth_phone_share_login_test.go`；`phoneBindingApi.test.js` |
| B-019b | 成功登录写入登录历史并可按身份查询 | [auth_login_history](backend/auth_login_history.intent.md) | ✅ [test](backend/auth_login_history.test-intent.md) | `taskAuth/src/login_history_handlers_test.go`；`taskAuth/domain/login_history_test.go` |
| B-019c | 服务号关注用临时 ID 对账并绑定 mp openId | [referral_mp_follow_bind](backend/referral_mp_follow_bind.intent.md) | ✅ [test](backend/referral_mp_follow_bind.test-intent.md) | `taskAuth/src/auth_wechat_mp_test.go`；`taskAuth/domain/wechat_mp_subscribe_test.go` |
| B-019d | 申请推荐资格须已绑定服务号 | [referral_apply_requires_mp](backend/referral_apply_requires_mp.intent.md) | ✅ [test](backend/referral_apply_requires_mp.test-intent.md) | `taskReferral/src/referral_mp_bind_gate_test.go` |

### 1.3 容器与任务执行

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-019 | 评论级容器令牌 | [comment-scoped-container-token](comment-scoped-container-token.intent.md) | ✅ [test](comment-scoped-container-token.test.intent.md) | `taskCredentialService/token_comment_scope_test.go`；`saasInboundScope*.test.mjs`；`saasTaskCloud.layerGraphPush.test.mjs`；`autoRunPrBackfill.test.mjs` |
| B-020 | 容器执行绕过 Django | [container_exec_bypass_django](backend/container_exec_bypass_django.intent.md) | ✅ [test](backend/container_exec_bypass_django.test-intent.md) | `taskContainerGateway/src/*_test.go` (16 files) |
| B-021 | 容器层 Git 合并 | [container_layer_git_merge](backend/container_layer_git_merge.intent.md) | ✅ [test](backend/container_layer_git_merge.test-intent.md) | `taskContainerGateway/src/l0_layer_graph_test.go` |
| B-022 | 嵌套 Git Repo 克隆 | [container_nested_git_repos_clone](backend/container_nested_git_repos_clone.intent.md) | ✅ [test](backend/container_nested_git_repos_clone.test-intent.md) | `taskContainerGateway/*_test.go` + `taskCredentialService/application/nested_repos_enrich_test.go` |
| B-023 | 容器任务 API 经 Gateway | [container_task_api_via_gateway](backend/container_task_api_via_gateway.intent.md) | ✅ [test](backend/container_task_api_via_gateway.test-intent.md) | `taskContainerGateway/src/handlers_test.go` |
| B-024 | Relay 启动 Session Gateway 直写 | [relay_startup_session_gateway_direct_write](backend/relay_startup_session_gateway_direct_write.intent.md) | ✅ [test](backend/relay_startup_session_gateway_direct_write.test-intent.md) | `taskContainerGateway/src/relay_lifecycle_test.go` |
| B-025 | Relay 状态推送到 Cloud 收敛 | [relay_status_push_cloud_converge](backend/relay_status_push_cloud_converge.intent.md) | ✅ [test](backend/relay_status_push_cloud_converge.test-intent.md) | `taskContainerGateway/src/relay_status_sse_test.go` |
| B-026 | Autorun skip on git inaccessible | [autorun_skip_on_git_inaccessible](backend/autorun_skip_on_git_inaccessible.intent.md) | ✅ [test](backend/autorun_skip_on_git_inaccessible.test-intent.md) | `taskContainerGateway/*_test.go` |
| B-027 | OnlineServiceJS bootstrap transient retry | [onlineServiceJS_bootstrap_transient_retry](backend/onlineServiceJS_bootstrap_transient_retry.intent.md) | ✅ [test](backend/onlineServiceJS_bootstrap_transient_retry.test-intent.md) | `go_relayToTrae/*_test.go` (15 files) |

### 1.4 云计算资源

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-030 | AI Endpoint Budget 直连云 | [ai_endpoint_budget_direct_cloud](backend/ai_endpoint_budget_direct_cloud.intent.md) | ✅ [test](backend/ai_endpoint_budget_direct_cloud.test-intent.md) | `taskAIEndPoint/*_test.go` (3 files) |
| B-031 | AI Endpoint Django Thin→Go | [ai_endpoint_django_thin_to_go](backend/ai_endpoint_django_thin_to_go.intent.md) | ✅ [test](backend/ai_endpoint_django_thin_to_go.test-intent.md) | `taskAIEndPoint/*_test.go` |
| B-032 | AI Provider OIDC Issuer Gateway | [ai_provider_oidc_issuer_gateway](backend/ai_provider_oidc_issuer_gateway.intent.md) | ✅ [test](backend/ai_provider_oidc_issuer_gateway.test-intent.md) | `taskAiProvider/*_test.go` (7 files) |
| B-033 | Cloud Domain Tables 迁移 | [cloud_domain_tables_task_cloud_migration](backend/cloud_domain_tables_task_cloud_migration.intent.md) | ❌ 缺测试定义 | — |
| B-034 | ECS Orphan Double Start Guard | [ecs_orphan_double_start_guard](backend/cloud/ecs_orphan_double_start_guard.intent.md) | ❌ 缺测试定义 | — |
| B-035 | Idle Reuse Boot Guard + Orphan Cross Check（复用已废弃 ADR-0013；孤儿交叉校验仍有效） | [idle_reuse_boot_guard_orphan_cross_check](backend/cloud/idle_reuse_boot_guard_orphan_cross_check.intent.md) | ✅ [test](backend/cloud/idle_reuse_boot_guard_orphan_cross_check.test-intent.md) | `taskCloudService/*_test.go` |
| B-036 | Idle Reuse Same Workspace User（已废弃 ADR-0013） | [idle_reuse_same_workspace_user](backend/cloud/idle_reuse_same_workspace_user.intent.md) | ✅ [test](backend/cloud/idle_reuse_same_workspace_user.test.intent.md) | `taskCloudService/src/compute_start_vm_idle_reuse_test.go` |
| B-037 | Comment Container Bindings | [comment_container_bindings](backend/cloud/comment_container_bindings.intent.md) | ✅ [test](backend/cloud/comment_container_bindings.test.intent.md) | `taskAIComment/*_test.go` (8 files) |
| B-038 | 容器转发目标解析评论级 CSC | [container_target_comment_csc](backend/cloud/container_target_comment_csc.intent.md) | ✅ [test](backend/cloud/container_target_comment_csc.test-intent.md) | `taskCloudService/src/container_target_comment_csc_test.go`；`taskContainerGateway/src/handlers_comment_id_forward_test.go`；`taskFE` `containerComputeRequest.test.js` / `containerForwardCommentId.scan.test.js` |
| B-039 | 评论容器启动日志按 workspace 哈希分表 | [ccb_logs_workspace_shard](backend/task-cloud/001_ccb_logs_workspace_shard.intent.md) | ✅ [test](backend/task-cloud/001_ccb_logs_workspace_shard.test-intent.md) | `taskCloudService/src/ccb_log_shard_test.go`；`ccb_log_store_test.go`；`TestCommentContainerBindingLogsTimeline` |
| B-057 | ztree 执行日志服务端持久化 | [ztree_exec_log_server_persist](backend/cloud/ztree_exec_log_server_persist.intent.md) | ✅ [test](backend/cloud/ztree_exec_log_server_persist.test-intent.md) | `taskCloudService/src/layer_graph_snapshot_handlers_test.go`；`taskCloudService/domain/layer_graph_snapshot_test.go`；`taskFE/.../taskDetailContainerFns.commentId.test.js`；**不**存克隆日志 |
| B-058 | ztree 层级落库 + step_full COS 归档 | [ztree_step_full_cos_archive](backend/cloud/ztree_step_full_cos_archive.intent.md) | ✅ [test](backend/cloud/ztree_step_full_cos_archive.test-intent.md) | `taskCloudService/domain/step_full_*_test.go`；`taskCloudService/src/step_full_*_test.go`；`trae-agent/onlineServiceJS/src/saasStepFullArchive.test.mjs` |
| B-059 | 评论启动日志 COS 归档 | [ccb_startup_logs_cos_archive](backend/cloud/ccb_startup_logs_cos_archive.intent.md) | ✅ [test](backend/cloud/ccb_startup_logs_cos_archive.test-intent.md) | `taskCloudService/domain/startup_log_*_test.go`；`taskCloudService/src/ccb_log_cos_test.go`；`taskCloudService/src/step_full_admin_test.go`；`taskFE/.../SystemAdminStepFullCOS.test.js` |

### 1.5 预算与计费

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-040 | Budget Console Cloud Admin API | [budget_console_cloud_admin_api](backend/budget_console_cloud_admin_api.intent.md) | ✅ [test](backend/budget_console_cloud_admin_api.test-intent.md) | `taskBill/*_test.go` (17 files) |
| B-041 | Budget Console User API Cloud | [budget_console_user_api_cloud](backend/budget_console_user_api_cloud.intent.md) | ❌ 缺测试定义 | `taskBill/*_test.go` |
| B-042 | Budget Record Usage Batch Thin Removed | [budget_record_usage_batch_thin_removed](backend/budget_record_usage_batch_thin_removed.intent.md) | ❌ 缺测试定义 | `taskBill/*_test.go` |
| B-043 | SaaS Budget Ledger HTTP Cutover | [saas_budget_ledger_http_cutover](backend/saas_budget_ledger_http_cutover.intent.md) | ❌ 缺测试定义 | `taskBill/*_test.go` |
| B-044 | Tenant GitLab Resource Purchase | [tenant_gitlab_resource_purchase](backend/tenant_gitlab_resource_purchase.intent.md) | ✅ [test](backend/tenant_gitlab_resource_purchase.test-intent.md) | `taskBill/*_test.go` |
| B-045 | Tenant GitLab OAuth Connection | [tenant_gitlab_oauth_connection](backend/tenant_gitlab_oauth_connection.intent.md) | ✅ [test](backend/tenant_gitlab_oauth_connection.test-intent.md) | `taskGitOauth/*_test.go` |
| B-046 | Tenant Budget Permission Cloud | [tenant_budget_permission_cloud](backend/tenant_budget_permission_cloud.intent.md) | ❌ 缺测试定义 | — |
| B-047 | GitLab 流量定价 | [gitlab-traffic-pricing](backend/gitlab-traffic-pricing.intent.md) | ✅ [test](backend/gitlab-traffic-pricing.test-intent.md) | `taskBill/*_test.go` |
| B-048 | GitLab 磁盘定价 | [gitlab-disk-pricing](../gitlab-disk-pricing.intent.md) | ✅ [test](../gitlab-disk-pricing.test-intent.md) | 定价模型验证 |
| B-049 | 可插拔多区域 gitService | [pluggable_multi_region_gitservice](backend/pluggable_multi_region_gitservice.intent.md) | ✅ [test](backend/pluggable_multi_region_gitservice.test-intent.md) | `taskBill/src/gitlab_region_routing_test.go` + `taskFE/.../useGitlabResourcePurchase.test.js` |
| B-049a | 管理端赠送 GitLab 须指定区域 | [admin_grant_gitlab_region](backend/admin_grant_gitlab_region.intent.md) | ✅ [test](backend/admin_grant_gitlab_region.test.intent.md) | `taskBill/src/admin_grant_region_test.go` + `taskFE/.../SystemAdminGrantPoints.region.unit.test.js` |
| B-049b | 租户购买 GitLab 须按行指定区域 | [tenant_purchase_gitlab_region](backend/tenant_purchase_gitlab_region.intent.md) | ✅ [test](backend/tenant_purchase_gitlab_region.test.intent.md) | `taskBill/src/orders_gitlab_region_test.go` + `OrderCreate.contract.test.js` |
| B-049j | 购买 GitLab 可选阿里云并由人工建节点开通 | [aliyun_gitlab_region_manual_node](backend/aliyun_gitlab_region_manual_node.intent.md) | ✅ [test](backend/aliyun_gitlab_region_manual_node.test-intent.md) | `taskBill/src/gitlab_region_aliyun_catalog_test.go` + `OrderCreate.contract.test.js` |
| B-049k | GitLab 磁盘购买须超管手动开通 | [gitlab_disk_manual_admin_fulfillment](backend/gitlab_disk_manual_admin_fulfillment.intent.md) | ✅ [test](backend/gitlab_disk_manual_admin_fulfillment.test-intent.md) | 待实现：`taskBill` pending_admin 禁止 GET ensure + provision 事件 |
| B-049c | 管理端赠送页可修改租户 VIP 等级 | [admin_grant_membership_tier](backend/admin_grant_membership_tier.intent.md) | ✅ [test](backend/admin_grant_membership_tier.test.intent.md) | `taskBill/src/admin_grant_membership_test.go` + `taskFE/.../SystemAdminGrantPoints.membership.unit.test.js` |
| B-049d | 资源订单号嵌入租户基因 | [billing_order_id_tenant_shard](backend/billing_order_id_tenant_shard.intent.md) | ✅ [test](backend/billing_order_id_tenant_shard.test-intent.md) | `taskBill/src/orders_number_test.go` + `orders_duplicate_test.go` + `handlers_orders_idor_test.go` |
| B-049e | 资源订单展示号全局唯一 | [billing_resource_order_number](backend/billing_resource_order_number.intent.md) | ✅ [test](backend/billing_resource_order_number.test-intent.md) | `taskBill/src/orders_number_test.go` + `orders_duplicate_test.go` |
| B-049f | 管理端按交易单号查询订单 | [billing_admin_order_trade_no_query](backend/billing_admin_order_trade_no_query.intent.md) | ✅ [test](backend/billing_admin_order_trade_no_query.test-intent.md) | `taskBill/src/orders_list_trade_no_test.go` |
| B-049g | GitLab 出站流量配额强制执行 | [gitlab_traffic_quota_enforcement](backend/gitlab_traffic_quota_enforcement.intent.md) | ✅ [test](backend/gitlab_traffic_quota_enforcement.test-intent.md) | `taskBill/src/gitlab_traffic_gate_test.go` + `gitService/scripts/test_zzz_traffic_quota_initializer.sh` |
| B-049h | 超管按推荐人查被推荐人打标订单分账结果 | [billing_admin_referrer_profit_sharing_query](backend/billing_admin_referrer_profit_sharing_query.intent.md) | ✅ [test](backend/billing_admin_referrer_profit_sharing_query.test-intent.md) | `taskBill` 列表 JOIN + refresh-wechat QueryOrder 单测 |
| B-049i | 完成超 25 天未成功分账由小时扫描兜底申请 | [billing_profit_sharing_25d_fallback](backend/billing_profit_sharing_25d_fallback.intent.md) | ✅ [test](backend/billing_profit_sharing_25d_fallback.test-intent.md) | `taskBill/src/wechat_profit_sharing_fallback_test.go` |
| B-056 | 交易列表资源流水账字段 | [billing_transactions_resource_ledger](backend/billing_transactions_resource_ledger.intent.md) | ✅ [test](backend/billing_transactions_resource_ledger.test-intent.md) | `taskBill/src/transaction_change_enrich_test.go` + `transaction_ledger_snapshot_test.go` |

### 1.6 任务系统

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-050 | 创建任务默认排序置顶 | [create_task_default_top_order](backend/create_task_default_top_order.intent.md) | ✅ [test](backend/create_task_default_top_order.test-intent.md) | `taskTaskService/*_test.go` (20 files) |
| B-051 | 任务评论 P0/P2 加固 | [task_comments_p0_p2_hardening](backend/task_comments_p0_p2_hardening.intent.md) | ✅ [test](backend/task_comments_p0_p2_hardening.test-intent.md) | `taskAIComment/*_test.go` (8 files) |
| B-052 | Schedule Rhythm Auto Close | [schedule_rhythm_auto_close](backend/schedule_rhythm_auto_close.intent.md) | ✅ [test](backend/schedule_rhythm_auto_close.test-intent.md) | `taskTaskService/*_test.go` |
| B-053 | Task Referral | [task-referral](backend/task-referral.intent.md) | ✅ [test](backend/task-referral.test-intent.md) | `taskReferral/src/*_test.go` |
| B-054 | Task Subtree Terminal Gate | [task_subtree_terminal_gate](backend/task_subtree_terminal_gate.intent.md) | ✅ [test](backend/task_subtree_terminal_gate.test-intent.md) | `taskContainerGateway/src/l0_layer_graph_test.go` |
| B-055 | Top Deliverable Queued Auto Run | [top_deliverable_queued_auto_run_schedule](backend/top_deliverable_queued_auto_run_schedule.intent.md) | ❌ 缺测试定义 | — |
| B-081 | 工作空间排队调度历史 | [queue_schedule_history](backend/queue_schedule_history.intent.md) | ✅ [test](backend/queue_schedule_history.test-intent.md) | `taskTaskService/src/queued_schedule_history_test.go` |
| B-085 | 任务与项目主要属性历史版本 | [task_project_entity_revisions](backend/task_project_entity_revisions.intent.md) | ✅ [test](backend/task_project_entity_revisions.test-intent.md) | `taskTaskService/src/task_revision_*_test.go`；`taskProjectService/src/project_revision_*_test.go` |
| B-086 | 任务/项目数据 GET 与 GitLab 探活分离 | [task_detail_get_fast_path](backend/task_detail_get_fast_path.intent.md) | ✅ [test](backend/task_detail_get_fast_path.test-intent.md) | GET project/task 去同步探活；独立 `POST validate-git-repos`（待落地） |

### 1.7 事件与数据流

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-060 | Work Panel 任务状态 SSE | [work_panel_task_status_sse](backend/work_panel_task_status_sse.intent.md) | ✅ [test](backend/work_panel_task_status_sse.test-intent.md) | `taskEvents/*_test.go` (59 files) + `taskSSE/*` |
| B-061 | Userdata Boot Progress SSE | [userdata_boot_progress_sse](backend/userdata_boot_progress_sse.intent.md) | ✅ [test](backend/userdata_boot_progress_sse.test-intent.md) | `taskSSE/*` |
| B-062 | Task Events SaaS HTTP Cutover | [task_events_saas_http_cutover](backend/task_events_saas_http_cutover.intent.md) | ❌ 缺测试定义 | `taskEvents/*_test.go` |
| B-063 | Task Cloud SaaS SQLite HTTP Cutover | [task_cloud_saas_sqlite_http_cutover](backend/task_cloud_saas_sqlite_http_cutover.intent.md) | ❌ 缺测试定义 | — |
| B-064 | Rewrite Sub Token Python Removed | [rewrite_sub_token_python_removed](backend/rewrite_sub_token_python_removed.intent.md) | ❌ 缺测试定义 | — |

### 1.8 多仓库与 Git

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-070 | Multirepo Prefer Remote GitLab Push OAuth | [multirepo_prefer_remote_gitlab_push_oauth](backend/multirepo_prefer_remote_gitlab_push_oauth.intent.md) | ✅ [test](backend/multirepo_prefer_remote_gitlab_push_oauth.test-intent.md) | `taskCredentialService/application/layer_oauth_test.go` |
| B-071 | High Traffic Hardening | [high_traffic_hardening](backend/high_traffic_hardening.intent.md) | ✅ [test](backend/high_traffic_hardening.test-intent.md) | 多个服务压力测试 |
| B-072 | 成员加入自动创建 Git 身份 | [member_joined_auto_git_identity](backend/member_joined_auto_git_identity.intent.md) | ✅ [test](backend/member_joined_auto_git_identity.test.intent.md) | `taskTaskService/src/git_identity_*_test.go` + `taskEvents/.../memberjoined` |

### 1.9 Vendor/市场

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| B-080 | Vendor Marketplace Unavailable Reason | [vendor_marketplace_unavailable_reason](backend/vendor_marketplace_unavailable_reason.intent.md) | ✅ [test](backend/vendor_marketplace_unavailable_reason.test-intent.md) | `taskAiProvider/*_test.go` |
| B-083 | 厂商撤回/下架/删除容器镜像版本 | [vendor_container_image_lifecycle](backend/vendor_container_image_lifecycle.intent.md) | ✅ [test](backend/vendor_container_image_lifecycle.test-intent.md) | `taskAiProvider/domain/entities_test.go` + `src/vendor_container_lifecycle_test.go` |

---

## 二、前端意图 (Frontend Intents)

### 2.1 任务详情页

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-001 | Task Detail Fork Auto Run Confirm | [task_detail_fork_auto_run_confirm](frontend/task_detail_fork_auto_run_confirm.intent.md) | ✅ [test](frontend/task_detail_fork_auto_run_confirm.test-intent.md) | `taskFE/tests/TaskDetail.*.playwright.test.js` |
| F-002 | Task Detail Layer Changes Scan Cap | [task_detail_layer_changes_scan_cap](frontend/task_detail_layer_changes_scan_cap.intent.md) | ✅ [test](frontend/task_detail_layer_changes_scan_cap.test-intent.md) | `taskFE/tests/TaskDetail.*.playwright.test.js` |
| F-003 | Task Detail Layer Changes Scroll Load More | [task_detail_layer_changes_scroll_load_more](frontend/task_detail_layer_changes_scroll_load_more.intent.md) | ✅ [test](frontend/task_detail_layer_changes_scroll_load_more.test-intent.md) | `taskFE/tests/TaskDetail.*.playwright.test.js` |
| F-004 | Task Detail Nested Repos Clone Status | [task_detail_nested_repos_clone_status](frontend/task_detail_nested_repos_clone_status.intent.md) | ✅ [test](frontend/task_detail_nested_repos_clone_status.test-intent.md) | `taskFE/tests/TaskDetail.*.playwright.test.js` |
| F-005 | Task Detail Resizable Split Pane | [task_detail_resizable_split_pane](frontend/task_detail_resizable_split_pane.intent.md) | ✅ [test](frontend/task_detail_resizable_split_pane.test-intent.md) | `taskFE/tests/TaskDetail.*.playwright.test.js` |
| F-006 | 容器页确保安全组 Ingress（013） | [013_open_container_page_ensure_sg_ingress](frontend/task_detail/013_open_container_page_ensure_sg_ingress.intent.md) | ✅ [test](frontend/task_detail/013_open_container_page_ensure_sg_ingress.test-intent.md) | `taskFE/tests/TaskDetail.start-vm-auto-*.playwright.test.js` |
| F-007 | 机器所有者提示（025） | [025_machine_owner_hint](frontend/task_detail/025_machine_owner_hint.intent.md) | ✅ [test](frontend/task_detail/025_machine_owner_hint.test-intent.md) | `taskFE/tests/TaskDetail.start-vm-auto-*.playwright.test.js` |
| F-008 | Fork 后新任务落在进度第一列（041） | [041_fork_first_progress_column](frontend/task_detail/041_fork_first_progress_column.intent.md) | ✅ [test](frontend/task_detail/041_fork_first_progress_column.test-intent.md) | `taskFE/app/src/composables/taskDetail/taskDetailEditing.test.js` / `taskFE/tests/TaskDetail.fork-popup.playwright.test.js` |
| F-117 | 代理步骤命令放到折叠区外（042） | [042_agent_step_command_outside_fold](frontend/task_detail/042_agent_step_command_outside_fold.intent.md) | ✅ [test](frontend/task_detail/042_agent_step_command_outside_fold.test-intent.md) | `TaskDetailAgentStepCard.commandOutsideFold.test.js` |

### 2.2 工作面板

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-010 | 创建任务编程语言 code_lang | [012_创建任务主要编程语言code_lang](frontend/work_panel/012_创建任务主要编程语言code_lang.intent.md) | ✅ [test](frontend/work_panel/012_创建任务主要编程语言code_lang.test-intent.md) | `taskFE/tests/CreateProject.*.playwright.test.js` |
| F-011 | Deliverable Section Per Column Counts | [013_deliverable_section_per_column_counts](frontend/work_panel/013_deliverable_section_per_column_counts.intent.md) | ✅ [test](frontend/work_panel/013_deliverable_section_per_column_counts.test-intent.md) | `taskFE/tests/*.playwright.test.js` |
| F-012 | Access Filter Person Member Name | [014_access_filter_person_member_name](frontend/work_panel/014_access_filter_person_member_name.intent.md) | ✅ [test](frontend/work_panel/014_access_filter_person_member_name.test-intent.md) | `taskFE/tests/WorkspaceAccess.*.playwright.test.js` |
| F-013 | Work Panel Task Card Creator Comments | [work_panel_task_card_creator_comments](frontend/work_panel_task_card_creator_comments.intent.md) | ✅ [test](frontend/work_panel_task_card_creator_comments.test-intent.md) | `taskFE/tests/TaskDetail.*.playwright.test.js` |
| F-014 | Work Panel Task Status SSE | [work_panel_task_status_sse](frontend/work_panel_task_status_sse.intent.md) | ✅ [test](frontend/work_panel_task_status_sse.test-intent.md) | `taskFE/tests/*.playwright.test.js` |
| F-015 | 工作面板顶栏合并为单行 | [work_panel_header_single_toolbar](frontend/work_panel/work_panel_header_single_toolbar.intent.md) | ✅ [test](frontend/work_panel/work_panel_header_single_toolbar.test-intent.md) | `WorkPanelHeader.legend.test.js` + `WorkPanelHeader.traceId.test.js` + `WorkPanel.auto-schedule-link.playwright.test.js` |

### 2.3 创建任务/项目

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-020 | Create Task Kind Options | [create_task_kind_options](frontend/create_task_kind_options.intent.md) | ✅ [test](frontend/create_task_kind_options.test-intent.md) | `taskFE/tests/CreateProject.*.playwright.test.js` |
| F-021 | Create Task Preferred Merge Target | [create_task_preferred_merge_target](frontend/create_task_preferred_merge_target.intent.md) | ✅ [test](frontend/create_task_preferred_merge_target.test-intent.md) | `taskFE/tests/CreateProject.*.playwright.test.js` |
| F-022 | Create Task Structured Fields | [create_task_structured_fields](frontend/create_task_structured_fields.intent.md) | ✅ [test](frontend/create_task_structured_fields.test-intent.md) | `taskFE/tests/CreateProject.form-validation.playwright.test.js` |
| F-023 | Git Repo Clone Alias | [git_repo_clone_alias](frontend/git_repo_clone_alias.intent.md) | ✅ [test](frontend/git_repo_clone_alias.test-intent.md) | `taskFE/tests/CreateProject.clone-alias.playwright.test.js` |
| F-024 | Branch Strategy Merged Inputs | [branch_strategy_merged_inputs](frontend/branch_strategy_merged_inputs.intent.md) | ✅ [test](frontend/branch_strategy_merged_inputs.test-intent.md) | `taskFE/tests/*.playwright.test.js` |
| F-025 | Common Merge Target Branches | [common_merge_target_branches](frontend/common_merge_target_branches.intent.md) | ✅ [test](frontend/common_merge_target_branches.test-intent.md) | `taskFE/tests/*.playwright.test.js` |
| F-026 | 创建任务时完成 Git OAuth 绑定 | [create_task_oauth_bind](frontend/create_task_oauth_bind.intent.md) | ✅ [test](frontend/create_task_oauth_bind.test-intent.md) | `createTaskOauthGate.test.js` + Path A T8b `TestBuildCredentialsPathAIPGitLabYAMLMissStillFetches` |
| F-027 | 创建项目支持 ssh:// Git URL | [create_project_ssh_git_url](frontend/create_project_ssh_git_url.intent.md) | ✅ [test](frontend/create_project_ssh_git_url.test-intent.md) | `gitRepoUrlUtils.test.js` + `CreateProject.form-validation.playwright.test.js` + `git_repo_url_normalize_test.go` |

### 2.4 评论系统

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-030 | Comment Execution Details | [comment_execution_details](frontend/comment_execution_details.intent.md) | ✅ [test](frontend/comment_execution_details.test.intent.md) | `TaskDetailCommentExecutionDetails.test.js` + `commentExecutionGitIdentity.test.js` + `taskFE/tests/*.playwright.test.js` |
| F-031 | Comment Runtime 无全局 header | [comment_runtime_server_tabs](frontend/comment_runtime_server_tabs.intent.md) | ✅ [test](frontend/comment_runtime_server_tabs.test.intent.md) | `TaskDetailCommentsPanel.noGlobalImageRuntimeHeader.test.js` |
| F-032 | Exec Log Per Step Push | [exec_log_per_step_push](frontend/exec_log_per_step_push.intent.md) | ✅ [test](frontend/exec_log_per_step_push.test.intent.md) | `taskFE/tests/*.playwright.test.js` |
| F-033 | Auto Run Steps MD | [auto_run_steps_md](frontend/auto_run_steps_md.intent.md) | ✅ [test](frontend/auto_run_steps_md.test.intent.md) | `taskFE/tests/*.playwright.test.js` |
| F-035 | 镜像容器技能列表 | [image-container-skill-list](image-container-skill-list.intent.md) | ✅ [test](image-container-skill-list.test.intent.md) | `parse_image_skills_test.go` + `ImageSkillsList.test.js` + `imageSkills.test.js` |
| F-034 | 评论级克隆进度条（TraceId 下） | [comment_clone_progress_under_trace](frontend/comment_clone_progress_under_trace.intent.md) | ✅ [test](frontend/comment_clone_progress_under_trace.test-intent.md) | `commentCloneProgressFromLogs.test.js` + `commentCloneProgressRepoCatalog.test.js` + `TaskDetailCommentCloneProgress.test.js` + `TaskDetailCommentExecutionDetails.test.js` + `taskDetailFetchFns.repoCloneIdentity.test.js` + `applyContainerGitCloneProgress.test.js` + `bootstrap.cloneFailureFooter.test.mjs` |
| F-102 | 自动执行队列与评论逐条执行同卡 | [comment_queue_serial_execution](frontend/comment_queue_serial_execution.intent.md) | ✅ [test](frontend/comment_queue_serial_execution.test.intent.md) | `CommentExecutionDependencyPicker.test.js` + `TaskDetailCommentComposer.test.js` + `TaskDetailTaskIdentityPanel.aux-info.test.js` + `taskFE/tests/TaskDetail.*.playwright.test.js` |
| F-104 | 加入自动执行队列前须启用工作空间自动调度 | [join_queue_requires_workspace_schedule](frontend/join_queue_requires_workspace_schedule.intent.md) | ✅ [test](frontend/join_queue_requires_workspace_schedule.test.intent.md) | `TaskDetailQueuedScheduleToggle.test.js` + `workspaceAutoScheduleEnabled.test.js` |
| F-106 | 未启用工作空间自动调度时拒绝 queued_auto_run 入队 | [workspace_schedule_disabled_enqueue_conflict](frontend/workspace_schedule_disabled_enqueue_conflict.intent.md) | ✅ [test](frontend/workspace_schedule_disabled_enqueue_conflict.test-intent.md) | `taskTaskService/src/queued_schedule_enqueue_guard_test.go` |
| F-109 | 自动调度安排页展示调度历史 | [queue_schedule_history](frontend/queue_schedule_history.intent.md) | ✅ [test](frontend/queue_schedule_history.test-intent.md) | `ScheduleHistoryCard.test.js` + `WorkspaceQueueSchedule.test.js` + `useWorkspaceQueueSchedule.test.js` |

### 2.5 导航栏与系统

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-040 | Navbar Multi Account Switcher | [navbar_multi_account_switcher](frontend/navbar_multi_account_switcher.intent.md) | ✅ [test](frontend/navbar_multi_account_switcher.test-intent.md) | `taskFE/tests/*.playwright.test.js` |
| F-041 | Navbar Public Client IP | [navbar_public_client_ip](frontend/navbar_public_client_ip.intent.md) | ✅ [test](frontend/navbar_public_client_ip.test-intent.md) | 前端组件测试 |
| F-042 | 工作面板标题行任务搜索过滤 | [navbar_task_search_filter](frontend/navbar_task_search_filter.intent.md) | ✅ [test](frontend/navbar_task_search_filter.test-intent.md) | `WorkPanelHeader.legend.test.js` + `Navbar.ui.test.js` + `NavbarTaskSearch.test.js` + `navbarTaskSearch.test.js` + `task_store_search_normalize_test.go` |
| F-043 | System Admin Non-Admin Redirect to Work Panel | [system_admin_non_admin_redirect_work_panel](frontend/system_admin_non_admin_redirect_work_panel.intent.md) | ✅ [test](frontend/system_admin_non_admin_redirect_work_panel.test-intent.md) | `taskFE/tests/SystemAdmin.*.playwright.test.js` |
| F-044 | PeopleManage 成员 Git 身份 | [people_manage_member_git_identities](frontend/people_manage_member_git_identities.intent.md) | ✅ [test](frontend/people_manage_member_git_identities.test.intent.md) | `MemberGitIdentitiesModal.test.js` |
| F-111 | 开放式邀请链接 UI | [open_invite_link](frontend/open_invite_link.intent.md) | ✅ [test](frontend/open_invite_link.test-intent.md) | `InviteLinkMethodPanel` / `peopleInviteLinkPayload` / `PendingInvitations` 单测 |
| F-114 | 邀请链接有效期可设到 365 天 | [people_invite_expiration](frontend/people_invite_expiration.intent.md) | ✅ [test](frontend/people_invite_expiration.test-intent.md) | `peopleInviteExpiration.test.js` + `InviteLinkMethodPanel.test.js` + `PendingInvitations.click-guard.test.js` |
| F-115 | 任务/项目详情查阅内容历史版本 | [task_project_entity_revisions](frontend/task_project_entity_revisions.intent.md) | ✅ [test](frontend/task_project_entity_revisions.test-intent.md) | `taskFE/app/src/components/entity-revision/*.test.js` |
| F-091 | 无推荐资格也展示不透明 accessCode | [user_referral_access_code](frontend/user_referral_access_code.intent.md) | ✅ [test](frontend/user_referral_access_code.test-intent.md) | `UserReferral.accessCode.test.js` + `taskReferral/src/referral_share_code_test.go` |
| F-101 | 申请推荐资格前先关注服务号 | [user_referral_mp_follow_gate](frontend/user_referral_mp_follow_gate.intent.md) | ✅ [test](frontend/user_referral_mp_follow_gate.test-intent.md) | `ReferralServiceAccountFollowGate.test.js` + `UserReferral.mpFollowGate.test.js` |
| F-107 | 工作空间「是否设为默认」须切换租户默认 | [workspace_set_default_takes_effect](frontend/workspace_set_default_takes_effect.intent.md) | ✅ [test](frontend/workspace_set_default_takes_effect.test-intent.md) | `WorkspaceSettingsTaskPanel.default-badge.test.js` + `workspace_default_test.go` |
| F-108 | 工作空间「套餐设置」打开任务存档档位 | [workspace_task_archive_settings](frontend/workspace_task_archive_settings.intent.md) | ✅ [test](frontend/workspace_task_archive_settings.test-intent.md) | `WorkspaceSettingsTaskPanel.archive-modal.test.js` + `WorkspaceSettingsTaskPanelActions.unit.test.js` |

### 2.6 登录与 OTP

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-050 | Login Phone OTP Enable | [login_phone_otp_enable](frontend/login_phone_otp_enable.intent.md) | ✅ [test](frontend/login_phone_otp_enable.test-intent.md) | `taskFE/tests/Login.phone-login-policy-toggle.playwright.test.js` |
| F-051 | OTP Go Full Native SMS | [otp_go_full_native_sms](frontend/otp_go_full_native_sms.intent.md) | ✅ [test](frontend/otp_go_full_native_sms.test-intent.md) | `taskAuth/*_test.go` |
| F-052 | OTP Go Migration inc1 + Add Account AC7 | [otp_go_migration_inc1_and_add_account_ac7](frontend/otp_go_migration_inc1_and_add_account_ac7.intent.md) | ✅ [test](frontend/otp_go_migration_inc1_and_add_account_ac7.test-intent.md) | `taskAuth/*_test.go` |

### 2.7 项目详情

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-060 | Project Internal Repo Disk Size | [project_internal_repo_disk_size](frontend/project_internal_repo_disk_size.intent.md) | ✅ [test](frontend/project_internal_repo_disk_size.test-intent.md) | `taskFE/tests/ProjectDetail.*.playwright.test.js` |
| F-061 | Project Nested Git Repos | [project_nested_git_repos](frontend/project_nested_git_repos.intent.md) | ✅ [test](frontend/project_nested_git_repos.test-intent.md) | `taskFE/tests/ProjectDetail.nested-git-*.playwright.test.js` |
| F-063 | Project Auto Clone Nested Repos | [project_auto_clone_nested_repos](frontend/project_auto_clone_nested_repos.intent.md) | ✅ [test](frontend/project_auto_clone_nested_repos.test-intent.md) | `useCreateProjectForm.autoClone.test.js` + `projectDetailAutoRunGitGate.test.js` + `ProjectDetailGitReposSection.nested-oauth.test.js` + Go auto_clone_*_test.go |
| F-062 | HTML Head Trae Service | [html_head_trae_service](frontend/html_head_trae_service.intent.md) | ✅ [test](frontend/html_head_trae_service.test-intent.md) | 前端 HTML 检查 |

### 2.8 Chrome 插件

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-070 | Badge 仅 5xx 显示 | [task_chrome_plugin_badge_5xx_only](frontend/task_chrome_plugin_badge_5xx_only.intent.md) | ✅ [test](frontend/task_chrome_plugin_badge_5xx_only.test-intent.md) | `taskChromePlugin/*` |
| F-071 | Sibling Range Pick | [task_chrome_plugin_sibling_range_pick](frontend/task_chrome_plugin_sibling_range_pick.intent.md) | ✅ [test](frontend/task_chrome_plugin_sibling_range_pick.test-intent.md) | `taskChromePlugin/*` |
| F-072 | User Guide | [task_chrome_plugin_user_guide](frontend/task_chrome_plugin_user_guide.intent.md) | ✅ [test](frontend/task_chrome_plugin_user_guide.test-intent.md) | `taskChromePlugin/*` |
| F-073 | 点选请求写入请求体 | [task_chrome_plugin_request_body_in_task_desc](frontend/task_chrome_plugin_request_body_in_task_desc.intent.md) | ✅ [test](frontend/task_chrome_plugin_request_body_in_task_desc.test-intent.md) | `taskChromePlugin/test/single-request-task-desc.test.js` + `request-body-cache.test.js` + `service-worker-request-body.test.js` + `devtools-request-body.test.js` |
| F-074 | 浮窗工作空间同名消歧 | [task_chrome_plugin_workspace_list_dedupe](frontend/task_chrome_plugin_workspace_list_dedupe.intent.md) | ✅ [test](frontend/task_chrome_plugin_workspace_list_dedupe.test-intent.md) | `taskChromePlugin/test/workspace-list.test.js` + `api-endpoints.test.js` |
| F-075 | 项目列表标注是否可自动运行 | [task_chrome_plugin_project_auto_run_label](frontend/task_chrome_plugin_project_auto_run_label.intent.md) | ✅ [test](frontend/task_chrome_plugin_project_auto_run_label.test-intent.md) | `taskChromePlugin/test/project-auto-run-label.test.js` |
| F-116 | DevTools 请求列表过滤与排序 | [task_chrome_plugin_devtools_request_filter_sort](frontend/task_chrome_plugin_devtools_request_filter_sort.intent.md) | ✅ [test](frontend/task_chrome_plugin_devtools_request_filter_sort.test-intent.md) | `taskChromePlugin/test/request-list-query.test.js` + `test/user-guide.test.js` |

### 2.9 ZTree

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-080 | ZTree Merge to Target Branch | [ztree_merge_to_target_branch](frontend/ztree_merge_to_target_branch.intent.md) | ✅ [test](frontend/ztree_merge_to_target_branch.test-intent.md) | 前端测试 |
| F-081 | ZTree Push Ahead after Success | [ztree_push_ahead_after_success](frontend/ztree_push_ahead_after_success.intent.md) | ✅ [test](frontend/ztree_push_ahead_after_success.test-intent.md) | 前端测试 |
| F-082 | ZTree Push and Create PR | [ztree_push_and_create_pr](frontend/ztree_push_and_create_pr.intent.md) | ✅ [test](frontend/ztree_push_and_create_pr.test-intent.md) | 前端测试 |
| F-117 | 层图 GitLab 评论 L2 换票勿误报未绑定 | [043_layer_oauth_gitlab_binding](frontend/task_detail/043_layer_oauth_gitlab_binding.intent.md) | ✅ [test](frontend/task_detail/043_layer_oauth_gitlab_binding.test-intent.md) | `layer_oauth_test.go` + `sqlite_business_comment_author_test.go` + `http_business_test.go` + `layerZtreePushError.test.js` |
| F-118 | Fork 自动运行评论继承源任务同用户 Git L2 | [044_fork_autorun_comment_oauth_l2](frontend/task_detail/044_fork_autorun_comment_oauth_l2.intent.md) | ✅ [test](frontend/task_detail/044_fork_autorun_comment_oauth_l2.test-intent.md) | `taskTaskService/src/comment_oauth_fork_seed_test.go` |
| F-094 | PR 链接回复、合并状态与一键合并 | [pr_reply_merge_status](frontend/pr_reply_merge_status.intent.md) | ✅ [test](frontend/pr_reply_merge_status.test-intent.md) | `taskDetailLayerActions.test.js` + `buildDisplayComments.test.js` + `CommentGitPrReply.test.js` + `taskGitOauth/domain/merge_request_ref_test.go` + `taskTaskService` git_pr 评论测 |
| F-105 | 创建任务自动运行时可加入自动调度队列 | [create_task_queued_auto_run](frontend/create_task_queued_auto_run.intent.md) | ✅ [test](frontend/create_task_queued_auto_run.test-intent.md) | `createTaskQueuedAutoRun.test.js` + `CreateTaskAutoRunSection.test.js` + `taskTaskService/src/auto_run_test.go` + `taskChromePlugin/test/workspace-auto-schedule.test.js` |
| F-083 | ZTree Submit Visible with File Changes | [ztree_submit_visible_with_file_changes](frontend/ztree_submit_visible_with_file_changes.intent.md) | ✅ [test](frontend/ztree_submit_visible_with_file_changes.test-intent.md) | 前端测试 |
| F-086 | ZTree 选中失焦后不隐藏关联面板 | [040_ztree_selection_persist_on_blur](frontend/task_detail/040_ztree_selection_persist_on_blur.intent.md) | ✅ [test](frontend/task_detail/040_ztree_selection_persist_on_blur.test-intent.md) | `TaskDetailTaskLayerAssociationPanel.click-outside.test.js` |
| F-084 | 创建订单后独立订单详情页 | [billing_order_detail_page](frontend/billing_order_detail_page.intent.md) | ✅ [test](frontend/billing_order_detail_page.test-intent.md) | `OrderCreate.contract.test.js` + `OrderDetail.contract.test.js` + `OrderDetail.refund.test.js` + `BillingRefundConfirmModal.test.js` + `router.billingOrderDetail.test.js` |
| F-085 | 定价页去掉会员等级体系描述 | [pricing_hide_membership_tier_copy](frontend/pricing_hide_membership_tier_copy.intent.md) | ✅ [test](frontend/pricing_hide_membership_tier_copy.test-intent.md) | `taskFE/app/src/views/Pricing.membership-section.unit.test.js` + `taskFE/tests/Home.nav-pricing.playwright.test.js` |

### 2.10 厂商门户 / 镜像市场

| ID | 意图 | 意图文件 | 测试定义 | Playwright/单元测试 |
|----|------|---------|---------|-------------------|
| F-087 | 厂商门户顶栏镜像 Demo GitHub 链接 | [image_demo_github_link](frontend/provider/image_demo_github_link.intent.md) | ✅ [test](frontend/provider/image_demo_github_link.test-intent.md) | `taskAiProvider/frontend/tests/imageDemoLink.unit.test.js` |
| F-088 | 厂商门户顶栏 SaaS 容器 Skill 链接 | [saas_machine_container_skill_link](frontend/provider/saas_machine_container_skill_link.intent.md) | ✅ [test](frontend/provider/saas_machine_container_skill_link.test-intent.md) | `saasMachineContainerSkill.unit.test.js` |
| F-089 | 厂商门户申请认证（镜像市场去掉申请） | [vendor_apply_on_provider_portal](frontend/provider/vendor_apply_on_provider_portal.intent.md) | ✅ [test](frontend/provider/vendor_apply_on_provider_portal.test-intent.md) | `vendorApplyCta.unit.test.js` + `ImageMarket.vendorStatus.test.js` + `vendor_applicant_auth_test.go` |
| F-110 | 平台审核镜像操作列提供查看详情 | [admin_image_review_actions](frontend/provider/admin_image_review_actions.intent.md) | ✅ [test](frontend/provider/admin_image_review_actions.test-intent.md) | `taskAiProvider/frontend/tests/adminImageReviewActions.unit.test.js` |
| F-112 | 厂商门户版本行提供下架/删除等管理 | [vendor_version_row_management](frontend/provider/vendor_version_row_management.intent.md) | ✅ [test](frontend/provider/vendor_version_row_management.test-intent.md) | `vendorVersionRowActions.unit.test.js` + `vendor_container_lifecycle_test.go` |
| F-090 | 系统管理 GitLab 区域删除须确认仓库地址 | [system_admin_gitlab_region_delete_confirm](frontend/system_admin_gitlab_region_delete_confirm.intent.md) | ✅ [test](frontend/system_admin_gitlab_region_delete_confirm.test-intent.md) | `SystemAdminGitlabRegionDeleteModal.test.js` + `SystemAdminGitlabResources.delete-confirm.test.js` |
| F-119 | GitLab 磁盘待开通队列与等待开通文案 | [gitlab_disk_manual_admin_fulfillment](frontend/gitlab_disk_manual_admin_fulfillment.intent.md) | ✅ [test](frontend/gitlab_disk_manual_admin_fulfillment.test-intent.md) | 待实现：`SystemAdminGitlabTenantPanel` 队列 + 连接页 pending |
| F-092 | 交易记录资源流水账展示 | [billing_transactions_resource_ledger](frontend/billing_transactions_resource_ledger.intent.md) | ✅ [test](frontend/billing_transactions_resource_ledger.test-intent.md) | `transactionChangeDisplay.test.js` + `BillingTransactionsTable.grantDisplay.test.js` + `BillingDashboard.grantDisplay.test.js` |
| F-095 | 推荐绩效抽屉微信分账 Tab | [system_admin_referral_wechat_profit_sharing_tab](frontend/system_admin_referral_wechat_profit_sharing_tab.intent.md) | ✅ [test](frontend/system_admin_referral_wechat_profit_sharing_tab.test-intent.md) | `SystemAdminReferralPerformanceDrawer` Tab / 同步按钮单测 |
| F-093 | 订单记录页按交易单号查询 | [system_admin_order_trade_no_query](frontend/system_admin_order_trade_no_query.intent.md) | ✅ [test](frontend/system_admin_order_trade_no_query.test-intent.md) | `SystemAdminOrderListPanel.orderNumberPaste.test.js` |
| F-096 | GitLab 连接页区域详情在下拉之下 | [gitlab_connection_region_details_below_picker](frontend/gitlab_connection_region_details_below_picker.intent.md) | ✅ [test](frontend/gitlab_connection_region_details_below_picker.test-intent.md) | `WorkspaceSettingsGitlabConnection.test.js` |
| F-113 | 自建 GitLab 平台可达性（内网豁免） | [gitlab_connection_self_hosted_reachability](frontend/gitlab_connection_self_hosted_reachability.intent.md) | ✅ [test](frontend/gitlab_connection_self_hosted_reachability.test-intent.md) | `GitlabSelfHostedReachability.test.js` + `taskGitOauth/domain/gitlab_reachability_test.go` + `tenant_connection_reachability_test.go` |
| F-097 | GitLab 连接页流量超额阻断提示 | [gitlab_connection_traffic_quota_exceeded](frontend/gitlab_connection_traffic_quota_exceeded.intent.md) | ✅ [test](frontend/gitlab_connection_traffic_quota_exceeded.test-intent.md) | `WorkspaceSettingsGitlabConnection.test.js` |
| F-098 | 购买页 GitLab 流量并入磁盘卡共用区域 | [order_create_gitlab_shared_region](frontend/order_create_gitlab_shared_region.intent.md) | ✅ [test](frontend/order_create_gitlab_shared_region.test-intent.md) | `OrderCreate.contract.test.js` |
| F-099 | 管理端赠送须显式填写数量，避免误赠 | [admin_grant_explicit_quantity](frontend/admin_grant_explicit_quantity.intent.md) | ✅ [test](frontend/admin_grant_explicit_quantity.test-intent.md) | `SystemAdminGrantPoints.explicit-quantity.unit.test.js` |
| F-100 | 账号中心侧边栏可查看登录历史 | [login_history_profile_sidebar](frontend/login_history_profile_sidebar.intent.md) | ✅ [test](frontend/login_history_profile_sidebar.test-intent.md) | `UserLoginHistory.test.js`；`LoginHistoryPanel.test.js`；`UserCenterSidebar.login-history.test.js`；`UserListRow.login-history.test.js` |
| F-100 | 账单页 GitLab 磁盘/流量按区域展示已用量 | [billing_gitlab_quota_region_usage](frontend/billing_gitlab_quota_region_usage.intent.md) | ✅ [test](frontend/billing_gitlab_quota_region_usage.test-intent.md) | `BillingDashboard.gitlabRegions.test.js` + `BillingDashboard.gitlabQuotaSplit.test.js` + `formatUsedGb.test.js` + `taskBill/src/gitlab_resources_test.go` |

---

## 三、容器意图 (Container Intents)

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| C-001 | SSH/HTTPS Clone + Selected Image Token Sync | [006](container/006_ssh_https_clone_and_selected_image_token_sync.intent.md) | ✅ [test](container/006_ssh_https_clone_and_selected_image_token_sync.test-intent.md) | `taskContainerGateway/*` + `taskCredentialService/*` |
| C-002 | Reclone HTTPS + UI Stale Token | [007](container/007_reclone_https_and_ui_stale_token.intent.md) | ✅ [test](container/007_reclone_https_and_ui_stale_token.test-intent.md) | `taskContainerGateway/src/relay_token_init_handlers_test.go` |
| C-003 | Selected Image Pull Hash Log | [008](container/008_selected_image_pull_hash_log.intent.md) | ✅ [test](container/008_selected_image_pull_hash_log.test-intent.md) | `taskContainerGateway/*` |
| C-004 | Scoped Container UI Path | [009](container/009_scoped_container_ui_path.intent.md) | ✅ [test](container/009_scoped_container_ui_path.test-intent.md) | `taskContainerGateway/*` |
| C-005 | Config SaaS Fallback | [010_config_saas_fallback](container/010_config_saas_fallback.intent.md) | ✅ [test](container/010_config_saas_fallback.test-intent.md) | `go_run_container/*` |
| C-006 | Relay Clear State on Container Start | [010_relay_clear_state_on_container_start](container/010_relay_clear_state_on_container_start.intent.md) | ✅ [test](container/010_relay_clear_state_on_container_start.test-intent.md) | `taskContainerGateway/src/relay_lifecycle_test.go` |
| C-007 | 克隆手动重试成功后恢复自动任务 | [011_reclone_resumes_agent_kickoff](container/011_reclone_resumes_agent_kickoff.intent.md) | ✅ [test](container/011_reclone_resumes_agent_kickoff.test-intent.md) | `trae-agent/onlineServiceJS/src/postBootstrapAgentKickoff.test.mjs` |

---

## 四、平台意图 (Platform Intents)

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| P-001 | Registration Invite Code | [registration-invite-code](platform/registration-invite-code.intent.md) | ✅ [test](platform/registration-invite-code.test-intent.md) | `taskAuth/*_test.go` + `taskFE/tests/AuthRegister.*.playwright.test.js` |
| P-002 | GitService Durable GitLab Home | [gitservice-durable-gitlab-home](platform/gitservice-durable-gitlab-home.intent.md) | ✅ [test](platform/gitservice-durable-gitlab-home.test.intent.md) | `gitService/*_test.go` (144 files) |
| P-003 | runAll Logs Clear via Truncate | [runall_logs_clear_via_truncate](platform/runall_logs_clear_via_truncate.intent.md) | ✅ [test](platform/runall_logs_clear_via_truncate.test-intent.md) | `runAll/*_test.go` (64 files) |
| P-004 | Daydaymoney YAML Metadata | [daydaymoney-yaml-metadata](platform/daydaymoney-yaml-metadata.intent.md) | ✅ [test](platform/daydaymoney-yaml-metadata.test-intent.md) | YAML 格式检查 |
| P-005 | runAll 全部重新编译清空精准登记 | [runall_build_all_clears_precise_restart](platform/runall_build_all_clears_precise_restart.intent.md) | ✅ [test](platform/runall_build_all_clears_precise_restart.test-intent.md) | `runAll/src/build_all_clears_precise_restart_test.go` |
| P-006 | :9999 失联时看门狗不自动重启 runAll | [watchdog_no_auto_restart_runall_on_9999_down](platform/watchdog_no_auto_restart_runall_on_9999_down.intent.md) | ✅ [test](platform/watchdog_no_auto_restart_runall_on_9999_down.test-intent.md) | `runAll/scripts/tests/test_ensure_services_healthy.py` |
| P-007 | 9999 初始化按钮标注未 migrate | [runall_pending_migrate_badge](platform/runall_pending_migrate_badge.intent.md) | ✅ [test](platform/runall_pending_migrate_badge.test-intent.md) | `runAll/src/domain/migrate_pending_test.go` + `ui_migrate_pending_test.go` |
| P-008 | 全部重启不编译、精准重启先编后切 | [runall_restart_compile_separation](platform/runall_restart_compile_separation.intent.md) | ✅ [test](platform/runall_restart_compile_separation.test-intent.md) | `runAll/src/*restart*` + `taskEvents/run.sh` |
| P-009 | conf-local 唯一 overlay（不读 config.local.yaml） | [conf-local-secrets-completion](platform/conf-local-secrets-completion.intent.md) | ✅ [test](platform/conf-local-secrets-completion.test-intent.md) | `shareLib/confload` + `check_conf_local_secrets.py` |
| P-010 | 新节点克隆 daydaymoney-deploy 不只靠 conf-local | [daydaymoney-deploy-new-node-clone](platform/daydaymoney-deploy-new-node-clone.intent.md) | ❌ 待设计批准后补 | seed `up.sh` 对齐 + README 清单 |

---

## 五、业务意图 (Business Intents)

| ID | 意图 | 意图文件 | 测试定义 | 可执行测试 |
|----|------|---------|---------|-----------|
| M-001 | 微信支付充值 (WeChat Pay Recharge) | [wechat-pay-recharge](../wechat-pay-recharge.intent.md) | ✅ [test](../wechat-pay-recharge.test.intent.md) | 支付流程 E2E |
| M-002 | KYC 充值门禁 | [kyc-recharge-gate](../kyc-recharge-gate.intent.md) | ✅ [test](../kyc-recharge-gate.test.intent.md) | KYC 验证流程 |
| M-003 | Tenant 退款申请 | [tenant-refund-application](../tenant-refund-application.intent.md) | ✅ [test](../tenant-refund-application.test.intent.md) | 退款流程测试 |
| M-004 | 历史定价包管理 | [historical-pricing-package-management](../historical-pricing-package-management.intent.md) | ✅ [test](../historical-pricing-package-management.test-intent.md) | 定价模型测试 |
| M-005 | Admin 充值消费概览 | [admin-recharge-consumption-overview](../admin-recharge-consumption-overview.intent.md) | ✅ [test](../admin-recharge-consumption-overview.test.intent.md) | 管理面板测试 |

---

## 六、E2E 测试意图 (.testIntent 伴随文档)

这些是位于测试文件旁边的 `.testIntent` 伴随文档：

### 6.1 基础设施 E2E (`e2e-tests/`)

| 测试文件 | 测试意图 |
|---------|---------|
| `oauth-redirect-e2e.js` | OAuth 重定向链验证（不跳转到 localhost） |
| `oauth-token-exchange-e2e.js` | OAuth Token 交换基础设施完整性 |
| `people-manage-permission-isolation-e2e.js` | 人员管理权限隔离（403 vs 200） |

### 6.2 前端 E2E (`taskFE/tests/`) — 30 个测试

覆盖：Onboarding、WorkspaceFeatureParams、WorkspaceAccess、TodoCreate、TaskDetail (8场景)、SystemAdmin、Projects、ProjectDetail (4场景)、Login、CreateProject (4场景)、AuthRegister

---

## 七、服务测试覆盖矩阵

| 服务 | Go 测试文件数 | .testIntent 数 | 意图覆盖 |
|------|-------------|---------------|---------|
| taskAuth | 136 | 0 | ✅ OIDC + OTP 意图 |
| taskBill | 17 | 0 | ✅ 定价 + 预算意图 |
| taskCloudService | 91 | 0 | ✅ Cloud 意图 |
| taskContainerGateway | 16 | 0 | ✅ 容器执行意图 |
| taskCredentialService | 12 | 0 | ✅ Token + OAuth 意图 |
| taskEvents | 59 | 0 | ✅ SSE + Events 意图 |
| taskGitOauth | 14 | 0 | ✅ OAuth 迁移意图 |
| taskProjectService | 25 | 0 | ⚠️ 缺专门意图 |
| taskReferral | 4 | 0 | ✅ Referral 意图 |
| taskTaskService | 20 | 0 | ✅ 任务系统意图 |
| taskTenantService | 7 | 0 | ✅ Feature Params + Member 意图 |
| taskAiProvider | 7 | 1 | ✅ AI Provider 意图 |
| taskAIComment | 8 | 0 | ✅ Comment 意图 |
| taskAIEndPoint | 3 | 0 | ✅ Budget 意图 |
| taskAgentSupport | 2 | 0 | ⚠️ 缺专门意图 |
| gitService | 144 | 2 | ✅ GitService + OIDC 意图 |
| go_relayToTrae | 15 | 0 | ✅ OnlineService 意图 |
| go_run_container | 6 | 0 | ✅ Config Fallback 意图 |
| runAll | 64 | 0 | ✅ runAll 意图 |
| valueStream | 16 | 0 | ⚠️ 缺专门意图 |
| taskFE | — | 33 | ✅ 前端全部意图 |
| taskSSE | 0 | 0 | ❌ 无测试覆盖 |
| taskGateway | 0 | 0 | ❌ 无测试覆盖 |
| taskChromePlugin | — | 0 | ✅ Chrome Plugin 意图 |

---

## 八、已知缺口 (Known Gaps)

### 需要补充测试定义的意图（有 intent 无 test-intent）
1. `feature_params_proxy_rewrite_go` — 缺测试定义
2. `cloud_domain_tables_task_cloud_migration` — 缺测试定义
3. `ecs_orphan_double_start_guard` — 缺测试定义
4. `budget_console_user_api_cloud` — 缺测试定义
5. `budget_record_usage_batch_thin_removed` — 缺测试定义
6. `saas_budget_ledger_http_cutover` — 缺测试定义
7. `tenant_budget_permission_cloud` — 缺测试定义
8. `top_deliverable_queued_auto_run_schedule` — 缺测试定义
9. `rewrite_sub_token_python_removed` — 缺测试定义
10. `task_events_saas_http_cutover` — 缺测试定义
11. `task_cloud_saas_sqlite_http_cutover` — 缺测试定义

### 缺测试覆盖的服务
- **taskSSE** — 0 测试文件
- **taskGateway** — 0 测试文件

### 缺意图文档的服务
- **taskProjectService** — 25 个测试文件但缺专门意图文档
- **taskAgentSupport** — 2 个测试文件但缺专门意图文档
- **valueStream** — 16 个测试文件但缺专门意图文档

---

## 九、运行方式

### 快速运行全部意图测试

```bash
# 运行所有可自动执行的意图测试
bash scripts/run-all-intent-tests.sh

# 仅运行后端意图测试
bash scripts/run-all-intent-tests.sh --category backend

# 仅运行前端意图测试
bash scripts/run-all-intent-tests.sh --category frontend

# 生成 JSON 报告
bash scripts/run-all-intent-tests.sh --format json

# 使用技能
/intent-test
```

### 技能

- **`/intent-test`** — 完整意图验证技能，自动发现并执行所有意图测试

---

## 十、维护约定

1. **新增意图** → 在对应目录创建 `{name}.intent.md` + `{name}.test-intent.md`
2. **新增测试** → 创建对应的 `${testFileName}.testIntent` 伴随文档
3. **意图变更** → 同步更新本 INDEX.md
4. **发现缺口** → 在 `.learnings/OPTIMIZATION_TODOS.md` 记录 OPT 条目
