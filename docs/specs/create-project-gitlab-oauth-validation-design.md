# Create Project 页面 GitLab OAuth 仓库校验优化 — 设计文档

## 问题描述

用户在 Create Project 页面 (`/tenant/{id}/create-project/`) 输入自托管 GitLab 仓库地址后，系统调用 `validate-git-repo` API 返回：

```json
{
  "is_accessible": false,
  "message": "GitLab 仓库需 OAuth 授权后验证可访问性 — 未检测到有效绑定令牌，请先完成 GitLab 授权绑定"
}
```

前端展示错误信息 + "OAuth 授权"按钮。用户点击按钮后存在以下问题：

1. **OAuth 完成后的回跳页没有自动触发仓库重校验** — 用户完成 GitLab 授权 → 回调 → 回到 CreateProject 页面后，仓库地址的校验状态仍是旧的错误，需手动重新输入或刷新
2. **未绑定 OAuth 时的初始文案对用户不够友好** — "未检测到有效绑定令牌" 对未完成授权的用户像是故障而非引导
3. **OAuth 流程中的中间态缺乏反馈** — 用户点击按钮后跳转到 GitLab，页面无法感知授权进行中

## 请求与响应追踪

### 完整调用链

```
前端 CreateProject.vue
  → validateGitRepoRow(rowId, repoUrl)
    → GET /api/tenant/{tenantId}/projects/validate-git-repo/?url={encoded}
      → utility_views.validate_git_repo_view
        → resolve_repo_access_tokens(user, repo_url)
          → is_gitlab_repo_url(repo_url) → True
          → resolve_provider_from_repo_url(repo_url) → ("gitlab", service_provider)
          → fetch_git_access_via_gitoauth_for_user(uid, provider="gitlab", provider_key="gitlab:{sp}")
            → gitOauth: POST /api/internal/gitlab/oauth/access-for-user/
              → GithubOAuthAccessForUserView.post()
                → GithubAppUserCredential.objects.filter(provider=provider_key, task2app_user_id=uid, bind_status="active")
                → 无记录 → 返回 404 {"detail": "not_found"}
            → r.status_code == 404 → 返回 (None, None)  # 未绑定
        → gl_token = None
        → is_git_repo_accessible(repo_url, gitlab_access_token=None)
          → _check_gitlab_repo(repo_url, gitlab_access_token=None)
            → GET {gitlab}/api/v4/projects/{owner}%2F{repo} (无 token)
            → 404 (私有仓库需认证)
            → used_token = False, sc = 404
            → 返回 (False, "GitLab 仓库需 OAuth 授权后验证可访问性 — 未检测到有效绑定令牌，请先完成 GitLab 授权绑定", meta)
```

### OAuth 授权流程

```
前端 CreateProject.vue
  → startRepoOAuthConnect(repoUrl, rowId)
    → resolveRepoOAuthProviderInfo(repoUrl) → {provider: "gitlab", service_provider: "default"}
    → createGithubAppReturnKey() → 32-char hex
    → setGithubAppReturnTarget(returnKey, currentPagePath)  # sessionStorage
    → GET /api/accounts/gitlab/oauth/start-from-gateway/?next=...&return_key=...&service_provider=...
      → APISIX 路由 → gitOauth GitlabOAuthStartFromGatewayView
        → X-User-Id header → 用户身份
        → _resolve_start_authorize_context(service_provider) → 获取 GitLab OAuth 配置
        → 存储 state 到 session
        → 返回 {authorize_url: "http://183.250.1.132:8012/oauth/authorize?client_id=...&..."}
    → window.location.href = authorize_url  # 跳转到 GitLab

GitLab 授权页面
  → 用户点击 Authorize
    → GitLab 重定向到 gitOauth callback
      → GitlabOAuthCallbackView.get()
        → exchange_authorization_code_for_tokens(code) → {access_token, refresh_token, scope}
        → fetch_gitlab_user_profile(access_token) → {id, username}
        → GithubAppUserCredential.objects.update_or_create(
            provider=provider_key,   # e.g. "gitlab:default"
            task2app_user_id=uid,
            github_user_id=gl_uid,
            defaults={refresh_token_cipher, github_login, scope, bind_status: "pending"}
          )
        → POST {task2app}/api/accounts/gitlab/internal/bind/  # 通知主站
          → GitlabAppInternalBindView: 校验 bridge secret + user 存在 + 缓存 access_token
        → 成功: bind_status → "active"
        → 重定向到前端: /oauth/github-app/callback/?returnKey=...&provider=gitlab&gitlab=ok

前端 GithubAppCallbackContinue.vue
  → consumeGithubAppReturnTarget(returnKey) → 读取 sessionStorage 中的目标路径
  → 合并 gitlab=ok 参数
  → router.replace(targetPath)  # 回到 CreateProject 页面

回到 CreateProject 页面 ← **问题点：没有自动触发重校验**
```

## 根因分析

| # | 问题 | 严重度 | 说明 |
|---|------|--------|------|
| 1 | **OAuth 回调后缺失自动重校验** | 🔴 高 | `GithubAppCallbackContinue.vue` 回到 CreateProject 页面后，没有任何逻辑监听 `gitlab=ok` 查询参数并触发 `validateGitRepoRow`。用户看到的是 OAuth 之前缓存的旧错误状态 |
| 2 | **validateGitRepoRow 在 finally 块中的调度时机错误** | 🟡 中 | `startRepoOAuthConnect` 的 `finally` 块中调用 `scheduleValidateGitRepoRow` 时，页面尚未跳转到 GitLab。这个校验在跳转前发出，校验结果会被页面跳转吞掉 |
| 3 | **错误文案在"未授权"状态不够区分** | 🟡 中 | "未检测到有效绑定令牌"对未完成授权的用户是预期状态，但文案读起来像故障。应区分"尚未授权" vs "授权失败" vs "授权后仍不可访问" |
| 4 | **gitOauth access-for-user 缺失时无区分** | 🟢 低 | `fetch_git_access_via_gitoauth_for_user` 对 404 返回 `(None, None)` 抹掉了错误详情，`validate_git_repo_view` 无法区分"未绑定"和"gitOauth 服务不可达" |
| 5 | **bind_status 生命周期不可见** | 🟢 低 | gitOauth callback 中 `bind_status` 经历 `pending → active/failed`，但主站侧无感知。若 bind 步骤失败 (`bind_status=failed`)，`access-for-user` 过滤 `bind_status="active"` 直接返回 404，前端只能看到"未绑定" |

## 设计方案

### 方案一：CreateProject 页面监听 OAuth 回调参数 (推荐)

**改动范围**: 前端 `CreateProject.vue` + `gitSiteOAuthCallbackUtils.js`

**逻辑**:
1. `CreateProject.vue` 在 `onMounted` 或 `watch` 中检测 URL query 参数 `gitlab` / `github`
2. 若值为 `ok`，遍历所有已输入的 git repo rows，触发重校验
3. 重校验完成后清除 URL 中的 query 参数（通过 `router.replace`）
4. 若值为 `bad_state` / `exchange_failed` 等错误码，显示 toast 错误提示

**伪代码**:
```javascript
// CreateProject.vue setup()
import { useRoute, useRouter } from 'vue-router'
import { applyOAuthCallbackFromRoute } from '../utils/gitSiteOAuthCallbackUtils'

onMounted(async () => {
  const result = await applyOAuthCallbackFromRoute(route, router, {
    onError: (msg) => { /* toast.error */ },
    onSuccess: (msg) => { /* toast.success */ },
  })
  if (result) {
    // OAuth 刚完成，重校验所有仓库行
    gitRepoRows.value.forEach(row => {
      const url = String(row.url || '').trim()
      if (url) scheduleValidateGitRepoRow(row.id, url)
    })
  }
})
```

### 方案二：修复 startRepoOAuthConnect 的校验时机

**改动范围**: 前端 `CreateProject.vue`

**逻辑**:
1. 移除 `finally` 块中的 `scheduleValidateGitRepoRow`（跳转前的无意义校验）
2. OAuth 按钮点击后设置 `repoOAuthPendingValidation` 标记
3. 页面从 OAuth 回跳后（方案一检测到 `gitlab=ok`），消费标记并重校验

### 方案三：优化后端错误消息区分度

**改动范围**: 后端 `projects/git_utils.py` `_check_gitlab_repo` + `projects/services/repo_access_token_resolver.py`

**逻辑**:
1. `resolve_repo_access_tokens` 返回值增加 `token_status` 字段:
   - `"not_bound"` — gitOauth 返回 404（用户从未授权）
   - `"token_error"` — gitOauth 返回错误（网络/内部异常）
   - `"token_available"` — 成功获取 token
   - `"not_applicable"` — 非 OAuth 仓库（纯 HTTP 公开仓库）

2. `validate_git_repo_view` 根据 `token_status` 返回不同消息:
   - `not_bound` + GitLab 404 → "此 GitLab 仓库为私有仓库，请点击下方「OAuth 授权」按钮完成授权后自动重试"
   - `token_error` + GitLab 404 → "授权令牌获取失败，请稍后重试或重新授权"
   - `token_available` + GitLab 404 → "授权成功但仓库不存在或无权访问，请检查仓库地址和权限"

3. `_check_gitlab_repo` 中的文案优化:
   - 当前: "GitLab 仓库需 OAuth 授权后验证可访问性 — 未检测到有效绑定令牌，请先完成 GitLab 授权绑定"
   - 新: "此仓库需要 GitLab 授权后才能访问，请点击下方「OAuth 授权」按钮完成授权"

### 方案四：后端增加 token 状态透传

**改动范围**: 后端 `validate_git_repo_view`

**逻辑**:
响应中增加 `token_status` 字段，前端可据此区分是否需要展示 OAuth 按钮:

```json
{
  "is_accessible": false,
  "message": "此仓库需要 GitLab 授权后才能访问",
  "token_status": "not_bound",
  "oauth_provider": "gitlab",
  "oauth_service_provider": "default"
}
```

## 推荐实施方案

**顺序实施**: 方案一 + 方案三 + 方案四（方案二自然包含在方案一中）

### 实施步骤

1. **前端**: CreateProject.vue 添加 OAuth 回调监听 → 自动重校验
2. **后端**: `resolve_repo_access_tokens` 返回 `token_status`
3. **后端**: `validate_git_repo_view` 根据 `token_status` 返回区分度消息
4. **后端**: `_check_gitlab_repo` 优化用户可见文案
5. **后端**: `validate_git_repo_view` 响应增加 `token_status` + `oauth_provider` 字段
6. **前端**: 根据 `token_status` 和 `oauth_provider` 决定按钮显隐和文案

## 领域概念清单

| 类别 | 概念 | 说明 |
|------|------|------|
| **Bounded Context** | Git OAuth Authorization | OAuth 授权流程：start → authorize → callback → bind → token exchange |
| **Bounded Context** | Project Repo Validation | 仓库可访问性校验：URL 解析 → token 解析 → Git API 探测 |
| **Key Entity** | GitOauthCredential (gitOauth) | OAuth 凭据：provider_key + user_id + refresh_token + bind_status |
| **Key Entity** | AccessTokenCache (task2app) | 本地 token 缓存：UserGithubAppAccessTokenCache |
| **Candidate Aggregate** | GitOauthBinding | 聚合根：凭据生命周期 (pending → active/failed) + 审计 |
| **Domain Event** | OAuthBindingCompleted | OAuth 回调成功 → bind_status=active → 通知主站刷新状态 |
| **Domain Service** | RepoAccessTokenResolver | 解析仓库 URL → 确定 provider → 获取 access token |
| **Domain Service** | GitRepoAccessChecker | 使用 token 探测 Git API 可访问性 |
| **Value Object** | RepoAccessResult | is_accessible + access_status + message + token_status |
| **Value Object** | ProviderKey | "gitlab:default", "gitlab:tencent-gitlab" 等 |

## 价值流影响分析

### 受影响的现有价值流

| 价值流 | 步骤 | 影响说明 |
|--------|------|---------|
| `project-detail-repo-oauth-row-action` | `project-detail-repo-oauth-button-thin-slice` | CreateProject 页的 OAuth 按钮逻辑与 ProjectDetail 共享 `repoOAuthAuthorizeUtils.js`、`useProjectGitOAuthCatalog.js` — 改动需保持兼容 |
| `project-detail-repo-oauth-row-action` | `project-detail-repo-oauth-provider-routing` | provider_key 解析逻辑共享 |
| `gitoauth-binding-state-persistence` | `bind-state-summary-contract-thin-slice` | `bind_status` 字段语义不变，但需确保 `access-for-user` 能正确处理 `pending→active` 过渡 |
| `oauth-callback-error-toast` | `callback-hints-guard-thin-slice` | OAuth 回调错误文案映射需扩充 CreateProject 场景 |
| `task-detail-oauth-binding-guidance` | `oauth-binding-two-phase-guidance-thin-slice` | validate-git-repo 的 token_status 新增字段影响 task-detail 预检链路 |
| `create-task-project-repo-access-label` | (step) | `test_project_repo_access_check_api.py` 测试需更新以覆盖新 `token_status` 字段 |

### 测试影响

| 测试文件 | 变更 |
|---------|------|
| `tests/test_validate_git_repo_api.py` | 新增测试用例：not_bound / token_error / token_available 状态返回 |
| `tests/test_project_branches_gitlab_auth_guard.py` | 验证新 token_status 字段不影响分支预览 |
| `tests/test_project_repo_access_check_api.py` | 验证 project_repo_access_check 响应包含 token_status |
| `front_project/app/src/tests/domain/oauth_callback/` | 新增 CreateProject 页面 OAuth 回跳 + 重校验的 domain model 测试 |
| `front_project/app/src/views/CreateProject.vue` 相关 E2E | Playwright 测试: OAuth 回跳后自动重校验的完整流程 |

## 总结清单

- **OAuth 回跳重校验**: 方案一（自动监听 query 参数 + 重校验）、方案二（修复 finally 块时机）
- **错误消息区分度**: 方案三（三层 token_status 驱动不同文案）、方案四（response 增加字段）
- **后端 token_status 透传**: 方案三（resolve_repo_access_tokens 返回值增强）、方案四（API 响应结构扩展）
- **前端按钮显隐逻辑**: 方案一（自动重校验后清除错误）、方案三+四（token_status 驱动）
