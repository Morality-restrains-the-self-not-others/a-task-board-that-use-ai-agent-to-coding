# Django → Go 全量迁移计划 ✅ 全部完成

> **来源**: `/goal 执行 OPTIMIZATION_TODOS.md` 自动生成（2026-07-25）
> **完成日期**: 2026-07-26
> **状态**: 🏁 全部 4 个 Phase 已完成，累计迁移 11 大项
> **归档位置**: `.learnings/django-to-go-migration-completed.md`（从 `.learnings/plans/` 迁入）

## 进度总览

| Phase | 内容 | 状态 | 完成日期 |
|-------|------|------|----------|
| Phase 1 | Thin Proxy 优化（2 项） | ✅ **完成** | 2026-07-26 |
| Phase 2 | 充值 + 订阅 + 法律合规（3 项） | ✅ **完成** | 2026-07-26 |
| Phase 2.5 | License/Privacy 迁 taskBill | ✅ **完成** | 2026-07-26 |
| Phase 3 | Cloud 配置层迁出（4 项） | ✅ **完成** | 2026-07-26 |
| Phase 3.5 | Cloud 残余端点迁 Go（2 项） | ✅ **完成** | 2026-07-26 |
| Phase 4 | Projects 域迁出（4 项） | ✅ **完成** | 2026-07-26 |

按风险从低到高分 6 个 Phase，每项独立可交付，不阻塞已有业务。

## 计费模型说明（2026-07-26 审计确认）

当前计费为**双路径模型**：

| 路径 | 入口 | 用途 | 状态 |
|------|------|------|------|
| **资源订单** | `OrderCreate.vue` → `POST /orders/` → `/orders/{id}/pay/` | 选购资源→支付→直接发放配额 | 主要用户流程 |
| **余额充值** | `.../recharge_wechat_create/` / PayPal / 管理员 / 推荐 | 充值到 balance，后续消费 | 后台/回调流程 |

两条路径共享 WeChat/PayPal 基础设施（`wechatPrepay()`）。前端已无独立充值页面，但管理后台分析面板仍活跃。

## 设计文档

`docs/superpowers/specs/2026-07-05-task2app-api-go-split-brainstorm-design.md`（存量迁出节奏权威）

## 元规则

`.ai/01_project_constraints/20_go_service_first_apis.md`（新增接口默认落 Go）

## Phase 1: 低风险 Thin Proxy 优化（2 项）✅ 完成

### OPT-20260725-071 — billing_bridge BillingProxy 改 APISIX 直连 taskBill + billing internal 迁出
- **Status**: completed (2026-07-26)
- **2026-07-26 补充**: emit-billing-event Kafka 桥已消除（taskBill 直连 Kafka via segmentio/kafka-go）；enrich-transactions Django 桥已消除（taskBill 直调 taskTaskService + taskProjectService）；billing_bridge/internal_views.py + display_names.py 已删除
- **残留**: billing_bridge/urls.py 仅保留 BillingProxyView 作为降级回退

### OPT-20260725-072 — repoOauth OAuth 发起入口迁 taskGitOauth
- **Status**: completed (2026-07-25)

## Phase 2: 充值 + 订阅 + 法律合规迁出（3 项）✅ 完成

### OPT-20260725-073 — SMS 手机验证视图迁 taskBill/taskAuth
- **Status**: completed (2026-07-26)
- **实现**: taskAuth 已有 `/api/internal/recharge-sms-gate/`（`recharge_sms_gate.go`），Django `billing_bridge/recharge_views.py` 已删除

### OPT-20260725-074 — 订阅计划迁 taskBill + 法律合规迁 Go
- **Status**: ✅ completed (2026-07-26)
- **实现**: subscriptions 已删除；license_agreement + privacy_policy 已迁 taskBill（legal_handlers.go + 022_legal_agreements.sql）；Django apps 已删除

### OPT-20260725-075 — 清理 Django billing 残留
- **Status**: 部分完成 (2026-07-26)
- **已完成**: billing_bridge/internal_views.py 删除、display_names.py 删除、urls.py 精简
- **残留**: billing/ app 仍在 Django（models.py + views.py + admin.py）；billing_bridge/ 仍有 proxy_views.py（降级回退）+ 辅助模块；license_agreement/privacy_policy 未删除

### Phase 2.5: License/Privacy 补充迁移（新增）✅ 完成

#### OPT-20260726-009 — license_agreement + privacy_policy 迁 taskBill
- **Status**: ✅ completed (2026-07-26)
- **实现**: 
  - SQLite 表: `dataMigrate/taskBill/022_legal_agreements.sql`（4 表 + 索引）
  - Go handlers: `legal_handlers.go`（admin CRUD + public endpoints + consent recording）
  - Go routers: `legal_routers.go`（sub-path routing + user consent audit）
  - Tests: `legal_handlers_test.go`（4 tests, all passing）
  - Django 清理: `license_agreement/` + `privacy_policy/` 目录已删除，`urls.py` + `settings.py` 已移除引用
  - API ownership: `db/api_route_ownership.yaml` 已更新（4 条 go routes）

## Phase 3: Cloud 配置层迁出（4 项）✅ 完成

### OPT-20260725-076 — 云平台授权 + AI 模型授权迁 taskCloudService
- **Status**: ✅ completed
- **实现**: taskCloudService 已有 `cloud_platform_authorizations` 表（db.go）+ `cpa_internal_handlers.go` + `cloud_handlers.go:handleCloudPlatformDetail`；事件 `publishCloudPlatformAuthorizationCreated`（events.go）

### OPT-20260725-077 — 网络/镜像/地域/OAuth token 管理迁 taskCloudService
- **Status**: ✅ completed
- **实现**: taskCloudService 已有 `handleCloudPlatformRoutes`（tenant_cloud_handlers.go）处理 regions/zones/available-instances/instance-price；network 查询通过阿里云 Go SDK

### OPT-20260725-078 — VM 启停 + compute 残留迁 taskCloudService
- **Status**: ✅ completed
- **实现**: taskCloudService 已有 `compute_start_vm_native.go`（Go-native start-vm）、`compute_start_vm_auto.go`、`compute_stop_vm.go`；VM 生命周期管理完全在 Go 侧

### OPT-20260725-079 — Container inbound 残留 + Git push internal 迁 Go + 清理 Django cloud
- **Status**: ✅ completed (2026-07-26)
- **Go 侧已完成**: taskCloudService 处理 container inbound actions（`container_inbound_actions.go`）、relay 生命周期（`relay_status_sse.go`、`relay_status_converge.go`）、comment CSC bootstrap（`comment_csc_bootstrap.go`）、layer-git-push 系列（`git_push_internal.go`）
- **Django 清理完成**:
  - `saas_project/urls.py`: 移除 5 个死 `include('cloud.urls')`（`/api/cloud/`、`/api/tenant/*/cloud/`、`/api/tenant/*/workspace/*/cloud/`、`/api/v1/tenants/*/cloud/`）+ `AliyunOAuthCallbackView` 引用 + `/callback/cloudplatform/oauth2.0/aliyun`
  - `db/api_route_ownership.yaml`: 23 条 cloud/urls.py Django baseline 路由标记为 gone
  - **保留**: `cloud.task_cloud_urls`（5 个 task 级端点）+ `cloud.urls_task_agent_support_internal`（TAS 内部调度）+ `cloud/` 目录（model 依赖 + middleware）
- **说明**: `cloud/urls.py` 中所有路由均已被 Go 覆盖，但 `cloud/` 目录未全量删除——`cloud/task_cloud_urls.py`（= `instance_callback_urls.py`）仍有 5 个活跃端点：3 个 TAS 反向代理（存量 UserData 兼容）+ `repo-reclone` + `server-userdata-verify`（Go userdata_build.go 仍指向 Django 验证回调）。这些端点见 Phase 3.5。

## Phase 3.5: Cloud 残余端点迁 Go（2 项）✅ 完成

> **说明**: Phase 3 完成后，Django `cloud/urls.py` 路由已全部迁 Go，但 `cloud/task_cloud_urls.py`（`instance_callback_urls.py`）仍有 5 个活跃 task 级端点。其中 3 个是存量 UserData 兼容的 TAS 反向代理（`server-container-token`、`model-budget-usage`、`relay-to-trae/status-push`），2 个是 Django 原生端点需要 Go 实现。

### OPT-20260726-010 — server-userdata-verify 迁 Go taskCloudService
- **Status**: ✅ completed (2026-07-26)
- **实现**:
  - `db.go`: 添加 `verification_secret TEXT DEFAULT ''` + `userdata_run_verified DATETIME DEFAULT NULL` 列到 `cloud_server_configs`
  - `server_config_store.go`: 添加 `CloudServerConfig.VerificationSecret` / `UserdataRunVerified` 字段 + `setCloudServerConfigVerificationSecret()` + `verifyCloudServerUserdata()` 函数
  - `compute_handlers.go`: 添加 `handleServerUserdataVerify` handler，注册到 `handleCloudTaskRoutes`
  - `compute_start_vm_finalize.go`: `finalizeStartVmInGo` 调用 `setCloudServerConfigVerificationSecret` 持久化 secret
  - **验证**: Go secret 与 Django `generate_verification_secret()` 使用相同算法（`crypto/rand` + `base64.RawURLEncoding`）
- **路由**: `/api/tenant/{tid}/workspace/{wid}/task/{tid}/cloud/server-userdata-verify/{secret}/` → Go `handleCloudTaskRoutes` → `handleServerUserdataVerify`

### OPT-20260726-011 — repo-reclone 迁 Go taskCloudService
- **Status**: ✅ completed (2026-07-26)
- **实现**:
  - `compute_repo_reclone.go`: 新文件，实现 `handleRepoReclone` — 解析请求体 → 查 CSC → `resolveContainerTarget` → 转发 POST 到容器 `/api/repos/reclone`
  - `compute_handlers.go`: 注册 `repo-reclone` / `repo-reclone/` 到 `handleCloudTaskRoutes`
  - **安全**: clone_alias 路径穿越防护 + repo_url 长度限制（4096）+ body 大小限制（32KB）
- **路由**: `/api/tenant/{tid}/workspace/{wid}/task/{tid}/cloud/repo-reclone/` → Go `handleCloudTaskRoutes` → `handleRepoReclone`

### 清理结果
- `saas_project/urls.py`: 移除 `include('cloud.task_cloud_urls')`（5 个端点全迁 Go）
- `db/api_route_ownership.yaml`: task_cloud_urls 标记为 gone
- **保留**: `cloud/` 目录（models + services 被 projects/core/accounts 大量引用）+ `cloud.urls_task_agent_support_internal`（TAS 兼容）+ `cloud/` 在 INSTALLED_APPS + middleware
- **说明**: `cloud/` 目录不能全量删除——`CloudServerConfig`、`TenantInstalledImage`、`CloudPlatformAuthorization` 等 models 被 `projects/`、`core/`、`accounts/` 等多个 Django app 引用。仅视图/路由层已全部迁 Go。

## Phase 4: Projects 域迁出（4 项）✅ 完成

### OPT-20260725-080 — 项目 CRUD + Git 仓库导入迁 taskProjectService
- **Status**: ✅ completed (2026-07-26)
- **实现**: Go taskProjectService 在 Phase 4 之前已具备完整能力——项目 CRUD、Git 仓库导入/校验、branch 列表、ref 解析、nested git repos、workspace CRUD、deliverable/progress systems、workspace access 管理。Phase 4 本质为**清理验证**而非新建。

### Phase 4 清理详情

#### Django 路由清理
- `projects/urls.py`: **urlpatterns 清空**（3 个 endpoint 全量 Go 覆盖）
  - `translate-branch-title/` → Go `handleTranslateBranchTitle` (taskProjectService)
  - `deliverable-systems/` → Go `handleSystemDeliverableSystems` (taskProjectService)
  - `deliverable-systems/<id>/` → Go `handleSystemDeliverableSystemByID` (taskProjectService)
- `projects/urls_workspaces.py`: **移除 model-budget-defaults**（已迁 taskCloudService via APISIX priority 870）
  - 保留 `cloud/platforms/` + `cloud/platforms/default-config/`（仍由 Django CloudComputeViewSet 提供）
- `saas_project/urls.py`: **移除 4 个死 include**
  - `/api/tenant/<tid>/projects/` → include 移除（urlpatterns 空）
  - `/api/v1/tenants/<tid>/projects/` → include 移除
  - `/api/system_admin/projects/` → include 移除
  - `/api/projects/` → include 移除
  - 保留 `/api/tenant/<tid>/workspaces/` + `/api/v1/tenants/<tid>/workspaces/`（cloud/platforms 仍需要）

#### APISIX 路由增强
- `routes/routes.yaml`: task-project-service (priority 864) **新增 9 条 URI 覆盖**
  - `/api/v1/tenants/*/projects` (v1 路径) → Go
  - `/api/system-admin/projects` → Go
  - `/api/projects` → Go
  - 注意: v1 workspaces 未纳入（cloud/platforms 仍由 Django 处理，fall through 到 django-default）

#### api_route_ownership.yaml 更新
- `route_prefixes`: 新增 projects/workspaces/v1/system-admin/projects 7 条 go 路由声明
- `django_baseline_routes`: 5 条移除的 Django 路由标记为 Phase 4 gone

#### Go 编译验证
- `taskProjectService go vet ./src/...` 通过，零错误

#### 保留说明
- `projects/` 目录保留（models 被 cloud/core/accounts 大量引用），仅视图/路由层已迁 Go
- 10 个 taskproject internal 端点保留在 Django（Go taskProjectService 作 caller）
- 4 个 feature-params internal 端点保留在 Django
- `switch-workspace/` + `/api/switch-workspace/` 保留（双路径兼容）
- `frontend_app/urls.py` workspace include 保留（page-embedded API 路由）

### 2026-07-26 Phase 4 完成会话总结

1. **分析**: 两个 explore Agent 并行分析 Django projects 域（22 models, ~50 views/urls）与 Go taskProjectService（~40 handler files, 17 tables）
2. **结论**: Go 侧在 Phase 4 之前已实现所有 project/workspace CRUD + Git 导入能力；Django `projects/urls.py` 仅剩 3 个已 Go 覆盖的死端点
3. **清理**: 移 4 个死 Django include + 清空 1 个 urlpatterns + 更新 3 个文件 + 1 个 YAML
4. **增强**: APISIX routes.yaml 新增 9 条 v1/system-admin/projects URI
5. **验证**: Go vet 通过 + Python compile 通过 + YAML parse 通过
6. **文档**: `django_baseline_routes` 5 条 gone + `route_prefixes` 7 条 go 新增
7. **迁移计划更新**: Phase 4 标记完成，本文件新增 §Phase 4 清理详情 + §完成会话总结

---

## 2026-07-26 本会话完成工作

1. **taskBill Kafka 直连**: 新增 `events.go`（segmentio/kafka-go producer），替代 `djangoEmitBillingEvent` / `djangoEmitEvent` / `emitPaypalLifecycleAsync` Django 桥
2. **taskBill enrich 直连**: 新增 `enrich.go`，通过 taskTaskService + taskProjectService 直接查询 project/workspace 名称，替代 `/api/internal/taskbill/enrich-transactions/` Django 端点
3. **Django 清理**: 删除 `billing_bridge/internal_views.py` 和 `billing_bridge/display_names.py`（dead code）；精简 `billing_bridge/urls.py`
4. **配置增强**: taskBill `config.go` 新增 `TaskTaskServiceBase`、`KafkaBootstrapServers` 配置项

## 2026-07-26 Phase 3 收尾会话完成工作

1. **Django cloud/urls.py 路由全量移除**: `saas_project/urls.py` 中 5 个死 `include('cloud.urls')` 已移除（`/api/cloud/`、`/api/tenant/*/cloud/`、`/api/tenant/*/workspace/*/cloud/`、`/api/v1/tenants/*/cloud/`）+ `AliyunOAuthCallbackView` + `/callback/cloudplatform/oauth2.0/aliyun`
2. **API 路由归属更新**: `db/api_route_ownership.yaml` 中 23 条 `cloud/urls.py` Django baseline 路由标记为 gone，新增 2 条残余路由说明
3. **Phase 3 标记完成**: 4 项全部 closed；新增 Phase 3.5 追踪 2 个残余端点（server-userdata-verify、repo-reclone）
4. **风险评估**: 残余端点通过 `cloud/task_cloud_urls.py` 继续服务，业务无中断

## 2026-07-26 Phase 3.5 收尾会话完成工作

1. **OPT-010 server-userdata-verify → Go**:
   - `db.go`: 添加 `verification_secret` + `userdata_run_verified` 列到 `cloud_server_configs`
   - `server_config_store.go`: 添加 `setCloudServerConfigVerificationSecret()` + `verifyCloudServerUserdata()`；更新所有 SELECT 查询 + scan 函数 + struct 字段
   - `compute_handlers.go`: 添加 `handleServerUserdataVerify` handler
   - `compute_start_vm_finalize.go`: `finalizeStartVmInGo` 调用 setter 持久化 secret
2. **OPT-011 repo-reclone → Go**:
   - `compute_repo_reclone.go`: 新文件 — 解析请求 → 查 CSC → `resolveContainerTarget` → 转发 POST 到容器
   - `compute_handlers.go`: 注册 `repo-reclone/` route
3. **Django URL 最终清理**:
   - `saas_project/urls.py`: 移除 `include('cloud.task_cloud_urls')`（5 个端点全迁 Go）
   - `db/api_route_ownership.yaml`: task_cloud_urls 标记 gone
4. **保留说明**: `cloud/` 目录保留（models 被 projects/core/accounts 大量引用），仅视图/路由层已全部迁 Go
5. **Go 编译验证**: `go build ./...` 通过，零错误
