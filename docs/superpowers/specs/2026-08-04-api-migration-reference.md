# API 路径迁移对照表 (FINAL — 不向后兼容)

> 生成日期: 2026-08-04 | 作者: claude  
> 契约: `/api/${serviceName}/${funcName}/${key1}/${value1}/${key2}/${value2}/...`  
> ⚠️ **旧路径已全部删除，仅保留新契约路径。前端须同步更新。**

---

## 修改的服务

| 服务 | 状态 | 变更类型 |
|------|------|---------|
| shareLib/gatewayauth | 🆕 新增 shared helper | `ParseConventionPath()` |
| taskCloudService | ✏️ 重写路由 | 移除 `/api/tenant/` `/api/system_admin/`，统一到 `/api/cloud/` |
| taskProjectService | ✏️ 重写路由 | 移除 `/api/tenant/` 嵌套，统一到 `/api/projects/` |
| taskTenantService | ✏️ 别名 | 移除 `system_admin` 下划线路径 |
| taskTaskService | ✏️ 重写路由 | 移除 tenant 嵌套路径，保留 `/api/tasks/` |
| taskBill | ✏️ 别名 | 移除 `system_admin` 下划线路径，保留 `system-admin` |
| taskGitOauth | ✏️ 扁平化 | 全部迁移到 `/api/git-oauth/` prefix |
| taskAiProvider | ✏️ 扁平化 | 全部迁移到 `/api/ai-provider/` prefix |
| taskGateway/routes.yaml | ✏️ 重写 | 移除 ~60 条旧路由，仅保留 convention prefixes |
| taskSSE/server.mjs | ✏️ 重写 | SSE 路径迁移到 `/api/sse/` |

---

## 最终路径清单

### taskCloudService — `/api/cloud/*`

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/cloud/tenant_id/{tid}/...` | * | 租户级云操作 (regions, images, server-images, server-config-default, userdata-templates, cloud-platform-authorizations, vpcs, oauth-tokens, ai-model-authorizations, start-server, installed-images, budget-permissions, feature-params, oauth/..., toggle-active, active-list) |
| `/api/cloud/...workspace_id/{wid}/...` | * | 工作区级云操作 (platforms, feature-params, model-budget-defaults) |
| `/api/cloud/...task_id/{tid}/...` | * | 任务级云操作 (server-config, server-container-token, compute/*) |

### taskProjectService — `/api/projects/*`

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/projects/tenant_id/{tid}` | GET/POST | 项目列表/创建 |
| `/api/projects/tenant_id/{tid}/{pid}` | GET/PUT/DELETE | 项目 CRUD |
| `/api/projects/workspaces/tenant_id/{tid}` | GET/POST | 工作区列表/创建 |
| `/api/projects/workspaces/tenant_id/{tid}/{wid}` | GET/PUT/DELETE | 工作区 CRUD |

### taskTenantService — `/api/tenant/*`

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/tenant/members/tenant_id/{tid}` | GET/POST | 成员管理 |
| `/api/tenant/groups/tenant_id/{tid}` | GET/POST | 组管理 |
| `/api/tenant/companies/tenant_id/{tid}` | GET/PUT | 公司详情 |

### taskTaskService — `/api/tasks/*`

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/tasks/{taskId}` | GET | 任务详情 |
| `/api/tasks/{taskId}/feature-params` | GET/PUT | 特性参数 |

### taskGitOauth — `/api/git-oauth/*`

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/git-oauth/github-start/` | GET | GitHub OAuth 发起 |
| `/api/git-oauth/github-callback/` | GET | GitHub OAuth 回调 |
| `/api/git-oauth/gitlab-start/` | GET | GitLab OAuth 发起 |
| `/api/git-oauth/gitlab-callback/` | GET | GitLab OAuth 回调 |
| `/api/git-oauth/providers/` | GET | OAuth 提供商列表 |
| `/api/git-oauth/user-app-connection/` | * | 用户应用连接 |
| `/api/git-oauth/tenant-connection/` | * | 租户连接 |

### taskAiProvider — `/api/ai-provider/*`

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/ai-provider/sso-exchange/` | POST | SSO 交换 |
| `/api/ai-provider/oidc-authorize/` | GET | OIDC 授权 |
| `/api/ai-provider/oidc-callback/` | GET | OIDC 回调 |
| `/api/ai-provider/vendor-auth-register/` | POST | 厂商注册 |
| `/api/ai-provider/vendor-auth-login/` | POST | 厂商登录 |
| `/api/ai-provider/vendor-me/` | GET | 厂商当前用户 |
| `/api/ai-provider/public-catalog/` | GET | 公开目录 |
| `/api/ai-provider/public-userdata-templates/` | GET | 公开 UserData 模板 |
| `/api/ai-provider/vendor-cloud-credentials/` | * | 厂商云凭证代理 |
| `/api/ai-provider/vendor-container-images/` | * | 厂商容器镜像 |
| `/api/ai-provider/admin-container-images/` | * | 管理后台容器镜像 |
| `/api/ai-provider/admin-cloud-server-images/` | * | 管理后台云服务器镜像 |
| `/api/ai-provider/vendor-cloud-server-images/` | * | 厂商云服务器镜像 |
| `/api/ai-provider/vendor-userdata-templates/` | * | 厂商 UserData 模板 |

### taskSSE — `/api/sse/*`

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/sse/server-startup-status/tenant_id/{tid}/workspace_id/{wid}/task_id/{taskId}` | GET (SSE) | 服务器启动状态流 |
| `/api/sse/recharge-events/tenant_id/{tid}` | GET (SSE) | 充值事件流 |
| `/api/sse/work-panel-events/tenant_id/{tid}/workspace_id/{wid}` | GET (SSE) | 工作面板事件流 |

### 其他 convention prefixes (已合规)

| Prefix | 服务 | 说明 |
|--------|------|------|
| `/api/oidc/*` | taskAuth | OIDC Provider |
| `/api/health/*` | taskAuth | 健康检查 |
| `/api/kyc/*` | taskAuth | KYC 认证 |
| `/api/accounts/*` | taskAuth | 账户/认证 |
| `/api/auth/*` | taskAuth | 登录/注册 |
| `/api/billing/*` | taskBill | 计费 |
| `/api/system-admin/*` | 各服务 | 管理后台 |
| `/api/git-identities/*` | taskTaskService | Git 身份 |
| `/api/referral/*` | taskReferral | 推荐系统 |
| `/api/container/*` | taskContainerGateway | 容器代理 |
| `/api/ai-comment/*` | taskAIComment | AI 注释 |
| `/api/internal/*` | 网关拒绝 | 内部 API (直连) |

---

## 删除的旧路径 (总数 ~100+)

| 旧 prefix 模式 | 说明 |
|--------------|------|
| `/api/tenant/*/cloud/*` | 全部迁移到 `/api/cloud/*` |
| `/api/tenant/*/workspace/*/cloud/*` | 迁移到 `/api/cloud/...workspace_id/{wid}` |
| `/api/tenant/*/workspace/*/task/*/cloud/*` | 迁移到 `/api/cloud/...task_id/{tid}` |
| `/api/tenant/*/cloud-platform/*` | 迁移到 `/api/cloud/cloud-platform/tenant_id/{tid}` |
| `/api/tenant/*/projects/*` | 迁移到 `/api/projects/tenant_id/{tid}` |
| `/api/tenant/*/workspaces/*` | 迁移到 `/api/projects/workspaces/tenant_id/{tid}` |
| `/api/tenant/*/accounts/members` | 迁移到 `/api/tenant/members/tenant_id/{tid}` |
| `/api/tenant/*/accounts/groups` | 迁移到 `/api/tenant/groups/tenant_id/{tid}` |
| `/api/tenant/*/accounts/companies` | 迁移到 `/api/tenant/companies/tenant_id/{tid}` |
| `/api/tenant/*/workspace/*/todos` | 迁移到 `/api/tasks/{taskId}` |
| `/api/tenant/*/billing/*` | 迁移到 `/api/billing/tenant_id/{tid}` |
| `/api/tenant/*/workspace/*/task-detail/*/ai-comments` | 迁移到 `/api/ai-comment/*` |
| `/api/tenant/*/workspace/*/task/*/cloud/compute/relay-to-trae/*` | 迁移到 `/api/container/*` |
| `/api/tenant/*/workspace/*/task/*/cloud/compute/mock-run-container/*` | 迁移到 `/api/container/*` |
| `/api/tenant/*/workspace/*/task/*/cloud/compute/container-*` | 迁移到 `/api/container/*` |
| `/api/accounts/github/oauth/*` | 迁移到 `/api/git-oauth/github-*` |
| `/api/accounts/gitlab/oauth/*` | 迁移到 `/api/git-oauth/gitlab-*` |
| `/api/auth/sso/exchange/` | 迁移到 `/api/ai-provider/sso-exchange/` |
| `/api/vendor/*` | 迁移到 `/api/ai-provider/vendor-*` 或 `/api/cloud/*` |
| `/api/admin/*` (ai-provider) | 迁移到 `/api/ai-provider/admin-*` |
| `/api/system_admin/*` | 统一为 `/api/system-admin/*` |
| `/api/user/*/accounts/users/me` | 删除 (直接使用 `/api/accounts/users/me/`) |
| `/api/user/*/profile/referral-stats` | 迁移到 `/api/referral/stats/user_id/{uid}` |
| `/api/tenant/*/workspace/*/task/*/cloud/server-startup-status-sse` | 迁移到 `/api/sse/server-startup-status/...` |
| `/api/tenant/*/billing/recharge-events-sse` | 迁移到 `/api/sse/recharge-events/...` |
| `/api/tenant/*/workspace/*/work-panel-events-sse` | 迁移到 `/api/sse/work-panel-events/...` |

## 统计

| 指标 | 数值 |
|------|------|
| 修改的 Go 服务 | 7 |
| 修改的 gateway 路由 | ~70 条删除 + 8 条新增 |
| 删除的旧路径总数 | ~100+ |
| 新 convention prefixes | 10 (`cloud`, `projects`, `tasks`, `tenant`, `git-oauth`, `ai-provider`, `sse`, `billing`, `container`, `ai-comment`) |
| 构建通过 | 8/8 Go 服务 |
| vet 通过 | 7/7 Go 服务 |
