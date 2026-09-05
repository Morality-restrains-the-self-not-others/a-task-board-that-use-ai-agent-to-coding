# 设计文档：relay 直启预检 token 换发失败与 OAuth 已授权矛盾

**日期：** 2026-05-27  
**状态：** 已批准（0-auto-flow）  
**页面：** `task-detail/.../?relayToTrae=true`  
**目标：** 用户已完成 OAuth + 克隆身份保存后，点击「启动」能直接启动；依赖不可用时给出可执行提示而非误导性「完成 OAuth」。

---

## 现象

关联项目显示 **OAuth 已授权**，点击「直接启动 → 启动」后出现：

> 检测到仓库克隆凭证不完整。请回到上方「关联项目」区域逐仓库完成授权保存后再启动。

## 根因（已验证）

| 层 | 事实 |
|----|------|
| DB | 任务已有 `TaskRepoIdentity`（克隆身份已保存） |
| gitOauth | `access-for-user` 换 token 时调用 GitLab `localhost:8012/oauth/token` |
| 运行时 | **GitLab 8012 未启动** → gitOauth 502 → `_build_repo_clone_credentials` 跳过该仓 |
| 预检 | 期望仓库 − 凭证键 = `missing_repo_credentials` → **409** |
| UI | 409 触发 amber「凭证不完整」引导，与「OAuth 已授权」矛盾 |

**结论：** 409 混入了「缺身份」与「token 换发失败」两类问题；前端无法区分，用户被引导重复 OAuth。

次要问题：relay 直启模式下「克隆所用 Git 身份」选择器被 `containerEndpointRegistered` 隐藏，无容器时难以完成第二步（本次任务已有 identity，非主 blocker）。

---

## 方案

### A. 后端：凭证构建返回 token 换发失败明细（核心）

扩展 `_build_repo_clone_credentials`（或并列 helper）返回：

```python
{
  "credentials": { repo_url: {...} },
  "token_refresh_failures": [
    {
      "repo_url": "...",
      "error_code": "GIT_HOST_UNREACHABLE",  # 或 GITOAUTH_UPSTREAM_ERROR
      "detail": "Connection refused localhost:8012",
      "git_host": "localhost:8012",
    }
  ],
  "missing_identity_repos": [ repo_url, ... ],  # 无 TaskRepoIdentity
}
```

`repo-clone-credentials` 响应分流：

| 条件 | HTTP | error_code |
|------|------|------------|
| 仅缺 identity | 409 | `REPO_CLONE_CREDENTIALS_INCOMPLETE` |
| 有 identity 但 token 换发失败 | 502 | `REPO_CLONE_TOKEN_REFRESH_FAILED` |
| 两者兼有 | 409 + `token_refresh_failures` 字段（或优先 502 若任一 token 失败） |

relay `repo-credentials-precheck` 透传上述字段与 status。

### B. 前端：预检错误分流（核心）

`ServerConfig.logic.vue`：

| 条件 | UI |
|------|-----|
| 409 + `missing_repo_credentials` | 现有 amber 引导 + 缺失仓库摘要 |
| 502 + `REPO_CLONE_TOKEN_REFRESH_FAILED` | **不展示** OAuth 引导；提示启动 GitLab/Git 托管（如 8012）或检查 gitOauth |
| 502 网关 | 现有 Django 不可达文案 |

### C. relay 直启：克隆身份选择器可见（增量）

`TaskDetailLinkedProjectsPanel.vue`：当 `relayToTrae=true`（由父组件传入 prop）时，展示「克隆所用 Git 身份」下拉，不依赖 `containerEndpointRegistered`。

### D. 关联项目：OAuth 状态文案（可选）

token 换发失败时，仓库行可显示「OAuth 已绑定，换发 token 失败」——本次不做，以后端+启动面板分流为主。

---

## 价值流影响

| 流 | 影响 |
|----|------|
| `task-detail-repo-clone-credentials-contract` | 409/502 契约扩展 `token_refresh_failures` |
| `task-detail-oauth-binding-guidance` | 预检引导分流 |
| `relay-precheck-internal-origin` | precheck 透传新 error_code |
| `gitoauth-binding-state-persistence` | 只读；不改 schema |

---

## 领域概念（轻量）

- **Bounded Context：** 任务协作 / 容器 runtime / relay 直启 / gitOauth 桥接
- **实体：** `TaskRepoIdentity`、`CloudServerConfig`
- **值对象：** `RepoCloneCredentialCoverage`、`RepoCloneTokenRefreshFailure`
- **领域服务：** `TaskRepoCloneCredentialsGuardService`（区分 missing vs refresh failed）
- **领域事件：** `RepoCloneTokenRefreshFailed`

---

## 测试计划

### 后端
- identity 存在 + mock token 502 → 502 + `REPO_CLONE_TOKEN_REFRESH_FAILED`
- 无 identity → 409 + `missing_repo_credentials`（回归）
- relay precheck 透传 502 契约

### 前端
- precheck 502 token refresh → 不展示 OAuth amber 引导
- precheck 409 missing → 仍展示引导

### E2E
- Playwright mock 502 token refresh → 断言文案含 GitLab/8012，不含「OAuth 绑定」

---

## 验收标准

1. GitLab 8012 + gitOauth 8002 + Django 8001 运行，OAuth 已授权且 identity 已保存 → 点击启动 **200 预检并继续 start**
2. GitLab 8012 停止 → 启动失败，提示 **Git 托管不可达**，不提示重复 OAuth
3. 未保存克隆 identity → 仍 409 + 关联项目引导

---

## 非目标

- 不自动拉起 GitLab（runAll 已有级联；本次仅清晰报错）
- 不改 gitOauth OAuth 绑定存储
- 不合并 GitHub `TaskGithubRepoOauthBinding` 与 `TaskRepoIdentity` 模型
