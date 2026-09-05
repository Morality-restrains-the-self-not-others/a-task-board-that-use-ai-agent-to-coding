# OAuth 绑定跳转地址错误修复设计

- **日期**: 2026-07-03
- **类型**: Bug 修复
- **影响范围**: gitOauth 服务 — OAuth 授权 URL 构造逻辑

---

## 问题描述

从外部地址 `http://183.250.1.132:4000/tenant/.../task-detail/.../` 点击「OAuth 绑定」按钮，浏览器跳转到 `http://127.0.0.1:8012/users/sign_in`，这是一个内部环回地址，外部用户浏览器无法访问。

**期望行为**: 跳转到 `http://183.250.1.132:8012/oauth/authorize?...`（外部可达的 GitLab 地址）

---

## 根因分析

### 完整调用链

```
用户浏览器 (183.250.1.132:4000)
  │  点击「OAuth 绑定」
  │  repo_url 包含 127.0.0.1:8012（仓库地址为内部 GitLab）
  ▼
Vue Frontend
  │  GET /api/accounts/gitlab/oauth/start-from-gateway/?repo_url=...
  ▼
APISIX Gateway (:18081)
  │  forward-auth → X-User-Id header
  │  路由到 upstream gitOauth (172.23.0.1:8002)
  ▼
GitlabOAuthStartFromGatewayView
  │  _resolve_start_authorize_context(service_provider, allowed_host=repo_url)
  ▼
resolve_provider_config("gitlab", allowed_host="http://127.0.0.1:8012/...")
  │  匹配 hostname: "127.0.0.1" == website.hostname of http-localhost-8012.yaml ✓
  │  返回: { website: "http://127.0.0.1:8012", ... }
  ▼
OauthAuthorizeRouteDomainService.resolve()
  │  RepoOrigin("http://127.0.0.1:8012") 精确匹配 allowed_origin ✓
  │  authorize_origin = "http://127.0.0.1:8012"  ← 问题！
  ▼
authorize_url = "http://127.0.0.1:8012/oauth/authorize?..."
  │
  ▼
浏览器重定向到 127.0.0.1:8012 → GitLab → /users/sign_in
  ✗ 外部用户浏览器无法访问 127.0.0.1
```

### 根本原因

`conf/auth/git-oauth/providers/http-localhost-8012.yaml` 中 `target.website` 字段身兼两职：

| 用途 | 当前值 | 需求 |
|------|--------|------|
| **匹配** — 根据 repo URL 主机名选择提供者配置 | `http://127.0.0.1:8012` | ✅ 需要保持（仓库地址确实是 127.0.0.1） |
| **重定向** — 构造浏览器 OAuth 授权跳转 URL | `http://127.0.0.1:8012` | ❌ 应该用外部可达地址 `http://183.250.1.132:8012` |

`website` 被用于匹配逻辑（`resolve_provider_config` 通过 hostname 匹配、`RepoOrigin` 精确匹配），同时又被用于构造浏览器重定向 URL（`authorize_url`）。两个用途需要不同的地址，但只有一个字段。

### 为什么 redirect_uri 是正确的

注意：同一个配置文件中的 `redirect_uri: http://183.250.1.132:18081/api/accounts/gitlab-local/oauth/callback/` **已经使用了外部地址**。这是因为 `redirect_uri` 是 OAuth 协议参数，GitLab 授权后会回调这个地址，必须从 GitLab 服务器可达（GitLab 和 APISIX 在同一 Docker 网络）。唯独 `website`/`authorize_origin` 用于浏览器跳转，需要考虑最终用户的网络位置。

---

## 设计方案

### 核心思路：分离「匹配源」和「授权跳转源」

在提供者配置中新增可选字段 `authorize_origin`，当存在时覆盖 `website` 作为浏览器授权跳转地址。`website` 仅用于匹配逻辑。

### 改动清单

#### 1. 配置文件 — 新增 `authorize_origin` 字段

**文件**: `conf/auth/git-oauth/providers/http-localhost-8012.yaml`

```yaml
provider: gitlab
service_provider: gitlab-local
target:
  website: http://127.0.0.1:8012              # 匹配用 — 保持不变
  authorize_origin: http://183.250.1.132:8012  # 🆕 浏览器跳转用 — 外部可达地址
  client_id: 8d98496251234865578283bbf0e8d65872872b7e435979baa3c8441c13770b5a
  client_secret: gloas-REDACTED
  redirect_uri: http://183.250.1.132:18081/api/accounts/gitlab-local/oauth/callback/
  scope: read_repository api read_user
service:
  allowedHost: http://127.0.0.1:8002
  host: 127.0.0.1
  port: 8002
```

> **不需要**修改 `http-183-250-1-132-8012.yaml`（其 `website` 已是外部地址，`authorize_origin` 省略时自动 fallback 到 `website`）。

#### 2. 仓库层 — 读取 `authorize_origin`，构建正确的 `OauthAuthorizeTarget`

**文件**: `gitOauth/api/infrastructure/repositories/settings_oauth_provider_route_rule_repository.py`

修改 `list_rules()` 方法（第 31-39 行附近）：

```python
# 原代码（第 37-40 行）：
parsed = urlparse(website)
authorize_origin = (
    f"{parsed.scheme}://{parsed.netloc}" if parsed.scheme and parsed.netloc else ""
).rstrip("/")

# 改为：
raw_authorize_origin = str(row.get("authorize_origin") or "").strip().rstrip("/")
if raw_authorize_origin:
    authorize_origin = raw_authorize_origin
else:
    parsed = urlparse(website)
    authorize_origin = (
        f"{parsed.scheme}://{parsed.netloc}" if parsed.scheme and parsed.netloc else ""
    ).rstrip("/")
```

同时 `allowed_origin`（用于匹配）**继续使用 `website` 派生值**，不受影响：

```python
# 第 58 行 — 不变
allowed_origin=RepoOrigin(authorize_origin_from_website),  # 匹配仍用 website
```

等等，这里有个问题：第 58 行和第 64 行目前使用的是同一个 `authorize_origin` 变量。需要拆分为两个变量：
- `match_origin` — 从 `website` 派生，用于 `allowed_origin`（匹配）
- `redirect_origin` — 从 `authorize_origin` 或 `website` 派生，用于 `authorize_target.authorize_origin`（跳转）

具体改动：

```python
# 匹配用 origin — 始终从 website 派生
parsed = urlparse(website)
match_origin = (
    f"{parsed.scheme}://{parsed.netloc}" if parsed.scheme and parsed.netloc else ""
).rstrip("/")

# 跳转用 origin — 优先使用 authorize_origin 字段
raw_authorize_origin = str(row.get("authorize_origin") or "").strip().rstrip("/")
redirect_origin = raw_authorize_origin if raw_authorize_origin else match_origin

# 构建 route rule
built.append(
    OauthProviderRouteRule(
        id=f"{provider_norm}:{idx}",
        allowed_origin=RepoOrigin(match_origin),       # 匹配用 website
        provider_key=OauthProviderKey(...),
        authorize_target=OauthAuthorizeTarget(
            authorize_origin=redirect_origin,            # 跳转用 authorize_origin
            client_id=client_id,
            redirect_uri=redirect_uri,
            scope=scope,
        ),
    )
)
```

#### 3. 视图层 — 消除二次配置查找

**文件**: `gitOauth/api/gitlab_browser_views.py`

`_resolve_start_authorize_context()` 第 183-186 行有一个二次配置查找：

```python
# 当前代码（问题行）：
resolved_cfg = resolve_provider_config(
    _PROVIDER,
    allowed_host=resolved.authorize_origin,  # ← 用 authorize_origin 反查配置
) or {}
```

这个二次查找的目的是获取 `client_id`、`redirect_uri`、`scope`。但当 `authorize_origin` 与 `website` 不同时（`183.250.1.132` vs `127.0.0.1`），这个查找要么失败（返回空），要么错误匹配到另一个配置（`http-183-250-1-132-8012.yaml`）。

**解决方案**：让 `OauthAuthorizeRouteResolved` 携带完整的 `authorize_target`（含 `client_id`/`redirect_uri`/`scope`），视图层直接使用，不再二次查找。

##### 3a. 领域事件 — 新增 `authorize_target` 字段

**文件**: `gitOauth/api/domain/events/oauth_authorize_route_resolved.py`

```python
@dataclass(frozen=True)
class OauthAuthorizeRouteResolved:
    provider: str
    repo_origin: RepoOrigin
    provider_key: OauthProviderKey
    authorize_origin: str
    authorize_target: OauthAuthorizeTarget  # 🆕 携带完整跳转参数
    occurred_at: datetime
```

##### 3b. 领域服务 — 传递 `authorize_target`

**文件**: `gitOauth/api/domain/services/oauth_authorize_route_domain_service.py`

```python
return (
    OauthAuthorizeRouteResolved(
        provider=provider_norm,
        repo_origin=repo_origin,
        provider_key=matched.provider_key,
        authorize_origin=matched.authorize_target.authorize_origin,
        authorize_target=matched.authorize_target,  # 🆕
        occurred_at=occurred_at,
    ),
    None,
)
```

##### 3c. 视图函数 — 直接使用 `authorize_target`

**文件**: `gitOauth/api/gitlab_browser_views.py`

```python
# 删除二次查找（第 183-189 行）：
# resolved_cfg = resolve_provider_config(...)  ← 删除
# client_id = str(resolved_cfg.get("client_id") ...)  ← 删除
# ...

# 改为直接从 domain event 取值：
target = resolved.authorize_target
client_id = target.client_id
redirect_uri = target.redirect_uri
scope = target.scope
origin = target.authorize_origin
```

---

## 影响评估

### 正面影响
- ✅ 外部用户点击 OAuth 绑定后正确跳转到 `http://183.250.1.132:8012/oauth/authorize`
- ✅ `website` 匹配逻辑完全不受影响（repo URL 中含 `127.0.0.1` 仍然正确匹配 `gitlab-local` 配置）
- ✅ `authorize_origin` 为可选字段，不设置时 fallback 到 `website`，完全向后兼容
- ✅ `redirect_uri` 保持不变（已经是正确的 `183.250.1.132:18081`）
- ✅ GitHub 和 `daydaymoney-gitlab` 配置无需修改（它们使用 `${subdomains.xxx}` 占位符，运行时正确解析）

### 潜在风险
- ⚠️ `OauthAuthorizeRouteResolved` 作为 frozen dataclass，新增字段后需检查所有构造点
- ⚠️ 需要确认 `OauthAuthorizeRouteResolved` 是否有其他消费者（当前仅 `_resolve_start_authorize_context` 使用）

### 不受影响的场景
| 场景 | 是否受影响 | 原因 |
|------|-----------|------|
| GitHub OAuth 绑定 | ❌ 不受影响 | `http-github-com.yaml` 的 `website` 为 `github.com`，外部可达 |
| `daydaymoney-gitlab` 绑定 | ❌ 不受影响 | 使用 `${subdomains.gitlab}` 占位符动态解析 |
| `synology-gitlab` 绑定 | ❌ 不受影响 | `website` 已是 `http://183.250.1.132:8012` |
| 本地开发环境 | ❌ 不受影响 | 本地访问 `127.0.0.1` 时，`127.0.0.1:8012` 可达 |
| Token 交换（回调） | ❌ 不受影响 | 服务间通信使用 `service.host: 127.0.0.1`，在 Docker 网络内可达 |
| `website` 匹配逻辑 | ❌ 不受影响 | `allowed_origin` 继续从 `website` 派生 |

---

## 领域概念清单

| 概念 | 类型 | 说明 |
|------|------|------|
| OAuth Provider Config | Entity | 提供者配置（GitLab/GitHub），含 website、client_id、redirect_uri |
| OAuth Route Rule | Value Object | 路由规则：allowed_origin + authorize_target |
| OauthAuthorizeTarget | Value Object | 授权跳转目标：authorize_origin、client_id、redirect_uri、scope |
| RepoOrigin | Value Object | 仓库源地址 `{scheme}://{netloc}` |
| OauthAuthorizeRouteDomainService | Domain Service | 路由解析：repo_url → route_rule |
| SettingsOauthProviderRouteRuleRepository | Repository | 从 settings/provider configs 构建路由规则 |

---

## 验证方式

1. **单元测试**: 新增测试用例 — `authorize_origin` 存在时 `OauthAuthorizeTarget.authorize_origin` 使用它
2. **集成测试**: 模拟外部请求 `repo_url=http://127.0.0.1:8012/...`，验证返回的 `authorize_url` 以 `http://183.250.1.132:8012` 开头
3. **手动验证**: 访问 `http://183.250.1.132:4000/.../task-detail/.../`，点击 OAuth 绑定，确认跳转到正确的外部地址

---

## 🏛️ 架构变更影响

- **类型**: Bug 修复 — 不涉及组件/服务/数据流的增删改
- **架构文件**: 无需更新（不满足架构变更条件）
- **影响视图**: 无
