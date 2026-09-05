# 设计文档：项目详情页 OAuth 授权状态未正确反映

**日期**: 2026-06-28
**问题**: Git 仓库已完成 OAuth 授权，但项目详情页仍显示"OAuth 授权"按钮

---

## 1. 问题分析

### 1.1 用户报告

访问 `http://183.250.1.132:4000/tenant/850256677331562496/projects/858546008673890304/`，Git 仓库已经通过 OAuth 授权（gitOauth 中 `bind_status="active"`），但页面仍显示"OAuth 授权"按钮。

### 1.2 根因分析（三个层次）

#### 根因 1（主要）：`ProjectDetail.vue` 从未检查授权状态

**文件**: `task2app/front_project/app/src/views/ProjectDetail.vue:257-262`

```javascript
const shouldShowRepoOAuthButton = (repoUrl) => {
  touchRepoOAuthButtons()
  const info = resolveRepoOAuthProviderInfo(repoUrl)
  if (!info) return false
  return Boolean(resolveRepoOAuthAuthorizeUrl(info.provider))
}
```

该函数**仅检查**：
1. repo URL 是否匹配已知 OAuth provider（从 catalog 查询）
2. 是否能解析出 OAuth authorize URL

**完全不检查**：
- 用户是否已完成授权（`token_status`）
- 授权是否失败（`token_error`）
- 仓库是否公开无需授权（`not_applicable`）

**对比 `CreateProject.vue:218-227`（正确实现）**：

```javascript
const shouldShowRepoOAuthButton = (repoUrl) => {
  touchRepoOAuthButtons()
  const info = resolveRepoOAuthProviderInfo(repoUrl)
  if (!info) return false
  const key = String(repoUrl || '').trim()
  const ts = gitRepoRowTokenStatus.value[key]
  // token_available + 不可访问 = 仓库权限问题，不显示 OAuth 按钮
  if (ts === 'token_available') return false
  return Boolean(resolveRepoOAuthAuthorizeUrl(info.provider))
}
```

`CreateProject.vue` 正确地在 `token_available` 时隐藏按钮（因为已授权），`token_error` 时显示"重试"。

#### 根因 2：后端项目详情 API 不返回授权状态

**文件**: `task2app/Saas_project/projects/serializers/project_serializer.py:128`

```python
ret['git_repos'] = [pr.repo_url for pr in instance.project_repos.all()]
```

`git_repos` 仅返回 URL 字符串列表，**不包含** `token_status`、`oauth_provider` 等字段。前端要获取授权状态必须额外调用其他 API。

已有的授权状态检查端点：
- `GET /api/tenant/{tenantId}/projects/validate-git-repo/?url=...` — 单 repo 校验（`CreateProject.vue` 使用）
- `GET /api/tenant/{tenantId}/projects/{projectId}/repo-access-check/` — 项目主 repo 检查（任务创建模态框使用）

#### 根因 3：OAuth 回调后无重新校验

**文件**: `task2app/front_project/app/src/utils/gitSiteOAuthCallbackUtils.js`

OAuth 回调返回项目详情页时（URL 带 `?gitlab=ok`），`gitSiteOAuthCallbackUtils.js` 处理回调消息，但 `ProjectDetail.vue` **不触发** repo 访问状态的重新检查。

对比 `CreateProject.vue:605-611`，它在 OAuth 回调后**重新调用** `validateGitRepoRow` 刷新状态。

### 1.3 补充发现：OAuth 回调重定向配置正确

经检查 `conf/frontend/vue/config.yaml`：
```yaml
publicBaseUrl: http://183.250.1.132:4000
```

`GitlabOAuthStartFromGatewayView:287` 从 `settings.TASK2APP_FRONTEND_BASE` 读取 `feb`，该值正确解析为 `http://183.250.1.132:4000`。重定向链路无问题。

---

## 2. 设计方案

### 2.1 核心思路

**后端增强**：在项目详情 API 响应中增加每个 repo 的 `token_status`，让前端一次性获取全部授权状态。

**前端增强**：
1. 使用 `token_status` 条件性显示 OAuth 按钮（对齐 `CreateProject.vue` 已有逻辑）
2. OAuth 回调后重新拉取项目详情刷新状态

### 2.2 后端改动

#### 2.2.1 项目详情 API 增加 `git_repos_status` 字段

**文件**: `task2app/Saas_project/projects/serializers/project_serializer.py`

在 `to_representation()` 中增加：

```python
# 现有
ret['git_repos'] = [pr.repo_url for pr in instance.project_repos.all()]

# 新增：每个 repo 的 token 状态
from projects.services.repo_access_token_resolver import resolve_repo_access_tokens

git_repos_status = []
for pr in instance.project_repos.all():
    tokens = resolve_repo_access_tokens(self.context['request'].user, pr.repo_url)
    git_repos_status.append({
        'repo_url': pr.repo_url,
        'token_status': tokens['token_status'],       # not_bound | token_error | token_available | not_applicable
        'oauth_provider': tokens.get('oauth_provider', ''),
        'oauth_service_provider': tokens.get('oauth_service_provider', ''),
    })
ret['git_repos_status'] = git_repos_status
```

**性能考量**：`resolve_repo_access_tokens` 对每个 repo 调用 gitOauth 的 `summary-for-user/` API。大多数项目只有 1-2 个 repo，影响可控。如需优化，可先调用一次 `summary-for-user/` 获取用户所有连接，再按 repo origin 匹配——此优化可后续迭代。

#### 2.2.2 复用已有 `resolve_repo_access_tokens` 服务

**文件**: `task2app/Saas_project/projects/services/repo_access_token_resolver.py`

此服务已实现完整的 token 状态判定逻辑，返回 `token_status` + `oauth_provider` + `oauth_service_provider`，无需修改。

### 2.3 前端改动

#### 2.3.1 `ProjectDetail.vue` — 使用 `token_status` 控制按钮

**文件**: `task2app/front_project/app/src/views/ProjectDetail.vue`

修改 `shouldShowRepoOAuthButton`（对齐 `CreateProject.vue` 逻辑）：

```javascript
// 新增：存储每个 repo URL 的 token_status
const gitRepoTokenStatus = ref({})

// 修改：从项目详情响应中初始化
const fetchProjectDetail = async () => {
  // ... 现有逻辑 ...
  const data = await response.json()
  // 初始化 token status
  if (data.git_repos_status) {
    const statusMap = {}
    data.git_repos_status.forEach(s => {
      statusMap[normalizeRepoUrlKey(s.repo_url)] = s.token_status
    })
    gitRepoTokenStatus.value = statusMap
  }
  // ... 现有逻辑 ...
}

// 修改：考虑 token_status
const shouldShowRepoOAuthButton = (repoUrl) => {
  touchRepoOAuthButtons()
  const info = resolveRepoOAuthProviderInfo(repoUrl)
  if (!info) return false
  const key = normalizeRepoUrlKey(repoUrl)
  const ts = gitRepoTokenStatus.value[key]
  if (ts === 'token_available') return false   // 已授权，不显示按钮
  if (ts === 'not_applicable') return false    // 无需授权，不显示按钮
  return Boolean(resolveRepoOAuthAuthorizeUrl(info.provider))
}

// 修改：token_error 时显示"重试"
const repoOAuthButtonLabel = (repoUrl) => {
  const key = normalizeRepoUrlKey(repoUrl)
  if (repoOAuthActionLoadingByUrl.value[key]) return '跳转中...'
  if (gitRepoTokenStatus.value[key] === 'token_error') return '重试'
  return 'OAuth 授权'
}
```

#### 2.3.2 OAuth 回调后刷新项目详情

**文件**: `task2app/front_project/app/src/views/ProjectDetail.vue`

在 `onMounted` 或已有的回调处理逻辑中：

```javascript
// OAuth 回调后重新拉取项目详情（刷新 token_status）
const handleOAuthCallback = async () => {
  const result = await applyOAuthCallbackFromRoute(route, router, {
    onSuccess: (msg) => { /* toast 由 gitSiteOAuthCallbackUtils 处理 */ },
  })
  if (result && result.severity === 'success') {
    // 重新拉取项目详情以刷新授权状态
    await fetchProjectDetail()
  }
}
```

#### 2.3.3 可选：显示授权状态标识

在 repo URL 旁增加状态图标（绿色勾 = 已授权，警告 = 需授权），提升用户体验。此为增强项，可作为第二优先级。

### 2.4 不变更的部分

- **gitOauth 服务**：授权存储、回调、token 签发逻辑均正确，无需修改
- **taskGateway 路由**：OAuth start/callback 路由正确，无需修改
- **OAuth 回调重定向**：`TASK2APP_FRONTEND_BASE` 已正确配置，无需修改
- **`CreateProject.vue`**：已有正确实现，作为参考模板

---

## 3. 领域概念清单

| 概念 | 类型 | 说明 |
|------|------|------|
| **GitOAuth 授权** | Bounded Context | gitOauth 服务管理的 OAuth 授权生命周期 |
| **TokenStatus** | Value Object | 授权状态：`not_bound` / `token_error` / `token_available` / `not_applicable` |
| **GitOAuthAppUserCredential** | Entity | 用户 OAuth 凭据（gitOauth 中），`bind_status` 控制生命周期 |
| **ProjectRepo** | Entity | 项目关联的 Git 仓库 URL |
| **RepoAccessTokenResolver** | Domain Service | 解析用户对特定 repo 的 token 可用性 |

---

## 4. 价值流影响分析

### 4.1 受影响的现有流

| 价值流 | 步骤 | 影响 |
|--------|------|------|
| 项目管理 - 项目详情 | 查看项目详情 | API 响应增加 `git_repos_status` 字段 |
| 项目管理 - 项目详情 | OAuth 授权状态展示 | 前端按钮展示逻辑从"只看 provider"升级为"看 token_status" |
| 用户认证 - Git OAuth 授权 | 授权回调 | 回调后触发项目详情刷新 |

### 4.2 字段影响

| 字段 | 变更 |
|------|------|
| `django.projects_projectrepo.repo_url` | 读（已有） |
| `gitOauth.api_gitoauthappusercredential.bind_status` | 读（已有，通过 `resolve_repo_access_tokens`） |
| `django.projects_project.git_repos_status` | **新增**（JSON 数组，在 API 响应中） |

### 4.3 测试影响

- 新增测试：验证项目详情 API 返回 `git_repos_status`
- 新增 Playwright E2E：验证已授权 repo 不显示"OAuth 授权"按钮
- 已有测试：`CreateProject.vue` 相关测试不受影响（逻辑已正确）

---

## 5. 任务分解（概要）

| # | 任务 | 涉及文件 | 预估 |
|---|------|---------|------|
| 1 | 后端：项目详情 API 增加 `git_repos_status` | `project_serializer.py` | 中 |
| 2 | 后端：单元测试 | `tests/` | 中 |
| 3 | 前端：`ProjectDetail.vue` 使用 `token_status` 控制按钮 | `ProjectDetail.vue` | 中 |
| 4 | 前端：OAuth 回调后刷新状态 | `ProjectDetail.vue` | 小 |
| 5 | E2E：验证授权状态正确展示 | `playwright/` | 中 |

---

## 6. 备选方案（已否决）

### 方案 B：纯前端修复（不修改后端 API）

前端在 `ProjectDetail.vue` 挂载后对每个 repo URL 调用 `validate-git-repo/` API。

**否决原因**：N+1 API 调用模式，性能差；与"项目详情"语义不符（校验是独立操作）。

### 方案 C：新增独立 endpoint

新增 `GET /api/tenant/{tenantId}/projects/{projectId}/repos-token-status/`。

**否决原因**：增加 API 面；前端需要两次请求才能渲染完整页面；token_status 是项目详情的一部分，应内聚在详情 API 中。
