# API 路径契约合规审计

- **日期**: 2026-08-04
- **作者**: claude
- **类型**: 审计 / 头脑风暴
- **状态**: draft
- **契约**: `/api/${serviceName}/${funcName}/${key1}/${value1}/${key2}/${value2}/...`

---

## 审计范围

扫描了全部 17 个 Go 服务 + 1 个 Node.js 服务 (taskSSE) + 1 个 Flask 服务 (mock_run_container)，以及 `taskGateway/routes/routes.yaml` 中的全部网关路由定义（共 ~80 条网关路由规则，派发到 ~300+ 个具体端点路径）。

Django (task2app) 已退役 (OPT-049, 2026-07-30)，无存量 Python API 端点。

---

## 违规分类汇总

| 分类 | 说明 | 影响范围 | 严重度 |
|------|------|---------|--------|
| **A: 深层嵌套 tenant 模式** | `/api/tenant/{tid}/workspace/{wid}/...` 位置参数 | ~40 条网关路由, ~150+ 具体端点 | 🔴 高 |
| **B: `system_admin` 下划线** | 与 `system-admin` 不一致 | ~8 条网关路由 + ~5 条内部路由 | 🟡 中 |
| **C: funcName 深层嵌套** | `/api/auth/oidc/authorize/` 三段 funcName | ~20 条路由 | 🟡 中 |
| **D: 缺 serviceName 的 audience 前缀** | `/api/vendor/`, `/api/admin/` 不含具体服务名 | ~15 条路由 | 🟡 中 |
| **E: 缺 key=value 参数化** | 纯位置参数路径 | todo/SSE/billing 子路径 | 🟡 中 |
| **F: serviceName 归属模糊** | 同一 prefix 路由到多个服务 | `/api/accounts/`, `/api/user/` | 🟠 低 |
| **G: 存量反例匹配** | 与规范文档中的禁止示例完全匹配 | 3 处精确命中 | 🔴 高 |

---

## A: 深层嵌套 tenant 模式（最严重）

> **契约要求**: `/api/cloud/server-config-default/tenant_id/{tid}/`  
> **当前现状**: `/api/tenant/{tid}/cloud/server-config-default/`

### A1. 网关层 — taskCloudService 路由

| 当前网关路由 | 涉及服务 | 建议目标路径 |
|------------|---------|------------|
| `/api/tenant/*/cloud/*` | taskCloudService | `/api/cloud/*` (已并存) |
| `/api/tenant/*/cloud-platform/*` | taskCloudService | `/api/cloud/cloud-platform/*/tenant_id/{tid}` |
| `/api/tenant/*/installed-images/*` | taskCloudService | `/api/cloud/installed-images/tenant_id/{tid}` |
| `/api/tenant/*/workspace/*/cloud/*` | taskCloudService | `/api/cloud/workspace-resources/workspace_id/{wid}` |
| `/api/tenant/*/workspace/*/task/*/cloud/*` | taskCloudService | `/api/cloud/task-resources/task_id/{tid}` |
| `/api/tenant/*/workspaces/*/cloud/*` | taskCloudService | 同上 (复数变体) |
| `/api/tenant/*/feature-params/*` | taskCloudService | `/api/cloud/feature-params/tenant_id/{tid}` |
| `/api/tenant/*/workspace/*/feature-params/*` | taskCloudService | `/api/cloud/feature-params/workspace_id/{wid}` |
| `/api/tenant/*/budget-permissions/*` | taskCloudService | `/api/cloud/budget-permissions/tenant_id/{tid}` |
| `/api/tenant/*/workspaces/*/model-budget-defaults/*` | taskCloudService | `/api/cloud/model-budget-defaults/workspace_id/{wid}` |
| `/api/tenant/*/workspace/*/todos/*/model-budgets/*` | taskCloudService | `/api/cloud/model-budgets/task_id/{tid}` |

### A2. 网关层 — SSE 路由

| 当前网关路由 | 建议目标路径 |
|------------|------------|
| `/api/tenant/*/workspace/*/task/*/cloud/server-startup-status-sse*` | `/api/sse/server-startup-status/task_id/{tid}` |
| `/api/tenant/*/billing/recharge-events-sse*` | `/api/sse/recharge-events/tenant_id/{tid}` |
| `/api/tenant/*/workspace/*/work-panel-events-sse*` | `/api/sse/work-panel-events/workspace_id/{wid}` |

### A3. 网关层 — Container Gateway 路由

| 当前网关路由 | 建议目标路径 |
|------------|------------|
| `/api/tenant/*/workspace/*/task/*/cloud/compute/relay-to-trae/*` | `/api/container/relay-to-trae/{action}/task_id/{tid}` |
| `/api/tenant/*/workspace/*/task/*/cloud/compute/mock-run-container/*` | `/api/container/mock-run/{action}/task_id/{tid}` |
| `/api/tenant/*/workspace/*/task/*/cloud/compute/container-*` | `/api/container/{action}/task_id/{tid}` |
| `/api/tenant/*/workspace/*/task/*/cloud/server-container-token*` | `/api/container/server-token/task_id/{tid}` |

### A4. 网关层 — 项目/任务/租户路由

| 当前网关路由 | 涉及服务 | 建议目标路径 |
|------------|---------|------------|
| `/api/tenant/*/projects/*` | taskProjectService | `/api/projects/tenant_id/{tid}` |
| `/api/tenant/*/workspaces/*` | taskProjectService | `/api/projects/workspaces/tenant_id/{tid}` |
| `/api/tenant/*/daydaymoney/*` | taskProjectService | `/api/projects/daydaymoney/tenant_id/{tid}` |
| `/api/tenant/*/workspace-access/*` | taskProjectService | `/api/projects/workspace-access/tenant_id/{tid}` |
| `/api/tenant/*/deliverable-systems/*` | taskProjectService | `/api/projects/deliverable-systems/tenant_id/{tid}` |
| `/api/tenant/*/progress-systems/*` | taskProjectService | `/api/projects/progress-systems/tenant_id/{tid}` |
| `/api/tenant/*/manage-progress-column/*` | taskProjectService | `/api/projects/manage-progress-column/tenant_id/{tid}` |
| `/api/tenant/*/manage-deliverable-system/*` | taskProjectService | `/api/projects/manage-deliverable-system/tenant_id/{tid}` |
| `/api/tenant/*/settings/*` | taskProjectService | `/api/projects/settings/tenant_id/{tid}` |
| `/api/tenant/*/work-panel/*` | taskProjectService | `/api/projects/work-panel/tenant_id/{tid}` |
| `/api/tenant/*/accounts/members/*` | taskTenantService | `/api/tenant/members/tenant_id/{tid}` |
| `/api/tenant/*/accounts/groups/*` | taskTenantService | `/api/tenant/groups/tenant_id/{tid}` |
| `/api/tenant/*/accounts/companies/*` | taskTenantService | `/api/tenant/companies/tenant_id/{tid}` |
| `/api/tenant/*/workspace/*/todos/*` | taskTaskService | `/api/tasks/todos/workspace_id/{wid}` |
| `/api/tenant/*/tasks/*` | taskTaskService | `/api/tasks/task/tenant_id/{tid}` |
| `/api/tenant/*/workspace/*/task-detail/*/ai-comments/*` | taskAIComment | `/api/ai-comment/task-detail/task_id/{tid}` |
| `/api/tenant/*/workspace/*/task/*/container-agent-comments/*` | taskAIComment | `/api/ai-comment/container-agent/task_id/{tid}` |
| `/api/tenant/*/billing/*` | taskBill | `/api/billing/tenant_id/{tid}` |
| `/api/tenant/*/gitlab-oauth-connection/*` | taskGitOauth | `/api/git-oauth/tenant-connection/tenant_id/{tid}` |

### A5. 内部 (internal) — taskCloudService 子路由

以下为 Go 代码中 `handleCloudTenantRoutes` 的 switch-case 分发，通过 `/api/tenant/{tid}/cloud/{action}` 到达。这些虽为旧的路径模式，但在 `/api/cloud/` 和 `/api/system-admin/cloud/` 下已有等价路由并存。

| 内部子路径 | 等价新路径 |
|----------|----------|
| `cloud-platform-authorizations` | `/api/cloud/cloud-platform-authorizations` (内部) |
| `regions` | `/api/cloud/regions` |
| `images` | `/api/cloud/images` |
| `server-images` | `/api/cloud/server-images` |
| `server-config-default` | `/api/cloud/server-config-default` |
| `userdata-templates` | `/api/cloud/userdata-templates` |
| `ai-model-authorizations` | `/api/cloud/ai-model-authorizations` |
| `start-server` | `/api/cloud/start-server` |
| `vpcs`, `vswitches`, `security-groups` | `/api/cloud/vpcs` 等 |

---

## B: `system_admin` vs `system-admin` 不一致

| 当前路径 | 问题 | 建议 |
|---------|------|------|
| `/api/system_admin/license-agreement/*` | 下划线 | `/api/system-admin/license-agreement/*` |
| `/api/system_admin/privacy-policy/*` | 下划线 | `/api/system-admin/privacy-policy/*` |
| `/api/system_admin/orders/*` | 下划线 | `/api/system-admin/orders/*` |
| `/api/system_admin/gitlab-regions/*` | 下划线 | `/api/system-admin/gitlab-regions/*` |
| `/api/system_admin/users/{uid}/recharges/*` | 下划线 + A类嵌套 | `/api/billing/user-recharges/user_id/{uid}` |
| `/api/system_admin/deliverable-systems/*` | 下划线 | `/api/system-admin/deliverable-systems/*` |
| `/api/system_admin/projects/deliverable-systems/*` | 下划线 | `/api/system-admin/deliverable-systems/projects/*` |
| `/api/system_admin/accounts/admin/tenant-options/*` | 下划线 | `/api/tenant/admin-options` |

**注意**: 部分路由同时存在 `system_admin` 和 `system-admin` 两个版本（如 `deliverable-systems`），增加维护负担。

---

## C: funcName 深层嵌套

> **契约要求**: `/api/${serviceName}/${funcName}/...` — funcName 为一段  
> **当前现状**: `/api/auth/oidc/authorize/` — funcName 为三段

| 当前路径 | 段数 | 建议 |
|---------|------|------|
| `/api/auth/sso/exchange/` | 3 | `/api/auth/sso-exchange/` |
| `/api/auth/oidc/authorize/` | 3 | `/api/oidc/authorize/` (已并存于 OIDC 路由) |
| `/api/auth/oidc/callback/` | 3 | `/api/oidc/callback/` |
| `/api/auth/wechat/login/` | 3 | `/api/auth/wechat-login/` |
| `/api/auth/wechat/callback/` | 3 | `/api/auth/wechat-callback/` |
| `/api/vendor/auth/register/` | 3 | `/api/vendor/auth-register/` |
| `/api/vendor/auth/login/` | 3 | `/api/vendor/auth-login/` |
| `/api/admin/auth/login/` | 3 | `/api/admin/auth-login/` |
| `/api/accounts/github/oauth/start/` | 4 🔴 | `/api/git-oauth/github-start/` |
| `/api/accounts/github/oauth/callback/` | 4 🔴 | `/api/git-oauth/github-callback/` |
| `/api/accounts/gitlab/oauth/start/` | 4 🔴 | `/api/git-oauth/gitlab-start/` |
| `/api/accounts/gitlab/oauth/callback/` | 4 🔴 | `/api/git-oauth/gitlab-callback/` |
| `/api/accounts/github/app/start/` | 4 🔴 | `/api/git-oauth/github-app-start/` |
| `/api/accounts/gitlab/app/start/` | 4 🔴 | `/api/git-oauth/gitlab-app-start/` |
| `/api/internal/github/oauth/refresh/` | 5 🔴 | `/api/internal/git-oauth/github-refresh/` |
| `/api/internal/github/oauth/access-for-user/` | 5 🔴 | `/api/internal/git-oauth/github-access-for-user/` |
| `/api/internal/github/oauth/token-use-report/` | 5 🔴 | `/api/internal/git-oauth/github-token-report/` |
| `/api/internal/github/oauth/user-credential/delete/` | 5 🔴 | `/api/internal/git-oauth/github-credential-delete/` |
| `/api/internal/github/oauth/user-credential/user-ids/` | 5 🔴 | `/api/internal/git-oauth/github-credential-user-ids/` |
| `/api/internal/github/oauth/user-credential/summary-for-user/` | 5 🔴 | `/api/internal/git-oauth/github-credential-summary/` |
| （GitLab 内部路由同上模式 × 7） | 5 🔴 | 同模式缩短 |

---

## D: 缺 serviceName 的 audience 前缀

| 当前路径 | 问题 | 建议 |
|---------|------|------|
| `/api/vendor/cloud-platform-credentials/` | 无具体 serviceName | `/api/cloud/vendor-credentials/` 或 `/api/ai-provider/vendor-credentials/` |
| `/api/vendor/cloud-server-images/` | 同上 | `/api/cloud/vendor-images/` |
| `/api/vendor/container-images/` | 同上 | `/api/ai-provider/vendor-container-images/` |
| `/api/vendor/image-groups/` | 同上 | `/api/ai-provider/vendor-image-groups/` |
| `/api/vendor/userdata-templates/` | 同上 | `/api/cloud/vendor-userdata-templates/` |
| `/api/vendor/auth/register/` | taskAiProvider 非 cloud | `/api/ai-provider/vendor-auth-register/` |
| `/api/vendor/auth/login/` | 同上 | `/api/ai-provider/vendor-auth-login/` |
| `/api/vendor/auth/me/` | 同上 | `/api/ai-provider/vendor-me/` |
| `/api/admin/vendors/` | 无具体 serviceName | `/api/ai-provider/admin-vendors/` |
| `/api/admin/container-images/` | 同上 | `/api/ai-provider/admin-container-images/` |
| `/api/admin/cloud-server-images/` | 同上 | `/api/ai-provider/admin-cloud-images/` |
| `/api/admin/userdata-templates/` | 同上 | `/api/ai-provider/admin-userdata-templates/` |
| `/api/public/catalog/` | 同上 | `/api/ai-provider/public-catalog/` |
| `/api/public/userdata-templates/` | 同上 | `/api/ai-provider/public-userdata-templates/` |
| `/api/personal/feature-params-configs/` | personal 非标准 serviceName | `/api/cloud/personal-feature-params/` |

---

## E: 缺 key=value 参数化的路径

| 当前路径 | 问题 | 建议 |
|---------|------|------|
| `/api/user/{userId}/profile/referral-stats/` | `{userId}` 位置参数 | `/api/referral/stats/user_id/{userId}` |
| `/api/user/{user_id}/accounts/users/me/` | 位置参数 | `/api/accounts/me/user_id/{user_id}` (legacy 兼容) |
| `/api/system-admin/users/{uid}/referral-performance/` | 位置参数 | `/api/referral/performance/user_id/{uid}` |
| `/api/system_admin/users/{uid}/recharges/` | 位置参数 + 下划线 (已在规范文档中列为反例) | `/api/billing/user-recharges/user_id/{uid}` |

---

## F: serviceName 归属模糊（同 prefix 多服务）

| Prefix | 路由到的服务 | 数量 |
|--------|------------|------|
| `/api/accounts/*` | taskAuth, taskGitOauth, taskReferral | 3 服务 |
| `/api/user/*` | taskAuth, taskGitOauth, taskTaskService, taskReferral | 4 服务 |
| `/api/tenant/*` | taskCloudService, taskProjectService, taskTenantService, taskTaskService, taskAIComment, taskAIEndPoint, taskBill, taskGitOauth, taskCredentialService | **9 服务** 🔴 |

`/api/tenant/*` 是最严重的问题：9 个服务共用一个 prefix，网关依赖 `/*/*/*` 多级通配符匹配来区分，这正是规范文档中指出的"APISIX 通配符路由优先级冲突"的根因。

---

## G: 与规范文档禁止示例的精确匹配

规范文档 (memory: `api-url-path-design-convention`) 列出了 4 个反例。其中 3 个在当前代码中**仍然存在**：

| 反例 | 当前代码 | 文件位置 |
|------|---------|---------|
| ❌ `/api/tenant/{tid}/workspaces/{wid}/cloud/platforms/` | `/api/tenant/*/workspaces/*/cloud/platforms` | `routes.yaml:1090` + `cloud_handlers.go:973` |
| ❌ `/api/tenant/{tid}/cloud/server-config-default/` | `/api/tenant/*/cloud` → `handleCloudTenantRoutes` → `case "server-config-default"` | `cloud_handlers.go:336` |
| ❌ `/api/system_admin/users/{uid}/recharges/` | `/api/system_admin/users/{uid}/recharges/` | `handlers.go:71` — **精确匹配！** |

---

## 合规端点（已遵守契约的路径）

以下端点已经符合 `/api/${serviceName}/${funcName}/...` 模式：

| 路径模式 | serviceName | funcName | 备注 |
|---------|------------|----------|------|
| `/api/cloud/*` | cloud | (动态) | taskCloudService 现行模式 |
| `/api/oidc/*` | oidc | authorize, token, jwks, userinfo, endsession | ✅ 完美合规 |
| `/api/health/*` | health | — | 基础设施端点 |
| `/api/projects/*` | projects | — | taskProjectService (与 tenant/projects 并存) |
| `/api/tasks/*` | tasks | — | taskTaskService (与 tenant/tasks 并存) |
| `/api/billing/wechat/notify/` | billing | wechat-notify | ✅ |
| `/api/billing/profitsharing/notify/` | billing | profitsharing-notify | ✅ |
| `/api/kyc/me/` | kyc | me | ✅ |
| `/api/git-identities/*` | git-identities | — | taskTaskService |
| `/api/sub-token-providers/*` | sub-token-providers | — | taskCloudService |
| `/api/recommended-llm-providers/*` | recommended-llm-providers | — | taskCloudService |
| `/api/license-agreement/public/current/` | license-agreement | public-current | 🟡 可接受 |
| `/api/privacy-policy/public/current/` | privacy-policy | public-current | 🟡 可接受 |
| `/api/public/product-pricing/` | public | product-pricing | 🟡 可接受 |
| `/api/public/resource-pricing/` | public | resource-pricing | 🟡 可接受 |
| `/api/public/system-feature-policy/` | public | system-feature-policy | 🟡 可接受 |
| `/api/system-admin/dashboard/` | system-admin | dashboard | ✅ |
| `/api/system-admin/users/` | system-admin | users | ✅ |
| `/api/system-admin/projects/` | system-admin | projects | ✅ |
| `/api/system-admin/cloud/` | system-admin | cloud | ✅ |
| `/api/system-admin/resource-pricing/` | system-admin | resource-pricing | ✅ |
| `/api/internal/*` | internal | (功能名) | ✅ 内部 API 统一 prefix |

---

## 统计摘要

| 指标 | 数值 |
|------|------|
| 扫描的服务总数 | 20 (17 Go + 1 Node + 1 Flask + 1 网关配置) |
| 网关路由规则总数 | ~80 |
| 唯一端点路径估算 | ~300+ |
| 严重违规 (A+G 类) | ~55 条路由 |
| 中等违规 (B+C+D 类) | ~50 条路由 |
| 低优先违规 (E+F 类) | ~15 条路由 |
| 已合规端点 | ~40 条路由 |
| 与规范文档反例精确匹配 | 3 处 |

---

## 建议优先级

### P0 — 新端点强制合规（可立即执行）
所有**新增加**的 API 端点必须使用 `/api/${serviceName}/${funcName}/key/value/...` 模式，CI 添加检查。

### P1 — Gateway 层并存桥接（短期）
为新路径添加网关路由，与旧路径并存。Go 服务通过 `mux.HandleFunc` 同时注册新旧两种路径，前端逐步迁移。

### P2 — 消除 `/api/tenant/{tid}/...` 嵌套（中期）
这是最影响路由优先级冲突的类别。迁移策略：
1. 每服务新增 `/api/${serviceName}/${funcName}/tenant_id/{tid}/...` 路由
2. Gateway 添加新路由，旧路由保留
3. 前端逐批迁移
4. 旧路由标记 `[DEPRECATED]`，在完全无流量后移除

### P3 — 统一 `system_admin` → `system-admin`（中期）
纯重命名，Go 代码内 `mux.HandleFunc` 同时注册两版本，Gateway 路由同时配置，逐步迁移。

### P4 — funcName 扁平化 + serviceName 归属明确（长期）
涉及路径语义变化，需要跨团队协调。

---

## 业务意图 → 事件对照

本次为纯审计，无新增业务意图，无对应领域事件。
