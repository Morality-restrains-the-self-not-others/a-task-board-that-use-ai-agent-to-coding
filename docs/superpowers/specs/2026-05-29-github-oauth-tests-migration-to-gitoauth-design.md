# 设计文档：GitHub OAuth 测例从 task2app 迁移至 gitOauth

**日期：** 2026-05-29  
**状态：** 待批准（Step 1 / 0-auto-flow）  
**触发：** task2app 中部分 GitHub OAuth 测例实际验证的是 gitOauth 内部 API / 凭据模型，应归属 `gitOauth/` 服务。

---

## 背景与动机

架构已明确：**GitHub App 凭据、OAuth 换票、callback 落库** 在 `gitOauth`；`task2app` 仅保留 **桥接**（JWT start、内部 HTTP 客户端、任务级 `TaskGithubRepoOauthBinding`、容器 layer 消费）。

现状问题：

| 现象 | 影响 |
|------|------|
| `task2app/Saas_project/tests/test_fetch_gitoauth_credential_*.py` 用 mock 测 HTTP 客户端，却未触达 `GithubAppUserCredential` | `value-stream.yaml` 将 `git-oauth.api_githubappusercredential.*` 字段绑到这些文件，**真源与测例错位** |
| `gitOauth/api/tests.py` 已覆盖 internal summary/user-ids/callback/bind 等 | 与 task2app 侧 mock 测例 **重复且分层错误** |
| Playwright `GitSiteOAuth.github-*.playwright.test.js` 跨三服务 E2E | 合理留在 task2app/playwright（用户旅程），不纳入本次迁移 |

用户目标：**GitHub OAuth 的「服务内契约」测例应在 gitOauth 运行；task2app 只保留桥接与业务绑定的薄测。**

---

## 价值流影响（Step 1 输入）

读取 `value-stream.yaml` 后，下列 step 的 `test_file` 指向待迁移或需拆分的文件：

| 价值流 | Step | 当前 test_file | 调整方向 |
|--------|------|----------------|----------|
| `cloud-github-binding-string-contract` 同域邻近 | `cloud-credential-summary-string-contract` | `tests/test_fetch_gitoauth_credential_summary_for_user.py` | → `gitOauth/api/tests.py`（或拆出 `gitOauth/api/tests/test_github_internal_views.py`） |
| `gitoauth-binding-state-persistence` | `bind-state-summary-contract-thin-slice` | 同上 | 在 gitOauth 增加 **summary 返回 bind_status/bind_error** 的 HTTP 断言 |
| `git-provider-scope-failfast`（若存在） | scope 相关 | `tests/test_git_oauth_scope_validation.py` | **本次不迁**（见下文「保留在 task2app」） |

Step 3（价值流）将据此更新 `test_file` 路径与 runner `working_dir`（gitOauth 需独立 pytest 入口或 runAll 子目标）。

---

## 领域概念清单（供 Step 5 DDD）

| 限界上下文 | 实体 / 聚合 | 说明 |
|------------|-------------|------|
| **gitOauth** | `GithubAppUserCredential` | OAuth refresh 密文、github_user_id/login、bind_status/bind_error、多连接 |
| **gitOauth** | Internal API：`summary-for-user`、`user-ids`、`access-for-user`、`delete` | task2app 桥接的消费端点 |
| **task2app accounts** | `fetch_gitoauth_*` 客户端 | 仅负责 JWT、service_base、provider_key 路由 |
| **task2app projects** | `TaskGithubRepoOauthBinding` | 任务 × 仓库 × GitHub 账号，非 OAuth 核心 |

---

## 测例分类（迁移矩阵）

### A. 迁移至 gitOauth（主工作量）

在 `gitOauth/api/tests.py`（或按模块拆分）**补强/合并**，然后 **删除** task2app 中对应「纯 gitOauth 契约」文件：

| task2app 文件 | 当前断言 | gitOauth 已有覆盖 | 迁移动作 |
|---------------|----------|-------------------|----------|
| `tests/test_fetch_gitoauth_credential_summary_for_user.py` | mock `_request_gitoauth_internal_sync`，检查 POST body / URL | `GithubOAuthCredentialSummaryForUserViewTests` 已测 200 + connected/connections | **删除** task2app 文件；在 gitOauth 补：`provider_key` 过滤、`bind_status`/`bind_error` 在 summary JSON 中的契约（含 failed 行） |
| `tests/test_fetch_gitoauth_credential_user_ids.py` | mock GET user-ids | `GithubOAuthCredentialUserIdsViewTests` | **删除** task2app 文件；gitOauth 已有，无需重复 |

`test_gitoauth_per_provider_service_base.py`：**保留在 task2app**（测 `resolve_gitoauth_service_base` 与桥接 URL 拼装，属主站配置消费，非 gitOauth HTTP 本体）。

### B. 保留在 task2app（不迁移）

| 文件 | 理由 |
|------|------|
| `tests/test_github_app_start_redirect_uri.py` | 桥接 JWT → gitOauth `/oauth/start/` |
| `tests/test_github_app_connection_get_user_scoped.py` | 用户 scoped REST + mock 桥接 |
| `tests/test_github_app_connection_delete_auth.py` | CSRF/Token 与 DELETE 编排 |
| `tests/test_github_task_repo_oauth_binding.py` | `TaskGithubRepoOauthBinding` 业务 API |
| `tests/test_layer_github_oauth_tokens.py` | 容器 layer 绑定解析 |
| `tests/test_report_gitoauth_task_credential_audit.py` | 主站上报 audit 至 gitOauth |
| `tests/test_git_oauth_scope_validation.py` | `saas_project.settings` 启动期 fail-fast（逻辑在 `accounts.domain`，非 gitOauth 进程） |
| Playwright `GitSiteOAuth.github-*.playwright.test.js` | 端到端用户路径 |

可选：迁移后为桥接保留 **极简** 测例（如 `test_github_bridge_summary_calls_correct_path`），仅 1 个 mock 断言 URL 含 `provider_key`，避免回归桥接接线错误。

### C. gitOauth 需新增的断言（缺口）

相对 task2app mock 测例与 value-stream 字段，gitOauth 侧建议补齐：

1. **summary**：`bind_status`/`bind_error` 在 connected / failed / pending 行上的 JSON 契约（DB 已有 callback 测试，summary 视图需显式 HTTP 断言）。
2. **summary**：请求体 `provider_key`（如 `github:github-official`）仅返回该 provider 下的 connections。
3. **user-ids**：排序、空表（已有 sorted 测试，确认保留）。

---

## 实施方案

### Phase 1 — gitOauth 补强（TDD 红→绿）

1. 在 `gitOauth/api/tests.py` 增加/扩展 `GithubOAuthCredentialSummaryForUserViewTests`（bind_status 契约、provider_key 过滤）。
2. 运行：`cd gitOauth && python -m pytest api/tests.py -q`（或项目既有 `run.sh`）。

### Phase 2 — 删除错位测例 + 薄桥接（可选）

1. 删除 `test_fetch_gitoauth_credential_summary_for_user.py`、`test_fetch_gitoauth_credential_user_ids.py`。
2. （可选）新增 `tests/test_github_app_tokens_bridge.py`：仅 2 个 mock 测试验证 `fetch_*` 在配置缺失时返回「未配置」、配置存在时 URL/method/service 标签正确。

### Phase 3 — 价值流与 runAll

1. 更新 `value-stream.yaml` 中上述 step 的 `test_file` 为 gitOauth 路径（例如 `gitOauth/api/tests.py` 或拆分后的模块路径）。
2. 确认 `runAll.yaml` / `value-stream.yaml` 的 `runner` 是否需 **多 working_dir** 或 gitOauth 独立 pytest 步骤（若当前 runner 仅 `task2app/Saas_project`，需在 runAll 为 git-oauth 服务增加 pytest 健康检查，与 `runall` 中已有 `git-oauth` 服务名对齐）。

### Phase 4 — 文档与计划产物

- Step 3–6 产出 value-stream / NFR / plan 时引用本设计的路径约定。
- 不修改 `test_git_oauth_scope_validation.py` 归属（GitLab/GitHub scope fail-fast 仍在主站 settings 加载链）。

---

## 验收标准

| # | 标准 |
|---|------|
| 1 | 删除的 task2app 测例无「仅 mock gitOauth 响应、无业务逻辑」的重复覆盖 |
| 2 | `gitOauth` pytest 全绿，且覆盖 value-stream 中 `git-oauth.api_githubappusercredential.*` 相关 step |
| 3 | `task2app` pytest 全绿；桥接与 binding/layer 测例仍通过 |
| 4 | `value-stream.yaml` 中 `test_file` 指向与真源服务一致 |
| 5 | （若启用 runAll pytest）git-oauth 服务测试可在 CI/本地 runAll 中执行 |

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| runAll 仅跑 task2app pytest，迁移后 value-stream 绿但 gitOauth 未跑 | 在 runAll 为 `git-oauth` 增加 `pytest api/tests.py` health 或文档化手动命令 |
| 删除 mock 测例后桥接 URL 回归 | 可选保留 1–2 个薄桥接测试 |
| 多 provider（gitlab）测例误删 | 本设计仅动 **GitHub OAuth** 列出的两个 fetch_* 文件；GitLab 同类文件若存在另开任务 |

---

## 非目标

- 不迁移 Playwright E2E。
- 不迁移 `test_git_oauth_scope_validation.py`（主站 settings 域）。
- 不重构 `accounts/github_app_tokens.py` 实现（仅测例归属调整）。
- 不在本任务合并 `gitOauth/api/tests.py` 大文件拆分（可后续单独做）。

---

## 建议的 Step 6 任务切片（预览）

1. gitOauth：summary bind_status HTTP 测试  
2. gitOauth：summary provider_key 过滤测试  
3. 删除 task2app `test_fetch_gitoauth_credential_*`  
4. 更新 `value-stream.yaml`  
5. （可选）task2app 桥接薄测  
6. 全量 pytest + runAll 验证  

---

## 待决问题（实现前可默认）

| 问题 | 默认 |
|------|------|
| gitOauth 测试是否拆文件？ | 先扩展现有 `api/tests.py`，超 1200 行再拆 |
| value-stream runner | Step 3 定为 gitOauth 独立 pytest 条目或 runAll 子命令 |
