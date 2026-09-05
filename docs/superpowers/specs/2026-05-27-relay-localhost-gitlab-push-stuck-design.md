# 设计文档：relay 任务 zTree 推送卡住（本地 GitLab somanyad）

**日期：** 2026-05-27  
**状态：** 已批准（0-auto-flow 2026-05-27）  
**页面：** `task-detail/.../?relayToTrae=true` — zTree「推送」  
**任务示例：** `846269443533955072`（`http://localhost:8012/ljy/somanyad.git`）

---

## 现象

用户在 relay 直接启动流程中：启动 → zTree 发指令（somanyad 内写 hello world）→ 节点 completed → **提交** → 点击 **推送** 后长时间无响应（按钮显示 `…`，或数分钟后才报错）。

克隆与 Agent 执行均正常，说明 OAuth 换票与容器业务端点可用。

---

## 根因（已验证）

### 1. 容器 `oauth-access-push` 误跳过本地 GitLab 远端

`onlineServiceJS/src/layerGitOauthPush.mjs` 用 hostname 含 `gitlab` 的正则识别 GitLab。  
`http://localhost:8012/ljy/somanyad.git` **不匹配**，仓库被 `continue` 跳过 → `anyPushed=false` → 400，或在不走 OAuth 时 `git push` 无凭据挂起。

SaaS 侧 `forward_container_layer_git_push` 已正确下发 `oauth_auth_by_repo`（canonical key 与任务 `repo_url` 一致），断点在容器未消费该映射。

### 2. `git push` 无超时

`gitExecAsync` 对不可达/等待凭据的远端可无限阻塞；Django `CONTAINER_LAYER_GIT_PUSH_READ_TIMEOUT` 默认 120s，前端 `layerGraphBusyActionKey` 在此期间一直 busy，用户感知为「卡住」。

### 3. 关联设计（已部分交付）

| 项 | 状态 |
|----|------|
| 前端移除 GitHub 专用 push 预检（`layerGitGithubAppOauthConnected`） | ✅ 已移除 |
| SaaS 多 provider 换票 + `oauth-access-push` 转发 | ✅ 已有 |
| `prefer_container_remote` 仅 dev-local-token / 多仓 | 保持；relay 单仓走 OAuth 路径 |

详见 `2026-05-27-ztree-push-oauth-precheck-mismatch-design.md`（预检误拦）；本设计聚焦 **容器 localhost GitLab 推送执行**。

---

## 推送全链路（点击「推送」后）

```
浏览器 POST …/container-layer-git-push/
  → Django：identity_id + GitLab OAuth 换票 → oauth_auth_by_repo
  → 容器 POST …/api/layers/{id}/git/oauth-access-push
  → 层内 git push（GIT_ASKPASS + oauth2）
  → Django 可选 GitHub PR follow-up（GitLab 仓跳过）
```

Playwright 用例须断言：SaaS 200、`prefer_container_remote !== true`、`identity_id` 非空；可选 `PLAYWRIGHT_TRACE_PUSH=1` 打链路日志。

---

## 方案

### A. 容器：按 `oauth_auth_by_repo` 解析推送上下文（必须）

- 新增 `resolveOAuthPushRepoContext`：`canonicalRepoKey(origin)` 命中 `oauth_auth_by_repo` 时，用 `parseOwnerRepoFromPathUrl` 支持 `localhost:8012/owner/repo.git`。
- GitLab 使用 `originUrl` 作为 push remote，`GIT_ASKPASS` 用户名为 `oauth2`。

### B. 容器：`git push` 超时（必须）

- `GIT_PUSH_TIMEOUT_MS` 默认 90000；超时返回明确错误，避免无限 busy。

### C. 测试（必须）

| 层级 | 内容 |
|------|------|
| 单元 | `layerGitOauthPush.test.mjs` — localhost GitLab + oauth_auth_by_repo 执行 push 而非 skip |
| Playwright | `TaskDetail.relay-to-trae-somanyad-hello-world-push.playwright.test.js` — relay 全流程至推送 |
| 回归 | 现有 `test_layer_git_push_with_gitlab_oauth_only`、前端 `taskDetailLayerActions.test.js` 保持绿 |

### D. 运维提示

修改的是 **onlineServiceJS**；relay 场景需 **停止再启动** onlineServiceJS 后修复生效。

---

## 价值流影响

| 流 | 影响 |
|----|------|
| `oauth-token-fetch-timeout-governance` / `oauth-multi-repo-enhancement` | 推送与克隆共用换票；补容器执行与超时治理 |
| `task-detail-oauth-binding-guidance` | 减少「已授权仍卡住」误导 |
| `layer-oauth-fetch-multi-provider`（进行中） | 与多 provider 推送对齐 |

**Cross-stream：** `2026-05-27-ztree-push-oauth-precheck-mismatch`、`2026-05-27-relay-precheck-local-origin`。

---

## 领域概念（轻量）

| 概念 | 说明 |
|------|------|
| **BC: 任务协作 / 容器 Git** | zTree 提交/推送 |
| **BC: Git OAuth** | gitOauth provider、`oauth_auth_by_repo` |
| **VO: GitOAuthProviderKey** | `gitlab:gitlab-local` |
| **VO: CanonicalRepoKey** | 推送时 SaaS 与容器对齐的仓库键 |
| **Entity: TaskRepoIdentity** | 克隆/推送共享身份 |

---

## 验收

1. GitLab 本地仓（`localhost:8012`）+ 已 OAuth + 已选克隆身份 → 点击推送 **120s 内** 得到成功或可读失败（非无限 `…`）。
2. `git-push.log` 含 `oauth … git_push ok` 或明确 `fail`，无 `skip=non_github_remote`。
3. 单元测试与 Playwright 用例（凭据齐全时）通过。
4. 重启 onlineServiceJS 后手工复现任务 `846269443533955072` 推送成功。

---

## 非目标

- 不改变 `prefer_container_remote` 默认策略（relay 单仓仍走 OAuth 转发）。
- 不在此变更实现 GitLab MR 自动创建。

---

## 实现状态（Step 7 预置）

以下已在工作区落地，待 pipeline 8/9 验证与 PR：

- `trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs`
- `trae-agent/onlineServiceJS/src/layerGitOauthPush.test.mjs`
- `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-somanyad-hello-world-push.playwright.test.js`
