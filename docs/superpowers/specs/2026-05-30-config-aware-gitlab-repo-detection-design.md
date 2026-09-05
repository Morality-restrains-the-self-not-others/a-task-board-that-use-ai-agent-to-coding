# 设计文档：配置感知的 GitLab 仓库识别（分支预览 + OAuth 按钮）

**日期**：2026-05-30  
**状态**：已实现  
**关联问题**：项目详情页「分支列表预览」对 `http://1.117.67.121:8012/...` 等 IP 自托管 GitLab 返回 `Branches cannot be retrieved from generic Git repositories without credentials`，且不显示 OAuth 授权按钮。

---

## 1. 背景与问题

### 1.1 现象

用户在项目详情页（如 `/tenant/.../projects/848537693488873472/`）点击「分支列表预览」时，仓库 URL 指向腾讯云 GitLab（`http://1.117.67.121:8012/...`）会报错：

> Branches cannot be retrieved from generic Git repositories without credentials

同时仓库行不显示「OAuth 授权」按钮。

### 1.2 根因（已验证）

系统存在**两套 GitLab 识别逻辑**，未对齐：

| 路径 | 识别方式 | 对 `1.117.67.121:8012` |
|------|----------|--------------------------|
| OAuth 授权 / git-site-oauth 设置页 | `infer_git_provider_from_repo_url()` → 匹配 `port_config.json` 的 `target.website` | ✅ 识别为 `gitlab:tencent-gitlab` |
| 分支预览 `get_branches` / `get_repo_branches` | `is_gitlab_repo_url()` 硬编码（gitlab 域名 / localhost） | ❌ 识别为 generic |
| 前端 `resolveRepoOAuthProvider()` | hostname 启发式（github.com / localhost / 含 gitlab） | ❌ 返回空，不显示按钮 |

**结论**：OAuth 已在 `port_config.json` 配置完成，授权链路本身可用；分支预览与项目页 OAuth 入口未消费同一份配置。

### 1.3 影响范围

- 所有在 `gitOauth` 中配置、但 hostname **不含 `gitlab` 且非 localhost** 的自托管 GitLab（含群晖 `183.250.1.132:8012`、腾讯云 `1.117.67.121:8012`）。
- `localhost:8012` 因历史特例仍可用，不应回归。

---

## 2. 目标与非目标

### 2.1 目标

1. 分支预览对 `gitOauth` 已配置的 GitLab 站点走 GitLab 分支 API + OAuth token 路径，不再落入 generic 报错。
2. 项目详情页（及 CreateTaskModal 等复用 `repoOAuthAuthorizeUtils` 的入口）对配置内 GitLab 站点显示 OAuth 按钮。
3. 新增 `gitOauth` 条目后，**无需改代码**即可支持分支预览与 OAuth 入口（配置驱动）。
4. 保持 GitHub、gitlab.com、localhost 现有行为不回归。

### 2.2 非目标（本轮不做）

- 为 generic Git（非 GitLab/GitHub）实现分支列举。
- SSH 仓库 URL 的分支预览（现有行为：需凭据提示，不变）。
- 授权成功后自动刷新分支预览（Enhancement，留后续迭代）。
- 修改 gitOauth 服务本身或 OAuth 回调契约。

---

## 3. 价值流影响

读取 `value-stream.yaml`，本变更影响：

| 现有 Stream | 影响 |
|-------------|------|
| `project-detail-repo-oauth-row-action` | 扩展 provider 路由至 IP 自托管 GitLab；补回归测试 |
| `git-site-oauth-multi-service-provider-catalog` | 复用已有 catalog API，不新增配置源 |
| `gitoauth-binding-state-persistence` | 分支预览将正确触发 token 换取（间接依赖） |

**无需新建 value stream**；在 `project-detail-repo-oauth-row-action` 下新增 step：`project-detail-config-aware-gitlab-repo-detection`（active）。

**测试影响**：

- 更新/新增：`tests/test_git_utils.py`（或等价）、`tests/test_project_branches_gitlab_auth_guard.py`
- 新增 Playwright：`ProjectDetail.branch-preview-tencent-git.playwright.test.js`（或参数化 IP 场景）
- 更新：`repoOAuthAuthorizeUtils.test.js`

**字段影响**：无数据库 schema 变更；只读 `GIT_OAUTH_PROVIDER_CONFIGS` / `port_config.json`。

---

## 4. 领域概念清单（供 DDD 步骤输入）

| 概念 | 类型 | 说明 |
|------|------|------|
| **Git OAuth 配置目录** | 值对象/实体 | 来自 `port_config.gitOauth`，含 `website`、`service_provider` |
| **RepoUrl** | 值对象 | 规范化后的仓库 HTTP(S) URL |
| **GitProviderDetection** | 领域服务 | 由 repo URL 解析 provider（github/gitlab/bitbucket/unknown） |
| **Bounded Context: accounts** | 上下文 | OAuth provider 配置、token 换取、catalog API |
| **Bounded Context: projects** | 上下文 | 分支预览 API、git_utils |
| **领域事件** | 无新增 | 只读查询路径，无持久化事件 |

---

## 5. 方案对比

### 方案 A：后端统一识别 + 前端复用 catalog API（推荐）

- **后端**：`is_gitlab_repo_url()` 改为委托 `infer_git_provider_from_repo_url(repo_url) == "gitlab"`（或提取共享 `is_configured_gitlab_repo_url()`）。
- **前端**：应用启动或 ProjectDetail 挂载时拉取 `/api/accounts/git-oauth/providers/`，按 `website` origin 匹配 repo URL，决定 provider 与 OAuth 按钮可见性。
- **优点**：配置单源；新增站点零代码；与 git-site-oauth 一致。
- **缺点**：前端需异步加载 catalog（可缓存 sessionStorage）。

### 方案 B：新增 `resolve` API，前后端均调用

- 新增 `GET /api/accounts/git-oauth/resolve/?repo_url=...` 返回 `{ provider, service_provider, supports_oauth }`。
- 前端分支预览错误时也可据此显示按钮。
- **优点**：识别逻辑完全在后端，前端最薄。
- **缺点**：多一次 API；ProjectDetail 需改数据流。

### 方案 C：前端硬编码扩展 IP 列表

- 在 `repoOAuthAuthorizeUtils.js` 增加 `1.117.67.121` 等。
- **优点**：改动最小。
- **缺点**：每增站点改代码；与「配置驱动」目标相悖。**不推荐**。

**推荐方案 A**：后端改动一行级委托即可修复分支预览；前端 catalog 匹配与 UserGitSiteOAuthSettings 已有模式一致，无需新 API。

---

## 6. 详细设计（方案 A）

### 6.1 后端：统一 GitLab 识别

**文件**：`task2app/Saas_project/projects/git_utils.py`

```python
def is_gitlab_repo_url(repo_url: str | None) -> bool:
    from accounts.git_oauth_providers import infer_git_provider_from_repo_url
    return infer_git_provider_from_repo_url(repo_url) == "gitlab"
```

保留原 localhost 路径段检查的逻辑**由** `infer_git_provider_from_repo_url` + 配置覆盖，无需重复。

**行为变化**：

- `http://1.117.67.121:8012/group/repo` → GitLab 路径 → 尝试 `fetch_git_access_via_gitoauth_for_user(..., provider_key=gitlab:tencent-gitlab)`
- 无 token 且无 localhost cookie 时 → 返回现有中文鉴权提示（非 generic 报错）
- 未在 `gitOauth` 配置的未知 host → 仍走 generic（符合预期）

**session cookie 兜底**：仅保留 `localhost` / `127.0.0.1`（跨域 IP 站点无法转发浏览器 cookie，不变）。

### 6.2 前端：catalog 驱动的 OAuth 识别

**新增模块**（或扩展 `repoOAuthAuthorizeUtils.js`）：

1. `loadGitOAuthProviderCatalog()` — 调用 `/api/accounts/git-oauth/providers/`，内存缓存。
2. `resolveRepoOAuthProviderFromCatalog(repoUrl, catalog)` — 将 repo URL 的 origin（scheme + host + port）与每条 `website` 比对；命中则返回 `provider`（github/gitlab）。
3. `resolveRepoOAuthProvider(repoUrl)` — 先走现有 hostname 启发式；未命中则走 catalog 匹配。

**ProjectDetail.vue / CreateTaskModal.vue**：

- 挂载时 `await loadGitOAuthProviderCatalog()`（失败时降级为启发式，不阻塞页面）。
- `shouldShowRepoOAuthButton` 使用增强后的 `resolveRepoOAuthProvider`。

**OAuth start URL**：仍为 `/api/accounts/{provider}/app/start/?repo_url=...`（后端已支持按 repo_url 路由 service_provider）。

### 6.3 错误文案契约

| 场景 | 期望 error |
|------|------------|
| 已配置 GitLab、无 OAuth | 「无法获取 GitLab 分支：未检测到可用授权…」（现有） |
| 未配置 GitLab host | generic 报错（不变） |
| 有 token、仓库不存在 | `GitLab repository not found`（不变） |

前端 `shouldShowRepoOAuthAuthorizeButton` 继续仅在 generic 报错时作为 CreateTaskModal 的补充条件；ProjectDetail 以 catalog 支持为准显示按钮。

### 6.4 数据流（修复后）

```
用户点击「分支列表预览」
  → GET /branches/?repo_url=http://1.117.67.121:8012/...
  → infer_git_provider_from_repo_url → gitlab:tencent-gitlab
  → fetch_git_access_via_gitoauth_for_user
  → get_repo_branches (GitLab API)
  → 返回 branches 或明确鉴权错误

用户点击「OAuth 授权」
  → catalog 匹配 website → provider=gitlab
  → GET /api/accounts/gitlab/app/start/?repo_url=...
  → 跳转 gitOauth → GitLab authorize
```

---

## 7. 测试策略

### 7.1 单元测试（Django）

| 用例 | 断言 |
|------|------|
| `is_gitlab_repo_url("http://1.117.67.121:8012/a/b")` | True（@override_settings GIT_OAUTH_PROVIDER_CONFIGS） |
| `is_gitlab_repo_url("http://unknown.example/a/b")` | False |
| `get_branches` 对 tencent-gitlab 无 token | 返回鉴权提示，**不**调用 generic |
| localhost 回归 | 行为与现有一致 |

### 7.2 前端单元测试（Vitest）

- `resolveRepoOAuthProvider` 对 catalog 中 `website: http://1.117.67.121:8012` 返回 `gitlab`
- 未知 host 仍返回 `''`

### 7.3 Playwright

- 项目详情：IP GitLab 仓库行显示 OAuth 按钮
- 分支预览：mock branches API 或 stub 下不应出现 generic 报错字符串

---

## 8. 风险与回滚

| 风险 | 缓解 |
|------|------|
| `infer_git_provider_from_repo_url` 误将非 GitLab host 判为 gitlab | 仅匹配 `gitOauth` 显式配置的 `website` origin；补 unknown host 负例测试 |
| 前端 catalog 加载失败 | 降级启发式；OAuth 按钮可能仍缺失，但后端分支预览已修复 |
| 性能：catalog 重复请求 | session 级缓存 |

回滚：还原 `is_gitlab_repo_url` 与前端 utils 即可。

---

## 9. 验收标准

1. 项目 `848537693488873472`（仓库 `http://1.117.67.121:8012/...`）点击分支预览：**不再**出现 generic 报错；未授权时显示 GitLab 鉴权提示。
2. 同项目仓库行显示「OAuth 授权」按钮，点击可进入 tencent-gitlab 授权流。
3. 完成 OAuth 后分支预览返回分支列表（或明确的 404/403 业务错误）。
4. `localhost:8012` 与 `gitlab.daydaymoney.com` 现有 Playwright/单测全部通过。
5. 在 `port_config.json` 新增一条 `gitOauth["http://new-host:8012"]` 后，无需改代码即可识别（手动冒烟）。

---

## 10. 实现范围估算

| 区域 | 文件 | 改动量 |
|------|------|--------|
| 后端识别 | `projects/git_utils.py` | ~10 行 |
| 后端测试 | `tests/test_git_utils.py`, `test_project_branches_gitlab_auth_guard.py` | 新增 ~60 行 |
| 前端 utils | `repoOAuthAuthorizeUtils.js` + test | ~80 行 |
| 前端页面 | `ProjectDetail.vue`, `CreateTaskModal.vue`（可选预加载 catalog） | ~20 行 |
| E2E | 新 Playwright 用例 | ~50 行 |

**总估**：小中型修复，1 个 PR 可交付。
