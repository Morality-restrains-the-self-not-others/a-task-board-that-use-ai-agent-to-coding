# 设计文档：gitOauth 内部 API 按 service_provider 路由

**日期：** 2026-05-27  
**状态：** 已实施  
**关联页面：** `http://localhost:4000/user/827923618451263488/profile/git-site-oauth/`

---

## 背景与动机

当前主站（task2app Django）调用 gitOauth 内部 API（summary、access-for-user、delete 等）时，统一使用全局配置：

- `conf/port_config.json` → `django.gitoauth`
- 环境变量 `DJANGO_GITOAUTH_BASE` 可覆盖

本地开发时 `django.gitoauth = http://gitoauth.api.daydaymoney.com`，公网 nginx 返回 **502**；而 `gitOauth.http://localhost:8012` 条目的 `service.allowedHost = http://localhost:8002` 实际可达。

**授权 start 跳转** 已通过 `GitOauthAuthorizeRouteService` + 各条目的 `service_base`（来自 `service.allowedHost`）按 provider 路由；**内部 API 桥接**仍走全局 `DJANGO_GITOAUTH_BASE`，行为不一致。

### 用户目标

1. **去掉** `DJANGO_GITOAUTH_BASE` 环境变量及 `django.gitoauth` 全局根 URL 依赖  
2. 主站从 `conf/port_config.json` 的 `gitOauth` **按 `service_provider` 读取 `service.allowedHost`**（运行时字段 `service_base`）  
3. 公网不通时，`gitlab-local` 等本地条目仍可正常查询绑定状态，**不应因其他 provider 的公网 502 而失败**

---

## 现象（已验证）

| 调用目标 | HTTP |
|----------|------|
| `http://gitoauth.api.daydaymoney.com/.../summary-for-user/` | 502（nginx） |
| `http://127.0.0.1:8002/.../summary-for-user/` | 200 |
| Django `fetch_*` 使用 `DJANGO_GITOAUTH_BASE` | 全部 `http 502` |
| 前端 `GET .../connection/?service_provider=gitlab-local` | 503（上游 502） |

---

## 根因

`accounts/github_app_tokens.py` 中 6 处函数硬编码 `settings.DJANGO_GITOAUTH_BASE` 构造 URL，**忽略** `GIT_OAUTH_PROVIDER_CONFIGS[*].service_base`，与 authorize start 的路由策略分裂。

---

## 方案

### 1. 新增统一解析函数（accounts 层）

在 `accounts/git_oauth_providers.py` 增加：

```python
def resolve_gitoauth_service_base(
    provider: str,
    *,
    service_provider: str | None = None,
    provider_key: str | None = None,
) -> str | None:
    """从 GIT_OAUTH_PROVIDER_CONFIGS 解析 service.allowedHost（service_base）。"""
```

逻辑与现有 `DjangoGitOauthProviderRoutingRepository._resolve_service_base` / `github_app_views._resolve_provider_gitoauth_base` 一致：

1. `resolve_provider_config(provider, service_provider=..., allowed_host=...)`
2. 取 `cfg["service_base"]`，空则回退 `http://{host}:{port}`

**不再**回退 `DJANGO_GITOAUTH_BASE`（与 2026-05-26 project-detail OAuth NFR 一致）。

配置缺失时返回 `None`，调用方返回明确错误（如 `未配置 gitOauth service.allowedHost（provider_key=…）`）。

### 2. 改造 `github_app_tokens.py` 全部内部调用

| 函数 | 路由键 |
|------|--------|
| `fetch_gitoauth_provider_credential_summary_for_user` | `provider` + `provider_key`（已有） |
| `fetch_git_access_via_gitoauth_for_user` | `provider` + `provider_key`（已有） |
| `delete_gitoauth_user_credential` | `provider` + `provider_key`（已有） |
| `fetch_gitoauth_credential_summary_for_user` | 改为默认 `provider_key=github:github-official`（或第一个 github 配置） |
| `fetch_gitoauth_credential_user_ids` | 见 §3 |
| `report_gitoauth_*` | 绑定具体 `provider_key` 或按 §3 |

每个函数在构造 URL 前：

```python
base = resolve_gitoauth_service_base(provider, provider_key=provider_key_norm, ...)
if not base:
    return None, "未配置 gitOauth service.allowedHost ..."
url = f"{base}/api/internal/{provider_norm}/oauth/..."
```

### 3. 无 provider_key 的管理/审计类调用

**`fetch_gitoauth_credential_user_ids(provider)`**（orphan cleanup 命令）：

- 收集该 `provider` 下所有配置条目的 **distinct `service_base`**
- 对每个 base 调用一次 internal API，合并 `user_ids` 去重
- 任一条 base 失败：记录 warning，继续其他 base；全部失败才返回 error

**`report_gitoauth_github_app_token_use` / `report_gitoauth_task_credential_audit`**：

- 默认使用 `github:github-official` 的 `service_base`（与生产主 GitHub 条目一致）
- 可选后续：audit payload 增加 `provider_key`（非本变更必须）

### 4. 移除全局配置

| 移除项 | 说明 |
|--------|------|
| `settings.DJANGO_GITOAUTH_BASE` | 从 `saas_project/settings.py` 删除加载逻辑 |
| `django.gitoauth` | 从 `port_config.json` 删除（文档同步） |
| `DJANGO_GITOAUTH_BASE` env | 从 `.env_django.yaml.example` 删除；CI/文档不再推荐 |
| 测试 `@override_settings(DJANGO_GITOAUTH_BASE=...)` | 改为 mock `resolve_gitoauth_service_base` 或配置 `GIT_OAUTH_PROVIDER_CONFIGS` |

**不修改** `gitOauth/` 子项目内的 `DJANGO_GITOAUTH_BASE`（gitOauth 服务自身配置，范围外）。

### 5. 顺带修补（同文件小改）

- `GitlabAppConnectionView.delete` 当前硬编码 `provider_key="gitlab:default"`，应改为与 GET 一致，使用请求中的 `service_provider` 解析 `provider_key`（避免 gitlab-local 解绑走错键）

---

## 预期行为（验收）

| service_provider | service.allowedHost | connection GET |
|------------------|---------------------|----------------|
| `github-official` | `http://gitoauth.api.daydaymoney.com` | 公网不通 → **503** + detail `http 502`（仅该 tab） |
| `daydaymoney-gitlab` | `http://gitoauth.api.daydaymoney.com` | 同上 |
| `gitlab-local` | `http://localhost:8002` | **200**，`connected: false/true` |

切换 tab 时**互不影响**；providers 目录接口仍 200。

---

## 价值流影响

受影响流（`value-stream.yaml`）：

- `gitoauth-binding-state-persistence` — summary 路由按 provider_key
- `git-site-oauth-multi-service-provider-catalog` — git-site-oauth 多 tab 独立可达
- `oauth-token-fetch-timeout-governance` — access-for-user 路由
- `project-detail-repo-oauth-row-action` — start 已按 provider；内部换票对齐
- `task-detail-oauth-binding-guidance` — connection API 按 repo/service_provider

新增/更新测试：

- `tests/test_fetch_gitoauth_credential_summary_for_user.py` — 按 provider_key mock 不同 base
- 新增 `tests/test_gitoauth_per_provider_service_base.py` — gitlab-local vs github-official 路由隔离
- Playwright：`GitSiteOAuth.connection-503-degraded` 补充 gitlab-local 200 场景

---

## 领域概念（轻量，供 Step 5）

| 概念 | 说明 |
|------|------|
| **Bounded Context** | accounts（主站 OAuth 桥接） |
| **值对象** | `GitOauthServiceBase`（已有）、`OAuthProviderKey`（已有） |
| **领域服务** | 扩展 `GitOauthAuthorizeRouteService` 或新增 `GitOauthInternalApiRouteService` 解析 internal API base |
| **仓储** | 复用 `DjangoGitOauthProviderRoutingRepository` 或提取 shared `resolve_gitoauth_service_base` |
| **领域事件** | 可选：`GitOauthInternalRouteResolved` / `Rejected`（观测用，非必须） |

---

## 非目标

- 不自动探测/修复公网 `gitoauth.api.daydaymoney.com` nginx
- 不改变前端 `apiConnectionUrl` 构造（已传 `service_provider`）
- 不合并多个 gitOauth 物理实例的数据模型
- 不修改 gitOauth 服务内部 API 契约

---

## 风险

| 风险 | 缓解 |
|------|------|
| 旧部署依赖 `DJANGO_GITOAUTH_BASE` env | 发布说明：改配 `gitOauth.*.service.allowedHost` |
| `fetch_gitoauth_credential_summary_for_user()` 无参调用默认键变更 | 全仓 grep 调用方，默认 `github:github-official` |
| user-ids 多 base fan-out 重复 | 对 distinct base 去重后再请求 |

---

## 实施顺序（Step 6 输入）

1. 添加 `resolve_gitoauth_service_base` + 单元测试  
2. 重构 `github_app_tokens.py` 六处 URL 构造  
3. 修补 `GitlabAppConnectionView.delete` provider_key  
4. 删除 `DJANGO_GITOAUTH_BASE` / `django.gitoauth`  
5. 更新文档与现有 pytest  
6. Playwright：gitlab-local tab connection 200  
