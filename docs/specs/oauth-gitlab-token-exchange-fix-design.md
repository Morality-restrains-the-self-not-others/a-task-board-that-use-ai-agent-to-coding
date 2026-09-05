# 设计文档：GitLab OAuth 授权回调 Token 交换失败修复

**日期**: 2026-06-27
**状态**: 待审批

---

## 1. 问题描述

用户登录后点击项目页 OAuth 授权 → 跳转 GitLab 授权 → 回调后显示：
> **"授权失败：无法与 GitLab 交换令牌"**

对应后端错误码：`exchange_failed`，即 `exchange_authorization_code_for_tokens()` 调用失败。

## 2. 完整 OAuth 链路追踪

```
用户浏览器 (port 4000)                        gitOauth (port 8002)              GitLab (port 8012)
     │                                              │                              │
     │ 1. 点击 OAuth 授权                             │                              │
     ├──→ Django GET /gitlab/oauth/start-from-gateway/│                              │
     │    (taskGateway :18081 → Django :8001)         │                              │
     │ ←── { authorize_url: "gitOauth/start?token=JWT"}                              │
     │                                              │                              │
     │ 2. 浏览器跳转 authorize_url                    │                              │
     ├──→ GET /api/accounts/gitlab/oauth/start/?token=JWT                             │
     │    (taskGateway :18081 → gitOauth :8002)       │                              │
     │    GitlabOAuthStartView: 校验JWT → session     │                              │
     │ ←── 302 → /oauth/authorize?...                 │                              │
     │                                              │                              │
     │ 3. 浏览器跟随 302 跳转 GitLab                   │                              │
     ├─────────────────────────────────────────────→ GET /oauth/authorize            │
     │ ←───────────────────────────────────────────── 授权页面                      │
     │                                              │                              │
     │ 4. 用户登录 GitLab + 点击授权                    │                              │
     │ ←── 302 → redirect_uri?code=CODE&state=CSRF   │                              │
     │                                              │                              │
     │ 5. 浏览器跟随 302 回调                          │                              │
     ├──→ GET /api/accounts/synology-gitlab/oauth/callback/?code=...&state=...       │
     │    (taskGateway :18081 → gitOauth :8002)       │                              │
     │    GitlabOAuthCallbackView.get()                │                              │
     │                                              │                              │
     │    ★ exchange_authorization_code_for_tokens()   │                              │
     │    POST {website}/oauth/token                   │                              │
     │    = POST http://183.250.1.132:8012/oauth/token ├───────────────────────────→ │
     │                                              │   ←── ❌ 请求失败              │
     │                                              │                              │
     │ ←── 302 → /profile/git-site-oauth/?gitlab=exchange_failed                     │
     │                                              │                              │
     │ 6. 前端显示 "授权失败：无法与 GitLab 交换令牌"    │                              │
```

## 3. 根因分析

### 3.1 主要根因（高置信度）

**SOCKS5 代理劫持 gitOauth 出站 HTTP 请求。**

`api/gitlab_tokens.py` 第 63 行：
```python
r = requests.post(token_url, data=data, auth=(cid, sec), timeout=30)
```

`requests.post()` 默认 `trust_env=True`，会读取宿主机的 `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` 环境变量。

已知宿主配置了 `ALL_PROXY=socks5://...`（用于 Clash/翻墙），gitOauth 进程发起 `POST http://183.250.1.132:8012/oauth/token` 时，请求被路由到 SOCKS5 代理。代理无法连接本地局域网地址 `183.250.1.132:8012`，导致 `SOCKSHTTPConnectionPool ... [Errno 111] Connection refused`。

**证据链:**

1. 同项目 `task2app/Saas_project/core/utils/http_client.py` 已因相同问题添加 `trust_env=False`（见 memory: `httpclient-socks-proxy-bypass.md`）
2. gitOauth 所有 `requests` 调用均未设置 `trust_env=False`，直接使用顶层 `requests.post()` / `requests.get()`
3. `run.sh` 第 8 行 `export NO_PROXY="127.0.0.1,localhost..."` 仅排除环回地址，`183.250.1.132` 不在其中
4. `gitlab_browser_views.py` 第 422-438 行 `requests.post()` 到内部 bind API 同样未做代理隔离

### 3.2 次要根因

**错误日志缺乏可观测性。** `exchange_authorization_code_for_tokens()` 捕获 `requests.RequestException` 后仅记录 `"GitLab OAuth exchange 失败: %s"`，未区分代理失败 / 连接超时 / GitLab 返回错误码等不同失败模式。

## 4. 修复方案

### 修复 A：gitOauth 创建统一 HTTP Session（推荐，必须）

在 gitOauth 中创建 `api/http_client.py`，提供 `trust_env=False` 的 thread-local session，供所有出站 HTTP 调用使用。

**受影响文件:**

| 文件 | 当前调用 | 替换方式 |
|------|---------|---------|
| `gitOauth/api/gitlab_tokens.py` | `requests.post()` ×2, `requests.get()` ×1 | `session.post()`, `session.get()` |
| `gitOauth/api/github_tokens.py` | `requests.post()` ×2, `requests.get()` ×1 | 同上 |
| `gitOauth/api/gitlab_browser_views.py` | `requests.post()` ×1 | 同上 |
| `gitOauth/api/github_browser_views.py` | `requests.post()` ×1 | 同上 |

**新增文件:** `gitOauth/api/http_client.py`

### 修复 B：增强错误可观测性

在 `exchange_authorization_code_for_tokens()` 的 except 块中，区分并记录：
- 代理类错误（`SOCKSHTTPConnectionPool` / `ProxyError`）
- 连接错误（`ConnectionError`，超时等）
- GitLab 业务错误（HTTP 4xx/5xx，记录 status_code 和 body 摘要）

### 修复 C：run.sh NO_PROXY 加固

在 `run.sh` 中将 `183.250.1.132` 加入 `NO_PROXY`（防御纵深，治标不治本）。

## 5. 价值流影响分析

### 受影响的价值流

| 价值流 | 影响 |
|--------|------|
| `gitoauth-binding-state-persistence` (lines 979-1012) | 本次修复涉及的 GitLab OAuth 回调 → token exchange → bind 主站链路 |
| `project-detail-repo-oauth-row-action` (lines 1013-1093) | OAuth start 入口到回调的完整生命周期，exchange 失败会在前端显示错误 |
| `gitlab-oauth-scope-failfast-governance` (lines 1167-1217) | 与 GitLab OAuth 启动配置校验同属 GitLab OAuth 闭环 |

### 字段影响

本次为基础设施修复（HTTP outbound client），不新增/修改业务字段。

### 测试影响

- 需新增：`gitOauth/api/test_http_client.py` — HTTP client trust_env 行为测试
- 需更新：`gitOauth/api/tests.py` — 已有 mock `requests.post` 的测试需适配（改用 mock session）

## 6. 领域概念清单

| 概念 | 说明 |
|------|------|
| **Bounded Context: OAuth 授权** | gitOauth 服务负责 Git 站点 OAuth 授权的启动、回调、token 交换 |
| **实体: GitOAuthAppUserCredential** | 用户 OAuth 凭据（已有），含 bind_status / bind_error |
| **值对象: OAuthProviderRoute** | provider + service_provider + redirect_uri 的路由决策 |
| **领域服务: OAuth Token Exchange** | `exchange_authorization_code_for_tokens` — 与 GitLab 的 token 端点通信 |
| **基础设施关注点: HTTP Outbound** | 出站 HTTP 调用需绕过宿主代理（trust_env=False）|

## 7. 实施步骤概要

1. 创建 `gitOauth/api/http_client.py`（thread-local session with `trust_env=False`）
2. 替换 `gitlab_tokens.py` 中的 `requests` 调用
3. 替换 `github_tokens.py` 中的 `requests` 调用
4. 替换 `gitlab_browser_views.py` 和 `github_browser_views.py` 中的 `requests` 调用
5. 增强 `exchange_authorization_code_for_tokens()` 错误日志
6. 更新 `run.sh` NO_PROXY
7. 编写单元测试
8. 运行现有测试确保无回归
