# 设计文档：relay 直启 GitLab 克隆 HTTP Basic 认证失败

**日期：** 2026-05-27  
**状态：** 已实现  
**页面：** `http://localhost:4000/tenant/827923618468040704/workspace/827923618602258432/task-detail/846269443533955072/?relayToTrae=true`

---

## 现象

用户在任务详情「直接启动」点击「启动」后，relayToTrae 与 onlineServiceJS 正常拉起，token-exchange 首轮 500 后重试成功，但 bootstrap 克隆失败：

```text
[bootstrap (post-listen) error] git exit 128: ...
remote: HTTP Basic: Access denied. ...
fatal: Authentication failed for 'http://localhost:8012/ljy/somanyad.git/'
```

预检与凭证接口已通过（否则不会进入 clone），说明 `repo-clone-credentials` 已返回 token，但 Git HTTP 认证被拒绝。

---

## 根因（已验证）

### 1. bootstrap 克隆与 push 的 GitLab HTTP 用户名策略不一致

| 路径 | GitLab HTTP 用户名 | 密码 |
|------|-------------------|------|
| `bootstrap.mjs` 克隆 | URL 路径首段（如 `ljy`） | `ephemeral_oauth_access_token` |
| `layerGitOauthPush.mjs` 推送 | `oauth2` | OAuth access_token |

GitLab 官方约定：使用 OAuth2 access token 或 PAT 通过 HTTPS 克隆/推送时，HTTP Basic **用户名应为 `oauth2`**，密码为 token。使用 namespace 路径段（`ljy`）配合 OAuth token 会导致 **HTTP Basic: Access denied**——与用户日志完全一致。

当前 `buildHttpAuthFromRepoCredential` 实现：

```javascript
const username = usernameFromRepoUrl(repoUrl); // → "ljy"
const password = rawCredential.ephemeral_oauth_access_token;
```

对 `http://localhost:8012/ljy/somanyad.git` 会发送 `ljy:<oauth_token>`，GitLab 拒绝。

### 2. 陈旧仓库 URL 不是本次直接原因

克隆目标已是当前项目地址 `http://localhost:8012/ljy/somanyad`（非陈旧 `example-user`），说明 `resolve_task_project_git_repos` 与凭证按仓名匹配已生效。本次失败发生在 **认证格式**，而非 URL 解析。

### 3. exchange-refresh 首轮 500（次要）

日志显示首轮 `exchange-refresh` 返回 500 后重试成功，不阻塞启动。可作为独立可靠性改进项，**不纳入本次必修复范围**（retry 已兜底）。

---

## 方案

### A. 后端：凭证 payload 携带 provider 与 HTTP 用户名（核心）

在 `_build_repo_clone_credentials` 返回的每仓条目中增加：

```json
{
  "http://localhost:8012/ljy/somanyad": {
    "ephemeral_oauth_access_token": "...",
    "provider": "gitlab",
    "git_http_username": "oauth2"
  }
}
```

**用户名解析规则（按优先级）：**

1. 若 identity 配置了非空 `git_remote_username` 且 provider 非 gitlab/github 标准 OAuth 场景 → 使用该值（保留自定义 Git 托管兼容）
2. `provider === "gitlab"` → `oauth2`
3. `provider === "github"` → `x-access-token`
4. 兜底 → URL 路径首段（向后兼容未知 provider）

`provider` 已由 `resolve_provider_from_repo_url(repo_url)` 解析，无需新 DB 字段。

### B. onlineServiceJS：按凭证字段构建 HTTP Basic（核心）

更新 `buildHttpAuthFromRepoCredential(rawCredential, repoUrl)`：

1. 优先 `rawCredential.git_http_username`
2. 否则按 `rawCredential.provider` 使用 provider 默认（gitlab→oauth2，github→x-access-token）
3. 最后 fallback 到 `usernameFromRepoUrl(repoUrl)`

与 `layerGitOauthPush.mjs` 的 push 行为对齐，消除 clone/push 分叉。

### C. 可观测性（增量）

bootstrap 克隆前在 outbound 日志（DEBUG 级别或现有 append 通道）记录：

- `repo_url`
- `provider`
- `git_http_username`（不记录 token）

便于后续排障「凭证有但 auth 失败」类问题。

---

## 价值流影响

| 流 | 影响 |
|----|------|
| `task-detail-repo-clone-credentials-contract` | 凭证条目新增可选字段 `provider`、`git_http_username`（向后兼容，旧客户端忽略新字段仍可用 fallback） |
| `task-detail-repo-clone-credentials-decoupling` | bootstrap 消费凭证字段，两段式调用不变 |
| `task-detail-runtime-relay` | relay 直启 clone 成功率提升 |
| `gitoauth-binding-state-persistence` | 无 schema 变更，仍通过 gitoauth 换票 |

不涉及 OAuth 发证流程或 DB migration。

---

## 领域概念（轻量）

- **Bounded Context：** 任务协作 / 容器 runtime / Git OAuth
- **实体：** `TaskRepoIdentity`、`UserCompanyGitIdentity`、`CloudServerConfig`
- **值对象：** `RepoCloneCredentialEntry`（扩展 provider + git_http_username）
- **领域服务：** `RepoCloneCredentialsFetchService`、容器侧 bootstrap clone
- **领域事件：** 无新增

---

## 测试计划

### 后端（Django）

- `test_fetch_container_repo_clone_credentials_returns_git_http_username_for_gitlab`  
  断言 gitlab 仓返回 `git_http_username: "oauth2"`、`provider: "gitlab"`
- `test_fetch_container_repo_clone_credentials_returns_x_access_token_username_for_github`  
  GitHub 仓返回 `x-access-token`
- 回归 `test_fetch_container_repo_clone_credentials_matches_identity_by_repo_name_when_url_stale`

### Node 单测

- `buildHttpAuthFromRepoCredential uses git_http_username from credential`
- `buildHttpAuthFromRepoCredential defaults gitlab to oauth2`
- `buildHttpAuthFromRepoCredential defaults github to x-access-token`
- 保留 path fallback 用例（无 provider 字段时）

### E2E

- 更新 `bootstrap-clone-host-alias-credential.api.spec.mjs`：mock 凭证含 `provider: "gitlab"`, `git_http_username: "oauth2"`，断言 fake git 收到 `git_http_username:oauth2`
- （可选）relay 直启 playwright：mock 凭证 + 断言 bootstrap 不因 auth denied 失败

---

## 非目标

- 不修改 exchange-refresh 500 根因（单独 reliability 项）
- 不改变陈旧仓库 URL 门控（已实现）
- 不重构预检为进程内调用
- 不在此变更中支持 SSH clone

---

## 风险

| 风险 | 缓解 |
|------|------|
| 少数自建 GitLab 强制使用真实用户名 | identity.git_remote_username 可 override；仅标准 OAuth 场景用 oauth2 |
| 旧版 onlineServiceJS 未升级 | 新字段 optional；fallback 仍为 path 段（行为与现网相同） |
| GitHub 误用 oauth2 | provider 分支明确 github→x-access-token |

---

## 验收标准

1. 本地 GitLab `localhost:8012`，任务已 OAuth 绑定且账号已保存 → relay 直启后 bootstrap **成功克隆** `ljy/somanyad`
2. `repo-clone-credentials` 对 GitLab 仓返回 `git_http_username: oauth2`
3. onlineServiceJS bootstrap 对 GitLab 使用 `GIT_HTTP_USERNAME=oauth2`，不再使用 namespace 路径段
4. GitHub 仓克隆/推送用户名策略与 push 一致（`x-access-token`）
5. 现有 stale-repo、precheck-internal-origin 回归测试仍通过
