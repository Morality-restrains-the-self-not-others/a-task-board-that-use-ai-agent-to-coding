# 设计文档：zTree 推送 OAuth 预检与克隆凭证矛盾

**日期：** 2026-05-27  
**状态：** 待批准（0-auto-flow Step 1）  
**页面：** `task-detail/.../?relayToTrae=true` — zTree 层级节点「推送」  
**任务示例：** `846269443533955072`

---

## 现象

用户在任务详情 zTree 点击层级节点「推送」时，**尚未发起后端请求**即弹出：

> 用于推送需先完成个人资料中的 GitHub 网站 OAuth 授权（refresh_token），以便平台换发 access_token 并由容器 HTTPS 推送。

同一任务仓库**已成功克隆**（说明 `TaskRepoIdentity` + gitOauth `access-for-user` 换票链路可用，或容器内已有有效 access_token）。

用户感知：OAuth/凭证已就绪，推送却被要求重复完成「GitHub 网站 OAuth」。

---

## 根因（已验证）

### 1. 前端存在 GitHub 专用硬编码预检

`taskDetailLayerActions.js` 在调用 `container-layer-git-push` **之前**拦截：

```javascript
if (!preferContainerRemote && pushIdentityId && !layerGitGithubAppOauthConnected.value) {
  window.alert('用于推送需先完成个人资料中的 GitHub 网站 OAuth 授权（refresh_token）...');
  return;
}
```

`layerGitGithubAppOauthConnected` 来自个人资料 Git 身份 API 的 **`github_app_user_oauth_connected`**，仅反映 **GitHub provider** 的 `fetch_gitoauth_credential_summary_for_user(provider=github)` 是否 `connected`。

### 2. 克隆与推送后端能力不对称（前端放大了分裂）

| 路径 | Provider 感知 | 凭证来源 |
|------|---------------|----------|
| **克隆** `repo-clone-credentials` | ✅ 多 provider（GitHub + GitLab + port_config gitOauth） | `TaskRepoIdentity` + `fetch_git_access_via_gitoauth_for_user(provider=…)` |
| **推送** `forward_container_layer_git_push` | ✅ 已支持 GitHub + GitLab 换票 | 同上 + `resolve_github_auth_by_repo_for_user` |
| **推送前端预检** | ❌ 仅 GitHub `connected` | 与个人资料 GitHub 摘要单字段绑定 |

典型触发场景（与本任务环境一致）：

- 关联仓库为 **GitLab 本地**（如 `http://localhost:8012/...`），用户已在个人资料完成 **GitLab OAuth**，克隆正常。
- 前端仍检查 **GitHub** OAuth 标志 → `false` → 误拦。

次要场景：GitHub 仓库已通过任务级 `TaskGithubRepoOauthBinding` / 容器 `.task2app_access_token` 具备推送能力，但 profile 级 GitHub summary 未标记 `connected` 时，同样误拦。

### 3. `prefer_container_remote` 未覆盖 relay 单仓场景

`preferContainerRemote` 仅在「多仓库」或 URL 含 `/ui/dev-local-token` 时为 true。relay 直启、单 GitLab 仓任务不满足 → 必走 GitHub 预检。

---

## 方案

### 原则

1. **后端为推送凭证真源** — 前端不应在 GitHub 单字段上硬拦；与 relay 预检 token 换发失败设计（2026-05-27-relay-precheck-token-refresh-failure）一致：区分「缺授权」与「换票失败」，避免误导文案。
2. **与克隆对齐** — 推送 OAuth 就绪判定应基于**任务关联仓库的 provider**，而非全局 GitHub 标志。
3. **最小破坏** — 保留身份选择与容器 unreachable 等已有校验；仅修正 OAuth 预检逻辑与文案。

### A. 前端：移除 GitHub 专用硬预检（必须）

删除 `taskDetailLayerActions.js` 中对 `layerGitGithubAppOauthConnected` 的阻断逻辑（L137–139）。

推送请求照常携带 `identity_id` / `prefer_container_remote` / `repo_url`；失败时展示后端 `409/502` 的 `detail`（后端已有 provider 感知错误，含 GitLab 换票失败文案）。

**可选清理：** 若 `layerGitGithubAppOauthConnected` 仅服务此预检，可从 push deps 移除；若关联项目面板仍展示 GitHub OAuth 状态则保留 fetch，不再用于 push gate。

### B. 后端：扩展 push auth context（推荐，小改）

`get_layer_git_push_auth_context` 当前仅描述 GitHub 仓库与 `push_requires_github_oauth`。

扩展响应（向后兼容）：

```json
{
  "has_github_repo_projects": false,
  "has_gitlab_repo_projects": true,
  "push_requires_github_oauth": false,
  "push_requires_git_oauth": true,
  "git_oauth_providers_required": ["gitlab:gitlab-local"],
  "git_oauth_ready_for_push": true,
  "github_app_user_oauth_connected": false,
  "gitlab_app_user_oauth_connected": true,
  "message": "..."
}
```

逻辑：对任务关联仓库 URL 调用 `resolve_provider_from_repo_url`，逐 provider 查 `fetch_gitoauth_provider_credential_summary_for_user`；全部 connected → `git_oauth_ready_for_push=true`。

前端**不强制**用此字段预检（A 已足够修复）；可用于关联项目区提示文案，避免再写「请完成 GitHub OAuth」误导 GitLab 用户。

### C. 文案（必须）

| 位置 | 现文案 | 改为 |
|------|--------|------|
| 前端 push 误拦（删除后 N/A） | GitHub refresh_token | — |
| `get_layer_git_push_auth_context.message` | 一律 GitHub | 按任务仓库 provider 动态生成（GitHub / GitLab / 混合） |
| 后端 409 `forward_container_layer_git_push` | 首句仍提 GitHub | 无 GitHub 仓时改为「请完成对应 Git 站点 OAuth 授权」 |

### D. 测试（必须）

| 层级 | 内容 |
|------|------|
| 前端单元 | `onLayerGraphLayerPush`：GitHub connected=false、有 identity、应 **发起** push API（mock fetch），不应 alert |
| 后端 pytest | `test_layer_git_push_with_gitlab_oauth_only`：无 GitHub connected、GitLab connected + token → 200 + `oauth_auth_by_repo` |
| 后端 pytest | `test_auth_context_gitlab_repo_flags`：GitLab 仓返回 `git_oauth_ready_for_push` |
| 回归 | 现有 `test_layer_git_push_policy.py` / `test_layer_git_push_auth_context.py` 保持绿 |

---

## 价值流影响

读取 `value-stream.yaml`：

| 流 | 步骤 | 影响 |
|----|------|------|
| `task-detail-oauth-binding-guidance` | OAuth 绑定引导 | 推送误导向 GitHub 重复授权 → 修正为 provider 感知 |
| `oauth-token-fetch-timeout-governance` | 容器 OAuth 换票 | 推送与克隆共用换票路径，预检不再分裂 |
| `project-detail-repo-oauth-provider-routing` | provider 路由 | 复用 `resolve_provider_from_repo_url` |
| `layer-oauth-fetch-multi-provider`（进行中） | 多 provider 拉 token | 与本修复同向；推送 gate 应对齐 |

**Cross-stream：** 与 `2026-05-27-relay-precheck-token-refresh-failure`、`2026-05-27-layer-oauth-fetch-multi-provider` 共享「克隆/推送/provider 一致」目标。

不涉及新数据库字段；API 响应为 additive JSON 字段。

---

## 领域概念（轻量，供 DDD）

| 概念 | 说明 |
|------|------|
| **Bounded Context: 任务协作 / 容器 Git** | zTree 推送、层内 git push |
| **Bounded Context: Git OAuth** | gitOauth provider 配置、换票 |
| **Entity: TaskRepoIdentity** | 克隆/推送共享的身份绑定 |
| **Value Object: GitOAuthProviderKey** | `github:…` / `gitlab:gitlab-local` |
| **Value Object: PushOAuthReadiness** | 按任务仓库 providers 聚合的 connected 状态 |
| **Domain Event** | （可选）`LayerPushBlockedByMisleadingPrecheck` — 本次为 UI 策略修复，可不落事件 |

---

## 非目标

- 不重构容器 `layerGitOauthRefreshPush.mjs` 全链路（除非测试暴露后端 GitLab push 缺口）
- 不改变 `prefer_container_remote` 默认策略
- 不在此变更实现 PR 自动创建逻辑变更

---

## 风险

| 风险 | 缓解 |
|------|------|
| 去掉预检后用户看到后端 409 而非即时 alert | 409 文案已含可执行说明；可选 B 改善面板提示 |
| GitHub 仓确实未 OAuth | 后端 409 仍会拦截，行为正确 |
| 混合 GitHub+GitLab 任务 | auth context 按 provider 列表检查；push 仍逐仓换票 |

---

## 验收

1. GitLab 本地仓任务：OAuth 已授权 + 克隆身份已选 → 点击 zTree「推送」**不再**出现 GitHub refresh_token alert；请求到达后端。
2. 若 gitOauth/GitLab 不可用 → 后端 409/502，文案指明 GitLab 换票或站点不可达，而非「完成 GitHub OAuth」。
3. GitHub 仓、未 OAuth → 后端 409，文案仍引导 GitHub 授权。
4. 现有 layer push 相关 pytest 全绿。
